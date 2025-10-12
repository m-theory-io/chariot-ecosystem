#!/bin/bash

# Enhanced Azure Deployment Script - VM deployment with ACR
# Combines cross-platform builds with container registry deployment

set -e

echo "🚀 Enhanced Azure VM deployment for Chariot Ecosystem..."

# Configuration
AZURE_VM="${AZURE_VM:-chariot-vm}"
AZURE_VM_USER="${AZURE_VM_USER:-azureuser}"
AZURE_VM_HOST="${AZURE_VM_HOST}"
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-chariot-ecosystem}"
REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_building() { echo -e "${BLUE}[BUILD]${NC} $1"; }

# Check if VM host is provided
if [ -z "$AZURE_VM_HOST" ]; then
    print_error "AZURE_VM_HOST environment variable must be set"
    print_status "Example: export AZURE_VM_HOST=your-vm.eastus.cloudapp.azure.com"
    exit 1
fi

print_status "Target VM: $AZURE_VM_USER@$AZURE_VM_HOST"

# Step 1: Build and push images to ACR (canonical flow)
print_building "Building and pushing images to Azure Container Registry (canonical flow)..."
TAG=${TAG:-latest}
./scripts/build-azure-cross-platform.sh "$TAG" all
./scripts/push-images.sh "$TAG" all

# Step 2: Prepare deployment package
print_status "Preparing deployment package..."
mkdir -p tmp/azure-deploy

# Copy essential files
cp docker-compose.azure.yml tmp/azure-deploy/docker-compose.yml
cp -r databases tmp/azure-deploy/
cp .env.azure tmp/azure-deploy/.env 2>/dev/null || echo "# Azure environment" > tmp/azure-deploy/.env

# Create deployment script for VM
cat > tmp/azure-deploy/deploy-on-vm.sh << 'EOF'
#!/bin/bash
set -e

echo "🔄 Deploying Chariot Ecosystem on Azure VM..."

# Login to ACR from VM
az acr login --name mtheorycontainerregistry

# Pull latest images
echo "📥 Pulling latest images..."
docker compose pull

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker compose down || true

# Start with new images
echo "🚀 Starting updated services..."
docker compose up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to start..."
sleep 30

# Check service health
echo "🏥 Checking service health..."
docker compose ps

echo "✅ Deployment complete!"
echo "🌐 Services available at:"
echo "  • Visual DSL: http://$(hostname):3000"
echo "  • Charioteer: http://$(hostname):8080"  
echo "  • API: http://$(hostname):8087"
echo "  • Nginx: http://$(hostname)"
EOF

chmod +x tmp/azure-deploy/deploy-on-vm.sh

# Step 3: Deploy to VM
print_status "Deploying to Azure VM..."

# Copy files to VM
print_status "Copying deployment files to VM..."
scp -r tmp/azure-deploy/* $AZURE_VM_USER@$AZURE_VM_HOST:~/chariot-deploy/

# Execute deployment on VM
print_status "Executing deployment on VM..."
ssh $AZURE_VM_USER@$AZURE_VM_HOST "cd ~/chariot-deploy && ./deploy-on-vm.sh"

# Cleanup
rm -rf tmp/azure-deploy

print_status "✅ Deployment complete!"
print_status "🌐 Your Chariot Ecosystem is running at: http://$AZURE_VM_HOST"
