#!/bin/bash
# Build script for compliance-server Linux binary
# This script cross-compiles the Go application for Linux AMD64

set -e

echo "====================================="
echo "Building compliance-server for Linux"
echo "====================================="

# Navigate to project root (parent of docker directory)
cd "$(dirname "$0")/.."

# Clean previous builds
echo "Cleaning previous builds..."
rm -f docker/bin/compliance-server

# Create bin directory if it doesn't exist
mkdir -p docker/bin

# Build for Linux AMD64
echo "Building Linux AMD64 binary..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o docker/bin/compliance-server \
    ./cmd/compliance-server

# Verify the binary was created
if [ -f docker/bin/compliance-server ]; then
    echo "✓ Build successful!"
    echo "Binary location: docker/bin/compliance-server"
    ls -lh docker/bin/compliance-server
else
    echo "✗ Build failed - binary not found"
    exit 1
fi

echo ""
echo "====================================="
echo "Build complete!"
echo "====================================="
echo ""
echo "Next steps:"
echo "1. cd docker"
echo "2. docker-compose -f docker-compose.binary.yml build"
echo "3. docker-compose -f docker-compose.binary.yml up -d"
echo ""
