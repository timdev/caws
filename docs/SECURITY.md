# Security

This document describes caws' security model, best practices, and important considerations for secure usage.

## Security Model

### Core Principles

1. **Long-term credentials never touch disk in plaintext**
   - All AWS Access Keys and Secret Keys encrypted at rest
   - Encryption: Argon2id + AES-256-GCM (industry standard)

2. **Password-based encryption**
   - Master password protects the vault
   - Password never cached in memory
   - No password recovery (by design)

3. **Temporary credentials with limited lifetime**
   - STS credentials valid for 1 hour maximum
   - Cached with 5-minute safety buffer (~55 min effective)
   - Limited blast radius if cache compromised

4. **Strict file permissions**
   - Vault: `0600` (owner read/write only)
   - Cache directory: `0700` (owner only)
   - Cache files: `0600` (owner read/write only)

5. **No credential logging**
   - Credentials never written to shell history
   - No logging to stdout/stderr
   - Secret input hidden during entry

## Encryption Details

### Password Derivation

**Algorithm:** Argon2id

**Parameters:**
- Time cost: 1 iteration
- Memory: 64 MB (65536 KB)
- Threads: 4
- Salt: 32 random bytes (unique per vault)
- Output: 32-byte AES-256 key

**Security properties:**
- Memory-hard: 64 MB per brute-force attempt
- GPU-resistant: Memory requirements make GPU attacks impractical
- Recommended by OWASP, used by 1Password, Bitwarden

### Encryption

**Algorithm:** AES-256-GCM

**Properties:**
- Key size: 256 bits
- Authenticated encryption (tamper-proof)
- Nonce: 12 random bytes per encryption
- NIST-approved, hardware-accelerated

## Threat Model

### What caws Protects Against

✅ **Disk access without password**
- Long-term credentials encrypted on disk
- Brute-force attacks expensive (Argon2id)
- Cannot decrypt vault without password

✅ **Vault file tampering**
- GCM provides authenticated encryption
- Any modification detected during decryption
- Prevents malicious vault edits

✅ **Unauthorized file access**
- File permissions enforced (0600/0700)
- Other users on system cannot read vault or cache

✅ **Weak passwords**
- Argon2id makes weak passwords harder to crack
- Still requires user to choose strong password

### What caws Does NOT Protect Against

❌ **Compromised system**
- Malware can read cache files (valid for 1 hour)
- Keyloggers can capture vault password
- Memory inspection can extract credentials

❌ **Process inspection**
- Commands inherit credentials via environment variables
- System administrator can inspect process environment
- Use on trusted systems only

❌ **Stolen cache files**
- Cached STS credentials valid for up to 1 hour
- No password required to use cache
- Mitigated by short expiration time

❌ **Physical access**
- Someone with physical access can install keylogger
- Can modify binary to log credentials
- Requires disk encryption at OS level

## Best Practices

### 1. Choose a Strong Master Password

Your credentials are only as secure as your master password.

**Recommended:**
- 20+ characters
- Mix of letters, numbers, symbols
- Unique (not used elsewhere)
- Use a password manager to generate and store it

**Avoid:**
- Dictionary words
- Personal information (names, dates)
- Short passwords (<12 characters)
- Reused passwords

**Example strong password:**
```
X7$mK9!pL2@qR5^wT8&nZ3
```

### 2. Enable MFA on Your AWS IAM User

MFA adds a second factor even if long-term credentials are compromised.

**Setup:**
```bash
# When adding profile, include MFA serial
caws add production
# ... enter credentials ...
MFA Serial ARN: arn:aws:iam::123456789012:mfa/your-username
```

**Benefit:**
- Getting NEW temporary credentials requires MFA code
- Prevents attacker from refreshing stolen cached credentials without MFA
- Cached credentials still work until expiration (up to 1 hour) without MFA
- Recommended for all production accounts

### 3. Rotate AWS Credentials Regularly

**Recommended schedule:**
- Production accounts: Every 90 days
- Development accounts: Every 180 days

**Process:**
1. Create new Access Key in AWS IAM
2. Update caws profile with new key
3. Test new key works
4. Delete old key in AWS IAM

```bash
# Update existing profile
caws remove production
caws add production
# ... enter new credentials ...
```

### 4. Backup Your Vault Securely

If you lose `~/.local/share/caws/vault.enc`, you'll need to re-enter all credentials.

**Backup location options:**
- Encrypted USB drive
- Password manager secure notes
- Encrypted cloud storage

**Remember:**
- Backup is useless without master password
- Store password separately (password manager)
- Don't backup cache files (temporary, expired)

```bash
# Backup vault
cp ~/.local/share/caws/vault.enc ~/Backups/caws-vault-$(date +%Y%m%d).enc
```

### 5. Use Separate Profiles for Different Security Levels

**Don't:**
```bash
caws add my-aws  # All accounts in one profile
```

**Do:**
```bash
caws add personal-dev      # Low-security dev account
caws add work-production   # High-security prod account
caws add work-dev          # Medium-security work dev
```

**Benefit:**
- Different MFA requirements per profile
- Easier to audit which credentials are used
- Limits blast radius of compromised cache

### 6. Clear Cache When Done

If you're on a shared system or finishing a work session:

```bash
# Clear cache for specific profile
rm ~/.cache/caws/production.json

# Clear all caches
rm -rf ~/.cache/caws/
```

Next command will prompt for password and call STS.

### 7. Use System Disk Encryption

caws protects long-term credentials, but not cached STS credentials.

**Recommended:**
- macOS: FileVault
- Linux: LUKS, dm-crypt
- Windows: BitLocker

**Benefit:**
- Protects cache files if laptop stolen
- Protects vault if disk removed
- Defense-in-depth

### 8. Run on Trusted Systems Only

Don't use caws on:
- ❌ Shared/multi-user systems
- ❌ Untrusted/public machines
- ❌ Virtual machines in untrusted environments

Do use caws on:
- ✅ Your personal laptop
- ✅ Secure development workstations
- ✅ Trusted servers with disk encryption

### 9. Audit AWS Usage Regularly

Check CloudTrail for unexpected API calls using your credentials.

```bash
# See who you are (verifies credentials work)
caws exec production -- aws sts get-caller-identity
```

### 10. No Password Recovery = No Excuses

**Important:** There is absolutely no way to recover a forgotten password.

**Backup strategy:**
1. Store master password in password manager
2. Test password regularly
3. Backup vault file
4. Document recovery process

## Security Considerations

### Master Password

⚠️ **Your credentials are only as secure as your master password.**

- Use a strong, unique password (20+ characters recommended)
- Store in a password manager
- No password recovery exists (by design)

### Cache Directory

⚠️ **Temporary credentials are cached in `~/.cache/caws/`**

- Files contain valid AWS credentials for up to 1 hour
- Proper permissions (0600) set automatically
- Consider clearing cache on shared systems
- Use disk encryption for additional protection

### MFA Recommendation

⚠️ **Enable MFA on your IAM user for sensitive accounts**

- Adds second factor even if credentials leaked
- Required for many production environments
- Minimal inconvenience (enter 6-digit code)

### Credential Rotation

⚠️ **Rotate long-term AWS credentials regularly**

- Follow your organization's security policy
- Recommended: 90 days for production
- Use AWS IAM to create new keys, update caws
- Delete old keys after migration

### No Password Recovery

⚠️ **If you forget your master password, there's no way to recover it.**

- You'll need to reinitialize (`rm ~/.local/share/caws/vault.enc && caws init`)
- You'll need to re-add all credentials
- Backup vault file to avoid re-entering credentials
- Store password in a password manager

### Vault Backups

⚠️ **Consider backing up `~/.local/share/caws/vault.enc` to a secure location**

- Without password, backup is useless to attacker
- Allows recovery if file corrupted/deleted
- Store password separately (password manager)
- Don't backup to unencrypted cloud storage

## Attack Scenarios

### Scenario 1: Vault File Stolen

**Attacker obtains:** `~/.local/share/caws/vault.enc`

**Can attacker decrypt?**
- No (if strong password used)
- Must brute-force password (Argon2id makes this expensive)
- 64 MB memory per attempt limits parallelization

**Mitigation:**
- Use strong master password (20+ characters)
- Argon2id parameters make brute-force expensive

### Scenario 2: Cache File Stolen

**Attacker obtains:** `~/.cache/caws/production.json`

**Can attacker use credentials?**
- Yes (cache file is plaintext JSON)
- Valid for up to 1 hour
- Limited by STS credential expiration

**Mitigation:**
- Short expiration (1 hour max)
- Use MFA (attacker needs MFA code to refresh)
- Clear cache when done (`rm ~/.cache/caws/*.json`)
- Use disk encryption

### Scenario 3: System Compromised (Malware)

**Attacker has:** Root access, keylogger, memory access

**Can attacker get credentials?**
- Yes (game over - malware can do anything)
- Keylogger captures master password
- Memory inspection extracts credentials
- Can modify binary to log credentials

**Mitigation:**
- Keep system patched and secure
- Use antivirus/endpoint protection
- Don't use caws on compromised systems
- Rotate credentials if compromise suspected

### Scenario 4: Process Inspection

**Attacker has:** System administrator access

**Can attacker see credentials?**
- Yes (via `/proc/<pid>/environ`)
- Credentials passed as environment variables
- Visible to system administrators

**Mitigation:**
- Use caws only on trusted systems
- Trust your system administrators
- Use separate AWS accounts for sensitive resources

## Reporting Security Issues

If you discover a security vulnerability in caws:

1. **Do NOT open a public GitHub issue**
2. Email the maintainer directly (see GitHub profile)
3. Include:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

Responsible disclosure appreciated. Security fixes will be prioritized.

## Compliance Considerations

### SOC 2 / ISO 27001

If your organization requires compliance:

- ✅ Credentials encrypted at rest (Argon2id + AES-256-GCM)
- ✅ Strong password enforcement (user responsibility)
- ✅ Audit trail (AWS CloudTrail for API usage)
- ✅ Temporary credentials (1-hour max)
- ⚠️ MFA support available (user must enable)
- ⚠️ Password rotation manual (no automatic enforcement)

### PCI-DSS

For environments storing cardholder data:

- May require additional controls (disk encryption, MFA mandatory)
- Consult your compliance team
- Consider using AWS IAM roles instead of long-term credentials

### HIPAA

For healthcare applications:

- caws provides encrypted storage (technical safeguard)
- Requires administrative safeguards (policies, training)
- Consider using AWS STS with IAM roles instead

## Comparison to Other Tools

### vs. Storing Credentials in `~/.aws/credentials`

| Security Aspect | caws | ~/.aws/credentials |
|----------------|------|-------------------|
| Credentials encrypted at rest | ✅ Yes | ❌ No (plaintext) |
| Password required for access | ✅ Yes | ❌ No |
| Temporary credentials | ✅ Yes | ❌ No (long-term) |
| File permissions | ✅ 0600 enforced | ⚠️ User responsibility |

**Verdict:** caws significantly more secure

### vs. aws-vault

| Security Aspect | caws | aws-vault |
|----------------|------|-----------|
| Credentials encrypted at rest | ✅ Yes (password) | ✅ Yes (keyring/pass) |
| Backend options | Password only | Keyring/pass/file |
| MFA support | ✅ Yes | ✅ Yes |
| Temporary credentials | ✅ Yes | ✅ Yes |
| Dependencies | None | None |

**Verdict:** Similar security model, different backends

### vs. Environment Variables

| Security Aspect | caws | Environment Variables |
|----------------|------|-----------------------|
| Credentials encrypted at rest | ✅ Yes | ❌ No |
| Visible in process list | ❌ No | ✅ Yes (`ps aux`) |
| Temporary credentials | ✅ Yes | Manual |
| Credential rotation | Easy | Manual, error-prone |

**Verdict:** caws far more secure

## Security Checklist

Before using caws in production:

- [ ] Strong master password chosen (20+ characters)
- [ ] Master password stored in password manager
- [ ] MFA enabled on AWS IAM user
- [ ] System has disk encryption enabled
- [ ] Vault file backed up securely
- [ ] Credential rotation schedule established
- [ ] Using separate profiles for different security levels
- [ ] CloudTrail monitoring enabled for AWS account
- [ ] Team trained on password security
- [ ] Incident response plan includes credential rotation

## Further Reading

- [OWASP Password Storage Cheat Sheet](https://cheatsheetsecurity.com/cheatsheets/password-storage-cheat-sheet/)
- [AWS Security Best Practices](https://aws.amazon.com/architecture/security-identity-compliance/)
- [Argon2 Specification](https://github.com/P-H-C/phc-winner-argon2)
- [NIST Guidelines on Password Security](https://pages.nist.gov/800-63-3/sp800-63b.html)
