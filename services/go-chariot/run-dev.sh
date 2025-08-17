#!/bin/bash

echo "Starting go-chariot in development mode with live code mounting..."
echo "This mounts your source code directly into the container"
echo "Rebuild/restart to see changes"

# Stop existing container
docker stop chariot-go-chariot 2>/dev/null || true
docker rm chariot-go-chariot 2>/dev/null || true

# Run with source code mounted
docker run -d \
  --name chariot-go-chariot-dev \
  --network host \
  -v /home/nvidia/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot:/app/src \
  -v ~/.azure:/home/gochariot/.azure:rw \
  -w /app/src \
  -e GO_ENV=development \
  -e SKIP_DB_WAIT=true \
  -e CHARIOT_CERT_PATH=/tmp/ssl \
  -e CHARIOT_SSL=false \
  -e CHARIOT_VERBOSE=true \
  -e COUCHBASE_URL=couchbase://localhost \
  -e COUCHBASE_USERNAME=Administrator \
  -e COUCHBASE_PASSWORD=chariot123 \
  -e COUCHBASE_BUCKET=chariot \
  -e MYSQL_HOST=localhost \
  -e MYSQL_PORT=3306 \
  -e MYSQL_DATABASE=chariot \
  -e MYSQL_USERNAME=chariot \
  -e MYSQL_PASSWORD=chariot123 \
  -e CHARIOTEER_URL=http://localhost:8080 \
  -e AZURE_TENANT_ID=82fbfa53-3046-4f39-a182-f5e0082313d4 \
  -e AZURE_KEY_VAULT_URL=https://chariot-vault.vault.azure.net \
  -e AZURE_CONFIG_DIR=/home/gochariot/.azure \
  -e CHARIOT_VAULT_KEY_PREFIX=docker \
  golang:1.24-alpine \
  sh -c "go mod download && go run ./cmd"

echo "Container started. Check logs with: docker logs -f chariot-go-chariot-dev"
echo "To rebuild after changes: docker restart chariot-go-chariot-dev"
