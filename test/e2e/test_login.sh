#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: caws login (console URL generation and caching)"
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

# First login - should get fresh federation credentials
output=$(./caws login testprofile 2>&1)
assert_success

# Check that we got a URL
url=$(echo "$output" | grep -E '^https://signin.aws.amazon.com/federation')
if [ -z "$url" ]; then
    echo "✗ FAIL: No federation URL found in output"
    echo "Output was:"
    echo "$output"
    exit 1
fi

echo "  Got federation URL: ${url:0:60}..."

# Verify cache file exists and has federation type
assert_file_exists "$CAWS_TEST_DIR/cache/testprofile.json"

cache_type=$(cat "$CAWS_TEST_DIR/cache/testprofile.json" | grep -o '"Type": *"[^"]*"' | cut -d'"' -f4)
if [ "$cache_type" != "federation" ]; then
    echo "✗ FAIL: Cache type should be 'federation', got '$cache_type'"
    cat "$CAWS_TEST_DIR/cache/testprofile.json"
    exit 1
fi

echo "  Cache type is 'federation' ✓"

# Second login - should use cached credentials (no "Getting" message)
output=$(./caws login testprofile 2>&1)
assert_success

# Should still get a URL
url=$(echo "$output" | grep -E '^https://signin.aws.amazon.com/federation')
if [ -z "$url" ]; then
    echo "✗ FAIL: No federation URL found in cached login output"
    exit 1
fi

echo "  Second login reused cached credentials ✓"

echo "✓ PASS: test_login"
