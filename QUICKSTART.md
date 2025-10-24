# bw-aws Quick Start Guide

Get up and running with bw-aws in 5 minutes!

## Prerequisites Check

```bash
# Check if Bitwarden CLI is installed
bw --version

# Check if AWS CLI is installed
aws --version
```

If either is missing:
- **Bitwarden CLI**: https://bitwarden.com/help/cli/
  - Ubuntu: `sudo snap install bw`
  - macOS: `brew install bitwarden-cli`
- **AWS CLI**: https://aws.amazon.com/cli/

## Installation

### Option 1: Using the install script (recommended)

```bash
cd bw-aws
./install.sh
```

### Option 2: Manual installation

```bash
cd bw-aws
go build -o bw-aws
sudo mv bw-aws /usr/local/bin/
# or for user install: mv bw-aws ~/.local/bin/
```

### Option 3: Use the pre-built binary

The `bw-aws` binary is already compiled! Just move it:

```bash
cd bw-aws
sudo mv bw-aws /usr/local/bin/
# or: mv bw-aws ~/.local/bin/
```

## Setup (5 steps)

### 1. Login to Bitwarden

```bash
bw-aws login
```

You'll be prompted for your Bitwarden master password.

### 2. Export the session key

Copy the session key from the output and export it:

```bash
export BW_SESSION="your-session-key-here"
```

**Important**: Add this to your `~/.bashrc` or `~/.zshrc` to make it persistent:

```bash
echo 'export BW_SESSION="your-session-key-here"' >> ~/.bashrc
source ~/.bashrc
```

### 3. Add your first AWS profile

```bash
bw-aws add myprofile
```

Enter when prompted:
- AWS Access Key ID (from AWS IAM)
- AWS Secret Access Key (input will be hidden)
- Default Region (e.g., `us-east-1`) - optional
- MFA Serial ARN - optional

### 4. Verify it worked

```bash
bw-aws list
```

You should see your profile listed.

### 5. Use it!

```bash
bw-aws exec myprofile -- aws sts get-caller-identity
```

If this shows your AWS account info, you're all set! 🎉

## Common Commands

```bash
# List S3 buckets
bw-aws exec myprofile -- aws s3 ls

# Check who you are
bw-aws exec myprofile -- aws sts get-caller-identity

# Run any AWS command
bw-aws exec myprofile -- aws ec2 describe-instances

# Use with Terraform
bw-aws exec myprofile -- terraform plan

# See environment variables
bw-aws exec myprofile -- env | grep AWS
```

## Tips

### 1. Create shell aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias bwa='bw-aws'
alias awsexec='bw-aws exec'
```

Then use:

```bash
awsexec myprofile -- aws s3 ls
```

### 2. Add multiple profiles

```bash
bw-aws add personal
bw-aws add work-dev
bw-aws add work-prod
```

### 3. If your session expires

Just login again:

```bash
bw-aws login
export BW_SESSION="new-key"
```

## Troubleshooting

**"Not logged in to Bitwarden"**
→ Run `bw-aws login` and export the `BW_SESSION`

**"Failed to get session token"**
→ Check that AWS CLI is installed and your credentials are valid

**"Item not found"**
→ The profile doesn't exist, run `bw-aws list` to see available profiles

## What's Happening Behind the Scenes?

1. Your AWS credentials (Access Key + Secret) are stored **encrypted** in Bitwarden
2. When you run a command, bw-aws:
   - Fetches credentials from Bitwarden
   - Calls AWS STS to get **temporary credentials** (valid for 1 hour)
   - Caches the temporary credentials locally
   - Runs your command with the temp credentials

3. The next time you run a command within an hour, it uses the cached credentials (no need to call STS again)

**Security**: Your long-term AWS credentials stay encrypted in Bitwarden and are never written to disk unencrypted!

## Next Steps

- Read the full [README.md](README.md) for advanced features
- Set up MFA for extra security
- Add more AWS profiles for different accounts/roles

## Getting Help

```bash
bw-aws help
```

Or check the [README.md](README.md) for detailed documentation.

---

**Having issues?** Open an issue on GitHub or check the Troubleshooting section in the README.
