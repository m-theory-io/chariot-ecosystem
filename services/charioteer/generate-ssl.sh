#!/bin/sh

# Generate SSL certificates for Charioteer
set -e

CERT_DIR="/app/ssl"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"

# Create SSL directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Check if certificates already exist
if [ -f "$CERT_FILE" ] && [ -f "$KEY_FILE" ]; then
    echo "SSL certificates already exist for Charioteer, skipping generation..."
    exit 0
fi

echo "Generating SSL certificates for Charioteer..."

# Generate private key
openssl genrsa -out "$KEY_FILE" 2048

# Generate certificate signing request
openssl req -new -key "$KEY_FILE" -out "$CERT_DIR/server.csr" -subj "/C=US/ST=State/L=City/O=Chariot/OU=Charioteer/CN=charioteer.chariot.local"

# Generate self-signed certificate
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" -signkey "$KEY_FILE" -out "$CERT_FILE" \
    -extensions v3_req -extfile <(
cat <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = charioteer.chariot.local
DNS.2 = localhost
DNS.3 = *.chariot.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# Clean up CSR file
rm "$CERT_DIR/server.csr"

# Set proper permissions
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"

echo "SSL certificates generated successfully for Charioteer!"
echo "Certificate: $CERT_FILE"
echo "Private Key: $KEY_FILE"
