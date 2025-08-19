#!/bin/sh

# Minimal entrypoint script for go-chariot (no SSL generation)
set -e

echo "Starting go-chariot..."

# Seed default data into mounted volume if empty
if [ ! -d "/app/data" ] || [ -z "$(ls -A /app/data 2>/dev/null)" ]; then
    if [ -d "/app/data.default" ]; then
        echo "Seeding /app/data from /app/data.default..."
        mkdir -p /app/data
        cp -r /app/data.default/* /app/data/ || true
    fi
fi

# If bootstrap is missing but a default exists, copy it without modifying content
if [ ! -f "/app/data/bootstrap.ch" ] && [ -f "/app/data.default/bootstrap.ch" ]; then
    echo "Installing missing bootstrap.ch into /app/data..."
    cp /app/data.default/bootstrap.ch /app/data/bootstrap.ch 2>/dev/null || true
fi

# Ensure tree directory and essential agent files exist
if [ ! -d "/app/data/trees" ]; then
    echo "Creating /app/data/trees directory..."
    mkdir -p /app/data/trees || true
fi

# If usersAgent.json (or any default trees) are missing, copy from defaults
if [ -d "/app/data.default/trees" ]; then
    if [ ! -f "/app/data/trees/usersAgent.json" ]; then
        echo "Seeding missing tree files from /app/data.default/trees..."
        cp -r /app/data.default/trees/* /app/data/trees/ 2>/dev/null || true
    fi
fi

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
