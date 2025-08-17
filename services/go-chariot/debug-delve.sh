#!/bin/bash

# Stop Docker container first
echo "Stopping Docker container..."
docker stop chariot-go-chariot 2>/dev/null || true

# Set environment variables
export GO_ENV=development
export SKIP_DB_WAIT=true
export CHARIOT_CERT_PATH=/tmp/ssl
export CHARIOT_SSL=false
export CHARIOT_VERBOSE=true
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
export AZURE_CONFIG_DIR=$HOME/.azure
export CHARIOT_VAULT_KEY_PREFIX=docker

echo "Starting go-chariot with Delve debugger..."
echo "Debugger will listen on :2345"
echo "Connect with your IDE or use dlv connect :2345"
echo ""
echo "Common dlv commands:"
echo "  (dlv) break dispatchers.go:212     # Set breakpoint at mapSetAt call"
echo "  (dlv) break mapSetAt                # Set breakpoint at function start"
echo "  (dlv) continue                      # Start/resume execution"
echo "  (dlv) next                          # Step to next line"
echo "  (dlv) step                          # Step into function"
echo "  (dlv) print args                    # Print variable values"
echo "  (dlv) list                          # Show current code"
echo ""

cd /home/nvidia/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot
dlv debug ./cmd --listen=:2345 --headless=true --api-version=2 --accept-multiclient
