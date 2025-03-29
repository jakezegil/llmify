package language

import (
	"path/filepath"
	"strings"
)

// Mapping from lower-case extension to language name
var extensionMap = map[string]string{
	// Web
	".ts":     "typescript",
	".tsx":    "typescript",
	".js":     "javascript",
	".jsx":    "javascript",
	".mjs":    "javascript",
	".cjs":    "javascript",
	".html":   "html",
	".htm":    "html",
	".css":    "css",
	".scss":   "scss",
	".sass":   "sass",
	".less":   "less",
	".vue":    "vue",    // Often contains JS/TS, CSS, HTML
	".svelte": "svelte", // Similar to Vue

	// Backend / General Purpose
	".py":    "python",
	".pyw":   "python",
	".go":    "go",
	".java":  "java",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".scala": "scala",
	".rs":    "rust",
	".rb":    "ruby",
	".php":   "php",
	".cs":    "csharp",
	".fs":    "fsharp", // F#
	".swift": "swift",
	".m":     "objectivec", // Can also be Matlab
	".mm":    "objectivec", // Objective-C++
	".c":     "c",
	".h":     "c", // Or C++ - context might be needed for C vs C++ headers
	".cpp":   "cpp",
	".hpp":   "cpp",
	".cxx":   "cpp",
	".hxx":   "cpp",
	".cc":    "cpp",
	".hh":    "cpp",
	".pl":    "perl", // Perl
	".pm":    "perl",
	".lua":   "lua", // Lua

	// Shell / Scripting
	".sh":   "shell",
	".bash": "shell",
	".zsh":  "shell",
	".ps1":  "powershell", // PowerShell

	// Data / Config
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
	".toml": "toml",
	".xml":  "xml",
	".sql":  "sql",
	".csv":  "csv", // Treat as distinct type?
	".md":   "markdown",
	".rst":  "rst", // ReStructuredText

	// DevOps / Infra
	".tf":         "terraform", // Terraform
	".tfvars":     "terraform",
	".hcl":        "hcl", // HashiCorp Config Language (also Terraform)
	".dockerfile": "dockerfile",
	"Dockerfile":  "dockerfile", // Also check basename

	// Others
	".r": "r", // R language
}

// Detect determines the programming/markup language of a file based on its extension.
// Returns the language name (lowercase) or an empty string if unknown.
func Detect(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := extensionMap[ext]; ok {
		return lang
	}

	// Handle files without extensions like Dockerfile, Makefile?
	baseName := filepath.Base(filePath)
	if lang, ok := extensionMap[baseName]; ok { // Check basename directly
		return lang
	}

	// Add more sophisticated checks if needed (e.g., shebang line analysis)

	return "" // Unknown
}
