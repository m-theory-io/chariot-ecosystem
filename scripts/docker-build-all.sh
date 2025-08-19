#!/usr/bin/env zsh
set -euo pipefail

# Adjust these tags as needed
ACR="mtheorycontainerregistry.azurecr.io"
GO_CHARIOT_TAG="v0.004"
CHARIOTEER_TAG="v0.004"
VISUAL_DSL_TAG="v0.004"
NGINX_TAG="v0.002"

# Optional: ensure you're logged in
# docker login ${ACR}

# NOTE: go-chariot and charioteer Dockerfiles expect prebuilt AMD64 binaries at:
#   services/go-chariot/build/go-chariot-linux-amd64
#   services/charioteer/build/charioteer-linux-amd64
# Build those first in your CI/local as needed.

# go-chariot (API)
docker buildx build \
  --platform linux/amd64 \
  -t ${ACR}/go-chariot:${GO_CHARIOT_TAG} \
  -f infrastructure/docker/go-chariot/Dockerfile.azure . \
  --push

# charioteer (web editor)
docker buildx build \
  --platform linux/amd64 \
  -t ${ACR}/charioteer:${CHARIOTEER_TAG} \
  -f infrastructure/docker/charioteer/Dockerfile.azure . \
  --push

# visual-dsl (frontend) — Dockerfile expects repo-root context
docker buildx build \
  --platform linux/amd64 \
  -t ${ACR}/visual-dsl:${VISUAL_DSL_TAG} \
  -f infrastructure/docker/visual-dsl/Dockerfile.azure . \
  --push

# nginx (reverse proxy) — use nginx folder as build context so COPY paths resolve
docker buildx build \
  --platform linux/amd64 \
  -t ${ACR}/nginx:${NGINX_TAG} \
  -f infrastructure/docker/nginx/Dockerfile.azure infrastructure/docker/nginx \
  --push

echo "Done. Update .env with:
GO_CHARIOT_TAG=${GO_CHARIOT_TAG}
CHARIOTEER_TAG=${CHARIOTEER_TAG}
VISUAL_DSL_TAG=${VISUAL_DSL_TAG}
NGINX_TAG=${NGINX_TAG}"