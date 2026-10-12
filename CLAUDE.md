# ReputeCast

ReputeCast is an open-source, self-hosted newsletter platform. It works with any email sending provider, and AWS SES is fully supported. It is a sibling of HostRepute (hostrepute.com).

The license is AGPL-3.0. Every feature in this repo is free. Never add paywalls, license checks, seat limits or "Pro/Enterprise" gates.

## Source of truth
- Product spec: `docs/spec/reputecast-design-spec.md`
- Add-on spec for accounts, teams, access and approvals: `docs/spec/reputecast-design-addon-team-access.md`
- Design prototype: `design/`. Match its screens, flows, copy and states. Reuse its design tokens and components; don't invent new visual styles.
- Roadmap and progress: `docs/ROADMAP.md`. Update it at the end of every milestone.
- When the spec and the design disagree, follow the design for UI and the spec for behavior, and note the conflict in the PR description.

## Stack (decided; do not change without asking)
Exact versions and the reasons behind them are in `docs/TECH_STACK.md`. Read it before adding or upgrading a dependency.
- **Backend:** Go 1.27.2. It ships as a single binary, `cmd/reputecast`, with cobra commands: serve, worker, install, migrate, user.
  - Router: chi v5. API: Huma v2, OpenAPI 3.1, errors in RFC 9457 format.
  - Postgres only: 18 by default, 16–18 supported and tested in CI. Don't use 18-only SQL syntax, because sqlc's parser follows the PG17 grammar. UUIDv7 keys are generated in Go. Use pgx v5, sqlc (queries in `internal/store/queries/*.sql`) and goose migrations in `migrations/`.
  - Background jobs: River (pre-1.0; pin the minor version).
  - Config: koanf (file + env + flags). Logging: log/slog. Metrics: Prometheus.
  - Auth: argon2id, pquerna/otp, go-webauthn, coreos/go-oidc. Authorization is a small in-house package, `internal/authz`; don't use Casbin.
  - Templates: Liquid via osteele/liquid. MJML is compiled client-side, and both the MJML source and the HTML are stored. Stay on MJML 4: grapesjs-mjml and gomjml are both MJML 4.
  - Mail and DNS: aws-sdk-go-v2 (sesv2), wneessen/go-mail, miekg/dns.
  - MCP server: modelcontextprotocol/go-sdk.
- **Frontend:** in `web/`. Node 24 LTS + pnpm. React 19.3 + TypeScript 6.0 (not 7, which breaks openapi-typescript and typescript-eslint) + Vite 8, TanStack Router v1 / Query v5 / Table v9, Tailwind 4 + shadcn/ui, Recharts, CodeMirror 6.
  - Composer: grapesjs + grapesjs-mjml behind our own thin wrapper; don't use @grapesjs/react.
  - Automations: @xyflow/react.
  - API types are generated with openapi-typescript from the Huma spec. Never hand-write API types.
  - The production build is embedded into the binary with go:embed.
- **Dev and test:** docker-compose (Postgres + Mailpit), air, golangci-lint v2, testcontainers-go. Every provider adapter must pass the shared adapter contract-test suite.

## Architecture rules
- Sending providers are pluggable. Each one implements `Provider` (Capabilities, Send, ParseWebhook) plus optional interfaces (DomainManager, WebhookRegistrar, QuotaReporter, SuppressionSyncer).
  - Adapters live in `internal/providers/<name>`.
  - Core code never imports a specific adapter.
- Webhook ingest: verify the signature, store the raw event, return 200 immediately, and normalize later in a River job.
- User-written SQL segments run in a READ ONLY transaction with statement_timeout, as a SELECT-only role on a contacts view.
- Every API endpoint declares its required permission, using keys like `campaigns:send`. Users, API keys and MCP agents go through the same authz check. Nobody can grant a permission they don't hold.
- Every state-changing action writes an audit log entry.
- Secrets (provider credentials, OIDC secrets, API keys) are write-only: encrypted at rest and never returned by the API.
- `--role=web|worker|ingest` splits the single process. The default runs everything.

## Working rules
- Work milestone by milestone, following `docs/ROADMAP.md`. Use one branch and one PR per milestone.
- Before writing code for a milestone, read the relevant spec sections and design screens, then post a short plan.
- Write tests along with the code: Go unit tests, integration tests against real Postgres, and Mailpit for sending.
- Keep `make dev`, `make test`, `make lint` and `make build` working at all times.
- Don't add dependencies outside the stack above without explaining why in the PR.
