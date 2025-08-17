# Multi-stage Dockerfile for Charioteer
# Go HTML template-driven web server with Monaco editor

# Stage 1: Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx

# Set working directory
WORKDIR /src

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o charioteer \
    ./cmd/charioteer

# Compress binary with UPX (optional, comment out if issues)
RUN upx --best --lzma charioteer

# Stage 2: Production stage
FROM alpine:latest AS production

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    openssl

# Create non-root user
RUN addgroup -g 1001 charioteer && \
    adduser -D -s /bin/sh -u 1001 -G charioteer charioteer

# Set working directory
WORKDIR /app

# Copy SSL generation script
COPY docker/charioteer/generate-ssl.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/generate-ssl.sh

# Copy configuration files
COPY --chown=charioteer:charioteer config/ ./config/
COPY --chown=charioteer:charioteer templates/ ./templates/
COPY --chown=charioteer:charioteer static/ ./static/

# Copy binary from builder
COPY --from=builder --chown=charioteer:charioteer /src/charioteer ./

# Create required directories
RUN mkdir -p /app/logs /app/ssl /app/uploads && \
    chown -R charioteer:charioteer /app

# Copy entrypoint script
COPY docker/charioteer/entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER charioteer

# Expose ports
EXPOSE 8080 8443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Stage 3: Development stage
FROM golang:1.21-alpine AS development

# Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    openssl \
    make \
    gcc \
    musl-dev

# Install air for hot reload
RUN go install github.com/cosmtrek/air@latest

# Install other useful development tools
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install golang.org/x/tools/cmd/goimports@latest

# Create non-root user for development
RUN addgroup -g 1001 charioteer && \
    adduser -D -s /bin/sh -u 1001 -G charioteer charioteer

# Set working directory
WORKDIR /app

# Copy air configuration
COPY docker/charioteer/.air.toml ./

# Create required directories
RUN mkdir -p /app/logs /app/ssl /app/uploads && \
    chown -R charioteer:charioteer /app

# Switch to non-root user
USER charioteer

# Expose ports (including debugger port)
EXPOSE 8080 8443 2345

# Default command for development
CMD ["air", "-c", ".air.toml"]
