# LLMify Build and Deployment Scripts

This directory contains scripts to simplify building, installing, and deploying LLMify.

## Available Scripts

### `build.sh`

Builds LLMify binaries for the current platform and optionally for all supported platforms.

```bash
# Run from the project root
./scripts/build.sh
```

Features:
- Builds a binary for your current platform
- Optionally builds for all supported platforms (Linux, macOS, Windows)
- Embeds version info from git
- Also builds the npm package

### `install.sh`

Installs LLMify to your system (requires a built binary).

```bash
# Run from the project root
./scripts/install.sh
```

Features:
- Automatically detects the best installation location
- Adds LLMify to your PATH
- Supports both user-local and system-wide installation

### `deploy.sh`

Prepares a release and optionally publishes it to GitHub.

```bash
# Run from the project root
./scripts/deploy.sh
```

Features:
- Creates release archives for all platforms
- Updates the changelog
- Creates a git tag
- Can create a GitHub release (requires GitHub CLI)
- Can publish the npm package

### `quickstart.sh`

The easiest way to get started with LLMify - builds and runs in one command.

```bash
# Run from the project root
./scripts/quickstart.sh
```

Features:
- Builds LLMify
- Runs it on a directory of your choice
- Shows a preview of the output

## Making Scripts Executable

Before using these scripts, make them executable:

```bash
# Run from the project root
chmod +x scripts/*.sh
```

## For Windows Users

These scripts are designed for Unix-like systems (Linux, macOS). Windows users can:

1. Use Git Bash or WSL to run these scripts
2. Use the manual build commands from the main README.md
3. Download pre-built binaries from the releases page 