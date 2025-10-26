# Contributing to caws

Thank you for your interest in contributing! This guide will help you get set up for development.

## Prerequisites

- **Go 1.24.1+** ([download](https://go.dev/dl/))
- **Git**
- **AWS CLI** (optional, for testing)

That's it! No external dependencies required.

## Building from Source

### Clone the repository

```bash
git clone https://github.com/timdev/caws.git
cd caws
```

### Download dependencies

```bash
go mod download
go mod verify
```

### Build the binary

```bash
go build -o caws
```

This creates a `caws` binary in the current directory.

### Verify the build

```bash
./caws --help
```

## Installation

### System-wide installation (requires sudo)

```bash
sudo mv caws /usr/local/bin/
```

### User installation (no sudo required)

```bash
# Ensure ~/.local/bin exists and is in PATH
mkdir -p ~/.local/bin
mv caws ~/.local/bin/

# Add to PATH if not already (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"
```

## Development Workflow

### Make changes

Edit the relevant files:
- `main.go` - Entry point and command routing
- `commands.go` - Command handlers
- `vault.go` - Vault management
- `crypto.go` - Encryption primitives
- `aws.go` - AWS STS integration
- `config.go` - AWS config file reading
- `validation.go` - Input validation
- `xdg.go` - XDG directory utilities

### Build and test locally

```bash
# Build
go build -o caws

# Test your changes
./caws init
./caws add testprofile
./caws list
./caws exec testprofile -- echo "test"
./caws remove testprofile
```

### Clean up test data

```bash
rm -rf ~/.caws/
```

## Testing

### Manual Testing

See [TEST.md](TEST.md) for detailed manual testing instructions.

**Quick test:**
```bash
# Clean environment
rm -rf ~/.caws/

# Initialize and test
./caws init
./caws add test
./caws list
./caws exec test -- echo "success"
./caws remove test

# Cleanup
rm -rf ~/.caws/
```

### Automated Tests

caws has both unit tests and end-to-end tests.

**Run all tests:**
```bash
go test ./... -v
```

**Run specific test:**
```bash
go test -v -run TestExecCommand
```

**With coverage:**
```bash
go test ./... -cover
```

### Docker Sandbox

For isolated development with Claude Code, see [DOCKER_SANDBOX.md](DOCKER_SANDBOX.md).

## Code Style

### Formatting

Use `gofmt` (run automatically in CI):

```bash
# Format all files
go fmt ./...

# Check before committing
gofmt -l .
```

### Linting

Recommended (not enforced currently):

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Code Organization Principles

- **`crypto.go`** - Pure crypto primitives (no I/O, no user interaction)
- **`vault.go`** - Vault management (file I/O, user prompts)
- **`commands.go`** - Command handlers (orchestration)
- **`main.go`** - Entry point (routing only)
- **`aws.go`** - AWS-specific logic

Keep crypto operations separate from business logic for easier auditing and testing.

## Commit Messages

Use clear, descriptive commit messages:

**Good:**
```
Add MFA support for STS token generation
Fix vault corruption on concurrent writes
Update README with installation instructions
```

**Avoid:**
```
fix bug
update code
wip
```

**Format:**
```
Short summary (50 chars or less)

Longer explanation if needed. Wrap at 72 characters.
Explain what and why, not how.

- Bullet points are okay
- Reference issues: Fixes #123
```

## Pull Request Process

### Before submitting

1. **Test your changes:**
   ```bash
   go test ./...
   go fmt ./...
   go build -o caws
   ./caws --help
   ```

2. **Update documentation** if needed:
   - README.md - for user-facing changes
   - ARCHITECTURE.md - for internal changes
   - USAGE.md - for new features/commands

3. **Clean commit history:**
   ```bash
   # Squash WIP commits if needed
   git rebase -i HEAD~3
   ```

### Submitting

1. Fork the repository
2. Create a feature branch:
   ```bash
   git checkout -b feature/my-new-feature
   ```

3. Commit your changes:
   ```bash
   git add .
   git commit -m "Add my new feature"
   ```

4. Push to your fork:
   ```bash
   git push origin feature/my-new-feature
   ```

5. Open a Pull Request on GitHub

### PR description

Include:
- **What**: Summary of changes
- **Why**: Motivation and context
- **How**: Brief explanation of approach
- **Testing**: How you tested the changes
- **Checklist**:
  - [ ] Tests pass
  - [ ] Code formatted (`go fmt`)
  - [ ] Documentation updated
  - [ ] Manual testing completed

## Development Environment

### Recommended setup

**Editor:** Any editor with Go support
- VS Code + Go extension
- GoLand
- vim + vim-go
- Emacs + go-mode

**Required tools:**
```bash
# Go
go version  # Should be 1.24+

# Git
git --version
```

**Optional tools:**
```bash
# AWS CLI (for testing)
aws --version

# jq (for inspecting JSON files)
jq --version

# golangci-lint (for linting)
golangci-lint --version
```

### Environment variables for testing

For automated testing (not secure for production):

```bash
export CAWS_TEST_AWS_ACCESS_KEY=AKIAWF6WMLRP7EX56ZYK
export CAWS_TEST_AWS_SECRET_KEY=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
export CAWS_TEST_DIR=/tmp/caws-test
export CAWS_PASSWORD=testpass
export CAWS_AUTO_CONFIRM=yes
```

## Architecture

For understanding how caws works internally, see [ARCHITECTURE.md](ARCHITECTURE.md).

**Key components:**

1. **VaultClient** (`vault.go`)
   - Manages encrypted credential vault
   - Prompts for password
   - CRUD operations on profiles

2. **Encryption** (`crypto.go`)
   - Argon2id key derivation
   - AES-256-GCM encryption/decryption
   - Versioned vault format

3. **AWS Integration** (`aws.go`)
   - STS API calls
   - Credential caching
   - Environment variable injection

## Adding New Features

### Example: Adding a new command

1. **Add command handler** in `commands.go`:
   ```go
   func handleMyCommand() {
       // Implementation
   }
   ```

2. **Add routing** in `main.go`:
   ```go
   case "mycommand":
       handleMyCommand()
   ```

3. **Update usage** in `main.go`:
   ```go
   func printUsage() {
       fmt.Println("  mycommand - Do something useful")
   }
   ```

4. **Update documentation**:
   - README.md - Usage section
   - USAGE.md - Detailed command reference
   - QUICKSTART.md - If relevant for beginners

5. **Test**:
   ```bash
   go build -o caws
   ./caws mycommand
   ```

6. **Add tests** (if applicable):
   ```go
   func TestMyCommand(t *testing.T) {
       // Test implementation
   }
   ```

### Example: Adding vault encryption field

1. **Update vault structure** in `vault.go`:
   ```go
   type Profile struct {
       AccessKey   string `json:"access_key"`
       SecretKey   string `json:"secret_key"`
       Region      string `json:"region,omitempty"`
       MFASerial   string `json:"mfa_serial,omitempty"`
       NewField    string `json:"new_field,omitempty"`  // Add this
   }
   ```

2. **Update command handlers** to prompt/use new field

3. **Consider vault version migration** if changing structure

4. **Update documentation**:
   - ARCHITECTURE.md - Vault structure
   - USAGE.md - If user-facing

## Release Process

Releases are automated via GitHub Actions when you push a tag.

### Creating a release

1. **Update version** (if you track it):
   ```bash
   # Update any version constants
   ```

2. **Tag the release:**
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. **GitHub Actions automatically:**
   - Builds binaries for Linux (amd64/arm64), macOS (amd64/arm64)
   - Generates SHA256 checksums
   - Creates GitHub Release
   - Uploads binaries

4. **Release appears at:**
   `https://github.com/timdev/caws/releases/tag/v0.2.0`

## Project Philosophy

This project is an experiment in LLM-assisted development. Some notes:

- **AI-assisted**: Much of this code was written with AI assistance (Claude)
- **Learning by doing**: The author is learning Go through this project
- **Practical focus**: Solve real problems first, refine later
- **Security-conscious**: Crypto and security are taken seriously
- **Open to experimentation**: PRs welcome, even unconventional approaches

## Questions?

- Check [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues
- Open an issue on GitHub for questions
- Email maintainer for security issues (see [SECURITY.md](SECURITY.md))

## Code of Conduct

Be respectful, constructive, and collaborative. This is a learning project and a tool to help developers - let's keep it friendly and productive.

---

**Thank you for contributing to caws!** 🎉
