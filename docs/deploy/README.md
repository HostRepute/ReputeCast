# Self-hosting ReputeCast

ReputeCast is a single binary that embeds the web app and its database
migrations. It needs Postgres 16, 17 or 18 and nothing else.

- [Install with Docker](#install-with-docker)
- [One-click installs](#one-click-installs)
- [Kubernetes (Helm)](#kubernetes-helm)
- [Install the binary](#install-the-binary)
- [First sign-in](#first-sign-in)
- [Commands](#commands)
- [Upgrades](#upgrades)
- [Backups](#backups)
- [Scaling](#scaling)
- [Health checks and metrics](#health-checks-and-metrics)

More guides: [Configuration](configuration.md) ·
[Reverse proxy and TLS](reverse-proxy.md) · [Postgres](postgres-tuning.md)

## Install with Docker

You need Docker with Compose. The compose file runs Postgres and ReputeCast:

```bash
curl -fsSL -o docker-compose.yml \
  https://github.com/hostrepute/reputecast/releases/latest/download/docker-compose.yml
```

Create a `.env` file next to it before the first start:

```bash
REPUTECAST_PUBLIC_URL=https://news.example.com   # the address people and providers reach
REPUTECAST_ADMIN_EMAIL=you@example.com           # the first Owner
POSTGRES_PASSWORD=a-long-random-password         # used when the database is first created
```

Then start it and read the first Owner's temporary password from the logs:

```bash
docker compose up -d
docker compose logs app | grep -A4 "ReputeCast is installed"
```

Open the public URL (or <http://localhost:8080> while you try it locally).
The compose file also reads `REPUTECAST_PORT` (default `8080`) and
`REPUTECAST_VERSION` (default `latest`; pin a release tag such as `v1.0.0` in
production).

Images are published to `ghcr.io/hostrepute/reputecast` with each release. To
run your own build instead, build the image from a checkout and point the
compose file at it:

```bash
make docker VERSION=local
REPUTECAST_IMAGE=reputecast REPUTECAST_VERSION=local REPUTECAST_ADMIN_EMAIL=you@example.com \
  docker compose -f deploy/docker-compose.yml up -d
```

## One-click installs

Each of these runs one ReputeCast instance from the release image, with
Postgres beside it. The first start applies the migrations and creates the
first Owner from the email you enter; their temporary password is in the
service's logs (search for `ReputeCast is installed`).

| Platform | Start from | What it needs |
|---|---|---|
| Render | [Deploy to Render](https://render.com/deploy?repo=https://github.com/hostrepute/reputecast) ([`render.yaml`](../../render.yaml)) | A paid web service (the 10 GB disk at `/var/lib/reputecast`) and a Render Postgres database. Render asks for the public URL and the first Owner's email. |
| Railway | [`deploy/railway`](../../deploy/railway/README.md): the template, three steps by hand, or the Railway CLI | Railway Postgres and a volume at `/var/lib/reputecast`. The public URL follows the domain Railway generates. |
| DigitalOcean | [Deploy to DigitalOcean](https://cloud.digitalocean.com/apps/new?repo=https://github.com/hostrepute/reputecast/tree/main) ([`.do/deploy.template.yaml`](../../.do/deploy.template.yaml)) | App Platform has no disks: a Spaces bucket and access key for uploaded files, and a secret key you generate (`openssl rand -base64 32`). The button creates a dev database; convert it to a managed one in the app's settings before you rely on it. |
| Coolify | [`deploy/coolify/reputecast.yaml`](../../deploy/coolify/reputecast.yaml) in a **Docker Compose Empty** resource | A Coolify server with a domain. Coolify generates the URL and the Postgres password; you enter the first Owner's email. |
| Kubernetes | [Helm chart](#kubernetes-helm) | Kubernetes 1.29+, Postgres 16–18 (CloudNativePG or a managed database) and an Ingress controller. |
| Docker Compose | [Install with Docker](#install-with-docker) | Docker 24+ with Compose and a domain you control. |

All of them set `REPUTECAST_HTTP__TRUSTED_PROXIES=private`, which trusts the
platform's router on private networks ([configuration](configuration.md)).
When you move to your own domain, change `REPUTECAST_PUBLIC_URL` (Render,
Coolify), add the domain in Railway (its variable follows) or in
DigitalOcean (`${APP_URL}` follows), then confirm it in Instance admin ›
Instance & updates. Back up the secret key with the database: the
`secret.key` file on the volume, or the `REPUTECAST_SECRETS__KEY` you entered
on DigitalOcean.

## Kubernetes (Helm)

The chart in [`deploy/helm/reputecast`](../../deploy/helm/reputecast/README.md)
is published to `oci://ghcr.io/hostrepute/charts/reputecast` with each
release. It runs one StatefulSet pod with a volume (`mode: single`), or web,
worker and webhook pods that share an S3-compatible bucket (`mode: split`).
Migrations run in a Job before every install and upgrade.

```bash
helm install reputecast oci://ghcr.io/hostrepute/charts/reputecast \
  --namespace reputecast --create-namespace \
  --set publicURL=https://news.example.com \
  --set database.existingSecret=reputecast-db-app \
  --set admin.email=you@example.com \
  --set ingress.enabled=true
kubectl logs --namespace reputecast job/reputecast-install   # the temporary password
```

The chart's README covers Postgres with CloudNativePG or a managed database,
split mode and the bucket, the generated secret key, the SMTP relay's
LoadBalancer, GeoIP, metrics, network policy and every value.

## Install the binary

Download the archive for your platform (Linux, macOS or Windows; amd64 or
arm64) from the [releases](https://github.com/hostrepute/reputecast/releases),
point it at Postgres and run:

```bash
export REPUTECAST_DATABASE__URL=postgres://reputecast:secret@localhost:5432/reputecast?sslmode=disable
export REPUTECAST_PUBLIC_URL=https://news.example.com
reputecast install --email you@example.com --name "Your Name"   # migrates and creates the first Owner
reputecast serve
```

## First sign-in

At first sign-in you replace the temporary password, add a passkey or an
authenticator app, save your recovery codes, confirm the public URL and create
your first workspace. ReputeCast then walks you through connecting a sending
provider: DNS records are checked live, a test message is sent, and the
provider is assigned to streams.

Providers deliver webhooks to `<public URL>/hooks/…`, so the public URL must be
reachable from the internet. See [Reverse proxy and TLS](reverse-proxy.md).

## Commands

| Command | What it does |
|---|---|
| `reputecast serve` | Runs the web app, API, webhook ingest and workers. `--role=web,worker,ingest` splits them across processes. `--auto-migrate` and `--auto-install` are for containers. |
| `reputecast worker` | Runs only background jobs (same as `serve --role=worker`). |
| `reputecast install` | Applies migrations and creates the first Owner (an instance admin with break-glass sign-in). |
| `reputecast migrate up\|down\|status` | Manages the schema. |
| `reputecast migrate partition-events` | Splits `delivery_events` into monthly partitions without copying it, ahead of the upgrade that would ([partitioning](partitioning.md)). |
| `reputecast migrate segments-role` | Hands the SQL segments function back to its role after a restore on a new server ([backup and restore](backup-restore.md)). |
| `reputecast doctor` | Checks the database, schema, secret key, files folder and roles; run it after a restore or a move. |
| `reputecast healthcheck [--live]` | Asks the local process's `/readyz` (or `/healthz`); exits 1 when it isn't ready. For container health checks. |
| `reputecast user reset-password --email …` | Prints a one-time reset link. Works before any sending provider exists. |
| `reputecast user reset-2fa --email … --reason …` | Removes the authenticator app; the reason goes to the audit log. |
| `reputecast user make-admin\|unlock --email …` | Sets the instance admin flag, or clears a sign-in lockout. |
| `reputecast secrets generate-key` | Prints a new encryption key. |
| `reputecast secrets reseal [--dry-run]` | Re-encrypts every stored secret with the current key, so a previous key can be dropped. |
| `reputecast adapter install\|set-url\|secret\|check\|list\|remove` | Installs a community adapter from its manifest, tells ReputeCast where its process runs and issues its signing secret ([adapters guide](../adapters.md#community-adapters)). |
| `reputecast openapi [--yaml]` | Prints the OpenAPI 3.1 document. |
| `reputecast seed` | Loads the Acme demo data (development only). |
| `reputecast demo send "<campaign>" [--scenario …] [--elapsed …] [--reset]` | Sends a demo campaign through simulated providers (development only; see [CONTRIBUTING.md](../../CONTRIBUTING.md#live-sending-in-the-demo)). |
| `reputecast version` | Prints the version. |

## Upgrades

- Docker: `docker compose pull && docker compose up -d`. Migrations run at
  start.
- Binary: replace it, then run `reputecast migrate up`.
- Helm: `helm upgrade` with the new chart version; a Job migrates before the
  pods change (give a large upgrade `--timeout 30m`).
- Render, Railway, DigitalOcean: redeploy the service to pull the image
  again. Coolify: set `REPUTECAST_VERSION` to the new tag and redeploy.
  Migrations run at start.

The upgrade to the release with review 2026-11 converts `delivery_events` to
monthly partitions if it isn't already; on a very large table run
`reputecast migrate partition-events` first, at a quiet hour
([partitioning](partitioning.md)).

SNS messages are trusted only from a provider's own topic, because SNS
signs every AWS account's messages alike. Amazon SES connections have had
theirs saved since they were connected. An AWS End User Messaging (SMS)
provider whose webhook was set up by hand in AWS keeps, at the upgrade, the
topic its verified events came from. If they came from several topics, or
none came yet, its receipts and replies are refused until SNS confirms the
subscription again: in the SNS console, delete the HTTPS subscription to
ReputeCast's address and create it again (or use Register again on the
provider, when its API can register webhooks). The first confirmation
keeps its topic.

WhatsApp's callback URL ends with a token of the account's own,
`<public URL>/hooks/whatsapp/<workspace>/<token>`, so nobody can post to it
from the workspace's name alone. A WhatsApp account connected from this
release on has only that address. One connected before keeps its old
address, `<public URL>/hooks/whatsapp/<workspace>`, until Meta calls the new
one, so nothing stops at the upgrade. To move it, open Sending providers ›
WhatsApp as someone who can manage providers: a note shows the new callback
URL to copy. In Meta for Developers › your app › WhatsApp › Configuration,
paste it as the Callback URL (the verify token stays the same) and click
Verify and save. From Meta's first call to the new URL, its verification
request or a signed delivery, the old address refuses the account's
deliveries and the Events tab counts them. New callback URL, on the
Connection card, replaces the token if a URL leaked: the previous one stops
at once, and the account waits for Meta's verification request at the new
one. Rolling back to an earlier release after the move needs the old
address pasted in Meta again: earlier releases don't know the new one.

A process whose database is missing migrations it needs serves read-only:
pages and the API read as usual with a banner ("ReputeCast is being
upgraded"), writes answer 503 `maintenance`, provider webhooks are still
stored, workers wait, and `/readyz` answers 503 so a load balancer keeps
using the previous release. It carries on by itself once `reputecast migrate
up` (or another process's `--auto-migrate`) has run. Downgrades are covered
in [backup and restore](backup-restore.md#downgrades).

Read the [release notes](https://github.com/hostrepute/reputecast/releases)
first. Instance admin › Instance & updates shows the running version and the
latest release.

## Backups

Back up Postgres (`pg_dump`) **and** the secret key: `REPUTECAST_SECRETS__KEY`,
or the key file (`data/secret.key`, or `/var/lib/reputecast/secret.key` in
Docker). Restoring needs both, because provider credentials and other secrets
are encrypted with that key. Back up the uploaded files folder as well
(`REPUTECAST_FILES__DIR`, `files` next to the key file): digital products and
lead magnets live there, not in the database.

[Backup and restore](backup-restore.md) has the restore runbook (including
the role SQL segments run as, which `pg_dump` doesn't carry), key rotation
and the downgrade policy.

## Scaling

[Performance](performance.md) has timings on a million contacts and how
to measure your own; [partitioning](partitioning.md) splits the largest
table by month.

One process runs everything. For more throughput, run several
`reputecast serve --role=web,ingest` instances behind the proxy and one or more
`reputecast worker` processes. They coordinate through Postgres (River) and
need the same uploaded files folder.

## Health checks and metrics

- `/healthz`: liveness.
- `/readyz`: ready to work: the database answers, its schema has every
  migration this release needs and, in a worker, the workers run. A
  worker-only process answers both on the metrics listener (`:9464`).
  `reputecast healthcheck` asks them for container health checks; the
  Docker image and `deploy/docker-compose.yml` use it.
- `/metrics`: Prometheus metrics on their own listener, `:9464` by default
  (`REPUTECAST_METRICS__ADDR`). Don't publish that port or proxy it. To
  scrape the main listener instead, set `REPUTECAST_METRICS__TOKEN` and send
  it as a bearer token:

  ```yaml
  scrape_configs:
    - job_name: reputecast
      authorization:
        credentials: <the token>
      static_configs:
        - targets: ["news.example.com:443"]
      scheme: https
  ```

  Without a token the main listener answers `/metrics` with 404. Earlier
  versions served `/metrics` openly on the main listener: when you upgrade,
  point Prometheus at port 9464 or set a token.

Besides the HTTP and Go runtime metrics, every process exports:

| Metric | What it measures |
|---|---|
| `reputecast_jobs{queue,kind,state}` | Background jobs waiting, scheduled, retrying, running or discarded. |
| `reputecast_job_queue_lag_seconds{queue}` | How long the oldest due job has waited. Alert when it grows. |
| `reputecast_job_failures_total{kind,outcome}` | Failed attempts: `retry`, `discarded` (out of attempts) or `panic`. |
| `reputecast_provider_messages_total{provider,result}` | Messages handed to providers: `accepted`, `rejected`, `deferred`, `failed`. |
| `reputecast_provider_calls_total{provider,result}` | Send calls: `ok` or `error`. |
| `reputecast_ingest_requests_total{channel,result}` | Provider webhooks received: `stored`, `rejected` (signature), `unknown`, `invalid`, `retry_later`. |
| `reputecast_ingest_pending{channel}`, `reputecast_ingest_lag_seconds{channel}` | Stored webhook events not normalized yet, and the oldest one's age. |
| `reputecast_webhook_deliveries_total{result}` | Outgoing webhook attempts: `delivered`, `retrying`, `dead`. |
| `reputecast_webhook_deliveries_pending`, `reputecast_webhook_deliveries_lag_seconds` | Outgoing webhooks waiting, and how long the oldest due one has. |
| `reputecast_db_pool_*` | The database pool: connections by state, size, acquires, waits. |
| `reputecast_metrics_scrape_errors` | Queue readings that failed at the last scrape. |

Queue readings come from the database at most every 15 seconds, whatever
the scrape interval. Log lines of a request carry its `request_id`, `route`,
`workspace` and `caller` (`user:<id>`, `key:<id>` or `agent:<id>`); a job's
carry `job` (`kind:id`).

Several processes started with `--auto-migrate` take turns: one runs the
migrations while the others wait for it (up to 15 minutes). Migrations need
a direct or session-mode connection, not PgBouncer in transaction mode.
