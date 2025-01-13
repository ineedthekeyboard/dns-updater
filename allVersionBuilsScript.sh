#!/bin/bash

# Script name: build.sh

# Configuration
VERSION="1.0.0"
BUILD_TIME=$(date +%F-%T)
PROGRAM_NAME="do-ddns"
OUTPUT_DIR="dist"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create output directory
mkdir -p $OUTPUT_DIR

# Clean previous builds
echo -e "${BLUE}Cleaning previous builds...${NC}"
rm -rf $OUTPUT_DIR/*

# Run tests
echo -e "${BLUE}Running tests...${NC}"
go test ./...

if [ $? -ne 0 ]; then
    echo -e "${RED}Tests failed! Aborting build.${NC}"
    exit 1
fi

# Build function
build() {
    local os=$1
    local arch=$2
    local extension=""

    # Add .exe extension for Windows
    if [ "$os" == "windows" ]; then
        extension=".exe"
    fi

    local binary_name="${PROGRAM_NAME}_${os}_${arch}${extension}"
    local output_path="$OUTPUT_DIR/$binary_name"

    echo -e "${BLUE}Building for ${os}/${arch}...${NC}"

    GOOS=$os GOARCH=$arch go build \
        -ldflags="-s -w \
            -X main.Version=${VERSION} \
            -X main.BuildTime=${BUILD_TIME} \
            -extldflags '-static'" \
        -trimpath \
        -tags=timetzdata \
        -o "$output_path"

    if [ $? -eq 0 ]; then
        local size=$(stat -f%z "$output_path" 2>/dev/null || stat -c%s "$output_path" 2>/dev/null)
        echo -e "${GREEN}Successfully built $binary_name ($(($size/1024)) KB)${NC}"
    else
        echo -e "${RED}Failed to build for ${os}/${arch}${NC}"
        return 1
    fi
}

# Build for different platforms
build "windows" "amd64" # Windows 64-bit
build "windows" "386"   # Windows 32-bit
build "darwin" "amd64"  # macOS Intel
build "darwin" "arm64"  # macOS Apple Silicon
build "linux" "amd64"   # Linux 64-bit
build "linux" "arm64"   # Linux ARM 64-bit (for Raspberry Pi, etc.)

# Create checksums
echo -e "${BLUE}Generating checksums...${NC}"
cd $OUTPUT_DIR
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum * > checksums.txt
else
    # For macOS where sha256sum isn't available
    for file in *; do
        if [ "$file" != "checksums.txt" ]; then
            shasum -a 256 "$file" >> checksums.txt
        fi
    done
fi
cd ..

echo -e "${GREEN}Build process complete!${NC}"
echo -e "Binaries and checksums available in ${BLUE}${OUTPUT_DIR}${NC} directory"

# Optional: run specific version
if [ "$1" = "--run" ]; then
    # Detect OS and architecture
    case "$(uname -s)" in
        Darwin*)
            case "$(uname -m)" in
                arm64)  binary="${OUTPUT_DIR}/${PROGRAM_NAME}_darwin_arm64" ;;
                *)      binary="${OUTPUT_DIR}/${PROGRAM_NAME}_darwin_amd64" ;;
            esac
            ;;
        Linux*)
            case "$(uname -m)" in
                aarch64) binary="${OUTPUT_DIR}/${PROGRAM_NAME}_linux_arm64" ;;
                *)       binary="${OUTPUT_DIR}/${PROGRAM_NAME}_linux_amd64" ;;
            esac
            ;;
        MINGW*|MSYS*|CYGWIN*)
            binary="${OUTPUT_DIR}/${PROGRAM_NAME}_windows_amd64.exe"
            ;;
    esac

    if [ -f "$binary" ]; then
        echo -e "${BLUE}Running $binary...${NC}"
        chmod +x "$binary"
        "$binary"
    else
        echo -e "${RED}Could not find appropriate binary to run${NC}"
    fi
fi