# caws

A lightweight AWS credential manager with password-based encryption. Fast, local-first alternative to aws-vault with zero external dependencies.

Store your AWS credentials securely using industry-standard Argon2id + AES-256-GCM encryption. No GPG, no gopass, no external password managers - just a single self-contained binary.

## Features

- Secure password-based encryption (Argon2id + AES-256-GCM)
- Zero runtime dependencies (single static binary)
- Smart credential caching (minimize AWS STS calls)
- MFA support for sensitive accounts
- Fast operation (~50ms overhead for cached credentials)
- Local-first (no network calls for credential retrieval)
- Simple setup (one command to initialize)

## Installation

### Download Binary (Recommended)

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

### Build from Source

```bash
git clone https://github.com/timdev/caws.git
cd caws
go build -o caws
sudo mv caws /usr/local/bin/
```

Requires Go 1.24+. See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for details.

## Quick Start

```bash
# 1. Initialize encrypted vault
caws init

# 2. Add AWS credentials
caws add production

# 3. Use your credentials
caws exec production -- aws s3 ls
caws exec production -- terraform plan
```

See [docs/QUICKSTART.md](docs/QUICKSTART.md) for a detailed walkthrough.

## Usage

```bash
caws init                        # Initialize encrypted vault
caws add <profile>               # Add AWS profile
caws list                        # List profiles
caws exec <profile> -- <cmd>     # Execute command with credentials
caws login <profile>             # Generate AWS Console login URL
caws remove <profile>            # Remove profile
```

See [docs/USAGE.md](docs/USAGE.md) for full command reference and advanced features.

## How It Works

caws encrypts your long-term AWS credentials in `~/.local/share/caws/vault.enc` using a password-based key derivation (Argon2id) and AES-256-GCM encryption. When you run a command, it:

1. Prompts for your vault password
2. Decrypts your long-term credentials
3. Exchanges them for temporary AWS STS credentials (1-hour validity)
4. Caches the temporary credentials
5. Executes your command with credentials in the environment

Temporary credentials are cached for ~55 minutes to minimize password prompts and AWS API calls.

For technical details, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Comparison to aws-vault

| Feature | caws | aws-vault |
|---------|------|-----------|
| Encryption backend | Password + AES-256-GCM | OS keyring/pass/file |
| Setup | One command (`caws init`) | One command |
| External dependencies | Zero | Zero |
| MFA support | Yes | Yes |
| Credential caching | Yes (~55 min) | Yes |
| Performance | Very fast (~50ms overhead) | Very fast |
| IAM role assumption | Not yet | Yes |

Both tools are excellent. Choose caws if you prefer password-based encryption without OS keyring dependencies.

## Why caws?

- **Simple setup** - No GPG, gopass, or external password managers required
- **Self-contained** - Single binary, zero runtime dependencies
- **Fast** - Pure Go encryption with minimal overhead
- **Secure** - Industry-standard Argon2id + AES-256-GCM
- **Local-first** - No network calls for credential retrieval
- **Transparent** - Small codebase, easy to audit

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md) - Step-by-step tutorial
- [Usage Guide](docs/USAGE.md) - Complete command reference
- [Architecture](docs/ARCHITECTURE.md) - Technical deep dive
- [Security](docs/SECURITY.md) - Security model and best practices
- [Contributing](docs/CONTRIBUTING.md) - Development guide

## Project Status

This project is in active development and is functional for daily use. It's also an experiment in LLM-assisted development - much of the code was written with AI assistance as the author learns Go.

The tool handles real AWS credentials and uses production-grade encryption. While young, the core security model is sound and the codebase is small enough to audit personally.

Use at your own discretion. Feedback and contributions welcome!

## License

MIT License - see LICENSE file for details.
