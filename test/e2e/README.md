# E2E Tests for caws

End-to-end tests that exercise the full `caws` CLI binary.

## Running Tests

### Run all e2e tests (mock mode)

```bash
go test ./test/e2e/...
```

### Run with verbose output

```bash
go test -v ./test/e2e/...
```

### Run a specific test

```bash
go test -v ./test/e2e/... -run TestCompleteWorkflow
```

### Run with real AWS credentials

```bash
export CAWS_TEST_AWS_ACCESS_KEY=AKIA...
export CAWS_TEST_AWS_SECRET_KEY=...
go test -v ./test/e2e/...
```

**Note:** Real AWS tests will make actual STS API calls.

## Test Coverage

| Test | Description |
|------|-------------|
| `TestCompleteWorkflow` | Full lifecycle: init → add → list → exec → remove |
| `TestCredentialCaching` | Verifies credential caching and reuse |
| `TestFileLocking` | Ensures concurrent vault access is prevented |
| `TestCLIFlags` | Tests --version, --help, and error handling |
| `TestEnvironmentPropagation` | Verifies all AWS_* env vars are set correctly |
| `TestCredentialTypeIsolation` | Ensures session/federation caches don't interfere |

## Test Modes

### Mock Mode (default)

Uses `CAWS_MOCK_STS=1` to simulate AWS STS calls without hitting real AWS.

- Fast (no network calls)
- No AWS credentials required
- Deterministic output

### Real AWS Mode

Set `CAWS_TEST_AWS_ACCESS_KEY` and `CAWS_TEST_AWS_SECRET_KEY` to use real AWS credentials.

- Makes actual STS API calls
- Verifies real-world behavior
- Requires valid AWS credentials

## Architecture

- **e2e_test.go**: Test infrastructure (TestMain, TestEnv type, helpers)
- **main_test.go**: Actual test implementations

### TestMain

Builds the `caws` binary once before running all tests, improving performance.

### TestEnv

Each test gets an isolated `TestEnv` with:
- Dedicated temp directory (auto-cleanup)
- Isolated vault and cache
- Configurable env vars

### Parallelization

All tests run in parallel using `t.Parallel()` for faster execution.

## What's NOT Tested Here

Items covered by unit tests:
- Encryption/decryption logic
- Input validation (profile names, access keys)
- Config file parsing edge cases
- Key derivation

## Troubleshooting

### Tests fail with "permission denied"

Make sure you're running from the repository root or `test/e2e` directory.

### "vault is locked" errors

Tests should clean up lock files automatically. If you see this, manually remove:
```bash
rm -f /tmp/caws-test-*/vault.enc.lock
```

### Slow tests

- Use mock mode (default) for fast tests
- Tests run in parallel, so total time should be ~2-5 seconds
- Real AWS mode will be slower due to STS API calls
