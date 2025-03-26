#!/bin/bash

# Exit on error
set -e

# In our new structure, npm files are in a subdirectory
NPM_DIR="$(cd npm && pwd)"
OUTPUT_DIR="$NPM_DIR/bin"
GO_PROJECT_ROOT="." # Current directory

# Ensure the output directory exists
mkdir -p "$OUTPUT_DIR"

# Define targets (OS/Architecture pairs)
TARGETS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

# Package name (used in Go build)
PACKAGE="github.com/jake/llmify" # Adjust to match your go.mod module path

# Version - can be set dynamically from git tags or other sources
VERSION="0.1.0"

echo "Building llmify binaries for version $VERSION..."

for TARGET in "${TARGETS[@]}"; do
    # Split GOOS and GOARCH
    IFS='/' read -r GOOS GOARCH <<< "$TARGET"

    # Set binary name
    BINARY_NAME="llmify-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    OUTPUT_PATH="$OUTPUT_DIR/$BINARY_NAME"

    echo "Building for $GOOS/$GOARCH -> $OUTPUT_PATH"

    # Build the Go binary
    # -ldflags="-s -w" strips debug symbols and DWARF info, reducing binary size
    GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="-s -w" -o "$OUTPUT_PATH" "$GO_PROJECT_ROOT"

    # Make sure binaries are executable (especially for Linux/macOS)
    if [ "$GOOS" != "windows" ]; then
        chmod +x "$OUTPUT_PATH"
    fi
done

echo "Build complete. Binaries are in $OUTPUT_DIR" 