#!/bin/bash
# Setup network infrastructure for Aetherium VMs

set -e

echo "=== Setting up Aetherium network infrastructure ==="

# Bridge configuration
BRIDGE_NAME="aetherium0"
BRIDGE_IP="172.16.0.1/24"
SUBNET_CIDR="172.16.0.0/24"

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

echo ""
echo "=== Network setup complete ==="
echo "Bridge: $BRIDGE_NAME ($BRIDGE_IP)"
echo "NAT: $SUBNET_CIDR -> $DEFAULT_IFACE"
echo ""
echo "You can now start the worker to create VMs with network connectivity."
