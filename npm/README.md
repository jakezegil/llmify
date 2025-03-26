# LLMify

A CLI tool that generates a comprehensive text file containing your codebase context for Large Language Models (LLMs).

## Features

- Crawls project directories and creates a single output file with:
  - A visual tree structure of your project
  - The content of all included text files
- Respects `.gitignore` and `.llmignore` patterns
- Intelligently excludes binary files based on content analysis
- Offers customizable filtering (include/exclude patterns, max depth, specific sub-path)
- Formats output in a way that's optimized for LLM context

## Installation

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

## Usage

```bash
# Basic usage - creates llm.txt in the current directory
llmify

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

## Using .llmignore

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

## Full CLI Options

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

## Example Output

The generated file will have this structure:

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

## License

MIT License - See LICENSE file for details. 