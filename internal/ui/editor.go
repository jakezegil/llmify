package ui

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// DefaultEditorOrder defines the fallback order for text editors.
var DefaultEditorOrder = []string{"vim", "nano", "vi", "emacs", "code", "notepad"} // Add more as needed

// findEditor determines the editor to use based on environment or defaults.
func findEditor() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		// Ensure the editor command exists
		path, err := exec.LookPath(editor)
		if err == nil {
			return path, nil
		}
	}
	// Try default editors
	for _, ed := range DefaultEditorOrder {
		path, err := exec.LookPath(ed)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no suitable text editor found in PATH or $EDITOR")
}

// EditCommitMessage launches an editor to allow modification of the message.
func EditCommitMessage(initialMessage string) (string, error) {
	editorPath, err := findEditor()
	if err != nil {
		return "", fmt.Errorf("cannot find editor: %w", err)
	}

	tmpfile, err := ioutil.TempFile("", "llmify-commit-*.msg")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up

	// Write initial message to temp file
	if _, err := tmpfile.WriteString(initialMessage); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Prepare and run the editor command
	cmd := exec.Command(editorPath, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Launching editor (%s) for commit message...\n", editorPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor '%s' failed: %w", editorPath, err)
	}

	// Read the potentially modified content back
	contentBytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read back from temporary file: %w", err)
	}

	return string(contentBytes), nil
}

// ConfirmCommit prompts the user unless force is true. Returns true if commit should proceed.
func ConfirmCommit(force bool) (bool, bool, error) { // Returns (proceed, editAgain, error)
	if force {
		return true, false, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Commit with the edited message? [Y/n/e(dit)] ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
	case "y", "yes", "": // Default to yes
		return true, false, nil
	case "e", "edit":
		return false, true, nil // Signal to edit again
	case "n", "no":
		return false, false, nil // Abort
	default:
		fmt.Println("Invalid response. Aborting commit.")
		return false, false, nil
	}
}
