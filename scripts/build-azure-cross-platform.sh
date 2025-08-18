#!/bin/bash

# Cross-platform build script for Azure deployment
# Handles M1 Mac -> AMD64 Linux builds with proper platform targeting

set -e

echo "🔨 Building Chariot services for Azure (AMD64) from M1 Mac..."

# Configuration
REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
TARGET_PLATFORM="linux/amd64"

# Colors for output (simplified to avoid terminal issues)
print_status() { echo "[INFO] $1"; }
print_warning() { echo "[WARN] $1"; }
print_error() { echo "[ERROR] $1"; }
print_building() { echo "[BUILD] $1"; }

# Function definitions already included above

# Check prerequisites
print_status "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker not found. Please install Docker."
    exit 1
fi

# Check if Docker buildx is available
if ! docker buildx version &> /dev/null; then
    print_error "Docker buildx not found. Please ensure Docker Desktop is updated."
    exit 1
fi

# Create and use a builder instance for cross-platform builds
print_status "Setting up cross-platform builder..."
docker buildx create --name chariot-cross-builder --driver docker-container --use 2>/dev/null || docker buildx use chariot-cross-builder

# Inspect the builder to ensure it supports the target platform
print_status "Checking builder capabilities..."
docker buildx inspect --bootstrap

print_status "Building for platform: $TARGET_PLATFORM"

# Build Go services with cross-compilation
print_building "Building Go binaries for AMD64..."

# Build go-chariot binary
print_building "Building go-chariot binary..."
cd services/go-chariot
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o build/go-chariot-linux-amd64 ./cmd
cd ../..

# Build charioteer binary
print_building "Building charioteer binary..."
cd services/charioteer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o build/charioteer-linux-amd64 .
cd ../..

print_status "Go binaries built successfully!"

# Build Docker images with platform specification
print_building "Building Docker images for $TARGET_PLATFORM..."

# Build visual-dsl with cross-platform support
print_building "Building visual-dsl image..."
docker buildx build \
    --platform $TARGET_PLATFORM \
    -f infrastructure/docker/visual-dsl/Dockerfile.azure \
    -t visual-dsl:amd64 \
    --load \
    ./services/visual-dsl

# Build nginx image
print_building "Building nginx image..."
docker buildx build \
    --platform $TARGET_PLATFORM \
    -f infrastructure/docker/nginx/Dockerfile.azure \
    -t nginx:amd64 \
    --load \
    ./infrastructure/docker/nginx

# Build go-chariot image
print_building "Building go-chariot image..."
docker buildx build \
    --platform $TARGET_PLATFORM \
    -f infrastructure/docker/go-chariot/Dockerfile.azure \
    -t go-chariot:amd64 \
    --load \
    ./services/go-chariot

# Build charioteer image
print_building "Building charioteer image..."
docker buildx build \
    --platform $TARGET_PLATFORM \
    -f infrastructure/docker/charioteer/Dockerfile.azure \
    -t charioteer:amd64 \
    --load \
    ./services/charioteer

print_status "✅ All images built successfully for $TARGET_PLATFORM!"

# Tag images for registry if registry name is provided
if [ -n "$REGISTRY_NAME" ]; then
    print_status "Tagging images for registry: $REGISTRY_NAME.azurecr.io"
    
    docker tag visual-dsl:amd64 "$REGISTRY_NAME.azurecr.io/visual-dsl:amd64"
    docker tag nginx:amd64 "$REGISTRY_NAME.azurecr.io/nginx:amd64"
    docker tag go-chariot:amd64 "$REGISTRY_NAME.azurecr.io/go-chariot:amd64"
    docker tag charioteer:amd64 "$REGISTRY_NAME.azurecr.io/charioteer:amd64"
    
    print_status "Images tagged for registry!"
    print_warning "To push to registry, run: docker push $REGISTRY_NAME.azurecr.io/IMAGE_NAME:amd64"
fi

print_status "Cross-platform build complete!"
print_status "Built images:"
echo "  • visual-dsl:amd64"
echo "  • nginx:amd64" 
echo "  • go-chariot:amd64"
echo "  • charioteer:amd64"

# Verify images are for correct architecture
print_status "Verifying image architectures..."
for image in visual-dsl:amd64 nginx:amd64 go-chariot:amd64 charioteer:amd64; do
    arch=$(docker inspect $image --format='{{.Architecture}}')
    echo "  • $image: $arch"
done

print_status "🎉 Build complete! Images are ready for Azure deployment."
