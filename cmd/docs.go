package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/diff"
	"github.com/jake/llmify/internal/git"
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	docsStaged      bool
	docsCommits     int
	docsInteractive bool
	docsPath        string
	docsTargetFiles string
	docsPrompt      string
	docsForce       bool
	docsDryRun      bool
	docsNoDiff      bool
	docsStage       bool
	docsNoStage     bool
)

var DocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Update documentation based on code changes using an LLM.",
	Long: `Analyzes code changes (staged, recent commits, or specific commits) and updates
documentation files using an LLM. Can target specific docs or automatically discover
relevant documentation files.`,
	RunE: runDocs,
}

func init() {
	// Source of changes (mutually exclusive)
	DocsCmd.Flags().BoolVar(&docsStaged, "staged", true, "Analyze staged changes (default)")
	DocsCmd.Flags().IntVar(&docsCommits, "commits", 0, "Analyze changes from the last N commits")
	DocsCmd.Flags().BoolVar(&docsInteractive, "interactive", false, "Interactively select commits to analyze")

	// Scoping
	DocsCmd.Flags().StringVar(&docsPath, "path", "", "Only consider changes within this path")
	DocsCmd.Flags().StringVar(&docsTargetFiles, "target-docs", "", "Comma-separated list of specific documentation files to update")

	// Control
	DocsCmd.Flags().StringVar(&docsPrompt, "prompt", "", "Custom prompt to use for the LLM")
	DocsCmd.Flags().BoolVar(&docsForce, "force", false, "Apply changes without confirmation")
	DocsCmd.Flags().BoolVar(&docsDryRun, "dry-run", false, "Show proposed changes without applying them")
	DocsCmd.Flags().BoolVar(&docsNoDiff, "no-diff", false, "Don't show diffs of proposed changes")
	DocsCmd.Flags().BoolVar(&docsStage, "stage", true, "Stage updated documentation files")
	DocsCmd.Flags().BoolVar(&docsNoStage, "no-stage", false, "Don't stage updated documentation files")
}

func runDocs(cmd *cobra.Command, args []string) error {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// --- 1. Determine Change Source ---
	var gitDiff string
	var commitMessages []string
	var err error

	if docsInteractive {
		// TODO: Implement interactive commit selection
		return fmt.Errorf("interactive mode not yet implemented")
	} else if docsCommits > 0 {
		gitDiff, commitMessages, err = git.GetDiffFromCommits(docsCommits)
	} else {
		// Default to staged changes
		gitDiff, err = git.GetStagedDiff()
	}

	if err != nil {
		return fmt.Errorf("failed to get changes: %w", err)
	}

	// --- 2. Filter Diff if Path Specified ---
	if docsPath != "" {
		gitDiff, err = git.FilterDiffByPath(gitDiff, docsPath)
		if err != nil {
			return fmt.Errorf("failed to filter diff by path: %w", err)
		}
	}

	// --- 3. Identify Target Documentation Files ---
	var targetFiles []string
	if docsTargetFiles != "" {
		targetFiles = strings.Split(docsTargetFiles, ",")
	} else {
		targetFiles, err = git.FindRelevantDocs(gitDiff)
		if err != nil {
			return fmt.Errorf("failed to find relevant documentation files: %w", err)
		}
	}

	if len(targetFiles) == 0 {
		fmt.Println("No documentation files found to update.")
		return nil
	}

	if viper.GetBool("verbose") {
		log.Printf("Found %d documentation files to process", len(targetFiles))
	}

	// --- 4. Process Each Documentation File ---
	updatedFiles := make([]string, 0)
	skippedFiles := make([]string, 0)
	rejectedFiles := make([]string, 0)

	for _, docPath := range targetFiles {
		if viper.GetBool("verbose") {
			log.Printf("Processing documentation file: %s", docPath)
		}

		// Read current content
		content, err := git.ReadFile(docPath)
		if err != nil {
			log.Printf("Warning: could not read doc file %s: %v", docPath, err)
			continue
		}

		// Create LLM prompt
		prompt := llm.CreateDocsUpdatePrompt(gitDiff, string(content))
		if docsPrompt != "" {
			prompt = docsPrompt + "\n\n" + prompt
		}

		// Add commit messages to context if available
		if len(commitMessages) > 0 {
			prompt += "\n\nRecent commit messages:\n"
			for _, msg := range commitMessages {
				prompt += fmt.Sprintf("- %s\n", msg)
			}
		}

		// Get LLM response
		timeout := time.Duration(viper.GetInt("llm.timeout_seconds")) * time.Second
		if timeout == 0 {
			timeout = 120 * time.Second // Default to 2 minutes if not set
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		llmClient, err := llm.NewLLMClient(&config.GlobalConfig)
		if err != nil {
			return fmt.Errorf("failed to create LLM client: %w", err)
		}

		response, err := llmClient.Generate(ctx, prompt, config.GlobalConfig.Docs.Model)
		if err != nil {
			if err == context.DeadlineExceeded {
				log.Printf("Warning: LLM request timed out after %v for doc %s", timeout, docPath)
			} else {
				log.Printf("Warning: LLM failed to process doc %s: %v", docPath, err)
			}
			continue
		}

		// Check if update is needed
		needsUpdate, newContent := llm.NeedsDocUpdate(response)
		if !needsUpdate {
			skippedFiles = append(skippedFiles, docPath)
			continue
		}

		// Show diff unless suppressed
		if !docsNoDiff {
			fmt.Printf("\nProposed changes for %s:\n", docPath)
			diff.ShowDiff(string(content), newContent)
		}

		// Handle dry run
		if docsDryRun {
			continue
		}

		// Confirm unless forced
		if !docsForce {
			confirmed, err := ui.Confirm("Apply changes to "+docPath+"?", "Y/n")
			if err != nil {
				log.Printf("Warning: failed to get confirmation: %v", err)
				rejectedFiles = append(rejectedFiles, docPath)
				continue
			}
			if !confirmed {
				rejectedFiles = append(rejectedFiles, docPath)
				continue
			}
		}

		// Write changes
		err = git.WriteFile(docPath, []byte(newContent))
		if err != nil {
			log.Printf("Warning: failed to write updated doc %s: %v", docPath, err)
			continue
		}

		// Stage if enabled
		if docsStage && !docsNoStage {
			err = git.AddFiles([]string{docPath})
			if err != nil {
				log.Printf("Warning: failed to stage updated doc %s: %v", docPath, err)
			}
		}

		updatedFiles = append(updatedFiles, docPath)
	}

	// --- 5. Report Summary ---
	fmt.Printf("\nDocumentation Update Summary:\n")
	fmt.Printf("Updated: %d files\n", len(updatedFiles))
	fmt.Printf("Skipped: %d files\n", len(skippedFiles))
	fmt.Printf("Rejected: %d files\n", len(rejectedFiles))

	if len(updatedFiles) > 0 {
		fmt.Printf("\nUpdated files:\n")
		for _, file := range updatedFiles {
			fmt.Printf("- %s\n", file)
		}
	}

	if len(skippedFiles) > 0 {
		fmt.Printf("\nSkipped files (no updates needed):\n")
		for _, file := range skippedFiles {
			fmt.Printf("- %s\n", file)
		}
	}

	if len(rejectedFiles) > 0 {
		fmt.Printf("\nRejected files:\n")
		for _, file := range rejectedFiles {
			fmt.Printf("- %s\n", file)
		}
	}

	return nil
}
