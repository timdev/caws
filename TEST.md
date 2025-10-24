# Manual Testing Instructions for bw-aws (gopass backend)

## Prerequisites

1. **gopass** must be installed and initialized
2. **GPG** must be configured with a working key
3. **AWS CLI** must be installed

## Setup Isolated Test Environment

```bash
# Create isolated password store for testing
export TEST_STORE=/tmp/bw-aws-test-store
rm -rf $TEST_STORE

# Initialize gopass with your GPG key (replace with your key ID)
PASSWORD_STORE_DIR=$TEST_STORE gopass init --path $TEST_STORE <your-gpg-key-id>

# Or if you want to use your default key:
PASSWORD_STORE_DIR=$TEST_STORE gopass init --path $TEST_STORE
```

## Test 1: Add a Profile

```bash
PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws add testprofile
```

Enter the test AWS credentials:
- Access Key: `AKIAWF6WMLRP7EX56ZYK`
- Secret Key: `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`
- Region: `us-west-2`
- MFA: (leave empty)

**Expected:** Success message about profile being added to gopass

## Test 2: List Profiles

```bash
PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws list
```

**Expected:** Should show `testprofile` with region `us-west-2`

## Test 3: Verify with gopass CLI

```bash
PASSWORD_STORE_DIR=$TEST_STORE gopass show aws/testprofile
```

**Expected:** Should show the password and fields including `access_key`, `secret_key`, `region`

## Test 4: Execute Command (Performance Test)

```bash
# Clear any existing cache
rm -rf ~/.bw-aws/cache/testprofile.json

# First run (will call STS)
time PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws exec testprofile -- aws sts get-caller-identity
```

**Expected:**
- "Getting temporary credentials..." message
- STS call succeeds
- Shows Account and ARN
- Total time: < 5 seconds (most is AWS STS call)

## Test 5: Cached Credentials (Fast Path)

```bash
# Second run (should use cache)
time PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws exec testprofile -- aws sts get-caller-identity
```

**Expected:**
- "Using cached credentials..." message
- Total time: < 500ms (this is the key performance test!)

## Test 6: Simple Command Execution

```bash
PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws exec testprofile -- echo "Success!"
```

**Expected:**
- Uses cached credentials
- Prints "Success!"
- Fast execution

## Test 7: Remove Profile

```bash
PASSWORD_STORE_DIR=$TEST_STORE ./bw-aws remove testprofile
```

Type `yes` to confirm.

**Expected:** Profile removed successfully

## Performance Comparison

### Before (Bitwarden CLI):
- Cold start: ~8 seconds
- With cache: ~2.8 seconds

### After (gopass library):
- Cold start: ~3-5 seconds (mostly AWS STS call)
- With cache: **< 500ms** (10-20x faster!)

## Cleanup

```bash
rm -rf $TEST_STORE
rm -rf ~/.bw-aws/cache/testprofile.json
```

## Troubleshooting

### "No secret key" or GPG errors
- Make sure your GPG agent is running: `gpg-agent`
- Check GPG_TTY is set: `export GPG_TTY=$(tty)`
- Try: `gpgconf --kill gpg-agent` to restart agent

### "Failed to open gopass store"
- Make sure gopass is initialized: `gopass ls`
- Check PASSWORD_STORE_DIR points to valid store

### "Profile not found"
- Use `PASSWORD_STORE_DIR=$TEST_STORE gopass ls` to see what's actually stored
- Profiles are stored under `aws/<profile-name>`

## Using with Your Real gopass Store

Once testing is complete, you can use bw-aws with your default gopass store by simply omitting `PASSWORD_STORE_DIR`:

```bash
# Add to your real store
./bw-aws add production

# Use it
./bw-aws exec production -- aws s3 ls
```

Credentials will be stored at `~/.local/share/gopass/stores/root/aws/<profile>`.
