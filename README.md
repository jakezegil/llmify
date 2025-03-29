# 🚀 LLMify

> **A collection of tools that optimize your codebase for LLMs and agents**

[![npm version](https://img.shields.io/npm/v/llmify.svg?style=flat-square)](https://www.npmjs.com/package/llmify)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/jakezegil/llmify/blob/main/LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://github.com/jakezegil/llmify/pulls)

LLMify is made for LLMs by LLMs - a suite of powerful tools that transform your codebase into the perfect context for AI agents. Get started with a single command: `npx llmify`

## ✨ Features

- 📊 **Visual Project Structure** - Creates a tree view of your codebase
- 📄 **Smart Content Extraction** - Pulls content from all relevant files
- 🧠 **Context Optimization** - Formats output specifically for LLMs
- 🔍 **Intelligent Filtering** - Respects `.gitignore` and auto-creates `.llmignore`
- 🛠️ **Highly Customizable** - Control depth, paths, and patterns
- 💬 **AI-Powered Commit Messages** - Generate detailed commit messages using LLMs
- 📝 **Documentation Updates** - Automatically update docs based on code changes

## 🔥 Why LLMify?

Getting the right context to an LLM is critical for quality results. LLMify solves this by:

- **Saving Time** - No more manual file copying or writing commit messages
- **Improving Responses** - Gives LLMs better structural understanding of your codebase
- **Reducing Token Usage** - Smart filtering excludes irrelevant files
- **Working Everywhere** - Supports all major platforms
- **Better Commits** - Generate clear, conventional commit messages
- **Up-to-Date Docs** - Keep documentation in sync with code changes

## 📦 Installation

### NPM (Recommended)

```bash
npm install -g llmify
```

### Direct Download

Grab the [latest release](https://github.com/jakezegil/llmify/releases) for your platform.

## 🚀 Quick Start

Generate context for your current directory:

```bash
# Creates llm.txt in current directory
llmify

# Paste into your favorite LLM
cat llm.txt | pbcopy  # macOS
cat llm.txt | xclip   # Linux
type llm.txt | clip   # Windows
```

Generate a commit message for staged changes:

```bash
# Stage your changes
git add .

# Generate and edit commit message
llmify commit

# Force commit without confirmation
llmify commit -f

# Update docs and commit
llmify commit --docs
```

## 👩‍💻 Usage Examples

### Context Generation

```bash
# Specify a different project
llmify /path/to/project

# Only include a specific subdirectory
llmify -p src/components

# Custom output file 
llmify -o context_for_llm.txt

# Limit directory depth for large projects
llmify -d 3 

# Exclude specific files
llmify -e "*.test.js" -e "**/__mocks__/*"

# Include files that would otherwise be excluded
llmify -i "important-config.json"

# See what's happening (helpful for debugging)
llmify -v
```

### Commit Message Generation

```bash
# Generate commit message with default settings
llmify commit

# Skip confirmation prompt
llmify commit -f

# Update docs and commit
llmify commit --docs

# Verbose output
llmify commit -v

# Set LLM timeout (in seconds)
llmify commit --llm-timeout 60
```

## ⚙️ Configuration

LLMify can be configured via a `.llmifyrc.yaml` file in your project root or `~/.config/llmify/config.yaml`:

```yaml
# LLM Configuration
llm:
  # The LLM provider to use (e.g., "openai", "anthropic", "ollama")
  provider: "openai"
  
  # The default model to use for general tasks
  model: "gpt-4o"
  
  # Provider-specific settings
  ollama_base_url: "http://localhost:11434"  # Only used for Ollama provider

# Commit-specific settings
commit:
  # Optional: Override the default model for commit message generation
  model: "gpt-4o"

# Documentation update settings
docs:
  # Optional: Override the default model for documentation updates
  model: "gpt-4o"
```

Environment variables can also be used:
- `LLMIFY_LLM_PROVIDER` - Set the LLM provider
- `LLMIFY_LLM_MODEL` - Set the default model
- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key

## 🔧 `.llmignore` - Control What's Included

LLMify automatically creates a `.llmignore` file with sensible defaults. Customize it to exclude any files irrelevant to your LLM conversations:

```
# Example .llmignore
*.min.js
*.csv
node_modules/
dist/
coverage/
```

## 🎯 Example Output

The generated file has a clean, LLM-friendly structure:

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
```
</details>

## 💡 Pro Tips

- Include a `.llmignore` in your project templates
- Use with `--path` to focus on specific parts of your codebase
- Combine with project-specific prompts for best results
- For very large codebases, use `-d` to limit directory depth
- Use `llmify commit --docs` to keep documentation in sync
- Configure different models for different tasks in `.llmifyrc.yaml`

## 🤝 Contributing

Contributions are welcome! Feel free to:
- Report bugs
- Suggest features
- Submit pull requests

## 📝 License

[MIT](https://github.com/jakezegil/llmify/blob/main/LICENSE) © Jake Zegil

---

<p align="center">
  Made with ❤️ for better LLM interactions
  <br>
  <a href="https://github.com/jakezegil/llmify">Star on GitHub</a> •
  <a href="https://www.npmjs.com/package/llmify">View on npm</a>
</p>
