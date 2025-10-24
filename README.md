# bw-aws

A lightweight AWS credential manager that uses Bitwarden as the backend storage. Built as a modern alternative to aws-vault with the security and convenience of Bitwarden.

## Features

- 🔐 **Secure storage** - Credentials encrypted in your Bitwarden vault
- 🔄 **Automatic credential rotation** - Uses AWS STS for temporary credentials
- 💾 **Smart caching** - Caches temporary credentials to minimize STS calls
- 🔑 **MFA support** - Works with AWS MFA requirements
- 🌍 **Cross-platform sync** - Access credentials from any device with Bitwarden
- 🚀 **Zero dependencies** - Single binary, no runtime dependencies

## Prerequisites

- **Bitwarden CLI** (`bw`) - [Installation guide](https://bitwarden.com/help/cli/)
- **AWS CLI** (`aws`) - [Installation guide](https://aws.amazon.com/cli/)
- Go 1.22+ (for building from source)

### Install Bitwarden CLI

```bash
# Ubuntu/Debian
sudo snap install bw

# macOS
brew install bitwarden-cli

# Or download from https://bitwarden.com/help/cli/
```

## Installation

### From Source

```bash
git clone <your-repo-url>
cd bw-aws
go build -o bw-aws
sudo mv bw-aws /usr/local/bin/
```

### Binary Release

Download the latest binary from the releases page and move it to your PATH:

```bash
chmod +x bw-aws
sudo mv bw-aws /usr/local/bin/
```

## Quick Start

### 1. Login to Bitwarden

```bash
bw-aws login
```

This will prompt for your Bitwarden master password and provide you with a session key. Export it:

```bash
export BW_SESSION="<your-session-key>"
```

💡 **Tip**: Add this to your shell's rc file for persistence:

```bash
echo 'export BW_SESSION="<your-session-key>"' >> ~/.bashrc
```

### 2. Add AWS Credentials

```bash
bw-aws add production
```

You'll be prompted for:
- AWS Access Key ID
- AWS Secret Access Key
- Default Region (optional)
- MFA Serial ARN (optional)

### 3. Use Your Credentials

Execute any AWS command with temporary credentials:

```bash
bw-aws exec production -- aws s3 ls
bw-aws exec production -- aws ec2 describe-instances
bw-aws exec production -- terraform plan
```

## Commands

### `bw-aws login`

Authenticate with Bitwarden and get a session key.

```bash
bw-aws login
```

### `bw-aws add <profile>`

Add a new AWS profile to Bitwarden.

```bash
bw-aws add my-profile
```

The credentials are stored as a Bitwarden Secure Note with the name `bw-aws:<profile>`.

### `bw-aws list`

List all AWS profiles stored in Bitwarden.

```bash
bw-aws list
```

Example output:
```
Available AWS profiles:
  • production (region: us-east-1) [MFA enabled]
  • development (region: us-west-2)
  • staging
```

### `bw-aws exec <profile> -- <command>`

Execute a command with AWS credentials injected into the environment.

```bash
bw-aws exec production -- aws sts get-caller-identity
bw-aws exec dev -- env | grep AWS
```

The tool will:
1. Fetch credentials from Bitwarden
2. Request temporary credentials from AWS STS (1 hour duration)
3. Cache the temporary credentials
4. Execute your command with credentials in the environment

### `bw-aws remove <profile>`

Remove a profile from Bitwarden.

```bash
bw-aws remove old-profile
```

## How It Works

### Credential Flow

1. **Storage**: Long-term AWS credentials (Access Key ID + Secret Access Key) are stored in Bitwarden as Secure Notes
2. **Retrieval**: When you run a command, bw-aws fetches the credentials from Bitwarden
3. **STS Exchange**: bw-aws calls AWS STS to exchange long-term credentials for temporary credentials (1 hour validity)
4. **Caching**: Temporary credentials are cached locally in `~/.bw-aws/cache/` to avoid repeated STS calls
5. **Execution**: Your command runs with temporary credentials in the environment

### Security Model

- ✅ Long-term credentials never leave Bitwarden encryption
- ✅ Temporary credentials have a 1-hour lifetime
- ✅ MFA can be required for STS token generation
- ✅ Cache files are stored with 0600 permissions (owner read/write only)
- ✅ No credentials are written to shell history

### Bitwarden Structure

Credentials are stored as Secure Notes with this structure:

```
Name: bw-aws:production
Type: Secure Note
Fields:
  - aws_access_key_id: AKIA...
  - aws_secret_access_key: ****
  - region: us-east-1 (optional)
  - mfa_serial: arn:aws:iam::123456789012:mfa/user (optional)
```

## Advanced Usage

### MFA Support

If your AWS account requires MFA, add the MFA serial ARN when creating the profile:

```bash
bw-aws add production
# ... enter credentials ...
MFA Serial ARN: arn:aws:iam::123456789012:mfa/your-username
```

When executing commands, you'll be prompted for your MFA code:

```bash
bw-aws exec production -- aws s3 ls
Enter MFA code: 123456
```

### Credential Caching

Temporary credentials are cached for 1 hour. The cache location is:

```
~/.bw-aws/cache/<profile>.json
```

To clear the cache for a profile:

```bash
rm ~/.bw-aws/cache/production.json
```

### Multiple Profiles

You can manage multiple AWS accounts:

```bash
bw-aws add personal
bw-aws add work-dev
bw-aws add work-prod

bw-aws exec personal -- aws s3 ls
bw-aws exec work-prod -- aws ec2 describe-instances
```

### Shell Integration

For easier access, you can create shell functions:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias bwa='bw-aws'
aws-exec() {
    bw-aws exec "$@"
}
```

Then use:

```bash
bwa list
aws-exec production -- aws s3 ls
```

## Comparison with aws-vault

| Feature | bw-aws | aws-vault |
|---------|--------|-----------|
| Backend | Bitwarden | OS keyring/pass/file |
| Cross-device sync | ✅ Yes | ❌ No |
| MFA support | ✅ Yes | ✅ Yes |
| Credential caching | ✅ Yes | ✅ Yes |
| Active development | ✅ Yes | ⚠️ Limited |
| Server mode | ❌ No | ✅ Yes |
| IAM role assumption | ❌ Not yet | ✅ Yes |

## Troubleshooting

### "Not logged in to Bitwarden"

Run `bw-aws login` and export the session key:

```bash
bw-aws login
export BW_SESSION="<session-key>"
```

### "Failed to get session token"

1. Verify AWS CLI is installed: `aws --version`
2. Check that your credentials are valid
3. If using MFA, ensure the code is correct and not expired

### "Item not found"

The profile doesn't exist in Bitwarden. List profiles with:

```bash
bw-aws list
```

### Bitwarden session expired

Re-login to Bitwarden:

```bash
bw-aws login
export BW_SESSION="<new-session-key>"
```

## Building from Source

```bash
# Clone the repository
git clone <repo-url>
cd bw-aws

# Build
go build -o bw-aws

# Install
sudo mv bw-aws /usr/local/bin/

# Or for local user install
mv bw-aws ~/bin/  # Make sure ~/bin is in your PATH
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - feel free to use this in your own projects!

## Roadmap

- [ ] Support for IAM role assumption
- [ ] Support for role chaining
- [ ] Config file support
- [ ] Shell completion scripts
- [ ] Windows support improvements
- [ ] Export credentials to other formats
- [ ] Integration with other password managers

## Security Notes

⚠️ **Important Security Considerations:**

1. **Session Key**: The `BW_SESSION` environment variable provides access to your Bitwarden vault. Keep it secure and don't commit it to version control.

2. **Cache Directory**: Temporary credentials are cached in `~/.bw-aws/cache/`. These files contain valid AWS credentials for up to 1 hour. Ensure your home directory has proper permissions.

3. **MFA Recommended**: If your AWS account contains sensitive resources, enable MFA on your IAM user.

4. **Regular Rotation**: Rotate your long-term AWS credentials regularly according to your security policy.

## Author

Created as a modern alternative to aws-vault for users who prefer Bitwarden for credential management.
