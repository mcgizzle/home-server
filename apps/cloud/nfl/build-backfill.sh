#!/bin/bash

# Build script for NFL Backfill CLI tool

set -e

echo "Building NFL Backfill CLI tool..."

# Build the backfill CLI
go build -o backfill-cli ./cmd/backfill/

echo "✅ Build complete! Backfill CLI tool built as: ./backfill-cli"
echo ""
echo "Usage examples:"
echo "  ./backfill-cli -season 2024                 # Backfill 2024 season"
echo "  ./backfill-cli -season 2024 -limit 5       # Backfill 2024, stop after 5 competitions"
echo "  ./backfill-cli -season 2023 -json          # Backfill 2023 with JSON output"
echo "  ./backfill-cli -help                       # Show help"
echo "" 