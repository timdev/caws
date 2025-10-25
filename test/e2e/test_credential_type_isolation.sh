#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: credential type isolation (session vs federation)"
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

echo "  Test 1: exec (session) should not be reused by login (federation)"

# First: run exec to cache session credentials
output=$(./caws exec testprofile -- echo "test" 2>&1)
assert_success

# Verify session type cached
cache_type=$(cat "$CAWS_TEST_DIR/cache/testprofile.json" | grep -o '"Type": *"[^"]*"' | cut -d'"' -f4)
if [ "$cache_type" != "session" ]; then
    echo "✗ FAIL: After exec, cache type should be 'session', got '$cache_type'"
    exit 1
fi
echo "    Exec cached 'session' credentials ✓"

# Now: run login - should get NEW federation credentials, not reuse session
output=$(./caws login testprofile 2>&1)
assert_success

# Verify federation type cached
cache_type=$(cat "$CAWS_TEST_DIR/cache/testprofile.json" | grep -o '"Type": *"[^"]*"' | cut -d'"' -f4)
if [ "$cache_type" != "federation" ]; then
    echo "✗ FAIL: After login, cache type should be 'federation', got '$cache_type'"
    exit 1
fi
echo "    Login got new 'federation' credentials ✓"

# Clean cache for reverse test
rm -f "$CAWS_TEST_DIR/cache/testprofile.json"

echo "  Test 2: login (federation) should not be reused by exec (session)"

# First: run login to cache federation credentials
output=$(./caws login testprofile 2>&1)
assert_success

# Verify federation type cached
cache_type=$(cat "$CAWS_TEST_DIR/cache/testprofile.json" | grep -o '"Type": *"[^"]*"' | cut -d'"' -f4)
if [ "$cache_type" != "federation" ]; then
    echo "✗ FAIL: After login, cache type should be 'federation', got '$cache_type'"
    exit 1
fi
echo "    Login cached 'federation' credentials ✓"

# Now: run exec - should get NEW session credentials, not reuse federation
output=$(./caws exec testprofile -- echo "test" 2>&1)
assert_success

if [ -n "$CAWS_MOCK_STS" ]; then
    # In mock mode, just verify it ran
    assert_contains "$output" "test"
else
    # With real AWS, verify we got fresh credentials
    assert_contains "$output" "Getting temporary credentials"
fi

# Verify session type cached
cache_type=$(cat "$CAWS_TEST_DIR/cache/testprofile.json" | grep -o '"Type": *"[^"]*"' | cut -d'"' -f4)
if [ "$cache_type" != "session" ]; then
    echo "✗ FAIL: After exec, cache type should be 'session', got '$cache_type'"
    exit 1
fi
echo "    Exec got new 'session' credentials ✓"

echo "✓ PASS: test_credential_type_isolation"
