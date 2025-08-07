#!/bin/bash

# Build ESPN Client
echo "🔨 Building ESPN Client..."

# Build the executable
go build -o espn-client ./cmd/espn-client/

if [ $? -eq 0 ]; then
    echo "✅ ESPN Client built successfully!"
    echo ""
    echo "Usage examples:"
    echo "  ./espn-client -cmd=list-events"
    echo "  ./espn-client -cmd=list-specific -season=2024 -week=1"
    echo "  ./espn-client -cmd=get-event -event=401671708"
    echo "  ./espn-client -help"
else
    echo "❌ Build failed!"
    exit 1
fi