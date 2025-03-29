package editor

import (
	"regexp"
	"strings"
)

// Edit represents a single edit operation suggested by the LLM
type Edit struct {
	Type             string // REPLACE, INSERT_AFTER, DELETE
	OriginalBlock    string // For REPLACE: the original lines to be replaced
	ReplacementBlock string // For REPLACE: the new lines to replace the original block
	ContextLine      string // For INSERT_AFTER: the line content immediately preceding the insertion point
	InsertionBlock   string // For INSERT_AFTER: the new lines to be inserted
	Content          string // For DELETE: the exact lines to be deleted
}

// Regular expressions for parsing LLM edit blocks
var (
	replaceRegex = regexp.MustCompile(`--- LLMIFY REPLACE START ---\n<<< ORIGINAL >>>\n(.*?)\n<<< REPLACEMENT >>>\n(.*?)\n--- LLMIFY REPLACE END ---`)
	insertRegex  = regexp.MustCompile(`--- LLMIFY INSERT_AFTER START ---\n<<< CONTEXT_LINE >>>\n(.*?)\n<<< INSERTION >>>\n(.*?)\n--- LLMIFY INSERT_AFTER END ---`)
	deleteRegex  = regexp.MustCompile(`--- LLMIFY DELETE START ---\n<<< CONTENT >>>\n(.*?)\n--- LLMIFY DELETE END ---`)
)

// ParseLLMResponse analyzes the LLM response to extract edits or detect full file content.
// Returns:
// - edits: slice of Edit structs if structured edits are found
// - fullContent: cleaned full file content if the response is a complete file
// - err: any error that occurred during parsing
func ParseLLMResponse(response string) ([]Edit, string, error) {
	// First check if this is a full file response
	if strings.HasPrefix(response, "```") {
		// Clean up the response to get just the content
		content := cleanLLMResponse(response)
		return nil, content, nil
	}

	// Look for structured edits
	var edits []Edit

	// Check for replace blocks
	replaceMatches := replaceRegex.FindAllStringSubmatch(response, -1)
	for _, match := range replaceMatches {
		if len(match) != 3 {
			continue
		}
		edits = append(edits, Edit{
			Type:             "REPLACE",
			OriginalBlock:    strings.TrimSpace(match[1]),
			ReplacementBlock: strings.TrimSpace(match[2]),
		})
	}

	// Check for insert blocks
	insertMatches := insertRegex.FindAllStringSubmatch(response, -1)
	for _, match := range insertMatches {
		if len(match) != 3 {
			continue
		}
		edits = append(edits, Edit{
			Type:           "INSERT_AFTER",
			ContextLine:    strings.TrimSpace(match[1]),
			InsertionBlock: strings.TrimSpace(match[2]),
		})
	}

	// Check for delete blocks
	deleteMatches := deleteRegex.FindAllStringSubmatch(response, -1)
	for _, match := range deleteMatches {
		if len(match) != 2 {
			continue
		}
		edits = append(edits, Edit{
			Type:    "DELETE",
			Content: strings.TrimSpace(match[1]),
		})
	}

	// If we found any edits, return them
	if len(edits) > 0 {
		return edits, "", nil
	}

	// If we didn't find any structured edits or full file content,
	// assume this is a full file response without code blocks
	return nil, cleanLLMResponse(response), nil
}

// ApplyEdits applies the parsed edits to the original content.
// Returns:
// - newContent: the content after applying all edits
// - err: any error that occurred during application
func ApplyEdits(originalContent string, edits []Edit) (string, error) {
	lines := strings.Split(originalContent, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		matched := false

		for _, edit := range edits {
			switch edit.Type {
			case "REPLACE":
				// Check if the next few lines match the original block
				originalLines := strings.Split(edit.OriginalBlock, "\n")
				if i+len(originalLines) <= len(lines) {
					block := strings.Join(lines[i:i+len(originalLines)], "\n")
					if block == edit.OriginalBlock {
						// Replace the block with the new content
						result = append(result, strings.Split(edit.ReplacementBlock, "\n")...)
						i += len(originalLines)
						matched = true
						break
					}
				}

			case "INSERT_AFTER":
				// Check if this line matches the context line
				if line == edit.ContextLine {
					result = append(result, line)
					result = append(result, strings.Split(edit.InsertionBlock, "\n")...)
					i++
					matched = true
					break
				}

			case "DELETE":
				// Check if the next few lines match the content to delete
				contentLines := strings.Split(edit.Content, "\n")
				if i+len(contentLines) <= len(lines) {
					block := strings.Join(lines[i:i+len(contentLines)], "\n")
					if block == edit.Content {
						i += len(contentLines)
						matched = true
						break
					}
				}
			}
		}

		if !matched {
			result = append(result, line)
			i++
		}
	}

	return strings.Join(result, "\n"), nil
}

// cleanLLMResponse removes markdown code fences and other formatting from LLM responses
func cleanLLMResponse(response string) string {
	// Trim leading/trailing whitespace
	cleaned := strings.TrimSpace(response)

	// Remove markdown code fences if present
	codeBlockPatterns := []string{
		"```typescript",
		"```tsx",
		"```js",
		"```javascript",
		"```ts",
		"```",
	}

	// Remove opening code fence
	for _, pattern := range codeBlockPatterns {
		if strings.HasPrefix(cleaned, pattern) {
			cleaned = strings.TrimPrefix(cleaned, pattern)
			cleaned = strings.TrimSpace(cleaned)
			break
		}
	}

	// Remove closing code fence if present
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

// limitString truncates a string to a maximum length for error messages
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
