# Configuration

Settings come from defaults, an optional YAML file (`--config` or
`REPUTECAST_CONFIG`), environment variables and flags, in that order.
Environment variables use the `REPUTECAST_` prefix and a double underscore
between levels (`http.trusted_proxies` → `REPUTECAST_HTTP__TRUSTED_PROXIES`). The public URL
has one name everywhere: `REPUTECAST_PUBLIC_URL`.

## Settings

| Setting | Default | Notes |
|---|---|---|
| `REPUTECAST_DATABASE__URL` (or `DATABASE_URL`) | `postgres://reputecast:reputecast@localhost:5432/reputecast?sslmode=disable` | Postgres 16–18. |
| `REPUTECAST_DATABASE__MAX_CONNS` | `20` | Pool size per process. |
| `REPUTECAST_HTTP__ADDR` | `:8080` | Listen address. |
| `REPUTECAST_PUBLIC_URL` | `http://localhost:8080` | The address people and providers reach. (`REPUTECAST_HTTP__PUBLIC_URL`, its earlier name, still works when `REPUTECAST_PUBLIC_URL` isn't set.) Used for webhooks, SSO redirects, passkeys, public pages and links in emails. Use `https://` in production. An address saved in Instance admin › Instance & updates replaces it for links, pages and webhooks (moving provider webhooks with it); HTTPS-only cookies and HSTS follow this setting, so keep the two the same. |
| `REPUTECAST_HTTP__PUBLIC_URL_LOCKED` | `false` | Pins the public URL to `REPUTECAST_PUBLIC_URL`: an address saved in Instance admin is ignored, the field there and at the first sign-in is read only, and the API refuses another address (`409`, code `operator_locked`). At start the locked address becomes the saved one and provider webhooks move to it, audited as the server configuration's; webhooks that providers keep in their own settings are listed in the log. So changing the locked address and restarting moves the instance. For a hosting company or an IT team that runs the instance for someone else and owns its DNS and TLS; `reputecast doctor` warns when it pins an address only the server reaches. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | — | Comma-separated CIDRs or IPs of your reverse proxies; only then is `X-Forwarded-For` trusted. Allowed IPs on API keys and MCP agents (Settings › Keys & MCP) compare the client address ReputeCast resolves with this setting: behind a proxy that isn't listed, every call seems to come from the proxy, so list it before relying on allowed IPs (the dialog warns while untrusted proxies are seen). The SMTP relay checks the connecting address itself. |
| `REPUTECAST_HTTP__RATE_LIMIT_AUTH` | `30` | Sign-in requests per minute per client IP (on top of per-email lockout). |
| `REPUTECAST_HTTP__RATE_LIMIT_API` | `600` | API requests per minute: per API key, per signed-in browser session, and per IP for calls with neither. |
| `REPUTECAST_HTTP__RATE_LIMIT_IP` | `6000` | API requests per minute from signed-in browsers at one address together (an office behind one NAT shares it; each session still has `RATE_LIMIT_API`). Limits are counted by each web process: behind a load balancer with N web processes, a caller can reach N times them. |
| `REPUTECAST_SECRETS__KEY` | — | Base64 32-byte key that encrypts provider credentials, OIDC secrets and TOTP seeds. If unset, `REPUTECAST_SECRETS__KEY_FILE` is used. |
| `REPUTECAST_SECRETS__KEY_FILE` | `data/secret.key` (`/var/lib/reputecast/secret.key` in Docker) | Created with a fresh key on first start. **Back it up with the database**; without it, stored credentials can't be decrypted. |
| `REPUTECAST_SECRETS__PREVIOUS_KEYS` | — | Old keys during a rotation (values sealed with them still open). After `reputecast secrets reseal` reports nothing left on a key, wait a day and remove it ([rotating the key](backup-restore.md#rotating-the-secret-key)). |
| `REPUTECAST_FILES__DIR` | `files` next to the secret key file (`data/files`, `/var/lib/reputecast/files` in Docker) | Uploaded files: digital products and lead magnets. They stay out of the database, so back the folder up, and give every process the same folder when you split roles. |
| `REPUTECAST_I18N__PSEUDO_LOCALE` | `false` | Offers "Pseudo (+40%)" in Language & region outside development, to check layouts with longer text on a staging instance. |
| `REPUTECAST_ROLES` | `web,worker,ingest` | Same as `--role`. |
| `REPUTECAST_WORKER__CONCURRENCY` | `50` | Concurrent jobs per worker process. |
| `REPUTECAST_LOG__LEVEL` / `__FORMAT` | `info` / `text` | `debug…error`; `json` for log shippers. |
| `REPUTECAST_METRICS__ENABLED` / `__ADDR` | `true` / `:9464` | Prometheus metrics at `/metrics` on their own listener, which also answers `/healthz` and `/readyz`. Keep that port off the internet (the Docker image doesn't publish it). Empty `__ADDR` serves no separate listener, except in a worker-only process, which always opens it for health checks. |
| `REPUTECAST_METRICS__TOKEN` | — | At least 16 characters. Also serves `/metrics` on the main listener, to a request with `Authorization: Bearer <token>`; without it the main listener answers `/metrics` with 404. |
| `REPUTECAST_UPDATES__CHECK` | `true` | Daily check of GitHub releases (only the version number is fetched). Can also be turned off in Instance admin. ReputeCast never sends usage data anywhere. |
| `REPUTECAST_UPDATES__MANAGED_BY` | — | Who upgrades this instance, when someone runs it for its users (`Northwind Hosting`). Instance admin › Instance & updates then says upgrades are managed by them instead of showing the `docker compose` command and the backup line. |
| `REPUTECAST_AUTH__COOKIE_SECURE` | `true` | Secure session cookie (always on when the public URL is https). |
| `REPUTECAST_PROVIDERS__ALLOW_PRIVATE_NETWORKS` | `false` | Let provider URLs entered in a workspace (a Postal server, a generic HTTP adapter) and community adapters' addresses (a sidecar; plain `http://` only there) resolve to loopback or private addresses, for providers on your own network. Cloud metadata and link-local addresses are always refused. |
| `REPUTECAST_WEBHOOKS__ALLOW_PRIVATE_NETWORKS` | `false` | The same for outgoing webhooks and the audit stream. |
| `REPUTECAST_ADMIN_EMAIL` / `_NAME` / `_PASSWORD` | — | Used by `install` and `serve --auto-install`. Without a password a temporary one is printed. |
| `REPUTECAST_AUTO_MIGRATE`, `REPUTECAST_AUTO_INSTALL` | `false` | Container conveniences (the Docker image enables both). |

## Docker Compose variables

[`deploy/docker-compose.yml`](../../deploy/docker-compose.yml) reads these from
a `.env` file next to it or from your shell:

| Variable | Default | Notes |
|---|---|---|
| `REPUTECAST_PUBLIC_URL` | `http://localhost:8080` | Passed to ReputeCast as it is. |
| `REPUTECAST_ADMIN_EMAIL` | `admin@example.com` | The first Owner, created on the first start. |
| `POSTGRES_PASSWORD` | `reputecast` | Set it before the first start: Postgres only applies it when the database volume is created. |
| `REPUTECAST_PORT` | `8080` | Host port. |
| `REPUTECAST_IMAGE` / `REPUTECAST_VERSION` | `ghcr.io/hostrepute/reputecast` / `latest` | Pin a release tag such as `v1.0.0` in production. |

## Settings in the app

Sign-in policies (methods, password length, breached-password check,
two-factor enforcement and grace period, session lifetime and idle timeout,
lockout, SSO and social providers) are managed in **Instance admin ›
Authentication**, not in configuration.
