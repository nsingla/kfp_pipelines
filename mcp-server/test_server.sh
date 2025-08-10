#!/bin/bash

# Test script for MCP Pipeline Server

echo "Testing MCP Pipeline Server..."

# Test initialize
echo "Testing initialize..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | go run main.go

echo ""

# Test tools/list
echo "Testing tools/list..."
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | go run main.go

echo ""

# Test invalid method
echo "Testing invalid method..."
echo '{"jsonrpc":"2.0","id":3,"method":"invalid_method"}' | go run main.go

echo ""

echo "All tests completed."