#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws init"
setup_test_env

# Run init
./caws init
assert_success

# Verify vault file created
assert_file_exists "$CAWS_TEST_DIR/vault.enc"

# Verify proper permissions
perms=$(stat -f "%OLp" "$CAWS_TEST_DIR/vault.enc" 2>/dev/null || stat -c "%a" "$CAWS_TEST_DIR/vault.enc")
if [ "$perms" != "600" ]; then
    echo "✗ FAIL: Vault permissions should be 600, got $perms"
    exit 1
fi

echo "✓ PASS: test_init"
