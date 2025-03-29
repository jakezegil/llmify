package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/cmd"
	"github.com/jake/llmify/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	verboseFlag   bool
	includeHeader bool
	rootDir       string // Root directory for the crawl
	llmTimeout    int    // Timeout in seconds for LLM API calls
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "llmify [directory]",
	Short: "Tools to optimize codebases for LLMs and assist with development workflows.",
	Long: `llmify provides tools to prepare codebases for Large Language Models (LLMs)
and leverage LLMs for tasks like generating commit messages.

Default behavior (without subcommand):
Crawls a project directory, respects ignore rules, and creates a single text file
('llm.txt') containing a file tree and relevant file contents for LLM context.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Bind the verbose flag to viper BEFORE loading config
		viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
		viper.BindPFlag("llm.timeout_seconds", cmd.PersistentFlags().Lookup("llm-timeout")) // Bind timeout config

		// Load configuration once for all commands
		if err := config.LoadConfig(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		// Set verbose based on viper AFTER loading config/env vars
		verboseFlag = viper.GetBool("verbose") // Update global var if needed elsewhere
		return nil
	},
	Args: cobra.MaximumNArgs(1), // Root command still takes optional directory for default action
	RunE: runRootCmd,            // Separate function for root command's logic
}

// runRootCmd contains the original logic of the root command (context dumping)
func runRootCmd(cmd *cobra.Command, args []string) error {
	// Check if a subcommand was executed, if so, don't run root logic
	// Cobra usually handles this, but an explicit check can be added if needed.
	// if cmd.HasSubCommands() && cmd.CalledAs() == "llmify" { ... }

	// --- Start of original RunE logic ---
	if verboseFlag { // Use the flag variable bound to viper
		fmt.Fprintln(os.Stderr, "Running default context generation...")
	}

	// Determine root directory (same as before)
	if len(args) > 0 {
		rootDir = args[0]
	} else {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	// Validate root directory (same as before)
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

	// Normalize target path (same as before)
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

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "Starting crawl in: %s\n", rootDir)
		if targetPath != "" {
			fmt.Fprintf(os.Stderr, "Filtering for specific path: %s (absolute: %s)\n", targetPath, absTargetPath)
		}
		fmt.Fprintf(os.Stderr, "Output file: %s\n", outputFile)
		fmt.Fprintf(os.Stderr, "Using .gitignore: %t\n", !noGitignore)
		fmt.Fprintf(os.Stderr, "Using .llmignore: %t\n", !noLLMignore)
		fmt.Fprintf(os.Stderr, "Excluding binary files: %t\n", excludeBinary)
		fmt.Fprintf(os.Stderr, "Max depth: %d (0 means unlimited)\n", maxDepth)
		fmt.Fprintf(os.Stderr, "Command excludes: %v\n", excludes)
		fmt.Fprintf(os.Stderr, "Command includes: %v\n", includes)
	}

	// --- Main Logic ---
	// 1. Load ignore rules
	ignorer, err := LoadIgnoreMatcher(rootDir, !noGitignore, !noLLMignore)
	if err != nil {
		return fmt.Errorf("failed to load ignore patterns: %w", err)
	}

	// 2. Crawl project
	crawlResult, err := CrawlProject(rootDir, outputFile, targetPath, ignorer, excludes, includes, maxDepth, excludeBinary, verboseFlag)
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
	// --- End of original RunE logic ---
	return nil
}

func init() {
	// Add global flags to rootCmd PersistentFlags
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose logging to stderr")
	rootCmd.PersistentFlags().IntVar(&llmTimeout, "llm-timeout", 120, "Timeout in seconds for LLM API calls")
	// Add other global flags (e.g., --config path, --provider, --model overrides) if desired

	// Flags specific to the root command (context dumping)
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "llm.txt", "Name of the output file for context dump")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{}, "Glob patterns to exclude for context dump")
	rootCmd.Flags().StringSliceVarP(&includes, "include", "i", []string{}, "Glob patterns to include for context dump (overrides excludes)")
	rootCmd.Flags().StringVarP(&targetPath, "path", "p", "", "Only include files/dirs within this path for context dump")
	rootCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "Max directory depth for context dump (0 for unlimited)")
	rootCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Do not use .gitignore rules for context dump")
	rootCmd.Flags().BoolVar(&noLLMignore, "no-llmignore", false, "Do not use .llmignore rules for context dump")
	rootCmd.Flags().BoolVar(&excludeBinary, "exclude-binary", true, "Attempt to exclude binary files for context dump")
	rootCmd.Flags().BoolVar(&includeHeader, "header", true, "Include a header in the context dump output file")

	// Add the new commit command
	rootCmd.AddCommand(cmd.CommitCmd)

	// Add the docs command
	rootCmd.AddCommand(cmd.DocsCmd)

	// Add other commands here later
}
