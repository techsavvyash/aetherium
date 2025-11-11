# HashiCorp Vault Integration

Complete guide for using HashiCorp Vault for secret management in Aetherium.

## Overview

Aetherium uses HashiCorp Vault to centralize and secure all sensitive data:

- ✅ Database credentials (PostgreSQL)
- ✅ Redis connection info
- ✅ Integration tokens (GitHub, Slack)
- ✅ API keys and secrets
- ✅ Service-to-service authentication tokens

**Benefits:**
- **Centralized**: All secrets in one place
- **Encrypted**: Secrets encrypted at rest and in transit
- **Audited**: Full audit trail of secret access
- **Dynamic**: Rotate secrets without redeploying
- **Access Control**: Fine-grained policies per service

## Architecture

```
┌─────────────────────────────────────────────────┐
│            HashiCorp Vault                      │
│         (Port 8200)                             │
│                                                  │
│  ┌─────────────────────────────────────────┐   │
│  │  KV v2 Secrets Engine                   │   │
│  │                                          │   │
│  │  secret/database/postgres               │   │
│  │  secret/redis/config                    │   │
│  │  secret/integrations/github             │   │
│  │  secret/integrations/slack              │   │
│  └─────────────────────────────────────────┘   │
│                                                  │
│  Storage: Consul                                │
│  UI: http://localhost:8200                      │
└───────────┬─────────────┬──────────────┬────────┘
            │             │              │
     ┌──────▼─────┐ ┌────▼──────┐ ┌─────▼──────┐
     │   Worker   │ │ API       │ │  Services  │
     │            │ │ Gateway   │ │            │
     │ (Policy:   │ │ (Policy:  │ │ (Policy:   │
     │  worker)   │ │  api-gw)  │ │  service)  │
     └────────────┘ └───────────┘ └────────────┘
```

## Quick Start

### 1. Start Vault

```bash
# Start all infrastructure including Vault
docker compose up -d vault

# Wait for Vault to be ready
docker logs -f aetherium-vault
```

### 2. Initialize Vault with Secrets

```bash
# Run initialization script
./scripts/init-vault.sh

# Or manually initialize
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=aetherium-dev-token

vault kv put secret/database/postgres \
    host=aetherium-postgres \
    user=aetherium \
    password=aetherium \
    database=aetherium
```

### 3. Use Vault in Your Service

```go
import "github.com/aetherium/aetherium/pkg/vault"

// Create Vault client
vaultClient, err := vault.NewSecretStore(&vault.Config{
    Address: "http://localhost:8200",
    Token:   os.Getenv("VAULT_TOKEN"),
})

// Get database credentials
host, user, password, database, err := vaultClient.GetDatabaseCredentials(ctx)

// Get integration token
githubToken, err := vaultClient.GetIntegrationToken(ctx, "github")
```

## Configuration

### Environment Variables

```bash
# Vault server address
VAULT_ADDR=http://localhost:8200

# Authentication token
VAULT_TOKEN=aetherium-dev-token

# Optional: custom KV mount
VAULT_MOUNT=secret
```

### Docker Compose

Vault is configured in `docker-compose.yml`:

```yaml
vault:
  image: hashicorp/vault:1.15
  container_name: aetherium-vault
  ports:
    - "8200:8200"
  environment:
    VAULT_ADDR: http://0.0.0.0:8200
    VAULT_DEV_ROOT_TOKEN_ID: aetherium-dev-token
  volumes:
    - ./config/vault-config.hcl:/vault/config/vault-config.hcl:ro
    - vault_data:/vault/data
```

**Dev Mode Features:**
- Auto-unsealed
- In-memory storage (data lost on restart)
- Root token: `aetherium-dev-token`
- UI enabled at http://localhost:8200

**Production Mode:**
- Uses Consul for storage (HA)
- Must be manually unsealed
- Uses real tokens with expiration
- TLS enabled

## Secret Structure

### Database Credentials

**Path:** `secret/database/postgres`

```json
{
  "host": "aetherium-postgres",
  "port": "5432",
  "user": "aetherium",
  "password": "secure_password_here",
  "database": "aetherium"
}
```

**Usage:**
```go
host, user, password, database, err := vaultClient.GetDatabaseCredentials(ctx)
```

### Redis Configuration

**Path:** `secret/redis/config`

```json
{
  "addr": "aetherium-redis:6379",
  "password": "redis_password"
}
```

**Usage:**
```go
addr, password, err := vaultClient.GetRedisCredentials(ctx)
```

### Integration Secrets

**GitHub Path:** `secret/integrations/github`

```json
{
  "token": "ghp_xxxxxxxxxxxxx",
  "webhook_secret": "webhook_secret_here",
  "app_id": "123456",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n..."
}
```

**Slack Path:** `secret/integrations/slack`

```json
{
  "bot_token": "xoxb-xxxxxxxxxxxxx",
  "signing_secret": "signing_secret_here",
  "app_token": "xapp-xxxxxxxxxxxxx"
}
```

**Usage:**
```go
githubToken, err := vaultClient.GetIntegrationToken(ctx, "github")
slackData, err := vaultClient.GetIntegrationSecret(ctx, "slack")
```

## Access Policies

### Service Policy (Full Access)

```hcl
path "secret/data/*" {
  capabilities = ["read", "list"]
}

path "secret/metadata/*" {
  capabilities = ["list"]
}
```

**Token Creation:**
```bash
vault token create -policy=aetherium-service -ttl=24h
```

### Worker Policy (Limited Access)

```hcl
# Can read database and Redis
path "secret/data/database/*" {
  capabilities = ["read"]
}

path "secret/data/redis/*" {
  capabilities = ["read"]
}

# Can list but not read integrations
path "secret/metadata/integrations/*" {
  capabilities = ["list"]
}
```

**Token Creation:**
```bash
vault token create -policy=aetherium-worker -ttl=24h
```

### API Gateway Policy

```hcl
# Full read access to all secrets
path "secret/data/*" {
  capabilities = ["read", "list"]
}
```

**Token Creation:**
```bash
vault token create -policy=aetherium-api-gateway -ttl=24h
```

## CLI Usage

### View Secrets

```bash
# Set environment
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=aetherium-dev-token

# Read a secret
vault kv get secret/database/postgres

# Read specific field
vault kv get -field=password secret/database/postgres

# List secrets
vault kv list secret/
vault kv list secret/integrations/
```

### Update Secrets

```bash
# Update existing secret
vault kv put secret/database/postgres password=new_password

# Add new secret
vault kv put secret/api/keys \
    api_key_1=key1 \
    api_key_2=key2

# Delete secret
vault kv delete secret/api/keys
```

### Manage Tokens

```bash
# Create token with policy
vault token create -policy=aetherium-service -ttl=24h

# Revoke token
vault token revoke <token>

# Lookup token info
vault token lookup <token>

# Renew token
vault token renew <token>
```

## Production Deployment

### 1. Enable TLS

Update `config/vault-config.hcl`:

```hcl
listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 0  # Enable TLS
  tls_cert_file = "/vault/tls/vault.crt"
  tls_key_file  = "/vault/tls/vault.key"
}
```

### 2. Use Consul Storage (HA)

Already configured in `vault-config.hcl`:

```hcl
storage "consul" {
  address = "consul:8500"
  path    = "vault/"
}
```

### 3. Initialize and Unseal

```bash
# Initialize Vault (first time only)
vault operator init -key-shares=5 -key-threshold=3

# Save unseal keys and root token securely!

# Unseal Vault (need 3 of 5 keys)
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>
```

### 4. Enable Auto-Unseal (AWS KMS)

Update `config/vault-config.hcl`:

```hcl
seal "awskms" {
  region     = "us-east-1"
  kms_key_id = "your-kms-key-id"
}
```

### 5. Setup Audit Logging

```bash
# Enable file audit
vault audit enable file file_path=/vault/logs/audit.log

# Enable syslog audit
vault audit enable syslog
```

### 6. Rotate Root Token

```bash
# Generate new root token
vault operator generate-root -init
vault operator generate-root -nonce=<nonce> <unseal-key>

# Revoke old root token
vault token revoke <old-root-token>
```

## Integration Examples

### Worker Service

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/aetherium/aetherium/pkg/vault"
    "github.com/aetherium/aetherium/pkg/storage/postgres"
)

func main() {
    ctx := context.Background()

    // Create Vault client
    vaultClient, err := vault.NewSecretStore(&vault.Config{
        Address: os.Getenv("VAULT_ADDR"),
        Token:   os.Getenv("VAULT_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Get database credentials from Vault
    host, user, password, database, err := vaultClient.GetDatabaseCredentials(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Connect to database
    db, err := postgres.Connect(host, user, password, database)
    if err != nil {
        log.Fatal(err)
    }

    // Get Redis credentials
    redisAddr, redisPassword, err := vaultClient.GetRedisCredentials(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Connect to Redis
    // ...
}
```

### API Gateway

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/aetherium/aetherium/pkg/vault"
)

func main() {
    ctx := context.Background()

    // Create Vault client
    vaultClient, err := vault.NewSecretStore(&vault.Config{
        Address: os.Getenv("VAULT_ADDR"),
        Token:   os.Getenv("VAULT_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Get GitHub token
    githubToken, err := vaultClient.GetIntegrationToken(ctx, "github")
    if err != nil {
        log.Fatal(err)
    }

    // Get Slack secrets
    slackSecrets, err := vaultClient.GetIntegrationSecret(ctx, "slack")
    if err != nil {
        log.Fatal(err)
    }

    botToken := slackSecrets["bot_token"].(string)
    signingSecret := slackSecrets["signing_secret"].(string)

    // Initialize integrations with secrets
    // ...
}
```

## Monitoring

### Health Check

```bash
# Check Vault status
curl http://localhost:8200/v1/sys/health

# Response:
{
  "initialized": true,
  "sealed": false,
  "standby": false,
  "performance_standby": false,
  "replication_performance_mode": "disabled",
  "replication_dr_mode": "disabled",
  "server_time_utc": 1699999999,
  "version": "1.15.0",
  "cluster_name": "vault-cluster-xxxxx",
  "cluster_id": "xxxxx-xxxxx-xxxxx"
}
```

### Metrics

Vault exposes Prometheus metrics at `/v1/sys/metrics`:

```bash
curl http://localhost:8200/v1/sys/metrics?format=prometheus
```

**Key Metrics:**
- `vault_core_unsealed` - Unseal status
- `vault_runtime_total_gc_runs` - GC runs
- `vault_expire_leases_by_expiration` - Lease expirations
- `vault_token_creation` - Token creations

### Audit Logs

```bash
# View audit logs
docker exec aetherium-vault cat /vault/logs/audit.log | tail -100

# Search for specific operations
docker exec aetherium-vault cat /vault/logs/audit.log | grep "secret/database"
```

## Troubleshooting

### Vault is Sealed

**Problem:** `vault status` shows sealed

**Solution:**
```bash
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>
```

### Permission Denied

**Problem:** Service can't read secrets

**Solution:** Check token policy
```bash
vault token lookup <token>
vault policy read aetherium-service
```

### Connection Refused

**Problem:** Can't connect to Vault

**Solution:**
```bash
# Check Vault is running
docker ps | grep vault

# Check network
curl http://localhost:8200/v1/sys/health

# Check firewall
netstat -tlnp | grep 8200
```

### Token Expired

**Problem:** `permission denied` after token expires

**Solution:**
```bash
# Renew token
vault token renew

# Or create new token
vault token create -policy=aetherium-service -ttl=24h
```

## Security Best Practices

1. **Use TLS in Production**
   - Never use `tls_disable = 1` in production
   - Use valid SSL certificates

2. **Rotate Tokens Regularly**
   - Set reasonable TTLs (24h for services)
   - Automate token renewal

3. **Use Minimal Policies**
   - Grant least privilege
   - Separate policies per service

4. **Enable Audit Logging**
   - Monitor all secret access
   - Alert on suspicious activity

5. **Backup Unseal Keys**
   - Store securely (not in repo!)
   - Use auto-unseal in production

6. **Regular Secret Rotation**
   - Rotate database passwords monthly
   - Rotate integration tokens quarterly

7. **Monitor Vault Health**
   - Alert on seal status
   - Monitor token usage

## Migration from Environment Variables

### Before (Environment Variables)

```yaml
# docker-compose.yml
environment:
  POSTGRES_PASSWORD: aetherium
  GITHUB_TOKEN: ghp_xxxxx
```

### After (Vault)

```yaml
# docker-compose.yml
environment:
  VAULT_ADDR: http://vault:8200
  VAULT_TOKEN: ${VAULT_SERVICE_TOKEN}
```

```go
// Code changes
// Before:
dbPassword := os.Getenv("POSTGRES_PASSWORD")

// After:
vaultClient, _ := vault.NewSecretStore(&vault.Config{...})
_, _, dbPassword, _, _ := vaultClient.GetDatabaseCredentials(ctx)
```

## References

- **Vault Documentation:** https://developer.hashicorp.com/vault/docs
- **KV Secrets Engine:** https://developer.hashicorp.com/vault/docs/secrets/kv
- **Policies:** https://developer.hashicorp.com/vault/docs/concepts/policies
- **Production Hardening:** https://developer.hashicorp.com/vault/tutorials/operations/production-hardening

## FAQ

**Q: Is Vault required for development?**
A: No, services fall back to environment variables if Vault is unavailable.

**Q: How do I add a new secret?**
A: Use `vault kv put secret/path key=value` or the `SetSecret` methods in Go.

**Q: Can I use Vault with Kubernetes?**
A: Yes, use Vault Agent for sidecar injection or CSI provider.

**Q: What happens if Vault is down?**
A: Services will fail to start. Implement retry logic and fallback to cached secrets.

**Q: How do I migrate existing secrets to Vault?**
A: Use `./scripts/init-vault.sh` or manually migrate with `vault kv put`.

---

**Next Steps:**
1. Start Vault: `docker compose up -d vault`
2. Initialize: `./scripts/init-vault.sh`
3. Update services to use Vault
4. Remove hardcoded secrets from configs
5. Enable TLS for production
