#!/bin/bash
set -euo pipefail

if [ $# -lt 3 ]; then
  echo "Usage: $0 <linux/arch> <dockerfile> <image:tag> [<context-dir>]"
  echo "Example: $0 linux/amd64 infrastructure/docker/nginx/Dockerfile.azure mtheorycontainerregistry.azurecr.io/nginx:amd64 infrastructure/docker/nginx"
  exit 2
fi

PLATFORM="$1"
DOCKERFILE="$2"
IMAGE="$3"
CONTEXT_DIR="${4:-.}"

echo "Building $IMAGE for $PLATFORM with --no-cache and --pull from $DOCKERFILE (context $CONTEXT_DIR)"

docker buildx build \
  --platform "$PLATFORM" \
  --no-cache --pull \
  -f "$DOCKERFILE" \
  -t "$IMAGE" \
  --push \
  "$CONTEXT_DIR"

echo "Done."
