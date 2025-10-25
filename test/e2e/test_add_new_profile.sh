#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws add (profile not in config - should create)"
setup_test_env

# Init vault
./caws init

# Add profile that doesn't exist in config yet
# Auto-confirm should say "yes" to creating it
if [ -n "$CAWS_TEST_AWS_ACCESS_KEY" ]; then
    ./caws add newprofile << EOF
$CAWS_TEST_AWS_ACCESS_KEY
$CAWS_TEST_AWS_SECRET_KEY
EOF
else
    ./caws add newprofile << EOF
AKIANEW123
newsecret123
EOF
fi
assert_success

# Verify config file was created with profile
assert_file_exists "$CAWS_TEST_DIR/config"
grep -q "\[profile newprofile\]" "$CAWS_TEST_DIR/config"
assert_success

echo "✓ PASS: test_add_new_profile"
