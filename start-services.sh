#!/bin/bash
set -e

echo "🚀 Starting Aetherium Services for WebSocket Testing"
echo "=================================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Stop existing processes
echo -e "\n${YELLOW}Step 1: Stopping existing processes...${NC}"
pkill -f "/usr/local/bin/worker" 2>/dev/null || true
pkill -f "/usr/local/bin/api-gateway" 2>/dev/null || true
pkill -f "kubectl port-forward" 2>/dev/null || true
pkill -9 firecracker 2>/dev/null || true
sleep 2
echo -e "${GREEN}✓ Processes stopped${NC}"

# Step 2: Set environment variables
echo -e "\n${YELLOW}Step 2: Setting environment variables...${NC}"
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5434
export POSTGRES_USER=aetherium
export POSTGRES_PASSWORD=aetherium
export POSTGRES_DB=aetherium
export REDIS_ADDR=localhost:6380
export FIRECRACKER_KERNEL=/var/firecracker/vmlinux
export FIRECRACKER_ROOTFS=/var/firecracker/rootfs-template.ext4
export FIRECRACKER_SOCKET_DIR=/tmp
export FIRECRACKER_DEFAULT_VCPU=1
export FIRECRACKER_DEFAULT_MEMORY_MB=512
export WORKSPACE_ENCRYPTION_KEY=$(openssl rand -hex 32)
export PORT=8080

echo -e "${GREEN}✓ Environment variables set${NC}"
echo "  - PostgreSQL: ${POSTGRES_HOST}:${POSTGRES_PORT}"
echo "  - Redis: ${REDIS_ADDR}"
echo "  - API Port: ${PORT}"

# Step 3: Verify containers are running
echo -e "\n${YELLOW}Step 3: Verifying containers...${NC}"
if ! docker ps | grep -q aetherium-postgres; then
    echo -e "${YELLOW}Starting PostgreSQL container...${NC}"
    docker start aetherium-postgres || echo -e "${RED}Failed to start PostgreSQL${NC}"
fi

if ! docker ps | grep -q aetherium-redis; then
    echo -e "${YELLOW}Starting Redis container...${NC}"
    docker start aetherium-redis || echo -e "${RED}Failed to start Redis${NC}"
fi

sleep 3
echo -e "${GREEN}✓ Containers running${NC}"

# Step 4: Check if migrations are needed
echo -e "\n${YELLOW}Step 4: Checking database migrations...${NC}"
MIGRATION_VERSION=$(docker exec aetherium-postgres psql -U aetherium -d aetherium -tAc "SELECT version FROM schema_migrations;" 2>/dev/null || echo "0")
echo "Current migration version: $MIGRATION_VERSION"

if [ "$MIGRATION_VERSION" -lt "3" ]; then
    echo -e "${YELLOW}Running migrations...${NC}"
    if command -v migrate &> /dev/null; then
        migrate -database "postgres://aetherium:aetherium@localhost:5434/aetherium?sslmode=disable" -path ./migrations up
        echo -e "${GREEN}✓ Migrations completed${NC}"
    else
        echo -e "${RED}⚠ migrate tool not found. Please run migrations manually${NC}"
        echo "  Install: https://github.com/golang-migrate/migrate"
    fi
else
    echo -e "${GREEN}✓ Migrations up to date${NC}"
fi

# Step 5: Verify Firecracker resources
echo -e "\n${YELLOW}Step 5: Verifying Firecracker resources...${NC}"
if [ ! -f "/var/firecracker/vmlinux" ]; then
    echo -e "${RED}⚠ Kernel not found: /var/firecracker/vmlinux${NC}"
    echo "  Run: sudo ./scripts/download-vsock-kernel.sh"
    exit 1
fi

if [ ! -f "/var/firecracker/rootfs-template.ext4" ]; then
    echo -e "${RED}⚠ Rootfs template not found: /var/firecracker/rootfs-template.ext4${NC}"
    echo "  Run: sudo ./scripts/prepare-rootfs-with-tools.sh"
    exit 1
fi

echo -e "${GREEN}✓ Firecracker resources available${NC}"
echo "  - Kernel: $(ls -lh /var/firecracker/vmlinux | awk '{print $5}')"
echo "  - Rootfs: $(ls -lh /var/firecracker/rootfs-template.ext4 | awk '{print $5}')"

# Step 6: Rebuild binaries
echo -e "\n${YELLOW}Step 6: Rebuilding binaries with WebSocket support...${NC}"
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/worker ./cmd/worker
go build -o bin/fc-agent ./cmd/fc-agent
echo -e "${GREEN}✓ Binaries built${NC}"

# Step 7: Start worker (requires sudo)
echo -e "\n${YELLOW}Step 7: Starting worker (requires sudo)...${NC}"
echo -e "${YELLOW}This will prompt for your password${NC}"

# Create a wrapper script that preserves environment variables
cat > /tmp/start-worker.sh <<'EOF'
#!/bin/bash
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5434
export POSTGRES_USER=aetherium
export POSTGRES_PASSWORD=aetherium
export POSTGRES_DB=aetherium
export REDIS_ADDR=localhost:6380
export FIRECRACKER_KERNEL=/var/firecracker/vmlinux
export FIRECRACKER_ROOTFS=/var/firecracker/rootfs-template.ext4
export FIRECRACKER_SOCKET_DIR=/tmp
export FIRECRACKER_DEFAULT_VCPU=1
export FIRECRACKER_DEFAULT_MEMORY_MB=512

cd /home/techsavvyash/sweatAndBlood/remote-agents/aetherium
./bin/worker > /tmp/aetherium-worker.log 2>&1 &
echo $! > /tmp/aetherium-worker.pid
echo "Worker started with PID: $(cat /tmp/aetherium-worker.pid)"
EOF

chmod +x /tmp/start-worker.sh
sudo /tmp/start-worker.sh

sleep 2
if [ -f /tmp/aetherium-worker.pid ]; then
    WORKER_PID=$(cat /tmp/aetherium-worker.pid)
    if ps -p $WORKER_PID > /dev/null; then
        echo -e "${GREEN}✓ Worker started (PID: $WORKER_PID)${NC}"
    else
        echo -e "${RED}✗ Worker failed to start${NC}"
        echo "Check logs: tail -f /tmp/aetherium-worker.log"
        exit 1
    fi
fi

# Step 8: Start API gateway
echo -e "\n${YELLOW}Step 8: Starting API gateway...${NC}"
./bin/api-gateway > /tmp/aetherium-api.log 2>&1 &
API_PID=$!
echo $API_PID > /tmp/aetherium-api.pid
sleep 3

if ps -p $API_PID > /dev/null; then
    echo -e "${GREEN}✓ API Gateway started (PID: $API_PID)${NC}"
else
    echo -e "${RED}✗ API Gateway failed to start${NC}"
    echo "Check logs: tail -f /tmp/aetherium-api.log"
    exit 1
fi

# Step 9: Verify health endpoint
echo -e "\n${YELLOW}Step 9: Verifying API health...${NC}"
sleep 2
HEALTH_CHECK=$(curl -s http://localhost:8080/api/v1/health 2>&1 || echo "failed")

if echo "$HEALTH_CHECK" | grep -q "ok"; then
    echo -e "${GREEN}✓ API Gateway is healthy${NC}"
    echo "$HEALTH_CHECK" | jq . 2>/dev/null || echo "$HEALTH_CHECK"
else
    echo -e "${RED}✗ API Gateway health check failed${NC}"
    echo "Response: $HEALTH_CHECK"
    echo -e "\n${YELLOW}API Gateway logs:${NC}"
    tail -20 /tmp/aetherium-api.log
    exit 1
fi

# Success!
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}✓ All services started successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Service Status:"
echo "  • Worker:      Running (PID: $(cat /tmp/aetherium-worker.pid))"
echo "  • API Gateway: Running (PID: $(cat /tmp/aetherium-api.pid))"
echo "  • PostgreSQL:  Running (container)"
echo "  • Redis:       Running (container)"
echo ""
echo "Endpoints:"
echo "  • API:       http://localhost:8080"
echo "  • Health:    http://localhost:8080/api/v1/health"
echo "  • Dashboard: http://localhost:3000"
echo ""
echo "Logs:"
echo "  • Worker: tail -f /tmp/aetherium-worker.log"
echo "  • API:    tail -f /tmp/aetherium-api.log"
echo ""
echo -e "${GREEN}Ready for WebSocket testing!${NC}"
echo "Open http://localhost:3000/workspaces to create a workspace"
