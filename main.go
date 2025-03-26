package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	outputFile    string
	excludes      []string
	includes      []string
	targetPath    string
	maxDepth      int
	noGitignore   bool
	noLLMignore   bool
	excludeBinary bool
	verbose       bool
	includeHeader bool
	rootDir       string // Root directory for the crawl
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "llmify [directory]",
	Short: "Generates a concatenated text file (llm.txt) of project code and file tree for LLM context.",
	Long: `Crawls a project directory, respects .gitignore and .llmignore rules,
optionally filters by a specific sub-path, and creates a single text file
containing a file tree visualization followed by the contents of all
included text files. Useful for providing context to Large Language Models.

By default, it operates in the current working directory.`,
	Args: cobra.MaximumNArgs(1), // Allow zero or one argument (the directory)
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine root directory
		if len(args) > 0 {
			rootDir = args[0]
		} else {
			var err error
			rootDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
		}

		// Validate root directory exists
		info, err := os.Stat(rootDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("root directory not found: %s", rootDir)
			}
			return fmt.Errorf("failed to access root directory %s: %w", rootDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("specified path is not a directory: %s", rootDir)
		}

		// Normalize the target path relative to the root directory
		var absTargetPath string
		if targetPath != "" {
			// Clean the path and make it relative to rootDir
			targetPath = filepath.Clean(targetPath)
			// Ensure it doesn't try to escape the root directory
			if strings.HasPrefix(targetPath, ".."+string(filepath.Separator)) || targetPath == ".." {
				return fmt.Errorf("target path cannot be outside the root directory: %s", targetPath)
			}
			absTargetPath = filepath.Join(rootDir, targetPath)

			// Check if the target path exists
			_, err := os.Stat(absTargetPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("target path not found: %s", absTargetPath)
				}
				return fmt.Errorf("failed to access target path %s: %w", absTargetPath, err)
			}
		}

		if verbose {
			fmt.Printf("Starting crawl in: %s\n", rootDir)
			if targetPath != "" {
				fmt.Printf("Filtering for specific path: %s (absolute: %s)\n", targetPath, absTargetPath)
			}
			fmt.Printf("Output file: %s\n", outputFile)
			fmt.Printf("Using .gitignore: %t\n", !noGitignore)
			fmt.Printf("Using .llmignore: %t\n", !noLLMignore)
			fmt.Printf("Excluding binary files: %t\n", excludeBinary)
			fmt.Printf("Max depth: %d (0 means unlimited)\n", maxDepth)
			fmt.Printf("Command excludes: %v\n", excludes)
			fmt.Printf("Command includes: %v\n", includes)
		}

		// --- Main Logic ---
		// 1. Load ignore rules
		ignorer, err := LoadIgnoreMatcher(rootDir, !noGitignore, !noLLMignore)
		if err != nil {
			return fmt.Errorf("failed to load ignore patterns: %w", err)
		}

		// 2. Crawl project
		crawlResult, err := CrawlProject(rootDir, outputFile, targetPath, ignorer, excludes, includes, maxDepth, excludeBinary, verbose)
		if err != nil {
			return fmt.Errorf("failed to crawl project: %w", err)
		}

		// 3. Build the final output content
		outputContent, err := BuildOutputContent(rootDir, crawlResult, includeHeader)
		if err != nil {
			return fmt.Errorf("failed to build output content: %w", err)
		}

		// 4. Write to output file
		// Ensure the output path is relative to the CWD *unless* an absolute path was given
		outputPath := outputFile
		if !filepath.IsAbs(outputFile) {
			cwd, _ := os.Getwd() // Error getting CWD unlikely after initial checks
			outputPath = filepath.Join(cwd, outputFile)
		} else {
			// If outputFile is absolute, ensure rootDir comparison works correctly in CrawlProject
			// This was handled by using absolute paths internally in CrawlProject
		}

		err = WriteStringToFile(outputPath, outputContent)
		if err != nil {
			return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
		}

		fmt.Printf("Successfully generated LLM context file: %s\n", outputPath)
		fmt.Printf("Included %d files/directories in the context.\n", crawlResult.IncludedCount)
		if crawlResult.ExcludedCount > 0 {
			fmt.Printf("Excluded %d files/directories based on rules.\n", crawlResult.ExcludedCount)
		}
		return nil
	},
}

func init() {
	// Setup flags using Cobra
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "llm.txt", "Name of the output file")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{}, "Glob patterns to exclude (can be used multiple times)")
	rootCmd.Flags().StringSliceVarP(&includes, "include", "i", []string{}, "Glob patterns to include (overrides excludes, use carefully)")
	rootCmd.Flags().StringVarP(&targetPath, "path", "p", "", "Only include files/directories within this specific relative path")
	rootCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "Maximum directory depth to crawl (0 for unlimited)")
	rootCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Do not use .gitignore rules")
	rootCmd.Flags().BoolVar(&noLLMignore, "no-llmignore", false, "Do not use .llmignore rules")
	rootCmd.Flags().BoolVar(&excludeBinary, "exclude-binary", true, "Attempt to exclude binary files (based on content detection)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging to stderr")
	rootCmd.Flags().BoolVar(&includeHeader, "header", true, "Include a header with project root and timestamp in the output file")

	// Tie persistent flags if needed across subcommands (though we only have one here)
	// rootCmd.PersistentFlags().StringVar(...)
}
