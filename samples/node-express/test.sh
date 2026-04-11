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

# Debug: show paths
echo "DEBUG: SCRIPT_DIR=$SCRIPT_DIR"
echo "DEBUG: OUTPUT_DIR=$OUTPUT_DIR"
echo "DEBUG: Absolute output path=$(cd "$OUTPUT_DIR" && pwd)/api.md"

# Run apilot
echo "Running apilot on $SAMPLE_NAME..."
if ! apilot "$SCRIPT_DIR" --formatter markdown --output "$(cd "$OUTPUT_DIR" && pwd)/api.md" 2>&1; then
    echo "Error: apilot command failed"
    exit 1
fi

echo "apilot command completed successfully"

# Debug: check if file exists
echo "DEBUG: Checking if output file exists..."
if [ -f "$(cd "$OUTPUT_DIR" && pwd)/api.md" ]; then
    echo "DEBUG: Output file exists at $(cd "$OUTPUT_DIR" && pwd)/api.md"
    echo "DEBUG: File size: $(wc -c < "$(cd "$OUTPUT_DIR" && pwd)/api.md") bytes"
else
    echo "DEBUG: Output file NOT found at $(cd "$OUTPUT_DIR" && pwd)/api.md"
    echo "DEBUG: Listing files in OUTPUT_DIR:"
    ls -la "$OUTPUT_DIR"
fi

# Verify output
if [ ! -f "$OUTPUT_DIR/api.md" ]; then
    echo "Error: Output file not created"
    exit 1
fi

# Check if output contains expected endpoints
if grep -q "GET /users" "$OUTPUT_DIR/api.md" && \
   grep -q "POST /users" "$OUTPUT_DIR/api.md" && \
   grep -q "GET /users/{id}" "$OUTPUT_DIR/api.md" || grep -q "GET /users/:id" "$OUTPUT_DIR/api.md"; then
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
