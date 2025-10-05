#!/bin/bash
set -e

echo "=== Aetherium Integration Test Setup ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: Integration tests require root access for Firecracker"
  echo "Please run with sudo: sudo ./scripts/run-integration-test.sh"
  exit 1
fi

echo ""
echo "Step 1: Checking prerequisites..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed"
    exit 1
fi

# Check if Firecracker is installed
if ! command -v firecracker &> /dev/null; then
    echo "Error: Firecracker is not installed"
    exit 1
fi

# Check if kernel exists
if [ ! -f "/var/firecracker/vmlinux" ]; then
    echo "Error: Kernel not found at /var/firecracker/vmlinux"
    exit 1
fi

# Check if rootfs exists
if [ ! -f "/var/firecracker/rootfs.ext4" ]; then
    echo "Error: Rootfs not found at /var/firecracker/rootfs.ext4"
    exit 1
fi

# Check if vhost_vsock module is loaded
if ! lsmod | grep -q vhost_vsock; then
    echo "Loading vhost_vsock module..."
    modprobe vhost_vsock
fi

echo "✓ All prerequisites met"

echo ""
echo "Step 2: Starting infrastructure..."

# Start PostgreSQL if not running
if ! docker ps | grep -q aetherium-postgres; then
    echo "Starting PostgreSQL..."
    docker run -d \
      --name aetherium-postgres \
      -e POSTGRES_USER=aetherium \
      -e POSTGRES_PASSWORD=aetherium \
      -e POSTGRES_DB=aetherium \
      -p 5432:5432 \
      postgres:15-alpine

    # Wait for PostgreSQL to be ready
    echo "Waiting for PostgreSQL to be ready..."
    sleep 5
else
    echo "✓ PostgreSQL already running"
fi

# Start Redis if not running
if ! docker ps | grep -q aetherium-redis; then
    echo "Starting Redis..."
    docker run -d \
      --name aetherium-redis \
      -p 6379:6379 \
      redis:7-alpine

    sleep 2
else
    echo "✓ Redis already running"
fi

echo "✓ Infrastructure running"

echo ""
echo "Step 3: Running database migrations..."

cd "$(dirname "$0")/.."

# Check if migrations have been run
if ! docker exec aetherium-postgres psql -U aetherium -d aetherium -c '\dt' 2>/dev/null | grep -q vms; then
    echo "Running migrations..."

    # Install migrate if not present
    if ! command -v migrate &> /dev/null; then
        echo "Error: migrate tool not found. Install it with:"
        echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        exit 1
    fi

    migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" \
            -path ./migrations up

    echo "✓ Migrations applied"
else
    echo "✓ Migrations already applied"
fi

echo ""
echo "Step 4: Building test binary..."
cd tests/integration
go test -c -o /tmp/aetherium_integration_test

echo ""
echo "Step 5: Running integration test..."
echo ""
/tmp/aetherium_integration_test -test.v -test.run TestThreeVMClone -test.timeout 30m

echo ""
echo "=== Test completed ==="
echo ""
echo "To view results in database:"
echo "  docker exec -it aetherium-postgres psql -U aetherium -c 'SELECT id, name, status FROM vms;'"
echo "  docker exec -it aetherium-postgres psql -U aetherium -c 'SELECT vm_id, command, exit_code FROM executions ORDER BY started_at DESC LIMIT 20;'"
echo ""
echo "To clean up infrastructure:"
echo "  docker stop aetherium-postgres aetherium-redis"
echo "  docker rm aetherium-postgres aetherium-redis"
