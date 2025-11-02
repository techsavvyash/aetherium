#!/bin/bash

set -e

echo "==================================="
echo "Aetherium Quick Start"
echo "==================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if running from correct directory
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}Error: Must run from aetherium root directory${NC}"
    exit 1
fi

# Step 1: Start Docker services
echo -e "${YELLOW}Step 1: Starting Docker services...${NC}"
docker-compose up -d

echo "Waiting for services to be healthy..."
sleep 10

# Check service health
echo "Checking PostgreSQL..."
until docker exec aetherium-postgres pg_isready -U aetherium > /dev/null 2>&1; do
    echo "  Waiting for PostgreSQL..."
    sleep 2
done
echo -e "${GREEN}✓ PostgreSQL ready${NC}"

echo "Checking Redis..."
until docker exec aetherium-redis redis-cli ping > /dev/null 2>&1; do
    echo "  Waiting for Redis..."
    sleep 2
done
echo -e "${GREEN}✓ Redis ready${NC}"

echo "Checking Consul..."
until docker exec aetherium-consul consul members > /dev/null 2>&1; do
    echo "  Waiting for Consul..."
    sleep 2
done
echo -e "${GREEN}✓ Consul ready${NC}"

echo ""

# Step 2: Run database migrations
echo -e "${YELLOW}Step 2: Running database migrations...${NC}"

if ! command -v migrate &> /dev/null; then
    echo -e "${RED}Error: golang-migrate not installed${NC}"
    echo "Install with: brew install golang-migrate (macOS) or go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo ""
    echo "Skipping migrations - you'll need to run them manually:"
    echo "  cd migrations"
    echo "  migrate -database \"postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable\" -path . up"
else
    cd migrations
    migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" \
        -path . up || echo -e "${YELLOW}Migrations may have already been applied${NC}"
    cd ..
    echo -e "${GREEN}✓ Database migrations complete${NC}"
fi

echo ""

# Step 3: Build binaries
echo -e "${YELLOW}Step 3: Building binaries...${NC}"

if [ ! -f "bin/api-gateway" ] || [ ! -f "bin/worker" ]; then
    echo "Building Go binaries..."
    make build || (echo -e "${RED}Build failed. Try: go build -o bin/api-gateway ./cmd/api-gateway && go build -o bin/worker ./cmd/worker${NC}" && exit 1)
    echo -e "${GREEN}✓ Binaries built${NC}"
else
    echo -e "${GREEN}✓ Binaries already exist${NC}"
fi

echo ""

# Step 4: Check Firecracker setup
echo -e "${YELLOW}Step 4: Checking Firecracker setup...${NC}"

if [ ! -f "/var/firecracker/vmlinux" ]; then
    echo -e "${YELLOW}Warning: Kernel not found at /var/firecracker/vmlinux${NC}"
    echo "Run: sudo ./scripts/download-vsock-kernel.sh"
else
    echo -e "${GREEN}✓ Kernel found${NC}"
fi

if [ ! -f "/var/firecracker/rootfs.ext4" ]; then
    echo -e "${YELLOW}Warning: Rootfs not found at /var/firecracker/rootfs.ext4${NC}"
    echo "Run: sudo ./scripts/prepare-rootfs-with-tools.sh"
else
    echo -e "${GREEN}✓ Rootfs found${NC}"
fi

echo ""

# Step 5: Display next steps
echo -e "${GREEN}==================================="
echo "✓ Infrastructure Ready!"
echo "===================================${NC}"
echo ""
echo "Services running:"
echo "  - PostgreSQL:  localhost:5432"
echo "  - Redis:       localhost:6379"
echo "  - Consul UI:   http://localhost:8500"
echo "  - Grafana:     http://localhost:3000"
echo "  - Loki:        http://localhost:3100"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo "1. Start API Gateway (Terminal 1):"
echo "   ${GREEN}export POSTGRES_HOST=localhost"
echo "   export POSTGRES_USER=aetherium"
echo "   export POSTGRES_PASSWORD=aetherium"
echo "   export POSTGRES_DB=aetherium"
echo "   export REDIS_ADDR=localhost:6379"
echo "   export CONSUL_ADDR=localhost:8500"
echo "   export PORT=8080"
echo "   ./bin/api-gateway${NC}"
echo ""
echo "2. Start Worker (Terminal 2):"
echo "   ${GREEN}export POSTGRES_HOST=localhost"
echo "   export POSTGRES_USER=aetherium"
echo "   export POSTGRES_PASSWORD=aetherium"
echo "   export POSTGRES_DB=aetherium"
echo "   export REDIS_ADDR=localhost:6379"
echo "   export CONSUL_ADDR=localhost:8500"
echo "   export WORKER_ID=worker-01"
echo "   export WORKER_ZONE=us-west-1a"
echo "   export WORKER_ADDRESS=localhost:8081"
echo "   export WORKER_LABELS=env=dev,tier=compute"
echo "   sudo -E ./bin/worker${NC}"
echo ""
echo "3. Open Web UI:"
echo "   ${GREEN}http://localhost:8080/ui${NC}"
echo ""
echo "4. Or use helper script:"
echo "   ${GREEN}./scripts/start-api-gateway.sh    # Terminal 1"
echo "   sudo ./scripts/start-worker.sh       # Terminal 2${NC}"
echo ""
echo "To stop infrastructure:"
echo "   ${RED}docker-compose down${NC}"
echo ""
