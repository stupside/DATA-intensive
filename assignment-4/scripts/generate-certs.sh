#!/bin/bash
# Generate certificates for TLS/mTLS setup
# - CA certificate (Certificate Authority)
# - Server certificate (for TLS)

set -e

CERTS_DIR="${1:-./.certs}"

echo "=== Certificate Generation Script ==="
echo "Output directory: $CERTS_DIR"
echo ""

# Create certs directory if it doesn't exist
mkdir -p "$CERTS_DIR"

# 1. Generate CA Certificate (if it doesn't exist)
if [ -f "$CERTS_DIR/ca.crt" ] && [ -f "$CERTS_DIR/ca.key" ]; then
    echo "[CA] Certificate already exists, skipping..."
else
    echo "[CA] Generating Certificate Authority..."

    # Generate CA private key (Ed25519)
    openssl genpkey -algorithm ED25519 -out "$CERTS_DIR/ca.key"

    # Generate self-signed CA certificate (valid for 10 years)
    openssl req -new -x509 -key "$CERTS_DIR/ca.key" -out "$CERTS_DIR/ca.crt" \
        -days 3650 \
        -subj "/CN=Assignment 4 CA/O=LUT/OU=Certificate Authority" \
        -addext "basicConstraints=critical,CA:TRUE" \
        -addext "keyUsage=critical,keyCertSign,cRLSign"

    echo "[CA] Generated: $CERTS_DIR/ca.crt, $CERTS_DIR/ca.key"
fi

echo ""

# 2. Generate Server Certificate
if [ -f "$CERTS_DIR/server.crt" ] && [ -f "$CERTS_DIR/server.key" ]; then
    echo "[SERVER] Certificate already exists, skipping..."
else
    echo "[SERVER] Generating server certificate..."

    # Generate server private key (Ed25519)
    openssl genpkey -algorithm ED25519 -out "$CERTS_DIR/server.key"

    # Create server CSR
    openssl req -new -key "$CERTS_DIR/server.key" -out "$CERTS_DIR/server.csr" \
        -subj "/CN=localhost/O=LUT/OU=Assignment 4 Server"

    # Sign server certificate with CA (valid for 2 years)
    openssl x509 -req -in "$CERTS_DIR/server.csr" \
        -CA "$CERTS_DIR/ca.crt" -CAkey "$CERTS_DIR/ca.key" \
        -CAcreateserial \
        -out "$CERTS_DIR/server.crt" \
        -days 730 \
        -extfile <(printf "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth")

    # Clean up CSR
    rm "$CERTS_DIR/server.csr"

    echo "[SERVER] Generated: $CERTS_DIR/server.crt, $CERTS_DIR/server.key"
fi

echo ""
echo "=== Certificate Generation Complete ==="
echo ""
echo "Generated certificates:"
echo "  CA:     $CERTS_DIR/ca.crt, $CERTS_DIR/ca.key"
echo "  Server: $CERTS_DIR/server.crt, $CERTS_DIR/server.key"
echo ""
echo "Usage:"
echo "  - CA:     Used to sign server and device certificates"
echo "  - Server: Used by the server for TLS"
echo ""
echo "Client distribution:"
echo "  - Bundle ca.crt with clients for server verification"
echo ""
echo "IMPORTANT: Keep ca.key and server.key secure!"
