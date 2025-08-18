#!/bin/bash

# Open ports 80 and 443 for Chariot Ecosystem
# Run this script to add firewall rules

set -e

echo "🔓 Opening HTTP/HTTPS ports for Chariot Ecosystem..."

# Configuration
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-chariot-ecosystem}"
VM_NAME="${VM_NAME:-chariot-vm}"
NSG_NAME="${NSG_NAME:-chariot-vm-nsg}"  # Default NSG name

# Get the NSG name associated with the VM if not specified
if [ "$NSG_NAME" = "chariot-vm-nsg" ]; then
    NSG_NAME=$(az vm show --resource-group $RESOURCE_GROUP --name $VM_NAME --query "networkProfile.networkInterfaces[0].id" -o tsv | xargs az network nic show --ids | jq -r '.networkSecurityGroup.id' | cut -d'/' -f9)
    echo "📋 Found NSG: $NSG_NAME"
fi

# Add HTTP rule (port 80)
echo "🌐 Adding HTTP (port 80) rule..."
az network nsg rule create \
    --resource-group $RESOURCE_GROUP \
    --nsg-name $NSG_NAME \
    --name Allow-HTTP \
    --priority 1000 \
    --protocol Tcp \
    --destination-port-ranges 80 \
    --access Allow \
    --direction Inbound \
    --source-address-prefixes "*"

# Add HTTPS rule (port 443)
echo "🔒 Adding HTTPS (port 443) rule..."
az network nsg rule create \
    --resource-group $RESOURCE_GROUP \
    --nsg-name $NSG_NAME \
    --name Allow-HTTPS \
    --priority 1001 \
    --protocol Tcp \
    --destination-port-ranges 443 \
    --access Allow \
    --direction Inbound \
    --source-address-prefixes "*"

echo "✅ Firewall rules added successfully!"
echo "🌐 Your VM now accepts traffic on ports 80 and 443"
echo "🔒 Ready for Let's Encrypt certificate generation"
