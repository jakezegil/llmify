#!/bin/bash
set -e

# Installation script for LLMify
# This script installs the LLMify binary to a location in your PATH

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default installation directory
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_LOCAL_BIN="$HOME/bin"
USER_LOCAL_BIN_ALTERNATE="$HOME/.local/bin"

echo -e "${BLUE}LLMify Installation Script${NC}"

# Check for binary in bin directory
if [ -f "bin/llmify" ]; then
    BINARY_PATH="bin/llmify"
elif [ -f "./llmify" ]; then
    BINARY_PATH="./llmify"
else
    echo -e "${RED}Error: LLMify binary not found!${NC}"
    echo -e "Please run ${YELLOW}./scripts/build.sh${NC} first or make sure you're in the correct directory."
    exit 1
fi

echo -e "${GREEN}Found binary at:${NC} $BINARY_PATH"

# Determine best installation directory
if [ -d "$DEFAULT_INSTALL_DIR" ] && [ -w "$DEFAULT_INSTALL_DIR" ]; then
    # System-wide installation (if we have permission)
    SUGGESTED_DIR="$DEFAULT_INSTALL_DIR"
elif [ -d "$USER_LOCAL_BIN" ]; then
    # User's ~/bin directory exists
    SUGGESTED_DIR="$USER_LOCAL_BIN"
elif [ -d "$USER_LOCAL_BIN_ALTERNATE" ]; then
    # User's ~/.local/bin directory exists
    SUGGESTED_DIR="$USER_LOCAL_BIN_ALTERNATE"
else
    # Create user's bin directory
    mkdir -p "$USER_LOCAL_BIN"
    SUGGESTED_DIR="$USER_LOCAL_BIN"
    echo -e "${YELLOW}Created directory: ${SUGGESTED_DIR}${NC}"
    echo -e "${YELLOW}You may need to add this to your PATH.${NC}"
fi

# Ask for installation directory
read -p "Install directory [$SUGGESTED_DIR]: " INSTALL_DIR
INSTALL_DIR=${INSTALL_DIR:-$SUGGESTED_DIR}

# Create directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Directory $INSTALL_DIR doesn't exist. Creating it...${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Check write permissions
if [ ! -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}You don't have write permissions to $INSTALL_DIR${NC}"
    echo -e "Trying with sudo..."
    sudo cp "$BINARY_PATH" "$INSTALL_DIR/llmify"
else
    cp "$BINARY_PATH" "$INSTALL_DIR/llmify"
fi

# Make executable
chmod +x "$INSTALL_DIR/llmify"

echo -e "${GREEN}LLMify installed successfully to ${INSTALL_DIR}/llmify${NC}"

# Check if directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH!${NC}"
    echo -e "Add the following line to your ~/.bashrc, ~/.zshrc, or equivalent shell configuration file:"
    echo -e "${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    echo -e "Then reload your shell configuration with: ${BLUE}source ~/.bashrc${NC} (or equivalent)"
fi

# Verify installation
echo -e "${BLUE}Verifying installation...${NC}"
if command -v "$INSTALL_DIR/llmify" &> /dev/null; then
    echo -e "${GREEN}Installation verified! You can now use 'llmify' command.${NC}"
else
    echo -e "${YELLOW}Could not verify installation. You may need to restart your terminal.${NC}"
fi

echo -e "\n${GREEN}Installation complete!${NC}" 