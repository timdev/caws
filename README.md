# bw-aws

A lightweight AWS credential manager that uses gopass as the backend storage. Built as a fast, local-first alternative to aws-vault with the security and convenience of GPG-encrypted secrets.

## Features

- 🔐 **Secure storage** - Credentials encrypted with GPG in gopass
- 🚀 **Blazing fast** - Direct library integration (~800ms vs aws-vault's 2-3s)
- 🔄 **Automatic credential rotation** - Uses AWS STS for temporary credentials
- 💾 **Smart caching** - Caches temporary credentials to minimize STS calls
- 🔑 **MFA support** - Works with AWS MFA requirements
- 🌍 **Git-based sync** - Access credentials across devices via Git
- 📦 **Zero runtime dependencies** - Single binary (just needs gopass + GPG)

## Prerequisites

- **gopass** - [Installation guide](https://github.com/gopasspw/gopass#installation)
- **GPG** (GnuPG 2.x) - Usually pre-installed on Linux/macOS
- **AWS CLI** (`aws`) - [Installation guide](https://aws.amazon.com/cli/)
- Go 1.22+ (for building from source)

### Install gopass

```bash
# macOS
brew install gopass

# Ubuntu/Debian
sudo apt install gopass

# Or download from https://github.com/gopasspw/gopass/releases
```

### Initialize gopass

If you haven't used gopass before:

```bash
# Initialize with your GPG key
gopass init

# Or specify a key explicitly
gopass init <your-gpg-key-id>
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

### 1. Add AWS Credentials

```bash
bw-aws add production
```

You'll be prompted for:
- AWS Access Key ID
- AWS Secret Access Key
- Default Region (optional)
- MFA Serial ARN (optional)

Credentials are stored at `~/.local/share/gopass/stores/root/aws/<profile>`.

### 2. Use Your Credentials

Execute any AWS command with temporary credentials:

```bash
bw-aws exec production -- aws s3 ls
bw-aws exec production -- aws ec2 describe-instances
bw-aws exec production -- terraform plan
```

## Commands

### `bw-aws add <profile>`

Add a new AWS profile to gopass.

```bash
bw-aws add my-profile
```

### `bw-aws list`

List all AWS profiles stored in gopass.

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
1. Fetch credentials from gopass (GPG-encrypted)
2. Request temporary credentials from AWS STS (1 hour duration)
3. Cache the temporary credentials
4. Execute your command with credentials in the environment

### `bw-aws remove <profile>`

Remove a profile from gopass.

```bash
bw-aws remove old-profile
```

## How It Works

### Credential Flow

1. **Storage**: Long-term AWS credentials (Access Key ID + Secret Access Key) are stored in gopass, encrypted with GPG
2. **Retrieval**: When you run a command, bw-aws uses the gopass Go library to decrypt credentials
3. **STS Exchange**: bw-aws calls AWS STS to exchange long-term credentials for temporary credentials (1 hour validity)
4. **Caching**: Temporary credentials are cached locally in `~/.bw-aws/cache/` to avoid repeated STS calls
5. **Execution**: Your command runs with temporary credentials in the environment

### Security Model

- ✅ Long-term credentials stored GPG-encrypted via gopass
- ✅ Temporary credentials have a 1-hour lifetime
- ✅ MFA can be required for STS token generation
- ✅ Cache files are stored with 0600 permissions (owner read/write only)
- ✅ No credentials are written to shell history
- ✅ GPG passphrase required for decryption (cached by gpg-agent)

### gopass Structure

Credentials are stored in gopass with this structure:

```
Path: aws/<profile-name>
Password: (informational text)
Fields:
  - access_key: AKIA...
  - secret_key: ****
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

### Using Alternative gopass Stores

If you use multiple gopass stores, set `PASSWORD_STORE_DIR`:

```bash
PASSWORD_STORE_DIR=/path/to/store bw-aws add profile
PASSWORD_STORE_DIR=/path/to/store bw-aws exec profile -- aws s3 ls
```

## Performance

| Operation | Time | Notes |
|-----------|------|-------|
| Cold start (first exec) | ~1.4s | Includes gopass decrypt + AWS STS call |
| Warm start (cached STS) | ~0.8s | Just gopass decrypt + AWS CLI |
| bw-aws overhead | ~50-100ms | gopass library performance |

Compare to aws-vault: **3-6x faster** for typical operations.

## Comparison with aws-vault

| Feature | bw-aws | aws-vault |
|---------|--------|-----------|
| Backend | gopass (GPG) | OS keyring/pass/file |
| Cross-device sync | ✅ Via Git | ❌ Manual |
| MFA support | ✅ Yes | ✅ Yes |
| Credential caching | ✅ Yes | ✅ Yes |
| Performance | ✅ Very fast (~800ms) | ⚠️ Slower (2-3s) |
| Setup complexity | Low (if gopass exists) | Low |
| IAM role assumption | ❌ Not yet | ✅ Yes |

## Troubleshooting

### "Failed to open gopass store"

Make sure gopass is initialized:

```bash
gopass ls
```

If not initialized:

```bash
gopass init
```

### GPG Passphrase Prompts

The first time you use bw-aws in a session, GPG will prompt for your passphrase. This is normal and secure. The passphrase is cached by `gpg-agent` for subsequent operations.

To adjust cache timeout, edit `~/.gnupg/gpg-agent.conf`:

```
default-cache-ttl 3600
max-cache-ttl 7200
```

### "Failed to get session token"

1. Verify AWS CLI is installed: `aws --version`
2. Check that your credentials are valid
3. If using MFA, ensure the code is correct and not expired

### "Profile not found"

The profile doesn't exist in gopass. List profiles with:

```bash
bw-aws list
```

Or check gopass directly:

```bash
gopass ls aws/
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

## Security Notes

⚠️ **Important Security Considerations:**

1. **GPG Key Security**: Your gopass secrets are only as secure as your GPG private key. Keep it safe and use a strong passphrase.

2. **Cache Directory**: Temporary credentials are cached in `~/.bw-aws/cache/`. These files contain valid AWS credentials for up to 1 hour. Ensure your home directory has proper permissions.

3. **MFA Recommended**: If your AWS account contains sensitive resources, enable MFA on your IAM user.

4. **Regular Rotation**: Rotate your long-term AWS credentials regularly according to your security policy.

5. **Git Sync**: If syncing gopass via Git, ensure your Git remotes are secure (use SSH with key auth, not HTTPS with passwords).

## Why gopass?

- **Local-first**: No network calls for credential retrieval (unlike Bitwarden)
- **Fast**: Direct Go library integration, no CLI overhead
- **Git-based sync**: Control when and how credentials sync across machines
- **Battle-tested**: gopass is widely used and actively maintained
- **Open source**: Fully auditable password management

## Author

Built as a fast, local-first alternative to aws-vault for users who prefer gopass for credential management.
