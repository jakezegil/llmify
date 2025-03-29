package git

import (
	"bytes"
	"fmt"
	"os/exec"
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
