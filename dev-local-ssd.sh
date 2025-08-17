#!/bin/bash

# Development script for local (non-Docker) Chariot development on StudioSSD
# This bypasses Docker build issues and disk space problems

set -e

echo "🚀 Starting Chariot Local Development on StudioSSD"

# Set workspace paths
export GOPATH="/media/nvidia/StudioSSD/go"
export CHARIOT_WORKSPACE="/media/nvidia/StudioSSD/go/src/github.com/bhouse1273/chariot-ecosystem"
export GO_CHARIOT_DIR="$CHARIOT_WORKSPACE/services/go-chariot"

# Change to workspace
cd "$CHARIOT_WORKSPACE"

echo "📁 Working in: $CHARIOT_WORKSPACE"
echo "💾 Available space:"
df -h /media/nvidia/StudioSSD | tail -1

# Start support services with Docker Compose (only database/external services)
echo "🐳 Starting infrastructure services (MySQL, Couchbase, nginx)..."
cd "$CHARIOT_WORKSPACE"
docker compose up -d mysql couchbase nginx

# Wait for services to be ready
echo "⏳ Waiting for infrastructure services to start..."
sleep 10

# Build and start Charioteer locally
echo "🔨 Building Charioteer locally..."
cd "$CHARIOT_WORKSPACE/services/charioteer"
go mod tidy
go build -buildvcs=false -o bin/charioteer-local .

echo "▶️ Starting Charioteer service locally (in background)..."
export CHARIOT_BACKEND_URL=http://localhost:8087
export NODE_ENV=development
nohup ./bin/charioteer-local > "$CHARIOT_WORKSPACE/logs/charioteer/charioteer.log" 2>&1 &
CHARIOTEER_PID=$!
echo "📝 Charioteer PID: $CHARIOTEER_PID (logs at logs/charioteer/charioteer.log)"

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 5

# Set environment for Go development
export GO_ENV=development
export SKIP_DB_WAIT=true
export CHARIOT_CERT_PATH=/tmp/ssl
export CHARIOT_SSL=false
export COUCHBASE_URL=couchbase://localhost
export COUCHBASE_USERNAME=Administrator  
export COUCHBASE_PASSWORD=chariot123
export COUCHBASE_BUCKET=chariot
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
export MYSQL_DATABASE=chariot
export MYSQL_USERNAME=chariot
export MYSQL_PASSWORD=chariot123
export CHARIOTEER_URL=http://localhost:8080
export AZURE_TENANT_ID=82fbfa53-3046-4f39-a182-f5e0082313d4
export AZURE_KEY_VAULT_URL=https://chariot-vault.vault.azure.net
export AZURE_CONFIG_DIR=/home/nvidia/.azure
export CHARIOT_VAULT_KEY_PREFIX=docker

# Build go-chariot locally (much faster than Docker)
echo "🔨 Building go-chariot locally..."
cd "$GO_CHARIOT_DIR"

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Build binary
echo "🏗️ Building go-chariot binary..."
go build -buildvcs=false -o bin/go-chariot-local ./cmd

# Run the service
echo "▶️ Starting go-chariot service locally..."
echo "🎯 Running from: $(pwd)"
echo "🔗 Go-Chariot will be available at: http://localhost:8087"
echo "🌐 Charioteer UI available at: http://localhost:8080"
echo ""
echo "💡 To debug, use VS Code debugger or add breakpoints"
echo "🛑 Press Ctrl+C to stop all services"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    if [ ! -z "$CHARIOTEER_PID" ]; then
        kill $CHARIOTEER_PID 2>/dev/null || true
        echo "📝 Stopped Charioteer (PID: $CHARIOTEER_PID)"
    fi
    echo "🐳 Stopping Docker services..."
    cd "$CHARIOT_WORKSPACE"
    docker compose down
    echo "✅ All services stopped"
    exit 0
}

# Set trap to cleanup on exit
trap cleanup INT TERM EXIT

./bin/go-chariot-local
