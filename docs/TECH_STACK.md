# ReputeCast — Tech stack

Last verified: 2026-10-13 against go.dev, proxy.golang.org, the npm registry, Docker Hub and the GitHub releases of each project.

Versions are minimums to start from. Upgrade within the same major version freely. Changing a major version needs a note in the PR.

## Runtime and infrastructure

| Component | Version | Notes |
|---|---|---|
| Go | **1.27.2** | `go 1.27` in go.mod. CI also tests 1.26. |
| PostgreSQL | **18.6** (default) | Supported: **16, 17 and 18**, with all three in the CI matrix. Postgres is the only database. See "Postgres 18 notes". |
| Docker image (app) | `gcr.io/distroless/static-debian13:nonroot` | CGO is disabled, so the binary is static. |
| Docker image (DB) | `postgres:18-alpine` or `postgres:18-trixie` | Mount the volume at `/var/lib/postgresql`, not `/var/lib/postgresql/data`; the path changed in 18. |
| Mailpit (dev/test SMTP) | **1.31.5** | `axllent/mailpit` |
| Node.js (build only) | **24 LTS (24.21)** | Node 26 becomes LTS on 2026-10-28; switch then. Node is never needed at runtime. |
| pnpm | **12.11** | |

## Backend (Go modules)

| Purpose | Module | Version |
|---|---|---|
| Router | github.com/go-chi/chi/v5 | v5.4.0 |
| API + OpenAPI 3.1 | github.com/danielgtaylor/huma/v2 (adapter `humachi`) | v2.39.1 |
| Rate limiting (HTTP) | github.com/go-chi/httprate | v0.16.1 |
| Rate limiting (per provider) | golang.org/x/time/rate | v0.16.0 |
| Postgres driver | github.com/jackc/pgx/v5 | v5.11.0 |
| Query codegen | sqlc (CLI) | v1.31.1 |
| Migrations | github.com/pressly/goose/v3 | v3.29.0 |
| Job queue + scheduler | github.com/riverqueue/river + riverdriver/riverpgxv5 | v0.49.0 |
| Job dashboard | riverqueue.com/riverui | v0.21.0 |
| UUID (v7 primary keys) | github.com/google/uuid (`uuid.NewV7`) | v1.6.0 |
| Templates | github.com/osteele/liquid | v1.9.2 |
| Server-side MJML (API/automation sends only) | github.com/preslavrachev/gomjml | v0.13.0 (targets MJML 4.15.3) |
| AWS SES | github.com/aws/aws-sdk-go-v2 + service/sesv2 | v1.47.3 / v1.79.1 |
| SMTP | github.com/wneessen/go-mail | v0.8.1 |
| Bounce mailbox (IMAP) | github.com/emersion/go-imap/v2 | v2.0.0-beta.8 (beta; isolated in one package) |
| DNS checks | github.com/miekg/dns | v1.1.73 (behind `internal/dnscheck`; see notes) |
| DKIM/DMARC parsing | github.com/emersion/go-msgauth | v0.7.0 |
| SMTP relay for transactional email (server, AUTH, MIME) | github.com/emersion/go-smtp + go-sasl + go-message | v0.25.0 / v0.0.0-20241020182733-b788ff22d5a6 (no tagged release) / v0.18.2 |
| SMS through AWS End User Messaging | github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2 | v1.42.0 |
| Phone numbers (E.164, country, type) | github.com/nyaruka/phonenumbers | v1.8.1 |
| App languages: language names, plural rules, collation, currencies, Unicode normalization | golang.org/x/text | v0.43.0 |
| Password hashing | golang.org/x/crypto/argon2 (argon2id) | v0.58.0 |
| TOTP 2FA | github.com/pquerna/otp | v1.5.0 |
| Passkeys | github.com/go-webauthn/webauthn | v0.18.2 |
| OIDC SSO + social sign-in | github.com/coreos/go-oidc/v3 + golang.org/x/oauth2 | v3.21.0 / v0.37.0 |
| Sessions | In-house: a Postgres table storing hashed tokens. (Alternative: alexedwards/scs/v2 v2.9.0 + pgxstore.) | |
| API keys | In-house: random 32-byte token with a visible prefix; only the SHA-256 hash is stored. | |
| Authorization | In-house package `internal/authz` | |
| MCP server | github.com/modelcontextprotocol/go-sdk | v1.8.0 |
| HostRepute client | github.com/hostrepute/hostrepute-go | The repo is currently private; it must be public and tagged before ReputeCast depends on it. |
| CLI | github.com/spf13/cobra | v1.10.2 |
| Config | github.com/knadh/koanf/v2 | v2.3.8 |
| Logging | log/slog (stdlib) | |
| Metrics | github.com/prometheus/client_golang | v1.25.0 |
| Tracing (optional) | go.opentelemetry.io/otel | v1.47.0 |
| Integration tests | github.com/testcontainers/testcontainers-go + modules/postgres | v0.44.0 |
| Future plugins (WASM) | github.com/tetratelabs/wazero (+ Extism go-sdk v1.7.1, re-evaluate) | v1.12.0 |
| Deferred: SAML | github.com/russellhaering/gosaml2 (not crewjam/saml, which is stale) | v0.12.0 |
| Deferred: SCIM | github.com/elimity-com/scim | untagged (pseudo-version 2026-07) |

Dev tools:
- air v1.67.4
- golangci-lint v2.14.0
- GoReleaser v2.18.3
- sqlc and goose, pinned with `go get -tool`

## Frontend (npm)

| Purpose | Package | Version |
|---|---|---|
| UI | react, react-dom | 19.3.0 |
| Language | typescript | **6.0.3** (not 7; see notes) |
| Build | vite + @vitejs/plugin-react | 8.3.4 / 6.1.2 |
| Routing | @tanstack/react-router + @tanstack/router-plugin | 1.170.x / 1.168.x |
| Server state | @tanstack/react-query | 5.104.2 |
| Tables | @tanstack/react-table | 9.3.1 (v9; shadcn data-table uses v9) |
| Styling | tailwindcss + @tailwindcss/vite | 4.3.3 |
| Components | shadcn (CLI; generated components live in our repo) | 4.21.4 |
| Forms | react-hook-form + zod | 7.90.0 / 4.6.7 |
| Charts | recharts | 3.10.1 |
| Code editors (SQL, Liquid, HTML) | @codemirror/view, @codemirror/lang-sql, @codemirror/lang-liquid, @codemirror/lang-html | 6.43.x / 6.10.0 / 6.3.3 / 6.4.12 |
| Email composer | grapesjs + grapesjs-mjml (own thin React wrapper, not @grapesjs/react) | 0.23.6 / 1.0.8 (bundles mjml-browser ^4.18) |
| Automation canvas | @xyflow/react | 12.12.0 |
| API types | openapi-typescript (generated from the Huma OpenAPI spec) | 7.13.0 |
| API client | openapi-fetch + openapi-react-query | 0.17.0 / 0.5.4 |
| App languages: ICU messages (plurals, selects, numbers, dates) | intl-messageformat + @formatjs/icu-messageformat-parser | 12.1.2 / 3.5.20 |
| Lint | eslint + typescript-eslint + eslint-plugin-react-hooks | 10.12 / 8.71 / 7.1 |
| Unit tests | vitest | 5.0.4 |
| E2E tests | @playwright/test | 1.64.0 |

## Notes and decisions

1. **TypeScript 6, not 7.** TypeScript 7.0 is a native build and no longer exposes the JavaScript compiler API.
   - openapi-typescript 7.13.0 crashes with TS 7: `Cannot read properties of undefined (reading 'createKeywordTypeNode')`. We reproduced this locally (upstream issue #2841).
   - typescript-eslint 8.71 supports only `<6.1`.
   - openapi-typescript works with TS 6.0.3 (verified), but its peer range still says `^5.x`, so allow it in pnpm:
     ```json
     "pnpm": { "peerDependencyRules": { "allowedVersions": { "openapi-typescript>typescript": "6" } } }
     ```
   - Move to TS 7 when both tools support it.
2. **MJML 4, not 5.** MJML 5 has existed since 2026-04-16 (latest 5.4.1), but grapesjs-mjml 1.0.8 bundles mjml-browser 4.x and gomjml targets MJML 4.15.3. The client-side and server-side compilers must produce the same output, so we stay on MJML 4 until both sides support 5.
3. **Postgres 18 notes.**
   - Use 18 by default for its async I/O and faster upgrades, but support 16–18.
   - sqlc's parser follows the Postgres 17 grammar, so don't use 18-only SQL syntax in queries or migrations, such as `RETURNING OLD/NEW` or virtual generated columns.
   - UUIDv7 keys are generated in Go with `uuid.NewV7`, not with the Postgres 18 `uuidv7()` function, so 16 and 17 keep working.
   - Postgres 18 Docker images moved the data volume to `/var/lib/postgresql`.
4. **River is pre-1.0 (v0.49).** Pin the minor version and read the changelog before every upgrade. River's own CI tests Postgres 14–18.
5. **DNS library.** miekg/dns v1 is in maintenance mode, and v2 lives at codeberg.org/miekg/dns (still 0.x). Use v1.1.73 behind the `internal/dnscheck` interface and switch when v2 reaches 1.0.
6. **go-imap v2 is still in beta.** It's only used for the optional bounce-mailbox feature, which lives in its own package.
7. **SAML is deferred.** If it's needed later, use gosaml2. crewjam/saml has had no release since 2025-04.
8. **hostrepute-go must be public** (with a version tag) before ReputeCast imports it; otherwise public builds and contributors can't fetch it.
9. **Single binary.** `go:embed` bundles the built SPA and the goose migrations. GoReleaser builds linux/darwin/windows for amd64/arm64, plus a multi-arch Docker image.
10. **Every provider adapter** must pass the shared contract-test suite: Send, ParseWebhook (signature verification and event normalization), and Capabilities. These tests run against recorded fixtures, plus Mailpit for SMTP.
11. **Integration apps** (`integrations/`) are JavaScript and PHP projects with their own package managers. The n8n nodes are MIT and the WordPress plugin GPL-2.0-or-later, because each platform's directory requires it; the Zapier and Make apps are AGPL-3.0 like the rest. `integrations/go.mod` makes the folder a module of its own, so the server module's `./...` never reaches their installed dependencies.
