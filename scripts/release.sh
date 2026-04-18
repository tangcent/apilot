#!/usr/bin/env bash
# release.sh — prepare a new apilot release
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

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

cd "$ROOT_DIR"

CURRENT_VERSION=$(node -p "require('./package.json').version")
echo "Current version: $CURRENT_VERSION"
echo "New version: $VERSION"

if [ "$CURRENT_VERSION" = "$VERSION" ]; then
  echo "Version is already $VERSION, nothing to do."
  exit 0
fi

echo ""
echo "Updating version in package.json..."
npm pkg set version="$VERSION"

echo "Updating version in vscode-plugin/package.json..."
cd vscode-plugin
npm pkg set version="$VERSION"
cd "$ROOT_DIR"

echo ""
echo "Updating CHANGELOG.md..."
if [ -f CHANGELOG.md ]; then
  if grep -q "## \[$VERSION\]" CHANGELOG.md; then
    echo "  CHANGELOG already has entry for $VERSION"
  else
    echo "  Adding placeholder entry for $VERSION"
    sed -i.bak "s/## \[Unreleased\]/## [Unreleased]\n\n## [$VERSION] - $(date +%Y-%m-%d)/" CHANGELOG.md && rm -f CHANGELOG.md.bak
  fi
fi

echo ""
echo "Version updated. Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Commit: git add -A && git commit -m \"chore: release v$VERSION\""
echo "  3. Tag: git tag -a v$VERSION -m \"Release v$VERSION\""
echo "  4. Push: git push origin master --tags"
echo ""
echo "Or run: git add -A && git commit -m \"chore: release v$VERSION\" && git tag -a v$VERSION -m \"Release v$VERSION\""
