#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$SCRIPT_DIR/.."

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  Aetherium Distributed Task Processing Demo          ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

echo "${BLUE}Step 1: Starting infrastructure...${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Start PostgreSQL
if ! docker ps | grep -q aetherium-postgres; then
    echo "Starting PostgreSQL..."
    docker run -d --rm \
        --name aetherium-postgres \
        -p 5432:5432 \
        -e POSTGRES_USER=aetherium \
        -e POSTGRES_PASSWORD=aetherium \
        -e POSTGRES_DB=aetherium \
        postgres:15 > /dev/null
    echo "✓ PostgreSQL started on port 5432"
    sleep 3 # Wait for PostgreSQL to be ready
else
    echo "✓ PostgreSQL already running"
fi

# Start Redis
if ! docker ps | grep -q aetherium-redis; then
    echo "Starting Redis..."
    docker run -d --rm \
        --name aetherium-redis \
        -p 6379:6379 \
        redis:7 > /dev/null
    echo "✓ Redis started on port 6379"
else
    echo "✓ Redis already running"
fi

echo ""
echo "${BLUE}Step 2: Running database migrations...${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cd "$PROJECT_DIR"

# Build migration tool
go build -o bin/migrate ./cmd/migrate

# Run migrations
./bin/migrate -config config/example.yaml -migrations migrations

echo ""
echo "${BLUE}Step 3: Building demo binaries...${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Build worker
go build -o bin/worker-demo ./cmd/worker-demo
echo "✓ Worker built: bin/worker-demo"

# Build task submitter
go build -o bin/task-submit ./cmd/task-submit
echo "✓ Task submitter built: bin/task-submit"

# Build agent
go build -o bin/fc-agent ./cmd/fc-agent
echo "✓ FC agent built: bin/fc-agent"

echo ""
echo "${BLUE}Step 4: Setting up Firecracker environment...${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Ensure vsock is set up
if ! lsmod | grep -q vhost_vsock; then
    echo "Loading vhost-vsock module (requires sudo)..."
    sudo modprobe vhost-vsock
fi

# Set permissions
if [ -c /dev/vhost-vsock ]; then
    sudo chmod 666 /dev/vhost-vsock
    echo "✓ Vsock configured"
fi

# Deploy agent to rootfs if needed
if [ -f "/var/firecracker/rootfs.ext4" ]; then
    echo "✓ Rootfs found at /var/firecracker/rootfs.ext4"

    # Check if agent is deployed
    NEEDS_DEPLOY=false
    if sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt 2>/dev/null; then
        if [ ! -f /mnt/usr/local/bin/fc-agent ]; then
            NEEDS_DEPLOY=true
        fi
        sudo umount /mnt
    fi

    if [ "$NEEDS_DEPLOY" = true ]; then
        echo "Deploying agent to rootfs (requires sudo)..."
        sudo ./scripts/setup-and-test.sh || true
    else
        echo "✓ Agent already deployed"
    fi
else
    echo "${YELLOW}⚠ Warning: /var/firecracker/rootfs.ext4 not found${NC}"
    echo "  You may need to set up Firecracker first"
fi

echo ""
echo "${GREEN}╔═══════════════════════════════════════════════════════╗${NC}"
echo "${GREEN}║  ✓ Demo Environment Ready!                            ║${NC}"
echo "${GREEN}╚═══════════════════════════════════════════════════════╝${NC}"
echo ""

echo "To start the worker in a new terminal:"
echo "  ${YELLOW}./bin/worker-demo${NC}"
echo ""
echo "To submit tasks:"
echo "  ${YELLOW}# Create a VM${NC}"
echo "  ./bin/task-submit -type vm:create -name demo-vm"
echo ""
echo "  ${YELLOW}# Execute command (replace <vm-id> with actual ID from creation)${NC}"
echo "  ./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd echo -args 'Hello from VM'"
echo ""
echo "  ${YELLOW}# Clone a GitHub repo${NC}"
echo "  ./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd git -args 'clone https://github.com/torvalds/linux'"
echo ""
echo "  ${YELLOW}# Delete VM${NC}"
echo "  ./bin/task-submit -type vm:delete -vm-id <vm-id>"
echo ""

echo "To stop infrastructure:"
echo "  docker stop aetherium-postgres aetherium-redis"
