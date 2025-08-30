FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create app user
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy pre-built binary
COPY services/go-chariot/build/go-chariot-linux-amd64 /app/go-chariot
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