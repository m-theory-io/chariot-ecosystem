#!/bin/bash
# Cross-platform Azure build script
# Builds binaries locally and creates Docker images for Azure deployment
# Handles M1 Mac -> AMD64 Linux builds with proper platform targeting
# Usage: ./build-azure-cross-platform.sh [TAG] [SERVICE]
#   TAG: Docker tag (default: latest)
#   SERVICE: Specific service to build (go-chariot, charioteer, visual-dsl, nginx) or 'all' (default: all)

set -e

REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
TARGET_PLATFORM="linux/amd64"
TAG=${1:-latest}
SERVICE=${2:-all}

# Validate service argument
case "$SERVICE" in
    go-chariot|charioteer|visual-dsl|nginx|all)
        ;;
    *)
        echo "❌ Invalid service: $SERVICE"
        echo "Valid options: go-chariot, charioteer, visual-dsl, nginx, all"
        exit 1
        ;;
esac

if [ "$SERVICE" = "all" ]; then
    echo "🔨 Building all Chariot services for Azure (AMD64) with tag: $TAG..."
else
    echo "🔨 Building $SERVICE service for Azure (AMD64) with tag: $TAG..."
fi

# Colors for output (simplified to avoid terminal issues)
print_status() { echo "[INFO] $1"; }
print_warning() { echo "[WARN] $1"; }
print_error() { echo "[ERROR] $1"; }
print_building() { echo "[BUILD] $1"; }

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

# Function to build go-chariot
build_go_chariot() {
    print_building "Building go-chariot binary..."
    cd services/go-chariot
    mkdir -p build
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o build/go-chariot-linux-amd64 ./cmd
    cd ../..

    print_building "Building go-chariot Docker image..."
    docker buildx build \
        --platform $TARGET_PLATFORM \
        -f infrastructure/docker/go-chariot/Dockerfile \
        -t go-chariot:$TAG \
        --load \
        .
}

# Function to build charioteer
build_charioteer() {
    print_building "Building charioteer binary..."
    cd services/charioteer
    mkdir -p build
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o build/charioteer-linux-amd64 .
    cd ../..

    print_building "Building charioteer Docker image..."
    docker buildx build \
        --platform $TARGET_PLATFORM \
        -f infrastructure/docker/charioteer/Dockerfile \
        -t charioteer:$TAG \
        --load \
        .
}

# Function to build visual-dsl
build_visual_dsl() {
    print_building "Building visual-dsl Docker image..."
    docker buildx build \
        --platform $TARGET_PLATFORM \
        -f infrastructure/docker/visual-dsl/Dockerfile \
        -t visual-dsl:$TAG \
        --load \
        --no-cache \
        .
}

# Function to build nginx
build_nginx() {
    print_building "Building nginx Docker image..."
    docker buildx build \
        --platform $TARGET_PLATFORM \
        -f infrastructure/docker/nginx/Dockerfile.azure \
        -t nginx:$TAG \
        --load \
        ./infrastructure/docker/nginx

    # Also tag the local image as :amd64 so default compose tag pulls the latest fix
    if [ "$TAG" != "amd64" ]; then
        print_status "Tagging nginx:$TAG also as nginx:amd64 (local convenience tag)"
        docker tag nginx:$TAG nginx:amd64 || true
    fi
}

# Build services based on argument
case "$SERVICE" in
    go-chariot)
        build_go_chariot
        BUILT_IMAGES=("go-chariot:$TAG")
        ;;
    charioteer)
        build_charioteer
        BUILT_IMAGES=("charioteer:$TAG")
        ;;
    visual-dsl)
        build_visual_dsl
        BUILT_IMAGES=("visual-dsl:$TAG")
        ;;
    nginx)
        build_nginx
        BUILT_IMAGES=("nginx:$TAG")
        ;;
    all)
        build_go_chariot
        build_charioteer
        build_visual_dsl
        build_nginx
        BUILT_IMAGES=(
            "go-chariot:$TAG"
            "charioteer:$TAG"
            "visual-dsl:$TAG"
            "nginx:$TAG"
        )
        ;;
esac

print_status "✅ Build completed successfully for $TARGET_PLATFORM!"

# Tag images for registry if registry name is provided
if [ -n "$REGISTRY_NAME" ]; then
    print_status "Tagging images for registry: $REGISTRY_NAME.azurecr.io"
    
    for image in "${BUILT_IMAGES[@]}"; do
        service_name=$(echo $image | cut -d: -f1)
        docker tag $image "$REGISTRY_NAME.azurecr.io/$service_name:$TAG"

        # Special-case nginx: also tag registry image as :amd64 to match compose default
        if [ "$service_name" = "nginx" ] && [ "$TAG" != "amd64" ]; then
            print_status "Creating additional registry tag for nginx: amd64"
            docker tag $image "$REGISTRY_NAME.azurecr.io/nginx:amd64"
        fi
    done
    
    print_status "Images tagged for registry!"
fi

print_status "📦 Images created:"
for image in "${BUILT_IMAGES[@]}"; do
    echo "   - $image"
done

# Verify images are for correct architecture
print_status "Verifying image architectures..."
for image in "${BUILT_IMAGES[@]}"; do
    arch=$(docker inspect $image --format='{{.Architecture}}')
    echo "  • $image: $arch"
done

echo ""
print_status "🚀 To push to registry:"
for image in "${BUILT_IMAGES[@]}"; do
    service_name=$(echo $image | cut -d: -f1)
    echo "   docker push $REGISTRY_NAME.azurecr.io/$service_name:$TAG"

    # Show extra push command for nginx default tag
    if [ "$service_name" = "nginx" ] && [ "$TAG" != "amd64" ]; then
        echo "   docker push $REGISTRY_NAME.azurecr.io/nginx:amd64"
    fi
done

print_status "🎉 Cross-platform build complete! Images are ready for Azure deployment."
