# Architecture

This document explains how caws works internally, including the credential flow, encryption implementation, and performance characteristics.

## Overview

caws is a credential manager that securely stores AWS long-term credentials and exchanges them for temporary credentials via AWS STS. The core workflow:

1. Long-term AWS credentials stored encrypted in `~/.local/share/caws/vault.enc` (XDG Data Home)
2. Tool checks for valid cached credentials first
3. If cache expired/missing: prompts for vault password and decrypts credentials
4. Exchanges long-term credentials for temporary STS credentials (1-hour duration)
5. Caches temporary credentials in `~/.cache/caws/<profile>.json` (XDG Cache Home)
6. Executes commands with credentials injected as environment variables

## Credential Flow

### Step-by-Step Execution

When you run `caws exec production -- aws s3 ls`:

1. **Vault Access**
   - Prompt user for vault password
   - Read `~/.caws/vault.enc`
   - Derive encryption key from password using Argon2id
   - Decrypt vault to access long-term credentials for `production` profile

2. **Cache Check**
   - Check if `~/.caws/cache/production.json` exists
   - Verify cached credentials haven't expired (with 5-minute buffer)
   - If valid cache exists, skip to step 4

3. **STS Exchange** (if no valid cache)
   - Call AWS STS `GetSessionToken` API with long-term credentials
   - If MFA configured, prompt for MFA code
   - Receive temporary credentials (AccessKeyId, SecretAccessKey, SessionToken)
   - Expiration: 3600 seconds (1 hour) from now

4. **Cache Storage**
   - Write temporary credentials to `~/.caws/cache/production.json`
   - Set file permissions to 0600 (owner read/write only)

5. **Command Execution**
   - Inject temporary credentials as environment variables:
     - `AWS_ACCESS_KEY_ID` - Temporary access key
     - `AWS_SECRET_ACCESS_KEY` - Temporary secret key
     - `AWS_SESSION_TOKEN` - Session token
     - `AWS_REGION` - Region from config (if set)
     - `AWS_DEFAULT_REGION` - Same as AWS_REGION
     - `AWS_VAULT` - Profile name (for shell prompts)
     - `AWS_CREDENTIAL_EXPIRATION` - Expiration timestamp
   - Execute user's command
   - Return command's exit code

## Encryption Implementation

### Password Derivation: Argon2id

caws uses **Argon2id** to derive an encryption key from the user's password. Argon2id is the winner of the Password Hashing Competition and is resistant to GPU/ASIC attacks.

**Parameters:**
- Algorithm: Argon2id
- Time cost: 1 iteration
- Memory: 64 MB (65536 KB)
- Threads: 4
- Salt: 32 random bytes (unique per vault)
- Output: 32 bytes (256-bit key for AES-256)

**Why Argon2id?**
- Memory-hard: Makes brute-force attacks expensive (requires 64MB per attempt)
- GPU-resistant: Memory requirements make GPU attacks impractical
- Tunable: Can increase memory/time if needed for future security
- Industry standard: Used by 1Password, Bitwarden, etc.

### Encryption: AES-256-GCM

After deriving the key, credentials are encrypted using **AES-256-GCM**.

**Properties:**
- Algorithm: AES-256-GCM (Galois/Counter Mode)
- Key size: 256 bits (from Argon2id output)
- Nonce: 12 random bytes (unique per encryption)
- Authentication: Built-in (GCM mode provides authenticated encryption)

**Why AES-256-GCM?**
- AEAD: Authenticated Encryption with Associated Data
- Tamper-proof: Any modification to ciphertext is detected
- Fast: Hardware-accelerated on modern CPUs (AES-NI)
- Standard: NIST-approved, widely audited

### Vault File Format

**Location:** `~/.caws/vault.enc`
**Permissions:** `0600` (owner read/write only)

**Encrypted format (on disk):**
```json
{
  "version": 1,
  "salt": "base64-encoded-32-bytes",
  "nonce": "base64-encoded-12-bytes",
  "data": "base64-encoded-aes-gcm-ciphertext"
}
```

**Decrypted data structure (in memory only):**
```json
{
  "profiles": {
    "production": {
      "access_key": "AKIA...",
      "secret_key": "..."
    },
    "development": {
      "access_key": "AKIA...",
      "secret_key": "..."
    }
  }
}
```

**Note:** Region and MFA serial are NOT stored in the vault. They are read from `~/.aws/config` when needed.

**Vault Operations:**
- **Read**: Decrypt entire vault, operate on in-memory data, re-encrypt, write
- **Write**: Use atomic writes (write to `.tmp` file, then rename)
- **Version**: Currently version 1; allows future format migrations

## Caching

### Temporary Credential Cache

**Location:** `~/.cache/caws/<profile>.json` (XDG Cache Home)
**Permissions:** `0600` (owner read/write only)
**Directory permissions:** `0700` (owner only)

**Cache file format:**
```json
{
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "...",
  "SessionToken": "...",
  "Expiration": "2025-10-26T15:30:00Z"
}
```

**Expiration logic:**
- STS credentials valid for 3600 seconds (1 hour)
- caws uses 5-minute safety buffer
- Credentials refreshed when <5 minutes remain
- Effective cache duration: ~55 minutes

**Why cache?**
- Reduces AWS API calls (STS rate limits apply)
- Faster execution (~0.1s vs ~1.4s for cold start)
- Less password prompting (once per hour vs every command)

## File Structure

```
~/.local/share/caws/   # XDG Data Home
├── vault.enc          # Encrypted credential vault (0600)
└── .lock              # Vault lock file with PID (0600)

~/.cache/caws/         # XDG Cache Home (0700)
├── production.json    # Cached STS creds for 'production' (0600)
└── development.json   # Cached STS creds for 'development' (0600)

~/.aws/                # AWS configuration (standard location)
└── config             # Profile settings: region, MFA serial
```

## Code Organization

### Source Files

**`main.go`**
- Entry point
- Command routing (switch statement)
- Usage text

**`commands.go`**
- Command handlers: `handleInit()`, `handleAdd()`, `handleList()`, `handleExec()`, `handleLogin()`, `handleRemove()`
- User interaction
- Orchestration logic

**`vault.go`**
- `VaultClient` struct and methods
- Vault CRUD operations: `GetCredentials()`, `CreateCredentials()`, `ListProfiles()`, `RemoveProfile()`
- Password prompting (`term.ReadPassword()`)
- File I/O with atomic writes

**`crypto.go`**
- Pure cryptographic primitives (no I/O, no user interaction)
- `deriveKey()` - Argon2id key derivation
- `encryptVault()` - AES-256-GCM encryption
- `decryptVault()` - AES-256-GCM decryption

**`aws.go`**
- AWS STS integration
- `GetCachedCredentials()` - Read cache
- `GetSessionToken()` - Call AWS STS `GetSessionToken` API
- `GetFederationToken()` - Call AWS STS `GetFederationToken` API (for login command)
- Cache management
- Environment variable injection

**`config.go`**
- AWS config file reading (`~/.aws/config`)
- Profile settings: region, MFA serial
- Config file parsing

**`validation.go`**
- Input validation utilities
- Profile name validation
- Credential format validation

**`xdg.go`**
- XDG Base Directory specification support
- Path resolution for vault and cache directories

### Key Design Patterns

**VaultClient lifecycle:**
```go
client, err := NewVaultClient()  // Prompts password, verifies by decrypting, acquires lock
if err != nil {
    // Handle error
}
defer client.Close()  // Releases vault lock file

// Use client...
```

**Vault locking:**
- Lock file created at `~/.local/share/caws/.lock` when vault is accessed
- Contains process PID for identification
- Prevents concurrent vault access
- Automatically cleaned up by `Close()`

**Atomic vault writes:**
```go
// Write to temporary file
vault.enc.tmp

// Rename (atomic operation)
rename(vault.enc.tmp, vault.enc)
```

**Error handling:**
- Library functions return errors
- Command handlers print errors and call `os.Exit(1)`
- Consistent pattern throughout codebase

## Performance

### Benchmarks

| Operation | Time | Notes |
|-----------|------|-------|
| Cold start (first exec) | ~1.4s | Includes vault decrypt + AWS STS call |
| Warm start (cached STS) | ~0.1s | Uses cached credentials |
| Vault decrypt | ~10-50ms | Argon2id + AES-256-GCM |
| caws overhead | ~50ms | Pure Go operations |

### Performance Characteristics

**Why so fast?**
- Pure Go implementation (no external processes)
- Direct memory operations (no temporary files)
- Hardware-accelerated AES (AES-NI on modern CPUs)
- Efficient Argon2id parameters (balanced security/speed)

**Where time is spent:**
- Argon2id derivation: ~10-30ms (memory-hard operation)
- AES-256-GCM: <5ms (hardware accelerated)
- AWS STS call: ~1.2-1.4s (network round-trip)
- Command execution: Varies by command

**Comparison to aws-vault:**
- Similar performance for typical operations
- Both use local caching (STS calls are the bottleneck)
- caws avoids GPG overhead (pure Go crypto)

### Password Entry Frequency

With STS credential caching:
- **Approximately once per hour per active profile** during normal usage
- Vault password required for every vault operation (list, add, remove)
- Vault password required for exec (but credentials cached for ~55 min)
- No password caching in memory (security trade-off)

## Security Boundaries

### What's Encrypted

✅ **Always encrypted on disk:**
- Long-term AWS Access Keys
- Long-term AWS Secret Keys
- Profile metadata (region, MFA serial)

❌ **Not encrypted (temporary, time-limited):**
- Temporary STS credentials in cache (valid 1 hour, 0600 permissions)
- Credentials in memory during command execution
- Environment variables passed to subprocess

### Threat Model

**Protected against:**
- ✅ Disk access without password (long-term creds encrypted)
- ✅ Brute-force password attacks (Argon2id memory-hard)
- ✅ Vault tampering (GCM authenticated encryption)
- ✅ File permission issues (0600/0700 enforced)

**Not protected against:**
- ❌ Memory access (credentials in memory during execution)
- ❌ Process inspection (subprocess inherits credentials)
- ❌ Compromised system (malware can read cache files)
- ❌ Stolen cache files (valid for up to 1 hour)

**Mitigation strategies:**
- Short cache expiration (1 hour max)
- No long-term credentials on disk in plaintext
- File permissions prevent other users
- MFA adds second factor

## Dependencies

### Runtime

**Zero external dependencies**
- Self-contained static binary
- No AWS CLI required
- No GPG, gopass, or external password managers

### Build-time (Go modules)

**Cryptography:**
- `golang.org/x/crypto` - Argon2id implementation
- `golang.org/x/term` - Secure password input (ReadPassword)
- `golang.org/x/sys` - System calls (used by term)

**AWS:**
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for Go v2
- `github.com/aws/aws-sdk-go-v2/config` - AWS config loading
- `github.com/aws/aws-sdk-go-v2/credentials` - Static credential provider
- `github.com/aws/aws-sdk-go-v2/service/sts` - AWS STS API client

**Testing:**
- `github.com/stretchr/testify` - Test assertions

All dependencies are Go modules (compiled into binary).

### AWS API Calls

caws makes the following AWS STS API calls:

- **`sts:GetSessionToken`** - Used by `caws exec` command to get temporary credentials for command execution
- **`sts:GetFederationToken`** - Used by `caws login` command to generate AWS Console sign-in URLs

Both APIs require valid long-term AWS credentials (Access Key ID and Secret Access Key).

## Future Considerations

### Potential Optimizations

1. **Memory locking**: Use `mlock()` to prevent credentials from swapping to disk
2. **Secure memory**: Zero out sensitive data after use
3. **Key derivation caching**: Cache derived key in memory during session (security trade-off)
4. **Parallel Argon2id**: Use more CPU cores for key derivation

### Potential Features

1. **Vault versioning**: Support multiple vault format versions
2. **Key rotation**: Change vault password without re-entering all credentials
3. **Role assumption**: Support AWS IAM role assumption chains
4. **Session management**: Longer-lived sessions across vault operations

### Known Limitations

1. **No password recovery**: Forgotten password = lost credentials (by design)
2. **Single vault**: All profiles in one vault (per-profile vaults planned)
3. **No role assumption**: Only direct credential usage (no `sts:AssumeRole`)
4. **Unix-focused**: Some input code uses `stty` (not Windows-compatible)
