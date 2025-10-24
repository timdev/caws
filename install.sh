#!/bin/bash
set -e

echo "=== caws Installation Script ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

# Check for aws
if ! command -v aws &> /dev/null; then
    echo "⚠️  AWS CLI (aws) is not installed"
    echo "   The tool will build but you won't be able to get temporary credentials"
    echo "   Install from: https://aws.amazon.com/cli/"
    echo ""
    echo "   Quick install:"
    echo "   - macOS: brew install awscli"
    echo "   - Ubuntu/Debian: sudo apt install awscli"
    echo ""
else
    echo "✅ AWS CLI found: $(aws --version 2>&1 | head -1)"
fi

# Check for go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed (required for building)"
    echo "   Install from: https://golang.org/dl/"
    exit 1
else
    echo "✅ Go found: $(go version)"
fi

echo ""

# Build the binary
echo "Building caws..."
go build -o caws

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Cannot write to $INSTALL_DIR, will install to ~/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Install
echo "Installing to $INSTALL_DIR..."
mv caws "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/caws"

echo ""
echo "✅ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Initialize vault:      caws init"
echo "  2. Add a profile:         caws add <profile-name>"
echo "  3. Use it:                caws exec <profile> -- aws s3 ls"
echo ""
echo "Run 'caws help' for more information"
