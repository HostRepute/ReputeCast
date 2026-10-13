# Roadmap

Milestones follow the kickoff plan (M0–M14). Work is organised in
workstreams (vertical slices, `docs/dev/WORKSTREAMS.md`) that map onto the
milestones; each milestone is shippable and testable on its own.

Status: ✅ done · 🚧 in progress · ⏳ planned

| Milestone | Scope | Workstreams | Status |
|---|---|---|---|
| **M0 Skeleton** | Repo layout, Makefile, docker compose (Postgres 18 + Mailpit), goose, sqlc, River, chi + Huma (`/healthz`, `/readyz`, `/metrics`, OpenAPI 3.1), embedded SPA, openapi-typescript, CI (Go 1.26/1.27 × PG 16/17/18, web lint/typecheck/test/build), GoReleaser, Dockerfile. Design system foundation in the web app (tokens, components, shell). | core | ✅ |
| **M1 Install and first sign-in** | `reputecast install`, identifier-first sign-in, argon2id, throttling and lockout, TOTP + recovery codes, sessions, first sign-in setup API; passkeys, password reset, invite accept, Account & security screens | core, W04 | 🚧 |
| **M2 Workspaces, team, roles, audit** | `internal/authz`, built-in and custom roles, escalation guard, invitations, member management, API keys and MCP agents, audit log views | core, W03 | 🚧 |
| **M3 Sending providers** | Provider contract and contract tests, SES, SMTP, Postmark, Postal adapters, webhook ingest and normalization, shared suppression list and sync, streams and routing, failover, warm-up | W01 | 🚧 |
| **M4 Senders, domains, Send Readiness** | DNS checks, sending domains and senders, connect-a-provider wizard, Send Readiness engine | W10, W11 | ⏳ |
| **M5 Audience** | Contacts, lists, custom fields, tiers and sunset policy, segments (builder and SQL), suppressions, importer | W02 | 🚧 |
| **M6 Composer** | GrapesJS + MJML wrapper, code view, previews, Liquid tokens, templates, versions, inline warnings | W05 | 🚧 |
| **M7 Review & send, approvals, live monitor** | Review & send, approval policy/inbox/review, sending pipeline, live monitor (SSE), auto-pause, outage handling | W11, W12 | ⏳ |
| **M8 Reports** | Campaign report, honest analytics (human vs machine opens), click map, provider comparisons, A/B, share links; Overview | W09 | ⏳ (campaigns list ✅) |
| **M9 Automations** | React Flow editor, triggers and steps, runs engine, live stats, versions and publish diff, templates | W06 | 🚧 |
| **M10 Forms and public pages** | Signup forms, double opt-in, preference center, unsubscribe (RFC 8058), archive | W13 | ⏳ |
| **M11 SSO, social sign-in, API keys, MCP, webhooks** | OIDC SSO with JIT and group mapping, social sign-in, instance admin, API keys & MCP server, outgoing webhooks | W14, W03 | ⏳ |
| **M12 Remaining adapters** | Mailgun, SendGrid, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, generic HTTP | W01 (follow-up) | ⏳ |
| **M13 Deliverability and HostRepute** | Domain/IP health, provider account health, blocklists, Postmaster, SNDS, DMARC, incidents, Connect HostRepute | W07 | 🚧 |
| **M14 Migration, landing, docs, v1.0** | Importers from Brevo, Mailchimp, MailerLite, listmonk, Keila; landing page and README banner; documentation; release | W02, W08 | 🚧 (landing page ✅, README banner ✅) |

## Done so far (M0)

- Single binary with `serve`, `worker`, `install`, `migrate`, `user`, `seed`, `openapi`, `secrets`, `version`.
- Schema for every domain (migrations 00001–00010), sqlc, River with a maintenance worker.
- Access layer: permissions from the Roles screen, built-in roles, escalation guard, one middleware for users, API keys and MCP agents, RFC 9457 errors with "who can help".
- Security: CSP and security headers, Go cross-origin protection, trusted-proxy client IPs, AES-GCM secrets with key rotation, hashed session and API tokens, immutable audit log.
- Sign-in API (identify, password, two-factor, sign-out), `/me`, first-run setup API, integration tests on real Postgres.
- Web: Clearwater tokens and component vocabulary, app shell (sidebar → rail → tab bar), sign-in flow, campaigns list with KPIs, filters, search, pagination and states.
- Demo data (`reputecast seed`) and a visual QA tool comparing app screenshots with the design boards (`web/scripts/shot.ts`).

## Done so far (W08: landing, README banner, cross-cutting states)

- `site/`: the reputecast.com landing page (Landing board at 1440, 834 and 390) with its motion, a Pause control, final states for reduced motion, the editable cost calculator, prerendered HTML, meta and Open Graph tags, JSON-LD, sitemap, robots, favicons and self-hosted IBM Plex. `make site` and its own CI job.
- README banner: `docs/assets/readme-banner.svg` (outlined text) and `.png` (1280×640).
- In-shell 404, 403 (role-limited, "Request … access" with the read-only way in; `GET/POST /workspaces/{workspace}/access-requests`) and 500, for unknown paths and for screens that throw their load error (the route's error component picks the state from the status).
- Session: a warning two minutes before the idle timeout with "Extend 12 hours" (`GET /me/session`, `POST /me/session/extend`), and a sign-in dialog over the page that keeps unsaved work when the session has ended.
- Offline/unreachable and read-only maintenance banners and the API rate-limit notice, wired for every screen; reconnecting and quota-reached banners for the screens that own those states.
- Confirmation dialogs (typed name, reason, options, bulk results, unsaved changes), toasts with Undo, Retry and Hide, a form-level error summary, empty search results, disabled controls that say why, keyboard shortcuts with a `?` sheet, and the background jobs tray (job list behind `JobsSource` until the import/export API lands).
