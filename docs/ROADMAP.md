# Roadmap

Milestones follow the kickoff plan (M0–M14). Work is organised in
workstreams (vertical slices, `docs/dev/WORKSTREAMS.md`) that map onto the
milestones; each milestone is shippable and testable on its own.

Status: ✅ done · 🚧 in progress · ⏳ planned

| Milestone | Scope | Workstreams | Status |
|---|---|---|---|
| **M0 Skeleton** | Repo layout, Makefile, docker compose (Postgres 18 + Mailpit), goose, sqlc, River, chi + Huma (`/healthz`, `/readyz`, `/metrics`, OpenAPI 3.1), embedded SPA, openapi-typescript, CI (Go 1.26/1.27 × PG 16/17/18, web lint/typecheck/test/build), GoReleaser, Dockerfile. Design system foundation in the web app (tokens, components, shell). | core | ✅ |
| **M1 Install and first sign-in** | `reputecast install`, identifier-first sign-in, argon2id, throttling and lockout, TOTP + recovery codes, sessions, first sign-in setup API; passkeys, password reset, invite accept, Account & security screens | core, W04 | 🚧 |
| **M2 Workspaces, team, roles, audit** | `internal/authz`, built-in and custom roles, escalation guard, invitations, member management, API keys and MCP agents, audit log views | core, W03 | ✅ |
| **M3 Sending providers** | Provider contract and contract tests, SES, SMTP, Postmark, Postal adapters, webhook ingest and normalization, shared suppression list and sync, streams and routing, failover, warm-up | W01 | ✅ |
| **M4 Senders, domains, Send Readiness** | DNS checks, sending domains and senders, connect-a-provider wizard, Send Readiness engine | W10, W11 | ⏳ |
| **M5 Audience** | Contacts, lists, custom fields, tiers and sunset policy, segments (builder and SQL), suppressions, importer | W02 | ✅ |
| **M6 Composer** | GrapesJS + MJML wrapper, code view, previews, Liquid tokens, templates, versions, inline warnings | W05 | ✅ |
| **M7 Review & send, approvals, live monitor** | Review & send, approval policy/inbox/review, sending pipeline, live monitor (SSE), auto-pause, outage handling | W11, W12 | ⏳ |
| **M8 Reports** | Campaign report, honest analytics (human vs machine opens), click map, provider comparisons, A/B, share links; Overview | W09 | ⏳ (campaigns list ✅) |
| **M9 Automations** | React Flow editor, triggers and steps, runs engine, live stats, versions and publish diff, templates | W06 | ✅ (emails queue until W12's mailer sends them; approvals hook for W11) |
| **M10 Forms and public pages** | Signup forms, double opt-in, preference center, unsubscribe (RFC 8058), archive | W13 | ⏳ |
| **M11 SSO, social sign-in, API keys, MCP, webhooks** | OIDC SSO with JIT and group mapping, social sign-in, instance admin, API keys & MCP server, outgoing webhooks | W14, W03 | 🚧 (API keys, MCP server, webhooks ✅) |
| **M12 Remaining adapters** | Mailgun, SendGrid, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, generic HTTP | W01 (follow-up) | ⏳ |
| **M13 Deliverability and HostRepute** | Domain/IP health, provider account health, blocklists, Postmaster, SNDS, DMARC, incidents, Connect HostRepute | W07 | 🚧 |
| **M14 Migration, landing, docs, v1.0** | Importers from Brevo, Mailchimp, MailerLite, listmonk, Keila; landing page and README banner; documentation; release | W02, W08 | 🚧 (importers ✅, landing page ✅, README banner ✅) |

## Done so far (M0)

- Single binary with `serve`, `worker`, `install`, `migrate`, `user`, `seed`, `openapi`, `secrets`, `version`.
- Schema for every domain (migrations 00001–00010), sqlc, River with a maintenance worker.
- Access layer: permissions from the Roles screen, built-in roles, escalation guard, one middleware for users, API keys and MCP agents, RFC 9457 errors with "who can help".
- Security: CSP and security headers, Go cross-origin protection, trusted-proxy client IPs, AES-GCM secrets with key rotation, hashed session and API tokens, immutable audit log.
- Sign-in API (identify, password, two-factor, sign-out), `/me`, first-run setup API, integration tests on real Postgres.
- Web: Clearwater tokens and component vocabulary, app shell (sidebar → rail → tab bar), sign-in flow, campaigns list with KPIs, filters, search, pagination and states.
- Demo data (`reputecast seed`) and a visual QA tool comparing app screenshots with the design boards (`web/scripts/shot.ts`).

## Done so far (W01: sending providers & routing, M3)

- Adapters for Amazon SES (SESv2, SNS signature check and subscription handshake, Easy DKIM, configuration-set tracking, VDM flag, AWS status feed), Postmark (batch send, Broadcast stream, basic-auth webhooks, outbound stats as its own reputation), Postal (RS256 signed webhooks via JWKS, message backfill) and generic SMTP (go-mail, DSN/ARF parsing, IMAP and POP3 bounce mailboxes). All pass the shared contract suite (`internal/providers/contracttest`); optional interfaces for quota, domains, suppressions, webhooks, tracking, backfill and status pages.
- Webhook ingest `POST /hooks/{adapter}/{token}`: verify, store the raw payload, 200 at once; a River job normalizes into delivery events, messages, campaign counters, the shared suppression list and provider stats, with dedupe; sweeper and 90-day retention.
- Health job: quota and rate discovery, call windows, outage detection (error rate or two failed probes), automatic failover per stream policy, held Broadcast shares with wait/move/cancel decisions, recovery after ten healthy minutes, rejected credentials, webhooks silent after six hours with a diagnosis and backfill, incidents, notifications and system audit entries.
- Routing per stream (primary, weighted split, conditions on mailbox provider/list/sender domain, fallback, failover policy), bulk-terms validation with an acknowledged override, route tester; warm-up plans with guards, slowdowns, share steps and a daily job; suppression sync per provider; provider-side tracking checks; community and generic HTTP adapters (instance admins).
- Screens: Providers (healthy, outage, webhooks silent, rejected credentials, empty, loading), Routing & failover (with the bulk-terms fix view), Warm-up, Events & sync, Adapters, provider settings with write-only credentials. Demo data matches the boards; `internal/providers/state` and `manage.UsageOf` are the contract other screens read.

## Done so far (W02: audience and migration importer, M5 and the M14 importers)

- Contacts: filters (list, tier, mailbox provider, consent, source, tag, joined, segment), saved views with counts, paging, selection with bulk add to list, tag, resend opt-in, suppress, export (logged, needs `subscribers:export`) and erase (typed count), partial-result reporting. Contact profile with fields, tags, consent per list and its history, suppression, preference link (logged) and the activity timeline with the provider that delivered each email.
- Tiers and sunset policy: nightly tier job per workspace at 03:00 local, human opens and clicks only unless the policy says otherwise, re-engagement tagging and the Broadcast stop. List health: tier cards with month-over-month deltas, the 12-week trend with the policy column, this week's projection, and health per list and followed view.
- Segments: builder with groups and conditions over fields, lists, tiers, engagement and custom fields, live counts with the tier mix and a 10-second timeout, generated SQL; SQL mode runs one SELECT over per-workspace views as a read-only role (`reputecast_segments`) in a READ ONLY transaction with a statement timeout.
- Suppression list: counts, per-provider sync (pulled and pushed; providers without a suppression API are checked before sending), import groups, add, CSV upload, pull from providers, remove with a reason (audited), export.
- Importer: Brevo, Mailchimp, MailerLite, listmonk, Keila (API keys sealed, never returned) and CSV/XLSX; staging on River with rate-limit pauses, "Resume now" and "Slow down to 3 req/s"; field mapping with guesses, consent mapping with locked suppressions, a dry-run preview (duplicates, invalid, MX-less domains, role, disposable, typo, stale, trap-like), batched import with merges, suppressions pushed to providers, re-permission emails, report and skipped-row CSVs, and the jobs tray feed (`GET /workspaces/{workspace}/jobs`).
- Screens: Contacts, Contact profile, Segment builder, List health, Suppression list, the import wizard (source, fields, consent, preview, progress, summary) and their empty, loading, timed-out, partly-done, rejected-key, rate-limit and broken-file states. Demo data reproduces the boards' numbers; integration tests cover counts, permissions, audit entries, user SQL and a full CSV and Mailchimp import.

## Done so far (W08: landing, README banner, cross-cutting states)

- `site/`: the reputecast.com landing page (Landing board at 1440, 834 and 390) with its motion, a Pause control, final states for reduced motion, the editable cost calculator, prerendered HTML, meta and Open Graph tags, JSON-LD, sitemap, robots, favicons and self-hosted IBM Plex. `make site` and its own CI job.
- README banner: `docs/assets/readme-banner.svg` (outlined text) and `.png` (1280×640).
- In-shell 404, 403 (role-limited, "Request … access" with the read-only way in; `GET/POST /workspaces/{workspace}/access-requests`) and 500, for unknown paths and for screens that throw their load error (the route's error component picks the state from the status).
- Session: a warning two minutes before the idle timeout with "Extend 12 hours" (`GET /me/session`, `POST /me/session/extend`), and a sign-in dialog over the page that keeps unsaved work when the session has ended.
- Offline/unreachable and read-only maintenance banners and the API rate-limit notice, wired for every screen; reconnecting and quota-reached banners for the screens that own those states.
- Confirmation dialogs (typed name, reason, options, bulk results, unsaved changes), toasts with Undo, Retry and Hide, a form-level error summary, empty search results, disabled controls that say why, keyboard shortcuts with a `?` sheet, and the background jobs tray (job list behind `JobsSource` until the import/export API lands).

## Done so far (W03: workspace settings, team, roles, keys & MCP, webhooks, audit log)

- Settings › General: name, timezone and default sender, approvals and members summaries, Owner-only transfer (re-authentication, two-factor candidates) and delete (typed name, export ZIP first, 7-day restore, hourly purge job).
- Team: members with role, scope, sign-in, two-factor and status; filters, bulk change role / require two-factor / deactivate; invitations with scope, expiry and the copy-link fallback; member page with effective permissions and their source, recent activity, keys, approvals, sessions; change role or scope, deactivate (keep or revoke keys), remove, last-Owner guard.
- Roles: list and compare matrix; role editor with included and locked permissions, live impact preview and confirmation before adding a sensitive permission or removing access; duplicate and delete.
- Keys & MCP: API keys and MCP agents (escalation guard, list scope, expiry, shown once, rotate, revoke, replace an expired key with what still calls it); MCP server at `/mcp` (Streamable HTTP, 18 tools, resources, prompts, per-agent rate limit, proposals instead of sends, audit with the agent as actor).
- Webhooks: endpoints with event groups, HMAC-SHA256 signatures, write-only secrets, SSRF guard, River delivery with backoff over 6 hours, dead-letter queue, held deliveries while paused, retry and replay, payload view.
- Audit log: filters (actor type, action category, result, range), keyset paging, entry detail with before/after and related entries, CSV/JSON export, signed stream to a webhook, retention.

## Done so far (W06: automations)

- Automations list (AutomationsList board at 1440, 834 and 390): 30-day performance per flow, filters, the Automations route, 7-day KPIs, alerts for held sending and failing steps, starter templates, and the empty, loading and error states (AutomationsStates).
- Flow editor (AutomationFlow board): a React Flow canvas with the board's steps, orthogonal edges with Yes/No and A/B labels, the pulse-through-steps motion (pausable, off with reduced motion), count-up stats, minimap and controls, snap to grid, drag from the palette onto the canvas or onto an edge, drawing edges between handles, deletion that reconnects the flow, and side panels for every step type. The draft saves itself (revision-checked, ⌘S saves now); `/automations/new?template=…` starts from a template.
- Versions: publish dialog with the diff since the live version, "let them finish" or "move them at their current step", and pre-publish checks (AutomationPublish board); version history with compare and restore; pause and resume.
- Triggers: joined list, form submitted, tag added, API event (`POST /workspaces/{workspace}/automations/events`) and date fields. Steps: delay, send email (Automations route unless overridden, send window, skip rules), branch (opened, clicked, field, tag, list, event), filter, A/B split, webhook (SSRF-safe, signed, retried, logged), list and tag actions, goal and exit.
- Runs engine on River: enrollment from trigger events, delays, branch waits woken by engagement, goal detection, per-step daily stats, retiring old versions, alerts. Emails go through the `automations.Mailer` seam (queued until the sending pipeline lands); publishing goes through `automations.PublishGate` for approvals.
- Demo: the Welcome series (v3 live, v4 draft whose diff the publish dialog shows), Onboarding, Re-engagement, the Re-permission draft and the paused Beta invite, with 30 days of per-step stats and contacts in each flow.

## Done so far (M6 Composer, W05)

- `/campaigns/new` creates a draft and opens it on the template gallery; `/campaigns/:id/edit` is the full-screen composer (`?view=mjml|html|preview`, `?device=mobile`, `?variant=B`, and deep links `?select=2-0&panel=deliverability&left=layers`).
- GrapesJS 0.23 + grapesjs-mjml behind ReputeCast's own UI (lazy chunk, none of GrapesJS's panels): blocks (MJML layout and content, ReputeCast header, article and product cards, footer with address and unsubscribe, saved sections), layers, templates, a Style/Settings inspector limited to MJML attributes, selection toolbar (parent, move by drag or arrow keys, duplicate, delete), rich text toolbar (bold, italic, links, personalization tokens), image library with validated uploads, device switcher 600/375 with zoom-to-fit, undo/redo and shortcuts. The canvas renders the head's mj-attributes and mj-style as the email will.
- Code view (CodeMirror): MJML with compile results and layout warnings, read-only compiled HTML, import, format, structural checks before Apply.
- Autosave every 30 s and 2 s after structural changes, ⌘S versions, conflict detection with keep-mine/load-theirs/compare, offline backup and retry, version history with restore and line diffs, the shared leave guard.
- Inline checks (`internal/contentcheck`, reused by Send Readiness): size against Gmail's clip, alt text, image share, blocklisted/shortened/broken links, footer and postal address, plain text, personalization, subject heuristics; tags on the canvas and one-click fixes.
- Previews: personalized per contact (Liquid with fallbacks, `internal/liquid`), light and dark mode approximations with dark-mode checks, plain text, inbox line, A/B set-up (subject, content and send time), test sends through the System route.
- API: drafts, content saves, versions, restore, duplicate, previews, test sends, checks, MJML compile (gomjml), templates and saved sections, media; demo content for the Composer, ComposerWarnings and ComposerEmpty boards.
