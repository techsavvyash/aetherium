#!/bin/bash
# Script to start Squid proxy for VM whitelist control

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Starting Squid Proxy for Aetherium ==="_
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Error: This script must be run as root"
    echo "Usage: sudo $0"
    exit 1
fi

# Check if Squid is installed
if ! command -v squid &> /dev/null; then
    echo "Squid is not installed. Installing..."
    apt-get update -qq
    apt-get install -y squid
fi

echo "✓ Squid is installed"

# Create directories if they don't exist
mkdir -p /var/log/squid
mkdir -p /var/spool/squid
mkdir -p /etc/squid

# Copy configuration files
echo "Copying Squid configuration..."
cp "$PROJECT_ROOT/config/squid.conf" /etc/squid/squid.conf
cp "$PROJECT_ROOT/config/whitelist.txt" /etc/squid/whitelist.txt

echo "✓ Configuration files copied"

# Initialize Squid cache if needed
if [ ! -d "/var/spool/squid/00" ]; then
    echo "Initializing Squid cache..."
    squid -z
    echo "✓ Cache initialized"
fi

# Check if Squid is already running
if pgrep -x "squid" > /dev/null; then
    echo "Squid is already running, reloading configuration..."
    squid -k reconfigure
else
    echo "Starting Squid proxy..."
    squid
fi

# Wait for Squid to start
sleep 2

# Check if Squid is running
if pgrep -x "squid" > /dev/null; then
    echo "✓ Squid proxy is running on port 3128"
    echo ""
    echo "VM Proxy Configuration:"
    echo "  HTTP_PROXY=http://172.16.0.1:3128"
    echo "  HTTPS_PROXY=http://172.16.0.1:3128"
    echo ""
    echo "To test the proxy:"
    echo "  curl -x http://127.0.0.1:3128 https://github.com"
    echo ""
    echo "To view logs:"
    echo "  tail -f /var/log/squid/access.log"
else
    echo "❌ Failed to start Squid proxy"
    echo "Check logs: /var/log/squid/cache.log"
    exit 1
fi
