FROM golang:1.24-bookworm AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src/services/go-chariot

COPY services/go-chariot/go.mod go.mod
COPY services/go-chariot/go.sum go.sum
RUN go mod download

COPY services/go-chariot/ .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=1
RUN mkdir -p /out && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/go-chariot ./cmd

FROM debian:bookworm-slim AS production

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata wget libstdc++6 && \
    rm -rf /var/lib/apt/lists/*

# Create app user
RUN groupadd --gid 1001 appgroup && \
    useradd --uid 1001 --gid appgroup --shell /bin/bash --create-home appuser

# Set working directory
WORKDIR /app

# Copy built binary from builder stage
COPY --from=builder /out/go-chariot /app/go-chariot
RUN chmod +x /app/go-chariot

# Copy entrypoint and data
COPY infrastructure/docker/go-chariot/entrypoint-minimal.sh /app/entrypoint-minimal.sh
RUN chmod +x /app/entrypoint-minimal.sh
COPY services/go-chariot/data/ /app/data.default/

# Fix ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8087

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8087/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/entrypoint-minimal.sh"]