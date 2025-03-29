package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/diff"
	"github.com/jake/llmify/internal/editor"
	"github.com/jake/llmify/internal/git"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/walker"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var docsCmd = &cobra.Command{
	Use:   "docs [file or directory]",
	Short: "Update documentation using LLM",
	Long: `Update documentation using LLM. The command can target a single file or a directory.
If no file or directory is specified, the current directory will be used.

Examples:
  # Update all documentation in current directory based on code changes
  llmify docs

  # Update documentation with a specific goal
  llmify docs --prompt "Update installation instructions"

  # Update a specific file with default prompt
  llmify docs README.md

  # Update all documentation in a directory with a specific goal
  llmify docs docs/ --prompt "Update API documentation"

  # Update without staging changes
  llmify docs docs/api.md --no-stage`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		prompt, _ := cmd.Flags().GetString("prompt")
		if prompt == "" {
			prompt = "Review and update the documentation to accurately reflect any recent code changes. Focus on keeping the documentation clear, accurate, and up-to-date with the current codebase."
		}
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

		// Get repository root
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repository root: %w", err)
		}

		// Load config
		if err := config.LoadConfig(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		cfg := &config.GlobalConfig

		// Get git diff for context
		gitDiff, err := git.GetStagedDiff()
		if err != nil {
			log.Printf("Warning: Could not get git diff: %v", err)
			// Continue without diff context
		}

		// Initialize LLM client
		client, err := llm.NewLLMClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize LLM client: %w", err)
		}

		// Get target path (default to current directory if not specified)
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		// Process directory or single file
		info, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("failed to access target path %s: %w", targetPath, err)
		}

		if info.IsDir() {
			// Process all documentation files in the directory
			ignorer, err := gitignore.CompileIgnoreFile(filepath.Join(repoRoot, ".gitignore"))
			if err != nil {
				log.Printf("Warning: Could not load .gitignore: %v", err)
			}

			var processed, changed, errors, skipped int
			err = walker.WalkProjectFiles(repoRoot, targetPath, ignorer, func(repoRoot, filePathRel string, lang string, d fs.DirEntry) error {
				// Only process markdown files
				if lang != "markdown" {
					skipped++
					return nil
				}

				processed++
				absPath := filepath.Join(repoRoot, filePathRel)

				// Read file content
				content, err := os.ReadFile(absPath)
				if err != nil {
					errors++
					log.Printf("Error reading %s: %v", filePathRel, err)
					return nil
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

				// Get LLM response
				response, err := client.Generate(cmd.Context(), updatePrompt, cfg.LLM.Model)
				if err != nil {
					errors++
					log.Printf("Error getting LLM response for %s: %v", filePathRel, err)
					return nil
				}

				// Handle "NO_UPDATE_NEEDED" response
				if strings.TrimSpace(response) == "NO_UPDATE_NEEDED" {
					if verbose {
						log.Printf("No updates needed for %s", filePathRel)
					}
					skipped++
					return nil
				}

				// Apply changes using editor package
				edits, fullContent, err := editor.ParseLLMResponse(response)
				if err != nil {
					errors++
					log.Printf("Error parsing LLM response for %s: %v", filePathRel, err)
					return nil
				}

				var newContent string
				if fullContent != "" {
					newContent = fullContent
				} else if len(edits) > 0 {
					newContent, err = editor.ApplyEdits(string(content), edits)
					if err != nil {
						errors++
						log.Printf("Error applying edits to %s: %v", filePathRel, err)
						return nil
					}
				}

				if newContent != "" {
					// Show diff if enabled
					if showDiff {
						fmt.Printf("\n--- Proposed Changes for: %s ---\n", filePathRel)
						diff.ShowDiff(string(content), newContent)
						fmt.Println("------------------------------------")
					}

					// Apply changes if not in dry-run mode and either forced or confirmed
					if !dryRun && (force || confirmChanges(filePathRel)) {
						if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
							errors++
							log.Printf("Error writing changes to %s: %v", filePathRel, err)
							return nil
						}

						// Stage changes if requested
						if stage {
							if err := git.AddFiles([]string{filePathRel}); err != nil {
								log.Printf("Warning: Could not stage changes for %s: %v", filePathRel, err)
							}
							// Commit changes
							if err := git.Commit("docs: Update documentation based on code changes"); err != nil {
								log.Printf("Warning: Could not commit changes for %s: %v", filePathRel, err)
							}
						}

						changed++
						fmt.Printf("Updated %s\n", filePathRel)
					} else if !dryRun {
						skipped++
					} else {
						changed++
					}
				}

				return nil
			})

			if err != nil {
				return fmt.Errorf("error walking project files: %w", err)
			}

			// Print summary
			fmt.Printf("\nSummary:\n")
			fmt.Printf("Total files processed: %d\n", processed)
			fmt.Printf("Files changed: %d\n", changed)
			fmt.Printf("Files with errors: %d\n", errors)
			fmt.Printf("Files skipped: %d\n", skipped)

			return nil
		} else {
			// Process single file
			// Get relative path for standards matching
			relPath, err := filepath.Rel(repoRoot, targetPath)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// Read file content
			content, err := os.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
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

			// Get LLM response
			response, err := client.Generate(cmd.Context(), updatePrompt, cfg.LLM.Model)
			if err != nil {
				return fmt.Errorf("failed to get LLM response: %w", err)
			}

			// Handle "NO_UPDATE_NEEDED" response
			if strings.TrimSpace(response) == "NO_UPDATE_NEEDED" {
				if verbose {
					log.Printf("No updates needed for %s", relPath)
				}
				return nil
			}

			// Apply changes using editor package
			edits, fullContent, err := editor.ParseLLMResponse(response)
			if err != nil {
				return fmt.Errorf("failed to parse LLM response: %w", err)
			}

			var newContent string
			if fullContent != "" {
				newContent = fullContent
			} else if len(edits) > 0 {
				newContent, err = editor.ApplyEdits(string(content), edits)
				if err != nil {
					return fmt.Errorf("failed to apply edits: %w", err)
				}
			}

			if newContent != "" {
				// Show diff if enabled
				if showDiff {
					fmt.Printf("\n--- Proposed Changes for: %s ---\n", relPath)
					diff.ShowDiff(string(content), newContent)
					fmt.Println("------------------------------------")
				}

				// Apply changes if not in dry-run mode and either forced or confirmed
				if !dryRun && (force || confirmChanges(relPath)) {
					if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
						return fmt.Errorf("failed to write changes: %w", err)
					}

					// Stage changes if requested
					if stage {
						if err := git.AddFiles([]string{relPath}); err != nil {
							log.Printf("Warning: Could not stage changes for %s: %v", relPath, err)
						}
						// Commit changes
						if err := git.Commit("docs: Update documentation based on code changes"); err != nil {
							log.Printf("Warning: Could not commit changes for %s: %v", relPath, err)
						}
					}

					fmt.Printf("Updated %s\n", relPath)
				} else if !dryRun {
					fmt.Printf("Changes not applied to %s\n", relPath)
				} else {
					fmt.Printf("Would update %s\n", relPath)
				}
			} else {
				fmt.Printf("No changes needed for %s\n", relPath)
			}

			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)

	// Add flags
	docsCmd.Flags().StringP("prompt", "p", "", "Prompt describing the documentation update goal (optional, defaults to general update)")
	docsCmd.Flags().StringP("scope", "s", "", "Scope of documentation update (e.g., section name)")
	docsCmd.Flags().Bool("show-diff", true, "Show diff of proposed changes")
	docsCmd.Flags().Bool("no-diff", false, "Do not show diffs of proposed changes")
	docsCmd.Flags().Bool("dry-run", false, "Show proposed changes without applying them")
	docsCmd.Flags().BoolP("force", "f", false, "Apply changes without confirmation")
	docsCmd.Flags().Bool("stage", true, "Stage modified files in git")
	docsCmd.Flags().Bool("no-stage", false, "Do not stage modified files in git")
}

// confirmChanges prompts the user to confirm changes to a file
func confirmChanges(filePath string) bool {
	fmt.Printf("Apply changes to %s? [y/N] ", filePath)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}
