package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateFileTree creates a string representation of the directory structure.
func GenerateFileTree(root string, includeCriteria func(path string, d os.DirEntry) bool, maxDepth int) (string, error) {
	var builder strings.Builder
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("getting absolute path for %s: %w", root, err)
	}
	builder.WriteString(fmt.Sprintf("%s/\n", filepath.Base(absRoot))) // Add root dir name

	err = walkTree(absRoot, "", true, &builder, includeCriteria, maxDepth, 0)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func walkTree(
	currentPath string,
	prefix string,
	isRootDir bool, // Flag to handle prefix differently for root's children
	builder *strings.Builder,
	includeCriteria func(path string, d os.DirEntry) bool,
	maxDepth int,
	currentDepth int,
) error {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	// Read directory entries
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		// Don't fail the whole process, just note the error
		fmt.Fprintf(os.Stderr, "Warning: Cannot read directory %s: %v\n", currentPath, err)
		return nil // Continue walking other parts
	}

	// Filter entries based on include criteria
	filteredEntries := []os.DirEntry{}
	for _, entry := range entries {
		fullPath := filepath.Join(currentPath, entry.Name())
		if includeCriteria(fullPath, entry) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Sort entries: directories first, then files, alphabetically
	sort.Slice(filteredEntries, func(i, j int) bool {
		infoI, errI := filteredEntries[i].Info()
		infoJ, errJ := filteredEntries[j].Info()
		if errI != nil || errJ != nil {
			// Handle error case if needed, maybe sort by name only
			return strings.ToLower(filteredEntries[i].Name()) < strings.ToLower(filteredEntries[j].Name())
		}
		if infoI.IsDir() != infoJ.IsDir() {
			return infoI.IsDir() // Directories first
		}
		return strings.ToLower(infoI.Name()) < strings.ToLower(infoJ.Name()) // Then sort by name
	})

	for i, entry := range filteredEntries {
		isLast := i == len(filteredEntries)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		// For the immediate children of the root, don't add the initial prefix
		currentPrefix := prefix
		if isRootDir {
			currentPrefix = ""
		}

		entryPath := filepath.Join(currentPath, entry.Name())

		if entry.IsDir() {
			builder.WriteString(fmt.Sprintf("%s%s%s/\n", currentPrefix, connector, entry.Name()))
			// Recurse into subdirectory
			err := walkTree(entryPath, currentPrefix+childPrefix, false, builder, includeCriteria, maxDepth, currentDepth+1)
			if err != nil {
				// Propagate error up if needed, or log and continue
				fmt.Fprintf(os.Stderr, "Warning: Error walking subdirectory %s: %v\n", entryPath, err)
			}
		} else {
			builder.WriteString(fmt.Sprintf("%s%s%s\n", currentPrefix, connector, entry.Name()))
		}
	}
	return nil
}
