# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`caws` (Credential AWS) is a CLI tool that manages AWS credentials using password-based encryption. It's a fast, local-first alternative to aws-vault with zero external dependencies.

**Core workflow:**
1. Long-term AWS credentials stored encrypted in `~/.caws/vault.enc` (Argon2id + AES-256-GCM)
2. Tool prompts for vault password and decrypts credentials
3. Exchanges long-term credentials for temporary STS credentials (1-hour duration)
4. Caches temporary credentials in `~/.caws/cache/<profile>.json`
5. Executes commands with credentials injected as environment variables

## Build & Development Commands

```bash
# Build the binary
go build -o caws

# Run without installing
./caws <command>

# Install to system
sudo mv caws /usr/local/bin/

# Install for local user
mv caws ~/.local/bin/  # Ensure ~/.local/bin is in PATH
```

**Testing:** See `TEST.md` for manual testing instructions.

**No unit tests** currently exist in the repository.

## Architecture

### File Structure

- `main.go` - Entry point, command routing and usage text
- `commands.go` - Command handlers (add, list, exec, remove)
- `vault.go` - Vault client (CRUD operations on encrypted credentials)
- `crypto.go` - Encryption/decryption primitives (Argon2id + AES-256-GCM)
- `aws.go` - AWS STS integration, credential caching, environment variable management

### Key Components

**VaultClient** (`vault.go`)
- Manages encrypted credential vault at `~/.caws/vault.enc`
- Prompts for password via `golang.org/x/term.ReadPassword()`
- Methods: `GetCredentials()`, `CreateCredentials()`, `ListProfiles()`, `RemoveProfile()`
- Each operation: prompt password → decrypt → operate → re-encrypt (if modified) → write
- Must call `Close()` for interface compatibility (no-op in current implementation)

**Encryption** (`crypto.go`)
- **Password derivation**: Argon2id (memory-hard, GPU-resistant)
  - Time: 1 iteration
  - Memory: 64 MB
  - Threads: 4
  - Output: 32 bytes (AES-256 key)
- **Encryption**: AES-256-GCM (authenticated encryption)
- **Vault format**: JSON with version, salt, nonce, encrypted data
- Functions: `encryptVault()`, `decryptVault()`, `deriveKey()`

**Credential Flow** (`commands.go:handleExec()`)
1. Create VaultClient (prompts for password, verifies by decrypting)
2. Retrieve profile credentials via `GetCredentials(profile)` (decrypts vault)
3. Check cache for valid temporary credentials (`GetCachedCredentials()`)
4. If cache miss/expired: call AWS STS via `aws` CLI (`AssumeRole()`)
5. Prompt for MFA code if `mfa_serial` field present
6. Cache temporary credentials with 5-minute expiration buffer
7. Execute command with credentials in environment
8. Close VaultClient (no-op)

**Vault Storage Schema**
```
Location: ~/.caws/vault.enc
Permissions: 0600 (owner read/write only)

Encrypted format:
{
  "version": 1,
  "salt": "base64-encoded-32-bytes",
  "nonce": "base64-encoded-12-bytes",
  "data": "base64-encoded-aes-gcm-ciphertext"
}

Decrypted data structure:
{
  "profiles": {
    "<profile-name>": {
      "access_key": "AKIA...",
      "secret_key": "...",
      "region": "us-east-1",       // optional
      "mfa_serial": "arn:aws:..."  // optional
    }
  }
}
```

**Caching** (`aws.go`)
- Cache location: `~/.caws/cache/<profile>.json`
- Cache directory permissions: 0700 (owner only)
- Cache file permissions: 0600 (owner read/write only)
- Expiration buffer: 5 minutes before actual expiration
- Cache format: JSON with STS credentials + expiration timestamp

### External Dependencies

**Runtime requirements:**
- `aws` (AWS CLI) - Used for STS operations

**Go dependencies:**
- `golang.org/x/crypto` - Argon2id key derivation
- `golang.org/x/term` - Secure password input (ReadPassword)
- `golang.org/x/sys` - System calls (used by term)

**Key commands executed:**
- `aws sts get-session-token` - Get temporary credentials
- No other external commands

## Common Development Patterns

### Adding New Commands

1. Add case to switch in `main.go:main()`
2. Implement handler function in `commands.go` (prefix: `handle*`)
3. Update `printUsage()` in `main.go`
4. Remember to call `defer client.Close()` on VaultClient instances

### Vault Operations

Always create client with `NewVaultClient()` and defer `Close()`:

```go
client, err := NewVaultClient()
if err != nil {
    // Handle error
}
defer client.Close()  // No-op currently, but kept for interface compatibility
```

Password prompting happens in `NewVaultClient()`:
- Uses `term.ReadPassword()` for hidden input
- Verifies password by attempting to decrypt vault
- Returns error if password is wrong or vault is corrupted

### Error Handling

The codebase uses a consistent pattern:
- Print error to stdout with `fmt.Printf()`
- Call `os.Exit(1)` for fatal errors
- Return errors from library functions

### Input Handling

For password input (see `vault.go:NewVaultClient()`):
```go
fmt.Print("Enter vault password: ")
passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
fmt.Println() // New line after password
password := string(passwordBytes)
```

For secret input (see `commands.go:handleAdd()`):
```go
exec.Command("stty", "-echo").Run()  // Disable echo
// Read input
exec.Command("stty", "echo").Run()   // Re-enable echo
```

**Note:** `stty` approach is Unix-specific and won't work on Windows.

## Performance Characteristics

**Current implementation:**
- Vault decrypt: ~10-50ms (Argon2id + AES-256-GCM)
- Cold start (with STS): ~1.4s (mostly AWS STS call)
- Warm start (cached STS): ~0.1s (no vault access needed)
- caws overhead: ~50ms

**Why so fast:**
- Pure Go encryption (no external processes)
- No network calls for credential retrieval
- Minimal overhead (direct memory operations)

**Password entry frequency:**
- Approximately once per hour per active profile
- Vault decrypted on-demand (not cached in memory)
- STS credentials cached for ~55 minutes

## Security Considerations

- Long-term credentials encrypted with Argon2id + AES-256-GCM
- Vault file permissions: 0600 (owner read/write only)
- Password never cached in memory (re-prompted for each vault access)
- Temporary credentials cached with restrictive permissions (0600)
- MFA codes read from stdin, not passed as command arguments
- Cache expiration uses 5-minute safety buffer
- No credentials written to shell history or logs
- No plaintext long-term credentials ever touch disk

## Important Behaviors

- Vault must be initialized before use (`caws init`)
- The tool does NOT create or manage AWS IAM users/keys
- Temporary credentials are always 1 hour duration (3600 seconds)
- Failed commands preserve their exit codes via `exec.ExitError`
- Cache files are silently used if valid, ignored if expired/corrupt
- If password is forgotten, vault cannot be recovered (must reinitialize)
- Vault is atomic-write (writes to .tmp, then renames)

## Migration History

### From gopass/GPG (Previous Version)
**Before:**
- Direct gopass library calls
- Local GPG operations
- Automatic GPG/session handling via gpg-agent
- ~50-100ms overhead
- Required gopass + GPG installed

**After (Current):**
- Custom password-based encryption
- Pure Go crypto (Argon2id + AES-256-GCM)
- Password prompt on each vault access
- ~50ms overhead
- Zero external dependencies

The command interface remains identical - only the backend changed.

### From Bitwarden (Original Version)
**Before:**
- Shell exec to `bw` CLI
- Network calls to Bitwarden servers
- Manual session management
- ~2.8s per operation

**After:**
- Custom encryption (see above)
- Local operations only
- Simple password prompt
- ~50ms overhead

## Testing Notes

- Manual testing instructions in `TEST.md`
- Test vault can be created at custom location (modify `getVaultPath()`)
- No GPG setup needed
- First operation in session requires password
- STS cache works identically to previous versions

## Vault File Format Details

The vault uses a versioned format to allow future migrations:

**Version 1** (current):
- Salt: 32 random bytes (for Argon2id)
- Nonce: 12 bytes (for AES-GCM)
- Data: AES-GCM encrypted JSON (includes authentication tag)

If vault version doesn't match `vaultVersion` constant in `crypto.go`, decryption fails with error.

## Code Organization Principles

- `crypto.go` - Pure crypto primitives (no I/O, no user interaction)
- `vault.go` - Vault management (prompts, file I/O, CRUD)
- `commands.go` - Command handlers (orchestration)
- `main.go` - Entry point (routing only)
- `aws.go` - AWS-specific logic (unchanged from previous versions)

Keep crypto operations separate from business logic for easier auditing and testing.
