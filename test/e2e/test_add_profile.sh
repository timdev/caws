#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws add (with existing config profile)"
setup_test_env

# Create AWS config with profile
cat > "$CAWS_TEST_DIR/config" << EOF
[profile testprofile]
region = us-east-1
mfa_serial = arn:aws:iam::123456789012:mfa/testuser
EOF

# Init vault
./caws init
assert_success

# Add profile
if [ -n "$CAWS_TEST_AWS_ACCESS_KEY" ]; then
    # Use real test credentials
    ./caws add testprofile << EOF
$CAWS_TEST_AWS_ACCESS_KEY
$CAWS_TEST_AWS_SECRET_KEY
EOF
else
    # Use fake credentials
    ./caws add testprofile << EOF
AKIATEST123
fakesecret123
EOF
fi
assert_success

# Verify profile was added
output=$(./caws list)
assert_contains "$output" "testprofile"
assert_contains "$output" "us-east-1"
assert_contains "$output" "MFA enabled"

echo "✓ PASS: test_add_profile"
