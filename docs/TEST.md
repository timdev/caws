# Manual Testing Instructions for caws

## Prerequisites

1. **AWS CLI** must be installed
2. Go 1.22+ (for building)

That's it! No gopass, no GPG needed.

## Setup Test Environment

```bash
# Build the binary
go build -o caws

# Clean up any existing vault/cache for fresh test
rm -rf ~/.caws
```

## Test 1: Initialize Vault

```bash
./caws init
```

**Prompts:**
- Enter master password: `testpassword`
- Confirm password: `testpassword`

**Expected:** Success message with vault location `~/.caws/vault.enc`

## Test 2: Add a Profile

```bash
./caws add testprofile
```

Enter:
- Vault password: `testpassword`
- Access Key: `AKIAWF6WMLRP7EX56ZYK` (test key, won't work with real AWS)
- Secret Key: `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`
- Region: `us-west-2`
- MFA: (leave empty)

**Expected:** Success message about profile being added to vault

## Test 3: List Profiles

```bash
./caws list
```

**Prompts:** Vault password: `testpassword`

**Expected:** Should show `testprofile` with region `us-west-2`

## Test 4: Verify Vault File

```bash
# Check vault exists with correct permissions
ls -la ~/.caws/vault.enc
```

**Expected:**
- File exists
- Permissions: `-rw-------` (0600)
- File is JSON (you can `cat` it, but it's encrypted)

```bash
# View encrypted vault (won't be readable)
cat ~/.caws/vault.enc
```

**Expected:** JSON with `version`, `salt`, `nonce`, `data` fields (all base64)

## Test 5: Execute Command (Performance Test)

**Note:** This requires valid AWS credentials. If using test credentials, this will fail at STS step.

```bash
# First run (will call STS)
time ./caws exec testprofile -- aws sts get-caller-identity
```

**Expected (with valid credentials):**
- Vault password prompt
- "Getting temporary credentials..." message
- STS call succeeds
- Shows Account and ARN
- Total time: < 2 seconds

**Expected (with test credentials):**
- Will fail at STS step (expected, since test credentials aren't real)

## Test 6: Cached Credentials (Fast Path)

**Note:** Only works with valid AWS credentials

```bash
# Second run (should use cache)
time ./caws exec testprofile -- aws sts get-caller-identity
```

**Expected (with valid credentials):**
- Vault password prompt (password never cached)
- "Using cached credentials..." message
- Total time: < 200ms

## Test 7: Simple Command Execution

```bash
./caws exec testprofile -- echo "Success!"
```

**Expected:**
- Vault password prompt
- Uses cached credentials (if available)
- Prints "Success!"

## Test 8: Wrong Password

```bash
./caws list
```

Enter wrong password.

**Expected:** Error message "incorrect password or corrupted vault"

## Test 9: Add Second Profile

```bash
./caws add secondprofile
```

Enter different credentials.

**Expected:** Both profiles now in vault

```bash
./caws list
```

**Expected:** Shows both `testprofile` and `secondprofile`

## Test 10: Remove Profile

```bash
./caws remove testprofile
```

Type `yes` to confirm.

**Expected:** Profile removed successfully

```bash
./caws list
```

**Expected:** Only `secondprofile` remains

## Performance Comparison

### Current implementation (password-based):
- Vault decrypt: ~10-50ms
- Cold start (with STS): ~1.4s (mostly AWS STS call)
- Warm start (cached STS): ~0.1s (no vault decrypt needed)

### Comparison to gopass (previous version):
- Similar performance
- No external dependencies (major win)

## Security Verification

```bash
# Check file permissions
ls -la ~/.caws/vault.enc
ls -la ~/.caws/cache/

# Verify vault is encrypted (should be unreadable)
cat ~/.caws/vault.enc | jq .

# Verify no plaintext credentials anywhere
grep -r "AKIA" ~/.caws/ 2>/dev/null
```

**Expected:**
- Vault: 0600 permissions
- Cache dir: 0700 permissions
- No plaintext long-term credentials

## Cleanup

```bash
rm -rf ~/.caws
rm ./caws
```

## Testing with Real AWS Credentials

Once you're ready to test with real credentials:

```bash
# Build
go build -o caws

# Initialize
./caws init
# (choose strong password)

# Add your real profile
./caws add production
# (enter your real AWS credentials)

# Test
./caws exec production -- aws sts get-caller-identity
```

**Expected:** Should show your real AWS account info

## Troubleshooting

### "Vault not found"
- Run `./caws init` first

### "Incorrect password or corrupted vault"
- Check you're entering the correct password
- If forgotten, no recovery is possible - must reinitialize

### "Failed to get session token"
- Check AWS CLI is installed: `aws --version`
- Check credentials are valid (Access Key format starts with `AKIA`)
- If using MFA, enter the correct current code

### "Profile not found"
- Run `./caws list` to see available profiles
- Profile may have been removed or never created

## Advanced Testing

### Test Password Security

Try to extract credentials without password:

```bash
# Vault is encrypted - should not be readable
strings ~/.caws/vault.enc | grep -i "AKIA"
```

**Expected:** No credential data visible (only base64 gibberish)

### Test Atomic Writes

```bash
# Watch for .tmp file during writes
ls -la ~/.caws/*.tmp

# Should see .tmp briefly during add/remove operations
# Then it gets renamed to vault.enc
```

### Test Cache Expiration

```bash
# Execute command, check cache
./caws exec testprofile -- echo "test"
cat ~/.caws/cache/testprofile.json | jq .Expiration

# Wait for expiration (or modify file to past time)
# Try again - should call STS
```

## Testing Notes

- No unit tests currently exist
- All testing is manual/integration testing
- Test with fake credentials first, real credentials second
- Always test password security (no plaintext leaks)
