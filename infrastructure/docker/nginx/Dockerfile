# Dedicated Nginx Ingress Controller for Chariot Ecosystem
FROM nginx:alpine

# Install openssl for SSL certificate generation
RUN apk add --no-cache openssl

# Create SSL directory and generate self-signed certificate
RUN mkdir -p /etc/nginx/ssl && \
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/server.key \
        -out /etc/nginx/ssl/server.crt \
        -subj "/C=US/ST=CA/L=San Francisco/O=Chariot/OU=Development/CN=localhost"

# Copy landing page as index.html
COPY landing.html /usr/share/nginx/html/index.html

# Copy nginx configuration (replacing main config)
COPY nginx.conf /etc/nginx/nginx.conf

# Remove default nginx files and configurations
RUN rm -f /usr/share/nginx/html/index.nginx-debian.html && \
    rm -f /etc/nginx/conf.d/default.conf

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
