package refactor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/git"
	"github.com/spf13/viper"
)

// FindTSConfig searches upwards from startPath for tsconfig.json
func FindTSConfig(startPath string) (string, error) {
	current := startPath
	root, err := git.GetRepoRoot()
	if err != nil {
		log.Printf("Warning: Could not find repo root, searching from CWD: %v", err)
		root = "." // Fallback, might not be correct
	}
	absRoot, _ := filepath.Abs(root)

	for {
		tsconfigPath := filepath.Join(current, "tsconfig.json")
		if _, err := os.Stat(tsconfigPath); err == nil {
			return tsconfigPath, nil
		}

		parent := filepath.Dir(current)
		absCurrent, _ := filepath.Abs(current)
		if parent == current || absCurrent == absRoot || parent == "/" || parent == "" {
			// Reached root or top-level directory
			break
		}
		current = parent
	}
	return "", fmt.Errorf("tsconfig.json not found")
}

// CheckTypeScriptTypes runs `tsc --noEmit` in the directory containing tsconfig.json
// It operates on the provided file content written to a temporary file.
func CheckTypeScriptTypes(originalFilePath string, proposedContent string) (bool, string, error) {
	verbose := viper.GetBool("verbose")
	if verbose {
		log.Printf("Running TypeScript type check for proposed changes to: %s", originalFilePath)
	}

	// 1. Find tsconfig.json to determine the project root for tsc
	projectDir := filepath.Dir(originalFilePath)
	tsconfigPath, err := FindTSConfig(projectDir)
	if err != nil {
		// If no tsconfig found, maybe we can't reliably type check. Warn and skip.
		log.Printf("Warning: %v. Skipping type check for %s.", err, originalFilePath)
		return true, "Skipped: tsconfig.json not found", nil
	}
	projectRoot := filepath.Dir(tsconfigPath)
	if verbose {
		log.Printf("Found tsconfig at: %s (Project Root: %s)", tsconfigPath, projectRoot)
	}

	// 2. Create a temporary file with the proposed content
	// Safest: Backup original, write proposed, run tsc, restore original.
	backupPath := originalFilePath + ".llmify_bak"
	originalContent, err := os.ReadFile(originalFilePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to read original file %s for backup: %w", originalFilePath, err)
	}

	// Write proposed content to original file path (after backing up)
	err = os.WriteFile(originalFilePath, []byte(proposedContent), 0644)
	if err != nil {
		return false, "", fmt.Errorf("failed to write proposed content to %s for type check: %w", originalFilePath, err)
	}

	// Defer restoration of the original file
	defer func() {
		if writeErr := os.WriteFile(originalFilePath, originalContent, 0644); writeErr != nil {
			log.Printf("CRITICAL ERROR: Failed to restore original file content for %s from backup: %v", originalFilePath, writeErr)
		} else if verbose {
			log.Printf("Restored original content for %s", originalFilePath)
		}
		// Cleanup backup
		os.Remove(backupPath)
	}()

	// 3. Run tsc command
	cmd := exec.Command("tsc", "--noEmit", "--pretty")
	cmd.Dir = projectRoot // Run tsc from the project root where tsconfig is
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if verbose {
		log.Printf("Executing command: %s (in dir: %s)", cmd.String(), projectRoot)
	}
	err = cmd.Run()

	output := stdout.String() + "\n" + stderr.String()
	output = strings.TrimSpace(output)

	if err != nil {
		// tsc returns non-zero exit code on type errors
		if verbose {
			log.Printf("Type check failed for %s. Output:\n%s", originalFilePath, output)
		}
		// Distinguish execution errors from type errors if possible (e.g., tsc not found)
		if _, ok := err.(*exec.ExitError); ok {
			// It ran but exited with error code (likely type errors)
			return false, output, nil // Type errors found
		}
		// Some other error running the command
		log.Printf("Error executing tsc: %v", err)
		return false, output, fmt.Errorf("failed to execute tsc command: %w. Output: %s", err, output)
	}

	// No error means type check passed
	if verbose {
		log.Printf("Type check passed for %s.", originalFilePath)
	}
	return true, "Type check passed.", nil
}
