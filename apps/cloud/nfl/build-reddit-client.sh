#!/bin/bash
# Build script for reddit-client

set -e

echo "Building reddit-client..."
go build -o reddit-client ./cmd/reddit-client

echo "Build complete! Binary: ./reddit-client"
echo ""
echo "Usage examples:"
echo "  ./reddit-client -search \"rams seahawks\""
echo "  ./reddit-client -thread <url>"
echo "  ./reddit-client -thread <url> -export comments.json"
