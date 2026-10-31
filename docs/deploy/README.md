# Self-hosting ReputeCast

ReputeCast is a single binary that embeds the web app and its database
migrations. It needs Postgres 16, 17 or 18 and nothing else.

- [Install with Docker](#install-with-docker)
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
  https://raw.githubusercontent.com/hostrepute/reputecast/main/deploy/docker-compose.yml
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
| `reputecast user reset-password --email …` | Prints a one-time reset link. Works before any sending provider exists. |
| `reputecast user reset-2fa --email … --reason …` | Removes the authenticator app; the reason goes to the audit log. |
| `reputecast user make-admin\|unlock --email …` | Sets the instance admin flag, or clears a sign-in lockout. |
| `reputecast secrets generate-key` | Prints a new encryption key. |
| `reputecast adapter install\|set-url\|secret\|check\|list\|remove` | Installs a community adapter from its manifest, tells ReputeCast where its process runs and issues its signing secret ([adapters guide](../adapters.md#community-adapters)). |
| `reputecast openapi [--yaml]` | Prints the OpenAPI 3.1 document. |
| `reputecast seed` | Loads the Acme demo data (development only). |
| `reputecast demo send "<campaign>" [--scenario …] [--elapsed …] [--reset]` | Sends a demo campaign through simulated providers (development only; see [CONTRIBUTING.md](../../CONTRIBUTING.md#live-sending-in-the-demo)). |
| `reputecast version` | Prints the version. |

## Upgrades

- Docker: `docker compose pull && docker compose up -d`. Migrations run at
  start.
- Binary: replace it, then run `reputecast migrate up`.

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

## Scaling

One process runs everything. For more throughput, run several
`reputecast serve --role=web,ingest` instances behind the proxy and one or more
`reputecast worker` processes. They coordinate through Postgres (River) and
need the same uploaded files folder.

## Health checks and metrics

- `/healthz`: liveness.
- `/readyz`: the database is reachable.
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

Several processes started with `--auto-migrate` take turns: one runs the
migrations while the others wait for it (up to 15 minutes). Migrations need
a direct or session-mode connection, not PgBouncer in transaction mode.
