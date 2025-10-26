# Usage Guide

Complete reference for all caws commands and advanced usage patterns.

## Command Reference

### `caws init`

Initialize a new encrypted vault.

**Usage:**
```bash
caws init
```

**Prompts:**
- Master password (hidden input)
- Confirm password (hidden input)

**Behavior:**
- Creates `~/.local/share/caws/vault.enc` with proper permissions (0600)
- Fails if vault already exists (no overwrite)
- Password must be entered twice (confirmation)

**Output:**
```
Vault initialized successfully at /home/user/.caws/vault.enc
```

**Notes:**
- Run this once before using caws
- Choose a strong password (see [Security](SECURITY.md))
- No password recovery exists (backup your password!)

**Example:**
```bash
$ caws init
Enter master password: ************
Confirm password: ************
Vault initialized successfully at /home/user/.caws/vault.enc
```

---

### `caws add <profile>`

Add a new AWS profile to the vault.

**Usage:**
```bash
caws add PROFILE_NAME
```

**Arguments:**
- `PROFILE_NAME` - Name for this profile (e.g., "production", "dev")

**Prompts:**
1. Vault password (to decrypt vault)
2. AWS Access Key ID (e.g., `AKIA...`)
3. AWS Secret Access Key (hidden input)
4. Default Region (optional, e.g., `us-east-1`)
5. MFA Serial ARN (optional, e.g., `arn:aws:iam::123456789012:mfa/user`)

**Behavior:**
- Decrypts vault with password
- Adds new profile to vault
- Re-encrypts and saves vault
- Uses atomic write (`.tmp` then rename)

**Output:**
```
Profile 'production' added to vault successfully.
```

**Notes:**
- Profile name must be unique
- Use `-` or `_` in profile names (no spaces)
- Secret Access Key input is hidden
- MFA serial is optional but recommended for production

**Example:**
```bash
$ caws add production
Enter vault password: ************
Access Key ID: AKIAWF6WMLRP7EX56ZYK
Secret Access Key: (hidden input)
Default Region (optional): us-east-1
MFA Serial ARN (optional): arn:aws:iam::123456789012:mfa/alice
Profile 'production' added to vault successfully.
```

**Example (minimal):**
```bash
$ caws add dev
Enter vault password: ************
Access Key ID: AKIAWF6WMLRP9ABC1234
Secret Access Key: (hidden input)
Default Region (optional):
MFA Serial ARN (optional):
Profile 'dev' added to vault successfully.
```

---

### `caws list`

List all AWS profiles stored in the vault.

**Usage:**
```bash
caws list
```

**Prompts:**
- Vault password (to decrypt vault)

**Behavior:**
- Decrypts vault
- Displays all profiles with metadata
- Shows region and MFA status

**Output:**
```
Available AWS profiles:
  • production (region: us-east-1) [MFA enabled]
  • development (region: us-west-2)
  • staging
```

**Notes:**
- Does not display credentials (only metadata)
- Requires password each time (vault not cached)
- Profiles with no region show as bare name

**Example:**
```bash
$ caws list
Enter vault password: ************
Available AWS profiles:
  • production (region: us-east-1) [MFA enabled]
  • dev
```

---

### `caws exec <profile> -- <command>`

Execute a command with AWS credentials injected as environment variables.

**Usage:**
```bash
caws exec PROFILE_NAME -- COMMAND [ARGS...]
```

**Arguments:**
- `PROFILE_NAME` - Profile to use (from vault)
- `--` - Separator (required)
- `COMMAND` - Command to execute
- `ARGS` - Arguments for the command

**Prompts:**
1. Vault password (to decrypt credentials)
2. MFA code (if profile has MFA enabled and cache expired)

**Behavior:**
1. Decrypt vault and retrieve profile credentials
2. Check cache for valid temporary credentials
3. If no cache or expired:
   - Call AWS STS `GetSessionToken`
   - Prompt for MFA code if configured
   - Cache temporary credentials
4. Inject credentials as environment variables
5. Execute command
6. Return command's exit code

**Environment variables set:**
- `AWS_ACCESS_KEY_ID` - Temporary access key
- `AWS_SECRET_ACCESS_KEY` - Temporary secret key
- `AWS_SESSION_TOKEN` - Session token
- `AWS_REGION` - Region (if configured in profile)

**Output:**
```
(command output)
```

**Notes:**
- Password required every time (not cached)
- STS credentials cached for ~55 minutes
- Command exit code preserved
- Works with any AWS-aware tool

**Examples:**

**Check identity:**
```bash
$ caws exec production -- aws sts get-caller-identity
Enter vault password: ************
{
    "UserId": "AIDAI...",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/alice"
}
```

**List S3 buckets:**
```bash
$ caws exec production -- aws s3 ls
Enter vault password: ************
2025-01-15 10:30:00 my-bucket
2025-02-20 14:15:00 another-bucket
```

**Run Terraform:**
```bash
$ caws exec dev -- terraform plan
Enter vault password: ************
Terraform will perform the following actions:
  ...
```

**Check environment variables:**
```bash
$ caws exec production -- env | grep AWS
Enter vault password: ************
AWS_ACCESS_KEY_ID=ASIA...
AWS_SECRET_ACCESS_KEY=...
AWS_SESSION_TOKEN=...
AWS_REGION=us-east-1
```

**Run script:**
```bash
$ caws exec production -- ./deploy.sh
Enter vault password: ************
Deploying to production...
```

---

### `caws remove <profile>`

Remove a profile from the vault.

**Usage:**
```bash
caws remove PROFILE_NAME
```

**Arguments:**
- `PROFILE_NAME` - Profile to remove

**Prompts:**
1. Vault password (to decrypt vault)
2. Confirmation (type "yes")

**Behavior:**
- Decrypts vault
- Confirms deletion
- Removes profile from vault
- Re-encrypts and saves vault

**Output:**
```
Profile 'old-dev' removed from vault.
```

**Notes:**
- Irreversible (no undo)
- Does not clear cache (do manually if needed)
- Requires confirmation to prevent accidents

**Example:**
```bash
$ caws remove old-dev
Enter vault password: ************
Are you sure you want to remove profile 'old-dev'? (yes/no): yes
Profile 'old-dev' removed from vault.
```

**Clear cache after removal:**
```bash
$ caws remove old-dev
... (prompts)
$ rm ~/.cache/caws/old-dev.json
```

---

## Advanced Usage

### MFA Support

If your AWS IAM user requires MFA, add the MFA device ARN when creating the profile.

**Setup:**

1. Find your MFA serial ARN in AWS IAM Console:
   ```
   arn:aws:iam::123456789012:mfa/your-username
   ```

2. Add profile with MFA:
   ```bash
   caws add production
   Enter vault password: ************
   Access Key ID: AKIA...
   Secret Access Key: (hidden)
   Default Region (optional): us-east-1
   MFA Serial ARN (optional): arn:aws:iam::123456789012:mfa/alice
   ```

**Usage:**

When executing commands, you'll be prompted for MFA code:

```bash
$ caws exec production -- aws s3 ls
Enter vault password: ************
Getting temporary credentials...
Enter MFA code: 123456
2025-01-15 10:30:00 my-bucket
```

**Notes:**
- MFA code required when STS credentials expire (~1 hour)
- Use cached credentials within the hour (no MFA prompt)
- MFA code is 6 digits from your authenticator app

**MFA code sources:**
- Google Authenticator
- Authy
- 1Password
- Hardware MFA device

---

### Multiple Profiles

Manage multiple AWS accounts or roles with different profiles.

**Setup:**
```bash
caws add personal
caws add work-dev
caws add work-staging
caws add work-prod
```

**Usage:**
```bash
# Personal account
caws exec personal -- aws s3 ls

# Work development
caws exec work-dev -- terraform apply

# Work production (with MFA)
caws exec work-prod -- aws ec2 describe-instances
```

**Benefits:**
- Separate credentials for different accounts
- Different MFA requirements per profile
- Easy to switch contexts
- All encrypted in one vault

**List all profiles:**
```bash
$ caws list
Available AWS profiles:
  • personal (region: us-west-2)
  • work-dev (region: us-east-1)
  • work-staging (region: us-east-1) [MFA enabled]
  • work-prod (region: us-east-1) [MFA enabled]
```

---

### Password Entry Frequency

With STS credential caching:

**When you'll be prompted for password:**
- Every vault operation (`list`, `add`, `remove`)
- Every `exec` command (even with cached STS credentials)
- Approximately once per hour for sustained usage

**When you'll be prompted for MFA code:**
- When STS credentials expire (~1 hour)
- First `exec` command after credential expiration
- Not needed for commands within the cache window

**Example session:**
```bash
# First command - password + MFA
$ caws exec prod -- aws s3 ls
Enter vault password: ************
Getting temporary credentials...
Enter MFA code: 123456
...

# Second command (within 55 min) - password only, no MFA
$ caws exec prod -- aws s3 ls
Enter vault password: ************
Using cached credentials...
...

# After 1 hour - password + MFA again
$ caws exec prod -- aws s3 ls
Enter vault password: ************
Getting temporary credentials...
Enter MFA code: 789012
...
```

---

### Credential Caching

Temporary credentials are cached to minimize AWS STS API calls.

**Cache location:**
```
~/.cache/caws/<profile>.json
```

**Cache format:**
```json
{
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "...",
  "SessionToken": "...",
  "Expiration": "2025-10-26T15:30:00Z"
}
```

**Cache behavior:**
- Created after first STS call
- Valid for ~55 minutes (1 hour minus 5-minute buffer)
- Checked before every STS call
- Automatically refreshed when expired

**Clear cache:**

```bash
# Clear specific profile
rm ~/.cache/caws/production.json

# Clear all caches
rm -rf ~/.cache/caws/

# Clear everything (vault + cache)
rm -rf ~/.local/share/caws/ ~/.cache/caws/
```

**When to clear cache:**
- Testing credential rotation
- Security concern (cache may be compromised)
- Debugging STS issues
- Finishing work on shared system

---

### Shell Integration

Create shell aliases for easier usage.

**Bash/Zsh (~/.bashrc or ~/.zshrc):**

```bash
# Short alias for caws
alias ca='caws'

# Function for exec
aws-exec() {
    caws exec "$@"
}

# Profile-specific aliases
alias aws-prod='caws exec production --'
alias aws-dev='caws exec development --'
```

**Usage with aliases:**

```bash
# Using short alias
ca list

# Using exec function
aws-exec production -- aws s3 ls

# Using profile alias
aws-prod aws ec2 describe-instances
aws-dev terraform plan
```

**Fish (~/.config/fish/config.fish):**

```fish
# Short alias
alias ca='caws'

# Function for exec
function aws-exec
    caws exec $argv
end

# Profile-specific
alias aws-prod='caws exec production --'
```

---

### Working with Specific AWS Services

**S3:**
```bash
caws exec prod -- aws s3 ls
caws exec prod -- aws s3 cp file.txt s3://bucket/
caws exec prod -- aws s3 sync ./local/ s3://bucket/path/
```

**EC2:**
```bash
caws exec prod -- aws ec2 describe-instances
caws exec prod -- aws ec2 start-instances --instance-ids i-1234567890abcdef0
```

**IAM:**
```bash
caws exec prod -- aws iam list-users
caws exec prod -- aws iam get-user
```

**Terraform:**
```bash
caws exec prod -- terraform init
caws exec prod -- terraform plan
caws exec prod -- terraform apply
```

**AWS CDK:**
```bash
caws exec prod -- cdk deploy
caws exec prod -- cdk diff
```

**Serverless Framework:**
```bash
caws exec prod -- serverless deploy
caws exec prod -- serverless invoke -f functionName
```

---

### Debugging

**Check which credentials are being used:**
```bash
caws exec prod -- aws sts get-caller-identity
```

**Verify environment variables:**
```bash
caws exec prod -- env | grep AWS
```

**Test with a simple command:**
```bash
caws exec prod -- echo "Success!"
```

**Check cache contents:**
```bash
cat ~/.cache/caws/production.json | jq .
```

**Verify vault file:**
```bash
ls -la ~/.local/share/caws/vault.enc
cat ~/.local/share/caws/vault.enc | jq .
```

---

### Using with Docker

Run AWS CLI in Docker with caws credentials:

```bash
caws exec prod -- docker run --rm \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e AWS_SESSION_TOKEN \
  -e AWS_REGION \
  amazon/aws-cli s3 ls
```

---

### Scripting with caws

**Build script:**
```bash
#!/bin/bash
set -e

echo "Building application..."
go build -o myapp

echo "Deploying to S3..."
caws exec prod -- aws s3 cp myapp s3://my-bucket/releases/

echo "Deploying to EC2..."
caws exec prod -- aws ec2 run-instances ...

echo "Done!"
```

**CI/CD integration:**
```bash
# Note: For CI/CD, consider using IAM roles instead of long-term credentials
# caws is designed for local development, not CI/CD pipelines

# If you must use caws in CI:
export CAWS_PASSWORD="$VAULT_PASSWORD_SECRET"
export CAWS_AUTO_CONFIRM=yes
./caws exec prod -- aws s3 sync ./dist/ s3://website-bucket/
```

---

## Command Exit Codes

caws preserves the exit code of the executed command.

**Success:**
```bash
$ caws exec prod -- aws s3 ls
$ echo $?
0
```

**Failure:**
```bash
$ caws exec prod -- aws s3 ls s3://nonexistent-bucket
$ echo $?
1
```

**Command not found:**
```bash
$ caws exec prod -- nonexistent-command
$ echo $?
127
```

This allows caws to work seamlessly in scripts with error handling:

```bash
#!/bin/bash
set -e  # Exit on any error

caws exec prod -- aws s3 cp file.txt s3://bucket/ || {
  echo "Upload failed!"
  exit 1
}
```

---

## Tips and Tricks

### Avoid Re-entering Password

While caws doesn't cache passwords for security, you can minimize password entry:

**1. Batch commands:**
```bash
# Bad: Password entered twice
caws exec prod -- aws s3 ls
caws exec prod -- aws ec2 describe-instances

# Better: Use a script
caws exec prod -- bash -c '
  aws s3 ls
  aws ec2 describe-instances
'
```

**2. Use shell sessions:**
```bash
# Enter password once, run multiple commands
caws exec prod -- bash
# Now you're in a shell with credentials
aws s3 ls
aws ec2 describe-instances
terraform plan
exit
```

### Check Credential Expiration

```bash
cat ~/.cache/caws/production.json | jq -r '.Expiration'
```

### Rotate Credentials

```bash
# Create new access key in AWS IAM Console first!
caws remove production
caws add production
# Enter new credentials
```

### Backup Vault

```bash
cp ~/.local/share/caws/vault.enc ~/Backups/caws-vault-$(date +%Y%m%d).enc
```

### Test New Profile

```bash
caws add test-profile
caws exec test-profile -- aws sts get-caller-identity
# Verify it's the right account
```

---

## Limitations

### Current Limitations

1. **No IAM role assumption**
   - Can only use direct credentials (not `sts:AssumeRole`)
   - Planned for future release

2. **Single vault**
   - All profiles in one vault
   - Cannot have separate vaults per client/project
   - Planned: `--vault` flag

3. **No password caching**
   - Password required for every command
   - Security trade-off (no plaintext password in memory)

4. **Manual credential rotation**
   - No automatic enforcement of rotation policies
   - Must manually update credentials

5. **No Windows support for secret input**
   - Uses `stty` for hiding input (Unix-specific)
   - May have issues on Windows

---

## Getting Help

**View help:**
```bash
caws --help
caws help
```

**Check version (if implemented):**
```bash
caws --version
```

**Troubleshooting:**
See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.

**Security:**
See [SECURITY.md](SECURITY.md) for security best practices.

**Architecture:**
See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details.
