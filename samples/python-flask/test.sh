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

# Run apilot. The detailed variant is required: the simple one prints routes
# only, so a request body that silently lost its fields would still pass.
echo "Running apilot on $SAMPLE_NAME..."
if ! apilot --formatter markdown --format detailed --output "$(cd "$OUTPUT_DIR" && pwd)/api.md" "$SCRIPT_DIR" 2>&1; then
    echo "Error: apilot command failed"
    exit 1
fi

echo "apilot command completed successfully"

OUTPUT_FILE="$OUTPUT_DIR/api.md"

# Verify output
if [ ! -f "$OUTPUT_FILE" ]; then
    echo "Error: Output file not created"
    exit 1
fi

# Expectations. Routes prove the endpoints were discovered; the field rows
# prove the exported schema survived. Keep these in sync with the sample
# source: a field that disappears from the export must fail here.
MIN_ENDPOINTS=100

EXPECTED_ROUTES=(
    "/users"
    "/users/{user_id}"
)

EXPECTED_FIELDS=(
    "| user_id |  | YES |"
    "| email | string |"
    "| password | string |"
    "| total | int |"
    "| page_num | int |"
    "| page_size | int |"
)

failures=()

endpoint_count=$(grep -c '^\*\*Path:\*\*' "$OUTPUT_FILE" || true)
if [ "$endpoint_count" -lt "$MIN_ENDPOINTS" ]; then
    failures+=("expected at least $MIN_ENDPOINTS endpoints, found $endpoint_count")
fi

for route in "${EXPECTED_ROUTES[@]}"; do
    if ! grep -q "^\*\*Path:\*\* ${route}$" "$OUTPUT_FILE"; then
        failures+=("route not exported: $route")
    fi
done

for field in "${EXPECTED_FIELDS[@]}"; do
    if ! grep -qF -- "$field" "$OUTPUT_FILE"; then
        failures+=("field not exported: $field")
    fi
done

if [ ${#failures[@]} -gt 0 ]; then
    echo "✗ $SAMPLE_NAME test failed"
    for failure in "${failures[@]}"; do
        echo "  - $failure"
    done
    echo "  - Output content:"
    head -60 "$OUTPUT_FILE"
    exit 1
fi

echo "✓ $SAMPLE_NAME test passed"
echo "  - $endpoint_count endpoints exported"
echo "  - ${#EXPECTED_ROUTES[@]} routes and ${#EXPECTED_FIELDS[@]} fields verified"
exit 0
