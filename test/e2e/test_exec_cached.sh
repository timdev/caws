#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws exec (credential caching)"
setup_test_env

# Setup
cat > "$CAWS_TEST_DIR/config" << EOF
[profile testprofile]
region = us-east-1
EOF

./caws init

# Add credentials
if [ -n "$CAWS_TEST_AWS_ACCESS_KEY" ]; then
    ./caws add testprofile << EOF
$CAWS_TEST_AWS_ACCESS_KEY
$CAWS_TEST_AWS_SECRET_KEY
EOF
else
    ./caws add testprofile << EOF
AKIATEST123
fakesecret123
EOF
fi

# First exec - should get fresh credentials
output=$(./caws exec testprofile -- echo "first run")
assert_success
assert_contains "$output" "first run"

if [ -n "$CAWS_MOCK_STS" ]; then
    # In mock mode, just verify command ran
    assert_contains "$output" "first run"
else
    # With real AWS, verify STS was called
    assert_contains "$output" "Getting temporary credentials"
fi

# Second exec - should use cached credentials
output=$(./caws exec testprofile -- echo "second run")
assert_success
assert_contains "$output" "second run"
assert_contains "$output" "Using cached credentials"

# Verify cache file exists
assert_file_exists "$CAWS_TEST_DIR/cache/testprofile.json"

echo "✓ PASS: test_exec_cached"
