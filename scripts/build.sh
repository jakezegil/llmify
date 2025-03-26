#!/bin/bash
set -e

# Build script for LLMify
# This script creates binaries for multiple platforms

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Building LLMify...${NC}"

# Get current version from git tag or default to development
VERSION=$(git describe --tags --always --abbrev=0 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo -e "${BLUE}Version: ${VERSION}, Commit: ${COMMIT}${NC}"

# Create build directory if it doesn't exist
mkdir -p bin

# Build for the current platform first for quick testing
echo -e "${BLUE}Building for current platform...${NC}"
go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" -o bin/llmify .
echo -e "${GREEN}✓ Built for current platform${NC}"

# Ask if user wants to build for all platforms
read -p "Build for all platforms? (y/n): " BUILD_ALL

if [[ $BUILD_ALL == "y" ]]; then
    echo -e "${BLUE}Building for all supported platforms...${NC}"
    
    # Define platforms
    PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
    
    for PLATFORM in "${PLATFORMS[@]}"; do
        GOOS=${PLATFORM%/*}
        GOARCH=${PLATFORM#*/}
        OUTPUT_NAME="bin/llmify-${GOOS}-${GOARCH}"
        
        if [ "$GOOS" = "windows" ]; then
            OUTPUT_NAME="${OUTPUT_NAME}.exe"
        fi
        
        echo -e "${BLUE}Building for ${GOOS}/${GOARCH}...${NC}"
        GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" -o "${OUTPUT_NAME}" .
        echo -e "${GREEN}✓ Built ${OUTPUT_NAME}${NC}"
    done
fi

# Build the npm CLI wrapper
echo -e "${BLUE}Building npm package...${NC}"
(cd npm && npm install && npm run build)
echo -e "${GREEN}✓ Built npm package${NC}"

echo -e "${GREEN}Build complete!${NC}"
echo -e "Binaries are available in the ${BLUE}bin/${NC} directory" 