#!/bin/bash

# Stop the Docker container first
echo "Stopping Docker container..."
docker stop chariot-go-chariot 2>/dev/null || true

# Set environment variables for local development
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

echo "Starting go-chariot locally on port 8087..."
echo "Your changes will be immediately active!"
echo "Press Ctrl+C to stop"
echo ""

cd /home/nvidia/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot
go run ./cmd
