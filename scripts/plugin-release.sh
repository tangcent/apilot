#!/usr/bin/env bash
# plugin-release.sh — prepare a new VSCode extension release
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VSCODE_DIR="$ROOT_DIR/vscode-plugin"

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.2.0"
  exit 1
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: Version must be in semver format (e.g., 0.2.0)"
  exit 1
fi

cd "$VSCODE_DIR"

CURRENT_VERSION=$(node -p "require('./package.json').version")
echo "Current VSCode extension version: $CURRENT_VERSION"
echo "New version: $VERSION"

if [ "$CURRENT_VERSION" = "$VERSION" ]; then
  echo "Version is already $VERSION, nothing to do."
  exit 0
fi

echo ""
echo "Updating version in package.json..."
npm pkg set version="$VERSION"

echo ""
echo "Installing dependencies..."
if [ ! -f package-lock.json ]; then
  npm install
fi

echo ""
echo "Compiling TypeScript..."
npm run compile

echo ""
echo "Creating bin directory and downloading binaries..."
mkdir -p bin

BASE_URL="https://github.com/tangcent/apilot/releases/latest/download"

download_binary() {
  local os=$1
  local arch=$2
  local suffix=$3
  local target="bin/apilot-${os}-${arch}${suffix}"
  if [ ! -f "$target" ]; then
    echo "  Downloading apilot-${os}-${arch}${suffix}..."
    curl -fsSL "${BASE_URL}/apilot-${os}-${arch}${suffix}" -o "$target"
    chmod +x "$target"
  else
    echo "  apilot-${os}-${arch}${suffix} already exists, skipping..."
  fi
}

download_binary darwin arm64 ""
download_binary darwin amd64 ""
download_binary linux arm64 ""
download_binary linux amd64 ""
download_binary windows amd64 ".exe"

echo ""
echo "Installing vsce..."
if ! command -v vsce &> /dev/null; then
  npm install -g @vscode/vsce
fi

echo ""
echo "Creating .vsix package..."
vsce package --allow-missing-repository --skip-license

VSIX_FILE="apilot-${VERSION}.vsix"
if [ -f "$VSIX_FILE" ]; then
  echo ""
  echo "Done. Created: $VSCODE_DIR/$VSIX_FILE"
  ls -lh "$VSIX_FILE"
  echo ""
  echo "To publish to VSCode Marketplace:"
  echo "  vsce publish"
  echo ""
  echo "Or manually publish the .vsix file:"
  echo "  vsce publish --packagePath $VSIX_FILE"
else
  echo "Error: Failed to create .vsix package"
  exit 1
fi
