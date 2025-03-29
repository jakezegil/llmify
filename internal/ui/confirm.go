package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for a yes/no question and returns their response.
// The defaultAnswer parameter should be "Y" or "n" to indicate the default response.
func Confirm(prompt string, defaultAnswer string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [%s]: ", prompt, defaultAnswer)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" {
		response = strings.ToLower(defaultAnswer)
	}

	return response == "y" || response == "yes", nil
}
