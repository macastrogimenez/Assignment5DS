#!/bin/bash

# Setup script for the Distributed Auction System
# This script installs dependencies and generates protobuf files

set -e  # Exit on error

echo "=== Distributed Auction System Setup ==="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or higher."
    echo "Visit: https://golang.org/doc/install"
    exit 1
fi

echo "✓ Go version: $(go version)"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc (Protocol Buffers compiler) is not installed."
    echo ""
    echo "To install on macOS:"
    echo "  brew install protobuf"
    echo ""
    echo "To install on Linux:"
    echo "  apt-get install -y protobuf-compiler"
    echo ""
    exit 1
fi

echo "✓ protoc version: $(protoc --version)"

# Install Go protobuf plugins
echo ""
echo "Installing Go protobuf plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add GOPATH/bin to PATH if not already there
export PATH="$PATH:$(go env GOPATH)/bin"

echo "✓ Protobuf plugins installed"

# Download Go dependencies
echo ""
echo "Downloading Go dependencies..."
go mod tidy

echo "✓ Dependencies downloaded"

# Generate protobuf files
echo ""
echo "Generating protobuf files..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/auction.proto

echo "✓ Protobuf files generated"

# Make test script executable
chmod +x test_scenario.sh

echo ""
echo "=== Setup Complete! ==="
echo ""
echo "Next steps:"
echo "1. Open 4 terminal windows"
echo "2. In terminals 1-3, start the nodes:"
echo "   Terminal 1: make run-node1"
echo "   Terminal 2: make run-node2"
echo "   Terminal 3: make run-node3"
echo "3. In terminal 4, start the client:"
echo "   Terminal 4: make run-client"
echo ""
echo "Or run manually:"
echo "   go run node/main.go -id=1 -port=5001 -peers=localhost:5002,localhost:5003"
echo "   go run node/main.go -id=2 -port=5002 -peers=localhost:5001,localhost:5003"
echo "   go run node/main.go -id=3 -port=5003 -peers=localhost:5001,localhost:5002"
echo "   go run client/main.go -nodes=localhost:5001,localhost:5002,localhost:5003"
echo ""
echo "See README.md for more information."
