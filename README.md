# caws

A lightweight, self-contained AWS credential manager with password-based encryption. Built as a fast, local-first alternative to aws-vault with zero external dependencies.

## Features

- 🔐 **Secure storage** - Credentials encrypted with Argon2id + AES-256-GCM
- 🚀 **Blazing fast** - Direct encryption, no external processes (~50ms overhead)
- 🔄 **Automatic credential rotation** - Uses AWS STS for temporary credentials
- 💾 **Smart caching** - Caches temporary credentials to minimize STS calls
- 🔑 **MFA support** - Works with AWS MFA requirements
- 📦 **Zero dependencies** - Single self-contained binary
- 🔒 **Simple security** - Password-protected vault, no GPG setup required

## Status: Early Development!

This is in early development. 50% of the purpose of this is to use more LLM juice 
than I typically do. Please excuse the mess.

Still to do:

* Use XDG standard directories for config, vault, etc.
* Support multiple named vaults (with like --vault-id=favoriteClient)
* Improve tests with some docker-based isolation?
* Set up GitHub Actions 
  * Build/Package
  * Maybe more to support LLM-oriented development
* Go whole-hog with LLM-driven pull requests and issue-tracker interaction?

## Prerequisites

- Go 1.22+ (for building from source)

That's it! No AWS CLI, no gopass, no GPG, no external dependencies.

## Installation

### From Source

```bash
git clone https://github.com/timdev/caws
cd caws
go build -o caws
sudo mv caws /usr/local/bin/
```

### Binary Release

Download the latest binary from the releases page and move it to your PATH:

```bash
chmod +x caws
sudo mv caws /usr/local/bin/
```

## Quick Start

### 1. Initialize Vault

Create a new encrypted vault with a master password:

```bash
caws init
```

You'll be prompted to set a master password. This password encrypts all your AWS credentials.

### 2. Add AWS Credentials

```bash
caws add production
```

Enter your vault password, then provide:
- AWS Access Key ID
- AWS Secret Access Key
- Default Region (optional)
- MFA Serial ARN (optional)

Credentials are stored encrypted at `~/.caws/vault.enc`.

### 3. Use Your Credentials

Execute any AWS command with temporary credentials:

```bash
caws exec production -- aws s3 ls
caws exec production -- aws ec2 describe-instances
caws exec production -- terraform plan
```

## Commands

### `caws init`

Initialize a new encrypted vault.

```bash
caws init
```

Creates `~/.caws/vault.enc` protected by your master password.

### `caws add <profile>`

Add a new AWS profile to the vault.

```bash
caws add my-profile
```

### `caws list`

List all AWS profiles stored in the vault.

```bash
caws list
```

Example output:
```
Available AWS profiles:
  • production (region: us-east-1) [MFA enabled]
  • development (region: us-west-2)
  • staging
```

### `caws exec <profile> -- <command>`

Execute a command with AWS credentials injected into the environment.

```bash
caws exec production -- aws sts get-caller-identity
caws exec dev -- env | grep AWS
```

The tool will:
1. Prompt for vault password (first time, then every ~55 minutes)
2. Decrypt credentials from vault
3. Request temporary credentials from AWS STS (1 hour duration)
4. Cache the temporary credentials
5. Execute your command with credentials in the environment

### `caws remove <profile>`

Remove a profile from the vault.

```bash
caws remove old-profile
```

## How It Works

### Credential Flow

1. **Storage**: Long-term AWS credentials are encrypted with AES-256-GCM and stored in `~/.caws/vault.enc`
2. **Password Derivation**: Master password is converted to encryption key using Argon2id (memory-hard, GPU-resistant)
3. **Retrieval**: When you run a command, caws prompts for password and decrypts credentials
4. **STS Exchange**: caws calls AWS STS to exchange long-term credentials for temporary credentials (1 hour validity)
5. **Caching**: Temporary credentials are cached in `~/.caws/cache/` to avoid repeated STS calls
6. **Execution**: Your command runs with temporary credentials in the environment

### Security Model

- ✅ Long-term credentials encrypted with Argon2id + AES-256-GCM
- ✅ Vault file permissions: 0600 (owner read/write only)
- ✅ Password required approximately once per hour per active profile
- ✅ Temporary credentials have a 1-hour lifetime
- ✅ MFA can be required for STS token generation
- ✅ Cache files stored with 0600 permissions
- ✅ No credentials written to shell history
- ✅ No plaintext long-term credentials ever touch disk

### Vault Structure

The vault is a single encrypted JSON file at `~/.caws/vault.enc`:

```json
{
  "version": 1,
  "salt": "base64-encoded-random-salt",
  "nonce": "base64-encoded-random-nonce",
  "data": "base64-encoded-encrypted-credentials"
}
```

When decrypted, it contains:

```json
{
  "profiles": {
    "production": {
      "access_key": "AKIA...",
      "secret_key": "****",
      "region": "us-east-1",
      "mfa_serial": "arn:aws:iam::123456789012:mfa/user"
    }
  }
}
```

## Advanced Usage

### MFA Support

If your AWS account requires MFA, add the MFA serial ARN when creating the profile:

```bash
caws add production
# ... enter vault password and credentials ...
MFA Serial ARN: arn:aws:iam::123456789012:mfa/your-username
```

When executing commands, you'll be prompted for your MFA code:

```bash
caws exec production -- aws s3 ls
Enter vault password: ****
Getting temporary credentials...
Enter MFA code: 123456
```

### Password Entry Frequency

With STS credential caching, you typically enter your vault password:
- **Once per hour per active profile** during normal usage
- Credentials are decrypted on-demand
- STS temporary credentials cached for ~55 minutes
- No plaintext long-term credentials persist on disk

### Credential Caching

Temporary credentials are cached for 1 hour. The cache location is:

```
~/.caws/cache/<profile>.json
```

To clear the cache for a profile:

```bash
rm ~/.caws/cache/production.json
```

### Multiple Profiles

You can manage multiple AWS accounts:

```bash
caws add personal
caws add work-dev
caws add work-prod

caws exec personal -- aws s3 ls
caws exec work-prod -- aws ec2 describe-instances
```

### Shell Integration

For easier access, you can create shell aliases:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias ca='caws'
aws-exec() {
    caws exec "$@"
}
```

Then use:

```bash
ca list
aws-exec production -- aws s3 ls
```

## Performance

| Operation | Time | Notes |
|-----------|------|-------|
| Cold start (first exec) | ~1.4s | Includes vault decrypt + AWS STS call |
| Warm start (cached STS) | ~0.1s | No vault access needed |
| caws overhead | ~50ms | Pure Go encryption/decryption |

Compare to aws-vault: **Similar or faster** for typical operations.

## Comparison with aws-vault

| Feature | caws | aws-vault |
|---------|------|-----------|
| Backend | Password + AES-256-GCM | OS keyring/pass/file |
| Dependencies | Zero (self-contained) | None (self-contained) |
| Setup | One command (`caws init`) | One command |
| MFA support | ✅ Yes | ✅ Yes |
| Credential caching | ✅ Yes | ✅ Yes |
| Performance | ✅ Very fast (~50ms) | ✅ Very fast |
| IAM role assumption | ❌ Not yet | ✅ Yes |

## Troubleshooting

### "Vault not found"

Initialize the vault first:

```bash
caws init
```

### "Incorrect password or corrupted vault"

Your password is wrong, or the vault file is corrupted. If you've forgotten your password, there's no recovery - you'll need to delete `~/.caws/vault.enc` and start over.

### "Failed to get session token"

1. Check that your credentials are valid
2. If using MFA, ensure the code is correct and not expired
3. Verify your AWS credentials have the necessary permissions

### "Profile not found"

The profile doesn't exist in your vault. List profiles with:

```bash
caws list
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/timdev/caws
cd caws

# Build
go build -o caws

# Install
sudo mv caws /usr/local/bin/

# Or for local user install
mv caws ~/.local/bin/  # Make sure ~/.local/bin is in your PATH
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - feel free to use this in your own projects!

## Roadmap

- [ ] Support for IAM role assumption
- [ ] Support for role chaining
- [ ] Config file support (e.g., default profile)
- [ ] Shell completion scripts
- [ ] Windows support improvements
- [ ] Export credentials to other formats
- [ ] Vault password change command

## Security Notes

⚠️ **Important Security Considerations:**

1. **Master Password**: Your credentials are only as secure as your master password. Use a strong, unique password.

2. **Cache Directory**: Temporary credentials are cached in `~/.caws/cache/`. These files contain valid AWS credentials for up to 1 hour. The tool sets proper permissions (0600) automatically.

3. **MFA Recommended**: If your AWS account contains sensitive resources, enable MFA on your IAM user.

4. **Regular Rotation**: Rotate your long-term AWS credentials regularly according to your security policy.

5. **No Password Recovery**: If you forget your master password, there's no way to recover it. You'll need to reinitialize and re-add all credentials.

6. **Vault Backups**: Consider backing up `~/.caws/vault.enc` to a secure location. Without your password, the backup is useless to an attacker.

## Why caws?

- **Simple setup**: No GPG, no gopass, no external password managers
- **Self-contained**: Single binary with zero dependencies
- **Fast**: Pure Go encryption with minimal overhead
- **Secure**: Industry-standard Argon2id + AES-256-GCM encryption
- **Local-first**: No network calls for credential retrieval
- **Transparent**: Small codebase, easy to audit

## Author

Built as a fast, simple, local-first alternative to aws-vault for developers who want credential management without external dependencies.
