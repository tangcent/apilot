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
    "/users/{id}"
    "/api/categories/{id}/detail"
    "/api/categories/{id}/status"
    "/api/categories/{id}/cache"
)

EXPECTED_FIELDS=(
    "| id |  | YES |"
    "| startDate |  | NO |"
    "| endDate |  | NO |"
    "| granularity |"
    "| code | int |"
    "| message | string |"
    # Category fields reached through Response.ok(Result.success(category)):
    # the payload is only visible in the method body.
    "| &ensp;&ensp;&#124;─parentId | long |"
    "| &ensp;&ensp;&#124;─icon | string |"
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

# Every endpoint documents its response. A JAX-RS handler returning Response
# builds its payload in the method body, so a missing section means the
# collector dropped it instead of marking it opaque (issue #164).
response_count=$(grep -c '^> RESPONSE$' "$OUTPUT_FILE" || true)
if [ "$response_count" -ne "$endpoint_count" ]; then
    failures+=("$((endpoint_count - response_count)) endpoint(s) exported without a response section")
fi

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
echo "  - $endpoint_count endpoints exported, $response_count with a response section"
echo "  - ${#EXPECTED_ROUTES[@]} routes and ${#EXPECTED_FIELDS[@]} fields verified"
exit 0
