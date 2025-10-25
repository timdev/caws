#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws exec (environment variables)"
setup_test_env

# Setup
cat > "$CAWS_TEST_DIR/config" << EOF
[profile testprofile]
region = us-west-2
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

# Check environment variables are set
output=$(./caws exec testprofile -- env)
assert_success

assert_contains "$output" "AWS_ACCESS_KEY_ID="
assert_contains "$output" "AWS_SECRET_ACCESS_KEY="
assert_contains "$output" "AWS_SESSION_TOKEN="
assert_contains "$output" "AWS_VAULT=testprofile"
assert_contains "$output" "AWS_REGION=us-west-2"
assert_contains "$output" "AWS_DEFAULT_REGION=us-west-2"
assert_contains "$output" "AWS_CREDENTIAL_EXPIRATION="

# Verify AWS_PROFILE is NOT set
assert_not_contains "$output" "AWS_PROFILE="

echo "✓ PASS: test_exec_env"
