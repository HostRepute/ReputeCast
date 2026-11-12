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
| `PORT` | — | Set by platforms such as Render, Railway, Heroku and Cloud Run: ReputeCast listens on `:<PORT>` when `REPUTECAST_HTTP__ADDR` isn't set (`--addr` and `REPUTECAST_HTTP__ADDR` win). The Docker image sets `REPUTECAST_HTTP__ADDR=:8080`, so a container listens on 8080 whatever `PORT` says; the one-click templates set `PORT=8080`, which tells the platform where to route. A `PORT` that isn't a port number (1–65535) stops the start. |
| `REPUTECAST_PUBLIC_URL` | `http://localhost:8080` | The address people and providers reach. (`REPUTECAST_HTTP__PUBLIC_URL`, its earlier name, still works when `REPUTECAST_PUBLIC_URL` isn't set.) Used for webhooks, SSO redirects, passkeys, public pages and links in emails. Use `https://` in production. An address saved in Instance admin › Instance & updates replaces it for links, pages and webhooks (moving provider webhooks with it); HTTPS-only cookies and HSTS follow this setting, so keep the two the same. |
| `REPUTECAST_HTTP__PUBLIC_URL_LOCKED` | `false` | Pins the public URL to `REPUTECAST_PUBLIC_URL`: an address saved in Instance admin is ignored, the field there and at the first sign-in is read only, and the API refuses another address (`409`, code `operator_locked`). At start the locked address becomes the saved one and provider webhooks move to it, audited as the server configuration's; webhooks that providers keep in their own settings are listed in the log. So changing the locked address and restarting moves the instance. For a hosting company or an IT team that runs the instance for someone else and owns its DNS and TLS; `reputecast doctor` warns when it pins an address only the server reaches. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | — | Comma-separated CIDRs or IPs of your reverse proxies, or `private`; only then is `X-Forwarded-For` trusted. `private` stands for every private (RFC 1918), shared (`100.64.0.0/10`) and loopback network, IPv6 unique local and loopback included: the router in front of ReputeCast on Render, Railway, Coolify, DigitalOcean App Platform or a Kubernetes Ingress controller, whose address you can't know in advance. Don't use it when clients can reach ReputeCast directly from a private network (an office LAN): they could claim any address. Entries that are none of these are logged and ignored. Allowed IPs on API keys and MCP agents (Settings › Keys & MCP) compare the client address ReputeCast resolves with this setting: behind a proxy that isn't listed, every call seems to come from the proxy, so list it before relying on allowed IPs (the dialog warns while untrusted proxies are seen). The SMTP relay checks the connecting address itself. |
| `REPUTECAST_HTTP__RATE_LIMIT_AUTH` | `30` | Sign-in requests per minute per client IP (on top of per-email lockout). |
| `REPUTECAST_HTTP__RATE_LIMIT_API` | `600` | API requests per minute: per API key, per signed-in browser session, and per IP for calls with neither. |
| `REPUTECAST_HTTP__RATE_LIMIT_IP` | `6000` | API requests per minute from signed-in browsers at one address together (an office behind one NAT shares it; each session still has `RATE_LIMIT_API`). Limits are counted by each web process: behind a load balancer with N web processes, a caller can reach N times them. |
| `REPUTECAST_SECRETS__KEY` | — | Base64 32-byte key that encrypts provider credentials, OIDC secrets and TOTP seeds. If unset, `REPUTECAST_SECRETS__KEY_FILE` is used. |
| `REPUTECAST_SECRETS__KEY_FILE` | `data/secret.key` (`/var/lib/reputecast/secret.key` in Docker) | Created with a fresh key on first start. **Back it up with the database**; without it, stored credentials can't be decrypted. |
| `REPUTECAST_SECRETS__PREVIOUS_KEYS` | — | Old keys during a rotation (values sealed with them still open). After `reputecast secrets reseal` reports nothing left on a key, wait a day and remove it ([rotating the key](backup-restore.md#rotating-the-secret-key)). |
| `REPUTECAST_FILES__DIR` | `files` next to the secret key file (`data/files`, `/var/lib/reputecast/files` in Docker) | The files folder: digital products' files, lead magnets, uploads in progress and audience exports, with `REPUTECAST_STORAGE__BACKEND` `local` (the default). They stay out of the database, so back the folder up. Several web or worker processes either share this folder or keep the files in a bucket ([file storage](#file-storage)). |
| `REPUTECAST_GEOIP__DATABASE` | — (off) | A MaxMind DB file — GeoLite2-City or GeoLite2-Country (free MaxMind account), or DB-IP Lite City or Lite Country — that places the opens and clicks of people in a campaign's report by country and city, and lets a signup form show only in some countries (the visitor's country is looked up for that one answer and not kept). ReputeCast never downloads it: you download it, replace it to update it and restart. A City database takes about its file size in memory. Only the country code and the English city name are kept, on a message's first human open or click, and they are cleared with the reader's IP by retention and erasure; the IP itself isn't stored for this. `reputecast doctor` shows the file's type and build date. |
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

## File storage

The files ReputeCast keeps outside the database (digital products' files,
lead magnets, uploads in progress, audience exports) live in the files
folder or in an S3-compatible bucket: AWS S3, MinIO, Cloudflare R2,
DigitalOcean Spaces, Backblaze B2, Wasabi. A bucket lets several web and
worker processes (replicas on Kubernetes, an install split with `--role`)
share the files without a ReadWriteMany volume. Downloads still go through
ReputeCast either way, so download limits, expiry, counting and resumed
downloads work the same.

| Setting | Default | Notes |
|---|---|---|
| `REPUTECAST_STORAGE__BACKEND` (`storage.backend`) | `local` | `local`: the files folder (`REPUTECAST_FILES__DIR`). `s3`: the bucket below. |
| `REPUTECAST_STORAGE__S3__BUCKET` (`storage.s3.bucket`) | — | Required with `s3`. ReputeCast doesn't create the bucket. |
| `REPUTECAST_STORAGE__S3__REGION` (`storage.s3.region`) | — | The bucket's region. Empty: the AWS environment's (`AWS_REGION`, the shared config file), else `us-east-1`, which S3-compatible services accept. |
| `REPUTECAST_STORAGE__S3__ENDPOINT` (`storage.s3.endpoint`) | — (AWS) | The service's URL for a provider other than AWS: `http://minio:9000`, `https://<account>.r2.cloudflarestorage.com`, `https://nyc3.digitaloceanspaces.com`, `https://s3.eu-central-003.backblazeb2.com`. With an endpoint, ReputeCast sends request checksums only where S3 requires them, as several services don't take the AWS SDK's default ones. |
| `REPUTECAST_STORAGE__S3__PATH_STYLE` (`storage.s3.path_style`) | `false` | `true` puts the bucket in the path (`http://minio:9000/<bucket>/<key>`) instead of the host name: MinIO and most self-hosted services need it. |
| `REPUTECAST_STORAGE__S3__PREFIX` (`storage.s3.prefix`) | — | Put every key under it (`reputecast/`), to share a bucket. Not starting with `/`. |
| `REPUTECAST_STORAGE__S3__ACCESS_KEY_ID` / `__SECRET_ACCESS_KEY` (`storage.s3.access_key_id` / `secret_access_key`) | — | Static keys: set both, or neither for the default AWS credential chain (environment variables, the shared credentials file, an EC2 or ECS role, IRSA, EKS Pod Identity). Never shown in Instance admin. |
| `REPUTECAST_STORAGE__S3__SSE` (`storage.s3.sse`) | — (the bucket's default) | Server-side encryption of what ReputeCast writes: `AES256` (SSE-S3) or `aws:kms` (SSE-KMS). |
| `REPUTECAST_STORAGE__S3__KMS_KEY_ID` (`storage.s3.kms_key_id`) | — | The KMS key for `aws:kms`; empty uses the account's default key for S3. |

The credentials need `s3:ListBucket` on the bucket and `s3:GetObject`,
`s3:PutObject`, `s3:DeleteObject` and `s3:AbortMultipartUpload` on its keys
(under the prefix); with `aws:kms`, also `kms:GenerateDataKey` and
`kms:Decrypt` on the key. Without `s3:ListBucket`, S3 answers "access
denied" instead of "not found" for a file that isn't there. Files over
16 MB go up in 8 MB parts; a lifecycle rule that aborts incomplete
multipart uploads after a day clears the parts of a process stopped
mid-upload. `reputecast doctor` checks the bucket answers and takes, reads
back and deletes a test object under `.doctor/`; Instance admin › System
shows the bucket, its endpoint and prefix, and what it holds.

### Moving the files folder to a bucket

1. Create the bucket and set `REPUTECAST_STORAGE__S3__*`; leave
   `REPUTECAST_STORAGE__BACKEND` at `local` for now.
2. `reputecast storage migrate --to s3 --dry-run` counts the folder's files
   and those the bucket doesn't hold yet.
3. `reputecast storage migrate --to s3` copies every file under the same
   key, reads each copy back to check its SHA-256, and skips files the
   bucket holds already at the same size. It never deletes local files, and
   is safe to run while ReputeCast runs and to run again; it exits 1 when a
   file failed (named on stderr), which the next run copies.
4. Set `REPUTECAST_STORAGE__BACKEND=s3`, run the copy once more for what was
   uploaded meanwhile, and restart every process. An upload or an export
   in progress during the switch starts again. Keep the folder until
   downloads work from the bucket.

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
