# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`bw-aws` is a CLI tool that manages AWS credentials using gopass as the storage backend. It's a fast, local-first alternative to aws-vault that leverages gopass's GPG-encrypted password store.

**Core workflow:**
1. Long-term AWS credentials stored encrypted in gopass (at `~/.local/share/gopass/stores/root/aws/<profile>`)
2. Tool fetches credentials using gopass Go library (direct API calls, no CLI)
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

**Testing:** See `TEST.md` for manual testing instructions with isolated gopass store.

**No unit tests** currently exist in the repository.

## Architecture

### File Structure

- `main.go` - Entry point, command routing and usage text
- `commands.go` - Command handlers (add, list, exec, remove)
- `gopass.go` - gopass library integration (CRUD operations on secrets)
- `aws.go` - AWS STS integration, credential caching, environment variable management
- `bitwarden.go` - **DEPRECATED** - Old Bitwarden backend (kept for reference, not used)

### Key Components

**GopassClient** (`gopass.go`)
- Uses `github.com/gopasspw/gopass/pkg/gopass/api` directly (no shell exec)
- Credentials stored with path prefix `aws/`
- Secret format: AKV (Key-Value) with fields: `access_key`, `secret_key`, `region`, `mfa_serial`
- GPG handled automatically by gopass library
- Must call `Close()` to ensure pending git operations complete

**Credential Flow** (`commands.go:handleExec()`)
1. Create GopassClient (opens gopass store)
2. Retrieve profile credentials via `GetCredentials(profile)`
3. Check cache for valid temporary credentials (`GetCachedCredentials()`)
4. If cache miss/expired: call AWS STS via `aws` CLI (`AssumeRole()`)
5. Prompt for MFA code if `mfa_serial` field present
6. Cache temporary credentials with 5-minute expiration buffer
7. Execute command with credentials in environment
8. Close GopassClient (flushes pending operations)

**Credential Storage Schema**
```
gopass secret path: aws/<profile-name>
Format: AKV (Key-Value)
Password line: "AWS credentials for <profile>"
Fields:
  - access_key (required): AWS_ACCESS_KEY_ID
  - secret_key (required): AWS_SECRET_ACCESS_KEY
  - region (optional): Default AWS region
  - mfa_serial (optional): ARN for MFA device
```

**Caching** (`aws.go`)
- Cache location: `~/.bw-aws/cache/<profile>.json`
- Cache directory permissions: 0700 (owner only)
- Cache file permissions: 0600 (owner read/write only)
- Expiration buffer: 5 minutes before actual expiration
- Cache format: JSON with STS credentials + expiration timestamp

### External Dependencies

**Runtime requirements:**
- `gopass` - Must be initialized (`gopass init`)
- `gpg` (GnuPG 2.x) - For gopass encryption/decryption
- `aws` (AWS CLI) - Used for STS operations

**Go dependencies:**
- `github.com/gopasspw/gopass/pkg/gopass/api` - gopass store operations
- `github.com/gopasspw/gopass/pkg/gopass/secrets` - Secret creation/manipulation

**Key commands executed:**
- `aws sts get-session-token` - Get temporary credentials
- No gopass CLI calls (uses library directly)
- No GPG CLI calls (handled by gopass library)

## Common Development Patterns

### Adding New Commands

1. Add case to switch in `main.go:main()`
2. Implement handler function in `commands.go` (prefix: `handle*`)
3. Update `printUsage()` in `main.go`
4. Remember to call `defer gp.Close()` on GopassClient instances

### gopass Operations

Always create client with `NewGopassClient()` and defer `Close()`:

```go
gp, err := NewGopassClient()
if err != nil {
    // Handle error
}
defer gp.Close()  // CRITICAL - ensures git operations complete
```

The gopass library handles GPG automatically:
- User will see pinentry prompts for GPG passphrase
- gpg-agent caches passphrase for subsequent operations
- No manual GPG configuration needed

### Error Handling

The codebase uses a consistent pattern:
- Print error to stdout with `fmt.Printf()`
- Call `os.Exit(1)` for fatal errors
- Return errors from library functions

### Input Handling

For password/secret input (see `commands.go:handleAdd()`):
```go
exec.Command("stty", "-echo").Run()  // Disable echo
// Read input
exec.Command("stty", "echo").Run()   // Re-enable echo
```

**Note:** This is Unix-specific and won't work on Windows.

## Performance Characteristics

**gopass library approach** (current):
- GetCredentials: ~10-50ms (direct GPG decrypt)
- Cold start (with STS): ~1.4s (mostly AWS STS call)
- Warm start (cached STS): ~0.8s (mostly AWS CLI overhead)
- bw-aws overhead: ~50-100ms

**Why so fast:**
- No CLI overhead (direct library calls)
- No network calls for credential retrieval (local GPG decrypt)
- Minimal parsing (native Go structs)

Compare to alternatives:
- Bitwarden CLI: ~2.8s per operation (network + CLI overhead)
- aws-vault: ~2-3s (various backends, CLI overhead)

## Security Considerations

- Long-term credentials stored GPG-encrypted via gopass
- Temporary credentials cached with restrictive permissions (0600)
- GPG passphrase required for decrypt (managed by gpg-agent)
- MFA codes read from stdin, not passed as command arguments
- Cache expiration uses 5-minute safety buffer
- No credentials written to shell history or logs

## Important Behaviors

- Secrets are stored with `aws/` prefix in gopass (e.g., `aws/production`)
- The tool does NOT create or manage AWS IAM users/keys
- Temporary credentials are always 1 hour duration (3600 seconds)
- Failed commands preserve their exit codes via `exec.ExitError`
- Cache files are silently used if valid, ignored if expired/corrupt
- gopass store must be initialized before use (`gopass init`)

## Migration from Bitwarden

The old Bitwarden backend code is in `bitwarden.go` but is no longer used. The migration was:

**Before:**
- Shell exec to `bw` CLI
- Network calls to Bitwarden servers
- Manual session management
- ~2.8s per operation

**After:**
- Direct gopass library calls
- Local GPG operations only
- Automatic GPG/session handling
- ~50-100ms overhead

The command interface remains identical - only the backend changed.

## Testing Notes

- Manual testing instructions in `TEST.md`
- Use `PASSWORD_STORE_DIR=/tmp/test-store` for isolated testing
- GPG agent must be running for decryption
- Set `GPG_TTY=$(tty)` if seeing "inappropriate ioctl" errors
- First decrypt in a session requires GPG passphrase
- Subsequent operations use gpg-agent cache
