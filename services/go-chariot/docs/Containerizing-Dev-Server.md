# Containerizing go-chariot Server (Development Mode)

## Overview

This setup containerizes the go-chariot server in development mode with:
- Bootstrap configuration via `data/bootstrap.ch`
- Data trees included in container
- SSL termination at nginx proxy
- Azure-ready deployment with ACR

## Architecture

```
Internet → nginx (SSL termination) → go-chariot server (HTTP)
```

---

## 1. Dockerfile for go-chariot Server

**Dockerfile:**
```dockerfile
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy the go-chariot binary
COPY bin/go-chariot /usr/local/bin/go-chariot
RUN chmod +x /usr/local/bin/go-chariot

# Copy data directory with trees and bootstrap
COPY data/ /app/data/

# Copy configuration
COPY configs/ /app/configs/

# Create logs directory
RUN mkdir -p /app/logs

# Expose port (HTTP only, SSL handled by nginx)
EXPOSE 8087

# Set environment variables for development mode
ENV CHARIOT_DATA_PATH=/app/data
ENV CHARIOT_TREE_PATH=/app/data/trees
ENV CHARIOT_BOOTSTRAP_FILE=/app/data/bootstrap.ch
ENV CHARIOT_LOG_PATH=/app/logs
ENV CHARIOT_DEV_MODE=true

# Start the server
CMD ["/usr/local/bin/go-chariot", "-bootstrap", "/app/data/bootstrap.ch"]
```

---

## 2. Docker Compose Setup

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  go-chariot:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: go-chariot-server
    restart: unless-stopped
    ports:
      - "8087:8087"
    environment:
      - CHARIOT_DATA_PATH=/app/data
      - CHARIOT_TREE_PATH=/app/data/trees
      - CHARIOT_BOOTSTRAP_FILE=/app/data/bootstrap.ch
      - CHARIOT_LOG_PATH=/app/logs
      - CHARIOT_DEV_MODE=true
    volumes:
      - chariot-logs:/app/logs
      - chariot-data:/app/data
    networks:
      - chariot-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8087/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  nginx:
    image: nginx:alpine
    container_name: chariot-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - /etc/letsencrypt:/etc/letsencrypt:ro
      - /var/www/certbot:/var/www/certbot:ro
    depends_on:
      - go-chariot
    networks:
      - chariot-network

  certbot:
    image: certbot/certbot
    container_name: chariot-certbot
    volumes:
      - /etc/letsencrypt:/etc/letsencrypt
      - /var/www/certbot:/var/www/certbot
    command: certonly --webroot --webroot-path=/var/www/certbot --email your-email@domain.com --agree-tos --no-eff-email -d your-domain.com

volumes:
  chariot-logs:
  chariot-data:

networks:
  chariot-network:
    driver: bridge
```

---

## 3. Nginx Configuration

**nginx/nginx.conf:**
```nginx
events {
    worker_connections 1024;
}

http {
    upstream go-chariot {
        server go-chariot:8087;
    }

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name your-domain.com;
        
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        
        location / {
            return 301 https://$server_name$request_uri;
        }
    }

    # HTTPS server
    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
        
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        location / {
            proxy_pass http://go-chariot;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

---

## 4. Build and Deploy Scripts

**build.sh:**
```bash
#!/bin/bash
set -e

echo "Building go-chariot binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/go-chariot-linux .

echo "Building Docker image..."
docker build -t your-registry.azurecr.io/go-chariot:latest .

echo "Pushing to Azure Container Registry..."
docker push your-registry.azurecr.io/go-chariot:latest

echo "Build complete!"
```

**deploy.sh:**
```bash
#!/bin/bash
set -e

# Pull latest image
docker-compose pull

# Start services
docker-compose up -d

# Show status
docker-compose ps

echo "Deployment complete!"
echo "Server will be available at https://your-domain.com"
```

---

## 5. Azure VM Setup

**On your Azure Linux VM:**

1. **Install Docker and Docker Compose** (if not already installed):
```bash
# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

2. **Login to Azure Container Registry:**
```bash
az acr login --name your-registry
# or
docker login your-registry.azurecr.io
```

3. **Deploy the application:**
```bash
git clone your-repo
cd go-chariot
# Update domain name in nginx.conf and docker-compose.yml
./deploy.sh
```

4. **Setup SSL certificates:**
```bash
# Initial certificate generation
docker-compose run --rm certbot

# Setup automatic renewal
echo "0 12 * * * /usr/local/bin/docker-compose -f /path/to/docker-compose.yml run --rm certbot renew --quiet" | sudo crontab -
```

---

## 6. Environment Variables

For production, create a `.env` file:
```env
CHARIOT_DOMAIN=your-domain.com
CHARIOT_EMAIL=your-email@domain.com
ACR_REGISTRY=your-registry.azurecr.io
CHARIOT_IMAGE_TAG=latest
```

Update docker-compose.yml to use environment variables:
```yaml
services:
  go-chariot:
    image: ${ACR_REGISTRY}/go-chariot:${CHARIOT_IMAGE_TAG}
  # ... rest of config
```

---

## 7. Monitoring and Logs

**View logs:**
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f go-chariot
docker-compose logs -f nginx
```

**Health checks:**
```bash
# Check container status
docker-compose ps

# Check go-chariot health
curl https://your-domain.com/health
```

---

## Summary

This setup provides:
- ✅ Containerized go-chariot server with data/trees and bootstrap.ch
- ✅ SSL termination at nginx with LetsEncrypt
- ✅ Azure-ready deployment with ACR integration
- ✅ Development mode configuration
- ✅ Health checks and monitoring
- ✅ Automatic SSL renewal

**Next Steps:**
1. Update domain names in configuration files
2. Build and push initial image to ACR
3. Deploy to Azure VM
4. Setup SSL certificates
5. Configure monitoring/alerts
