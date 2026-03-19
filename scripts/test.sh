#!/bin/bash
# scripts/test.sh
set -e

echo "🔍 Running Pre-test Environment Check..."

# Ensure Git is installed
if ! command -v git &> /dev/null; then
    echo "❌ Error: git is not installed."
    exit 1
fi

# Ensure Docker is running (optional check)
if ! docker info &> /dev/null; then
    echo "⚠️  Warning: Docker is not running. Some container tests might skip."
fi

echo "🧪 Executing Go Tests..."
go test -v -cover ./...

echo "✅ All tests passed!"
