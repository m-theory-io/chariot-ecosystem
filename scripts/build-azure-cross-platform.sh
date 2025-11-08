#!/bin/bash
# Cross-platform Azure build script
# Builds binaries locally and creates Docker images for Azure deployment
# Handles M1 Mac -> AMD64 Linux builds with proper platform targeting
# Usage: ./build-azure-cross-platform.sh [TAG] [SERVICE] [PLATFORM]
#   TAG: Docker tag (default: latest)
#   SERVICE: Specific service to build (go-chariot, charioteer, visual-dsl, nginx) or 'all' (default: all)
#   PLATFORM: Platform target for go-chariot (cpu, cuda, metal) (default: cpu)

set -e

REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
TARGET_PLATFORM="linux/amd64"
TAG=${1:-latest}
SERVICE=${2:-all}
PLATFORM_TARGET=${3:-cpu}

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
    echo "🔨 Building all Chariot services for Azure (AMD64) with tag: $TAG (go-chariot platform: $PLATFORM_TARGET)..."
else
    echo "🔨 Building $SERVICE service for Azure (AMD64) with tag: $TAG..."
    if [ "$SERVICE" = "go-chariot" ]; then
        echo "   Platform target: $PLATFORM_TARGET"
    fi
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

# Validate vendored knapsack libraries exist for the target platform
if [ "$SERVICE" = "go-chariot" ] || [ "$SERVICE" = "all" ]; then
    case "$PLATFORM_TARGET" in
        cpu)
            if [ ! -f "services/go-chariot/knapsack-library/lib/linux-cpu/libknapsack_cpu.a" ]; then
                print_error "Vendored CPU library not found"
                print_error "Expected: services/go-chariot/knapsack-library/lib/linux-cpu/libknapsack_cpu.a"
                print_error "Please vendor the knapsack libraries before building"
                exit 1
            fi
            print_status "Found vendored CPU library ($(du -h services/go-chariot/knapsack-library/lib/linux-cpu/libknapsack_cpu.a | cut -f1))"
            ;;
        cuda)
            if [ ! -f "services/go-chariot/knapsack-library/lib/linux-cuda/libknapsack_cuda.a" ]; then
                print_error "Vendored CUDA library not found"
                print_error "Expected: services/go-chariot/knapsack-library/lib/linux-cuda/libknapsack_cuda.a"
                print_error "Please vendor the knapsack libraries before building"
                exit 1
            fi
            print_status "Found vendored CUDA library ($(du -h services/go-chariot/knapsack-library/lib/linux-cuda/libknapsack_cuda.a | cut -f1))"
            ;;
        metal)
            print_warning "Metal target only supported for local macOS builds"
            print_warning "Use: CGO_ENABLED=1 go build -tags cgo ./cmd"
            ;;
    esac
fi

# Create and use a builder instance for cross-platform builds
print_status "Setting up cross-platform builder..."
docker buildx create --name chariot-cross-builder --driver docker-container --use 2>/dev/null || docker buildx use chariot-cross-builder

# Inspect the builder to ensure it supports the target platform
print_status "Checking builder capabilities..."
docker buildx inspect --bootstrap

print_status "Building for platform: $TARGET_PLATFORM"

# Prebuild shared chariot-codegen package to avoid stale bundles/types
prebuild_codegen() {
    if [ -d "packages/chariot-codegen" ]; then
        print_building "Prebuilding packages/chariot-codegen..."
        pushd packages/chariot-codegen >/dev/null
        if command -v npm &> /dev/null; then
            npm ci && npm run build
        else
            print_warning "npm not found. Skipping local codegen prebuild. Docker builds will still build the package inside the image."
        fi
        popd >/dev/null
    else
        print_warning "packages/chariot-codegen not found. Skipping local codegen prebuild."
    fi
}

prebuild_codegen

# Function to build go-chariot
build_go_chariot() {
    local platform=${1:-cpu}  # cpu, cuda, or metal
    
    print_building "Building go-chariot for platform: $platform with tag: $TAG"
    
    case "$platform" in
        cpu)
            print_building "Building go-chariot Docker image (Linux AMD64 CPU-only)..."
            
            # Build using vendored CPU library
            docker buildx build \
                --platform linux/amd64 \
                -f infrastructure/docker/go-chariot/Dockerfile.cpu \
                -t go-chariot:${TAG}-cpu \
                -t go-chariot:latest-cpu \
                --load \
                .
            
            print_status "✅ go-chariot:${TAG}-cpu built successfully"
            ;;
            
        cuda)
            print_building "Building go-chariot Docker image (Linux ARM64 CUDA GPU)..."
            
            # Build using vendored CUDA library
            docker buildx build \
                --platform linux/arm64 \
                -f infrastructure/docker/go-chariot/Dockerfile.cuda \
                -t go-chariot:${TAG}-cuda \
                -t go-chariot:latest-cuda \
                --load \
                .
            
            print_status "✅ go-chariot:${TAG}-cuda built successfully"
            ;;
            
        metal)
            print_error "Metal builds not supported in Docker"
            print_status "For local macOS Metal builds, use:"
            print_status "  cd services/go-chariot"
            print_status "  CGO_ENABLED=1 go build -tags cgo ./cmd"
            return 1
            ;;
            
        *)
            print_error "Unknown platform: $platform"
            print_error "Valid platforms: cpu (Linux AMD64), cuda (Linux ARM64), metal (macOS local only)"
            exit 1
            ;;
    esac
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
        build_go_chariot "$PLATFORM_TARGET"
        BUILT_IMAGES=("go-chariot:${TAG}-${PLATFORM_TARGET}")
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
        build_go_chariot "$PLATFORM_TARGET"
        build_charioteer
        build_visual_dsl
        build_nginx
        BUILT_IMAGES=(
            "go-chariot:${TAG}-${PLATFORM_TARGET}"
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
