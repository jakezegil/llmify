package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gitignore "github.com/sabhiram/go-gitignore"
)

// CrawlResult holds the results of the crawl operation.
type CrawlResult struct {
	IncludedFiles []string // List of relative paths to include
	FileTree      string   // Generated file tree string
	ExcludedCount int      // Count of files/dirs excluded
	IncludedCount int      // Count of files/dirs included (in tree)
}

// LoadIgnoreMatcher loads ignore rules from .gitignore and .llmignore.
func LoadIgnoreMatcher(rootDir string, useGitignore bool, useLLMIgnore bool) (*gitignore.GitIgnore, error) {
	var patterns []string
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	llmignorePath := filepath.Join(rootDir, ".llmignore")

	readFileLines := func(path string) ([]string, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return []string{}, nil // File not existing is not an error here
			}
			return nil, fmt.Errorf("reading ignore file %s: %w", path, err)
		}
		return strings.Split(string(content), "\n"), nil
	}

	if useGitignore {
		gitLines, err := readFileLines(gitignorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err) // Log error but continue
		} else {
			patterns = append(patterns, gitLines...)
		}
	}

	if useLLMIgnore {
		llmLines, err := readFileLines(llmignorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err) // Log error but continue
		} else {
			patterns = append(patterns, llmLines...)
		}
	}

	// Add common defaults that should always be ignored
	patterns = append(patterns,
		".git/",         // Crucial
		"__pycache__/",  // Python cache
		"node_modules/", // Node.js dependencies
		"vendor/",       // Go dependencies (often)
		"build/",        // Common build output
		"dist/",         // Common distribution output
		"target/",       // Common build output (Java/Rust)
		"*.pyc",         // Python bytecode
		"*.pyo",
		"*.class",   // Java bytecode
		"*.log",     // Log files
		"*.swp",     // Vim swap files
		".DS_Store", // macOS metadata
		"Thumbs.db", // Windows metadata
		// Add more common temporary/build/cache files if needed
	)

	// Ensure rootDir is absolute for reliable matching
	_, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for root %s: %w", rootDir, err)
	}

	// go-gitignore expects patterns relative to the root where .gitignore would be
	ignorer := gitignore.CompileIgnoreLines(patterns...)
	// Note: The go-gitignore library doesn't have AddPatterns method
	// We're already compiling with patterns relative to the root

	return ignorer, nil
}

// CrawlProject walks the directory, applies filters, and gathers content.
func CrawlProject(
	rootDir string,
	outputFilename string,
	targetPathRel string, // New: Relative path to filter by
	ignorer *gitignore.GitIgnore,
	cmdExcludes []string, // Patterns from command line --exclude
	cmdIncludes []string, // Patterns from command line --include
	maxDepth int,
	excludeBinary bool,
	verbose bool,
) (*CrawlResult, error) {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for %s: %w", rootDir, err)
	}
	absOutputFilename := filepath.Join(absRootDir, outputFilename) // Get absolute path for output file

	// Determine absolute target path if provided
	var absTargetPath string
	var isTargetPathDir bool
	if targetPathRel != "" {
		absTargetPath = filepath.Join(absRootDir, targetPathRel)
		targetInfo, err := os.Stat(absTargetPath)
		// We assume stat worked because main.go checked it
		if err != nil {
			// This should ideally not happen due to checks in main.go, but handle defensively
			return nil, fmt.Errorf("cannot stat target path %s during crawl: %w", absTargetPath, err)
		}
		isTargetPathDir = targetInfo.IsDir()
	}

	// Compile command-line patterns (using gitignore syntax for simplicity)
	excludeMatcher := gitignore.CompileIgnoreLines(cmdExcludes...)
	includeMatcher := gitignore.CompileIgnoreLines(cmdIncludes...) // Note: includes need careful handling

	includedFiles := []string{}
	includedPathsForTree := make(map[string]os.DirEntry) // For building the tree
	excludedCount := 0
	includedCount := 0

	walkErr := filepath.WalkDir(absRootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Error accessing a file/directory, report and potentially skip
			fmt.Fprintf(os.Stderr, "Warning: Error accessing %s: %v\n", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir // Skip contents of this directory
			}
			return nil // Skip this file/entry
		}

		// Use absolute path for most checks
		absPath := path // WalkDir provides absolute paths if root is absolute

		// --- Filtering Logic ---

		// 0. Get relative path for matching against *patterns* and for final output list
		relPath, err := filepath.Rel(absRootDir, absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not get relative path for %s: %v\n", absPath, err)
			return nil // Skip if relative path fails
		}
		if relPath == "." { // Skip the root directory itself from inclusion checks, but don't skip walk
			return nil
		}

		// A. Check --path filter FIRST if provided
		if absTargetPath != "" {
			isInsideTargetPath := false
			if absPath == absTargetPath {
				isInsideTargetPath = true // Exact match (could be file or dir)
			} else if isTargetPathDir && strings.HasPrefix(absPath, absTargetPath+string(filepath.Separator)) {
				isInsideTargetPath = true // Path is inside the target directory
			}

			if !isInsideTargetPath {
				if verbose {
					fmt.Printf("Exclude (outside --path %s): %s\n", targetPathRel, relPath)
				}
				excludedCount++
				// Optimization: If it's a directory not matching the target prefix, skip it entirely
				if d.IsDir() && !strings.HasPrefix(absTargetPath, absPath+string(filepath.Separator)) {
					return filepath.SkipDir
				}
				return nil // Skip this file/entry
			}
		}

		// Ensure paths use forward slashes for consistent matching with gitignore patterns
		matchPath := filepath.ToSlash(relPath)
		if d.IsDir() {
			matchPath += "/" // Append slash for directories as gitignore patterns often expect
		}

		// 1. Check depth limit (relative path parts + 1 for root)
		if maxDepth > 0 {
			depth := len(strings.Split(filepath.ToSlash(relPath), "/"))
			// Adjust depth logic slightly: root is depth 0, its children are depth 1
			if depth > maxDepth {
				if verbose {
					fmt.Printf("Exclude (depth > %d): %s\n", maxDepth, relPath)
				}
				excludedCount++
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 2. Never include the output file itself
		if absPath == absOutputFilename {
			if verbose {
				fmt.Printf("Exclude (output file): %s\n", relPath)
			}
			excludedCount++
			if d.IsDir() {
				return filepath.SkipDir
			} // Should not happen for the output file, but check anyway
			return nil
		}

		// 3. Check .gitignore / .llmignore patterns
		if ignorer != nil && ignorer.MatchesPath(path) {
			// Is there an explicit command-line include that overrides this?
			isIncluded := includeMatcher != nil && includeMatcher.MatchesPath(path) // Check if explicitly included
			if !isIncluded {
				if verbose {
					fmt.Printf("Exclude (ignore file): %s\n", relPath)
				}
				excludedCount++
				if d.IsDir() {
					return filepath.SkipDir
				} // Skip ignored directories
				return nil
			}
			if verbose {
				fmt.Printf("Override ignore (cmd include): %s\n", relPath)
			}
		}

		// 4. Check command-line exclude patterns
		if excludeMatcher != nil && excludeMatcher.MatchesPath(path) {
			// Is there an explicit command-line include that overrides this?
			isIncluded := includeMatcher != nil && includeMatcher.MatchesPath(path) // Check if explicitly included
			if !isIncluded {
				if verbose {
					fmt.Printf("Exclude (cmd exclude): %s\n", relPath)
				}
				excludedCount++
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if verbose {
				fmt.Printf("Override cmd exclude (cmd include): %s\n", relPath)
			}
		}

		// 5. If it's a file, check if it's binary (unless overridden by include)
		if !d.IsDir() {
			isIncluded := includeMatcher != nil && includeMatcher.MatchesPath(path)
			if !isIncluded && excludeBinary {
				isText, textCheckErr := IsLikelyTextFile(path)
				if textCheckErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not check file type for %s: %v\n", path, textCheckErr)
					// Decide whether to include or exclude on error - let's exclude by default
					isText = false
				}
				if !isText {
					if verbose {
						fmt.Printf("Exclude (binary): %s\n", relPath)
					}
					excludedCount++
					return nil // Skip binary files
				}
			}
		}

		// --- If we reach here, the path should be included ---
		if verbose {
			fmt.Printf("Include: %s\n", relPath)
		}
		includedPathsForTree[path] = d // Store entry for tree building
		includedCount++
		if !d.IsDir() {
			includedFiles = append(includedFiles, relPath) // Add relative path to list
		}

		return nil // Continue walking
	})

	if walkErr != nil {
		return nil, fmt.Errorf("error during directory walk: %w", walkErr)
	}

	// Sort included files for consistent output
	sort.Strings(includedFiles)

	// Generate the file tree using only the included paths
	includeCriteria := func(p string, d os.DirEntry) bool {
		_, exists := includedPathsForTree[p]
		return exists
	}
	treeString, err := GenerateFileTree(absRootDir, includeCriteria, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("generating file tree: %w", err)
	}

	result := &CrawlResult{
		IncludedFiles: includedFiles,
		FileTree:      treeString,
		ExcludedCount: excludedCount,
		IncludedCount: includedCount, // This counts files and dirs added to the tree map
	}

	return result, nil
}

// BuildOutputContent combines tree and file contents into the final string.
func BuildOutputContent(rootDir string, result *CrawlResult, includeHeader bool) (string, error) {
	var builder strings.Builder
	absRootDir, _ := filepath.Abs(rootDir) // Assume rootDir is valid now

	// Optional Header
	if includeHeader {
		builder.WriteString("============================================================\n")
		builder.WriteString(fmt.Sprintf("Project Root: %s\n", absRootDir))
		builder.WriteString(fmt.Sprintf("Generated At: %s\n", time.Now().Format(time.RFC3339)))
		// Could add command-line args used here too
		builder.WriteString("============================================================\n\n")
	}

	// File Tree Section
	builder.WriteString("## File Tree Structure\n\n")
	builder.WriteString("```\n")
	builder.WriteString(result.FileTree)
	builder.WriteString("```\n\n")
	builder.WriteString("============================================================\n\n")

	// File Content Section
	builder.WriteString("## File Contents\n\n")

	separator := "\n\n---\n\n" // Separator between files

	for i, relPath := range result.IncludedFiles {
		fullPath := filepath.Join(absRootDir, relPath)
		content, err := ReadFileContent(fullPath)
		if err != nil {
			// Log error but try to continue with other files
			fmt.Fprintf(os.Stderr, "Warning: Failed to read content for %s: %v\n", relPath, err)
			content = fmt.Sprintf("Error reading file: %v", err) // Include error message in output
		}

		// Add file path header
		builder.WriteString(fmt.Sprintf("### File: %s\n\n", filepath.ToSlash(relPath))) // Use forward slashes
		builder.WriteString("```")
		// Try to detect language from extension for syntax highlighting hint
		ext := strings.TrimPrefix(filepath.Ext(relPath), ".")
		if ext != "" {
			builder.WriteString(ext) // e.g., ```go, ```python
		}
		builder.WriteString("\n")
		builder.WriteString(content)
		builder.WriteString("\n```") // End code block

		if i < len(result.IncludedFiles)-1 {
			builder.WriteString(separator)
		}
	}

	return builder.String(), nil
}
