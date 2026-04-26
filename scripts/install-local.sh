#!/usr/bin/env bash
set -euo pipefail

SCRIPT_SOURCE="$0"
while [[ -h "$SCRIPT_SOURCE" ]]; do
    scriptDir="$( cd -P "$( dirname "$SCRIPT_SOURCE" )" && pwd )"
    SCRIPT_SOURCE="$(readlink "$SCRIPT_SOURCE")"
    [[ ${SCRIPT_SOURCE} != /* ]] && SCRIPT_SOURCE="$scriptDir/$SCRIPT_SOURCE"
done
scriptDir="$( cd -P "$( dirname "$SCRIPT_SOURCE" )" && pwd )"
PROJECT_ROOT="$(cd "$scriptDir/.." && pwd)"

INSTALL_DIR="${1:-}"
VERSION="$(git -C "$PROJECT_ROOT" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
BINARY_NAME="apilot"

if [[ "$(uname -s)" == "Darwin" ]]; then
    ARCH="$(uname -m)"
    if [[ "$ARCH" == "arm64" ]]; then
        BINARY_NAME="apilot-darwin-arm64"
    else
        BINARY_NAME="apilot-darwin-amd64"
    fi
elif [[ "$(uname -s)" == "Linux" ]]; then
    ARCH="$(uname -m)"
    if [[ "$ARCH" == "aarch64" ]]; then
        BINARY_NAME="apilot-linux-arm64"
    else
        BINARY_NAME="apilot-linux-amd64"
    fi
fi

echo "Building apilot v${VERSION} from source..."

cd "$PROJECT_ROOT"

LDFLAGS="-s -w -X github.com/tangcent/apilot/apilot-cli/build.Version=${VERSION} -X github.com/tangcent/apilot/apilot-cli/build.Date=${BUILD_DATE}"

go build -ldflags "$LDFLAGS" -o "$BINARY_NAME" ./apilot-cli

echo "Build complete: ${BINARY_NAME}"

if [[ -z "$INSTALL_DIR" ]]; then
    if [[ -w "/usr/local/bin" ]]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="$HOME/.local/bin"
    fi
fi

mkdir -p "$INSTALL_DIR"

cp "$BINARY_NAME" "$INSTALL_DIR/apilot"
chmod +x "$INSTALL_DIR/apilot"

echo "Installed apilot to ${INSTALL_DIR}/apilot"

if ! command -v apilot &>/dev/null; then
    echo ""
    echo "WARNING: ${INSTALL_DIR} is not in your PATH."
    echo "Add it to your PATH with:"
    echo ""
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    if [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]]; then
        echo "Or add this line to your ~/.bashrc or ~/.zshrc:"
        echo ""
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
fi

echo ""
echo "Verify installation:"
echo "  apilot --version"
