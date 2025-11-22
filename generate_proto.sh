#!/bin/bash

# Quick setup script to install protoc plugins and generate files

echo "Installing protoc Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo ""
echo "Plugins installed. Now generating protobuf files..."
echo ""

# Add GOPATH/bin to PATH for this session
export PATH="$PATH:$(go env GOPATH)/bin"

# Generate protobuf files
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/auction.proto

echo ""
echo "✓ Protobuf files generated successfully!"
echo ""
echo "To avoid this issue in the future, add this line to your ~/.zshrc:"
echo "export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
