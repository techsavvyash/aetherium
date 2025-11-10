# Aetherium Security Guide

This document describes the security features for safely injecting secrets (API keys, tokens, credentials) into VM workloads.

## Table of Contents
- [Overview](#overview)
- [Quick Start](#quick-start)
- [Phase 1: Transient Secrets](#phase-1-transient-secrets)
- [Phase 2: Vault Integration](#phase-2-vault-integration)
- [Usage Examples](#usage-examples)
- [Best Practices](#best-practices)

---

## Overview

Aetherium provides two layers of secrets management:

### Phase 1: Transient Secrets (Implemented) ✅
- Secrets passed to VMs but **never persisted to database**
- Automatic redaction of secrets from stdout/stderr
- Pattern-based detection of leaked credentials
- Prevents accidental exposure via API

### Phase 2: Encrypted Secrets (Implemented) ✅
- HashiCorp Vault integration for encryption-as-a-service
- Encrypted storage in PostgreSQL
- Key rotation without data re-encryption
- Full audit trail for compliance

---

## Quick Start

### Using Transient Secrets (No Vault Required)

```bash
# Execute command with GitHub token
curl -X POST http://localhost:8080/api/v1/vms/{vm-id}/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "git",
    "args": ["clone", "https://github.com/myorg/private-repo"],
    "transient_secrets": {
      "GITHUB_TOKEN": "ghp_YourPersonalAccessToken"
    }
  }'
```

**Security Guarantee:**
- ✅ Token available in VM as `$GITHUB_TOKEN`
- ✅ Token **NOT** stored in `executions.env` column
- ✅ If token appears in output, replaced with `[REDACTED]`
- ✅ `secret_redacted=true` flag set on execution record

---

## Phase 1: Transient Secrets

### How It Works

```
┌─────────────────────────────────────────────────┐
│ 1. API Request with transient_secrets          │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ 2. Worker merges env + transient_secrets       │
│    (transient secrets override env)             │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ 3. VMM passes merged environment to VM          │
│    Command executes with all variables          │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ 4. Worker creates redactor with secret values  │
│    Redacts stdout/stderr before persistence     │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ 5. Database stores:                             │
│    - env: only non-secret variables             │
│    - transient_secrets: NOT STORED              │
│    - stdout/stderr: [REDACTED]                  │
│    - secret_redacted: true                      │
└─────────────────────────────────────────────────┘
```

### API Fields

```go
type ExecuteCommandRequest struct {
    Command          string            `json:"command"`
    Args             []string          `json:"args,omitempty"`
    Env              map[string]string `json:"env,omitempty"`              // Persisted to DB
    TransientSecrets map[string]string `json:"transient_secrets,omitempty"` // NOT persisted
}
```

### Redaction Patterns

The redactor automatically detects and redacts:

- **GitHub Tokens**: `ghp_*`, `gho_*`, `ghs_*`, `ghr_*`, `ghu_*`
- **AWS Keys**: `AKIA*`, `aws_secret_access_key`
- **Claude API Keys**: `sk-ant-api03-*`, `sk-ant-*`
- **OpenAI Keys**: `sk-*` (48 chars)
- **Generic API Keys**: `api_key=*`, `API_KEY: *`
- **Bearer Tokens**: `Authorization: Bearer *`
- **Private Keys**: `-----BEGIN * PRIVATE KEY-----`
- **JWTs**: `eyJ*.eyJ*.*`
- **Slack Tokens**: `xox[baprs]-*`
- **Stripe Keys**: `sk_live_*`, `rk_live_*`

**Custom values** are also redacted by exact match.

### Example: Before vs After

**Before (VULNERABLE - pre-Phase 1):**
```sql
SELECT stdout FROM executions WHERE vm_id = 'abc-123';
-- Output: "Cloning with token: ghp_abc123xyz789..."
```

**After (SECURE - with Phase 1):**
```sql
SELECT stdout FROM executions WHERE vm_id = 'abc-123';
-- Output: "Cloning with token: [REDACTED]"
```

---

## Phase 2: Vault Integration

### Architecture

```
┌──────────────┐
│ PostgreSQL   │ ← Encrypted ciphertext (base64)
│  secrets     │
│  table       │
└──────┬───────┘
       │
       │ Store/Retrieve ciphertext
       │
       ▼
┌──────────────────────────────────────────┐
│         Aetherium API/Worker             │
│  - Encrypts before storing               │
│  - Decrypts before using                 │
└──────────────┬───────────────────────────┘
               │
               │ Encrypt/Decrypt API calls
               │
               ▼
┌──────────────────────────────────────────┐
│      HashiCorp Vault (Transit)           │
│  - AES-256-GCM encryption                │
│  - Keys never leave Vault                │
│  - Supports key rotation                 │
└──────────────────────────────────────────┘
```

### Setup Vault

```bash
# 1. Start Vault
docker-compose up -d vault

# 2. Initialize transit engine and create application token
docker exec -it aetherium-vault sh
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=aetherium-root-token

# Run init script (already mounted in container)
/scripts/init-vault.sh

# Output:
# VAULT_ADDR=http://localhost:8200
# VAULT_TOKEN=hvs.ABC123XYZ... (save this!)
```

### Configure Aetherium

```yaml
# config.yaml
secrets:
  provider: vault
  vault:
    address: http://localhost:8200
    token: hvs.ABC123XYZ...  # From init-vault.sh
    mount_path: transit
    default_key: aetherium
```

### Store a Secret

```go
import (
    "github.com/aetherium/aetherium/pkg/security"
    "github.com/aetherium/aetherium/pkg/storage"
)

// 1. Create encryption service
encSvc, err := security.NewVaultEncryptionService(security.VaultConfig{
    Address:    "http://localhost:8200",
    Token:      "hvs.ABC123XYZ...",
    DefaultKey: "aetherium",
})

// 2. Encrypt the secret
result, err := encSvc.Encrypt(ctx, "ghp_MyGitHubToken123", "aetherium")

// 3. Store encrypted value in PostgreSQL
secret := &storage.Secret{
    ID:                uuid.New(),
    Name:              "github-pat",
    Description:       strPtr("GitHub Personal Access Token"),
    EncryptedValue:    result.Ciphertext,  // Vault ciphertext
    EncryptionKeyID:   result.KeyID,
    EncryptionVersion: result.Version,
    Scope:             "global",
    CreatedBy:         strPtr("admin"),
    Tags:              map[string]interface{}{"env": "prod"},
}

err = store.Secrets().Create(ctx, secret)
```

### Use a Secret

```go
// 1. Retrieve from database
secret, err := store.Secrets().GetByName(ctx, "github-pat")

// 2. Decrypt with Vault
plaintext, err := encSvc.Decrypt(ctx, secret.EncryptedValue, secret.EncryptionKeyID)

// 3. Use as transient secret
taskService.ExecuteCommandTaskWithEnv(ctx, vmID, "git", []string{"clone", "..."}, nil, map[string]string{
    "GITHUB_TOKEN": plaintext,
})

// 4. Record access for audit
store.Secrets().RecordAccess(ctx, secret.ID)
```

### Audit Trail

```go
// Get audit logs for a secret
logs, err := store.Secrets().GetAuditLogs(ctx, secretID, 100)

for _, log := range logs {
    fmt.Printf("%s: %s by %s from %s\n",
        log.Timestamp,
        log.Action,      // create, read, update, delete, rotate
        *log.Actor,
        *log.ActorIP,
    )
}
```

---

## Usage Examples

### Example 1: Clone Private GitHub Repo

```bash
curl -X POST http://localhost:8080/api/v1/vms/{vm-id}/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "git",
    "args": ["clone", "https://github.com/myorg/private-repo", "/workspace/repo"],
    "transient_secrets": {
      "GITHUB_TOKEN": "ghp_abc123def456ghi789jkl012mno345pqr678"
    }
  }'
```

**What happens:**
1. Git uses `$GITHUB_TOKEN` for authentication
2. Token **not** stored in database
3. If token appears in output, it's redacted: `"Cloning with [REDACTED]"`

### Example 2: Call Claude API

```bash
curl -X POST http://localhost:8080/api/v1/vms/{vm-id}/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "curl",
    "args": [
      "-X", "POST",
      "https://api.anthropic.com/v1/messages",
      "-H", "x-api-key: $CLAUDE_API_KEY",
      "-H", "Content-Type: application/json",
      "-d", "{\"model\":\"claude-3-5-sonnet-20241022\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}],\"max_tokens\":100}"
    ],
    "transient_secrets": {
      "CLAUDE_API_KEY": "sk-ant-api03-..."
    }
  }'
```

### Example 3: Smart Execute with Secrets

```bash
curl -X POST http://localhost:8080/api/v1/smart-execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "npm install && npm run deploy",
    "transient_secrets": {
      "NPM_TOKEN": "npm_abc123...",
      "AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
      "AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    },
    "required_tools": ["nodejs"],
    "vcpus": 2,
    "memory_mb": 2048
  }'
```

### Example 4: Using Stored Secrets (Vault)

```go
// In your application code
func deployWithSecrets(ctx context.Context, vmID string) error {
    // Retrieve and decrypt secrets from Vault
    npmToken, _ := getSecret(ctx, "npm-token")
    awsKey, _ := getSecret(ctx, "aws-access-key")
    awsSecret, _ := getSecret(ctx, "aws-secret-key")

    // Execute with transient secrets
    taskID, err := taskService.ExecuteCommandTaskWithEnv(
        ctx,
        vmID,
        "bash",
        []string{"-c", "npm install && npm run deploy"},
        nil, // no persistent env vars
        map[string]string{
            "NPM_TOKEN":            npmToken,
            "AWS_ACCESS_KEY_ID":    awsKey,
            "AWS_SECRET_ACCESS_KEY": awsSecret,
        },
    )

    return err
}
```

---

## Best Practices

### ✅ DO

- **Use transient_secrets for all sensitive data**
  - API keys, tokens, passwords, credentials

- **Store secrets in Vault for reuse**
  - Centralized management
  - Key rotation without code changes
  - Audit trail for compliance

- **Set expiration times**
  ```go
  secret.ExpiresAt = timePtr(time.Now().Add(90 * 24 * time.Hour))
  ```

- **Use scoped secrets**
  ```go
  secret.Scope = "vm"
  secret.ScopeID = &vmID  // Secret only for this VM
  ```

- **Rotate secrets regularly**
  ```bash
  vault write -f transit/keys/aetherium/rotate
  ```

- **Monitor audit logs**
  ```sql
  SELECT * FROM secret_audit_logs
  WHERE action = 'read' AND timestamp > NOW() - INTERVAL '1 day';
  ```

### ❌ DON'T

- **Don't use `env` for secrets**
  ```json
  {
    "env": {"API_KEY": "secret"},  // ❌ Will be stored in DB!
    "transient_secrets": {"API_KEY": "secret"}  // ✅ Not stored
  }
  ```

- **Don't log secrets**
  ```bash
  echo "Token: $GITHUB_TOKEN"  # ❌ Will leak in stdout
  ```

- **Don't commit secrets to git**
  ```bash
  # Use .env files (gitignored)
  echo "VAULT_TOKEN=hvs.ABC123" > .env
  ```

- **Don't share Vault root token**
  ```bash
  # Use application tokens from init-vault.sh
  VAULT_TOKEN=hvs.AppToken123  # ✅ Limited permissions
  ```

---

## Security Guarantees

### What's Protected ✅

1. **Secrets never persisted to database**
   - `transient_secrets` only in memory
   - Cleared after execution completes

2. **Automatic output redaction**
   - Pattern-based detection
   - Exact value replacement
   - Works for stdout, stderr, and logs

3. **Encrypted storage (Vault)**
   - AES-256-GCM encryption
   - Keys never leave Vault
   - Supports key rotation

4. **Audit trail**
   - Every secret access logged
   - Timestamps, actors, IPs tracked
   - Compliance-ready

5. **API security**
   - `env` field removed from ExecutionResponse
   - `secret_redacted` flag indicates secret usage
   - No way to retrieve secrets via API after execution

### What's NOT Protected ❌

1. **Network traffic to/from VM**
   - Use HTTPS for external API calls
   - Consider VPN for sensitive networks

2. **VM memory dumps**
   - Secrets in memory during execution
   - Use short-lived VMs

3. **Root access to PostgreSQL**
   - Vault ciphertext stored in DB
   - Limit database access

4. **Vault compromise**
   - If Vault is compromised, all secrets at risk
   - Use Vault's sealing and unsealing features
   - Regular key rotation

---

## Migration Path

### Existing Secrets in Database (Pre-Phase 1)

```sql
-- Check for leaked secrets
SELECT id, command, env FROM executions
WHERE env IS NOT NULL AND secret_redacted = false;

-- Phase 1 migration already purged env data:
UPDATE executions SET env = NULL WHERE env IS NOT NULL;
```

### Moving to Vault (Phase 1 → Phase 2)

```go
// 1. Extract secrets from your config/environment
secrets := map[string]string{
    "github-token": os.Getenv("GITHUB_TOKEN"),
    "claude-key":   os.Getenv("CLAUDE_API_KEY"),
}

// 2. Encrypt and store in Vault
for name, value := range secrets {
    result, _ := encSvc.Encrypt(ctx, value, "aetherium")
    secret := &storage.Secret{
        Name:              name,
        EncryptedValue:    result.Ciphertext,
        EncryptionKeyID:   "aetherium",
        EncryptionVersion: result.Version,
        Scope:             "global",
    }
    store.Secrets().Create(ctx, secret)
}

// 3. Update your code to retrieve from Vault
```

---

## Troubleshooting

### Secrets appearing in logs

```bash
# Check if redaction is working
SELECT stdout FROM executions WHERE secret_redacted = true LIMIT 1;
# Should see [REDACTED] instead of actual values
```

### Vault connection errors

```bash
# Check Vault health
curl http://localhost:8200/v1/sys/health

# Test encryption
vault write transit/encrypt/aetherium plaintext=$(echo -n "test" | base64)
```

### Audit log not created

```go
// Manually create audit log
log := &storage.SecretAuditLog{
    ID:         uuid.New(),
    SecretID:   secret.ID,
    SecretName: secret.Name,
    Action:     "read",
    Actor:      strPtr("system"),
    Timestamp:  time.Now(),
}
store.Secrets().CreateAuditLog(ctx, log)
```

---

## Related Files

- `/pkg/security/redactor.go` - Output redaction logic
- `/pkg/security/encryption.go` - Encryption service interface
- `/pkg/security/vault.go` - Vault implementation
- `/pkg/storage/postgres/secrets.go` - Secrets repository
- `/migrations/000003_add_secret_redacted.up.sql` - Phase 1 migration
- `/migrations/000004_add_secrets_table.up.sql` - Phase 2 migration
- `/scripts/init-vault.sh` - Vault initialization

---

## Support

For questions or issues:
- Check `CLAUDE.md` for architecture details
- Review test files in `/pkg/security/*_test.go`
- See examples in `/tests/integration/`

---

**Last Updated:** 2025-11-10
**Version:** 1.0.0
