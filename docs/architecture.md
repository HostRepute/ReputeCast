# How ReputeCast works

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
  adapter. Community adapters run in their own process and are called over a
  signed protocol, written with the TypeScript SDK in `sdk/`. The
  [adapters guide](adapters.md) shows how an adapter is written and
  contract-tested, and how to map any HTTP API without code.
- **Webhooks are cheap to receive.** Ingest verifies the signature, stores the
  raw payload and answers 200 immediately; a job normalizes events later, so
  provider retries never pile up.
- **Access control is one check.** Every API operation declares the permission
  it needs (visible as `x-permission` in the OpenAPI document). Users, API keys
  and MCP agents all pass the same check in `internal/authz`; keys and agents
  can never exceed their creator.
- **Everything that changes something is audited**, in the same transaction as
  the change, with before/after values, actor, IP, user agent and request id.
  Audit entries are immutable (enforced by a database trigger).
- **The web app** (React 19, TanStack Router and Query, Tailwind 4) is built
  with Vite and embedded in the binary; API types are generated from the
  OpenAPI document.
- **Sends are guarded.** Send Readiness runs again as a send starts; a
  scheduled send it blocks is held, and starts once a recheck passes. A send
  builds its recipient list in pages, routes each recipient, then hands
  batches to the providers from River jobs within each provider's rate and
  each mailbox provider's throttle; every message is claimed before it goes
  out, so nothing is sent twice. Bounce, complaint and deferral rates are
  checked every two seconds against the campaign's thresholds, and a rate that
  is over its limit with enough evidence pauses just the mailbox provider or
  sending provider it concerns. Opens and clicks come back through `/t/o` and
  `/t/c`, with privacy-proxy and scanner traffic counted apart.

The stack and the reasons behind each choice are in
[TECH_STACK.md](TECH_STACK.md).

## Security

- Passwords are hashed with argon2id, with an optional breached-password check
  (k-anonymity; it can be turned off for offline installs).
- Sessions and API keys are random 256-bit tokens; only their SHA-256 hashes
  are stored. Session cookies are `HttpOnly`, `SameSite=Lax` and `Secure` on
  https.
- Two-factor with TOTP, passkeys (WebAuthn) and one-time recovery codes;
  policies can require it for everyone or for anyone holding a sensitive
  permission.
- Identifier-first sign-in with generic errors (no account enumeration),
  per-email lockout and per-IP rate limits.
- Provider credentials, OIDC client secrets, HostRepute tokens and TOTP seeds
  are encrypted with AES-256-GCM bound to their row, are write-only and are
  never returned by the API. Keys can be rotated.
- Cross-site request protection (Go's `CrossOriginProtection`), a strict
  Content Security Policy, `X-Frame-Options: DENY`, HSTS on https, and client
  IPs taken from forwarded headers only behind trusted proxies.
- User-written SQL segments run in a read-only transaction with a statement
  timeout, as a role that can only read a per-query view of one workspace's
  contacts.
- Outgoing webhooks are signed (HMAC-SHA256) and blocked from private
  networks.
- The audit log records allowed and denied actions and cannot be edited from
  the application.

To report a vulnerability, see [SECURITY.md](../SECURITY.md).
