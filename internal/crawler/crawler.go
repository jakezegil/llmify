package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/ignore"
	"github.com/jake/llmify/internal/util"
)

// CrawlResult represents the results of crawling a project
type CrawlResult struct {
	IncludedFiles []string
	FileTree      string
	ExcludedCount int
	IncludedCount int
}

// LoadIgnoreMatcher loads ignore patterns from .gitignore and .llmignore files
func LoadIgnoreMatcher(projectRoot string, noGitignore, noLLMignore bool) (*ignore.IgnoreMatcher, error) {
	matcher := ignore.NewIgnoreMatcher(nil)

	if !noGitignore {
		gitignorePath := filepath.Join(projectRoot, ".gitignore")
		if patterns, err := ignore.LoadIgnoreFile(gitignorePath); err == nil {
			matcher.AddPatterns(patterns)
		}
	}

	if !noLLMignore {
		llmignorePath := filepath.Join(projectRoot, ".llmignore")
		if patterns, err := ignore.LoadIgnoreFile(llmignorePath); err == nil {
			matcher.AddPatterns(patterns)
		}
	}

	return matcher, nil
}

// CreateDefaultLLMIgnoreFile creates a default .llmignore file with common patterns
func CreateDefaultLLMIgnoreFile(projectRoot string) error {
	defaultPatterns := []string{
		".git/",
		".git/**",
		"node_modules/",
		"vendor/",
		"dist/",
		"build/",
		"*.log",
		"*.lock",
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",
		"*.class",
		"*.jar",
		"*.war",
		"*.ear",
		"*.zip",
		"*.tar",
		"*.gz",
		"*.rar",
		"*.7z",
		"*.pdf",
		"*.jpg",
		"*.jpeg",
		"*.png",
		"*.gif",
		"*.ico",
		"*.svg",
		"*.woff",
		"*.woff2",
		"*.ttf",
		"*.eot",
		"*.mp4",
		"*.webm",
		"*.mp3",
		"*.wav",
		"*.flac",
		"*.m4a",
		"*.db",
		"*.sqlite",
		"*.sqlite3",
		"*.cache",
		"*.tmp",
		"*.temp",
		"*.bak",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",
		"Thumbs.db",
		// Large binary files
		"*.bin",
		"*.dat",
		"*.iso",
		"*.img",

		// IDE and editor files
		".idea/",
		".vscode/",
		".vs/",
		"*.sublime-*",

		// Build artifacts
		"*.o",
		"*.obj",
		"*.a",
		"*.lib",
		"*.pyc",
		"*.pyo",
		"__pycache__/",
		".pytest_cache/",

		// Package manager directories
		"bower_components/",
		"jspm_packages/",
		".pnpm-store/",

		// Test coverage and reports
		"coverage/",
		".nyc_output/",
		"test-results/",
		"cypress/videos/",
		"cypress/screenshots/",

		// Temporary directories
		"tmp/",
		"temp/",
		"logs/",

		// Configuration files that might contain secrets
		".env",
		".env.local",
		".env.*.local",

		// Generated documentation
		"docs/_build/",
		"_site/",
		".docusaurus/",
		".vuepress/dist/",

		// Output file itself
		"llm.txt",
	}

	content := strings.Join(defaultPatterns, "\n")
	return util.WriteStringToFile(filepath.Join(projectRoot, ".llmignore"), content)
}

// CrawlProject crawls the project directory and returns a CrawlResult
func CrawlProject(projectRoot string, matcher *ignore.IgnoreMatcher, maxDepth int, excludeBinary bool) (*CrawlResult, error) {
	result := &CrawlResult{}

	// Generate file tree
	tree, err := generateFileTree(projectRoot, matcher, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("generating file tree: %w", err)
	}
	result.FileTree = tree

	// Walk through files
	err = filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}

		// Explicitly skip common ignored directories
		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") ||
			relPath == "node_modules" || strings.HasPrefix(relPath, "node_modules/") {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file should be ignored
		if matcher.ShouldIgnore(relPath) {
			result.ExcludedCount++
			return nil
		}

		// Check if file is binary
		if excludeBinary {
			isText, err := util.IsLikelyTextFile(path)
			if err != nil {
				return err
			}
			if !isText {
				result.ExcludedCount++
				return nil
			}
		}

		result.IncludedFiles = append(result.IncludedFiles, relPath)
		result.IncludedCount++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking project directory: %w", err)
	}

	return result, nil
}

// BuildOutputContent builds the final output content from the crawl results
func BuildOutputContent(result *CrawlResult, includeHeader bool) string {
	var content strings.Builder

	if includeHeader {
		content.WriteString("# Project Structure\n\n")
		content.WriteString(result.FileTree)
		content.WriteString("\n\n# File Contents\n\n")
	}

	for _, file := range result.IncludedFiles {
		content.WriteString(fmt.Sprintf("## %s\n\n", file))
		fileContent, err := util.ReadFileContent(file)
		if err != nil {
			content.WriteString(fmt.Sprintf("Error reading file: %v\n\n", err))
			continue
		}
		content.WriteString(util.LimitString(fileContent, 10000))
		content.WriteString("\n\n")
	}

	return content.String()
}

// generateFileTree generates a tree representation of the directory structure
func generateFileTree(root string, matcher *ignore.IgnoreMatcher, maxDepth int) (string, error) {
	var tree strings.Builder
	baseDir := filepath.Base(root)
	tree.WriteString(baseDir + "\n")

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Explicitly skip common ignored directories
		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") ||
			relPath == "node_modules" || strings.HasPrefix(relPath, "node_modules/") {
			return filepath.SkipDir
		}

		// Check depth
		depth := strings.Count(relPath, string(os.PathSeparator))
		if maxDepth > 0 && depth >= maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if path should be ignored
		if matcher.ShouldIgnore(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add indentation and tree characters
		indent := strings.Repeat("  ", depth)
		prefix := "├── "
		if info.IsDir() {
			prefix = "└── "
		}

		// Add the entry
		tree.WriteString(indent + prefix + info.Name() + "\n")

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("generating file tree: %w", err)
	}

	return tree.String(), nil
}
