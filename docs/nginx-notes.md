Got it — “connection refused” on the client side almost always means NGINX isn’t actually listening on that port/IP (or a local security layer blocked the bind). Here’s a tight checklist to find/fix the usual culprits with `stream {}`.

# Fix-it fast: 8 checks

1. **Is `stream` config actually loaded?**
   Many distros only `include` files **inside `http {}`**, so a `stream` server in `/etc/nginx/conf.d/*.conf` gets ignored.

   * Run: `nginx -T | sed -n '1,120p'` and look for your `stream { ... }` block in the full dump.
   * If it’s missing, add a top-level include (outside `http {}`) in `/etc/nginx/nginx.conf`:

     ```nginx
     # top-level (same level as events {})
     include /etc/nginx/streams-enabled/*.conf;
     ```

     Put your MySQL proxy file in that folder.

2. **Is the module present?**

   * Run: `nginx -V 2>&1 | tr ' ' '\n' | grep -i stream`
     You want to see `--with-stream` (or a dynamic module path).
   * If your build uses dynamic modules, load it at the very top of `nginx.conf`:

     ```nginx
     load_module modules/ngx_stream_module.so;
     # if you’ll ever do TLS preread:
     # load_module modules/ngx_stream_ssl_preread_module.so;
     ```

3. **Is NGINX binding the port/IP you expect?**

   * Run: `ss -ltnp | grep 3306`
     If nothing shows, NGINX isn’t listening there.
     In your `stream` server, be explicit:

     ```nginx
     server {
       listen 0.0.0.0:3306;     # add also: listen [::]:3306; if using IPv6
       proxy_pass 127.0.0.1:3306;
     }
     ```

     Then `nginx -s reload` (or `systemctl reload nginx`) and check `ss` again.

4. **Port conflict or wrong address?**
   You can listen on `0.0.0.0:3306` even if MySQL is bound to `127.0.0.1:3306` (different IPs are fine).
   But if you tried `listen 127.0.0.1:3306;`, that **will** collide with MySQL. Use `0.0.0.0:3306` or another port (e.g., `3307`) for NGINX.

5. **Firewall / security policy blocking the bind or traffic?**

   * **Ubuntu/UFW:** `sudo ufw status` (allow the port or your source CIDRs)
   * **RHEL/CentOS/SELinux:**

     * Allow httpd/nginx to bind nonstandard ports:
       `sudo semanage port -a -t http_port_t -p tcp 3306` (or `-m` if exists)
     * Allow outbound connects to MySQL:
       `sudo setsebool -P httpd_can_network_connect 1`
       If SELinux blocks the bind, NGINX error log will show `permission denied`.

6. **Are you running NGINX inside Docker?**

   * If yes, publish the listener: `-p 3306:3306`.
   * And make the **backend** reachable from the NGINX container (loopback in a container ≠ host loopback):

     * Prefer: `--add-host=host.docker.internal:host-gateway` and then `proxy_pass host.docker.internal:3306;`
     * Or run NGINX with `--network host` (Linux only), then `proxy_pass 127.0.0.1:3306;`.

7. **Did the reload actually take effect?**

   * `nginx -t` (should be OK)
   * `nginx -s reload`
   * Check errors: `tail -n 100 /var/log/nginx/error.log` (or your distro path).
     Look for messages like “unknown directive ‘stream’”, “permission denied”, or “bind() to 0.0.0.0:3306 failed”.

8. **Sanity test without NGINX** (proves the path to MySQL is good):
   From the same host, try:
   `socat -v TCP-LISTEN:3307,reuseaddr,fork TCP:127.0.0.1:3306`
   Then connect your MySQL client to `127.0.0.1:3307`. If that works, the network path/back-end are fine; focus on NGINX config/load.

# Minimal known-good config

Put this in **a file that is actually included at top-level**, e.g. `/etc/nginx/streams-enabled/mysql.conf`:

```nginx
# Make sure ngx_stream_module is present (see step 2).

stream {
  upstream mysql_backend {
    server 127.0.0.1:3306;   # MySQL bound to loopback on the host
  }

  server {
    listen 0.0.0.0:3306;     # change to 3307 if you prefer
    proxy_connect_timeout 5s;
    proxy_timeout         1h;
    proxy_pass mysql_backend;
    # Optional access log for debugging:
    # access_log /var/log/nginx/mysql_access.log;
  }
}
```

Then:

```bash
sudo nginx -t
sudo nginx -s reload
ss -ltnp | grep 3306
```

# Extra gotchas that look like “refused” in practice

* **IPv6-only bind**: If you wrote `listen 3306;` and your system defaulted to IPv6 only, IPv4 clients see “refused.” Fix by adding `listen 0.0.0.0:3306;` (and optionally `listen [::]:3306;`).
* **AppArmor/SELinux**: blocks the bind silently except for error.log entries.
* **Wrong include order**: a later `stream {}` is overridden by an earlier file that closes the block unexpectedly. `nginx -T` reveals the true merged config.
* **Proxying from a container to host loopback**: `127.0.0.1` from inside the NGINX container is **not** the host. Use `host.docker.internal` (with `--add-host`), host networking, or place both containers on a user-defined Docker network and point at the MySQL container name/port on that network instead of loopback.

---

If you want, paste the output of:

* `nginx -T | sed -n '1,200p'` (just the top where includes show),
* `ss -ltnp | grep -E '3306|3307'`,
* and the last 50 lines of `error.log`,

and I’ll pinpoint exactly which of the above is biting you.
