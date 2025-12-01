#!/bin/bash
# Setup Squid proxy for Aetherium
# This script installs and configures Squid for transparent proxying

set -e

echo "=== Aetherium Squid Proxy Setup ==="
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Detect package manager
if command -v apt-get &> /dev/null; then
    PKG_MGR="apt-get"
    PKG_UPDATE="apt-get update"
    PKG_INSTALL="apt-get install -y"
elif command -v yum &> /dev/null; then
    PKG_MGR="yum"
    PKG_UPDATE="yum check-update || true"
    PKG_INSTALL="yum install -y"
elif command -v pacman &> /dev/null; then
    PKG_MGR="pacman"
    PKG_UPDATE="pacman -Sy"
    PKG_INSTALL="pacman -S --noconfirm"
else
    echo "Error: Unsupported package manager. Please install Squid manually."
    exit 1
fi

# Install Squid if not already installed
if ! command -v squid &> /dev/null; then
    echo "Installing Squid proxy..."
    $PKG_UPDATE
    $PKG_INSTALL squid
    echo "✓ Squid installed"
else
    echo "✓ Squid is already installed"
fi

# Create necessary directories
echo "Creating directories..."
mkdir -p /etc/squid/certs
mkdir -p /var/spool/squid-aetherium
mkdir -p /var/lib/squid
mkdir -p /var/log/squid

# Set proper permissions
chown -R proxy:proxy /var/spool/squid-aetherium 2>/dev/null || chown -R squid:squid /var/spool/squid-aetherium
chown -R proxy:proxy /var/lib/squid 2>/dev/null || chown -R squid:squid /var/lib/squid
chown -R proxy:proxy /var/log/squid 2>/dev/null || chown -R squid:squid /var/log/squid

echo "✓ Directories created with proper permissions"

# Initialize SSL certificate database (required for SSL bumping)
if [ ! -d "/var/lib/squid/ssl_db" ]; then
    echo "Initializing SSL certificate database..."
    /usr/lib/squid/security_file_certgen -c -s /var/lib/squid/ssl_db -M 4MB
    chown -R proxy:proxy /var/lib/squid/ssl_db 2>/dev/null || chown -R squid:squid /var/lib/squid/ssl_db
    echo "✓ SSL certificate database initialized"
else
    echo "✓ SSL certificate database already exists"
fi

# Initialize Squid cache directory
if [ ! -d "/var/spool/squid-aetherium/00" ]; then
    echo "Initializing Squid cache directory..."
    squid -z -f /etc/squid/aetherium.conf 2>/dev/null || true
    echo "✓ Cache directory initialized"
else
    echo "✓ Cache directory already initialized"
fi

# Enable and start Squid service (but don't fail if systemd isn't available)
if command -v systemctl &> /dev/null; then
    systemctl enable squid 2>/dev/null || true
    echo "✓ Squid service enabled"
else
    echo "⚠ systemd not available - Squid service not enabled"
fi

echo
echo "=== Squid Setup Complete ==="
echo
echo "Next steps:"
echo "1. Run ./scripts/generate-ssl-certs.sh to create CA certificate"
echo "2. Configure Aetherium to enable proxy in config.yaml"
echo "3. Start worker with proxy enabled"
echo

exit 0
