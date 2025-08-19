# Two-lane auth architecture

**Lane A — Public (OSS users, explorers)**

* Keep it **first-party**: your backend does auth.
* Prefer **passwordless**: passkeys (WebAuthn) + email magic links; fall back to passwords only if you must.
* Issue **short-lived session cookies** for the IDE/site and **JWTs** (or PASETO) for APIs/CLI.
* Don’t expose implicit flow—use **Authorization Code + PKCE** for browser apps (Monaco IDE).
* Baseline hardening: Argon2id for hashes, TOTPs/WebAuthn for step-up, CSRF (SameSite=Lax + double-submit), device + IP throttling.

**Lane B — Customers (early adopters, branded apps)**

* Support **bring-your-own-IdP** via **OIDC and SAML**. Each customer’s admin connects their Entra/Okta/Google/ADFS directly to *your* SP/Client.
* This **does not require Microsoft publisher verification** because the customer is doing an explicit enterprise federation (their admin config, their tenant). No public self-consent involved.
* Optional extras per customer: **SCIM** for user/group provisioning, **SAML/OIDC group → role** mapping, **custom domains** and branding.

# Minimal data model (multi-tenant)

* `organizations(id, name, …)`
* `users(id, email, …)`
* `memberships(user_id, org_id, role, …)`
* `identities(id, user_id, provider, subject, attrs_json)`  ← stores local, OIDC, SAML links
* `sessions(id, user_id, org_id, last_seen, user_agent, …)`
* `api_tokens(id, user_id|org_id, scope, expires_at, …)`

# Flows to implement

**Public IDE / website**

1. Browser → `/oauth/authorize` (PKCE)
2. Backend → code exchange → session cookie (HttpOnly, Secure, SameSite=Lax)
3. IDE uses session cookie; API calls carry session or a short JWT.

**Customer SSO**

1. Org admin enters IdP metadata (OIDC `.well-known` or SAML XML).
2. You validate + activate provider at the **org** level.
3. On “Sign in with <Customer>”, route to that provider; map claims → `user` + `membership`.
4. (Optional) SCIM sync creates users/groups ahead of time.

# Self-hosted IdP option (keeps “public” easy & cheap)

If you want more features without rolling everything yourself, drop in one of these and still avoid MS/Okta fees:

* **Keycloak** (battle-tested; heavier, but full OIDC/SAML/SCIM, theming, policies)
* **authentik** (lighter, modern admin UX, OIDC/SAML, good for per-tenant providers)
* **Ory** (Hydra/Kratos) if you like modular, code-first control

You can:

* Use the IdP for **Lane A** (local users) and **Lane B** (brokering customer SSO).
* Or keep **Lane A** fully custom and use the IdP only as a broker for **Lane B**.

# Go-centric pointers (since your stack is Go)

* **OIDC Relying Party**: `go-oidc` (+ `golang.org/x/oauth2`) with PKCE.
* **JWT/JWKS**: `github.com/golang-jwt/jwt/v5` + JWKS caching, or PASETO.
* **WebAuthn**: `github.com/go-webauthn/webauthn` (passkeys).
* **Sessions**: `github.com/alexedwards/scs` (cookie store or Redis).
* **Argon2id**: `github.com/alexedwards/argon2id`.

# Security & UX niceties

* **BFF pattern** for the IDE: the browser talks to your backend, which holds tokens; the browser only gets a session cookie.
* **Incremental & dynamic consent** (for customer IdPs) by scoping claims tightly.
* **Per-org config**: store provider config under `organizations`; render the correct buttons dynamically.
* **Machine-to-machine**: client-credentials tokens for your code-gen/runner services; rotate keys via JWKS.

# What to do next (practical)

1. Implement **Lane A** now: passkeys + magic links + PKCE; lock in cookie/CSRF/CORS.
2. Add an **OIDC provider abstraction** (interface + registry).
3. Stand up **Keycloak or authentik** in dev to validate OIDC/SAML brokering and claim mapping.
4. Build the **per-tenant SSO setup wizard** (paste metadata → test login → role mapping).
5. Ship. Add SCIM and custom domains when the first customer asks.

If you want, I can sketch the exact claim → role mapping and PKCE handler skeletons for your Echo service, plus a minimal database schema migration to drop in.
