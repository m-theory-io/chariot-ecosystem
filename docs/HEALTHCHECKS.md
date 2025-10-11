# Chariot Ops Health Checks

This guide lists quick checks to verify the stack is healthy after a deploy and to diagnose routing/auth issues.

## External quick checks (from your laptop)

- Nginx alive over TLS
  - curl -Ik https://chariot.m-theory.io/health should return 200 and a minimal body.
- Visual DSL landing and assets content-types
  - HTML page: curl -Ik https://chariot.m-theory.io/visual-dsl/ → Content-Type: text/html
  - SVG favicon: curl -Ik https://chariot.m-theory.io/visual-dsl/visual_dsl_favicon.svg → Content-Type: image/svg+xml
  - If assets return text/html instead of their types, nginx is likely not stripping the /visual-dsl prefix before proxying.

Note: API and editors are behind OAuth; external unauthenticated calls may redirect to login. Use internal checks below to bypass auth.

## Internal checks (inside the nginx container on the VM)

Run these on the VM in the compose folder.

- Exec into nginx container
  - docker compose -f docker-compose.azure.yml exec nginx sh

- Verify visual-dsl upstream resolves and is reachable
  - getent hosts visual-dsl || nslookup visual-dsl || ping -c1 visual-dsl
  - curl -sS -I http://visual-dsl:80/health (expect 200)

- Verify go-chariot health (bypasses OAuth)
  - curl -sS -f http://go-chariot:8087/health

- Verify charioteer health (bypasses OAuth)
  - curl -sS -I http://charioteer:8080/charioteer/health (expect 200)

- Confirm nginx config contains the rewrite for /visual-dsl
  - grep -n "location /visual-dsl/" /etc/nginx/nginx.conf
  - grep -n "rewrite ^/visual-dsl/(.*)$ /$1 break;" /etc/nginx/nginx.conf
  - grep -n "proxy_pass http://\$visual_dsl_upstream:80\$uri;" /etc/nginx/nginx.conf

- Reload nginx (if you edited the config in-place)
  - nginx -t && nginx -s reload

## OAuth2 proxy sanity checks (internal)

- From nginx container, the auth endpoint should be reachable and return 401 without cookies:
  - curl -sS -o /dev/null -w "%{http_code}\n" http://oauth2-proxy:4180/oauth2/auth → 401
- Check oauth2-proxy logs for errors (in another terminal):
  - docker compose -f docker-compose.azure.yml logs -f oauth2-proxy

## Pull/restart after config or image updates

- Pull and restart only nginx and visual-dsl:
  - docker compose -f docker-compose.azure.yml pull nginx visual-dsl
  - docker compose -f docker-compose.azure.yml up -d nginx visual-dsl

- Pull everything and restart only nginx (depends_on ensures ordering):
  - docker compose -f docker-compose.azure.yml pull
  - docker compose -f docker-compose.azure.yml up -d nginx

## Common symptoms and fixes

- Symptom: 502 on /visual-dsl/
  - Cause: Stale upstream DNS in nginx after container restart
  - Fix: We use resolver 127.0.0.11 and variable-based proxying; ensure the running image tag includes this. Pull latest nginx:amd64 and restart.

- Symptom: Visual DSL assets (JS/CSS/SVG) return text/html
  - Cause: Upstream receives unstripped /visual-dsl path and serves index.html
  - Fix: Ensure the nginx location block contains:
    - rewrite ^/visual-dsl/(.*)$ /$1 break;
    - proxy_pass http://$visual_dsl_upstream:80$uri;
  - After updating image, pull and restart nginx.

- Symptom: OAuth redirects loop or 403
  - Check that oauth2-proxy is running and reachable: curl http://oauth2-proxy:4180/ping (200) or /oauth2/auth (401 without cookie)
  - Verify cookie secret and Azure env vars are set in docker-compose.azure.yml

## Tag conventions

- docker-compose.azure.yml defaults:
  - go-chariot: mtheorycontainerregistry.azurecr.io/go-chariot:${GO_CHARIOT_TAG:-v0.001}
  - charioteer: mtheorycontainerregistry.azurecr.io/charioteer:${CHARIOTEER_TAG:-amd64}
  - visual-dsl: mtheorycontainerregistry.azurecr.io/visual-dsl:${VISUAL_DSL_TAG:-v0.001}
  - nginx: mtheorycontainerregistry.azurecr.io/nginx:${NGINX_TAG:-amd64}
- Our build/push scripts maintain nginx:amd64 as the default alias; when you push a version (e.g., v0.026), the push script also updates nginx:amd64 to the same digest.

## Quick one-liners (optional)

- Check content-type for favicon (expect image/svg+xml):
  - curl -sI https://chariot.m-theory.io/visual-dsl/visual_dsl_favicon.svg | awk -F": " '/Content-Type/ {print $2}'

- Check upstream resolution from nginx container:
  - docker compose -f docker-compose.azure.yml exec nginx getent hosts visual-dsl
