#!/bin/bash
set -e

VERSION=${1:-"0.0.0"}
BIN_DIR="bin"

echo "Building markdown2confluence v${VERSION}..."

# Clean previous builds
rm -rf "${BIN_DIR}"

# Build for each platform
PLATFORMS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

for platform in "${PLATFORMS[@]}"; do
  read -r GOOS GOARCH <<< "$platform"
  
  if [ "$GOOS" = "windows" ]; then
    BIN_NAME="markdown2confluence.exe"
  else
    BIN_NAME="markdown2confluence"
  fi
  
  OUTPUT_DIR="${BIN_DIR}/${GOOS}-${GOARCH}"
  mkdir -p "$OUTPUT_DIR"
  
  echo "Building ${GOOS}/${GOARCH}..."
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${BIN_NAME}" \
    .
  
  # Make executable on Unix
  if [ "$GOOS" != "windows" ]; then
    chmod +x "${OUTPUT_DIR}/${BIN_NAME}"
  fi
done

echo "Build complete! Binaries are in ${BIN_DIR}/"
ls -la "${BIN_DIR}"/*/
