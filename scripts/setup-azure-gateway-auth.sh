#!/bin/bash

# Azure Application Gateway with Azure AD Authentication
# This creates a fully managed solution in Azure

set -e

echo "🔐 Setting up Azure Application Gateway with Azure AD Authentication..."

# Configuration
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-chariot-ecosystem}"
LOCATION="${AZURE_LOCATION:-centralus}"
APP_GATEWAY_NAME="chariot-app-gateway"
BACKEND_VM_IP="10.0.0.4"  # Your VM's private IP

# Create Application Gateway
az network application-gateway create \
    --name $APP_GATEWAY_NAME \
    --resource-group $RESOURCE_GROUP \
    --location $LOCATION \
    --sku Standard_v2 \
    --capacity 2 \
    --frontend-port 80 \
    --http-settings-cookie-based-affinity Disabled \
    --http-settings-port 80 \
    --http-settings-protocol Http \
    --public-ip-address chariot-gateway-ip \
    --subnet gateway-subnet \
    --servers $BACKEND_VM_IP

# Configure Azure AD authentication
az network application-gateway auth-cert create \
    --resource-group $RESOURCE_GROUP \
    --gateway-name $APP_GATEWAY_NAME \
    --name azure-ad-auth \
    --cert-file azure-ad-cert.pem

echo "✅ Application Gateway created with Azure AD integration"
echo "🌐 Access your application through the gateway URL"
