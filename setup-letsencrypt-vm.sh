#!/bin/bash

# Simple Let's Encrypt setup for Chariot on Azure VM
# Run this ON the Azure VM

set -e

echo "Setting up Let's Encrypt for chariot.centralus.cloudapp.azure.com..."

# Step 1: Stop current deployment
echo "Stopping current deployment..."
docker compose -f docker-compose-oauth2.yml down || true

# Step 2: Start nginx temporarily for certificate challenge
echo "Starting temporary nginx for Let's Encrypt challenge..."
docker run -d --name temp-nginx \
    -p 80:80 \
    -v certbot_www:/var/www/certbot \
    nginx:alpine

# Create a simple nginx config for Let's Encrypt
docker exec temp-nginx sh -c 'cat > /etc/nginx/conf.d/default.conf << EOF
server {
    listen 80;
    server_name chariot.centralus.cloudapp.azure.com;
    
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    location / {
        return 200 "Let\''s Encrypt setup in progress...";
        add_header Content-Type text/plain;
    }
}
EOF'

docker exec temp-nginx nginx -s reload

# Step 3: Get certificate
echo "Getting SSL certificate..."
docker run --rm \
    -v certbot_certs:/etc/letsencrypt \
    -v certbot_www:/var/www/certbot \
    certbot/certbot \
    certonly --webroot --webroot-path=/var/www/certbot \
    --email admin@mtheory.com --agree-tos --no-eff-email \
    -d chariot.centralus.cloudapp.azure.com

# Step 4: Stop temporary nginx
echo "Stopping temporary nginx..."
docker stop temp-nginx
docker rm temp-nginx

# Step 5: Start OAuth2 deployment with SSL
echo "Starting OAuth2 deployment with SSL..."
source ~/.env
docker compose -f docker-compose-letsencrypt.yml up -d

echo "✅ Let's Encrypt setup complete!"
echo "🌐 Visit: https://chariot.centralus.cloudapp.azure.com"
