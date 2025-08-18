#!/bin/bash

# Azure Container Instances Deployment for Chariot Ecosystem
# Fully managed container deployment without VMs

set -e

echo "☁️ Deploying Chariot Ecosystem to Azure Container Instances..."

# Configuration
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-chariot-ecosystem}"
LOCATION="${AZURE_LOCATION:-centralus}"
REGISTRY_NAME="${AZURE_REGISTRY:-mtheorycontainerregistry}"
CONTAINER_GROUP_NAME="${CONTAINER_GROUP_NAME:-chariot-ecosystem}"

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

# Step 1: Build and push images
print_building "Building and pushing images to ACR..."
./scripts/deploy-azure.sh

# Step 2: Get ACR credentials
print_status "Getting ACR credentials..."
ACR_LOGIN_SERVER=$(az acr show --name $REGISTRY_NAME --query loginServer --output tsv)
ACR_USERNAME=$(az acr credential show --name $REGISTRY_NAME --query username --output tsv)
ACR_PASSWORD=$(az acr credential show --name $REGISTRY_NAME --query passwords[0].value --output tsv)

# Step 3: Create or update container group
print_status "Creating Azure Container Instance deployment..."

cat > aci-deployment.yaml << EOF
apiVersion: 2019-12-01
location: $LOCATION
name: $CONTAINER_GROUP_NAME
properties:
  imageRegistryCredentials:
  - server: $ACR_LOGIN_SERVER
    username: $ACR_USERNAME
    password: $ACR_PASSWORD
  containers:
  - name: visual-dsl
    properties:
      image: $ACR_LOGIN_SERVER/visual-dsl:amd64
      resources:
        requests:
          cpu: 0.5
          memoryInGb: 1
      ports:
      - port: 3000
        protocol: TCP
  - name: nginx
    properties:
      image: $ACR_LOGIN_SERVER/nginx:amd64
      resources:
        requests:
          cpu: 0.25
          memoryInGb: 0.5
      ports:
      - port: 80
        protocol: TCP
  - name: go-chariot
    properties:
      image: $ACR_LOGIN_SERVER/go-chariot:amd64
      resources:
        requests:
          cpu: 0.5
          memoryInGb: 1
      ports:
      - port: 8087
        protocol: TCP
  - name: charioteer
    properties:
      image: $ACR_LOGIN_SERVER/charioteer:amd64
      resources:
        requests:
          cpu: 0.5
          memoryInGb: 1
      ports:
      - port: 8080
        protocol: TCP
  osType: Linux
  restartPolicy: Always
  ipAddress:
    type: Public
    ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 3000
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8087
    dnsNameLabel: $CONTAINER_GROUP_NAME
tags:
  project: chariot-ecosystem
  environment: production
type: Microsoft.ContainerInstance/containerGroups
EOF

# Deploy container group
print_status "Deploying container group to Azure..."
az container create \
    --resource-group $RESOURCE_GROUP \
    --file aci-deployment.yaml

# Get the FQDN
FQDN=$(az container show \
    --resource-group $RESOURCE_GROUP \
    --name $CONTAINER_GROUP_NAME \
    --query ipAddress.fqdn \
    --output tsv)

# Cleanup
rm -f aci-deployment.yaml

print_status "✅ Deployment complete!"
print_status "🌐 Your Chariot Ecosystem is running at:"
print_status "  • Main Site: http://$FQDN"
print_status "  • Visual DSL: http://$FQDN:3000"
print_status "  • Charioteer: http://$FQDN:8080"
print_status "  • API: http://$FQDN:8087"
print_status ""
print_status "📊 Monitor deployment:"
print_status "  az container logs --resource-group $RESOURCE_GROUP --name $CONTAINER_GROUP_NAME"
print_status "  az container show --resource-group $RESOURCE_GROUP --name $CONTAINER_GROUP_NAME"
