package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/diff"
	"github.com/jake/llmify/internal/editor"
	"github.com/jake/llmify/internal/git"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/refactor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var docsCmd = &cobra.Command{
	Use:   "docs [file or directory]",
	Short: "Update documentation using LLM",
	Long: `Update documentation using LLM. The command can target a single file or a directory.

Examples:
  # Update a single documentation file
  llmify docs README.md --prompt "Update installation instructions"

  # Update all documentation in a directory
  llmify docs docs/ --prompt "Update API documentation"

  # Update without staging changes
  llmify docs docs/api.md --prompt "Add new endpoint docs" --no-stage`,
	Args: cobra.ExactArgs(1),
	RunE: runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)

	docsCmd.Flags().StringP("prompt", "p", "", "Prompt describing the documentation update goal (required)")
	docsCmd.Flags().StringP("scope", "s", "", "Scope of documentation update (e.g., section name)")
	docsCmd.Flags().Bool("show-diff", true, "Show diff of proposed changes")
	docsCmd.Flags().Bool("no-diff", false, "Do not show diffs of proposed changes")
	docsCmd.Flags().Bool("dry-run", false, "Show proposed changes without applying them")
	docsCmd.Flags().BoolP("force", "f", false, "Apply changes without confirmation")
	docsCmd.Flags().Bool("stage", true, "Stage modified files in git")
	docsCmd.Flags().Bool("no-stage", false, "Do not stage modified files in git")

	docsCmd.MarkFlagRequired("prompt")
}

func runDocs(cmd *cobra.Command, args []string) error {
	// Get flags
	prompt, _ := cmd.Flags().GetString("prompt")
	showDiff, _ := cmd.Flags().GetBool("show-diff")
	noDiff, _ := cmd.Flags().GetBool("no-diff")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	stage, _ := cmd.Flags().GetBool("stage")
	noStage, _ := cmd.Flags().GetBool("no-stage")
	verbose := viper.GetBool("verbose")

	// Handle --no-diff and --no-stage flags
	if noDiff {
		showDiff = false
	}
	if noStage {
		stage = false
	}

	// Set flags in viper for other packages to access
	viper.Set("docs.show_diff", showDiff)
	viper.Set("docs.stage", stage)

	// Initialize LLM client
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg := &config.GlobalConfig

	llmClient, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	// Get target path
	targetPath := args[0]
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("failed to access target path %s: %w", targetPath, err)
	}

	var filesToProcess []string

	// Process directory or single file
	if info.IsDir() {
		// Walk directory to find documentation files
		err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and hidden files/directories
			if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				return nil
			}

			// Skip common build directories and binary files
			if shouldSkipPath(path) {
				return nil
			}

			// Check if file is likely a documentation file
			if isLikelyDocFile(path) {
				filesToProcess = append(filesToProcess, path)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", targetPath, err)
		}
	} else {
		// Single file
		if !isLikelyDocFile(targetPath) {
			return fmt.Errorf("file %s does not appear to be a documentation file", targetPath)
		}
		filesToProcess = []string{targetPath}
	}

	if len(filesToProcess) == 0 {
		return fmt.Errorf("no documentation files found to process in %s", targetPath)
	}

	// Process each file
	var (
		totalFiles     = len(filesToProcess)
		changedFiles   = 0
		errorFiles     = 0
		skippedFiles   = 0
		appliedChanges = 0
	)

	for _, file := range filesToProcess {
		if verbose {
			log.Printf("Processing %s...", file)
		}

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading %s: %v", file, err)
			errorFiles++
			continue
		}

		// Get git diff for context
		gitDiff, err := git.GetStagedDiff()
		if err != nil {
			log.Printf("Warning: Could not get git diff: %v", err)
			gitDiff = ""
		}

		// Create documentation update prompt
		updatePrompt := fmt.Sprintf(`
You are an expert technical writer specializing in clear and accurate documentation.
Your task is to update the provided documentation based on code changes, ensuring it remains accurate and helpful.

USER'S DOCUMENTATION UPDATE GOAL:
%s

CONTEXT (Code Changes):
--- CONTEXT START ---
%s
--- CONTEXT END ---

TARGET DOCUMENTATION:
--- TARGET START ---
%s
--- TARGET END ---

IMPORTANT INSTRUCTIONS:
1. Only update the documentation if necessary based on the code changes.
2. Focus on changes to:
   - Function signatures
   - Parameters
   - Return types
   - Added/removed features
   - Usage examples
   - Clarifications based on code changes
3. Do not make unnecessary changes or add speculative information.
4. Preserve existing formatting and style.
5. If no updates are needed, respond with exactly: NO_UPDATE_NEEDED

OUTPUT FORMAT:
If changes are needed, provide them in one of these formats:

1. For replacing existing content:
--- LLMIFY REPLACE START ---
<<< ORIGINAL >>>
[The exact lines to be replaced]
<<< REPLACEMENT >>>
[The new lines to replace the original block]
--- LLMIFY REPLACE END ---

2. For inserting new content:
--- LLMIFY INSERT_AFTER START ---
<<< CONTEXT_LINE >>>
[The exact line content *immediately preceding* the desired insertion point]
<<< INSERTION >>>
[The new lines to be inserted]
--- LLMIFY INSERT_AFTER END ---

3. For deleting content:
--- LLMIFY DELETE START ---
<<< CONTENT >>>
[The exact lines to be deleted]
--- LLMIFY DELETE END ---

If the changes are too extensive or complex for the edit format, provide the complete updated content enclosed in triple backticks:
`+"```"+`markdown
[Complete updated content]
`+"```"+`
`, prompt, gitDiff, string(content))

		// Call LLM
		response, err := llmClient.Generate(cmd.Context(), updatePrompt, cfg.LLM.Model)
		if err != nil {
			log.Printf("Error generating documentation update for %s: %v", file, err)
			errorFiles++
			continue
		}

		// Handle "NO_UPDATE_NEEDED" response
		if strings.TrimSpace(response) == "NO_UPDATE_NEEDED" {
			if verbose {
				log.Printf("No updates needed for %s", file)
			}
			skippedFiles++
			continue
		}

		// Parse the LLM response for edits or full file content
		edits, fullContent, err := editor.ParseLLMResponse(response)
		if err != nil {
			log.Printf("Error parsing LLM response for %s: %v", file, err)
			errorFiles++
			continue
		}

		var newContent string
		if fullContent != "" {
			newContent = fullContent
		} else if len(edits) > 0 {
			// Apply the parsed edits
			newContent, err = editor.ApplyEdits(string(content), edits)
			if err != nil {
				log.Printf("Error applying edits for %s: %v", file, err)
				errorFiles++
				continue
			}
		} else {
			log.Printf("No changes proposed for %s", file)
			skippedFiles++
			continue
		}

		// Show diff if enabled
		if showDiff {
			fmt.Printf("\n--- Proposed Changes for: %s ---\n", file)
			diff.ShowDiff(string(content), newContent)
			fmt.Println("------------------------------------")
		}

		// Apply changes if not in dry-run mode and either forced or confirmed
		if !dryRun && (force || confirmChanges(&refactor.RefactorResult{FilePath: file})) {
			if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
				log.Printf("Error writing changes to %s: %v", file, err)
				errorFiles++
				continue
			}

			// Stage changes if requested
			if stage {
				if err := git.AddFiles([]string{file}); err != nil {
					log.Printf("Warning: Could not stage changes for %s: %v", file, err)
				}
			}

			appliedChanges++
			changedFiles++
		} else if !dryRun {
			skippedFiles++
		} else {
			changedFiles++
		}
	}

	// Print summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total files processed: %d\n", totalFiles)
	fmt.Printf("Files with changes: %d\n", changedFiles)
	fmt.Printf("Files with errors: %d\n", errorFiles)
	fmt.Printf("Files skipped: %d\n", skippedFiles)
	if !dryRun {
		fmt.Printf("Changes applied: %d\n", appliedChanges)
	}

	return nil
}

// isLikelyDocFile returns true if the file is likely a documentation file
func isLikelyDocFile(path string) bool {
	// Common documentation file extensions and names
	docExtensions := []string{
		".md", ".mdx", // Markdown
		".rst",               // reStructuredText
		".txt",               // Plain text
		".adoc", ".asciidoc", // AsciiDoc
		".wiki", // Wiki markup
		".org",  // Org mode
		".pod",  // Perl POD
		".tex",  // LaTeX
	}

	docFileNames := []string{
		"readme", "changelog", "contributing", "license", "authors",
		"install", "installation", "setup", "getting-started",
		"api", "reference", "manual", "guide", "tutorial",
		"faq", "troubleshooting", "support",
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(path))
	for _, docExt := range docExtensions {
		if ext == docExt {
			return true
		}
	}

	// Check filename
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ext))
	for _, docName := range docFileNames {
		if base == docName {
			return true
		}
	}

	return false
}
