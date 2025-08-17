Short version: use NGINX’s **stream** (TCP) proxy, not the regular HTTP proxy. Here’s a clean, production-ish example that accepts connections on the host (e.g., port 3306) and forwards them to your Dockerized MySQL that’s bound to `127.0.0.1:3306`.

```nginx
# /etc/nginx/nginx.conf  (or drop this in /etc/nginx/streams-enabled/mysql.conf)
worker_processes auto;

events {}

stream {
  # Optional: separate access log for DB traffic
  # log_format basic '$remote_addr:$remote_port -> $server_addr:$server_port status=$status bytes=$bytes_sent';
  # access_log /var/log/nginx/mysql_access.log basic;

  upstream mysql_backend {
    # Your container is published to the HOST's loopback: 127.0.0.1:3306
    server 127.0.0.1:3306;
    # If you ever add replicas:
    # least_conn;
    # server 127.0.0.1:3306 max_fails=3 fail_timeout=10s;
  }

  server {
    # Listen on all interfaces; choose a port (3306 if you’re replacing, 3307 if coexisting)
    listen 0.0.0.0:3306;

    # Connection handling tuned for long-lived MySQL sessions
    proxy_connect_timeout 5s;
    proxy_timeout         1h;   # don't drop idle connections too aggressively
    proxy_pass mysql_backend;

    # Optional coarse IP ACL at the edge
    # allow 10.0.0.0/8;
    # allow 192.168.0.0/16;
    # deny all;
  }
}
```

### Gotchas & tips (important)

1. **Use the stream module (TCP), not HTTP.**
   Make sure your NGINX has `--with-stream` (most distro packages do). Check with `nginx -V`.

2. **Where NGINX runs matters.**

   * If NGINX runs **on the host**, `server 127.0.0.1:3306;` works (it reaches the port your container published to loopback with `-p 127.0.0.1:3306:3306`).
   * If NGINX runs **in another container**, `127.0.0.1` points to **that container**, not the host. Use one of:

     * `--add-host=host.docker.internal:host-gateway` and then `server host.docker.internal:3306;`
     * or run NGINX with `network_mode: host`
     * or publish MySQL on the Docker bridge IP and point to that.

3. **Client IP visibility (PROXY protocol).**
   By default MySQL sees connections coming **from NGINX**, not the real client.

   * If you rely on `user@host` grants or want the real client IP, enable **PROXY protocol** pass-through:

     ```nginx
     stream {
       upstream mysql_backend { server 127.0.0.1:3306; }

       server {
         listen 0.0.0.0:3306;
         proxy_pass mysql_backend;
         proxy_connect_timeout 5s;
         proxy_timeout 1h;
         proxy_protocol on;  # send PROXY header to MySQL
       }
     }
     ```
   * Then, in **MySQL 8**, allow PROXY protocol from NGINX’s IP (example my.cnf):

     ```
     [mysqld]
     proxy_protocol_networks=127.0.0.1/32,10.0.0.0/8
     ```

     Without this, MySQL will reject the connection that starts with a PROXY header.

4. **TLS/SSL considerations.**

   * If **MySQL requires TLS** (`require_secure_transport=ON`), do **TLS passthrough**: keep NGINX as a pure TCP forwarder (no `listen ... ssl;`), and let the MySQL server negotiate TLS directly with clients.
   * If you terminate TLS at NGINX (`listen ... ssl;` with certs), the backend sees **plaintext**, which breaks `require_secure_transport` and pushes certificate enforcement to NGINX—not typical for MySQL. Passthrough is the safer default.

5. **Authentication plugin quirks.**
   MySQL 8’s default `caching_sha2_password` prefers TLS; non-TLS clients may need extra flags to fetch the RSA key. With NGINX stream passthrough + TLS to MySQL, clients behave normally.

6. **Health checks & load balancing.**
   OSS NGINX lacks active TCP health checks (that’s an NGINX Plus feature). Failover only kicks in on connection errors. If you need MySQL-aware health checks, consider **HAProxy** (has `mysql-check`) or **MaxScale**.

7. **Firewalling and exposure.**
   Publishing MySQL behind NGINX doesn’t absolve firewalling: still restrict who can reach port 3306 (security groups, iptables/ufw, cloud firewall). Use NGINX `allow/deny` for another layer.

8. **Routing by MySQL username/database (advanced).**
   NGINX has `ngx_stream_mysql_preread_module` that can peek at the initial MySQL handshake to route by `$mysql_user` / `$mysql_db`. Only needed for multi-tenant fan-out; otherwise ignore.

9. **Grants may need adjusting.**
   If you’re **not** using PROXY protocol, MySQL sees `host='127.0.0.1'`. Ensure your users have grants like `'user'@'127.0.0.1'` (or `'user'@'localhost'` if appropriate). With PROXY protocol configured, you can keep `'user'@'client-net'`.

10. **Logs and debugging.**
    Enable a `stream` access log temporarily to troubleshoot connectivity and see byte counts. For MySQL-level errors, still check the MySQL error log.

---

If you share how you’re running NGINX (host vs container) and which port you want publicly, I can tailor the exact snippet (including IP ACLs and, if needed, PROXY protocol + the minimal MySQL settings) to your setup.
