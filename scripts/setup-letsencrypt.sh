#!/bin/bash

# Let's Encrypt SSL Certificate Setup for Chariot Ecosystem
# This script sets up automatic SSL certificates using Certbot

set -e

echo "🔐 Setting up Let's Encrypt SSL certificates for Chariot Ecosystem..."

# Configuration
DOMAIN="${CHARIOT_DOMAIN:-chariot.m-theory.io}"
EMAIL="${LETSENCRYPT_EMAIL:-admin@m-theory.io}"

echo "Domain: $DOMAIN"
echo "Email: $EMAIL"

# Step 1: Start nginx first for Let's Encrypt challenge
echo "📋 Step 1: Starting nginx for Let's Encrypt challenge..."
docker compose -f docker-compose.letsencrypt.yml up -d nginx

# Wait for nginx to be ready
echo "⏳ Waiting for nginx to be ready..."
sleep 10

# Step 2: Get initial certificate
echo "📋 Step 2: Obtaining SSL certificate from Let's Encrypt..."
docker compose -f docker-compose.letsencrypt.yml run --rm certbot \
    certonly --webroot --webroot-path=/var/www/certbot \
    --email $EMAIL --agree-tos --no-eff-email \
    --force-renewal -d $DOMAIN

# Step 3: Start all services with SSL
echo "📋 Step 3: Starting all services with SSL..."
docker compose -f docker-compose.letsencrypt.yml up -d

# Step 4: Set up automatic renewal
echo "📋 Step 4: Setting up automatic certificate renewal..."
cat > /tmp/renew-certs.sh << 'EOF'
#!/bin/bash
cd /home/mtheory
docker compose -f docker-compose.letsencrypt.yml run --rm certbot renew
docker compose -f docker-compose.letsencrypt.yml restart oauth2-proxy
EOF

# Copy renewal script to VM
chmod +x /tmp/renew-certs.sh
echo "0 12 * * * /home/mtheory/renew-certs.sh" | crontab -

echo "✅ Let's Encrypt setup complete!"
echo "🌐 Your Chariot Ecosystem is now available at: https://$DOMAIN"
echo "🔄 Certificates will auto-renew via cron job"
echo ""
echo "🔗 Access points:"
echo "  • Main site: https://$DOMAIN"
echo "  • After Azure AD login, you'll see the nginx landing page"
echo "  • Click through to access Charioteer and other services"
