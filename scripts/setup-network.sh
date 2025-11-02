#!/bin/bash
# Setup network infrastructure for Aetherium VMs

set -e

# Parse arguments
ENABLE_PROXY=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --with-proxy)
            ENABLE_PROXY=true
            shift
            ;;
        --help)
            echo "Usage: $0 [--with-proxy]"
            echo ""
            echo "Options:"
            echo "  --with-proxy    Enable transparent proxy redirect rules (requires Squid)"
            echo "  --help          Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run '$0 --help' for usage"
            exit 1
            ;;
    esac
done

echo "=== Setting up Aetherium network infrastructure ==="

# Bridge configuration
BRIDGE_NAME="aetherium0"
BRIDGE_IP="172.16.0.1/24"
SUBNET_CIDR="172.16.0.0/24"

# Proxy configuration
PROXY_HTTP_PORT=3128
PROXY_HTTPS_PORT=3129

# Detect default interface for NAT
DEFAULT_IFACE=$(ip route show default | awk '/default/ {print $5; exit}')
if [ -z "$DEFAULT_IFACE" ]; then
    echo "Error: Could not detect default network interface"
    exit 1
fi
echo "Detected default interface: $DEFAULT_IFACE"

# Create bridge if it doesn't exist
if ! ip link show $BRIDGE_NAME &>/dev/null; then
    echo "Creating bridge $BRIDGE_NAME..."
    ip link add $BRIDGE_NAME type bridge
    echo "✓ Bridge created"
else
    echo "✓ Bridge $BRIDGE_NAME already exists"
fi

# Set bridge IP
if ! ip addr show $BRIDGE_NAME | grep -q $BRIDGE_IP; then
    echo "Setting bridge IP to $BRIDGE_IP..."
    ip addr add $BRIDGE_IP dev $BRIDGE_NAME
    echo "✓ Bridge IP set"
else
    echo "✓ Bridge IP already configured"
fi

# Bring bridge up
echo "Bringing bridge up..."
ip link set $BRIDGE_NAME up
echo "✓ Bridge is up"

# Enable IP forwarding
echo "Enabling IP forwarding..."
sysctl -w net.ipv4.ip_forward=1 >/dev/null
echo "✓ IP forwarding enabled"

# Setup NAT
echo "Setting up NAT..."
if ! iptables -t nat -C POSTROUTING -s $SUBNET_CIDR -o $DEFAULT_IFACE -j MASQUERADE 2>/dev/null; then
    iptables -t nat -A POSTROUTING -s $SUBNET_CIDR -o $DEFAULT_IFACE -j MASQUERADE
    echo "✓ NAT rule added"
else
    echo "✓ NAT rule already exists"
fi

# Allow forwarding
if ! iptables -C FORWARD -i $BRIDGE_NAME -j ACCEPT 2>/dev/null; then
    iptables -A FORWARD -i $BRIDGE_NAME -j ACCEPT
    echo "✓ Forward rule added"
else
    echo "✓ Forward rule already exists"
fi

if ! iptables -C FORWARD -o $BRIDGE_NAME -j ACCEPT 2>/dev/null; then
    iptables -A FORWARD -o $BRIDGE_NAME -j ACCEPT
    echo "✓ Forward rule (outbound) added"
else
    echo "✓ Forward rule (outbound) already exists"
fi

# Setup transparent proxy redirect rules (optional)
if [ "$ENABLE_PROXY" = true ]; then
    echo ""
    echo "=== Setting up transparent proxy redirect rules ==="

    # HTTP redirect (port 80 -> 3128)
    echo "Setting up HTTP redirect (port 80 -> $PROXY_HTTP_PORT)..."
    if ! iptables -t nat -C PREROUTING -i $BRIDGE_NAME -p tcp --dport 80 -j REDIRECT --to-port $PROXY_HTTP_PORT 2>/dev/null; then
        iptables -t nat -A PREROUTING -i $BRIDGE_NAME -p tcp --dport 80 -j REDIRECT --to-port $PROXY_HTTP_PORT
        echo "✓ HTTP redirect rule added"
    else
        echo "✓ HTTP redirect rule already exists"
    fi

    # HTTPS redirect (port 443 -> 3129)
    echo "Setting up HTTPS redirect (port 443 -> $PROXY_HTTPS_PORT)..."
    if ! iptables -t nat -C PREROUTING -i $BRIDGE_NAME -p tcp --dport 443 -j REDIRECT --to-port $PROXY_HTTPS_PORT 2>/dev/null; then
        iptables -t nat -A PREROUTING -i $BRIDGE_NAME -p tcp --dport 443 -j REDIRECT --to-port $PROXY_HTTPS_PORT
        echo "✓ HTTPS redirect rule added"
    else
        echo "✓ HTTPS redirect rule already exists"
    fi

    echo ""
    echo "=== Transparent proxy setup complete ==="
    echo "HTTP traffic (port 80) -> $BRIDGE_IP:$PROXY_HTTP_PORT"
    echo "HTTPS traffic (port 443) -> $BRIDGE_IP:$PROXY_HTTPS_PORT"
    echo ""
    echo "Note: Ensure Squid proxy is running and configured for transparent interception."
    echo "Run: sudo ./scripts/setup-squid.sh"
    echo "Run: sudo ./scripts/generate-ssl-certs.sh"
fi

echo ""
echo "=== Network setup complete ==="
echo "Bridge: $BRIDGE_NAME ($BRIDGE_IP)"
echo "NAT: $SUBNET_CIDR -> $DEFAULT_IFACE"
if [ "$ENABLE_PROXY" = true ]; then
    echo "Proxy: Transparent interception enabled"
fi
echo ""
echo "You can now start the worker to create VMs with network connectivity."
