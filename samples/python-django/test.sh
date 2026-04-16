#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SAMPLE_NAME=$(basename "$SCRIPT_DIR")

echo "Testing $SAMPLE_NAME..."

# Check if apilot binary exists
if ! command -v apilot &> /dev/null; then
    echo "Error: apilot binary not found in PATH"
    echo "Please build apilot first: go build -o apilot ./apilot-cli"
    exit 1
fi

# Create output directory
OUTPUT_DIR="$SCRIPT_DIR/.output"
mkdir -p "$OUTPUT_DIR"

# Run apilot
echo "Running apilot on $SAMPLE_NAME..."
if ! apilot --formatter markdown --output "$(cd "$OUTPUT_DIR" && pwd)/api.md" "$SCRIPT_DIR" 2>&1; then
    echo "Error: apilot command failed"
    exit 1
fi

echo "apilot command completed successfully"

# Verify output
if [ ! -f "$OUTPUT_DIR/api.md" ]; then
    echo "Error: Output file not created"
    exit 1
fi

# Check if output contains expected endpoints
# The parser extracts endpoints from both urlpatterns and decorated views
if grep -q "GET /users" "$OUTPUT_DIR/api.md" && \
   (grep -q "GET /users/{pk}" "$OUTPUT_DIR/api.md" || grep -q "GET /users/:pk" "$OUTPUT_DIR/api.md"); then
    echo "✓ $SAMPLE_NAME test passed"
    echo "  - Found expected endpoints in output"
    exit 0
else
    echo "✗ $SAMPLE_NAME test failed"
    echo "  - Expected endpoints not found in output"
    echo "  - Output content:"
    cat "$OUTPUT_DIR/api.md"
    exit 1
fi
