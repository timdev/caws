# caws Testing

## Quick Start

```bash
# Run all tests (mock mode)
./test/e2e/run_all.sh

# Run with real AWS credentials
export CAWS_TEST_AWS_ACCESS_KEY=AKIA...
export CAWS_TEST_AWS_SECRET_KEY=...
./test/e2e/run_all.sh

# Run single test
./test/e2e/test_init.sh
```

## Test Modes

### Mock Mode (Default)
Fast, no AWS account needed. Uses fake STS credentials.

### Real AWS Mode
Provide test credentials with no permissions. Tests actual AWS integration.

```bash
export CAWS_TEST_AWS_ACCESS_KEY=AKIA...
export CAWS_TEST_AWS_SECRET_KEY=...
```

You can use the following test credentials, which are associated with an IAM with empty permissions:

access: `AKIAWF6WMLRP7EX56ZYK`
secret: `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`

## Test Structure

```
test/
  lib/
    setup.sh          # Test environment setup/teardown
    assertions.sh     # Test assertion helpers (assert_success, assert_contains, etc.)
  e2e/
    test_init.sh                      # Test vault initialization
    test_add_profile.sh               # Test adding profiles with config interaction
    test_add_new_profile.sh           # Test profile creation when not in config
    test_list.sh                      # Test listing profiles
    test_exec_cached.sh               # Test exec with cached credentials
    test_exec_env.sh                  # Test environment variable setting
    test_login.sh                     # Test console login URL generation and caching
    test_credential_type_isolation.sh # Test session vs federation credential isolation
    run_all.sh                        # Run all E2E tests
  bin/
    aws                    # Mock AWS CLI (for CAWS_MOCK_STS=1)
```

## Environment Variables

### Test Mode Control
- `CAWS_TEST_DIR=/tmp/caws-test` - Override all paths to this directory
- `CAWS_PASSWORD=testpass123` - Non-interactive password entry
- `CAWS_AUTO_CONFIRM=yes` - Auto-answer yes/no prompts

### Real AWS Testing
- `CAWS_TEST_AWS_ACCESS_KEY=AKIA...` - Real test AWS access key (no permissions needed)
- `CAWS_TEST_AWS_SECRET_KEY=...` - Real test AWS secret key
- `CAWS_TEST_AWS_REGION=us-east-1` - Test region (default: us-east-1)

### Mocking
- `CAWS_MOCK_STS=1` - Force mock STS (skip real AWS calls)

## Adding New Tests

1. Create `test/e2e/test_<name>.sh`
2. Source setup.sh and assertions.sh
3. Use assert_* functions
4. Make executable: `chmod +x test/e2e/test_<name>.sh`
5. Run with `./test/e2e/run_all.sh`

Example:
```bash
#!/bin/bash
set -e
source "$(dirname "$0")/../lib/setup.sh"
source "$(dirname "$0")/../lib/assertions.sh"

echo "TEST: my new test"
setup_test_env

# Your test code here
./caws init
assert_success

echo "✓ PASS: my_new_test"
```

## Available Assertions

- `assert_success` - Verify last command succeeded (exit code 0)
- `assert_failure` - Verify last command failed (non-zero exit)
- `assert_contains <output> <expected>` - Verify output contains string
- `assert_not_contains <output> <unexpected>` - Verify output doesn't contain string
- `assert_file_exists <path>` - Verify file exists
- `assert_dir_exists <path>` - Verify directory exists

## Benefits

✅ **Fast feedback** - Tests run in <5 seconds (mock) or <30 seconds (real AWS)
✅ **No manual interaction** - Fully automated
✅ **Isolated** - Uses /tmp/caws-test/, never touches real data
✅ **Claude Code friendly** - Simple bash commands, clear output
✅ **Comprehensive** - Covers init, add, list, exec, login, caching, credential type isolation
✅ **Flexible** - Mock or real AWS
✅ **Easy to extend** - Add new test_*.sh files
