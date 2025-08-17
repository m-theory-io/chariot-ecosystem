#!/bin/sh

# Entrypoint script for go-chariot
set -e

echo "Starting go-chariot..."

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

# Wait for database dependencies if in production
if [ "$GO_ENV" = "production" ]; then
    echo "Waiting for database dependencies..."
    
    # Wait for Couchbase
    if [ -n "$COUCHBASE_URL" ]; then
        echo "Waiting for Couchbase..."
        until curl -f "http://localhost:8091/pools" 2>/dev/null; do
            echo "Couchbase is unavailable - sleeping"
            sleep 5
        done
        echo "Couchbase is up!"
    fi
    
    # Wait for MySQL
    if [ -n "$MYSQL_HOST" ]; then
        echo "Waiting for MySQL..."
        until nc -z "$MYSQL_HOST" "${MYSQL_PORT:-3306}" 2>/dev/null; do
            echo "MySQL is unavailable - sleeping"
            sleep 5
        done
        echo "MySQL is up!"
    fi
fi

# Run database migrations if needed  
if [ -f "./migrate" ]; then
    echo "Running database migrations..."
    ./migrate || echo "Migration failed or not needed"
fi

# Start the application (CERT_PATH environment variable will be used)
echo "Starting go-chariot API server..."
exec /app/go-chariot