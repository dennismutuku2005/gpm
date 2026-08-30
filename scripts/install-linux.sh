#!/bin/bash
set -e

# Go Process Manager (GPM) Linux (Ubuntu) Installer

echo "=== GPM Linux (Ubuntu) Installation Start ==="

# Check OS target
if [ "$(uname -s)" != "Linux" ]; then
    echo "Error: GPM is supported exclusively on Linux (Ubuntu)."
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed on this machine. Go is required to compile GPM."
    exit 1
fi

# Build gpm
echo "Compiling GPM binary..."
go build -o gpm cmd/gpm/main.go

# Install binary
echo "Installing gpm to /usr/local/bin/gpm..."
sudo cp gpm /usr/local/bin/gpm
sudo chmod +x /usr/local/bin/gpm

# Create configuration directory
CONFIG_DIR="$HOME/.config/gpm"
echo "Creating GPM config directory at $CONFIG_DIR..."
mkdir -p "$CONFIG_DIR"

# Install systemd service
echo "Installing systemd service..."
sudo gpm startup

# Verify installation
echo "Verifying GPM installation..."
gpm version
gpm status

echo "=== GPM Linux (Ubuntu) Installation Completed Successfully! ==="
echo "You can now run 'gpm list' and 'gpm status' globally."

