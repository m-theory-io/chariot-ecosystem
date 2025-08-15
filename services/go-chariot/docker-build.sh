#!/bin/bash
set -e

echo "🔧 Building go-chariot for Linux..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/go-chariot .

echo "🐳 Building Docker image..."
docker build -t go-chariot:latest .

echo "✅ Build complete!"
echo ""
echo "To deploy locally:"
echo "  docker-compose up -d"
echo ""
echo "To push to Azure Container Registry:"
echo "  docker tag go-chariot:latest your-registry.azurecr.io/go-chariot:latest"
echo "  docker push your-registry.azurecr.io/go-chariot:latest"
