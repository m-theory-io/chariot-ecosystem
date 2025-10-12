#!/bin/bash

# Azure Deployment Script for Chariot Ecosystem
# Builds binaries and deploys to Azure Container Instances

set -e

echo "🚀 Starting Azure deployment for Chariot Ecosystem..."

# Configuration
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-chariot-ecosystem}"
LOCATION="${AZURE_LOCATION:-eastus}"
REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
SUBSCRIPTION="${AZURE_SUBSCRIPTION_ID}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
print_status "Checking prerequisites..."

if ! command -v az &> /dev/null; then
    print_error "Azure CLI not found. Please install Azure CLI."
    exit 1
fi

if ! command -v docker &> /dev/null; then
    print_error "Docker not found. Please install Docker."
    exit 1
fi

# Login to Azure
print_status "Logging into Azure..."
az login

# Set subscription if provided
if [ -n "$SUBSCRIPTION" ]; then
    print_status "Setting subscription to $SUBSCRIPTION"
    az account set --subscription "$SUBSCRIPTION"
fi

# Create resource group
print_status "Creating resource group '$RESOURCE_GROUP' in '$LOCATION'..."
az group create --name "$RESOURCE_GROUP" --location "$LOCATION"

# Create Azure Container Registry
print_status "Creating Azure Container Registry '$REGISTRY_NAME'..."
az acr create --resource-group "$RESOURCE_GROUP" --name "$REGISTRY_NAME" --sku Basic --admin-enabled true

# Get ACR login server
ACR_LOGIN_SERVER=$(az acr show --name "$REGISTRY_NAME" --resource-group "$RESOURCE_GROUP" --query "loginServer" --output tsv)
print_status "ACR Login Server: $ACR_LOGIN_SERVER"

# Login to ACR
print_status "Logging into Azure Container Registry..."
az acr login --name "$REGISTRY_NAME"

# Build AMD64 binaries and Docker images
print_status "Building AMD64 binaries and Docker images for production..."

# Use the cross-platform build script
./scripts/build-azure-cross-platform.sh

print_status "Cross-platform build completed!"

# Build and push Docker images
print_status "Pushing Docker images to ACR..."

# The images are already built by the cross-platform script, just push them
# Use repository names consistent with docker-compose.azure.yml and push-images.sh

print_status "Pushing go-chariot image..."
docker tag go-chariot:latest "$ACR_LOGIN_SERVER/go-chariot:latest"
docker push "$ACR_LOGIN_SERVER/go-chariot:latest"

print_status "Pushing charioteer image..."
docker tag charioteer:latest "$ACR_LOGIN_SERVER/charioteer:latest"
docker push "$ACR_LOGIN_SERVER/charioteer:latest"

print_status "Pushing visual-dsl image..."
docker tag visual-dsl:latest "$ACR_LOGIN_SERVER/visual-dsl:latest"
docker push "$ACR_LOGIN_SERVER/visual-dsl:latest"

print_status "Pushing nginx image..."
# Push both latest and amd64 tags; compose defaults to nginx:amd64
docker tag nginx:latest "$ACR_LOGIN_SERVER/nginx:latest"
docker push "$ACR_LOGIN_SERVER/nginx:latest"
# If local nginx:amd64 exists (built by cross-platform script), push it as the default alias
if docker image inspect nginx:amd64 >/dev/null 2>&1; then
    docker tag nginx:amd64 "$ACR_LOGIN_SERVER/nginx:amd64"
    docker push "$ACR_LOGIN_SERVER/nginx:amd64"
else
    print_warning "nginx:amd64 not found locally; only pushed nginx:latest. Run build-azure-cross-platform.sh to create amd64 alias if needed."
fi

print_status "All images pushed successfully!"

# Create Azure Container Instances or App Service (optional)
print_status "Deployment images are ready!"
print_status "ACR Login Server: $ACR_LOGIN_SERVER"
print_status ""
print_status "Next steps:"
print_status "1. Use Azure Container Instances:"
print_status "   az container create --resource-group $RESOURCE_GROUP --file azure-container-instances.yaml"
print_status ""
print_status "2. Use Azure App Service:"
print_status "   az webapp create --resource-group $RESOURCE_GROUP --plan myAppServicePlan --name chariot-app --deployment-container-image-name $ACR_LOGIN_SERVER/chariot-nginx:latest"
print_status ""
print_status "3. Use Azure Container Apps:"
print_status "   az containerapp create --resource-group $RESOURCE_GROUP --environment myEnvironment --image $ACR_LOGIN_SERVER/chariot-nginx:latest"

print_status "✅ Azure deployment preparation complete!"
