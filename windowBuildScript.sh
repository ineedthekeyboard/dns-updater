#!/bin/bash

# Script name: build.sh

# Configuration
BINARY_NAME="do-ddns.exe"
VERSION="1.0.0"
BUILD_TIME=$(date +%F-%T)

# Clean previous builds
echo "Cleaning previous builds..."
rm -f $BINARY_NAME

# Run tests
echo "Running tests..."
go test ./...

# Check test status
if [ $? -ne 0 ]; then
    echo "Tests failed! Aborting build."
    exit 1
fi

# Build the program with optimizations
echo "Building optimized binary..."
GOOS=windows GOARCH=amd64 go build \
    -ldflags="-s -w \
        -X main.Version=${VERSION} \
        -X main.BuildTime=${BUILD_TIME} \
        -extldflags '-static'" \
    -trimpath \
    -tags=timetzdata \
    -o $BINARY_NAME

# Check build status
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Get file size
size=$(stat -f%z "$BINARY_NAME" 2>/dev/null || stat -c%s "$BINARY_NAME" 2>/dev/null)
echo "Build successful! Created $BINARY_NAME ($(($size/1024)) KB)"

# Optional: Run the program
if [ "$1" = "--run" ]; then
    echo "Running program..."
    ./$BINARY_NAME
fi