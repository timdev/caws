#!/bin/bash

echo "======================================"
echo "caws E2E Test Suite"
echo "======================================"
echo ""

# Detect test mode
if [ -n "$CAWS_TEST_AWS_ACCESS_KEY" ]; then
    echo "✓ Using REAL AWS credentials"
    echo "  Access Key: ${CAWS_TEST_AWS_ACCESS_KEY:0:10}..."
else
    echo "⚠ Using MOCK STS"
    echo "  Set CAWS_TEST_AWS_ACCESS_KEY for real AWS testing"
fi
echo ""

FAILED=0
PASSED=0

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run all test_*.sh files
for test in "$SCRIPT_DIR"/test_*.sh; do
    test_name=$(basename "$test")
    echo "Running $test_name..."

    if bash "$test" 2>&1; then
        ((PASSED++))
    else
        ((FAILED++))
        echo "✗ FAILED: $test_name"
    fi
    echo ""
done

echo "======================================"
echo "Results: $PASSED passed, $FAILED failed"
echo "======================================"

exit $FAILED
