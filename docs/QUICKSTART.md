# caws Quick Start Guide

Get up and running with caws in 5 minutes!

## Prerequisites

**None!** caws is a fully self-contained binary with zero runtime dependencies. Just download and run!

## Installation

### Option 1: Download Binary (Recommended)

Download the latest release for your platform:

| Platform | Architecture | Download |
|----------|-------------|----------|
| Linux | x86_64 | [Download](https://github.com/timdev/caws/releases/latest/download/caws-linux-amd64) |
| Linux | ARM64 | [Download](https://github.com/timdev/caws/releases/latest/download/caws-linux-arm64) |
| macOS | Intel | [Download](https://github.com/timdev/caws/releases/latest/download/caws-darwin-amd64) |
| macOS | Apple Silicon | [Download](https://github.com/timdev/caws/releases/latest/download/caws-darwin-arm64) |

After downloading, make it executable and move to your PATH:

```bash
chmod +x caws-*
sudo mv caws-* /usr/local/bin/caws
```

### Option 2: Build from Source

```bash
git clone https://github.com/timdev/caws.git
cd caws
go build -o caws
sudo mv caws /usr/local/bin/
# or for user install: mv caws ~/.local/bin/
```

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed build instructions.

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
- When running `exec` or `login` commands IF cached credentials have expired (approximately once per hour)

When credentials are cached (valid for ~55 minutes), you won't be prompted for a password. Your password is never stored in memory for security.

## Troubleshooting

**"Vault not found"**
→ Run `caws init` first to create your vault

**"Incorrect password or corrupted vault"**
→ Check your password. If forgotten, there's no recovery - you'll need to delete `~/.caws/vault.enc` and start over.

**"Failed to get session token"**
→ Check that your AWS credentials are valid and have the necessary permissions

**"Profile not found"**
→ The profile doesn't exist, run `caws list` to see available profiles

## What's Happening Behind the Scenes?

1. Your AWS credentials (Access Key + Secret) are stored **encrypted** in `~/.local/share/caws/vault.enc`
   - Encryption: Argon2id + AES-256-GCM (industry standard)
   - File permissions: 0600 (only you can read/write)
   - Temporary credentials cached in `~/.cache/caws/`

2. When you run a command, caws:
   - Checks for valid cached credentials first
   - If cache is expired/missing: prompts for vault password
   - Decrypts your credentials
   - Calls AWS STS to get **temporary credentials** (valid for 1 hour)
   - Caches the temporary credentials locally
   - Runs your command with the temp credentials

3. The next time you run a command within an hour, it uses the cached credentials **without prompting for your password**

**Security**: Your long-term AWS credentials are always encrypted and never written to disk in plaintext!

## Next Steps

- Read the full [README.md](../README.md) for advanced features
- Set up MFA for extra security (configure in `~/.aws/config`)
- Add more AWS profiles for different accounts/roles
- Consider backing up `~/.local/share/caws/vault.enc` to a secure location

## Getting Help

```bash
caws help
```

Or check the documentation:
- [README.md](../README.md) - Project overview
- [USAGE.md](USAGE.md) - Complete command reference
- [SECURITY.md](SECURITY.md) - Security best practices

---

**Having issues?** Open an issue on [GitHub](https://github.com/timdev/caws/issues).
