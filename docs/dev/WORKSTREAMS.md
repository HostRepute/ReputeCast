# Workstreams

The product is built in parallel workstreams, each a vertical slice (API,
SQL, demo data, screens, tests) owning a set of design boards. This file says
who owns what so branches merge cleanly. Conventions: `CONVENTIONS.md`.

## Shared resources

| Thing | Rule |
|---|---|
| Migrations | Each workstream uses only its number range below. |
| Demo data | Each workstream owns one seeder (`internal/demo/seeders.go`) in its own `seed_<name>.go`. |
| Wiring | `internal/app/wire_<package>.go` with an `init()` (never edit `register.go`). |
| SQL | `internal/store/queries/<package>.sql`. |
| Routes | The route files listed for the workstream (stubs exist; replace them). |
| Shared, append-only | `web/src/components/icons/Icon.tsx` (add generic icons), `web/src/lib/format.ts` (add formatters), `web/src/styles/app.css` (only shared component styles, in a labelled section at the end; prefer feature CSS). |
| Read-only | `internal/api`, `internal/authz`, `internal/app/app.go`, `internal/providers/provider.go` (contract), `web/src/components/shell/*`, `web/src/components/ui/*` (extend with care: add variants, never change existing behaviour), `features/settings/SettingsTabs.ts`. |
| Dev ports / DB | API `81NN`, web `52NN`, database `reputecast_wNN` on the compose Postgres (port 5433). |
| Design server | `http://localhost:8765/<Board>.dc.html` (rendered boards) |

## Wave 1

| # | Workstream | Boards (plus their Phone/Tablet variants) | Routes | Packages | Migrations | Seeder |
|---|---|---|---|---|---|---|
| W01 | **Providers & routing** (7.3) | Providers, ProvidersOutage, ProvidersWebhooksSilent, ProvidersBulkWarning, ProvidersRouting, ProvidersWarmup, ProvidersEvents, ProvidersAdapters, ProvidersStates | `_app/providers/{index,routing,warmup,events,adapters}` | `providers` (registry, health), `providers/{ses,smtp,postmark,postal}`, `providers/contracttest`, `routing`, `warmup`, `ingest` (webhooks + normalization job), `suppressions` (sync) | 00100–00149 | `seedProviders` |
| W02 | **Audience & importer** (7.9, 7.11) | Contacts, ContactProfile, SegmentBuilder, ListHealth, Suppressions, AudienceStates, MigrationSource, MigrationFields, MigrationConsent, MigrationPreview, MigrationProgress, MigrationDone, MigrationStates | `_app/audience/**`, `import/**` | `audience` (contacts, lists, fields, tiers, sunset), `segments` (builder → SQL, read-only SQL mode), `imports` (CSV/XLSX + Brevo, Mailchimp, MailerLite, listmonk, Keila) | 00150–00199 | `seedAudience` |
| W03 | **Team, roles, keys, webhooks, audit** (A5.7–5.9, 5.13, 5.14, 7.13) | WorkspaceSettings, WorkspaceDialogs, SettingsTeam, TeamInvite, TeamInviteNoProvider, TeamMember, TeamViewer, TeamDialogs, Roles, RoleEditor, RoleEditorStates, SettingsKeys, SettingsKeysStates, SettingsMCP, SettingsAPI, SettingsAudit, SettingsStates | `_app/settings/{index,team/**,roles/**,keys,mcp,webhooks,audit}`, `/workspaces/new` | `workspaces`, `team`, `roles`, `apikeys`, `mcpserver` (MCP over HTTP with go-sdk, permission-gated tools), `webhooks` (outgoing + delivery job, SSRF guard, HMAC), `auditlog` (views, export, stream) | 00200–00249 | `seedTeam`, `seedIntegrations`, `seedAudit` |
| W04 | **Account, sign-in, notifications** (A5.1–5.6, 5.16) | AuthScreens, SignInStates, TwoFactorStates, InviteStates, InstanceSetup, AccountSecurity, AccountStates, Notifications, ShellNotifications, ShellUserMenu, ShellSwitcher | `sign-in`, `reset-password/**`, `invite.$token`, `setup`, `_app/account`, `_app/notifications`, shell popovers | `accounts` (extend: passkeys with go-webauthn, password reset + CLI link, invites accept, sessions, identities, email change, profile), `notify` (preferences, email delivery, center API) | 00250–00299 | `seedAccount`, `seedNotifications` |
| W05 | **Composer** (7.4) | Composer, ComposerEmpty, ComposerCode, ComposerMobile, ComposerPreview, ComposerWarnings, ComposerStates, ComposerRequestApproval | `campaigns/new`, `campaigns/$campaignId/edit` | `campaigns` (extend: create, update, autosave, versions, duplicate), `templates`, `liquid` (render with fallbacks), `mjml` (server-side with gomjml), content checks used by Send Readiness | 00300–00349 | `seedComposer` |
| W06 | **Automations** (7.10) | AutomationsList, AutomationFlow, AutomationPublish, AutomationsStates | `_app/automations/index`, `automations/$automationId` | `automations` (graph, versions, diff, publish, runs engine on River, per-step stats, starter templates) | 00350–00399 | `seedAutomations` |
| W07 | **Deliverability & HostRepute** (7.8, A5.15) | Deliverability, DeliverabilityNoHostRepute, DeliverabilityStates, ConnectHostRepute, HostReputeStates, SettingsIntegrations | `_app/deliverability`, `_app/settings/integrations` | `deliverability` (domain/IP health, provider account health, blocklists, Postmaster, SNDS, DMARC, incidents), `hostrepute` (client interface + HTTP client, OAuth/API key, sync job, alerts → notify) , `dnscheck` (miekg/dns behind an interface) | 00400–00449 | `seedDeliverability` |
| W08 | **Landing site & cross-cutting states** (8, 7.14) | Landing, ReadmeBanner, SystemStates, Dialogs, Feedback | `site/` (static landing, prerendered), app-wide 403/404/500/offline/session-expiry/maintenance/quota banners, keyboard shortcuts, background-jobs popover, `docs/assets/readme-banner.*` | `site` (new Vite project), `web/src/components/shell/*` additions for system states | 00450–00499 | — |

## Wave 2 (starts after wave 1 is merged)

| # | Workstream | Boards | Routes | Packages | Migrations | Seeder |
|---|---|---|---|---|---|---|
| W09 | **Overview & reports** (7.1, 7.7) | Main, OverviewEmpty, OverviewStates, Report, ReportStates | `_app/index`, `_app/campaigns/$campaignId/report`, `r.$token` | `overview`, `reports` (headline metrics, 72 h engagement, click map, mailbox/sending provider breakdowns, bounce taxonomy, unsubscribe reasons, A/B with confidence, export, share links) | 00500–00549 | `seedReports` |
| W10 | **Connect a provider & senders** (7.2, 7.13) | SetupCatalog, SetupSES, SetupSMTP, SetupError, SetupDomain, SetupTestSend, SetupStreams, SetupContacts, SetupDone, SettingsSenders, SettingsProviders | `providers/connect/**`, `_app/settings/{senders,providers}` | `domains` (sending domains, senders, live DNS verification), connect flows per adapter, system mailer (`mailer.Sender` over the System route) | 00550–00599 | `seedSenders` |
| W11 | **Review & send, approvals** (7.5, A5.12) | SendReadiness, SendReadinessBlocked, SendReadinessConfirm, SendReadinessStates, SendReadinessApproval, ApprovalInbox, ApprovalReview, ApprovalStates, ApprovalPolicy | `_app/campaigns/$campaignId/review`, `_app/approvals/**`, `_app/settings/approvals` | `readiness` (every check of spec 5.1 with evidence and fixes, estimates, throttling), `approvals` (policy, requests, snapshot + invalidation diff, comments, bypass, send-time recheck) | 00600–00649 | `seedReadiness` |
| W12 | **Sending & live monitor** (5.3, 7.6) | LiveMonitor, LiveMonitorMobile, LiveMonitorPaused, LiveMonitorOutage, LiveMonitorStates | `_app/campaigns/$campaignId/live` | `sending` (implements `sending.Starter`: audience build, suppressions, routing, per-provider and per-mailbox throttles, batches on River, counters, auto-pause, outage hold, failover policy), `tracking` (open pixel with machine-open detection, click redirect, List-Unsubscribe), SSE live stream | 00650–00699 | `seedSending` |
| W13 | **Forms & public pages** (7.12) | FormBuilder, FormsStates, PublicSubscribe, PublicConfirm, PublicPreferences, PublicUnsubscribe, PublicArchive | `_app/forms/**`, server-rendered `/p/{workspace}/…` | `forms` (builder, embed/popup/hosted, spam protection, double opt-in), `publicpages` (Go templates, RFC 8058 one-click, preference center, archive + RSS, SEO) | 00700–00749 | `seedForms` |
| W14 | **Instance admin & SSO** (A5.10, 5.11) | InstanceUsers, InstanceUserDialogs, InstanceAuth, InstanceSettings, SSOSetup, SSOStates, AuditInstance | `_app/admin/**` | `instance` (users, workspaces, policies, version & update check job), `sso` (OIDC with go-oidc: discovery, sign-in, callback, JIT, group mapping, enforcement, test sign-in), social sign-in (Google, Microsoft, GitHub with org restriction) | 00750–00799 | `seedInstance` |

Ports and databases: workstream `WNN` uses API `81NN`, web `52NN`, database `reputecast_wNN`.
