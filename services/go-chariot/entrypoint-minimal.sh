#!/bin/sh

# Minimal entrypoint script for go-chariot (no SSL generation)
set -e

echo "Starting go-chariot..."

# Only generate SSL certificates if SSL is enabled
if [ "${CHARIOT_SSL:-false}" = "true" ]; then
    echo "SSL enabled - generating certificates..."
    
    # Check if openssl is available
    if ! command -v openssl >/dev/null 2>&1; then
        echo "ERROR: SSL enabled but openssl not available in container"
        exit 1
    fi
    
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
        echo "SSL certificates generated in $SSL_DIR"
    else
        echo "SSL certificates already exist"
    fi
    
    # Create symlinks to expected locations if needed
    if [ -d "/app/ssl" ] && [ -w "/app/ssl" ]; then
        ln -sf "$SSL_DIR/server.key" "/app/ssl/server.key" || true
        ln -sf "$SSL_DIR/server.crt" "/app/ssl/server.crt" || true
    fi
else
    echo "SSL disabled - skipping certificate generation"
fi

# Start the application directly
echo "Starting go-chariot API server..."
exec /app/go-chariot
