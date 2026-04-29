#!/usr/bin/env bash
# plugin-package.sh — package the VSCode extension as a .vsix file
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VSCODE_DIR="$ROOT_DIR/vscode-plugin"
BIN_DIR="$VSCODE_DIR/bin"

VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.1.0")}

echo "Packaging apilot VSCode extension v${VERSION}..."

cd "$VSCODE_DIR"

if [ ! -f package-lock.json ]; then
  echo "Installing dependencies..."
  npm install
fi

echo "Compiling TypeScript..."
npm run compile

mkdir -p "$BIN_DIR" tmp

CLI_VERSION=$(git tag -l "v*.*.*" | sort -V | tail -n1 | sed 's/^v//')
if [ -z "$CLI_VERSION" ]; then
  CLI_VERSION="$VERSION"
fi

echo "Downloading apilot binaries (v${CLI_VERSION})..."
BASE_URL="https://github.com/tangcent/apilot/releases/download/v${CLI_VERSION}"

download_binary() {
  local archive_os=$1
  local archive_arch=$2
  local ext=$3
  local archive="apilot-${CLI_VERSION}-${archive_os}-${archive_arch}${ext}"

  local local_os="${archive_os}"
  if [ "$local_os" = "windows" ]; then
    local_os="win32"
  fi
  local local_arch="${archive_arch}"
  local target="$BIN_DIR/apilot-${local_os}-${local_arch}"
  if [ "$archive_os" = "windows" ]; then
    target="${target}.exe"
  fi

  if [ ! -f "$target" ]; then
    echo "  Downloading ${archive}..."
    if ! curl -fsSL "${BASE_URL}/${archive}" -o "tmp/${archive}"; then
      echo "  Warning: Failed to download ${archive}, skipping..."
      return 0
    fi
    if [ "$archive_os" = "windows" ]; then
      unzip -o "tmp/${archive}" -d tmp
    else
      tar -xzf "tmp/${archive}" -C tmp
    fi
    cp "tmp/apilot" "$target" 2>/dev/null || cp "tmp/apilot.exe" "$target" 2>/dev/null || true
    chmod +x "$target"
  else
    echo "  apilot-${local_os}-${local_arch} already exists, skipping..."
  fi
}

download_binary darwin arm64 ".tar.gz"
download_binary linux arm64 ".tar.gz"
download_binary linux x64 ".tar.gz"
download_binary windows x64 ".zip"

rm -rf tmp

echo "Installing vsce..."
if ! command -v vsce &> /dev/null; then
  npm install -g @vscode/vsce
fi

echo "Updating package version to ${VERSION}..."
npm pkg set version="$VERSION"

echo "Creating .vsix package..."
vsce package --allow-missing-repository --skip-license

VSIX_FILE="apilot-${VERSION}.vsix"
if [ -f "$VSIX_FILE" ]; then
  echo "Done. Created: $VSCODE_DIR/$VSIX_FILE"
  ls -lh "$VSIX_FILE"
else
  echo "Error: Failed to create .vsix package"
  exit 1
fi
