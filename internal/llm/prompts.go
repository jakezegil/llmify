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

const docsUpdatePromptTemplate = `
You are an expert technical writer tasked with updating documentation based on code changes.
Analyze the following code changes (provided as a git diff) and the existing documentation section.

Update the documentation provided below ONLY IF NECESSARY to accurately reflect the code changes.
Focus on:
- Changes to function signatures, parameters, or return types.
- Added or removed features relevant to the documentation.
- Changes in usage examples.
- Clarifications needed based on the code modifications.

If the documentation section does not need any updates based on the provided diff, respond with the exact phrase: "NO_UPDATE_NEEDED" and nothing else.

Otherwise, provide the COMPLETE, updated documentation section. Do NOT just describe the changes; output the full modified text.

Here is the git diff of the code changes:
--- DIFF START ---
%s
--- DIFF END ---

Here is the CURRENT documentation section to update:
--- DOCS START ---
%s
--- DOCS END ---

Provide the updated documentation section or "NO_UPDATE_NEEDED":
`

func CreateCommitPrompt(diff string, context string) string {
	return fmt.Sprintf(commitPromptTemplate, diff)
}

func CreateDocsUpdatePrompt(diff string, docContent string) string {
	return fmt.Sprintf(docsUpdatePromptTemplate, diff, docContent)
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
