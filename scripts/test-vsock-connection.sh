#!/bin/bash
# Test vsock connection manually

echo "Testing vsock connection to guest..."
echo ""

GUEST_CID=3
PORT=9999

if ! command -v socat &> /dev/null; then
    echo "Installing socat for vsock testing..."
    sudo apt-get update -qq
    sudo apt-get install -y socat
fi

echo "Attempting to connect to vsock CID=$GUEST_CID port=$PORT..."
echo ""

# Try to connect with timeout
timeout 5 socat - VSOCK-CONNECT:$GUEST_CID:$PORT <<< '{"cmd":"echo","args":["test"]}' 2>&1

result=$?

echo ""
if [ $result -eq 0 ]; then
    echo "✓ Connection successful!"
elif [ $result -eq 124 ]; then
    echo "✗ Connection timeout - agent not responding"
elif [ $result -eq 1 ]; then
    echo "✗ Connection refused - agent not listening or vsock not available in guest"
else
    echo "✗ Connection failed with code: $result"
fi

echo ""
echo "Checking if vsock socket exists..."
ls -l /tmp/firecracker-test-vm.sock.vsock 2>/dev/null || echo "Socket file not found"

echo ""
echo "Checking host vsock connections..."
ss -xa 2>/dev/null | grep vsock || echo "No vsock connections"
