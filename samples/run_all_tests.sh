#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

FRAMEWORK_FILTER="${1:-all}"

echo "========================================="
echo "Running APilot Integration Tests"
if [ "$FRAMEWORK_FILTER" != "all" ]; then
    echo "Framework Filter: $FRAMEWORK_FILTER"
fi
echo "========================================="
echo ""

declare -A FRAMEWORK_RESULTS
declare -a FRAMEWORK_ORDER

run_framework_tests() {
    local framework_name=$1
    shift
    local samples=("$@")
    
    echo "========================================="
    echo "Framework: $framework_name"
    echo "========================================="
    
    local total=0
    local passed=0
    local failed=0
    local failed_samples=()
    
    for sample in "${samples[@]}"; do
        local dir="$SCRIPT_DIR/$sample"
        if [ -f "$dir/test.sh" ]; then
            echo "----------------------------------------"
            echo "Testing: $sample"
            echo "----------------------------------------"
            
            total=$((total + 1))
            
            if bash "$dir/test.sh"; then
                passed=$((passed + 1))
                echo "✓ $sample PASSED"
            else
                failed=$((failed + 1))
                failed_samples+=("$sample")
                echo "✗ $sample FAILED"
            fi
            echo ""
        fi
    done
    
    echo "----------------------------------------"
    echo "$framework_name Summary: $passed/$total passed"
    if [ $failed -gt 0 ]; then
        echo "Failed samples: ${failed_samples[*]}"
    fi
    echo "----------------------------------------"
    echo ""
    
    FRAMEWORK_RESULTS["$framework_name"]="$passed/$total"
    FRAMEWORK_ORDER+=("$framework_name")
    
    return $failed
}

GO_SAMPLES=("go-echo" "go-fiber" "go-gin")
JAVA_SAMPLES=("java-feign" "java-jaxrs" "java-springmvc")
NODE_SAMPLES=("node-express" "node-fastify" "node-nestjs")
PYTHON_SAMPLES=("python-django" "python-fastapi" "python-flask")

TOTAL_FRAMEWORKS=0
PASSED_FRAMEWORKS=0
FAILED_FRAMEWORKS=0

run_if_matching() {
    local framework_name=$1
    shift
    local samples=("$@")
    
    if [ "$FRAMEWORK_FILTER" = "all" ] || [ "$FRAMEWORK_FILTER" = "$framework_name" ]; then
        run_framework_tests "$framework_name" "${samples[@]}"
        if [ $? -eq 0 ]; then
            PASSED_FRAMEWORKS=$((PASSED_FRAMEWORKS + 1))
        else
            FAILED_FRAMEWORKS=$((FAILED_FRAMEWORKS + 1))
        fi
        TOTAL_FRAMEWORKS=$((TOTAL_FRAMEWORKS + 1))
    fi
}

run_if_matching "Go" "${GO_SAMPLES[@]}"
run_if_matching "Java" "${JAVA_SAMPLES[@]}"
run_if_matching "Node.js" "${NODE_SAMPLES[@]}"
run_if_matching "Python" "${PYTHON_SAMPLES[@]}"

if [ "$FRAMEWORK_FILTER" != "all" ] && [ $TOTAL_FRAMEWORKS -eq 0 ]; then
    echo "Error: Unknown framework '$FRAMEWORK_FILTER'"
    echo "Valid frameworks: Go, Java, Node.js, Python, all"
    exit 1
fi

echo "========================================="
echo "Overall Integration Test Summary"
echo "========================================="
echo ""
echo "Framework Results:"
for framework in "${FRAMEWORK_ORDER[@]}"; do
    echo "  $framework: ${FRAMEWORK_RESULTS[$framework]}"
done
echo ""
echo "Total Frameworks:  $TOTAL_FRAMEWORKS"
echo "Passed Frameworks: $PASSED_FRAMEWORKS"
echo "Failed Frameworks: $FAILED_FRAMEWORKS"
echo "========================================="

if [ $FAILED_FRAMEWORKS -gt 0 ]; then
    exit 1
fi

exit 0
