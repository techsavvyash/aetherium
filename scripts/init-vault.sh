#!/bin/bash
set -e

echo "Initializing Vault for Aetherium..."

# Wait for Vault to be ready
until vault status &>/dev/null; do
    echo "Waiting for Vault to be ready..."
    sleep 2
done

echo "✓ Vault is ready"

# Enable transit secrets engine for encryption
if ! vault secrets list | grep -q "transit/"; then
    echo "Enabling transit secrets engine..."
    vault secrets enable transit
    echo "✓ Transit engine enabled"
else
    echo "✓ Transit engine already enabled"
fi

# Create encryption key for aetherium secrets
if ! vault list transit/keys 2>/dev/null | grep -q "aetherium"; then
    echo "Creating encryption key 'aetherium'..."
    vault write -f transit/keys/aetherium \
        type=aes256-gcm96 \
        derived=false \
        exportable=false
    echo "✓ Encryption key created"
else
    echo "✓ Encryption key already exists"
fi

# Enable KV v2 secrets engine for storing metadata
if ! vault secrets list | grep -q "kv/"; then
    echo "Enabling KV v2 secrets engine..."
    vault secrets enable -version=2 kv
    echo "✓ KV v2 engine enabled"
else
    echo "✓ KV v2 engine already enabled"
fi

# Create policy for aetherium application
echo "Creating Aetherium application policy..."
vault policy write aetherium-app - <<EOF
# Allow encryption and decryption using the aetherium key
path "transit/encrypt/aetherium" {
  capabilities = ["create", "update"]
}

path "transit/decrypt/aetherium" {
  capabilities = ["create", "update"]
}

# Allow key rotation
path "transit/keys/aetherium/rotate" {
  capabilities = ["update"]
}

# Allow reading key info (not the key itself)
path "transit/keys/aetherium" {
  capabilities = ["read"]
}

# Allow reading/writing secrets metadata in KV
path "kv/data/aetherium/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "kv/metadata/aetherium/*" {
  capabilities = ["list", "read", "delete"]
}
EOF

echo "✓ Policy created"

# Create a token for the aetherium application
APP_TOKEN=$(vault token create \
    -policy=aetherium-app \
    -ttl=0 \
    -renewable=true \
    -format=json | jq -r '.auth.client_token')

echo ""
echo "========================================"
echo "Vault Initialization Complete!"
echo "========================================"
echo ""
echo "Configuration for Aetherium:"
echo "  VAULT_ADDR=http://localhost:8200"
echo "  VAULT_TOKEN=$APP_TOKEN"
echo ""
echo "Root token (for admin): aetherium-root-token"
echo ""
echo "Save the application token to your environment or config!"
echo ""
