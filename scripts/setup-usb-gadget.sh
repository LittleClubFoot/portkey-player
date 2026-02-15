#!/bin/bash
set -euo pipefail

# Setup USB gadget mode for Raspberry Pi Zero 2 W.
# This allows the Pi to present itself as a USB mass storage device
# when connected to a computer, exposing the media folder for management.

echo "=== USB Gadget Mode Setup ==="

if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

CONFIG_TXT="/boot/config.txt"
CMDLINE_TXT="/boot/cmdline.txt"

# Enable dwc2 overlay in config.txt.
if ! grep -q "dtoverlay=dwc2" "$CONFIG_TXT"; then
    echo "Adding dwc2 overlay to $CONFIG_TXT..."
    echo "dtoverlay=dwc2" >> "$CONFIG_TXT"
else
    echo "dwc2 overlay already configured."
fi

# Add dwc2 module to boot command line.
if ! grep -q "modules-load=dwc2" "$CMDLINE_TXT"; then
    echo "Adding dwc2 module load to $CMDLINE_TXT..."
    sed -i 's/$/ modules-load=dwc2/' "$CMDLINE_TXT"
else
    echo "dwc2 module already in cmdline."
fi

# Ensure dwc2 loads at boot.
if ! grep -q "dwc2" /etc/modules; then
    echo "dwc2" >> /etc/modules
fi

echo ""
echo "=== USB Gadget setup complete ==="
echo "Reboot required for changes to take effect."
echo "After reboot, the kidsmedia service will auto-detect USB power mode."
echo ""
