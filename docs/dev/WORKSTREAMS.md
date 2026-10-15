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
| W15 | **More sending adapters** (5.2) | — (the W10 catalog lists them from the registry) | — | `providers/{mailgun,sendgrid,resend,sparkpost,mailjet,brevo,azureacs,httpgeneric}`, `docs/adapters.md` (adapter SDK guide) | 00800–00849 | — |
| W16 | **Community adapter runtime & SDK** (5.2) | ProvidersAdapters ("Build your own adapter") | — | the out-of-process adapter protocol (HTTP + JSON, signed), `providers/remote` (proxies Provider calls to installed community adapters), `sdk/adapter-sdk` (TypeScript `@reputecast/adapter-sdk`: `defineAdapter`, protocol server, contract-test runner), `sdk/adapter-template`, docs | 00850–00899 | — |

Ports and databases: workstream `WNN` uses API `81NN`, web `52NN`, database `reputecast_wNN`.

## Design expansion (M15–M32)

The expansion (`docs/spec/reputecast-design-expansion-plan.md`, one brief per
phase in `docs/spec/reputecast-design-addon-*.md`) is built on the
`expansion` branch by workstreams X01–X23. Workstream `XNN` uses API `83NN`,
web `53NN`, database `reputecast_xNN` and migrations
`01000 + 50·(NN−1)` to `+49` (X01 01000–01049, X02 01050–01099, …).

Rules on top of the shared resources above:

| Thing | Rule |
|---|---|
| Permissions | All of the expansion's keys are in `internal/authz` already (read-only). |
| App text | New strings go through `web/src/i18n` (`CONVENTIONS.md` › Text and app languages), English source only. Vietnamese comes with M18. `web/src/i18n/*` is read-only except for X07. |
| Shell | `web/src/components/shell/*` belongs to X01 in wave 1. |
| Registries, append-only | notification kinds (`internal/notify`), audit filter categories (`internal/auditlog`), webhook events (`internal/webhooks`), MCP tools (`internal/mcpserver`), Send Readiness checks (`internal/readiness`), `web/scripts/qa-map.ts`, `internal/demo/seeders.go`. |
| Existing screens | Change another workstream's screen only where your boards change it, with small insertions; put the new parts in your own feature folder. |
| Docs | Add rows to `docs/DESIGN_INVENTORY.md` (the integrator renumbers them) and a "Done so far (XNN …)" section at the end of `docs/ROADMAP.md`. |

### Wave 1: phases 1 and 2

| # | Workstream | Boards (plus Phone/Tablet) | Owns |
|---|---|---|---|
| X01 | **Shell, navigation and roles** | ShellMore; the sidebar of every app-shell board (Transactional, Grow, Earn); WorkspaceSettings › Sidebar sections; Roles, RoleEditor, RoleEditorStates (every expansion group); the shell layout fixes (avatar initials, tablet rail footer) | `web/src/components/shell/*`, `features/roles`, the workspace's sidebar-section setting, stub routes for Grow and Earn |
| X02 | **Transactional sending** | Transactional, TransactionalStates, TransactionalGuide*, TransactionalMessages, TransactionalMessage(+States), TransactionalSetup(+States), TransactionalSettings, PublicUnsubscribeCategory; changes to Main, ProvidersRouting, SetupStreams, Providers, SettingsSenders, SettingsKeys(+States), SettingsStates, ContactProfile (transactional activity, category opt-outs), Suppressions, PublicPreferences (categories), SettingsAPI (event group), Notifications, SettingsAudit, InstanceSettings (SMTP relay), SettingsMCP | `internal/transactional` (templates, versions, render, send API, log, categories, settings, alerts, retention), `internal/smtprelay`, the Transactional stream in `routing`, per-stream suppression rules, send-only and test keys; routes `_app/transactional/**`, `w/$workspace/transactional/guide` |
| X03 | **Transactional templates and migration** | TransactionalTemplate(+States), TransactionalPublish(+States), TransactionalImport(+States); changes to ApprovalPolicy, MigrationSource (Notifuse), ListHealth, FormBuilder and MigrationDone (welcome-series prompts) | the composer's transactional mode, template publish UI, `internal/transactional/importer` (Postmark, SendGrid, Mailgun, Notifuse, MJML, HTML → Liquid), the Notifuse importer in `internal/imports`, welcome-series prompts; routes `w/$workspace/transactional/templates/**`, `w/$workspace/transactional/import` |
| X04 | **Integrations hub** | IntegrationsCatalog, IntegrationsZapier, IntegrationsConnect, IntegrationsWordPress, IntegrationsStates; changes to SettingsIntegrations (HostRepute's page in the hub), SettingsKeys (managed keys), Dialogs (revoke warning), SettingsAPI (managed endpoint, `form.submitted`), TeamDialogs (take over), Notifications, SettingsAudit, SettingsMCP | `internal/integrations` (catalog, presets, integration keys, REST hooks, activity, WordPress sites), routes `_app/settings/integrations/**` |
| X05 | **Content languages: emails** | SettingsLanguages (Languages card), ComposerLanguages, ComposerCompare, LanguageStates, ReportLanguages, AutomationLanguages; language changes to the composer, Review & send, approvals, the transactional template editor, publish dialog, message log, message detail and setup | `internal/languages` (workspace languages, versions, statuses), language versions in campaigns, automation emails and transactional templates, the Languages check, results per language |
| X06 | **Accessibility checks** | ComposerAccessibility, ComposerContrast, SendReadinessAccessibility, AccessibilityStates, SettingsLanguages (Accessibility card); accessibility changes to the composer, gallery, Review & send, automation and template publishing | `internal/accessibility` (checks on compiled HTML, compile-time `lang`/`dir`/`role`, fixes, workspace policy) |
| X07 | **App languages** | LanguageRegion, the stress-test boards (Campaigns, Review & send, Team in Vietnamese and +40%); changes to AccountSecurity, ShellUserMenu, InstanceSettings and InstanceSetup (default language), TeamInvite (invitation language), SignInStates, AuthScreens, InviteStates | `web/src/i18n` (pseudo-locale, region formats, language loading), `lib/format` region formats, member language and region preferences, server messages and member emails in the member's language |

### Later waves

| # | Workstream | Milestone | Starts after |
|---|---|---|---|
| X08 | Content languages: forms, public pages and contacts (FormBuilderLanguages, the public language menu, contacts' language) | M17 | X05's workspace languages |
| X09 | Integration apps: Zapier, Make, n8n, WordPress plugin | M16 | X04's API |
| X10 | Recurring and RSS digests | M20 | wave 1 |
| X11 | Conditional content, polls and surveys | M21 | wave 1 |
| X12 | AI assistant | M22 | wave 1 |
| X13 | Contact scoring and growth sources | M23 | wave 1 |
| X14 | Address verification and placement tests | M24 | wave 1 |
| X15 | Referrals and recommendations | M25 | X13 |
| X16 | Payments and paid subscriptions | M26 | wave 2 |
| X17 | Offers, gifts, sponsorships and revenue | M27 | X16 |
| X18 | Publication website | M28 | X16 |
| X19 | Landing and sales pages, digital products | M29 | X18 |
| X20 | Shopify and WooCommerce | M30 | X09 |
| X21 | SMS | M31 | wave 2 |
| X22 | Web push and WhatsApp | M32 | X21 |
| X23 | Agency accounts | M32 | wave 2 |
