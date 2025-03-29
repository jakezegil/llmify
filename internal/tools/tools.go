package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jake/llmify/internal/util"
)

// Tool represents an external formatting or linting tool
type Tool struct {
	Name        string
	Command     string
	Args        []string
	InstallCmd  string
	CheckCmd    string
	VersionCmd  string
	IsInstalled bool
}

// NewTool creates a new Tool instance with the given configuration
func NewTool(name, command string, args []string, installCmd, checkCmd, versionCmd string) *Tool {
	return &Tool{
		Name:        name,
		Command:     command,
		Args:        args,
		InstallCmd:  installCmd,
		CheckCmd:    checkCmd,
		VersionCmd:  versionCmd,
		IsInstalled: false,
	}
}

// CheckInstallation verifies if the tool is installed and accessible
func (t *Tool) CheckInstallation() error {
	cmd := exec.Command(t.Command, strings.Fields(t.CheckCmd)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s is not installed: %v", t.Name, err)
	}
	t.IsInstalled = true
	return nil
}

// GetVersion returns the installed version of the tool
func (t *Tool) GetVersion() (string, error) {
	if !t.IsInstalled {
		return "", fmt.Errorf("%s is not installed", t.Name)
	}

	cmd := exec.Command(t.Command, strings.Fields(t.VersionCmd)...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get %s version: %v", t.Name, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Install installs the tool using the specified installation command
func (t *Tool) Install() error {
	cmd := exec.Command("sh", "-c", t.InstallCmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %v", t.Name, err)
	}
	t.IsInstalled = true
	return nil
}

// Format formats the given file using the tool
func (t *Tool) Format(filePath string) error {
	if !t.IsInstalled {
		return fmt.Errorf("%s is not installed", t.Name)
	}

	// Ensure file exists and is readable
	isText, err := util.IsLikelyTextFile(filePath)
	if err != nil {
		return fmt.Errorf("invalid file: %v", err)
	}
	if !isText {
		return fmt.Errorf("file is not a text file")
	}

	// Construct command with file path
	args := append(t.Args, filePath)
	cmd := exec.Command(t.Command, args...)

	// Run formatter
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("formatting failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// Lint checks the given file for issues using the tool
func (t *Tool) Lint(filePath string) (string, error) {
	if !t.IsInstalled {
		return "", fmt.Errorf("%s is not installed", t.Name)
	}

	// Ensure file exists and is readable
	isText, err := util.IsLikelyTextFile(filePath)
	if err != nil {
		return "", fmt.Errorf("invalid file: %v", err)
	}
	if !isText {
		return "", fmt.Errorf("file is not a text file")
	}

	// Construct command with file path
	args := append(t.Args, filePath)
	cmd := exec.Command(t.Command, args...)

	// Run linter
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("linting failed: %v", err)
	}

	return string(output), nil
}

// Common tool configurations
var (
	Prettier = NewTool(
		"prettier",
		"npx",
		[]string{"prettier", "--write"},
		"npm install -g prettier",
		"prettier --version",
		"prettier --version",
	)

	ESLint = NewTool(
		"eslint",
		"npx",
		[]string{"eslint", "--fix"},
		"npm install -g eslint",
		"eslint --version",
		"eslint --version",
	)

	GoFmt = NewTool(
		"gofmt",
		"gofmt",
		[]string{"-w"},
		"go install golang.org/x/tools/cmd/gofmt@latest",
		"gofmt -version",
		"gofmt -version",
	)

	Black = NewTool(
		"black",
		"black",
		[]string{},
		"pip install black",
		"black --version",
		"black --version",
	)

	Isort = NewTool(
		"isort",
		"isort",
		[]string{},
		"pip install isort",
		"isort --version",
		"isort --version",
	)
)

// GetToolForLanguage returns the appropriate formatting and linting tools for a given language
func GetToolForLanguage(lang string) (formatter, linter *Tool) {
	switch strings.ToLower(lang) {
	case "javascript", "typescript", "jsx", "tsx":
		return Prettier, ESLint
	case "go":
		return GoFmt, nil
	case "python":
		return Black, Isort
	default:
		return nil, nil
	}
}
