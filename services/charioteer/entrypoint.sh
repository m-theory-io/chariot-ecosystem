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

# Ensure volume directory exists and permissions are correct (needs root)
if [ "$(id -u)" = "0" ]; then
    mkdir -p /app/files
    # If mounted with root-only perms, fix ownership
    chown -R 1001:1001 /app/files || true
fi

# Seed default files into mounted volume if empty
if [ -d "/app/files.default" ] && [ -z "$(ls -A /app/files 2>/dev/null)" ]; then
    echo "Seeding /app/files from /app/files.default..."
    cp -r /app/files.default/* /app/files/ 2>/dev/null || cp -r /app/files.default/. /app/files/ || true
    # Ensure ownership is charioteer
    if [ "$(id -u)" = "0" ]; then
        chown -R 1001:1001 /app/files || true
    fi
fi

# Wait for go-chariot dependency
if [ "$NODE_ENV" = "production" ]; then
    echo "Waiting for go-chariot dependency..."
    
    # Wait for go-chariot service
    echo "Waiting for go-chariot..."
    until curl -f "http://go-chariot:8087/health" 2>/dev/null; do
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

# Start the application (drop privileges if running as root)
echo "Starting Charioteer server..."

# Build the final command
if [ "$#" -gt 0 ]; then
    echo "Launching with provided args: $*"
    set -- "$@"
else
    BACKEND_URL="${CHARIOT_BACKEND_URL:-http://go-chariot:8087}"
    echo "Launching with defaults: -ssl=false -backend=${BACKEND_URL}"
    set -- /app/charioteer -ssl=false -backend="${BACKEND_URL}"
fi

if [ "$(id -u)" = "0" ]; then
    if command -v su-exec >/dev/null 2>&1; then
        exec su-exec 1001:1001 "$@"
    elif command -v gosu >/dev/null 2>&1; then
        exec gosu 1001:1001 "$@"
    else
        echo "su-exec/gosu not found; running as root (not recommended)"
        exec "$@"
    fi
else
    exec "$@"
fi
