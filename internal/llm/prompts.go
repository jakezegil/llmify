package llm

import (
	"fmt"
	"strings"
)

const commitPromptTemplate = `
You are an expert programmer and Git user, tasked with writing a detailed and clear commit message. Be cheeky sometimes.
Analyze the following code changes (provided as a git diff) and the context of the changed files.

Follow the Conventional Commits specification (https://www.conventionalcommits.org/).
The commit message should have:
1. A type prefix (e.g., feat, fix, refactor, chore, docs, style, test, perf).
2. A concise subject line summarizing the change (imperative mood, lowercase).
3. A blank line.
4. A detailed body explaining the 'what' and 'why' of the changes. Be specific. Mention key functions/files modified and the reasoning. If it fixes an issue, reference it.

Here is the git diff:
--- DIFF START ---
%s
--- DIFF END ---

Generate the commit message now:
`

// docsUpdatePromptTemplate is used for updating documentation based on code changes
const docsUpdatePromptTemplate = `
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
` + "```" + `markdown
[Complete updated content]
` + "```" + `
`

// refactorPromptTemplate is used for refactoring code snippets
const refactorPromptTemplate = `
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
` + "```" + `language
[Complete updated content]
` + "```" + `
`

func CreateCommitPrompt(diff string, context string) string {
	return fmt.Sprintf(commitPromptTemplate, diff)
}

func CreateDocsUpdatePrompt(diff string, docContent string) string {
	return fmt.Sprintf(docsUpdatePromptTemplate, diff, docContent)
}

func CreateRefactorPrompt(userGoal, context, targetCode string) string {
	return fmt.Sprintf(refactorPromptTemplate, userGoal, context, targetCode)
}

// Helper function to check LLM response for docs update
func NeedsDocUpdate(response string) (bool, string) {
	trimmedResponse := strings.TrimSpace(response)
	if trimmedResponse == "NO_UPDATE_NEEDED" {
		return false, ""
	}
	// Assume any other non-empty response is the updated content
	return len(trimmedResponse) > 0, trimmedResponse
}
