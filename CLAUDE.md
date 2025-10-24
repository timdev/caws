# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`bw-aws` is a CLI tool that manages AWS credentials using Bitwarden as the storage backend. It's an alternative to aws-vault that leverages Bitwarden's cross-device sync capabilities.

**Core workflow:**
1. Long-term AWS credentials stored encrypted in Bitwarden as Secure Notes (name: `bw-aws:<profile>`)
2. Tool fetches credentials from Bitwarden using the `bw` CLI
3. Exchanges long-term credentials for temporary STS credentials (1-hour duration)
4. Caches temporary credentials in `~/.bw-aws/cache/<profile>.json`
5. Executes commands with credentials injected as environment variables

## Build & Development Commands

```bash
# Build the binary
go build -o bw-aws

# Run without installing
./bw-aws <command>

# Install to system
sudo mv bw-aws /usr/local/bin/

# Install for local user
mv bw-aws ~/.local/bin/  # Ensure ~/.local/bin is in PATH
```

**Note:** This is a pure Go CLI application with no external dependencies beyond the standard library. No tests currently exist in the repository.

## Architecture

### File Structure

- `main.go` - Entry point, command routing and usage text
- `commands.go` - Command handlers (add, list, exec, login, remove)
- `bitwarden.go` - Bitwarden CLI wrapper (CRUD operations on vault items)
- `aws.go` - AWS STS integration, credential caching, environment variable management

### Key Components

**BitwardenClient** (`bitwarden.go`)
- Wraps the `bw` CLI tool via `os/exec`
- Requires `BW_SESSION` environment variable for authentication
- Stores AWS credentials as Secure Notes (type 2) with custom fields
- Item naming convention: `bw-aws:<profile-name>`

**Credential Flow** (`commands.go:handleExec()`)
1. Check if user logged into Bitwarden (`BW_SESSION` env var)
2. Retrieve profile from Bitwarden (item name: `bw-aws:<profile>`)
3. Check cache for valid temporary credentials (`GetCachedCredentials()`)
4. If cache miss/expired: call AWS STS via `aws` CLI (`AssumeRole()`)
5. Prompt for MFA code if `mfa_serial` field present
6. Cache temporary credentials with 5-minute expiration buffer
7. Execute command with credentials in environment

**Credential Storage Schema**
```
Bitwarden Secure Note fields:
- aws_access_key_id (required)
- aws_secret_access_key (required)
- region (optional)
- mfa_serial (optional, format: arn:aws:iam::123456789012:mfa/user)
```

**Caching** (`aws.go`)
- Cache location: `~/.bw-aws/cache/<profile>.json`
- Cache directory permissions: 0700 (owner only)
- Cache file permissions: 0600 (owner read/write only)
- Expiration buffer: 5 minutes before actual expiration

### External Dependencies

**Runtime requirements:**
- `bw` (Bitwarden CLI) - Must be in PATH
- `aws` (AWS CLI) - Used for STS operations

**Key commands executed:**
- `bw status` - Check login status
- `bw unlock --raw` - Get session key
- `bw get item <name> --session <key>` - Retrieve credentials
- `bw create item <encoded> --session <key>` - Store credentials
- `bw sync --session <key>` - Sync vault with server
- `aws sts get-session-token` - Get temporary credentials

## Common Development Patterns

### Adding New Commands

1. Add case to switch in `main.go:main()`
2. Implement handler function in `commands.go` (prefix: `handle*`)
3. Update `printUsage()` in `main.go`

### Bitwarden Operations

Always check `IsLoggedIn()` before performing vault operations. The session check uses `bw sync` as a test command since it requires authentication.

### Error Handling

The codebase uses a consistent pattern:
- Print error to stdout with `fmt.Printf()`
- Call `os.Exit(1)` for fatal errors
- Return errors from functions for non-command contexts

### Input Handling

For password/secret input (see `commands.go:handleAdd()`):
```go
exec.Command("stty", "-echo").Run()  // Disable echo
// Read input
exec.Command("stty", "echo").Run()   // Re-enable echo
```

**Note:** This is Unix-specific and won't work on Windows.

## Security Considerations

- Long-term credentials never written to disk unencrypted
- Temporary credentials cached with restrictive permissions (0600)
- `BW_SESSION` provides access to entire vault - treat as sensitive
- MFA codes read from stdin, not passed as command arguments
- Cache expiration uses 5-minute safety buffer to avoid using near-expired credentials

## Important Behaviors

- Commands that require profile names will prefix them with `bw-aws:` when interacting with Bitwarden
- The tool does NOT create or manage AWS IAM users/keys - users must obtain keys from AWS Console/CLI
- Temporary credentials are always 1 hour duration (3600 seconds)
- Failed commands preserve their exit codes via `exec.ExitError`
- Cache files are silently ignored if corrupt/expired
