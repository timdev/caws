#!/bin/bash
set -e

echo "=== bw-aws Installation Script ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

# Check for bw
if ! command -v bw &> /dev/null; then
    echo "❌ Bitwarden CLI (bw) is not installed"
    echo "   Install from: https://bitwarden.com/help/cli/"
    echo ""
    echo "   Quick install:"
    echo "   - Ubuntu/Debian: sudo snap install bw"
    echo "   - macOS: brew install bitwarden-cli"
    exit 1
else
    echo "✅ Bitwarden CLI found: $(bw --version)"
fi

# Check for aws
if ! command -v aws &> /dev/null; then
    echo "⚠️  AWS CLI (aws) is not installed"
    echo "   The tool will work but you won't be able to get temporary credentials"
    echo "   Install from: https://aws.amazon.com/cli/"
    echo ""
else
    echo "✅ AWS CLI found: $(aws --version 2>&1 | head -1)"
fi

echo ""

# Build the binary
echo "Building bw-aws..."
go build -o bw-aws

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Cannot write to $INSTALL_DIR, will install to ~/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Install
echo "Installing to $INSTALL_DIR..."
mv bw-aws "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/bw-aws"

echo ""
echo "✅ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Login to Bitwarden:    bw-aws login"
echo "  2. Export session key:    export BW_SESSION=\"<key>\""
echo "  3. Add a profile:         bw-aws add <profile-name>"
echo "  4. Use it:                bw-aws exec <profile> -- aws s3 ls"
echo ""
echo "Run 'bw-aws help' for more information"
