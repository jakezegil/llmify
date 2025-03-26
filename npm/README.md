# 🚀 LLMify

> **A CLI tool that generates a comprehensive text file containing your codebase context for Large Language Models (LLMs)**

[![npm version](https://img.shields.io/npm/v/llmify.svg?style=flat-square)](https://www.npmjs.com/package/llmify)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/username/llmify/blob/main/LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://github.com/username/llmify/pulls)

## ✨ Features

- 📊 **Visual Project Structure** - Crawls project directories and creates a tree structure
- 📄 **File Content Extraction** - Includes the content of all relevant text files
- 🧠 **Context Optimization** - Formats output in a way that's optimized for LLM context
- 🔍 **Intelligent Filtering** - Respects `.gitignore` and `.llmignore` patterns
- 🛠️ **Highly Customizable** - Control depth, paths, include/exclude patterns

## 📦 Installation

### Pre-built Binaries

Download the appropriate binary for your platform from the [Releases page](https://github.com/username/llmify/releases).

### Build from Source

```bash
# Clone the repository
git clone https://github.com/username/llmify.git
cd llmify

# Build the binary
go build -o llmify .
# Or on Windows: go build -o llmify.exe .

# Optional: Move to a directory in your PATH
# Linux/macOS
sudo mv llmify /usr/local/bin/
# Or for user-local installation: mv llmify ~/bin/
```

## 🚀 Quick Start

```bash
# Basic usage - creates llm.txt in the current directory
llmify

# Paste into your favorite LLM
cat llm.txt | pbcopy  # macOS
cat llm.txt | xclip   # Linux
type llm.txt | clip   # Windows
```

## 👩‍💻 Usage Examples

```bash
# Specify a different root directory
llmify /path/to/your/project

# Specify a different output file
llmify -o context_for_gpt.txt

# Only include content from a specific subdirectory or file
llmify -p src/components
llmify --path main.go

# Exclude specific patterns
llmify -e "*.log" -e "**/.cache/*"

# Include specific files that would otherwise be excluded
llmify -i "config/important.json"

# Limit directory depth for large projects
llmify -d 3

# Disable .gitignore processing
llmify --no-gitignore

# See detailed output (helpful for debugging)
llmify -v
```

## 🔧 Using `.llmignore`

Create a `.llmignore` file in your project's root directory to specify patterns that should be excluded from LLM context. This uses the same syntax as `.gitignore`. These rules apply *after* the `--path` filter, if used.

Example `.llmignore`:

```
# Exclude large data files
data/*.csv
*.json.gz

# Exclude generated documentation
docs/generated/

# Exclude specific libraries
lib/external/
```

## 🎯 Full CLI Options

```
Usage:
  llmify [directory] [flags]

Flags:
  -e, --exclude strings      Glob patterns to exclude (can be used multiple times)
      --exclude-binary       Attempt to exclude binary files based on content detection (default: true)
      --header               Include a header with project info (default: true)
  -i, --include strings      Glob patterns to include (overrides excludes, use carefully)
  -d, --max-depth int        Maximum directory depth to crawl (0 for unlimited)
      --no-gitignore         Do not use .gitignore rules
      --no-llmignore         Do not use .llmignore rules
  -o, --output string        Name of the output file (default "llm.txt")
  -p, --path string          Only include files/directories within this specific relative path
  -v, --verbose              Enable verbose logging
  -h, --help                 Display help information
```

## 💡 Example Output

<details>
<summary>Click to see example output</summary>

```
============================================================
Project Root: /path/to/your/project
Generated At: 2023-06-15T10:30:45Z
============================================================

## File Tree Structure

```
yourproject/
├── .gitignore
├── main.go
├── utils.go
└── docs/
    ├── README.md
    └── usage.md
```

============================================================

## File Contents

### File: .gitignore

```
node_modules/
*.log
dist/
```

---

### File: main.go

```go
package main

import (
    "fmt"
)

func main() {
    fmt.Println("Hello, world!")
}
```

---

### File: utils.go

```go
package main

// ... file content here ...
```
```

</details>

## 📝 License

MIT License - See LICENSE file for details.

---

<p align="center">
  Made with ❤️ for better LLM interactions
  <br>
  <a href="https://github.com/username/llmify">Star on GitHub</a>
</p> 

