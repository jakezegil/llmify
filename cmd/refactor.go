package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/refactor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var refactorCmd = &cobra.Command{
	Use:   "refactor [file or directory]",
	Short: "Refactor code using LLM",
	Long: `Refactor code using LLM. The command can target a single file or a directory.

Examples:
  # Refactor a single file
  llmify refactor src/components/Button.tsx --prompt "Convert to functional component"

  # Refactor all files in a directory
  llmify refactor src/ --prompt "Add error handling to all API calls"

  # Refactor with type checking disabled
  llmify refactor src/utils.ts --prompt "Optimize performance" --no-check-types`,
	Args: cobra.ExactArgs(1),
	RunE: runRefactor,
}

func init() {
	rootCmd.AddCommand(refactorCmd)

	refactorCmd.Flags().StringP("prompt", "p", "", "Prompt describing the refactoring goal (required)")
	refactorCmd.Flags().StringP("scope", "s", "", "Scope of refactoring (e.g., function name, class name)")
	refactorCmd.Flags().Bool("check-types", true, "Run language-specific checks after refactoring (e.g., tsc --noEmit for TypeScript)")
	refactorCmd.Flags().Bool("no-check-types", false, "Disable language-specific checks")
	refactorCmd.Flags().Bool("show-diff", true, "Show diff of proposed changes")
	refactorCmd.Flags().Bool("dry-run", false, "Show proposed changes without applying them")
	refactorCmd.Flags().Bool("apply", false, "Apply changes without confirmation")

	refactorCmd.MarkFlagRequired("prompt")
}

func runRefactor(cmd *cobra.Command, args []string) error {
	// Get flags
	prompt, _ := cmd.Flags().GetString("prompt")
	scope, _ := cmd.Flags().GetString("scope")
	checkTypes, _ := cmd.Flags().GetBool("check-types")
	noCheckTypes, _ := cmd.Flags().GetBool("no-check-types")
	showDiff, _ := cmd.Flags().GetBool("show-diff")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	autoApply, _ := cmd.Flags().GetBool("apply")
	verbose := viper.GetBool("verbose")

	// Handle --no-check-types flag
	if noCheckTypes {
		checkTypes = false
	}

	// Set flags in viper for other packages to access
	viper.Set("refactor.check_types", checkTypes)
	viper.Set("refactor.show_diff", showDiff)

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
		// Walk directory to find text files
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

			// Check if file is likely a text file
			if isLikelyTextFile(path) {
				filesToProcess = append(filesToProcess, path)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", targetPath, err)
		}
	} else {
		// Single file
		if !isLikelyTextFile(targetPath) {
			return fmt.Errorf("file %s appears to be a binary file", targetPath)
		}
		filesToProcess = []string{targetPath}
	}

	if len(filesToProcess) == 0 {
		return fmt.Errorf("no text files found to process in %s", targetPath)
	}

	// Process each file
	var results []*refactor.RefactorResult
	for _, file := range filesToProcess {
		if verbose {
			log.Printf("Processing %s...", file)
		}

		result, err := refactor.ProcessFileRefactor(cmd.Context(), cfg, llmClient, file, scope, prompt)
		if err != nil {
			log.Printf("Error processing %s: %v", file, err)
			continue
		}
		results = append(results, result)
	}

	// Summarize results
	var (
		totalFiles     = len(results)
		changedFiles   = 0
		errorFiles     = 0
		skippedFiles   = 0
		typeErrors     = 0
		appliedChanges = 0
	)

	for _, result := range results {
		if result.LLMError != nil || result.TypeCheckError != nil || result.EditApplyError != nil {
			errorFiles++
			continue
		}
		if !result.NeedsConfirmation {
			skippedFiles++
			continue
		}
		changedFiles++
		if !result.TypeCheckOK {
			typeErrors++
		}
		if !dryRun && (autoApply || confirmChanges(result)) {
			result.Apply = true
			appliedChanges++
		}
	}

	// Apply changes
	if !dryRun {
		for _, result := range results {
			if result.Apply {
				if err := os.WriteFile(result.FilePath, []byte(result.ProposedContent), 0644); err != nil {
					log.Printf("Error writing changes to %s: %v", result.FilePath, err)
					errorFiles++
					appliedChanges--
				}
			}
		}
	}

	// Print summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total files processed: %d\n", totalFiles)
	fmt.Printf("Files with changes: %d\n", changedFiles)
	fmt.Printf("Files with errors: %d\n", errorFiles)
	fmt.Printf("Files skipped: %d\n", skippedFiles)
	fmt.Printf("Files with type errors: %d\n", typeErrors)
	if !dryRun {
		fmt.Printf("Changes applied: %d\n", appliedChanges)
	}

	return nil
}

// shouldSkipPath returns true if the path should be skipped
func shouldSkipPath(path string) bool {
	// Skip common build and dependency directories
	skipDirs := []string{
		"node_modules",
		"dist",
		"build",
		"bin",
		"obj",
		"target",
		"vendor",
		".git",
	}

	// Skip common binary and generated files
	skipExtensions := []string{
		".exe", ".dll", ".so", ".dylib", // Binaries
		".jpg", ".jpeg", ".png", ".gif", ".ico", // Images
		".pdf", ".doc", ".docx", // Documents
		".zip", ".tar", ".gz", ".7z", // Archives
		".min.js", ".min.css", // Minified files
	}

	// Check directory parts
	parts := strings.Split(path, string(os.PathSeparator))
	for _, part := range parts {
		for _, skip := range skipDirs {
			if part == skip {
				return true
			}
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	for _, skip := range skipExtensions {
		if ext == skip {
			return true
		}
	}

	return false
}

// isLikelyTextFile returns true if the file is likely a text file
func isLikelyTextFile(path string) bool {
	// Common text file extensions
	textExtensions := []string{
		// Programming languages
		".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".rb", ".php", ".java", ".cs", ".cpp", ".c", ".h",
		// Web
		".html", ".css", ".scss", ".sass", ".less", ".json", ".xml", ".yaml", ".yml",
		// Config
		".toml", ".ini", ".env", ".conf", ".config",
		// Documentation
		".md", ".txt", ".rst", ".adoc",
		// Shell
		".sh", ".bash", ".zsh", ".fish",
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	// For files without extensions or unknown extensions,
	// we could add more sophisticated checks here (e.g., reading first few bytes)
	// but for now, we'll be conservative and only process known text files
	return false
}

func confirmChanges(result *refactor.RefactorResult) bool {
	fmt.Printf("\nApply changes to %s? [y/N] ", result.FilePath)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}
