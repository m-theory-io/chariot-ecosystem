#!/bin/bash
# Build ARM64 images and deploy the local-only stack.

set -euo pipefail

COMPOSE_FILE="docker-compose.local.yml"
PLATFORM="linux/arm64/v8"

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "Compose file '$COMPOSE_FILE' not found. Run this script from the repo root." >&2
  exit 1
fi

export DOCKER_DEFAULT_PLATFORM="$PLATFORM"

echo "🔧 Building images for $PLATFORM via $COMPOSE_FILE..."
docker compose -f "$COMPOSE_FILE" build --pull

echo "📦 Deploying stack locally..."
docker compose -f "$COMPOSE_FILE" up -d

echo "✅ Local stack is up. Use 'docker compose -f $COMPOSE_FILE logs -f' to follow logs."
