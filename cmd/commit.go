package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/git"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	commitUpdateDocs bool
	commitForce      bool
)

var CommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message for staged changes using an LLM.",
	Long: `Analyzes staged code changes (git diff --staged), generates a detailed
commit message suggestion using the configured LLM, allows editing,
and optionally updates documentation files before committing.`,
	RunE: runCommit,
}

func init() {
	CommitCmd.Flags().BoolVar(&commitUpdateDocs, "docs", false, "Attempt to automatically update relevant documentation files (*.md) based on changes.")
	CommitCmd.Flags().BoolVarP(&commitForce, "force", "f", false, "Skip the final confirmation prompt before committing.")
	// Add other flags if necessary
}

func runCommit(cmd *cobra.Command, args []string) error {
	verbose := viper.GetBool("verbose") // Get verbose flag state if set globally
	if verbose {
		log.Println("Running commit command...")
	}

	// --- 0. Load Config ---
	// Config is loaded via root command's PersistentPreRun or called explicitly here
	// Assuming GlobalConfig is populated from config.LoadConfig() called elsewhere
	if err := config.LoadConfig(); err != nil { // Call here if not done globally
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg := &config.GlobalConfig // Use the globally loaded config

	// --- 1. Get Staged Changes ---
	if verbose {
		log.Println("Getting staged diff...")
	}
	diff, err := git.GetStagedDiff()
	if err != nil {
		if strings.Contains(err.Error(), "no changes staged") {
			fmt.Println("No changes staged for commit.")
			return nil // Not an error for the command
		}
		return fmt.Errorf("failed to get staged changes: %w", err)
	}

	if verbose {
		log.Println("Getting staged files...")
	}
	stagedFiles, err := git.GetStagedFiles()
	if err != nil {
		return fmt.Errorf("failed to get staged file list: %w", err)
	}
	if len(stagedFiles) == 0 {
		fmt.Println("No files staged for commit (diff reported changes, but file list is empty - check git status).")
		return nil
	}

	// --- 2. Gather Context ---
	if verbose {
		log.Println("Gathering context from staged files...")
	}
	var contextBuilder strings.Builder
	repoRoot, err := git.GetRepoRoot() // Get root to construct full paths
	if err != nil {
		log.Printf("Warning: could not get repo root, using relative paths: %v", err)
		repoRoot = "." // Fallback
	}

	// TODO: Add token limiting logic here if context gets too large
	const maxContextChars = 100000 // Example limit, adjust as needed
	currentChars := 0

	for _, fileRelPath := range stagedFiles {
		fullPath := filepath.Join(repoRoot, fileRelPath)
		// Check if file exists before reading (it might be a deleted file in the diff)
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			contextBuilder.WriteString(fmt.Sprintf("\n--- File (Deleted): %s ---\n", fileRelPath))
			continue
		}

		content, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			log.Printf("Warning: could not read file %s: %v", fileRelPath, readErr)
			content = []byte(fmt.Sprintf("Error reading file: %v", readErr))
		}

		fileHeader := fmt.Sprintf("\n--- File: %s ---\n", fileRelPath)
		if currentChars+len(fileHeader)+len(content) > maxContextChars {
			remainingSpace := maxContextChars - currentChars - len(fileHeader) - 20 // reserve space for truncation message
			if remainingSpace > 0 {
				contextBuilder.WriteString(fileHeader)
				contextBuilder.Write(content[:remainingSpace])
				contextBuilder.WriteString("\n... (file truncated)\n")
			}
			log.Printf("Warning: Context truncated due to size limits. Files included: %d of %d", len(contextBuilder.String()), len(stagedFiles)) // Crude count
			break                                                                                                                                 // Stop adding more files
		}

		contextBuilder.WriteString(fileHeader)
		contextBuilder.Write(content)
		currentChars += len(fileHeader) + len(content)
	}
	fullContext := contextBuilder.String()

	// --- 3. Create LLM Client ---
	if verbose {
		log.Printf("Initializing LLM client (Provider: %s)", cfg.LLM.Provider)
	}
	llmClient, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// --- 4. Generate Commit Message ---
	commitModel := cfg.Commit.Model // Use specific commit model
	if verbose {
		log.Printf("Generating commit message using model: %s...", commitModel)
	}
	commitPrompt := llm.CreateCommitPrompt(diff, fullContext)
	// Use context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("llm.timeout_seconds"))*time.Second) // Add timeout config
	defer cancel()

	proposedMessage, err := llmClient.Generate(ctx, commitPrompt, commitModel)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}
	proposedMessage = strings.TrimSpace(proposedMessage) // Clean up LLM output

	// --- 5. Handle --docs flag ---
	updatedDocs := []string{}
	if commitUpdateDocs {
		if verbose {
			log.Println("Processing --docs flag...")
		}
		docsModel := cfg.Docs.Model // Use specific docs model

		// Find candidate *.md files (simple approach: walk current dir)
		// TODO: Make this smarter (use repo root, respect ignores)
		var candidateDocs []string
		err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Basic filter: .md extension, not in .git, maybe respect .llmignore later
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") && !strings.Contains(path, ".git"+string(filepath.Separator)) {
				// Convert to relative path if needed, or use as is if WalkDir started from '.'
				candidateDocs = append(candidateDocs, path)
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: Error scanning for documentation files: %v", err)
		}

		if verbose {
			log.Printf("Found %d potential markdown files to check.", len(candidateDocs))
		}

		for _, docPath := range candidateDocs {
			if verbose {
				log.Printf("Checking doc: %s", docPath)
			}
			docContent, readErr := os.ReadFile(docPath)
			if readErr != nil {
				log.Printf("Warning: could not read doc file %s: %v", docPath, readErr)
				continue
			}

			docPrompt := llm.CreateDocsUpdatePrompt(diff, string(docContent))
			ctxDocs, cancelDocs := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("llm.timeout_seconds"))*time.Second) // Separate timeout

			docResponse, llmErr := llmClient.Generate(ctxDocs, docPrompt, docsModel)
			cancelDocs() // Release context resources
			if llmErr != nil {
				log.Printf("Warning: LLM failed to process doc %s: %v", docPath, llmErr)
				continue
			}

			needsUpdate, newContent := llm.NeedsDocUpdate(docResponse)
			if needsUpdate {
				if verbose {
					log.Printf("LLM proposed update for: %s", docPath)
				}
				// Write the new content back to the file
				writeErr := os.WriteFile(docPath, []byte(newContent), 0644)
				if writeErr != nil {
					log.Printf("Warning: Failed to write updated doc %s: %v", docPath, writeErr)
				} else {
					updatedDocs = append(updatedDocs, docPath) // Add to list for staging
				}
			} else {
				if verbose {
					log.Printf("No update needed for: %s", docPath)
				}
			}
		}

		// Stage updated docs
		if len(updatedDocs) > 0 {
			if verbose {
				log.Printf("Staging updated documentation files: %v", updatedDocs)
			}
			err = git.AddFiles(updatedDocs)
			if err != nil {
				// Log warning but maybe proceed with commit? Or make it fatal?
				log.Printf("Warning: Failed to stage updated docs: %v", err)
			} else {
				fmt.Printf("Updated and staged documentation files: %s\n", strings.Join(updatedDocs, ", "))
			}
		}
	}

	// --- 6. Edit, Confirm, Commit Loop ---
	finalMessage := proposedMessage
	for {
		fmt.Println("\n--- Proposed Commit Message ---")
		fmt.Println(finalMessage)
		fmt.Println("-----------------------------")

		// Allow editing
		editedMessage, editErr := ui.EditCommitMessage(finalMessage)
		if editErr != nil {
			return fmt.Errorf("failed during commit message editing: %w", editErr)
		}
		finalMessage = strings.TrimSpace(editedMessage)
		if finalMessage == "" {
			fmt.Println("Commit aborted: Empty commit message.")
			return nil
		}

		// Confirm
		proceed, editAgain, confirmErr := ui.ConfirmCommit(commitForce)
		if confirmErr != nil {
			return confirmErr // Error reading confirmation
		}

		if proceed {
			break // Exit loop to commit
		}
		if !editAgain {
			fmt.Println("Commit aborted.")
			return nil // User chose 'no'
		}
		// Otherwise, loop back to edit again
	}

	// --- 7. Execute Commit ---
	if verbose {
		log.Println("Executing git commit...")
	}
	err = git.Commit(finalMessage)
	if err != nil {
		return fmt.Errorf("git commit execution failed: %w", err)
	}

	fmt.Println("Commit successful.")
	return nil
}
