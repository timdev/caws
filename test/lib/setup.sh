#!/bin/bash

setup_test_env() {
    export CAWS_TEST_DIR="/tmp/caws-test"
    export CAWS_PASSWORD="testpassword123"
    export CAWS_AUTO_CONFIRM="yes"

    # Clean and recreate test directory
    rm -rf "$CAWS_TEST_DIR"
    mkdir -p "$CAWS_TEST_DIR/cache"

    # If real AWS credentials provided, use them
    if [ -n "$CAWS_TEST_AWS_ACCESS_KEY" ]; then
        echo "→ Using real AWS credentials for testing"
    else
        echo "→ Using mock STS (set CAWS_TEST_AWS_ACCESS_KEY for real AWS testing)"
        export CAWS_MOCK_STS=1
        # Add mock aws to PATH
        export PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/test/bin:$PATH"
    fi
}

cleanup_test_env() {
    rm -rf "$CAWS_TEST_DIR"
    unset CAWS_TEST_DIR CAWS_PASSWORD CAWS_AUTO_CONFIRM
    unset CAWS_MOCK_STS
}

# Trap to ensure cleanup on exit
trap cleanup_test_env EXIT
