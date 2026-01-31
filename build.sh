#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  FlashDB Build Script${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if wails is installed
if ! command -v wails &> /dev/null; then
    echo -e "${RED}Error: wails is not installed${NC}"
    echo "Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

# Clean previous builds
echo -e "${YELLOW}Cleaning previous builds...${NC}"
rm -rf build/bin frontend/dist

# Install frontend dependencies
echo -e "${YELLOW}Installing frontend dependencies...${NC}"
cd frontend && npm install && cd ..

# Determine platform
PLATFORM=${1:-""}
VERSION=${2:-"dev"}

build_macos_intel() {
    echo -e "${YELLOW}Building macOS (Intel)...${NC}"
    wails build -platform darwin/amd64 -o FlashDB-darwin-amd64
    echo -e "${GREEN}✓ macOS Intel build complete${NC}"
}

build_macos_arm() {
    echo -e "${YELLOW}Building macOS (Apple Silicon)...${NC}"
    wails build -platform darwin/arm64 -o FlashDB-darwin-arm64
    echo -e "${GREEN}✓ macOS Apple Silicon build complete${NC}"
}

build_windows() {
    echo -e "${YELLOW}Building Windows...${NC}"
    wails build -platform windows/amd64 -o FlashDB.exe
    echo -e "${GREEN}✓ Windows build complete${NC}"
}

build_all() {
    build_macos_intel
    build_macos_arm
    build_windows
}

case "$PLATFORM" in
    "macos-intel")
        build_macos_intel
        ;;
    "macos-arm")
        build_macos_arm
        ;;
    "macos")
        build_macos_intel
        build_macos_arm
        ;;
    "windows")
        build_windows
        ;;
    "all")
        build_all
        ;;
    "")
        # Default: build for current platform
        echo -e "${YELLOW}Building for current platform...${NC}"
        wails build
        echo -e "${GREEN}✓ Build complete${NC}"
        ;;
    *)
        echo -e "${RED}Unknown platform: $PLATFORM${NC}"
        echo "Usage: ./build.sh [platform] [version]"
        echo ""
        echo "Platforms:"
        echo "  (empty)      - Build for current platform"
        echo "  macos-intel  - Build for macOS Intel"
        echo "  macos-arm    - Build for macOS Apple Silicon"
        echo "  macos        - Build for both macOS architectures"
        echo "  windows      - Build for Windows"
        echo "  all          - Build for all platforms"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Output directory: ${YELLOW}build/bin/${NC}"
ls -la build/bin/
