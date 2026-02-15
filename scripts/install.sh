#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/kidsmedia"
BINARY_NAME="kidsmedia"
MEDIA_DIR="/media"

echo "=== Kids Media Player Installer ==="

# Check root.
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Install system dependencies.
echo "Installing dependencies..."
apt-get update -qq
apt-get install -y -qq mpv cifs-utils nfs-common

# Create directories.
echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$MEDIA_DIR"
mkdir -p /mnt/nas

# Copy binary.
if [ -f "$BINARY_NAME" ]; then
    echo "Installing binary..."
    cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Warning: Binary not found. Build first with 'make build-pi'"
fi

# Copy config if not present.
if [ ! -f "$INSTALL_DIR/config.json" ]; then
    echo "Installing example config..."
    cp configs/config.example.json "$INSTALL_DIR/config.json"
    echo "IMPORTANT: Edit $INSTALL_DIR/config.json with your media mappings"
fi

# Install systemd service.
echo "Installing systemd service..."
cp configs/systemd/kidsmedia.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable kidsmedia.service

echo ""
echo "=== Installation complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit $INSTALL_DIR/config.json with your media and tag mappings"
echo "  2. Place media files in $MEDIA_DIR/"
echo "  3. Start the service: sudo systemctl start kidsmedia"
echo "  4. View logs: sudo journalctl -u kidsmedia -f"
echo ""
