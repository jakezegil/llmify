package refactor

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jake/llmify/internal/config"
	"github.com/jake/llmify/internal/diff"
	"github.com/jake/llmify/internal/editor"
	"github.com/jake/llmify/internal/llm"
	"github.com/spf13/viper"
)

type RefactorResult struct {
	FilePath          string
	OriginalContent   string
	ProposedContent   string // Empty if no change proposed or error
	TypeCheckOK       bool
	TypeCheckOutput   string
	LLMError          error         // Error during LLM generation
	TypeCheckError    error         // Error *running* type check
	NeedsConfirmation bool          // Does this specific file need user confirmation?
	Apply             bool          // Should changes be applied (set after confirmation)?
	Edits             []editor.Edit // The parsed edits from the LLM
	IsFullReplacement bool          // Whether the LLM provided a full file replacement
	EditApplyError    error         // Error applying the edits
}

// ProcessFileRefactor handles the refactoring logic for a single file.
func ProcessFileRefactor(ctx context.Context, cfg *config.Config, llmClient llm.LLMClient, filePath string, scope string, userPrompt string) (*RefactorResult, error) {
	verbose := viper.GetBool("verbose")
	result := &RefactorResult{
		FilePath:          filePath,
		NeedsConfirmation: true, // Default to needing confirmation unless no changes
		Apply:             false,
	}

	// 1. Read File Content
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return result, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	result.OriginalContent = string(contentBytes)

	// 2. Identify Target Snippet & Context (Simplified for now)
	// TODO: Implement proper scope parsing (function/class/lines)
	// TODO: Implement context gathering (imports, related types)
	targetCode := result.OriginalContent                               // Default to whole file
	contextSnippet := "Imports:\n" + extractImports(targetCode) + "\n" // Basic context

	if scope != "" && verbose {
		log.Printf("Scope '%s' specified, but snippet extraction not yet implemented. Using full file.", scope)
		// Here you would add logic to extract the specific lines/function/class
		// and potentially gather more targeted context.
	}

	// 3. Call LLM
	refactorModel := cfg.LLM.Model // TODO: Allow specific refactor model override

	// Create refactoring prompt directly
	prompt := fmt.Sprintf(`
You are an expert developer specializing in safe and effective code refactoring.
Your task is to refactor the provided code snippet based on the user's request, ensuring correctness and maintaining necessary imports.

USER'S REFACTORING GOAL:
%s

CONTEXT (Imports, Type Definitions, Related Code - May be incomplete):
--- CONTEXT START ---
%s
--- CONTEXT END ---

TARGET CODE SNIPPET (or Full File Content):
--- TARGET CODE START ---
%s
--- TARGET CODE END ---

IMPORTANT INSTRUCTIONS:
1. Provide ONLY the complete refactored code with no additional text.
2. Do NOT include markdown code blocks or triple backticks.
3. Do NOT include any explanations or comments about your changes.
4. If refactoring the entire file, include necessary import statements.
5. The output should be valid code that can be directly saved to a file.
6. Do NOT add any unnecessary imports or modules.
7. Preserve existing imports and only add new ones if absolutely necessary.
8. Preserve original indentation and formatting.

OUTPUT FORMAT:
If the changes are targeted and specific, provide them in one of these formats:

1. For replacing existing code:
--- LLMIFY REPLACE START ---
<<< ORIGINAL >>>
[The exact lines to be replaced]
<<< REPLACEMENT >>>
[The new lines to replace the original block]
--- LLMIFY REPLACE END ---

2. For inserting new code:
--- LLMIFY INSERT_AFTER START ---
<<< CONTEXT_LINE >>>
[The exact line content *immediately preceding* the desired insertion point]
<<< INSERTION >>>
[The new lines to be inserted]
--- LLMIFY INSERT_AFTER END ---

3. For deleting code:
--- LLMIFY DELETE START ---
<<< CONTENT >>>
[The exact lines to be deleted]
--- LLMIFY DELETE END ---

If the changes are too extensive or complex for the edit format, provide the complete updated content enclosed in triple backticks:
`+"```"+`language
[Complete updated content]
`+"```"+`
`, userPrompt, contextSnippet, targetCode)

	// Get timeout from command line flags with fallback to a much larger value
	timeoutSeconds := viper.GetInt("llm.timeout_seconds")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300 // 5 minutes default if not specified
	}

	// Create a more generous timeout for the LLM call
	timeout := time.Duration(timeoutSeconds) * time.Second

	// Create a context with timeout, but use a background context if timeout is very short
	var llmCtx context.Context
	var cancel context.CancelFunc

	if timeout < 10*time.Second {
		// If timeout is suspiciously short, use a more reasonable value
		log.Printf("Warning: Specified timeout %d seconds is very short, using 3 minutes instead", timeoutSeconds)
		llmCtx, cancel = context.WithTimeout(ctx, 3*time.Minute)
	} else {
		llmCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if verbose {
		log.Printf("Generating refactoring for %s using model %s (timeout: %v)...",
			filePath, refactorModel, timeout)
	}
	proposedCode, llmErr := llmClient.Generate(llmCtx, prompt, refactorModel)
	result.LLMError = llmErr
	if llmErr != nil {
		log.Printf("Error generating refactoring for %s: %v", filePath, llmErr)
		result.NeedsConfirmation = false // Don't confirm if LLM failed
		return result, nil               // Don't return error, just store it in result
	}

	// Parse the LLM response for edits or full file content
	edits, fullContent, err := editor.ParseLLMResponse(proposedCode)
	if err != nil {
		log.Printf("Error parsing LLM response for %s: %v", filePath, err)
		result.NeedsConfirmation = false
		return result, nil
	}

	// If we got a full file replacement
	if fullContent != "" {
		result.IsFullReplacement = true
		result.ProposedContent = fullContent
	} else if len(edits) > 0 {
		// Apply the parsed edits
		result.Edits = edits
		newContent, err := editor.ApplyEdits(result.OriginalContent, edits)
		if err != nil {
			log.Printf("Error applying edits for %s: %v", filePath, err)
			result.EditApplyError = err
			result.NeedsConfirmation = false
			return result, nil
		}
		result.ProposedContent = newContent
	} else {
		// No changes proposed
		log.Printf("No changes proposed for %s.", filePath)
		result.ProposedContent = result.OriginalContent
		result.NeedsConfirmation = false
		result.TypeCheckOK = true
		result.TypeCheckOutput = "No changes proposed by LLM."
		return result, nil
	}

	// Handle LLM potentially just saying "no changes needed" or similar
	if len(result.ProposedContent) < 10 || strings.Contains(strings.ToLower(result.ProposedContent), "no changes needed") || result.ProposedContent == targetCode {
		log.Printf("LLM indicated no changes needed or returned original code for %s.", filePath)
		result.ProposedContent = result.OriginalContent // Ensure it matches original
		result.NeedsConfirmation = false
		result.TypeCheckOK = true
		result.TypeCheckOutput = "No changes proposed by LLM."
		return result, nil
	}

	// 4. Run Type Check (if enabled)
	checkTypes := viper.GetBool("refactor.check_types") // Assuming flag sets this
	if checkTypes {
		ok, output, checkErr := CheckTypeScriptTypes(filePath, result.ProposedContent)
		result.TypeCheckOK = ok
		result.TypeCheckOutput = output
		result.TypeCheckError = checkErr
		if checkErr != nil {
			log.Printf("Error running type check for %s: %v", filePath, checkErr)
			// Should we prevent applying changes if the check itself failed? Probably.
			result.NeedsConfirmation = false // Don't confirm if type check failed to run
			return result, nil
		}
		if !ok && verbose {
			log.Printf("Type check FAILED for proposed changes to %s.", filePath)
		}
	} else {
		result.TypeCheckOK = true // Assume OK if check is disabled
		result.TypeCheckOutput = "Type check skipped."
		if verbose {
			log.Printf("Skipping type check for %s as requested.", filePath)
		}
	}

	// 5. Display Diff (if enabled and changes proposed)
	showDiff := viper.GetBool("refactor.show_diff") // Assuming flag sets this
	if showDiff && result.NeedsConfirmation {       // Only show diff if there are changes to confirm
		fmt.Printf("\n--- Proposed Changes for: %s ---\n", filePath)
		diff.ShowDiff(result.OriginalContent, result.ProposedContent)
		fmt.Println("------------------------------------")
		fmt.Printf("Type Check Result: %s\n", result.TypeCheckOutput)
		if !result.TypeCheckOK {
			fmt.Println("\033[0;31mWARNING: Type errors detected!\033[0m")
		}
		fmt.Println("------------------------------------")
	}

	return result, nil
}

// extractImports is a very basic helper (replace with proper parsing if needed)
func extractImports(code string) string {
	var imports []string
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export * from") {
			imports = append(imports, line)
		}
	}
	return strings.Join(imports, "\n")
}
