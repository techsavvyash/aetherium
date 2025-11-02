#!/bin/bash
# Setup NAT for Aetherium VMs

set -e

echo "Setting up NAT for VM network..."

# Add MASQUERADE rule for VM traffic
iptables -t nat -C POSTROUTING -s 172.16.0.0/24 -o enp0s20f0u4u4 -j MASQUERADE 2>/dev/null || \
    iptables -t nat -A POSTROUTING -s 172.16.0.0/24 -o enp0s20f0u4u4 -j MASQUERADE

# Allow forwarding from bridge to external interface
iptables -C FORWARD -i aetherium0 -o enp0s20f0u4u4 -j ACCEPT 2>/dev/null || \
    iptables -A FORWARD -i aetherium0 -o enp0s20f0u4u4 -j ACCEPT

# Allow return traffic
iptables -C FORWARD -i enp0s20f0u4u4 -o aetherium0 -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
    iptables -A FORWARD -i enp0s20f0u4u4 -o aetherium0 -m state --state RELATED,ESTABLISHED -j ACCEPT

echo "✓ NAT configured successfully"
echo "VM subnet: 172.16.0.0/24"
echo "External interface: enp0s20f0u4u4"
echo ""
echo "Verifying NAT rules:"
iptables -t nat -L POSTROUTING -n -v | grep 172.16.0
