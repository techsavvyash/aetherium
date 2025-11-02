#!/bin/bash

set -e

echo "Starting Aetherium API Gateway..."

# Export environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5434
export POSTGRES_USER=aetherium
export POSTGRES_PASSWORD=aetherium
export POSTGRES_DB=aetherium
export REDIS_ADDR=localhost:6380
export CONSUL_ADDR=localhost:8500
export CONSUL_DATACENTER=dc1
export CONSUL_SCHEME=http
export CONSUL_SERVICE_NAME=aetherium-worker
export PORT=8080

# Optional: Loki logging
# export LOKI_URL=http://localhost:3100

# Start API Gateway
cd "$(dirname "$0")/.." || exit
./bin/api-gateway
