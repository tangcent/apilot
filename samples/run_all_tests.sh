#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "========================================="
echo "Running APilot Integration Tests"
echo "========================================="
echo ""

# Track results
TOTAL=0
PASSED=0
FAILED=0

# Run tests for each sample project
for dir in "$SCRIPT_DIR"/*/; do
    if [ -f "$dir/test.sh" ]; then
        SAMPLE_NAME=$(basename "$dir")
        echo "----------------------------------------"
        echo "Testing: $SAMPLE_NAME"
        echo "----------------------------------------"
        
        TOTAL=$((TOTAL + 1))
        
        if bash "$dir/test.sh"; then
            PASSED=$((PASSED + 1))
        else
            FAILED=$((FAILED + 1))
            echo "✗ $SAMPLE_NAME FAILED"
        fi
        echo ""
    fi
done

# Print summary
echo "========================================="
echo "Integration Test Summary"
echo "========================================="
echo "Total:  $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "========================================="

# Exit with error if any test failed
if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
