#!/bin/bash
set -euo pipefail

# Force a clean build/push of the nginx image without using cache
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO_ROOT"

IMAGE="mtheorycontainerregistry.azurecr.io/nginx:amd64"
DOCKERFILE="infrastructure/docker/nginx/Dockerfile.azure"
CONTEXT_DIR="infrastructure/docker/nginx"

echo "Building $IMAGE with --no-cache and --pull..."

docker buildx build \
  --platform linux/amd64 \
  --no-cache --pull \
  -f "$DOCKERFILE" \
  -t "$IMAGE" \
  --push \
  "$CONTEXT_DIR"

echo "Done."
