# Multi-stage Dockerfile for Charioteer (moved from services/charioteer/Dockerfile)
# Go HTML template-driven web server with Monaco editor

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

# Copy go mod files first for better caching
COPY services/charioteer/go.mod ./

# Download dependencies (if any)
RUN go mod download && go mod verify

# Copy source code
COPY services/charioteer/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o charioteer \
    .

# Compress binary with UPX (optional, comment out if issues)
# RUN upx --best --lzma charioteer

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

# Create necessary directories
RUN mkdir -p /app/config /app/logs /app/ssl /app/files && \
    chown -R charioteer:charioteer /app

# Copy binary from build stage
COPY --from=builder /src/charioteer /app/

# Copy application files
COPY --chown=charioteer:charioteer services/charioteer/files/ /app/files/

# Copy entrypoint script
COPY --chown=charioteer:charioteer services/charioteer/entrypoint.sh /app/
RUN chmod +x /app/entrypoint.sh

# Switch to non-root user
USER charioteer

# Expose port 8080
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Default command
CMD ["/app/charioteer"]
