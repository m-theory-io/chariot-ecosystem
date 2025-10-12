# Cross-Platform Docker Builds for Azure

This document explains how to build Docker images on M1 Mac for deployment to Azure Linux AMD64 VMs.

## Problem

When building React/Vite applications with esbuild on M1 Mac for AMD64 Linux targets, you may encounter:
- esbuild platform mismatches
- Binary incompatibility issues  
- Docker build failures on target platforms

## Solution

We use Docker's cross-platform building capabilities with proper platform targeting.

## Quick Start

### 1. Test Cross-Platform Capabilities
```bash
./scripts/test-cross-platform.sh
```

### 2. Build All Services for Azure
```bash
./scripts/build-azure-cross-platform.sh
```

### 3. Push to Azure Container Registry
```bash
./scripts/build-azure-cross-platform.sh <tag> [all|service]
./scripts/push-images.sh <tag> [all|service]
```

## Manual Cross-Platform Builds

### Build Individual Services

#### Visual DSL (React/Vite)
```bash
docker buildx build \
    --platform linux/amd64 \
    -f infrastructure/docker/visual-dsl/Dockerfile.azure \
    -t visual-dsl:amd64 \
    ./services/visual-dsl
```

#### Nginx
```bash
docker buildx build \
    --platform linux/amd64 \
    -f infrastructure/docker/nginx/Dockerfile.azure \
    -t nginx:amd64 \
    ./infrastructure/docker/nginx
```

### Using Docker Compose
```bash
# Build with cross-platform overrides
docker-compose -f docker-compose.yml -f docker-compose.cross-platform.yml build
```

## Key Changes Made

### 1. Dockerfile Updates
- Added `FROM --platform=linux/amd64` to all Azure Dockerfiles
- Set esbuild platform environment variables for React builds
- Updated visual-dsl to use `build:amd64` script

### 2. Build Scripts
- Created `build-azure-cross-platform.sh` for automated cross-platform builds
- Use `push-images.sh` to publish to ACR; `deploy-azure.sh` has been removed to avoid confusion
- Added platform verification steps

### 3. Package.json Updates
- Added `build:amd64` script with platform-specific esbuild settings
- Set environment variables for cross-compilation

## Verification

After building, verify the image architecture:
```bash
docker inspect visual-dsl:amd64 --format='{{.Architecture}}'
# Should output: amd64
```

## Troubleshooting

### Docker Buildx Not Available
```bash
# Enable buildx (usually enabled by default in Docker Desktop)
docker buildx install
```

### Builder Platform Support
```bash
# Check supported platforms
docker buildx inspect --bootstrap
```

### esbuild Issues
If you still encounter esbuild issues, the Dockerfile sets:
- `ESBUILD_PLATFORM=linux`
- `ESBUILD_ARCH=x64`

These force esbuild to use the correct target platform regardless of host architecture.

## Notes

- Images built with this process will be AMD64 Linux compatible
- They can be safely deployed to Azure Linux VMs
- Local testing may require Docker Desktop's emulation (will be slower)
- For local development, continue using the regular `docker-compose.yml`
