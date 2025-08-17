# Multi-stage Dockerfile for go-chariot (moved from services/go-chariot/Dockerfile)
# Go + Echo RESTful service

# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx

# Set working directory
WORKDIR /src

# Copy source code and modules together including vendor directory
COPY . .

# Build the application using vendor dependencies (offline build)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -mod=vendor \
    -buildvcs=false \
    -ldflags='-w -s -extldflags "-static"' \
    -o go-chariot \
    ./cmd

# Compress binary with UPX (optional, comment out if issues)
# RUN upx --best --lzma go-chariot

# Stage 2: Production stage
FROM alpine:latest AS production

# Install runtime dependencies including Azure CLI
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    wget \
    openssl \
    python3 \
    py3-pip \
    bash \
    gcc \
    musl-dev \
    linux-headers \
    python3-dev

# Install Azure CLI
RUN pip3 install azure-cli --break-system-packages

# Clean up build dependencies to reduce image size
RUN apk del gcc musl-dev linux-headers python3-dev

# Create non-root user with matching host UID
RUN addgroup -g 1000 gochariot && \
    adduser -D -s /bin/sh -u 1000 -G gochariot gochariot

# Set working directory
WORKDIR /app

# Create necessary directories
RUN mkdir -p /app/config /app/logs /app/ssl /app/data /home/gochariot/.azure && \
    chown -R gochariot:gochariot /app /home/gochariot/.azure

# Copy binary from build stage
COPY --from=builder /src/go-chariot /app/

# Copy data files including bootstrap.ch
COPY --chown=gochariot:gochariot data/ /app/data/

# Copy entrypoint script
COPY --chown=gochariot:gochariot entrypoint.sh /app/
RUN chmod +x /app/entrypoint.sh

# Switch to non-root user
USER gochariot

# Expose port 9080
EXPOSE 9080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:9080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Default command
CMD ["/app/go-chariot"]
