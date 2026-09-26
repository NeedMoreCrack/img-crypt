#!/usr/bin/env bash

set -e

DIST_DIR="dist"

echo "======================================"
echo "        ImgCrypt Build Script"
echo "======================================"
echo

mkdir -p "$DIST_DIR"

echo "[INFO] Cleaning old build files..."
rm -f "$DIST_DIR"/ImgCrypt-*

echo
echo "[INFO] Building Windows amd64..."
CGO_ENABLED=0 \
GOOS=windows \
GOARCH=amd64 \
go build -o "$DIST_DIR/ImgCrypt-windows-amd64.exe" .

echo "[OK] Windows amd64 build complete."

echo
echo "[INFO] Building macOS Intel..."
CGO_ENABLED=0 \
GOOS=darwin \
GOARCH=amd64 \
go build -o "$DIST_DIR/ImgCrypt-macos-amd64" .

echo "[OK] macOS Intel build complete."

echo
echo "[INFO] Building macOS Apple Silicon..."
CGO_ENABLED=0 \
GOOS=darwin \
GOARCH=arm64 \
go build -o "$DIST_DIR/ImgCrypt-macos-arm64" .

echo "[OK] macOS Apple Silicon build complete."

echo
echo "[INFO] Building Linux amd64..."
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64 \
go build -o "$DIST_DIR/ImgCrypt-linux-amd64" .

echo "[OK] Linux amd64 build complete."

echo
echo "[INFO] Building Linux arm64..."
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=arm64 \
go build -o "$DIST_DIR/ImgCrypt-linux-arm64" .

echo "[OK] Linux arm64 build complete."

echo
echo "======================================"
echo "          Build Complete"
echo "======================================"
echo

ls -lh "$DIST_DIR"