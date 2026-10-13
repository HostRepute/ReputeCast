<p align="center">
  <img src="docs/assets/readme-banner.png" alt="ReputeCast — the newsletter platform that protects your inbox placement" width="100%">
</p>

<p align="center">
  <b>Open source · self-hosted · sends through your own Amazon SES, Postmark, Mailgun, SendGrid, Resend, Postal or any SMTP server</b>
</p>

<p align="center">
  <a href="https://github.com/hostrepute/reputecast/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/hostrepute/reputecast/actions/workflows/ci.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="License: AGPL-3.0" src="https://img.shields.io/badge/license-AGPL--3.0-0F766E"></a>
</p>

---

ReputeCast is a newsletter and email-marketing platform you run yourself. It
sends through the sending provider you already trust — and it is the only
newsletter tool that **checks whether your mail will reach the inbox before
you send**, then **guards your sender reputation while it sends**, whichever
provider delivers it.

It is built by [HostRepute](https://hostrepute.com), the email-reputation
monitoring service, and it works with or without a HostRepute account.
**Every feature is free.** There is no paid edition, no seat limit, no
workspace limit and nothing behind an upgrade prompt.

## Contents

- [Features](#features)
- [Quick start](#quick-start)
- [Installing without Docker](#installing-without-docker)
- [Configuration](#configuration)
- [Running in production](#running-in-production)
- [Custom domains for public pages](#custom-domains-for-public-pages)
- [How it works](#how-it-works)
- [Security](#security)
- [API, webhooks and MCP](#api-webhooks-and-mcp)
- [Development](#development)
- [Project layout](#project-layout)
- [Roadmap](#roadmap) · [Contributing](#contributing) · [License](#license)

## Features

**Know before you send — Send Readiness.** Every campaign gets a pre-send
check with a verdict (Ready, Ready with warnings, Blocked), evidence and a fix
for each finding:

- sender reputation: domain and IP blocklists (80+ sources with HostRepute), and the bounce and complaint rates of every provider in the route against that provider's own limits (for example the Amazon SES review thresholds);
- authentication: SPF, DKIM, DMARC, return-path / MAIL FROM and BIMI, checked against the records your provider needs;
- content: spam triggers, image-to-text ratio, Gmail's 102 KB clipping limit, alt text, plain-text part;
- links: broken links and links on URI blocklists;
- compliance: one-click unsubscribe (RFC 8058), footer address, Gmail/Yahoo bulk-sender rules;
- audience quality: unengaged, recently bounced and role addresses;
- routing: a healthy provider for every stream, provider terms that allow bulk marketing, no double tracking;
- estimates: cost per provider, send duration from each provider's rate limit, throttling per mailbox provider.

**Guarded sending.** Live progress with delivered, bounced, complained,
opened and clicked counts per mailbox provider (Gmail, Outlook, Yahoo,
iCloud…) and per sending provider. When bounces or complaints cross your
thresholds the send pauses — the whole campaign or just one route — and
ReputeCast explains what tripped and offers resume, adjust, fail over or
cancel. Provider outages are handled the same way.

**Bring your own providers, several at once.** Built-in adapters for Amazon
SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure
Communication Services, self-hosted Postal and any SMTP server, plus an
adapter SDK and a generic HTTP adapter.

- Capability badges: bulk marketing allowed by the provider's terms, webhooks, domain setup by API, suppression sync, batching, dedicated IPs, sandbox, quota discovery, pricing.
- One event model: every provider's events (delivered, deferred, hard/soft bounce, complaint, rejected) map to one taxonomy that keeps the provider's original code. Providers without webhooks report through a bounce mailbox.
- One suppression list shared by every provider: a bounce at one provider stops mail to that address everywhere.
- Routing per stream (Broadcasts, Automations, System), per list or sender domain and per recipient mailbox provider, with weighted splits (90/10) for migrations and warm-up, and a route tester.
- Failover: automatic for System and Automations; Broadcasts pause and ask by default, or fail over only to a warmed-up provider.
- Warm-up planner for a new provider or dedicated IP: daily ramp, most-engaged first, automatic slowdown when bounces rise.
- Health per provider: status, quota and rate, webhook freshness, error rate and cost.

**Provider onboarding in about ten minutes.** Pick a provider, connect it
(SES through CloudFormation or IAM keys, with SNS topics and configuration
sets created for you, sandbox detection and a guided production-access
request; API key for API providers; server URL and key for Postal; host,
port, TLS, credentials and bounce mailbox for SMTP), verify DNS records
live, watch a test message arrive and assign the provider to streams.

**Composer.** A drag-and-drop email builder on GrapesJS with MJML, wrapped
in ReputeCast's own interface: blocks, layers, MJML-aware styles,
personalization with Liquid and fallbacks (`{{ first_name | default: "there" }}`),
MJML code view, device, email-client and dark-mode previews, an inbox
preview, A/B variants for subject, content and send time, inline
deliverability warnings, templates, saved sections, autosave and version
history.

**Honest analytics.** Human opens separated from machine opens (Apple Mail
Privacy Protection), a click map on the email, bounces by type with each
provider's original diagnostic code, complaints and unsubscribe reasons,
results per mailbox provider and per sending provider side by side, A/B
results with confidence, export and read-only share links.

**Audience.** Contacts with saved views, filters and bulk actions; profiles
with custom fields, per-list consent history and an activity timeline that
shows which provider delivered each email; a segment builder with AND/OR
groups, a live count and the generated SQL (and read-only SQL segments for
power users); list health with engagement tiers and a sunset policy.

**Migration that keeps consent.** Import from Brevo, Mailchimp, MailerLite,
listmonk, Keila, CSV or XLSX. Unsubscribed, bounced and complained contacts
become suppressions and are never re-subscribed; existing suppressions can be
pulled from your providers.

**Automations.** Visual flows on React Flow: triggers (joined a list, form
submitted, tag added, API event, date field), delays, emails, branches,
filters, A/B splits, webhooks, list actions and goals, with live stats per
step, draft/live/paused lifecycle and a version diff before publishing.

**Forms and public pages.** Embedded, popup and hosted signup forms with
double opt-in and spam protection; publisher-branded subscribe,
confirmation, preference-center (lists, frequency, pause) and unsubscribe
("fewer emails" option) pages; a public newsletter archive with RSS.

**Deliverability.** Health per sending domain and dedicated IP, account
health per provider, blocklist status with delisting links, Google
Postmaster Tools and Microsoft SNDS data, DMARC report summaries and an
incident timeline. Richer with [HostRepute](https://hostrepute.com), useful
without it.

**Teams, access and approvals.** Unlimited workspaces; members with
built-in or custom roles and 30+ granular permissions (sensitive ones are
flagged and require two-factor); scopes such as "Editor, Beta Program list
only"; send approvals for large or risky sends with a snapshot that is
invalidated when anything changes; passwords, passkeys, TOTP with recovery
codes, Google/Microsoft/GitHub sign-in and OIDC single sign-on (Google
Workspace, Microsoft Entra ID, Okta, Keycloak, Authentik) with
just-in-time accounts, group-to-role mapping and domain enforcement; an
immutable audit log; an instance admin area. Nobody can grant a permission
they don't hold — not to a role, a member, an API key or an AI agent.

## Quick start

You need Docker with Compose. One command starts Postgres and ReputeCast:

```bash
curl -fsSL https://raw.githubusercontent.com/hostrepute/reputecast/main/deploy/docker-compose.yml -o docker-compose.yml
REPUTECAST_ADMIN_EMAIL=you@company.com docker compose up -d
```

Then read the first Owner's temporary password from the logs and open
<http://localhost:8080>:

```bash
docker compose logs app | grep -A4 "ReputeCast is installed"
```

At first sign-in you replace the temporary password, add a passkey or an
authenticator app, save your recovery codes, confirm the public URL and
create your first workspace; then ReputeCast walks you through connecting a
sending provider.

Images are published to `ghcr.io/hostrepute/reputecast` with each release.
To run your own build instead (or before the first release), build the image
from a checkout and point the compose file at it:

```bash
make docker VERSION=local
REPUTECAST_IMAGE=reputecast REPUTECAST_VERSION=local REPUTECAST_ADMIN_EMAIL=you@company.com \
  docker compose -f deploy/docker-compose.yml up -d
```

> Want to look around first? `make dev && make seed` (see
> [Development](#development)) loads the Acme demo instance used throughout
> the design.

## Installing without Docker

ReputeCast is a single static binary (Linux, macOS, Windows; amd64 and arm64)
that embeds the web app and its migrations. Download it from the
[releases](https://github.com/hostrepute/reputecast/releases), point it at
Postgres 16, 17 or 18 and run:

```bash
export REPUTECAST_DATABASE__URL=postgres://reputecast:secret@localhost:5432/reputecast?sslmode=disable
export REPUTECAST_HTTP__PUBLIC_URL=https://news.example.com
reputecast install --email you@company.com --name "Your Name"   # migrates and creates the first Owner
reputecast serve
```

| Command | What it does |
|---|---|
| `reputecast serve` | Runs the web app, API, webhook ingest and workers. `--role=web,worker,ingest` splits them across processes. `--auto-migrate`, `--auto-install` for containers. |
| `reputecast worker` | Runs only background jobs (same as `serve --role=worker`). |
| `reputecast install` | Applies migrations and creates the first Owner (instance admin with break-glass sign-in). |
| `reputecast migrate up\|down\|status` | Manages the schema. |
| `reputecast user reset-password --email …` | Prints a one-time reset link — works before any sending provider exists. |
| `reputecast user reset-2fa --email … --reason …` | Removes the authenticator app; the reason goes to the audit log. |
| `reputecast user make-admin\|unlock --email …` | Instance admin flag, clear a sign-in lockout. |
| `reputecast secrets generate-key` | Prints a new encryption key. |
| `reputecast adapter install\|set-url\|secret\|check\|list\|remove` | Installs a community adapter from its manifest, tells ReputeCast where its process runs and issues its signing secret ([docs/adapters.md](docs/adapters.md#community-adapters)). |
| `reputecast openapi [--yaml]` | Prints the OpenAPI 3.1 document. |
| `reputecast seed` | Loads the Acme demo data (development only). |
| `reputecast version` | Prints the version. |

## Configuration

Settings come from defaults, an optional YAML file (`--config` or
`REPUTECAST_CONFIG`), environment variables and flags, in that order.
Environment variables use the `REPUTECAST_` prefix and a double underscore
between levels (`http.public_url` → `REPUTECAST_HTTP__PUBLIC_URL`).

| Setting | Default | Notes |
|---|---|---|
| `REPUTECAST_DATABASE__URL` (or `DATABASE_URL`) | `postgres://reputecast:reputecast@localhost:5432/reputecast?sslmode=disable` | Postgres 16–18. |
| `REPUTECAST_DATABASE__MAX_CONNS` | `20` | Pool size per process. |
| `REPUTECAST_HTTP__ADDR` | `:8080` | Listen address. |
| `REPUTECAST_HTTP__PUBLIC_URL` | `http://localhost:8080` | The address people and providers reach. Used for webhooks, SSO redirects, passkeys, public pages and links in emails. Use `https://` in production. An address saved in Instance admin › Instance & updates replaces it for links, pages and webhooks (moving provider webhooks with it); HTTPS-only cookies and HSTS follow this setting, so keep the two the same. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | — | Comma-separated CIDRs or IPs of your reverse proxies; only then is `X-Forwarded-For` trusted. |
| `REPUTECAST_HTTP__RATE_LIMIT_AUTH` | `30` | Sign-in requests per minute per client IP (on top of per-email lockout). |
| `REPUTECAST_HTTP__RATE_LIMIT_API` | `600` | API requests per minute per API key (per IP for browser sessions). |
| `REPUTECAST_SECRETS__KEY` | — | Base64 32-byte key that encrypts provider credentials, OIDC secrets and TOTP seeds. If unset, `REPUTECAST_SECRETS__KEY_FILE` is used. |
| `REPUTECAST_SECRETS__KEY_FILE` | `data/secret.key` (`/var/lib/reputecast/secret.key` in Docker) | Created with a fresh key on first start. **Back it up with the database**; without it, stored credentials can't be decrypted. |
| `REPUTECAST_SECRETS__PREVIOUS_KEYS` | — | Old keys during a rotation (values sealed with them still open). |
| `REPUTECAST_ROLES` | `web,worker,ingest` | Same as `--role`. |
| `REPUTECAST_WORKER__CONCURRENCY` | `50` | Concurrent jobs per worker process. |
| `REPUTECAST_LOG__LEVEL` / `__FORMAT` | `info` / `text` | `debug…error`; `json` for log shippers. |
| `REPUTECAST_METRICS__ENABLED` / `__ADDR` | `true` / main listener | Prometheus metrics at `/metrics`; set an address to serve them on a separate port. |
| `REPUTECAST_UPDATES__CHECK` | `true` | Daily check of GitHub releases (only the version number is fetched). Can also be turned off in Instance admin. ReputeCast never sends usage data anywhere. |
| `REPUTECAST_AUTH__COOKIE_SECURE` | `true` | Secure session cookie (always on when the public URL is https). |
| `REPUTECAST_PROVIDERS__ALLOW_PRIVATE_NETWORKS` | `false` | Let provider URLs entered in a workspace (a Postal server, a generic HTTP adapter) and community adapters' addresses (a sidecar; plain `http://` only there) resolve to loopback or private addresses, for providers on your own network. Cloud metadata and link-local addresses are always refused. |
| `REPUTECAST_WEBHOOKS__ALLOW_PRIVATE_NETWORKS` | `false` | The same for outgoing webhooks and the audit stream. |
| `REPUTECAST_ADMIN_EMAIL` / `_NAME` / `_PASSWORD` | — | Used by `install` and `serve --auto-install`. Without a password a temporary one is printed. |
| `REPUTECAST_AUTO_MIGRATE`, `REPUTECAST_AUTO_INSTALL` | `false` | Container conveniences (the Docker image enables both). |

Sign-in policies (methods, password length, breached-password check,
two-factor enforcement and grace period, session lifetime and idle timeout,
lockout, SSO and social providers) are managed in **Instance admin ›
Authentication**, not in configuration.

## Running in production

- **TLS and reverse proxy.** Put ReputeCast behind a TLS-terminating proxy
  (Caddy, nginx, Traefik), set `REPUTECAST_HTTP__PUBLIC_URL` to the https
  address and `REPUTECAST_HTTP__TRUSTED_PROXIES` to the proxy's address.
  Providers deliver webhooks to `<public URL>/hooks/…`, so the URL must be
  reachable from the internet; the first-run setup checks DNS, TLS and
  reachability for you.
- **Scaling.** One process runs everything. For more throughput run several
  `serve --role=web,ingest` instances behind the proxy and one or more
  `worker` processes; they coordinate through Postgres (River).
- **Backups.** Back up Postgres (`pg_dump`) **and** the secret key. Restoring
  needs both.
- **Upgrades.** `docker compose pull && docker compose up -d` (migrations run
  at start), or replace the binary and run `reputecast migrate up`. Read the
  release notes; Instance admin › Version & updates shows what changed.
- **Health.** `/healthz` (liveness) and `/readyz` (database reachable) for
  orchestrators; `/metrics` for Prometheus.

## Custom domains for public pages

Subscribers reach the public pages (subscribe, confirmation, preference
center, unsubscribe and the archive) on ReputeCast's own address under
`/p/<workspace>/…`, or on a domain of yours at the root
(`https://news.acme.io/subscribe`, `https://news.acme.io/archive`):

1. In **Forms & public pages › Branding**, enter the domain.
2. Point it at the server: a `CNAME` record to the host of
   `REPUTECAST_HTTP__PUBLIC_URL`, or `A`/`AAAA` records with the same
   addresses for an apex domain. Where neither works, add the TXT record
   `_reputecast.<domain>` with the value shown under the field.
3. ReputeCast checks DNS at once and every 10 minutes until the domain is
   verified, then daily; a verified domain survives one failed check. The
   pages answer on the domain only once it's verified.
4. Give the proxy a certificate for the domain. With Caddy, on-demand TLS
   asks ReputeCast before issuing one, so certificates exist only for
   verified domains:

   ```caddyfile
   {
   	on_demand_tls {
   		ask http://localhost:8080/p/_/tls-check
   	}
   }
   reputecast.example.com {
   	reverse_proxy localhost:8080
   }
   https:// {
   	tls {
   		on_demand
   	}
   	reverse_proxy localhost:8080
   }
   ```

   With nginx or Traefik, add the domain to a certificate and forward its
   requests with the original `Host` header.

Embed scripts and form submissions live under `/p/<workspace>/forms/…` on
either host, and links in emails keep using `REPUTECAST_HTTP__PUBLIC_URL`.

## How it works

```
                 ┌──────────────────── reputecast (single binary) ─────────────────────┐
 browser ──https──▶ web role: SPA (embedded) · REST API (Huma, OpenAPI 3.1) · public pages│
 providers ─────▶ ingest role: /hooks/<adapter>/<token> → verify → store raw → 200      │
 AI agents ─────▶ MCP server (permission-checked tools)                                  │
                 │ worker role (River on Postgres): sends, event normalization, sync,    │
                 │   health checks, warm-up, automations, imports, reports, retention    │
                 └──────────────┬───────────────────────────────┬──────────────────────┘
                                │                               │
                           PostgreSQL                 sending providers (adapters:
                    (data, jobs, audit log)            SES, Postmark, Postal, SMTP, …)
```

- **Providers are plugins.** Each adapter implements one contract
  (`internal/providers/provider.go`: capabilities, send, parse webhook, plus
  optional domain setup, webhook registration, quota and suppression sync)
  and must pass the shared contract-test suite. Core code never imports an
  adapter. Community adapters run in their own process and are called over
  a signed protocol, written with the TypeScript SDK in `sdk/`
  ([docs/adapters.md](docs/adapters.md#community-adapters)).
- **Webhooks are cheap to receive.** Ingest verifies the signature, stores
  the raw payload and answers 200 immediately; a job normalizes events later,
  so provider retries never pile up.
- **Access control is one check.** Every API operation declares the
  permission it needs (visible as `x-permission` in the OpenAPI document).
  Users, API keys and MCP agents all pass the same check in
  `internal/authz`; keys and agents can never exceed their creator.
- **Everything that changes something is audited**, in the same transaction
  as the change, with before/after values, actor, IP, user agent and request
  id. Audit entries are immutable (enforced by a database trigger).
- **The web app** (React 19, TanStack Router and Query, Tailwind 4) is built
  with Vite and embedded in the binary; API types are generated from the
  OpenAPI document.
- **Adding a provider:** [docs/adapters.md](docs/adapters.md) shows how an adapter is written and contract-tested, and how to map any HTTP API without code.

## Security

- Passwords hashed with argon2id; optional breached-password check (k-anonymity, can be turned off for offline installs).
- Sessions and API keys are random 256-bit tokens; only their SHA-256 hashes are stored. Session cookies are `HttpOnly`, `SameSite=Lax` and `Secure` on https.
- Two-factor with TOTP, passkeys (WebAuthn) and one-time recovery codes; policies can require it for everyone or for anyone holding a sensitive permission.
- Identifier-first sign-in with generic errors (no account enumeration), per-email lockout and per-IP rate limits.
- Provider credentials, OIDC client secrets, HostRepute tokens and TOTP seeds are encrypted with AES-256-GCM bound to their row, are write-only and are never returned by the API. Keys can be rotated.
- Cross-site request protection (Go's `CrossOriginProtection`), a strict Content Security Policy, `X-Frame-Options: DENY`, HSTS on https, and client IPs taken from forwarded headers only behind trusted proxies.
- User-written SQL segments run in a read-only transaction with a statement timeout, as a role that can only read a per-query view of one workspace's contacts.
- Outgoing webhooks are signed (HMAC-SHA256) and blocked from private networks.
- The audit log records allowed and denied actions and cannot be edited from the application.

Found a vulnerability? Please email security@hostrepute.com instead of opening an issue.

## API, webhooks and MCP

- **REST API** under `/api/v1`, documented by the OpenAPI 3.1 document at
  `/api/openapi.json` (or `reputecast openapi`). Errors are RFC 9457 problem
  documents with a stable `code`, and permission errors name who can help.
- **API keys** (Settings › Keys & MCP) carry a subset of your permissions,
  optionally scoped to lists and with an expiry. Send them as
  `Authorization: Bearer rc_live_…`. They are shown once.
- **Outgoing webhooks** (Settings › Webhooks) deliver signed events with
  retries and a delivery log.
- **MCP server** for AI agents: create an agent in Settings › Keys & MCP,
  give it permissions (for example reports and deliverability, or the right
  to pause a send but never to start one) and point your MCP client at
  `<public URL>/mcp` with the agent token. Agents' actions appear in the audit log.

## Development

Prerequisites: Go 1.27 (the toolchain downloads automatically), Node 24 LTS
with pnpm, Docker.

```bash
make deps        # Go modules + web dependencies
make dev         # Postgres + Mailpit (docker compose), API with live reload on :8080, web on :5173
make seed        # load the Acme demo data
```

Open <http://localhost:5173>. Demo accounts (password `reputecast-demo`):

| Account | Role | Notes |
|---|---|---|
| `linh@acme.io` | Owner, instance admin | Authenticator secret `JBSWY3DPEHPK3PXP`; recovery code `b4xk-9q2m-7hdn` |
| `aisha@acme.io` | Viewer | Password only (two-factor grace period) |

Mailpit shows every email the demo sends at <http://localhost:8025>.

| Task | Command |
|---|---|
| All tests (Go unit + integration on real Postgres, web unit tests) | `make test` |
| Linters (golangci-lint, ESLint, TypeScript) | `make lint` |
| Regenerate sqlc, OpenAPI and web API types | `make generate` |
| Production build (web app embedded in `bin/reputecast`) | `make build` |
| Docker image | `make docker` |
| Landing site (reputecast.com): build, or live reload on :5288 | `make site`, `make site-dev` |
| Adapter SDK and template: build, test, run through the Go runtime | `make sdk` |
| Compare a screen with its design board at 1440/834/390 | `cd web && pnpm shot --path /w/acme-newsletter/campaigns --design Campaigns` |
| Side-by-side QA of boards and screens, with a difference score (`scripts/qa-map.ts` maps boards to routes) | `cd web && pnpm qa --only Report,Campaigns` |
| Re-render the demo's placeholder images | `cd web && node scripts/demo-images.ts [file.png …]` |

The screenshot tools read the design boards from `http://localhost:8765`:
`python3 -m http.server 8765 -d design/boards` serves them. Static boards
render as they are; the phone and tablet wrappers and the boards with live
parts (composer, automation flow, landing page) also need the design canvas
runtime saved as `design/boards/support.js`, which is not part of this
repository.

Integration tests start Postgres 18 with Testcontainers, or reuse the
compose database when it runs (`make test` does that automatically). CI runs
the Go suite on Go 1.27 against Postgres 16, 17 and 18. Packages test in
parallel against one server; `go test -p 4 ./...` keeps the connection count
within Postgres' default `max_connections`.

Read [`docs/dev/CONVENTIONS.md`](docs/dev/CONVENTIONS.md) before
contributing: it explains how a screen is built from the design boards to
the API, the demo data and the tests.

## Project layout

```
cmd/reputecast/        the binary
internal/
  api/                 Huma setup, access rules, errors
  app/                 wiring (domains plug in from wire_<domain>.go)
  authz/ authn/        permissions and roles; passwords, tokens, TOTP
  audit/ secrets/      audit log; encryption at rest
  accounts/ workspaces/ campaigns/ providers/ …   one package per domain
  store/               sqlc queries (queries/*.sql) and generated code
  demo/                the Acme sample data
migrations/            goose SQL migrations (embedded)
web/                   React app (embedded into the binary)
site/                  reputecast.com landing page (static, prerendered)
design/                the design prototype boards and the Clearwater design system
docs/                  specs, roadmap, conventions, design inventory
deploy/                production docker compose and release Dockerfile
```

## Roadmap

See [`docs/ROADMAP.md`](docs/ROADMAP.md) for milestones and status, and
[`docs/DESIGN_INVENTORY.md`](docs/DESIGN_INVENTORY.md) for how every designed
screen maps to the spec.

## Contributing

Issues and pull requests are welcome. Keep `make test`, `make lint` and
`make build` green, add tests with your change, follow the design boards for
UI and the specs for behaviour, and never add paywalls or license checks:
every feature in this repository is free.

## License

ReputeCast is licensed under the [GNU Affero General Public License v3.0](LICENSE).

Built by [HostRepute](https://hostrepute.com).
