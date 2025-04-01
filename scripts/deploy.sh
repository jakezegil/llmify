#!/bin/bash
set -e

# Deployment script for LLMify
# This script creates release assets and helps with GitHub releases

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check for required tools
REQUIRED_TOOLS=("git" "go" "npm" "gh")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if ! command -v "$tool" &> /dev/null; then
        echo -e "${RED}Error: $tool is not installed or not in PATH.${NC}"
        
        if [ "$tool" = "gh" ]; then
            echo -e "The GitHub CLI ($tool) is required for this script."
            echo -e "Install it from: ${BLUE}https://cli.github.com/${NC}"
        fi
        
        exit 1
    fi
done

echo -e "${BLUE}LLMify Deployment Script${NC}"

# Check if we're in the repo root
if [ ! -f "main.go" ] || [ ! -d "npm" ]; then
    echo -e "${RED}Error: This script must be run from the repository root.${NC}"
    exit 1
fi

# Create directories
mkdir -p dist/release
# Ensure npm/bin directory exists
mkdir -p npm/bin

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Warning: You have uncommitted changes.${NC}"
    read -p "Continue anyway? (y/n): " CONTINUE
    if [[ $CONTINUE != "y" ]]; then
        echo -e "Exiting."
        exit 0
    fi
fi

# Get or prompt for version
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo -e "Current version: ${GREEN}${CURRENT_VERSION}${NC}"
read -p "New version (leave empty to use current): " NEW_VERSION

# Validate semantic version format if a new version is provided
if [[ -n "$NEW_VERSION" ]]; then
    # Ensure version starts with 'v'
    if [[ ! "$NEW_VERSION" =~ ^v ]]; then
        NEW_VERSION="v$NEW_VERSION"
        echo -e "${YELLOW}Added 'v' prefix: ${NEW_VERSION}${NC}"
    fi
    
    # Validate semver format (vX.Y.Z or vX.Y.Z-suffix)
    if [[ ! "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
        echo -e "${RED}Error: Version must follow semantic versioning format (vX.Y.Z or vX.Y.Z-suffix).${NC}"
        exit 1
    fi
    
    # Check if new version is already a tag
    if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
        echo -e "${RED}Error: Tag ${NEW_VERSION} already exists.${NC}"
        exit 1
    fi
fi

VERSION=${NEW_VERSION:-$CURRENT_VERSION}

# Remove 'v' prefix for filenames if present
VERSION_NUM=${VERSION#v}

echo -e "${BLUE}Building release for version: ${VERSION}${NC}"

# Build for all platforms
echo -e "${BLUE}Building binaries for all platforms...${NC}"
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    BINARY_NAME="llmify"
    ARCHIVE_NAME="llmify-${VERSION_NUM}-${GOOS}-${GOARCH}"
    NPM_BINARY_NAME="npm/bin/llmify-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
        ARCHIVE_NAME="${ARCHIVE_NAME}.zip"
        NPM_BINARY_NAME="${NPM_BINARY_NAME}.exe"
    else
        ARCHIVE_NAME="${ARCHIVE_NAME}.tar.gz"
    fi
    
    echo -e "${BLUE}Building for ${GOOS}/${GOARCH}...${NC}"
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=${VERSION} -X main.buildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o "dist/release/${BINARY_NAME}" .
    
    # Copy binary to npm/bin directory
    cp "dist/release/${BINARY_NAME}" "${NPM_BINARY_NAME}"
    echo -e "${GREEN}✓ Copied binary to npm/bin/${NC}"
    
    # Create archive
    pushd dist/release > /dev/null
    
    cp ../../README.md .
    cp ../../LICENSE . 2>/dev/null || echo "No LICENSE file found"
    
    if [ "$GOOS" = "windows" ]; then
        zip -q "${ARCHIVE_NAME}" "${BINARY_NAME}" README.md LICENSE 2>/dev/null
    else
        tar -czf "${ARCHIVE_NAME}" "${BINARY_NAME}" README.md LICENSE 2>/dev/null
    fi
    
    # Remove binary and docs after archiving
    rm "${BINARY_NAME}" README.md LICENSE 2>/dev/null
    
    popd > /dev/null
    
    echo -e "${GREEN}✓ Created ${ARCHIVE_NAME}${NC}"
done

# Build npm package
echo -e "${BLUE}Building npm package...${NC}"
(cd npm && npm version "$VERSION_NUM" --no-git-tag-version && npm install && npm run build)
echo -e "${GREEN}✓ Built npm package with version ${VERSION_NUM}${NC}"

# Verify npm package version matches desired version
NPM_VERSION=$(cd npm && node -e "console.log(require('./package.json').version)")
if [ "$NPM_VERSION" != "$VERSION_NUM" ]; then
    echo -e "${RED}Error: npm package version ($NPM_VERSION) doesn't match expected version ($VERSION_NUM)${NC}"
    exit 1
fi

# Create or update changelog
CHANGELOG_FILE="CHANGELOG.md"
if [ ! -f "$CHANGELOG_FILE" ]; then
    echo "# Changelog" > "$CHANGELOG_FILE"
    echo "" >> "$CHANGELOG_FILE"
fi

echo -e "${BLUE}Updating changelog...${NC}"
# Get git log since the last release tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$LAST_TAG" ]; then
    CHANGES=$(git log --pretty=format:"- %s" ${LAST_TAG}..HEAD)
else
    CHANGES=$(git log --pretty=format:"- %s")
fi

# Create new changelog section
TEMP_CHANGELOG=$(mktemp)
echo "# Changelog" > "$TEMP_CHANGELOG"
echo "" >> "$TEMP_CHANGELOG"
echo "## ${VERSION} ($(date '+%Y-%m-%d'))" >> "$TEMP_CHANGELOG"
echo "" >> "$TEMP_CHANGELOG"
echo "$CHANGES" >> "$TEMP_CHANGELOG"
echo "" >> "$TEMP_CHANGELOG"
tail -n +2 "$CHANGELOG_FILE" >> "$TEMP_CHANGELOG"
mv "$TEMP_CHANGELOG" "$CHANGELOG_FILE"

echo -e "${GREEN}✓ Updated changelog${NC}"

# Create GitHub release
echo -e "${BLUE}Do you want to create a GitHub release? (This requires gh CLI)${NC}"
read -p "Create GitHub release? (y/n): " CREATE_RELEASE

if [[ $CREATE_RELEASE == "y" ]]; then
    echo -e "${BLUE}Creating GitHub release...${NC}"
    
    # Create a new tag if it's a new version
    if [ "$NEW_VERSION" != "" ]; then
        echo -e "${BLUE}Adding files to git...${NC}"
        git add "$CHANGELOG_FILE" npm/package.json
        
        echo -e "${BLUE}Committing changes...${NC}"
        git commit -m "Release $VERSION"
        
        echo -e "${BLUE}Creating annotated git tag...${NC}"
        git tag -a "$VERSION" -m "Release $VERSION"
        echo -e "${GREEN}✓ Created git tag ${VERSION}${NC}"
    else
        echo -e "${YELLOW}Using existing tag ${VERSION}${NC}"
        
        # Check if tag exists locally
        if ! git rev-parse "$VERSION" >/dev/null 2>&1; then
            echo -e "${RED}Error: Tag ${VERSION} does not exist locally.${NC}"
            exit 1
        fi
    fi
    
    # Create release notes from changelog
    RELEASE_NOTES=$(sed -n "/## ${VERSION}/,/## /p" "$CHANGELOG_FILE" | sed '1d;$d')
    
    # Push to GitHub
    echo -e "${BLUE}Pushing commits to GitHub...${NC}"
    git push

    # Only push the current tag instead of all tags
    echo -e "${BLUE}Pushing tag ${VERSION} to GitHub...${NC}"
    if git push origin "$VERSION" 2>/dev/null; then
        echo -e "${GREEN}✓ Pushed tag ${VERSION}${NC}"
    else
        echo -e "${YELLOW}Warning: Couldn't push tag ${VERSION}, it may already exist on remote${NC}"
        # Make sure we have the remote tag locally if it exists
        git fetch origin tag "$VERSION" 2>/dev/null || true
    fi
    
    # Create GitHub release
    echo -e "${BLUE}Creating GitHub release ${VERSION} (npm: ${VERSION_NUM})...${NC}"
    cd dist/release
    gh release create "$VERSION" \
        --title "$VERSION" \
        --notes "$RELEASE_NOTES" \
        *.tar.gz *.zip
    
    echo -e "${GREEN}✓ Created GitHub release ${VERSION}${NC}"
    
    # Publish npm package
    echo -e "${BLUE}Do you want to publish the npm package?${NC}"
    read -p "Publish npm package? (y/n): " PUBLISH_NPM
    
    if [[ $PUBLISH_NPM == "y" ]]; then
        echo -e "${BLUE}Publishing npm package version ${VERSION_NUM}...${NC}"
        (cd ../../npm && npm publish)
        echo -e "${GREEN}✓ Published npm package version ${VERSION_NUM}${NC}"
    fi
fi

echo -e "\n${GREEN}Deployment preparation complete!${NC}"
echo -e "Release archives are available in ${BLUE}dist/release/${NC}" 