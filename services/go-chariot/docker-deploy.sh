#!/bin/bash
set -e

echo "🚀 Deploying go-chariot server..."

# Pull latest image if using ACR
if [ "$1" == "pull" ]; then
    echo "📦 Pulling latest image from registry..."
    docker-compose pull
fi

echo "🔄 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to start..."
sleep 10

echo "🏥 Checking service health..."
docker-compose ps

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Service Status:"
echo "  Go-Chariot: http://localhost:8087"
echo "  Nginx: http://localhost:80 (redirects to HTTPS)"
echo ""
echo "📋 Useful commands:"
echo "  View logs: docker-compose logs -f"
echo "  Stop services: docker-compose down"
echo "  Restart: docker-compose restart"
