# Production Dockerfile for visual-dsl - Cross-platform build
# Multi-stage build for optimized production image

FROM node:20-alpine AS builder

# Use monorepo root as build context and copy needed folders
WORKDIR /repo

# Copy package manifests for dependency resolution
COPY services/visual-dsl/package*.json services/visual-dsl/
COPY packages/chariot-codegen/package*.json packages/chariot-codegen/

# Copy codegen package source so local file dependency is available
COPY packages/chariot-codegen/ packages/chariot-codegen/

# Build the local shared package so its prepare script can run during dependency install
WORKDIR /repo/packages/chariot-codegen
RUN npm ci && npm run build

# Install Visual DSL deps (will install local file: ../../packages/chariot-codegen)
WORKDIR /repo/services/visual-dsl
RUN npm ci

# Copy Visual DSL source
COPY services/visual-dsl/ .

# Build the application with proper base path for nginx proxy
ENV VITE_BASE_PATH=/visual-dsl/
RUN npm run build

# Production stage - lightweight nginx
FROM nginx:alpine AS production

# Copy built application
COPY --from=builder /repo/services/visual-dsl/dist /usr/share/nginx/html

# Create nginx config using echo commands (VS Code friendly)
RUN echo 'server {' > /etc/nginx/conf.d/default.conf && \
    echo '    listen 80;' >> /etc/nginx/conf.d/default.conf && \
    echo '    server_name localhost;' >> /etc/nginx/conf.d/default.conf && \
    echo '    root /usr/share/nginx/html;' >> /etc/nginx/conf.d/default.conf && \
    echo '    index index.html;' >> /etc/nginx/conf.d/default.conf && \
    echo '' >> /etc/nginx/conf.d/default.conf && \
    echo '    # SPA routing - serve index.html for all routes' >> /etc/nginx/conf.d/default.conf && \
    echo '    location / {' >> /etc/nginx/conf.d/default.conf && \
    echo '        try_files $uri $uri/ /index.html;' >> /etc/nginx/conf.d/default.conf && \
    echo '    }' >> /etc/nginx/conf.d/default.conf && \
    echo '' >> /etc/nginx/conf.d/default.conf && \
    echo '    # Health check endpoint' >> /etc/nginx/conf.d/default.conf && \
    echo '    location /health {' >> /etc/nginx/conf.d/default.conf && \
    echo '        access_log off;' >> /etc/nginx/conf.d/default.conf && \
    echo '        return 200 "OK";' >> /etc/nginx/conf.d/default.conf && \
    echo '        add_header Content-Type text/plain;' >> /etc/nginx/conf.d/default.conf && \
    echo '    }' >> /etc/nginx/conf.d/default.conf && \
    echo '' >> /etc/nginx/conf.d/default.conf && \
    echo '    # Security headers' >> /etc/nginx/conf.d/default.conf && \
    echo '    add_header X-Frame-Options "SAMEORIGIN" always;' >> /etc/nginx/conf.d/default.conf && \
    echo '    add_header X-Content-Type-Options "nosniff" always;' >> /etc/nginx/conf.d/default.conf && \
    echo '    add_header X-XSS-Protection "1; mode=block" always;' >> /etc/nginx/conf.d/default.conf && \
    echo '}' >> /etc/nginx/conf.d/default.conf

# Health check using wget (available in alpine)
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:80/health || exit 1

# Expose port
EXPOSE 80

# Run nginx
CMD ["nginx", "-g", "daemon off;"]