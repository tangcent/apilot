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
MIN_ENDPOINTS=110

EXPECTED_ROUTES=(
    "/users"
    "/users/{id}"
    "/users/{id}/avatar"
    "/api/payments/callback"
)

EXPECTED_FIELDS=(
    "| id |  | YES |"
    "| startDate |  | NO |"
    "| endDate |  | NO |"
    "| granularity |"
    "| code | int |"
    "| message | string |"
    "| granularity | day | NO |"
    # A scalar body keeps its parameter name instead of an empty name cell.
    "| callbackData | string |"
    # A collection of scalars keeps the parameter name and reports the element
    # type on an indexed row: ids[0] with long, never [0] with long[].
    "| ids[0] | long |"
    # The declared Java type rides in the type column, so a Long id is
    # distinguishable from a String id (issue #169).
    "| id |  | YES | Long |"
    "| keyword |  | NO | String |"
    # A MultipartFile argument is a multipart form part: it belongs under Form
    # with type file, and the mapping declares the request media type.
    "| file |  | YES | file |"
    "**Content-Type:** multipart/form-data"
    "**Accept:** application/json"
)

failures=()

endpoint_count=$(grep -c '^\*\*Path:\*\*' "$OUTPUT_FILE" || true)
if [ "$endpoint_count" -lt "$MIN_ENDPOINTS" ]; then
    failures+=("expected at least $MIN_ENDPOINTS endpoints, found $endpoint_count")
fi

# The document has to say what it is: outside its own directory an export
# carrying just "# API" cannot be told apart from any other (issue #170).
first_line=$(head -1 "$OUTPUT_FILE")
if [[ "$first_line" != *"# $SAMPLE_NAME API"* ]]; then
    failures+=("heading should name the project, got: $first_line")
fi
if [[ "$first_line" != *"java / spring-mvc"* ]]; then
    failures+=("heading should name the collector and framework, got: $first_line")
fi
if [[ "$first_line" != *"$endpoint_count endpoints"* ]]; then
    failures+=("heading should carry the endpoint count, got: $first_line")
fi

# Resources share method names here, so a heading built from the name alone
# repeats. Every ### has to be addressable on its own (issue #170).
duplicate_headings=$(grep '^### ' "$OUTPUT_FILE" | sort | uniq -d)
if [ -n "$duplicate_headings" ]; then
    failures+=("endpoint headings are not unique: $duplicate_headings")
fi

# The > BASIC marker was emitted once per endpoint carrying nothing.
if grep -q '^> BASIC' "$OUTPUT_FILE"; then
    failures+=("the inert '> BASIC' marker should no longer be exported")
fi

# Fields follow declaration order, so a wrapper declared code, message, data
# is exported that way round and not as code, data, message (issue #170).
message_line=$(grep -n -m1 -F '| message | string |' "$OUTPUT_FILE" | cut -d: -f1)
data_line=$(grep -n -m1 -F '| data | object |' "$OUTPUT_FILE" | cut -d: -f1)
if [ -z "$message_line" ] || [ -z "$data_line" ]; then
    failures+=("expected the code/message/data wrapper rows to be exported")
elif [ "$message_line" -ge "$data_line" ]; then
    failures+=("fields should follow declaration order: message before data, got message at $message_line and data at $data_line")
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
