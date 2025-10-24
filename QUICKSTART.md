# caws Quick Start Guide

Get up and running with caws in 5 minutes!

## Prerequisites Check

```bash
# Check if AWS CLI is installed
aws --version
```

If missing:
- **AWS CLI**: https://aws.amazon.com/cli/
  - macOS: `brew install awscli`
  - Ubuntu: `sudo apt install awscli`

That's it! No other dependencies needed.

## Installation

### Option 1: Using the install script (recommended)

```bash
cd caws
./install.sh
```

### Option 2: Manual installation

```bash
cd caws
go build -o caws
sudo mv caws /usr/local/bin/
# or for user install: mv caws ~/.local/bin/
```

### Option 3: Use the pre-built binary

If a `caws` binary already exists:

```bash
cd caws
sudo mv caws /usr/local/bin/
# or: mv caws ~/.local/bin/
```

## Setup (4 steps)

### 1. Initialize your vault

```bash
caws init
```

You'll be prompted to create a master password. **Choose a strong password** - this encrypts all your AWS credentials.

**Important**: There is no password recovery! If you forget it, you'll need to start over.

### 2. Add your first AWS profile

```bash
caws add myprofile
```

Enter when prompted:
- Vault password (the one you just created)
- AWS Access Key ID (from AWS IAM)
- AWS Secret Access Key (input will be hidden)
- Default Region (e.g., `us-east-1`) - optional
- MFA Serial ARN - optional

### 3. Verify it worked

```bash
caws list
```

Enter your vault password. You should see your profile listed.

### 4. Use it!

```bash
caws exec myprofile -- aws sts get-caller-identity
```

If this shows your AWS account info, you're all set! 🎉

## Common Commands

```bash
# List S3 buckets
caws exec myprofile -- aws s3 ls

# Check who you are
caws exec myprofile -- aws sts get-caller-identity

# Run any AWS command
caws exec myprofile -- aws ec2 describe-instances

# Use with Terraform
caws exec myprofile -- terraform plan

# See environment variables
caws exec myprofile -- env | grep AWS
```

## Tips

### 1. Create shell aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias ca='caws'
alias awsexec='caws exec'
```

Then use:

```bash
awsexec myprofile -- aws s3 ls
```

### 2. Add multiple profiles

```bash
caws add personal
caws add work-dev
caws add work-prod
```

All profiles are stored in the same encrypted vault.

### 3. Understanding password prompts

You'll be prompted for your vault password:
- Every time you add/list/remove profiles
- Approximately once per hour when running commands (due to STS caching)

This is normal! Your password is never cached in memory for security.

## Troubleshooting

**"Vault not found"**
→ Run `caws init` first to create your vault

**"Incorrect password or corrupted vault"**
→ Check your password. If forgotten, there's no recovery - you'll need to delete `~/.caws/vault.enc` and start over.

**"Failed to get session token"**
→ Check that AWS CLI is installed and your credentials are valid

**"Profile not found"**
→ The profile doesn't exist, run `caws list` to see available profiles

## What's Happening Behind the Scenes?

1. Your AWS credentials (Access Key + Secret) are stored **encrypted** in `~/.caws/vault.enc`
   - Encryption: Argon2id + AES-256-GCM (industry standard)
   - File permissions: 0600 (only you can read/write)

2. When you run a command, caws:
   - Prompts for your vault password
   - Decrypts your credentials
   - Calls AWS STS to get **temporary credentials** (valid for 1 hour)
   - Caches the temporary credentials locally
   - Runs your command with the temp credentials

3. The next time you run a command within an hour, it uses the cached credentials (but still asks for your vault password)

**Security**: Your long-term AWS credentials are always encrypted and never written to disk in plaintext!

## Next Steps

- Read the full [README.md](README.md) for advanced features
- Set up MFA for extra security
- Add more AWS profiles for different accounts/roles
- Consider backing up `~/.caws/vault.enc` to a secure location

## Getting Help

```bash
caws help
```

Or check the [README.md](README.md) for detailed documentation.

---

**Having issues?** Open an issue on GitHub or check the Troubleshooting section in the README.
