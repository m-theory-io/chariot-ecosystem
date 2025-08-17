#!/bin/bash

# Set environment variables for local debugging
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
export AZURE_CONFIG_DIR=$HOME/.azure
export CHARIOT_VAULT_KEY_PREFIX=docker

echo "Environment variables set for go-chariot debugging"
echo "MySQL_PORT set to 3306 (direct connection, not through nginx proxy)"
echo ""
echo "To run with debugger:"
echo "  dlv debug ./cmd --listen=:2345 --headless=true --api-version=2 --accept-multiclient"
echo ""
echo "To run normally:"
echo "  go run ./cmd"
echo ""
echo "To build and run:"
echo "  go build -o bin/go-chariot-debug ./cmd && ./bin/go-chariot-debug"
