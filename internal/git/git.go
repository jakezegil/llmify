package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runGitCommand executes a git command and returns its stdout output.
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Ensure git commands run relative to the repo root if possible
	// This might need refinement depending on where llmify is executed from.
	// For now, assume execution within the repo.

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git command failed: 'git %s': %v\nStderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetStagedDiff returns the output of `git diff --staged`.
func GetStagedDiff() (string, error) {
	diff, err := runGitCommand("diff", "--staged")
	if err != nil {
		// Distinguish between error and no diff?
		// For now, treat any error as potentially problematic
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	if diff == "" {
		return "", fmt.Errorf("no changes staged for commit") // Specific error for no changes
	}
	return diff, nil
}

// GetStagedFiles returns a list of relative paths of staged files.
func GetStagedFiles() ([]string, error) {
	output, err := runGitCommand("diff", "--staged", "--name-only", "--relative")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}
	if output == "" {
		return []string{}, nil // No files staged
	}
	files := strings.Split(output, "\n")
	// Filter out empty strings if any
	result := []string{}
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// Commit performs git commit with the given message.
func Commit(message string) error {
	_, err := runGitCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}

// AddFiles stages the specified files.
func AddFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	_, err := runGitCommand(args...)
	if err != nil {
		return fmt.Errorf("git add failed for files %v: %w", files, err)
	}
	return nil
}

// GetRepoRoot finds the root directory of the git repository.
func GetRepoRoot() (string, error) {
	root, err := runGitCommand("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to find git repository root: %w", err)
	}
	return root, nil
}

// GetDiffFromCommits returns the diff and commit messages from the last N commits
func GetDiffFromCommits(n int) (string, []string, error) {
	// Get commit messages
	commitMsgs, err := runGitCommand("log", "-n", fmt.Sprintf("%d", n), "--pretty=format:%s")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	messages := strings.Split(commitMsgs, "\n")

	// Get diff
	diff, err := runGitCommand("diff", fmt.Sprintf("HEAD~%d", n), "HEAD")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get diff: %w", err)
	}

	return diff, messages, nil
}

// FilterDiffByPath filters a diff to only include changes in the specified path
func FilterDiffByPath(diff, path string) (string, error) {
	// Create a temporary file with the diff
	tmpFile, err := os.CreateTemp("", "llmify-diff-*.patch")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write diff to temp file
	if _, err := tmpFile.WriteString(diff); err != nil {
		return "", fmt.Errorf("failed to write diff to temp file: %w", err)
	}

	// Use git apply to filter the diff
	filteredDiff, err := runGitCommand("apply", "--cached", "--numstat", tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to filter diff: %w", err)
	}

	return filteredDiff, nil
}

// FindRelevantDocs finds documentation files that may need updates based on the diff
func FindRelevantDocs(diff string) ([]string, error) {
	// Get list of changed files from diff
	changedFiles, err := runGitCommand("diff", "--name-only", "HEAD~1", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Common documentation directories and file patterns
	docDirs := []string{
		"docs",
		"doc",
		"documentation",
		".",
	}
	docPatterns := []string{
		"*.md",
		"*.rst",
		"*.txt",
		"*.adoc",
		"*.asciidoc",
	}

	var relevantDocs []string
	for _, file := range strings.Split(changedFiles, "\n") {
		// Check if file is in a documentation directory
		dir := filepath.Dir(file)
		for _, docDir := range docDirs {
			if strings.HasPrefix(dir, docDir) {
				// Check if file matches documentation patterns
				for _, pattern := range docPatterns {
					if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
						relevantDocs = append(relevantDocs, file)
						break
					}
				}
				break
			}
		}
	}

	return relevantDocs, nil
}

// ReadFile reads a file from the repository
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes content to a file in the repository
func WriteFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}
