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
	"github.com/jake/llmify/internal/llm"
	"github.com/jake/llmify/internal/refactor"
	"github.com/jake/llmify/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	refactorScope        string
	refactorPrompt       string
	refactorCheckTypes   bool
	refactorNoCheckTypes bool
	refactorShowDiff     bool
	refactorNoDiff       bool
	refactorApply        bool
	refactorForce        bool
	refactorDryRun       bool
)

var RefactorCmd = &cobra.Command{
	Use:   "refactor <file_or_folder_path>",
	Short: "Refactor TypeScript code using an LLM based on a prompt.",
	Long: `Analyzes specified TypeScript code (file or folder), generates refactoring
suggestions based on your prompt, optionally performs type checking,
shows a diff, and allows interactive application of changes.`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Bind flags to viper for easy access in other packages
		viper.BindPFlag("refactor.check_types", cmd.Flags().Lookup("check-types"))
		viper.BindPFlag("refactor.show_diff", cmd.Flags().Lookup("show-diff"))

		// Validate mutually exclusive flags
		if refactorDryRun && refactorApply {
			return fmt.Errorf("--dry-run and --apply cannot be used together")
		}
		if refactorNoCheckTypes {
			viper.Set("refactor.check_types", false)
		}
		if refactorNoDiff {
			viper.Set("refactor.show_diff", false)
		}
		if refactorPrompt == "" {
			return fmt.Errorf("--prompt flag is required")
		}

		return nil
	},
	RunE: runRefactor,
}

func init() {
	RefactorCmd.Flags().StringVar(&refactorScope, "scope", "", `Scope of refactoring within file (e.g., "function MyFunc", "class MyClass", "lines 10-20"). Ignored for folders.`)
	RefactorCmd.Flags().StringVar(&refactorPrompt, "prompt", "", "Required: Describes the desired refactoring goal.")
	RefactorCmd.Flags().BoolVar(&refactorCheckTypes, "check-types", true, "Run tsc --noEmit to check for type errors after refactoring.")
	RefactorCmd.Flags().BoolVar(&refactorNoCheckTypes, "no-check-types", false, "Disable TypeScript type checking.")
	RefactorCmd.Flags().BoolVar(&refactorShowDiff, "show-diff", true, "Show a diff of the proposed changes.")
	RefactorCmd.Flags().BoolVar(&refactorNoDiff, "no-diff", false, "Do not show diffs of proposed changes.")
	RefactorCmd.Flags().BoolVar(&refactorApply, "apply", false, "Apply the proposed refactoring changes to the files.")
	RefactorCmd.Flags().BoolVarP(&refactorForce, "force", "f", false, "Skip confirmation prompts when applying changes.")
	RefactorCmd.Flags().BoolVar(&refactorDryRun, "dry-run", false, "Show proposed changes and type check results without applying.")

	// Mark prompt as required
	RefactorCmd.MarkFlagRequired("prompt")
}

func runRefactor(cmd *cobra.Command, args []string) error {
	targetPath := args[0]
	verbose := viper.GetBool("verbose")
	cfg := &config.GlobalConfig

	// --- Determine Target Files ---
	var targetFiles []string
	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("invalid target path %s: %w", targetPath, err)
	}

	if fileInfo.IsDir() {
		if refactorScope != "" && verbose {
			log.Printf("Warning: --scope '%s' ignored when targeting a directory.", refactorScope)
			refactorScope = "" // Clear scope for directory mode
		}
		if verbose {
			log.Printf("Target is a directory. Searching for TypeScript files in %s...", targetPath)
		}
		// Walk the directory - TODO: Respect .gitignore/.llmignore
		err = filepath.WalkDir(targetPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Include .ts and .tsx, exclude .d.ts and node_modules, .git etc.
			if !d.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) &&
				!strings.HasSuffix(path, ".d.ts") &&
				!strings.Contains(path, string(filepath.Separator)+"node_modules"+string(filepath.Separator)) &&
				!strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
				targetFiles = append(targetFiles, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan directory %s: %w", targetPath, err)
		}
		if len(targetFiles) == 0 {
			fmt.Printf("No TypeScript files found in directory: %s\n", targetPath)
			return nil
		}
		if verbose {
			log.Printf("Found %d TypeScript files to process.", len(targetFiles))
		}
	} else {
		// Target is a single file
		if !(strings.HasSuffix(targetPath, ".ts") || strings.HasSuffix(targetPath, ".tsx")) {
			return fmt.Errorf("target file %s is not a TypeScript file (.ts, .tsx)", targetPath)
		}
		targetFiles = []string{targetPath}
		if verbose {
			log.Printf("Target is a single file: %s", targetPath)
		}
	}

	// --- Initialize LLM Client ---
	llmClient, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// --- Process Files ---
	results := make([]*refactor.RefactorResult, 0, len(targetFiles))

	// Create a parent context with a reasonable timeout
	timeoutSeconds := viper.GetInt("llm.timeout_seconds")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300 // 5 minutes default
	}

	// If we have multiple files to process, allocate more time
	if len(targetFiles) > 1 {
		// Increase timeout proportionally to the number of files, with a reasonable cap
		maxMultiplier := 4 // Cap at 4x the base timeout
		fileMultiplier := float64(len(targetFiles))
		if fileMultiplier > float64(maxMultiplier) {
			fileMultiplier = float64(maxMultiplier)
		}
		timeoutSeconds = int(float64(timeoutSeconds) * fileMultiplier)
		if verbose {
			log.Printf("Processing %d files, increasing timeout to %d seconds", len(targetFiles), timeoutSeconds)
		}
	}

	parentCtx, parentCancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer parentCancel()

	for _, filePath := range targetFiles {
		if verbose {
			log.Printf("--- Processing file: %s ---", filePath)
		}

		// Check if parent context is already done
		if parentCtx.Err() != nil {
			log.Printf("Aborting processing: timeout exceeded for batch processing")
			break
		}

		result, _ := refactor.ProcessFileRefactor(parentCtx, cfg, llmClient, filePath, refactorScope, refactorPrompt)
		results = append(results, result)
	}

	// --- Summarize & Apply (If Applicable) ---
	fmt.Println("\n--- Refactoring Summary ---")
	filesToConfirm := []*refactor.RefactorResult{}
	canApplyCount := 0
	typeErrorCount := 0
	llmErrorCount := 0
	noChangeCount := 0

	for _, res := range results {
		status := "\033[0;32mOK\033[0m" // Green
		if res.LLMError != nil {
			status = "\033[0;31mLLM ERROR\033[0m" // Red
			llmErrorCount++
		} else if res.TypeCheckError != nil {
			status = "\033[0;31mTYPECHECK ERROR\033[0m" // Red
		} else if !res.TypeCheckOK {
			status = "\033[0;33mTYPE ERRORS\033[0m" // Yellow
			typeErrorCount++
		} else if res.ProposedContent == res.OriginalContent {
			status = "\033[0;90mNO CHANGE\033[0m" // Grey
			noChangeCount++
		}

		fmt.Printf("- %s: %s\n", res.FilePath, status)

		if res.NeedsConfirmation && !refactorDryRun && refactorApply {
			filesToConfirm = append(filesToConfirm, res)
			if res.TypeCheckOK {
				canApplyCount++
			}
		}
	}
	fmt.Println("-------------------------")

	if refactorDryRun {
		fmt.Println("Dry run complete. No changes were applied.")
		return nil
	}

	if !refactorApply {
		fmt.Println("Run with --apply to apply suggested changes.")
		return nil
	}

	// Apply changes (potentially with confirmation)
	appliedCount := 0
	skippedCount := 0
	forceApplyAll := refactorForce

	if len(filesToConfirm) == 0 {
		fmt.Println("No applicable changes found to apply.")
		return nil
	}

	fmt.Println("\n--- Applying Changes ---")
	for _, res := range filesToConfirm {
		applyThisFile := false
		if forceApplyAll {
			if res.TypeCheckOK {
				applyThisFile = true
				fmt.Printf("Applying changes to %s (forced, type check OK)...\n", res.FilePath)
			} else {
				fmt.Printf("Skipping force-apply for %s due to type check failures.\n", res.FilePath)
				skippedCount++
				continue
			}
		} else {
			// Individual confirmation
			prompt := fmt.Sprintf("Apply changes to %s?", res.FilePath)
			defaultChoice := "N"
			if res.TypeCheckOK {
				defaultChoice = "Y"
			} else {
				prompt = fmt.Sprintf("\033[0;33mWARNING: Type errors detected!\033[0m Apply changes to %s anyway?", res.FilePath)
			}

			confirmed, err := ui.Confirm(prompt, defaultChoice)
			if err != nil {
				log.Printf("Error during confirmation for %s: %v. Skipping file.", res.FilePath, err)
				skippedCount++
				continue
			}
			if confirmed {
				applyThisFile = true
				fmt.Printf("Applying changes to %s...\n", res.FilePath)
			} else {
				fmt.Printf("Skipping changes for %s.\n", res.FilePath)
				skippedCount++
			}
		}

		if applyThisFile {
			err := os.WriteFile(res.FilePath, []byte(res.ProposedContent), 0644)
			if err != nil {
				log.Printf("ERROR: Failed to write changes to %s: %v", res.FilePath, err)
			} else {
				appliedCount++
			}
		}
	}

	fmt.Println("----------------------")
	fmt.Printf("Applied changes to %d file(s).\n", appliedCount)
	if skippedCount > 0 {
		fmt.Printf("Skipped applying changes to %d file(s).\n", skippedCount)
	}

	return nil
}
