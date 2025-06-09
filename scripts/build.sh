#!/bin/bash
set -e

VERSION=${1:-"0.1.0"}
BINARY_NAME="cloudwatch-log-tui"

# ビルドディレクトリの作成
mkdir -p dist

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o "dist/${BINARY_NAME}-${VERSION}-darwin-amd64" .
tar -czf "dist/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz" -C dist "${BINARY_NAME}-${VERSION}-darwin-amd64"

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o "dist/${BINARY_NAME}-${VERSION}-darwin-arm64" .
tar -czf "dist/${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz" -C dist "${BINARY_NAME}-${VERSION}-darwin-arm64"

# Linux (64-bit)
GOOS=linux GOARCH=amd64 go build -o "dist/${BINARY_NAME}-${VERSION}-linux-amd64" .
tar -czf "dist/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz" -C dist "${BINARY_NAME}-${VERSION}-linux-amd64"

# Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o "dist/${BINARY_NAME}-${VERSION}-windows-amd64.exe" .
# Windows 用は zip 形式で圧縮（tar.gz より一般的）
cd dist && zip "${BINARY_NAME}-${VERSION}-windows-amd64.zip" "${BINARY_NAME}-${VERSION}-windows-amd64.exe" && cd ..

# Windows (ARM64) - Surface Pro X など向け
GOOS=windows GOARCH=arm64 go build -o "dist/${BINARY_NAME}-${VERSION}-windows-arm64.exe" .
cd dist && zip "${BINARY_NAME}-${VERSION}-windows-arm64.zip" "${BINARY_NAME}-${VERSION}-windows-arm64.exe" && cd ..
