# Contributing to ReputeCast

Thanks for helping. Bug reports, ideas, documentation fixes, new provider
adapters and code are all welcome.

- **Questions and ideas:** start a [discussion](https://github.com/hostrepute/reputecast/discussions).
- **Bugs and feature requests:** open an [issue](https://github.com/hostrepute/reputecast/issues/new/choose).
- **Security problems:** don't open an issue; follow [SECURITY.md](SECURITY.md).
- **A new sending provider:** read the [adapters guide](docs/adapters.md). Many
  HTTP APIs can be mapped without code, and community adapters can live in
  their own repository.

Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

You need Go 1.27 (the toolchain downloads automatically), Node 24 LTS with
pnpm, and Docker.

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

### Live sending in the demo

The demo's sending providers are simulated: nothing they send leaves the
machine. `reputecast demo send` starts a campaign's send through them as
Sofia Rossi, with Marcus Chen's approval recorded, and the running server's
worker keeps sending in real time, with deliveries, bounces, complaints, opens
and clicks at the rates of the live monitor's designs:

```bash
export REPUTECAST_ENV=development   # the command refuses to run otherwise
go run ./cmd/reputecast demo send "Weekly Digest #142" --elapsed 20m
go run ./cmd/reputecast demo send "Weekly Digest #142" --scenario outlook-complaints --elapsed 20m --reset
go run ./cmd/reputecast demo send "Weekly Digest #142" --scenario postal-outage --elapsed 17m --reset
```

`--elapsed` starts the send that long ago and fast-forwards it, so the monitor
opens mid-send. `outlook-complaints` makes lapsed Outlook readers complain
until the Outlook share auto-pauses; `postal-outage` makes Postal refuse
connections 14 minutes in, so its share is held. `--reset` clears an earlier
send of the campaign, including what it changed elsewhere (provider
statistics, suppressions from its bounces). It only runs in a workspace whose
providers are all demo providers.

## Everyday tasks

| Task | Command |
|---|---|
| All tests (Go unit + integration on real Postgres, web unit tests) | `make test` |
| Linters (golangci-lint, ESLint, TypeScript) | `make lint` |
| Every message in every app language, and no hard-coded text | `cd web && pnpm run i18n:check` |
| Regenerate sqlc, OpenAPI and web API types | `make generate` |
| Production build (web app embedded in `bin/reputecast`) | `make build` |
| Docker image | `make docker` |
| Landing site: build, or live reload on :5288 | `make site`, `make site-dev` |
| Adapter SDK and template: build, test, run through the Go runtime | `make sdk` |
| Compare a screen with its design board at 1440/834/390 | `cd web && pnpm shot --path /w/acme-newsletter/campaigns --design Campaigns` |
| Side-by-side QA of boards and screens, with a difference score (`scripts/qa-map.ts` maps boards to routes) | `cd web && pnpm qa --only Report,Campaigns` |
| Accessibility sweep of the same screens (axe-core, WCAG 2.0–2.2 A and AA) | `cd web && pnpm a11y` |
| Re-render the demo's placeholder images | `cd web && node scripts/demo-images.ts [file.png …]` |

The screenshot tools read the design boards from `http://localhost:8765`:
`python3 -m http.server 8765 -d design/boards` serves them. Static boards
render as they are; the phone and tablet wrappers and the boards with live
parts (composer, automation flow, landing page) also need the design canvas
runtime saved as `design/boards/support.js`, which is not part of this
repository.

Integration tests start Postgres 18 with Testcontainers, or reuse the compose
database when it runs (`make test` does that automatically). CI runs the Go
suite on Go 1.27 against Postgres 16, 17 and 18, each in three shards that
take about as long as each other: `scripts/testshard` splits the packages by
their test times in `.github/go-test-durations.txt`, where a new package
counts as the median until `make test-durations` times the suite again. The
slowest packages (`SPLIT_TESTS` in the Makefile, `internal/approvals` today)
are split test by test across the shards; such a package calls
`testsplit.Main` from its `TestMain`, and a test the file doesn't time yet
runs in one of its shards.
Packages test in parallel against one server; `go test -p 4 ./...` keeps the
connection count within Postgres' default `max_connections`.

## Making a change

Read [`docs/dev/CONVENTIONS.md`](docs/dev/CONVENTIONS.md) first: it explains
how a screen is built from the design boards to the API, the demo data and the
tests.

- Keep `make test`, `make lint` and `make build` green, and add tests with
  your change.
- Follow the design boards in `design/` for UI and the specs in `docs/spec/`
  for behaviour. When they disagree, note it in the pull request.
- Every API operation declares the permission it needs, and every change of
  state writes an audit entry.
- Every word people read comes from a message catalog, in the app and on the
  server ([`docs/dev/CONVENTIONS.md`](docs/dev/CONVENTIONS.md), "Text and app
  languages"). Check long text in the pseudo-locale, and see
  [`docs/i18n/translating.md`](docs/i18n/translating.md) to translate.
- Read [`docs/TECH_STACK.md`](docs/TECH_STACK.md) before adding or upgrading
  a dependency, and explain any new one in the pull request.
- Write commit messages and pull request titles as
  [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), for
  example `fix(sending): …`.
- Never add paywalls, license checks or seat limits: every feature in this
  repository is free.

Contributions are accepted under the project's license,
[AGPL-3.0](LICENSE).

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
sdk/                   TypeScript SDK and template for community adapters
integrations/          the Zapier, Make and n8n apps and the WordPress plugin
design/                the design prototype boards and the Clearwater design system
docs/                  guides, specs, roadmap, conventions, design inventory
deploy/                production docker compose and release Dockerfile
```

The roadmap and the status of each milestone are in
[`docs/ROADMAP.md`](docs/ROADMAP.md); [`docs/DESIGN_INVENTORY.md`](docs/DESIGN_INVENTORY.md)
maps every designed screen to the spec.
