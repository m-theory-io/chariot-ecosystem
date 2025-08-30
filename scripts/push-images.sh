#!/bin/bash
# Push Docker images to Azure Container Registry
# Usage: ./push-images.sh [TAG] [SERVICE]
#   TAG: Docker tag (default: latest)
#   SERVICE: Specific service to push (go-chariot, charioteer, visual-dsl, nginx) or 'all' (default: all)

set -e

REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
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
    echo "🚀 Pushing all Chariot images to $REGISTRY_NAME.azurecr.io with tag: $TAG..."
else
    echo "🚀 Pushing $SERVICE image to $REGISTRY_NAME.azurecr.io with tag: $TAG..."
fi

# Colors for output
print_status() { echo "[INFO] $1"; }
print_error() { echo "[ERROR] $1"; }
print_pushing() { echo "[PUSH] $1"; }

# Check prerequisites
print_status "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker not found. Please install Docker."
    exit 1
fi

# Check if we're logged into the registry
print_status "Checking Azure Container Registry login..."
if ! docker system info | grep -q "$REGISTRY_NAME.azurecr.io" 2>/dev/null; then
    print_status "Logging into Azure Container Registry..."
    if command -v az &> /dev/null; then
        az acr login --name $REGISTRY_NAME
    else
        print_error "Azure CLI not found. Please login manually with:"
        print_error "  az acr login --name $REGISTRY_NAME"
        print_error "Or use docker login directly:"
        print_error "  docker login $REGISTRY_NAME.azurecr.io"
        exit 1
    fi
fi

# Function to push go-chariot
push_go_chariot() {
    local local_image="go-chariot:$TAG"
    local remote_image="$REGISTRY_NAME.azurecr.io/go-chariot:$TAG"
    
    print_pushing "Checking if $local_image exists locally..."
    if ! docker image inspect $local_image >/dev/null 2>&1; then
        print_error "Local image $local_image not found. Please build it first."
        return 1
    fi
    
    print_pushing "Tagging $local_image as $remote_image..."
    docker tag $local_image $remote_image
    
    print_pushing "Pushing $remote_image..."
    docker push $remote_image
    
    print_status "✅ Successfully pushed $remote_image"
}

# Function to push charioteer
push_charioteer() {
    local local_image="charioteer:$TAG"
    local remote_image="$REGISTRY_NAME.azurecr.io/charioteer:$TAG"
    
    print_pushing "Checking if $local_image exists locally..."
    if ! docker image inspect $local_image >/dev/null 2>&1; then
        print_error "Local image $local_image not found. Please build it first."
        return 1
    fi
    
    print_pushing "Tagging $local_image as $remote_image..."
    docker tag $local_image $remote_image
    
    print_pushing "Pushing $remote_image..."
    docker push $remote_image
    
    print_status "✅ Successfully pushed $remote_image"
}

# Function to push visual-dsl
push_visual_dsl() {
    local local_image="visual-dsl:$TAG"
    local remote_image="$REGISTRY_NAME.azurecr.io/visual-dsl:$TAG"
    
    print_pushing "Checking if $local_image exists locally..."
    if ! docker image inspect $local_image >/dev/null 2>&1; then
        print_error "Local image $local_image not found. Please build it first."
        return 1
    fi
    
    print_pushing "Tagging $local_image as $remote_image..."
    docker tag $local_image $remote_image
    
    print_pushing "Pushing $remote_image..."
    docker push $remote_image
    
    print_status "✅ Successfully pushed $remote_image"
}

# Function to push nginx
push_nginx() {
    local local_image="nginx:$TAG"
    local remote_image="$REGISTRY_NAME.azurecr.io/nginx:$TAG"
    
    print_pushing "Checking if $local_image exists locally..."
    if ! docker image inspect $local_image >/dev/null 2>&1; then
        print_error "Local image $local_image not found. Please build it first."
        return 1
    fi
    
    print_pushing "Tagging $local_image as $remote_image..."
    docker tag $local_image $remote_image
    
    print_pushing "Pushing $remote_image..."
    docker push $remote_image
    
    print_status "✅ Successfully pushed $remote_image"
}

# Track successful pushes and failures
PUSHED_IMAGES=()
FAILED_IMAGES=()

# Push services based on argument
case "$SERVICE" in
    go-chariot)
        if push_go_chariot; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/go-chariot:$TAG")
        else
            FAILED_IMAGES+=("go-chariot:$TAG")
        fi
        ;;
    charioteer)
        if push_charioteer; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/charioteer:$TAG")
        else
            FAILED_IMAGES+=("charioteer:$TAG")
        fi
        ;;
    visual-dsl)
        if push_visual_dsl; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/visual-dsl:$TAG")
        else
            FAILED_IMAGES+=("visual-dsl:$TAG")
        fi
        ;;
    nginx)
        if push_nginx; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/nginx:$TAG")
        else
            FAILED_IMAGES+=("nginx:$TAG")
        fi
        ;;
    all)
        if push_go_chariot; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/go-chariot:$TAG")
        else
            FAILED_IMAGES+=("go-chariot:$TAG")
        fi
        
        if push_charioteer; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/charioteer:$TAG")
        else
            FAILED_IMAGES+=("charioteer:$TAG")
        fi
        
        if push_visual_dsl; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/visual-dsl:$TAG")
        else
            FAILED_IMAGES+=("visual-dsl:$TAG")
        fi
        
        if push_nginx; then
            PUSHED_IMAGES+=("$REGISTRY_NAME.azurecr.io/nginx:$TAG")
        else
            FAILED_IMAGES+=("nginx:$TAG")
        fi
        ;;
esac

# Summary
echo ""
print_status "📦 Push Summary:"

if [ ${#PUSHED_IMAGES[@]} -gt 0 ]; then
    echo "✅ Successfully pushed images:"
    for image in "${PUSHED_IMAGES[@]}"; do
        echo "   - $image"
    done
fi

if [ ${#FAILED_IMAGES[@]} -gt 0 ]; then
    echo "❌ Failed to push images:"
    for image in "${FAILED_IMAGES[@]}"; do
        echo "   - $image (not found locally)"
    done
    echo ""
    echo "💡 Build missing images first:"
    echo "   ./scripts/build-azure-cross-platform.sh $TAG $SERVICE"
    exit 1
fi

echo ""
print_status "🎉 All images pushed successfully to $REGISTRY_NAME.azurecr.io!"
print_status "Images are now available for Azure deployment."
