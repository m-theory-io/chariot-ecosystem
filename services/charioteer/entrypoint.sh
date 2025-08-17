#!/bin/sh

# Entrypoint script for Charioteer
set -e

echo "Starting Charioteer..."

# Create SSL directory in a writable location
SSL_DIR="/tmp/ssl"
mkdir -p "$SSL_DIR"

# Generate SSL certificates if they don't exist
if [ ! -f "$SSL_DIR/server.crt" ]; then
    echo "Generating SSL certificates..."
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$SSL_DIR/server.key" \
        -out "$SSL_DIR/server.crt" \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
    
    # Copy certificates with the expected names for charioteer
    cp "$SSL_DIR/server.crt" "$SSL_DIR/charioteer.crt"
    cp "$SSL_DIR/server.key" "$SSL_DIR/charioteer.key"
    
    echo "SSL certificates generated in $SSL_DIR"
else
    echo "SSL certificates already exist"
fi

# Create symlinks to expected locations if needed
if [ -d "/app/ssl" ] && [ -w "/app/ssl" ]; then
    ln -sf "$SSL_DIR/server.key" "/app/ssl/server.key" || true
    ln -sf "$SSL_DIR/server.crt" "/app/ssl/server.crt" || true
fi

# Wait for go-chariot dependency
if [ "$NODE_ENV" = "production" ]; then
    echo "Waiting for go-chariot dependency..."
    
    # Wait for go-chariot service
    echo "Waiting for go-chariot..."
    until curl -f "http://localhost:8087/health" 2>/dev/null; do
        echo "go-chariot is unavailable - sleeping"
        sleep 5
    done
    echo "go-chariot is up!"
fi

# Run database migrations if needed
if [ -f "./migrate" ]; then
    echo "Running database migrations..."
    ./migrate || echo "Migration failed or not needed"
fi

# Start the application
echo "Starting Charioteer server..."
exec /app/charioteer -ssl=false
