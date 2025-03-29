package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jake/llmify/internal/crawler"
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
)

var rootCmd = &cobra.Command{
	Use:   "llmify",
	Short: "Generate a text representation of your project for LLM consumption",
	Long: `llmify is a tool that generates a text representation of your project,
suitable for consumption by large language models. It creates a structured
output that includes your project's file tree and file contents, while
respecting .gitignore and .llmignore patterns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine root directory
		rootDir := "."
		if len(args) > 0 {
			rootDir = args[0]
		}

		// Get absolute path
		absRootDir, err := filepath.Abs(rootDir)
		if err != nil {
			return fmt.Errorf("getting absolute path: %w", err)
		}

		// Validate root directory exists
		if _, err := os.Stat(absRootDir); os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", absRootDir)
		}

		// Create default .llmignore if needed
		if !noLLMignore {
			llmignorePath := filepath.Join(absRootDir, ".llmignore")
			if _, err := os.Stat(llmignorePath); os.IsNotExist(err) {
				if err := crawler.CreateDefaultLLMIgnoreFile(absRootDir); err != nil {
					if verbose {
						fmt.Printf("Warning: Failed to create default .llmignore: %v\n", err)
					}
				} else if verbose {
					fmt.Printf("Created default .llmignore file at %s\n", llmignorePath)
				}
			}
		}

		// Load ignore matcher
		matcher, err := crawler.LoadIgnoreMatcher(absRootDir, noGitignore, noLLMignore)
		if err != nil {
			return fmt.Errorf("loading ignore matcher: %w", err)
		}

		// Add exclude patterns
		for _, pattern := range excludes {
			matcher.AddPattern(pattern)
		}

		// Add include patterns
		for _, pattern := range includes {
			matcher.AddPattern("!" + pattern) // Negate pattern to include
		}

		// Always ensure .git and node_modules are ignored
		matcher.AddPattern(".git/")
		matcher.AddPattern(".git/**")
		matcher.AddPattern("node_modules/")
		matcher.AddPattern("node_modules/**")

		// Normalize target path
		absTargetPath := absRootDir
		if targetPath != "" {
			absTargetPath = filepath.Join(absRootDir, targetPath)
			if _, err := os.Stat(absTargetPath); os.IsNotExist(err) {
				return fmt.Errorf("target path does not exist: %s", targetPath)
			}
		}

		// Crawl project
		result, err := crawler.CrawlProject(absTargetPath, matcher, maxDepth, excludeBinary)
		if err != nil {
			return fmt.Errorf("crawling project: %w", err)
		}

		// Build output content
		content := crawler.BuildOutputContent(result, includeHeader)

		// Write to file
		if outputFile == "" {
			outputFile = "llm.txt"
		}
		if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}

		if verbose {
			fmt.Printf("Generated %s with %d files included and %d files excluded\n",
				outputFile, result.IncludedCount, result.ExcludedCount)
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: llm.txt)")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "e", nil, "Patterns to exclude (can be specified multiple times)")
	rootCmd.Flags().StringSliceVarP(&includes, "include", "i", nil, "Patterns to include (can be specified multiple times)")
	rootCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target path within project (default: project root)")
	rootCmd.Flags().IntVarP(&maxDepth, "depth", "d", 0, "Maximum directory depth (0 for unlimited)")
	rootCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Do not respect .gitignore")
	rootCmd.Flags().BoolVar(&noLLMignore, "no-llmignore", false, "Do not respect .llmignore")
	rootCmd.Flags().BoolVar(&excludeBinary, "exclude-binary", true, "Exclude binary files")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&includeHeader, "include-header", true, "Include header in output")

	// Add the commit command
	rootCmd.AddCommand(CommitCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
