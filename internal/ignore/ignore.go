package ignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreMatcher handles ignore patterns from .gitignore and .llmignore files
type IgnoreMatcher struct {
	patterns []string
}

// NewIgnoreMatcher creates a new IgnoreMatcher with the given patterns
func NewIgnoreMatcher(patterns []string) *IgnoreMatcher {
	return &IgnoreMatcher{
		patterns: patterns,
	}
}

// LoadIgnoreFile loads ignore patterns from a file
func LoadIgnoreFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ignore file %s: %w", path, err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ignore file %s: %w", path, err)
	}
	return patterns, nil
}

// ShouldIgnore checks if a path should be ignored based on the patterns
func (m *IgnoreMatcher) ShouldIgnore(path string) bool {
	// Convert path to use forward slashes for consistency
	path = filepath.ToSlash(path)

	for _, pattern := range m.patterns {
		// Handle negated patterns
		if strings.HasPrefix(pattern, "!") {
			negatedPattern := pattern[1:]
			if matchPattern(path, negatedPattern) {
				return false
			}
			continue
		}

		// Special case handling for explicit directory patterns
		if strings.HasSuffix(pattern, "/") {
			dirPattern := pattern[:len(pattern)-1]
			if path == dirPattern || strings.HasPrefix(path, dirPattern+"/") {
				return true
			}
			continue
		}

		// Check if pattern matches exactly
		if matchPattern(path, pattern) {
			return true
		}

		// Check if pattern matches any part of the path
		pathParts := strings.Split(path, "/")
		for i := range pathParts {
			subPath := strings.Join(pathParts[i:], "/")
			if matchPattern(subPath, pattern) {
				return true
			}
		}
	}
	return false
}

// matchPattern checks if a path matches a pattern
func matchPattern(path, pattern string) bool {
	// Direct equality
	if path == pattern {
		return true
	}

	// Handle wildcards with filepath.Match
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}

	// Handle directory wildcards (e.g., "dir/**")
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	// Handle file patterns that should match in any directory
	if strings.HasPrefix(pattern, "**/*.") {
		suffix := pattern[4:]
		return strings.HasSuffix(path, suffix)
	}

	// Handle filename patterns
	if !strings.Contains(pattern, "/") {
		return filepath.Base(path) == pattern
	}

	return false
}

// AddPattern adds a new pattern to the matcher
func (m *IgnoreMatcher) AddPattern(pattern string) {
	m.patterns = append(m.patterns, pattern)
}

// AddPatterns adds multiple patterns to the matcher
func (m *IgnoreMatcher) AddPatterns(patterns []string) {
	m.patterns = append(m.patterns, patterns...)
}

// GetPatterns returns all patterns in the matcher
func (m *IgnoreMatcher) GetPatterns() []string {
	return m.patterns
}
