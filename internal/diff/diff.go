package diff

import (
	"fmt"
	"strings"
)

// ShowDiff displays a colorized diff between old and new content
func ShowDiff(oldContent, newContent string) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple diff algorithm - just show lines that are different
	// TODO: Implement a more sophisticated diff algorithm
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(oldLines) {
			// New lines
			fmt.Printf("\033[32m+ %s\033[0m\n", newLines[i])
			continue
		}
		if i >= len(newLines) {
			// Deleted lines
			fmt.Printf("\033[31m- %s\033[0m\n", oldLines[i])
			continue
		}

		if oldLines[i] != newLines[i] {
			fmt.Printf("\033[31m- %s\033[0m\n", oldLines[i])
			fmt.Printf("\033[32m+ %s\033[0m\n", newLines[i])
		} else {
			fmt.Printf("  %s\n", oldLines[i])
		}
	}
}
