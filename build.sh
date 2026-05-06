#!/bin/bash
set -e

VERSION=${1:-v1.0.1}
mkdir -p dist

echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o dist/gh-issuefs-linux-amd64 ./cmd/gh-issuefs

echo "Building for Darwin AMD64..."
GOOS=darwin GOARCH=amd64 go build -o dist/gh-issuefs-darwin-amd64 ./cmd/gh-issuefs

echo "Building for Darwin ARM64..."
GOOS=darwin GOARCH=arm64 go build -o dist/gh-issuefs-darwin-arm64 ./cmd/gh-issuefs

echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o dist/gh-issuefs-windows-amd64.exe ./cmd/gh-issuefs

echo "Build complete. Binaries in dist/"
ls -lh dist/
