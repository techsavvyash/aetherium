#!/bin/bash
# Generate SSL CA certificate for Squid HTTPS interception
# This creates a self-signed CA that Squid uses to generate certificates on-the-fly

set -e

echo "=== Aetherium SSL Certificate Generation ==="
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

CERT_DIR="/etc/squid/certs"
CA_CERT="$CERT_DIR/aetherium-ca.pem"
CA_KEY="$CERT_DIR/aetherium-ca-key.pem"
CA_DER="$CERT_DIR/aetherium-ca.der"

# Create cert directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Check if certificate already exists
if [ -f "$CA_CERT" ] && [ -f "$CA_KEY" ]; then
    echo "⚠ SSL certificate already exists at $CA_CERT"
    read -p "Do you want to regenerate it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping certificate generation"
        exit 0
    fi
    echo "Regenerating certificate..."
fi

# Generate CA private key
echo "Generating CA private key..."
openssl genrsa -out "$CA_KEY" 4096

# Generate CA certificate
echo "Generating CA certificate..."
openssl req -new -x509 -days 3650 -key "$CA_KEY" -out "$CA_CERT" \
    -subj "/C=US/ST=Cloud/L=Internet/O=Aetherium/OU=Proxy/CN=Aetherium Root CA"

# Convert to DER format (for easier installation in VMs)
echo "Converting to DER format..."
openssl x509 -in "$CA_CERT" -outform DER -out "$CA_DER"

# Set proper permissions
chmod 600 "$CA_KEY"
chmod 644 "$CA_CERT"
chmod 644 "$CA_DER"
chown proxy:proxy "$CERT_DIR"/* 2>/dev/null || chown squid:squid "$CERT_DIR"/* || true

echo "✓ SSL certificates generated successfully"
echo
echo "Certificate files:"
echo "  CA Certificate (PEM): $CA_CERT"
echo "  CA Private Key:       $CA_KEY"
echo "  CA Certificate (DER): $CA_DER"
echo
echo "Certificate details:"
openssl x509 -in "$CA_CERT" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After)"
echo
echo "=== Important: Installing CA in VMs ==="
echo
echo "For VMs to trust HTTPS connections through the proxy, you need to install"
echo "the CA certificate in the VM rootfs template:"
echo
echo "1. Copy CA certificate to rootfs:"
echo "   sudo mkdir -p /var/firecracker/rootfs-mount/usr/local/share/ca-certificates/"
echo "   sudo cp $CA_DER /var/firecracker/rootfs-mount/usr/local/share/ca-certificates/aetherium-ca.crt"
echo
echo "2. Update CA certificates in rootfs (after mounting):"
echo "   sudo chroot /var/firecracker/rootfs-mount /usr/sbin/update-ca-certificates"
echo
echo "Or use the automated script (if available):"
echo "   sudo ./scripts/install-ca-in-rootfs.sh"
echo

exit 0
