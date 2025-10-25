#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws list"
setup_test_env

# Init vault
./caws init

# Create config with multiple profiles
cat > "$CAWS_TEST_DIR/config" << EOF
[profile prod]
region = us-east-1

[profile dev]
region = us-west-2
mfa_serial = arn:aws:iam::123456789012:mfa/dev
EOF

# Add profiles to vault
for profile in prod dev; do
    ./caws add $profile << EOF
AKIA${profile}123
secret${profile}123
EOF
done

# List should show both
output=$(./caws list)
assert_contains "$output" "prod"
assert_contains "$output" "dev"
assert_contains "$output" "us-east-1"
assert_contains "$output" "us-west-2"

echo "✓ PASS: test_list"
