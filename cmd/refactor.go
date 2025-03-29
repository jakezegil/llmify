package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/editor"
	"github.com/jake/llmify/internal/git"
	"github.com/jake/llmify/internal/language"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/tools"
	"github.com/jake/llmify/internal/walker"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var refactorCmd = &cobra.Command{
	Use:   "refactor [file]",
	Short: "Refactor code using LLM",
	Long: `Refactor code using LLM. The command can target a single file or a directory.
The command analyzes the code and applies refactoring changes based on the provided prompt.

Examples:
  # Refactor a single file
  llmify refactor src/process.ts --prompt "Convert to functional style"

  # Refactor all TypeScript files in a directory
  llmify refactor src/ --prompt "Add error handling"`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		diff, err := git.GetStagedDiff()
		if err != nil {
			log.Printf("Warning: Could not get git diff: %v", err)
			// Continue without diff context
		}

		// Initialize LLM client
		client, err := llm.NewLLMClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize LLM client: %w", err)
		}

		// Process single file if specified
		if len(args) > 0 {
			filePath := args[0]
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return fmt.Errorf("invalid file path: %w", err)
			}

			// Get relative path for standards matching
			relPath, err := filepath.Rel(repoRoot, absPath)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// Read file content
			content, err := os.ReadFile(absPath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			// Get prompt from flag or use default
			prompt := viper.GetString("prompt")
			if prompt == "" {
				return fmt.Errorf("prompt is required for refactoring")
			}

			// Prepare context for LLM
			context := fmt.Sprintf("File: %s\n\nContent:\n%s\n\nChanges:\n%s", relPath, string(content), diff)

			// Get LLM response
			response, err := client.Generate(cmd.Context(), context, cfg.LLM.Model)
			if err != nil {
				return fmt.Errorf("failed to get LLM response: %w", err)
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
				if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
					return fmt.Errorf("failed to write changes: %w", err)
				}

				// Format and lint the file if tools are available
				lang := language.Detect(absPath)
				if formatter, linter := tools.GetToolForLanguage(lang); formatter != nil {
					if err := formatter.Format(absPath); err != nil {
						log.Printf("Warning: Failed to format %s: %v", relPath, err)
					}
					if linter != nil {
						if output, err := linter.Lint(absPath); err != nil {
							log.Printf("Warning: Failed to lint %s: %v\nOutput: %s", relPath, err, output)
						}
					}
				}

				fmt.Printf("Refactored %s\n", relPath)
			} else {
				fmt.Printf("No changes needed for %s\n", relPath)
			}

			return nil
		}

		// Process all files in the project
		ignorer, err := gitignore.CompileIgnoreFile(filepath.Join(repoRoot, ".gitignore"))
		if err != nil {
			log.Printf("Warning: Could not load .gitignore: %v", err)
		}

		var processed, changed, errors, skipped int
		err = walker.WalkProjectFiles(repoRoot, repoRoot, ignorer, func(repoRoot, filePathRel string, lang string, d fs.DirEntry) error {
			// Skip non-code files
			if lang == "" {
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

			// Get prompt from flag or use default
			prompt := viper.GetString("prompt")
			if prompt == "" {
				return fmt.Errorf("prompt is required for refactoring")
			}

			// Prepare context for LLM
			context := fmt.Sprintf("File: %s\n\nContent:\n%s\n\nChanges:\n%s", filePathRel, string(content), diff)

			// Get LLM response
			response, err := client.Generate(cmd.Context(), context, cfg.LLM.Model)
			if err != nil {
				errors++
				log.Printf("Error getting LLM response for %s: %v", filePathRel, err)
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
				if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
					errors++
					log.Printf("Error writing changes to %s: %v", filePathRel, err)
					return nil
				}

				// Format and lint the file if tools are available
				if formatter, linter := tools.GetToolForLanguage(lang); formatter != nil {
					if err := formatter.Format(absPath); err != nil {
						log.Printf("Warning: Failed to format %s: %v", filePathRel, err)
					}
					if linter != nil {
						if output, err := linter.Lint(absPath); err != nil {
							log.Printf("Warning: Failed to lint %s: %v\nOutput: %s", filePathRel, err, output)
						}
					}
				}

				changed++
				fmt.Printf("Refactored %s\n", filePathRel)
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
	},
}

func init() {
	rootCmd.AddCommand(refactorCmd)

	// Add flags
	refactorCmd.Flags().String("prompt", "", "Prompt describing the refactoring goal (required)")
	viper.BindPFlag("prompt", refactorCmd.Flags().Lookup("prompt"))
}
