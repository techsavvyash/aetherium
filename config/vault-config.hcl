# HashiCorp Vault Configuration for Aetherium
# Supports both development and production modes

# Storage Backend - Use Consul for HA and consistency
storage "consul" {
  address = "consul:8500"
  path    = "vault/"
}

# Alternative: File storage for development/testing
# storage "file" {
#   path = "/vault/data"
# }

# HTTP Listener
listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1  # For development - ENABLE TLS in production!
}

# UI
ui = true

# API address
api_addr = "http://0.0.0.0:8200"

# Cluster address (for HA setup)
cluster_addr = "http://0.0.0.0:8201"

# Telemetry
telemetry {
  prometheus_retention_time = "30s"
  disable_hostname = true
}

# Seal Configuration (auto-unseal for production)
# seal "awskms" {
#   region     = "us-east-1"
#   kms_key_id = "your-kms-key-id"
# }

# Log level
log_level = "Info"

# Disable mlock for containerized environments
disable_mlock = true
