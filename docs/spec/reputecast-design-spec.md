# ReputeCast — Complete product spec for Claude Design

## 0. How to use this document

This file is the complete brief. Read all of it before you start designing.

- Build one high-fidelity, clickable prototype of the ReputeCast web app. Use app navigation that reaches every screen in section 7.
- You decide the visual style, layout and design system.
- For every screen, include empty, loading, error and success states, plus any specific states listed for that screen.
- All copy in English. Use the sample data in section 6 everywhere.
- Build order: 7.1 Overview → 7.5 Review & send (Send Readiness) → 7.6 Live sending monitor → 7.3 Sending providers & routing → 7.2 Connect a sending provider → 7.4 Campaign composer → the rest in order. If the work needs several passes, follow this order and keep going until every screen exists.
- Section 8 (landing page and README banner) is a separate deliverable. Do it after the app.
- Accounts, teams, access control, SSO and send approvals are specified in a companion document, "ReputeCast — Add-on spec: accounts, teams, access and send approvals". If it is attached, build it too, after the screens in this document.

---

## 1. Product

ReputeCast is an open-source, self-hosted newsletter and email marketing platform that sends through the user's own email sending provider.

Amazon SES is first-class, but ReputeCast is provider-agnostic. It plugs into Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, a self-hosted Postal server or any SMTP server, and it can use several providers at once.

It is a sibling product of HostRepute (hostrepute.com), an email-reputation monitoring service. That is its edge: it is the only newsletter tool that tells you whether your mail will reach the inbox BEFORE you send, and protects your sender reputation automatically WHILE sending, whichever provider delivers it.

The goal is a product that people choose over every alternative in section 4.

## 2. Terms

- **Sending provider:** the service that delivers the mail (Amazon SES, Postmark, Mailgun, Postal, SMTP…).
- **Mailbox provider:** where the recipient reads the mail (Gmail, Outlook, Yahoo, iCloud…).
- **Stream:** the kind of mail.
  - Broadcasts: campaigns.
  - Automations: emails sent by flows.
  - System: double opt-in confirmations, preference-center links, team invites.
- **Route:** the rule that decides which sending provider delivers a given email.

Always label a "per provider" breakdown as either mailbox provider or sending provider.

## 3. Who it's for

Indie publishers, SaaS founders and small marketing teams leaving Brevo or Mailchimp ($40–300/month). They want to self-host on a sending provider they control: SES at about $0.10 per 1,000 emails, a provider they already pay for, or their own Postal server.

They are technical enough to run Docker, but they expect a polished product. They don't want to be locked into one provider.

## 4. Competitors and their gaps

- **listmonk:** fast, but the UI is dated, there are no automations, and segments require raw SQL. Sends via SMTP servers or custom HTTP "messengers".
- **Keila:** good editor and several sender adapters, but no automations.
- **Notifuse:** modern and supports several providers, but extra workspaces, SSO, audit logs and template translations need a paid licence.
- **Mautic:** powerful automations, but heavy and hard to use. SES needs a community plugin.
- **Plunk:** Amazon SES only, and SES setup (IAM, SNS, configuration sets) is manual.
- None of them checks deliverability before or during a send.

## 5. Product requirements

### 5.1 Send Readiness
A pre-send check covering:
- **Sender reputation:** domain and IP blacklists across 80+ sources via HostRepute. Also the bounce and complaint rates of every sending provider in the route, compared with that provider's limits (for example, AWS SES review thresholds).
- **Authentication:** SPF, DKIM, DMARC, custom return-path / MAIL FROM and BIMI, checked against the DNS records the chosen provider requires.
- **Content:** spam triggers, image-to-text ratio, Gmail's 102 KB clipping limit, alt text, plain-text version.
- **Links:** broken links, and links on URI blocklists.
- **Compliance:** one-click unsubscribe header, footer address, Gmail/Yahoo bulk-sender rules.
- **Audience quality:** unengaged, recently bounced and role addresses.
- **Routing:** every stream has a healthy provider, and the provider's terms allow bulk marketing. Provider-side open/click tracking won't double-track.

Each check is Pass, Warn or Block, and each comes with a fix. Send Readiness also shows:
- estimated cost per sending provider
- estimated send duration, based on each provider's rate limits
- throttling per mailbox provider

### 5.2 Pluggable sending providers
- **Built-in adapters:** Amazon SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, Postal (self-hosted) and generic SMTP.
- **More adapters:** developers can add more through an open adapter SDK or a generic HTTP adapter. Community adapters are listed with their version and maintainer.
- **Capability badges on each provider:**
  - bulk marketing allowed by its terms
  - bounce/complaint webhooks
  - domain setup via API
  - suppression-list sync
  - batch sending
  - dedicated IPs
  - sandbox mode
  - quota/rate discovery
  - pricing model
- **Normalized events:** every provider's events (delivered, deferred, hard/soft bounce, complaint, rejected) map to one model that keeps the original provider code. Providers without webhooks, such as plain SMTP, report through a bounce mailbox.
- **Shared suppression list:** one list across all providers. A bounce at one provider stops mail to that address everywhere.
- **Routing:**
  - several providers per workspace
  - rules per stream, per list or sender domain, and per recipient mailbox provider
  - weighted splits (for example 90/10) for migration and warm-up
- **Failover:**
  - Automatic for System and Automations.
  - Broadcasts default to pause-and-ask, because moving a bulk send to a cold provider mid-send can hurt reputation. Users can allow Broadcast failover to a provider that is already warmed up.
- **Warm-up planner** for a new provider or dedicated IP: daily volume ramp, most-engaged contacts first, and an automatic slowdown if bounces rise.
- **Health per provider:**
  - status: Healthy, Degraded, Down, Sandbox or Warming up
  - quota and rate usage
  - webhook freshness
  - error rate
  - cost to date

### 5.3 Guarded sending
- Live progress with delivered, bounced and complained counts. Break them down per mailbox provider (Gmail, Outlook, Yahoo, iCloud, Other) and per sending provider.
- When bounces or complaints exceed user-set thresholds, the campaign auto-pauses, either as a whole or just on one route.
- Provider outages (API errors, throttling) trigger the same handling.
- ReputeCast explains what happened and offers resume, adjust, fail over or cancel.

### 5.4 Deliverability
- Account health per sending provider (for example, SES account health and enforcement status), plus whatever reputation data each provider's API exposes.
- Blacklist status for sending domains and dedicated IPs. Shared IP pools are labeled as shared.
- Google Postmaster Tools, Microsoft SNDS, DMARC reports and an incident timeline.
- Works without HostRepute, and is richer when HostRepute is connected.

### 5.5 Honest analytics
- Human opens separated from machine opens (Apple Mail Privacy Protection).
- A click map on the email.
- Bounces by type, with each provider's original diagnostic code mapped to one taxonomy.
- Complaints and unsubscribe reasons.
- Results per mailbox provider and per sending provider, side by side when a send was split.
- A/B results.

### 5.6 List health
- Contacts tiered Active, Cooling, At risk and Sunset.
- A sunset policy.
- One suppression list shared by every provider, which can import each provider's existing suppressions.

### 5.7 Provider onboarding in about 10 minutes
The user picks a provider from a catalog. The connection step depends on the provider:
- **Amazon SES:** CloudFormation or IAM keys, auto-created SNS topics and configuration sets, sandbox detection and a guided production-access request.
- **API providers:** paste an API key. ReputeCast registers webhooks and fetches the required DNS records automatically where the provider's API allows it.
- **Postal:** server URL and API key.
- **SMTP:** host, port, TLS, credentials and a bounce mailbox.

Every path ends with DNS records with live verification, a test send, and assigning the provider to streams.

### 5.8 Migration
Import from Brevo, Mailchimp, MailerLite, listmonk, Keila or CSV. Unsubscribed and bounced contacts stay suppressed, and existing suppressions can be pulled from connected providers.

### 5.9 Composer
- The drag-and-drop email builder is built on the open-source GrapesJS framework with its MJML plugin (grapesjs-mjml). ReputeCast's own UI wraps the GrapesJS canvas.
- Modes: Visual (GrapesJS + MJML), MJML code, HTML.
- Previews by device, email client and dark mode, plus an inbox preview of subject and preheader.
- Personalization with fallbacks, and A/B variants.

### 5.10 Segments
A condition builder with a live count, plus the generated SQL for power users.

### 5.11 Automations
- Visual flows built on the open-source React Flow library (@xyflow/react).
- Steps: triggers, delays, emails, branches, A/B splits, webhooks, list actions, goals.
- Live stats per step.

### 5.12 Forms and public pages
Signup forms, double opt-in, preference center, unsubscribe page, public newsletter archive.

### 5.13 Team and integrations
- Roles and permissions, SSO, API keys, webhooks, an MCP server for AI agents, and an audit log.
- Every feature is free; nothing is locked behind a paid tier.
- Full detail (sign-in, 2FA, passkeys, SSO, roles, permissions, send approvals, audit log, Connect HostRepute) is in the add-on spec.

---

## 6. Sample data

**Workspace:** "Acme Newsletter"
- Sending domain: news.acme.io
- 48,213 subscribers in 3 lists: Weekly Digest, Product Updates, Beta Program.

**Sending providers**
- Amazon SES us-east-1: Broadcasts and Automations. Quota 50,000 per 24h, max rate 14/s. Healthy.
- Postmark: System stream. Healthy.
- Postal (self-hosted) at mta.acme.io, dedicated IP 203.0.113.25: Warming up, day 9 of 21. Receives 10% of Broadcasts.

**Routing**
- Broadcasts: 90% SES / 10% Postal.
- Automations: SES, with failover to Postmark.
- System: Postmark, with failover to SES.

**Last campaign: "October Product Update"**
- 47,980 sent (SES 43,182, Postal 4,798)
- 99.1% delivered
- 41.2% human opens, 18.6% machine opens
- 6.3% clicks
- 0.6% bounces, 0.04% complaints
- Mailbox providers: Gmail 46%, Outlook 21%, Yahoo 9%, iCloud 8%, Other 16%.

---

## 7. Screens

### 7.1 Overview
Shows:
- overall sender health and its reasons
- the next scheduled campaign and its readiness
- health of each sending provider: status, bounce and complaint rates against that provider's limits, quota and rate usage
- subscriber growth
- engagement tiers
- recent campaigns

Include the first-run empty state that leads into connecting a sending provider.

### 7.2 Connect a sending provider (setup wizard)
First-run setup, in six steps:

1. **Provider catalog:** Amazon SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, Postal, SMTP, and community adapters. Each entry shows capability badges, pricing model, and whether its terms allow bulk marketing.
2. **Connection,** which depends on the provider:
   - Amazon SES: connect AWS via CloudFormation or IAM keys, then choose a region. SNS topics and configuration sets are provisioned automatically, with progress and retry. Sandbox mode is detected, with a guided production-access request.
   - API providers: paste an API key, then pick the region, server or message stream. Webhooks are registered automatically.
   - Postal: server URL and API key.
   - SMTP: host, port, TLS and credentials, plus a bounce mailbox (IMAP/POP3), because SMTP has no event webhooks.
3. **Domain verification:** the DNS records this provider requires (DKIM, return-path / MAIL FROM, SPF, DMARC), with live status.
4. **Test send:** watch the delivery event arrive.
5. **Streams:** assign the provider to Broadcasts, Automations and/or System.
6. **Contacts:** import contacts or start empty.

End with a summary of what was configured and an "Add another provider" option.

States to show:
- the SES path
- the SMTP path
- an error state: invalid API key, or webhook endpoint not reachable

### 7.3 Sending providers & routing
- **Provider list.** For each provider:
  - status: Healthy, Degraded, Down, Sandbox or Warming up
  - quota and rate usage
  - bounce and complaint rates against that provider's limits
  - webhook freshness (last event received) and error rate
  - capability badges
  - shared or dedicated IPs
  - cost to date and per 1,000 emails, from the provider's API or a user-entered plan price
- **Routing rules per stream** (Broadcasts, Automations, System): primary provider, fallback, conditions (list, sender domain, recipient mailbox provider) and weighted splits.
- **Route tester:** answers "Which provider would send this email?" for a sample recipient.
- **Failover policy per stream:** automatic for System and Automations. Broadcasts default to pause-and-ask, with an option to fail over only to a warmed-up provider.
- **Warm-up planner** for a new provider or dedicated IP: daily volume ramp, engaged-first ordering, progress, and an automatic slowdown when bounces rise.
- **Event log:** each raw provider event next to the normalized ReputeCast event and the original code.
- **Suppression sync:** sync of the shared suppression list with each provider.
- **Provider-side tracking check:** warns when a provider rewrites links or adds its own open pixel, and offers to turn that off.
- **Adapters:** installed built-in and community adapters with version, maintainer and update status; a generic HTTP adapter; a link to the adapter SDK.

States to show:
- a healthy multi-provider setup using the sample data
- a provider outage: failover active for Automations and System, while Broadcasts wait for a decision
- webhooks silent for 6 hours
- a warning when Broadcasts are routed to a provider whose terms forbid bulk marketing

### 7.4 Campaign composer (GrapesJS + MJML)
The drag-and-drop builder is NOT custom-built. It uses the open-source GrapesJS framework (github.com/GrapesJS/grapesjs) with the grapesjs-mjml plugin, the same engine Mautic uses for its email builder.

Do not reproduce GrapesJS's default editor look. Design ReputeCast's own interface around the GrapesJS canvas. GrapesJS supports a fully custom UI via its API and the official @grapesjs/react wrapper.

The UI must map to what GrapesJS actually provides:
- **Canvas:** the email rendered live in an iframe. Click to select a component, drag to reorder, and use a toolbar on the selected component (move, duplicate, delete, select parent).
- **Blocks panel** (GrapesJS Block Manager):
  - MJML blocks: 1 / 2 / 3 columns (mj-section + mj-column), Text, Image, Button, Divider, Spacer, Social links, Navbar, Hero, Wrapper, Raw HTML.
  - ReputeCast blocks, built as custom GrapesJS blocks: Header with logo, Article card, Product card, Footer with address + unsubscribe link, saved sections.
- **Layers panel** (GrapesJS Layer Manager): the mjml > mj-body > mj-section > mj-column > element tree.
- **Style panel** (GrapesJS Style Manager), limited to properties MJML supports: background, padding, alignment, width, border, border radius, font family / size / color, line height.
- **Settings panel** (GrapesJS Trait Manager): link URL, image alt text, button text, CSS class, hide on mobile or desktop.
- **Inline rich text editing** inside Text blocks (GrapesJS Rich Text Editor): bold, italic, link, personalization token insertion.
- **Device switcher** (GrapesJS Device Manager): desktop 600px, mobile 375px.
- **Asset manager** (GrapesJS Asset Manager): upload and image library.
- **Code view** (GrapesJS Code Manager / MJML import): edit MJML directly and see the compiled HTML.
- **Undo / redo** (GrapesJS Undo Manager) and keyboard shortcuts (GrapesJS Keymaps).

ReputeCast features around the builder:
- **Header:** campaign name, audience picker with live count, sender identity, sending route (which provider or split will deliver it), status, "Review & send".
- **Subject and preheader** fields, with an inbox preview.
- **Previews** by email client (Gmail, Outlook, Apple Mail) and in dark mode.
- **Personalization tokens** with fallbacks, e.g. `{{ first_name | default: "there" }}` (Liquid syntax).
- **A/B variants:** subject, content, send time.
- **Inline deliverability warnings on the canvas:** image missing alt text, size approaching Gmail's 102 KB clipping limit, link on a blocklist, missing unsubscribe link.
- **Templates and history:** template gallery (start from a saved MJML template), saved reusable sections, autosave, version history.

States to show:
- empty canvas (choose a template or start blank)
- editing a 2-column section with the Style panel open
- mobile preview
- code view
- the inline warning state

### 7.5 Review & send (Send Readiness)
- **Send Readiness checklist:** the full checklist, with evidence and one-click fixes.
- **Verdict:** Ready, Ready with warnings, or Blocked.
- **Audience:** final count after exclusions.
- **Route breakdown:** how many recipients each sending provider will deliver.
- **Estimates:** cost per provider, and duration based on each provider's rate limit.
- **Throttling** per mailbox provider.
- **Editable auto-pause thresholds.**
- **Scheduling:** now, a date and time with timezone, or per-recipient best time.
- **Test sends.**

Confirming a large send must be deliberate.

States to show:
- the Blocked state

### 7.6 Live sending monitor
- **Progress:** progress and ETA.
- **Real-time counts:** delivered, bounced, complained, opened and clicked.
- **Breakdowns:** per mailbox provider and per sending provider.
- **Thresholds:** current values against each threshold.

States to show:
- **Auto-paused,** e.g. "Outlook complaint rate reached 0.18% (limit 0.10%)". Show what tripped, when, likely causes and recommended actions. Actions: resume, resume with changes, or cancel.
- **Sending-provider outage,** e.g. "Postal returned connection errors for 3 minutes". The Postal share is held while SES continues. Actions: retry, move the remaining Postal share to SES, or cancel.
- a mobile version

### 7.7 Campaign report
- **Headline metrics:** delivered, human vs machine opens, clicks, click-to-open, unsubscribes, complaints, bounces.
- **Engagement over time:** the first 72 hours.
- **Click map** on the email.
- **Results per mailbox provider.**
- **Results per sending provider,** side by side when the send was split.
- **Bounces by type:** each provider's original code mapped to one taxonomy, plus add-to-suppression.
- **Unsubscribe reasons.**
- **A/B results** with confidence.
- **Export** and a shareable read-only link.

### 7.8 Deliverability
- **Health per sending domain and dedicated IP.** Shared IP pools are labeled as shared and checked at domain level.
- **Account health per sending provider,** with that provider's thresholds (e.g. SES review and probation levels).
- **Blacklist status** across 80+ sources, with delisting links.
- **Google Postmaster data** and **Microsoft SNDS data.**
- **DMARC report summary.**
- **Incident timeline,** linked to campaigns and providers.

States to show:
- HostRepute not connected: provider data is still shown, with an invitation to connect and no paywall.

### 7.9 Audience
- **Contacts table** with saved views, filters and bulk actions.
- **Contact profile** with custom fields, per-list consent history, and an activity timeline that includes which provider delivered each email.
- **Segment builder** with AND/OR groups, live count and generated SQL.
- **List health** with tier trends and the sunset policy.
- **Suppression list,** shared across all providers, showing reasons and the source provider.

### 7.10 Automations
The flow canvas uses the open-source React Flow library (@xyflow/react). Design custom nodes and edges on a React Flow canvas, with pan, zoom, minimap, controls, snap to grid and connection handles.

- **Triggers:** joined list, form submitted, tag added, API event, date field.
- **Steps:** delay, send email (uses the Automations route unless overridden), branch, filter, A/B split, webhook, list actions, goal/exit.
- **Live stats** per step.
- **Lifecycle:** Draft, Live and Paused, with publish and a version diff.
- **Automation list** with performance.
- **Starter templates:** welcome series, re-engagement, onboarding.

### 7.11 Migration importer
1. Pick a source: Brevo, Mailchimp, MailerLite, listmonk, Keila, or CSV/XLSX.
2. Map fields.
3. Map consent status, so unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed. Optionally pull existing suppressions from connected sending providers.
4. Preview duplicates, and invalid or risky addresses.
5. Show progress, then a summary with a downloadable report of skipped rows.

### 7.12 Forms & public pages
- **Signup form builder:** embedded, popup and hosted forms, with double opt-in, spam protection and embed code.
- **Public pages,** branded by the publisher:
  - subscribe
  - opt-in confirmation
  - preference center (lists, frequency, pause)
  - unsubscribe, with a "fewer emails" option
  - public newsletter archive

### 7.13 Settings
- **Senders and domains,** with verification status per sending provider.
- **Sending providers,** with a summary and a link to 7.3.
- **Team** and custom roles, with SSO.
- **API keys,** and webhooks with delivery logs.
- **MCP server,** with per-agent permissions and recent agent actions.
- **Audit log.**

Everything is available in the open-source edition. The add-on spec splits Settings into Workspace settings and an Instance admin area and details each part.

---

## 8. Separate deliverable: landing page and README banner

Design the reputecast.com landing page and a GitHub README banner.

**Message:** the newsletter platform that protects your inbox placement. Open source, self-hosted, and it sends through your own Amazon SES, Postmark, Mailgun, SendGrid, Resend, Postal or any SMTP server, with no lock-in and several providers at once.

Include:
- an honest comparison with listmonk, Keila, Notifuse, Mautic and Brevo
- a cost calculator comparing providers at the user's volume, with editable prices
- a one-command docker compose install
- features
- "Built by HostRepute"
- GitHub and Discord calls to action
