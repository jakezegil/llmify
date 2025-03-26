#!/bin/bash
set -e

# Quickstart script for LLMify
# This script builds and runs LLMify with a single command - perfect for first-time users

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}LLMify Quickstart${NC}"
echo -e "This script will build and run LLMify in a single command."

# Check if we're in the repo root
if [ ! -f "main.go" ] || [ ! -d "npm" ]; then
    echo -e "${RED}Error: This script must be run from the repository root.${NC}"
    exit 1
fi

# Check for required tools
echo -e "${BLUE}Checking requirements...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH.${NC}"
    echo -e "Please install Go from: ${BLUE}https://golang.org/dl/${NC}"
    exit 1
fi

# Create binary directory if it doesn't exist
mkdir -p bin

# Build for the current platform
echo -e "${BLUE}Building LLMify...${NC}"
go build -o bin/llmify .
echo -e "${GREEN}✓ Build successful!${NC}"

# Make executable
chmod +x bin/llmify

# Ask for input directory
echo -e "${BLUE}Which directory would you like to analyze?${NC}"
echo -e "Press Enter to use the current directory (LLMify itself), or specify a different path:"
read -p "Directory: " TARGET_DIR
TARGET_DIR=${TARGET_DIR:-.}

# Ask for output file
echo -e "${BLUE}Where would you like to save the output?${NC}"
echo -e "Press Enter to use the default (llm.txt), or specify a different filename:"
read -p "Output file: " OUTPUT_FILE
OUTPUT_FILE=${OUTPUT_FILE:-llm.txt}

# Run LLMify
echo -e "${BLUE}Running LLMify on ${TARGET_DIR}...${NC}"
./bin/llmify -o "${OUTPUT_FILE}" "${TARGET_DIR}"

# Check if the output file was created
if [ -f "${OUTPUT_FILE}" ]; then
    echo -e "${GREEN}✓ Success! LLMify output saved to: ${OUTPUT_FILE}${NC}"
    echo -e "File size: $(du -h "${OUTPUT_FILE}" | cut -f1)"
    
    # Show first few lines
    echo -e "${BLUE}Preview:${NC}"
    head -n 10 "${OUTPUT_FILE}"
    echo -e "${YELLOW}... (output truncated) ...${NC}"
    
    echo -e "\n${GREEN}You can now use this file as context for your LLM!${NC}"
else
    echo -e "${RED}Something went wrong. The output file was not created.${NC}"
fi 