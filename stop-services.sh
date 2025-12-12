#!/bin/bash

echo "🛑 Stopping Aetherium Services"
echo "=============================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Stop API Gateway
if [ -f /tmp/aetherium-api.pid ]; then
    API_PID=$(cat /tmp/aetherium-api.pid)
    echo -e "${YELLOW}Stopping API Gateway (PID: $API_PID)...${NC}"
    kill $API_PID 2>/dev/null || true
    rm /tmp/aetherium-api.pid
    echo -e "${GREEN}✓ API Gateway stopped${NC}"
else
    echo -e "${YELLOW}No API Gateway PID file found${NC}"
    pkill -f "/usr/local/bin/api-gateway" 2>/dev/null || true
    pkill -f "bin/api-gateway" 2>/dev/null || true
fi

# Stop Worker (requires sudo)
if [ -f /tmp/aetherium-worker.pid ]; then
    WORKER_PID=$(cat /tmp/aetherium-worker.pid)
    echo -e "${YELLOW}Stopping Worker (PID: $WORKER_PID)...${NC}"
    sudo kill $WORKER_PID 2>/dev/null || true
    rm /tmp/aetherium-worker.pid
    echo -e "${GREEN}✓ Worker stopped${NC}"
else
    echo -e "${YELLOW}No Worker PID file found${NC}"
    sudo pkill -f "/usr/local/bin/worker" 2>/dev/null || true
    sudo pkill -f "bin/worker" 2>/dev/null || true
fi

# Kill any remaining Firecracker VMs
echo -e "${YELLOW}Cleaning up Firecracker VMs...${NC}"
sudo pkill -9 firecracker 2>/dev/null || true
echo -e "${GREEN}✓ VMs cleaned up${NC}"

# Stop port-forward if running
pkill -f "kubectl port-forward" 2>/dev/null || true

# Clean up temp files
rm -f /tmp/start-worker.sh

echo -e "\n${GREEN}✓ All services stopped${NC}"
echo ""
echo "To start services again, run:"
echo "  ./start-services.sh"
