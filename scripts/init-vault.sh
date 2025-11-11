#!/bin/bash
# Initialize HashiCorp Vault with Aetherium secrets
# This script should be run after Vault is started for the first time

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Vault configuration
VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-aetherium-dev-token}"

echo "=== Initializing Vault for Aetherium ==="
echo "Vault Address: $VAULT_ADDR"
echo ""

# Export for vault CLI
export VAULT_ADDR
export VAULT_TOKEN

# Wait for Vault to be ready
echo "Waiting for Vault to be ready..."
for i in {1..30}; do
    if vault status >/dev/null 2>&1; then
        echo "✓ Vault is ready"
        break
    fi
    echo "  Attempt $i/30..."
    sleep 2
done

# Check if Vault is sealed
if ! vault status >/dev/null 2>&1; then
    echo "❌ Vault is not accessible or sealed"
    echo "Run: vault operator unseal"
    exit 1
fi

echo ""
echo "=== Enabling KV v2 Secrets Engine ==="

# Enable KV v2 secrets engine if not already enabled
if ! vault secrets list | grep -q "^secret/"; then
    vault secrets enable -version=2 -path=secret kv
    echo "✓ KV v2 engine enabled at secret/"
else
    echo "✓ KV v2 engine already enabled"
fi

echo ""
echo "=== Storing Database Credentials ==="

# Store PostgreSQL credentials
vault kv put secret/database/postgres \
    host="aetherium-postgres" \
    port="5432" \
    user="aetherium" \
    password="aetherium" \
    database="aetherium"

echo "✓ PostgreSQL credentials stored"

echo ""
echo "=== Storing Redis Credentials ==="

# Store Redis credentials
vault kv put secret/redis/config \
    addr="aetherium-redis:6379" \
    password=""

echo "✓ Redis credentials stored"

echo ""
echo "=== Storing Integration Secrets ==="

# GitHub integration (example - update with real values)
vault kv put secret/integrations/github \
    token="${GITHUB_TOKEN:-ghp_example_token_replace_me}" \
    webhook_secret="${GITHUB_WEBHOOK_SECRET:-example_secret}"

echo "✓ GitHub integration secrets stored"

# Slack integration (example - update with real values)
vault kv put secret/integrations/slack \
    bot_token="${SLACK_BOT_TOKEN:-xoxb-example-token}" \
    signing_secret="${SLACK_SIGNING_SECRET:-example_secret}"

echo "✓ Slack integration secrets stored"

echo ""
echo "=== Creating Vault Policies ==="

# Create policy for aetherium services
vault policy write aetherium-service - <<EOF
# Allow reading all secrets under secret/
path "secret/data/*" {
  capabilities = ["read", "list"]
}

# Allow reading database credentials
path "secret/data/database/*" {
  capabilities = ["read"]
}

# Allow reading Redis credentials
path "secret/data/redis/*" {
  capabilities = ["read"]
}

# Allow reading integration secrets
path "secret/data/integrations/*" {
  capabilities = ["read"]
}

# Allow listing secrets
path "secret/metadata/*" {
  capabilities = ["list"]
}
EOF

echo "✓ Aetherium service policy created"

# Create policy for workers
vault policy write aetherium-worker - <<EOF
# Workers can read database and redis credentials
path "secret/data/database/*" {
  capabilities = ["read"]
}

path "secret/data/redis/*" {
  capabilities = ["read"]
}

# Workers can list but not read integration secrets
path "secret/metadata/integrations/*" {
  capabilities = ["list"]
}
EOF

echo "✓ Worker policy created"

# Create policy for API gateway
vault policy write aetherium-api-gateway - <<EOF
# API Gateway needs full access to secrets
path "secret/data/*" {
  capabilities = ["read", "list"]
}

path "secret/metadata/*" {
  capabilities = ["list"]
}
EOF

echo "✓ API Gateway policy created"

echo ""
echo "=== Creating Service Tokens ==="

# Create token for services (24 hour TTL)
SERVICE_TOKEN=$(vault token create \
    -policy=aetherium-service \
    -ttl=24h \
    -format=json | jq -r '.auth.client_token')

# Create token for workers
WORKER_TOKEN=$(vault token create \
    -policy=aetherium-worker \
    -ttl=24h \
    -format=json | jq -r '.auth.client_token')

# Create token for API gateway
API_TOKEN=$(vault token create \
    -policy=aetherium-api-gateway \
    -ttl=24h \
    -format=json | jq -r '.auth.client_token')

echo "✓ Service tokens created"

echo ""
echo "=== Vault Initialization Complete ==="
echo ""
echo "Service Token:     $SERVICE_TOKEN"
echo "Worker Token:      $WORKER_TOKEN"
echo "API Gateway Token: $API_TOKEN"
echo ""
echo "Store these tokens securely and use them to configure services:"
echo "  export VAULT_TOKEN=<token>"
echo ""
echo "View secrets:"
echo "  vault kv get secret/database/postgres"
echo "  vault kv get secret/redis/config"
echo "  vault kv get secret/integrations/github"
echo ""
echo "Access Vault UI:"
echo "  http://localhost:8200"
echo "  Token: aetherium-dev-token (dev mode)"
