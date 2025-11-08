#!/bin/bash

# Azure AD App Registration for OAuth2 Proxy
# Run this script to create the required Azure AD application

set -e

echo "🔐 Creating Azure AD App Registration for Chariot Ecosystem..."

# Configuration
APP_NAME="Chariot-Ecosystem-OAuth"
REDIRECT_URI="https://chariot.centralus.cloudapp.azure.com/oauth2/callback"
HOME_PAGE_URL="https://chariot.centralus.cloudapp.azure.com"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }

# Create the app registration
print_status "Creating Azure AD app registration..."
APP_ID=$(az ad app create \
    --display-name "$APP_NAME" \
    --web-redirect-uris "$REDIRECT_URI" \
    --web-home-page-url "$HOME_PAGE_URL" \
    --query appId -o tsv)

print_status "App registration created with ID: $APP_ID"

# Create a service principal
print_status "Creating service principal..."
az ad sp create --id $APP_ID

# Generate client secret
print_status "Generating client secret..."
CLIENT_SECRET=$(az ad app credential reset \
    --id $APP_ID \
    --credential-description "OAuth2 Proxy Secret" \
    --query password -o tsv)

# Get tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)

# Generate cookie secret
COOKIE_SECRET=$(openssl rand -base64 32)

print_status "✅ Azure AD setup complete!"
print_info ""
print_info "📋 Save these values to your .env.azure file:"
print_info ""
echo "# Azure AD OAuth2 Configuration"
echo "export AZURE_TENANT_ID=$TENANT_ID"
echo "export AZURE_CLIENT_ID=$APP_ID"
echo "export AZURE_CLIENT_SECRET=$CLIENT_SECRET"
echo "export OAUTH2_COOKIE_SECRET=$COOKIE_SECRET"
echo ""
print_warning "⚠️  IMPORTANT: Save the CLIENT_SECRET now - you won't be able to see it again!"
print_info ""
print_info "🌐 Redirect URI configured: $REDIRECT_URI"
print_info "🏠 Home page URL: $HOME_PAGE_URL"
