# ReputeCast — Claude Design prompts (product only, no style direction)

Paste PROMPT 0 first. Then paste the screen prompts one at a time in the same project.
If you already started a Claude Design project with the old SES-only brief, paste the UPDATE PROMPT instead of starting over.

---

## PROMPT 0 — Product brief

```
Design ReputeCast, an open-source, self-hosted newsletter and email marketing platform that sends through the user's own email sending provider. Amazon SES is first-class, but ReputeCast is provider-agnostic. It plugs into Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, a self-hosted Postal server or any SMTP server, and it can use several providers at once. It is a sibling product of HostRepute (hostrepute.com), an email-reputation monitoring service. That is its edge: it is the only newsletter tool that tells you whether your mail will reach the inbox BEFORE you send, and protects your sender reputation automatically WHILE sending, whichever provider delivers it.

You decide the visual style, layout and design system. The goal is a product that people choose over every alternative below.

TERMS USED IN THESE PROMPTS
- Sending provider: the service that delivers the mail (Amazon SES, Postmark, Mailgun, Postal, SMTP…).
- Mailbox provider: where the recipient reads the mail (Gmail, Outlook, Yahoo, iCloud…).
- Stream: the kind of mail. Broadcasts (campaigns), Automations (emails sent by flows) and System (double opt-in confirmations, preference-center links, team invites).
- Route: the rule that decides which sending provider delivers a given email.

WHO IT'S FOR
Indie publishers, SaaS founders and small marketing teams leaving Brevo or Mailchimp ($40–300/month) to self-host on a sending provider they control: SES at about $0.10 per 1,000 emails, a provider they already pay for, or their own Postal server. Technical enough to run Docker, but they expect a polished product. They don't want to be locked into one provider.

COMPETITORS AND THEIR GAPS
- listmonk: fast, but dated UI, no automations, and segments require raw SQL. Sends via SMTP servers or custom HTTP "messengers".
- Keila: good editor and several sender adapters, but no automations.
- Notifuse: modern and supports several providers, but extra workspaces, SSO, audit logs and template translations need a paid licence.
- Mautic: powerful automations, but heavy and hard to use. SES needs a community plugin.
- Plunk: Amazon SES only, and SES setup (IAM, SNS, configuration sets) is manual.
- None of them checks deliverability before or during a send.

PRODUCT REQUIREMENTS
1. Send Readiness: a pre-send check covering:
   - Sender reputation: domain and IP blacklists across 80+ sources via HostRepute, plus the bounce and complaint rates of every sending provider in the route against that provider's limits (for example, AWS SES review thresholds).
   - Authentication: SPF, DKIM, DMARC, custom return-path / MAIL FROM and BIMI, checked against the DNS records the chosen provider requires.
   - Content: spam triggers, image-to-text ratio, Gmail's 102 KB clipping limit, alt text, plain-text version.
   - Links: broken links, links on URI blocklists.
   - Compliance: one-click unsubscribe header, footer address, Gmail/Yahoo bulk-sender rules.
   - Audience quality: unengaged, recently bounced and role addresses.
   - Routing: every stream has a healthy provider, the provider's terms allow bulk marketing, and provider-side open/click tracking won't double-track.
   Each check is Pass, Warn or Block and comes with a fix. Send Readiness also shows estimated cost per sending provider, estimated send duration based on each provider's rate limits, and throttling per mailbox provider.
2. Pluggable sending providers:
   - Built-in adapters: Amazon SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, Postal (self-hosted) and generic SMTP. Developers can add more through an open adapter SDK or a generic HTTP adapter. Community adapters are listed with their version and maintainer.
   - Capability badges on each provider: bulk marketing allowed by its terms, bounce/complaint webhooks, domain setup via API, suppression-list sync, batch sending, dedicated IPs, sandbox mode, quota/rate discovery, pricing model.
   - Normalized events: every provider's events (delivered, deferred, hard/soft bounce, complaint, rejected) map into one model that keeps the original provider code. Providers without webhooks, such as plain SMTP, report through a bounce mailbox.
   - One suppression list shared across all providers. A bounce at one provider stops mail to that address everywhere.
   - Routing: several providers per workspace, with rules per stream, per list or sender domain, and per recipient mailbox provider. Weighted splits (for example 90/10) support migration and warm-up.
   - Failover: automatic for System and Automations. For Broadcasts the default is pause-and-ask, because moving a bulk send to a cold provider mid-send can hurt reputation. Users can allow Broadcast failover to a provider that is already warmed up.
   - Warm-up planner for a new provider or dedicated IP: daily volume ramp, most-engaged contacts first, and an automatic slowdown if bounces rise.
   - Health per provider: status (Healthy, Degraded, Down, Sandbox, Warming up), quota and rate usage, webhook freshness, error rate and cost to date.
3. Guarded sending: live progress with delivered, bounced and complained counts per mailbox provider (Gmail, Outlook, Yahoo, iCloud, Other) and per sending provider. When bounces or complaints exceed user-set thresholds, the campaign auto-pauses as a whole or just on one route. Provider outages (API errors, throttling) trigger the same handling. ReputeCast explains what happened and offers resume, adjust, fail over or cancel.
4. Deliverability:
   - Account health per sending provider: for example, SES account health and enforcement status, plus whatever reputation data each provider's API exposes.
   - Blacklist status for sending domains and dedicated IPs. Shared IP pools are labeled as shared.
   - Google Postmaster Tools, Microsoft SNDS, DMARC reports and an incident timeline.
   - Works without HostRepute and is richer when HostRepute is connected.
5. Honest analytics:
   - Human opens separated from machine opens (Apple Mail Privacy Protection), and a click map on the email.
   - Bounces by type, with each provider's original diagnostic code mapped to one taxonomy. Complaints and unsubscribe reasons.
   - Results per mailbox provider and per sending provider, with a side-by-side comparison when a send was split. A/B results.
6. List health: contacts tiered Active, Cooling, At risk, Sunset. A sunset policy. One suppression list shared by every provider, which can import each provider's existing suppressions.
7. Provider onboarding in about 10 minutes. The user picks a provider from a catalog, then:
   - Amazon SES: CloudFormation or IAM keys, auto-created SNS topics and configuration sets, sandbox detection and a guided production-access request.
   - API providers: paste an API key. ReputeCast registers webhooks and fetches the required DNS records automatically where the provider's API allows it.
   - Postal: server URL and API key.
   - SMTP: host, port, TLS, credentials and a bounce mailbox.
   Every path ends with DNS records with live verification, a test send, and assigning the provider to streams.
8. Migration from Brevo, Mailchimp, MailerLite, listmonk, Keila or CSV. Unsubscribed and bounced contacts stay suppressed, and existing suppressions can be pulled from connected providers.
9. Composer: the drag-and-drop email builder is built on the open-source GrapesJS framework with its MJML plugin (grapesjs-mjml). ReputeCast's own UI wraps the GrapesJS canvas.
   - Modes: Visual (GrapesJS + MJML), MJML code, HTML.
   - Previews by device, email client and dark mode, plus an inbox preview of subject and preheader.
   - Personalization with fallbacks, and A/B variants.
10. Segments: condition builder with live count, plus the generated SQL for power users.
11. Automations: visual flows built on the open-source React Flow library (@xyflow/react). Steps include triggers, delays, emails, branches, A/B splits, webhooks, list actions and goals, with live stats per step.
12. Forms and public pages: signup forms, double opt-in, preference center, unsubscribe page, public newsletter archive.
13. Team and integrations: roles and permissions, SSO, API keys, webhooks, an MCP server for AI agents, audit log. Every feature is free; nothing is locked behind a paid tier.

SAMPLE DATA
Workspace "Acme Newsletter". Sending domain news.acme.io. 48,213 subscribers in 3 lists (Weekly Digest, Product Updates, Beta Program).
Sending providers:
- Amazon SES us-east-1: Broadcasts and Automations, quota 50,000 per 24h, max rate 14/s, Healthy.
- Postmark: System stream, Healthy.
- Postal (self-hosted) at mta.acme.io, dedicated IP 203.0.113.25: Warming up, day 9 of 21, receives 10% of Broadcasts.
Routing: Broadcasts 90% SES / 10% Postal; Automations SES with failover to Postmark; System Postmark with failover to SES.
Last campaign "October Product Update": 47,980 sent (SES 43,182, Postal 4,798), 99.1% delivered, 41.2% human opens, 18.6% machine opens, 6.3% clicks, 0.6% bounces, 0.04% complaints. Mailbox providers: Gmail 46%, Outlook 21%, Yahoo 9%, iCloud 8%, Other 16%.

DELIVERABLE
A high-fidelity clickable prototype. Start with these screens:
- the Overview dashboard
- Send Readiness
- the live sending monitor, including the auto-paused state
- Sending providers & routing
Include empty, loading, error and success states. All copy in English.
```

---

## UPDATE PROMPT (only if you already designed with the SES-only brief)

```
Update to the product brief: ReputeCast is no longer tied to Amazon SES. It is provider-agnostic, and SES is one first-class option among several. Built-in adapters: Amazon SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, self-hosted Postal and generic SMTP. Developers can add more through an adapter SDK or a generic HTTP adapter.

New concepts:
- Sending provider (who delivers) vs mailbox provider (Gmail, Outlook…).
- Streams: Broadcasts, Automations, System.
- Routing rules per stream, list, sender domain or recipient mailbox provider, with weighted splits.
- Failover: automatic for System and Automations; pause-and-ask for Broadcasts.
- Warm-up planner, normalized events with the original provider codes, and one suppression list shared across providers.
- Capability badges per provider, and health per provider.

Apply this across the existing screens:
- Replace SES-only wording (SES cost, SES codes, SES account health, SES setup) with per-provider equivalents.
- Label every "per provider" breakdown as mailbox provider or sending provider.
- Turn the SES setup wizard into "Connect a sending provider".
- Add a "Sending providers & routing" screen.

Use the new sample data: SES for Broadcasts and Automations, Postmark for System, and a Postal server warming up with 10% of Broadcasts.
```

---

## Screen prompts (paste one at a time)

**1. Overview**
```
Design the Overview: overall sender health and its reasons, the next scheduled campaign and its readiness, health of each sending provider (status, bounce and complaint rates against that provider's limits, quota and rate usage), subscriber growth, engagement tiers, recent campaigns. Include the first-run empty state that leads into connecting a sending provider.
```

**2. Connect a sending provider (setup wizard)**
```
Design first-run setup for connecting a sending provider.

1. Provider catalog: Amazon SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, Postal, SMTP, and community adapters. Each entry shows capability badges, pricing model and whether its terms allow bulk marketing.
2. Connection, which depends on the provider:
   - Amazon SES: connect AWS via CloudFormation or IAM keys, choose a region, auto-provision SNS topics and configuration sets with progress and retry, and detect sandbox mode with a guided production-access request.
   - API providers: paste an API key, pick region / server / message stream, and auto-register webhooks.
   - Postal: server URL and API key.
   - SMTP: host, port, TLS and credentials, plus a bounce mailbox (IMAP/POP3) because SMTP has no event webhooks.
3. Domain verification: the DNS records this provider requires (DKIM, return-path / MAIL FROM, SPF, DMARC), with live status.
4. Test send: watch the delivery event arrive.
5. Streams: assign the provider to Broadcasts, Automations and/or System.
6. Contacts: import contacts or start empty.

End with a summary of what was configured and an "Add another provider" option. Show the SES path, the SMTP path, and an error state (invalid API key, webhook endpoint not reachable).
```

**2b. Sending providers & routing**
```
Design the Sending providers & routing screen.

- Provider list. For each provider:
  - status (Healthy, Degraded, Down, Sandbox, Warming up)
  - quota and rate usage
  - bounce and complaint rates against that provider's limits
  - webhook freshness (last event received) and error rate
  - capability badges
  - shared or dedicated IPs
  - cost to date and per 1,000 emails, from the provider's API or a user-entered plan price
- Routing rules per stream (Broadcasts, Automations, System): primary provider, fallback, conditions (list, sender domain, recipient mailbox provider) and weighted splits. A tester answers "Which provider would send this email?" for a sample recipient.
- Failover policy per stream: automatic for System and Automations. Broadcasts default to pause-and-ask, with an option to fail over only to a warmed-up provider.
- Warm-up planner for a new provider or dedicated IP: daily volume ramp, engaged-first ordering, progress, and an automatic slowdown when bounces rise.
- Event log: each raw provider event next to the normalized ReputeCast event and the original code.
- Sync of the shared suppression list with each provider.
- Provider-side tracking check: warn when a provider rewrites links or adds its own open pixel, and offer to turn that off.
- Adapters: installed built-in and community adapters with version, maintainer and update status; a generic HTTP adapter; a link to the adapter SDK.

Show these states:
- a healthy multi-provider setup using the sample data
- a provider outage, with failover active for Automations and System while Broadcasts wait for a decision
- webhooks silent for 6 hours
- a warning when Broadcasts are routed to a provider whose terms forbid bulk marketing
```

**3. Campaign composer (GrapesJS + MJML)**
```
Design the campaign composer. The drag-and-drop builder is NOT custom-built. It uses the open-source GrapesJS framework (github.com/GrapesJS/grapesjs) with the grapesjs-mjml plugin, the same engine Mautic uses for its email builder. Do not reproduce GrapesJS's default editor look. Design ReputeCast's own interface around the GrapesJS canvas (GrapesJS supports a fully custom UI via its API and the official @grapesjs/react wrapper).

The UI must map to what GrapesJS actually provides:
- Canvas: the email rendered live in an iframe. Click to select a component, drag to reorder, and use a toolbar on the selected component (move, duplicate, delete, select parent).
- Blocks panel (GrapesJS Block Manager):
  - MJML blocks: 1 / 2 / 3 columns (mj-section + mj-column), Text, Image, Button, Divider, Spacer, Social links, Navbar, Hero, Wrapper, Raw HTML.
  - ReputeCast blocks, built as custom GrapesJS blocks: Header with logo, Article card, Product card, Footer with address + unsubscribe link, saved sections.
- Layers panel (GrapesJS Layer Manager): the mjml > mj-body > mj-section > mj-column > element tree.
- Style panel (GrapesJS Style Manager), limited to properties MJML supports: background, padding, alignment, width, border, border radius, font family / size / color, line height.
- Settings panel (GrapesJS Trait Manager): link URL, image alt text, button text, CSS class, hide on mobile or desktop.
- Inline rich text editing inside Text blocks (GrapesJS Rich Text Editor): bold, italic, link, personalization token insertion.
- Device switcher (GrapesJS Device Manager): desktop 600px, mobile 375px.
- Asset manager (GrapesJS Asset Manager): upload and image library.
- Code view (GrapesJS Code Manager / MJML import): edit MJML directly and see the compiled HTML.
- Undo / redo (GrapesJS Undo Manager) and keyboard shortcuts (GrapesJS Keymaps).

ReputeCast features around the builder:
- Header: campaign name, audience picker with live count, sender identity, sending route (which provider or split will deliver it), status, "Review & send".
- Subject and preheader fields with an inbox preview.
- Previews by email client (Gmail, Outlook, Apple Mail) and in dark mode.
- Personalization tokens with fallbacks, e.g. {{ first_name | default: "there" }} (Liquid syntax).
- A/B variants (subject, content, send time).
- Inline deliverability warnings on the canvas: image missing alt text, size approaching Gmail's 102 KB clipping limit, link on a blocklist, missing unsubscribe link.
- Template gallery (start from a saved MJML template), saved reusable sections, autosave, version history.

Show these states:
- empty canvas (choose a template or start blank)
- editing a 2-column section with the Style panel open
- mobile preview
- code view
- the inline warning state
```

**4. Review & send**
```
Design the review and send flow:
- The full Send Readiness checklist with evidence and one-click fixes.
- An overall verdict: Ready, Ready with warnings, or Blocked.
- Final audience count after exclusions.
- Route breakdown: how many recipients each sending provider will deliver.
- Estimated cost per provider, and estimated duration based on each provider's rate limit.
- Throttling per mailbox provider.
- Editable auto-pause thresholds.
- Scheduling: now, a date and time with timezone, or per-recipient best time.
- Test sends.

Confirming a large send must be deliberate. Show the Blocked state.
```

**5. Live sending monitor**
```
Design the live sending monitor: progress and ETA; real-time delivered, bounced, complained, opened and clicked; breakdowns per mailbox provider and per sending provider; current values against each threshold.

Then show two problem states:
- Auto-paused: e.g. "Outlook complaint rate reached 0.18% (limit 0.10%)". Show what tripped, when, likely causes and recommended actions, with resume, resume with changes, or cancel.
- Sending-provider outage: e.g. "Postal returned connection errors for 3 minutes". The Postal share is held while SES continues, with options to retry, move the remaining Postal share to SES, or cancel.

Include a mobile version.
```

**6. Campaign report**
```
Design the campaign report:
- Delivered, human vs machine opens, clicks, click-to-open, unsubscribes, complaints, bounces.
- Engagement over the first 72 hours.
- Click map on the email.
- Results per mailbox provider.
- Results per sending provider, side by side when the send was split.
- Bounces by type, with each provider's original code mapped to one taxonomy, plus add-to-suppression.
- Unsubscribe reasons.
- A/B results with confidence.
- Export and a shareable read-only link.
```

**7. Deliverability**
```
Design the Deliverability section:
- Health per sending domain and dedicated IP. Shared IP pools are labeled as shared and checked at domain level.
- Account health per sending provider with that provider's thresholds (e.g. SES review and probation levels).
- Blacklist status across 80+ sources, with delisting links.
- Google Postmaster data and Microsoft SNDS data.
- DMARC report summary.
- An incident timeline linked to campaigns and providers.

Show the state where HostRepute is not connected: provider data is still shown, with an invitation to connect and no paywall.
```

**8. Audience**
```
Design the Audience section:
- Contacts table with saved views, filters and bulk actions.
- Contact profile with custom fields, per-list consent history and an activity timeline (including which provider delivered each email).
- Segment builder with AND/OR groups, live count and generated SQL.
- List health with tier trends and the sunset policy.
- The suppression list, shared across all providers, showing reasons and the source provider.
```

**9. Automations**
```
Design automations. The flow canvas uses the open-source React Flow library (@xyflow/react), so design custom nodes and edges on a React Flow canvas (pan, zoom, minimap, controls, snap to grid, connection handles).

Include:
- Flow builder triggers: joined list, form submitted, tag added, API event, date field.
- Flow builder steps: delay, send email (uses the Automations route unless overridden), branch, filter, A/B split, webhook, list actions, goal/exit.
- Live stats per step.
- Draft / Live / Paused states with publish and a version diff.
- An automation list with performance.
- Starter templates: welcome series, re-engagement, onboarding.
```

**10. Migration importer**
```
Design the importer:
1. Pick a source: Brevo, Mailchimp, MailerLite, listmonk, Keila, or CSV/XLSX.
2. Map fields.
3. Map consent status so unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed. Optionally pull existing suppressions from connected sending providers.
4. Preview duplicates and invalid or risky addresses.
5. Show progress, then a summary with a downloadable report of skipped rows.
```

**11. Forms & public pages**
```
Design Forms & Pages:
- Signup form builder (embedded, popup, hosted) with double opt-in, spam protection and embed code.
- Publisher-branded public pages:
  - subscribe
  - opt-in confirmation
  - preference center (lists, frequency, pause)
  - unsubscribe, with a "fewer emails" option
  - public newsletter archive
```

**12. Settings**
```
Design Settings:
- Senders and domains, with verification status per sending provider.
- Sending providers, with a summary and a link to Sending providers & routing.
- Team and custom roles, with SSO.
- API keys, and webhooks with delivery logs.
- MCP server, with per-agent permissions and recent agent actions.
- Audit log.

Everything is available in the open-source edition.
```

**13. Landing page (optional)**
```
Design the reputecast.com landing page and a GitHub README banner.

Message: the newsletter platform that protects your inbox placement. Open source, self-hosted, and sends through your own Amazon SES, Postmark, Mailgun, SendGrid, Resend, Postal or any SMTP server, with no lock-in and several providers at once.

Include:
- An honest comparison with listmonk, Keila, Notifuse, Mautic and Brevo.
- A cost calculator comparing providers at the user's volume, with editable prices.
- A one-command docker compose install.
- Features.
- "Built by HostRepute".
- GitHub and Discord calls to action.
```
