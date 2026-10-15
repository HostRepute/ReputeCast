# ReputeCast — Design plan: features after 1.0

This plan orders and scopes the design of every feature chosen after the October 2026 competitor research. It is the index for that work. Each phase gets its own design brief: an add-on spec in this folder, written in the same format as `reputecast-design-addon-team-access.md`.

- **Phase 1 is ready to design:** `reputecast-design-addon-transactional.md`.
- **Later briefs** are written one phase ahead of the design work, so each can use what the previous phase taught. Section 5 already lists every phase's scope, screens and states, so any phase can be sketched now.

---

## 0. How to design from this plan

- Add new pages to the existing ReputeCast prototype canvas, one per phase: "P1 Transactional email" … "P8 SMS, push & agencies".
  - Reuse the Clearwater design system (`design/design-system`), the app shell and the existing components.
  - Don't invent new visual styles. Section 6 lists the few new patterns each phase needs; add them to the "9 Design system" page before that phase's screens.
- Same rules as the main spec:
  - Every screen has empty, loading, error and success states, plus the states its brief lists.
  - All copy in English.
  - Sample data extends the main spec's and the team add-on's: Acme Newsletter, Linh Nguyen, Marcus Chen, Sofia Rossi, Daniel Park and Aisha Bello.
- Board sizes:
  - Desktop boards at 1440 px.
  - Tablet (834 px) and phone (390 px) boards for each phase's primary screens.
  - Public pages at 720 px, plus phone.

**A phase's design is done when:**
1. Every screen and state in its brief exists. That includes the read-only variant for members without the permission, which names the permission and who can help.
2. The boards pass the DSAccessibility checklist (WCAG 2.2 AA).
3. Every new or changed board was screenshotted at its design width, and every visible flaw was fixed before publishing.
4. DSCoverage (screen × state matrix) and `docs/DESIGN_INVENTORY.md` list the new boards, and every conflict with a spec is written down.

---

## 1. Principles for every phase

1. **Everything is free.** No paywall, upgrade prompt, lock icon, plan badge or limit, as in the main spec.
   - Paid subscriptions, paywalls and products (phases 6 and 7) are tools for publishers to charge *their* readers through *their* own Stripe account.
   - ReputeCast takes no fee and gates nothing.
2. **Self-hosted first.** Every outside service is optional and uses the workspace's own account: Zapier, Stripe, an AI model, a verification service, an SMS provider.
   - Credentials are write-only.
   - Every screen still works when the service isn't connected, and says what is missing.
3. **Pluggable, like sending providers.** Payment, verification, AI, SMS and push providers each get the same things as the provider catalog:
   - a catalog with capability badges
   - one connection pattern
   - health status
   - a write-only credential form
4. **Sender reputation first.** Every new way to send goes through routing, the shared suppression list, guarded sending and a pre-send check suited to it. That covers the transactional API, recurring digests, SMS and push.
5. **One editor, one public theme.**
   - Every email-like editor is the GrapesJS + MJML composer in a mode, and new email blocks are custom GrapesJS blocks.
   - Every public page uses the publisher branding from Forms & pages.
6. **Same access rules as the team add-on.**
   - Every new action has a permission key.
   - The escalation guard, write-only secrets and audit entries apply as before.
   - MCP agents propose; they never send by themselves.
7. **Stay on ReputeCast's side of the HostRepute line** (section 2).

---

## 2. The HostRepute boundary

HostRepute watches the reputation of hosts (domains, IPs, mail servers) over time and alerts when it changes. ReputeCast writes and sends mail. The new features stay on ReputeCast's side.

| Area | ReputeCast (free, this repo) | HostRepute (separate paid service) |
|---|---|---|
| Blocklists | The built-in 8 DNSBLs, daily and on demand. Not expanded. | 80+ sources on a schedule, history, delisting |
| Mailbox-provider data | DMARC report upload, as today | Google Postmaster Tools, Microsoft SNDS, DMARC report collection |
| DNS and certificates | Checks on sending domains when they're added or used (SPF, DKIM, DMARC, BIMI, MAIL FROM) | Scheduled DNS checks; monitoring of certificates (TLS, STARTTLS, CAA), MTA-STS, TLS-RPT and uptime |
| Alerts | In-app and email notifications about ReputeCast's own sending | Reputation alerts by email, Slack, Telegram, Discord, PagerDuty, Pushover and webhooks; status pages; scheduled reports |
| AI | Writing, translating, alt text, accessibility fixes and segment drafts, with the workspace's own model key (phase 3) | AI Reputation Assistant, AI incident summaries |
| Inbox placement (new) | Tests with seed mailboxes the workspace owns (phase 4) | Nothing today. A managed seed network, if HostRepute ever offers one, plugs into the same screen as a second source. |
| Address verification (new) | Today's built-in checks, plus third-party verification services (phase 4) | Nothing |

Rules:
1. **No new continuous monitoring in ReputeCast.** Anything that watches a host over time and alerts belongs in HostRepute. ReputeCast keeps its built-in baseline so it stays fully usable on its own.
2. **New deliverability tools in ReputeCast are one-off checks** tied to a campaign, template, list or provider: accessibility checks, placement tests, address verification.
3. **The data boundary doesn't move.** ReputeCast never sends contacts, email content or statistics to HostRepute.
4. **Touchpoints, never gates.**
   - When HostRepute is connected, a new sending domain or IP offers "Add to HostRepute monitoring". Examples: the transactional subdomain (phase 1), a new provider's IP.
   - A reputation question in the AI assistant links to HostRepute's assistant.
   - When HostRepute isn't connected, nothing goes beyond today's invitation on Deliverability.

---

## 3. Phases at a glance

| Phase | Features | Why this order | Main users | New canvas pages | Milestones |
|---|---|---|---|---|---|
| **1 Transactional** | Transactional email: templates, API, SMTP relay, message log. Also template import from other providers, a Notifuse importer and welcome-series prompts. | The one table-stakes gap: listmonk, Keila, Notifuse and Plunk all send transactional email, and the API keys screen already promises it. | SaaS founders | P1 Transactional email | M15 |
| **2 Connect and reach** | Integrations hub (Zapier, Make, n8n, WordPress); content languages; app languages; accessibility checks | The research's three should-haves. The hub and languages are also foundations for later phases. | Everyone | P2 Integrations, languages & accessibility | M16–M19 |
| **3 Create** | Recurring and RSS digests; conditional content; polls and surveys; AI assistant | Mostly composer work, best done after phase 2 has reshaped the composer | Publishers, marketing teams | P3 Create | M20–M22 |
| **4 Know your audience** | Contact scoring; address verification services; placement tests; growth sources | Audience quality and confidence before sending | Marketing teams, SaaS founders | P4 Audience insights & placement tests | M23–M24 |
| **5 Grow** | Referral program; recommendations | Publisher growth; feeds growth sources | Publishers | P5 Grow | M25 |
| **6 Earn** | Payments (Stripe first); paid subscriptions; offers and gifts; sponsorships; revenue and attribution | Publisher income; uses conditional content and growth sources | Publishers | P6 Earn | M26–M27 |
| **7 Publish and sell** | Publication website with paywall and member sign-in; landing and sales pages; digital products; Shopify and WooCommerce | Needs payments; the largest scope | Publishers, shop owners | P7 Website, pages & commerce | M28–M30 |
| **8 More channels and agencies** | SMS; web push; WhatsApp (optional); agency client workspaces | Reuses providers, routing, consent and campaigns for new channels | Marketing teams, agencies | P8 SMS, push & agencies | M31–M32 |

The research behind phases 1 and 2 was verified. The scope of phases 3–8 is the owner's choice: the research found no verified evidence for or against those features.

---

## 4. Navigation after all phases

| Sidebar item | Contents (phase) |
|---|---|
| Overview | Adds a Transactional card (1) and an Earn card (6) |
| Campaigns | Regular, A/B and recurring/RSS email (3); SMS and push campaigns, with a channel filter (8) |
| Automations | Adds triggers and steps for polls, scoring, referrals, purchases, SMS and push |
| **Transactional** (1, new) | Templates · Messages · Setup · Settings |
| Audience | Contacts · Segments · List health · Scoring (4) · Growth (4) · Suppressions · Import |
| **Grow** (5, new) | Referrals · Recommendations |
| **Earn** (6, new) | Overview · Plans & offers · Paid members · Gifts · Sponsors · Products (7) |
| Forms & pages | Forms · Website (7) · Pages (7) · Branding & domain |
| Deliverability | Adds Placement tests (4) |
| Sending providers | Email providers, plus SMS and push providers (8) |
| Settings | No new tabs. General gains Sidebar sections (1) and Languages (2); Integrations becomes the hub (2), which also lists AI models (3) and payment providers (6) |

- **Phone tab bar.**
  - Today it shows every sidebar item as a 44 px icon. A ninth item doesn't fit at 390 px.
  - Phase 1 changes it to four items plus More. The proposed four are Overview, Campaigns, Audience and Deliverability.
  - More opens a sheet with the remaining items. Later phases add their items to the sheet.
- **Tablet icon rail:** unchanged. It fits up to 11 items.
- **Sidebar sections.** Settings › General gets switches for the sections a workspace may not use: Transactional (1), Grow (5) and Earn (6). A newsletter-only workspace hides Transactional; a SaaS workspace hides Grow and Earn. Hiding a section only takes it out of the sidebar; nothing stops working.
- **No renames.** Forms & pages keeps its name when the website arrives (7), and Sending providers keeps its name when SMS and push providers arrive (8).

---

## 5. Phase details

In each phase, "new boards" are screens to design from scratch and "changed boards" are existing screens to update.

### Phase 1 — Transactional

The full brief is `reputecast-design-addon-transactional.md`.

- **Transactional email:**
  - templates edited in the composer and published by version
  - an HTTP API (send, batch, render) with idempotency keys
  - an SMTP relay for apps that only speak SMTP
  - a message log with a timeline per message
  - send-only and test-mode keys
  - a fourth routing stream, with its own suppression rules
- **Template import:** from Postmark, SendGrid, Mailgun and Notifuse, or from MJML and HTML files. Templates are converted to Liquid, with a report of anything that needs a manual fix.
- **Notifuse importer:** contacts, lists and templates, next to the existing importers.
- **Welcome-series prompts:** after a list is created, a form goes live or an import finishes, offer the welcome-series starter. On listmonk, a welcome email is the most requested missing feature; ReputeCast already has one.
- **New boards:**
  - Transactional overview, with the first-run guide
  - template editor and publish dialog
  - message log and message detail
  - setup (API and SMTP), settings, template import
- **Changed boards:**
  - app shell and Overview
  - Routing, Setup › Streams, Senders
  - API keys, Roles, Approvals
  - Contact profile, Suppressions, the preference center
  - Webhooks, Notifications
  - Migration source, Instance settings
  - List health, Form builder, Migration done

### Phase 2 — Connect and reach

**2.1 Integrations hub**
- **Goal:** connect ReputeCast to the tools teams already use, without code.
- **Scope:**
  - Settings › Integrations becomes a catalog grouped by kind:
    - automation platforms (Zapier, Make, n8n)
    - websites (WordPress)
    - deliverability (HostRepute)
    - developer (webhooks, MCP)
    - groups that later phases fill: AI models, verification, payments, commerce
  - Each integration has a detail page:
    - what it can do: its triggers and actions
    - how to connect: a key preset with exactly the permissions it needs (shown once), plus the instance URL to paste
    - activity: recent calls and errors by that key, and the webhooks it manages
    - Disconnect, which revokes its key and webhooks
  - Zapier, Make and n8n apps:
    - Triggers: contact subscribed, unsubscribed or updated; form submitted; campaign sent; transactional message bounced.
    - Actions: create or update a contact, add to a list, add a tag, record an event (which starts automations), send a transactional email.
  - WordPress plugin:
    - connect a site
    - a signup-form block and shortcode
    - "Send new posts as a campaign draft"
    - the site's feed for recurring digests (phase 3)
- **Out of scope:**
  - Native CRM sync: Zapier, Make or n8n cover it for now.
  - Slack notifications: Zapier can forward ReputeCast's webhooks, and reputation alerts belong to HostRepute.
- **New boards:** catalog; detail page (Zapier); connect dialog; WordPress sites.
- **States:** not connected; connected with activity; key revoked or expired; webhook deliveries failing (dead-letter); site unreachable; plugin outdated.
- **Changed boards:**
  - Settings › Integrations: the HostRepute card moves into the catalog.
  - API keys: a "Managed by Zapier" badge.
  - Webhooks: endpoints that an integration manages can't be edited by hand.
- **New permission:** integrations:manage ⚑.
- **Evidence:** Notifuse shipped a Zapier app, with a send-transactional action, in August–September 2026.

**2.2 Content languages**
- **Goal:** a campaign, automation email or transactional template in several languages, each contact getting theirs.
- **Scope:**
  - Settings › General › Languages: the workspace's languages, the default and the fallback.
  - The contact's language uses the existing field. Forms fill it from the browser language; imports, the API and the preference center also set it.
  - Composer:
    - a language switcher in the header
    - a status per variant: Up to date, Outdated (the source changed after translation) or Missing (falls back to the default)
    - a side-by-side compare view
    - "Add language", which copies the source layout
    - a subject and preheader per language
    - "Translate with AI", shown disabled until phase 3
  - Send Readiness gets a "Languages" check: recipients per language, who gets the fallback, outdated variants.
  - Reports show results per language.
  - Public pages and forms get their copy per language: subscribe, confirmation, preference center, unsubscribe and archive labels.
- **New boards:** Settings › General with its Languages section; the composer with the language switcher; the compare view; the Add language dialog.
- **Changed boards:** Send Readiness, Report, Form builder, public-page settings, Contact profile.
- **States:**
  - missing translation
  - outdated translation
  - a language without public-page copy, which falls back to English
  - a right-to-left language: not supported yet, and the screen says so
- **Evidence:** Notifuse shipped template translations in March 2026 and has charged for them since September. ReputeCast stores each contact's language but can't use it yet.

**2.3 App languages**
- **Goal:** the ReputeCast app itself in other languages.
- **Scope:**
  - Account › Language & region: app language, date and number formats.
  - Instance › default language.
  - System emails in the recipient's language.
  - English and Vietnamese first; left-to-right languages only at first.
- **New boards:**
  - the two pickers
  - stress-test boards of three dense screens (Campaigns, Review & send, Team), shown in Vietnamese and in a pseudo-locale with 40% longer text, to settle truncation and wrapping rules

**2.4 Accessibility checks**
- **Goal:** catch the accessibility failures a machine can find, before sending, and fix them in one click where possible.
- **Scope:**
  - A Composer › Accessibility panel. It lists issues, links each one to its element and offers a fix for:
    - link text that says nothing: "click here", a bare URL, or no text at all
    - skipped heading levels
    - text below the minimum size
    - missing alt text, with a "decorative image" choice
  - A contrast-ratio chip in the Style panel while picking a color: AA or AAA, with a suggested color.
  - Added automatically at compile and shown as passing checks: `lang` and `dir` from the campaign's language, and `role="presentation"` on layout tables.
  - Send Readiness gets an Accessibility group. It warns by default; a workspace setting can make serious issues block.
  - The template gallery shows each template's accessibility result.
- **New boards:** the Accessibility panel; the color picker with contrast; the Send Readiness accessibility group.
- **States:** issues found; all clear; contrast failing while picking; fix applied; workspace policy set to block.
- **Evidence:** in the Email Markup Consortium's 2026 report, 99.88% of 376,348 emails had serious or critical issues, and the most common issues can all be checked by a machine. The most frequent: missing `dir`, missing `lang`, layout tables without `role="presentation"`, links without text.

### Phase 3 — Create

**3.1 Recurring and RSS digests**
- **Scope:**
  - A campaign-type picker: Regular or Recurring digest.
  - Feed sources: RSS, Atom, JSON Feed, or a connected WordPress site, with a preview of the items.
  - Schedule:
    - daily, weekly or monthly, at a time in a chosen time zone
    - or "when N new items arrive", at most once a day
    - skipped when there are no new items
  - A "Feed items" block: list, cards or grid layout; chosen fields; a maximum number of items. Feed tokens are available in the subject.
  - Mode: send each issue automatically, or create a draft for review.
  - Approval rules and Send Readiness run on every issue. A Blocked issue is held, and the owner is notified.
  - Issue history: sent, skipped or held, each with its report.
- **States:** feed unreachable or invalid; no new items (skipped); held by Send Readiness; waiting for approval; paused.

**3.2 Conditional content**
- **Scope:**
  - "Visibility" in the inspector for any section or block.
    - Show it to everyone, or only to contacts matching conditions: segment, list, tag, field, language or engagement tier.
    - Score (phase 4) and paid tier (phase 6) are added later.
  - Badges on conditional blocks in the canvas.
  - "Preview as" a contact.
  - A summary of the combinations, such as "3 conditional blocks, 4 versions".
  - Send Readiness renders every combination and flags an empty email or one over the size limit.
- **States:** a condition referring to a deleted field or segment; a combination that renders empty; previewing as a contact who sees nothing.

**3.3 Polls and surveys**
- **Scope:**
  - A Poll block: a question with two to five answers, a 0–10 NPS scale, or an emoji scale.
  - Recording answers:
    - One click opens a branded thank-you page, and the page records the answer, not the link. That way link scanners can't vote. Without scripts, the page shows a confirm button.
    - People can change their answer and add a comment.
    - The answer is saved to a contact field or tag. "Answered a poll" is an automation trigger.
  - A poll section in the report: results by answer, segment and time, plus the number of automated clicks ignored.
  - A hosted multi-question survey page, linked from an email.
  - CSV export.
- **States:** poll closed; answer changed; automated clicks filtered; an answer from someone who isn't a contact.

**3.4 AI assistant, with the workspace's own model**
- **Scope:**
  - Settings › Integrations › AI assistant:
    - a provider catalog: Anthropic, OpenAI, Google, Azure OpenAI, and OpenAI-compatible servers such as Ollama
    - the model, a write-only key and a monthly usage cap
    - a plain statement of what is sent: the content being edited, never contact data
  - Composer:
    - subject-line ideas, checked by the existing heuristics
    - rewrite, shorten or change the tone
    - translate a language variant (2.2)
    - alt text from an image
    - fix an accessibility issue
  - Segments: "Describe the segment" drafts conditions for review.
  - Reports: a plain-language summary of a campaign's results.
  - Every suggestion appears as a diff to accept or reject.
  - New permission: ai:use.
- **Out of scope:**
  - reputation diagnosis (links to HostRepute's assistant when connected)
  - sending
  - contact data in prompts
- **States:** not configured (an invitation, no paywall); provider error or rate limit; refusal; usage cap reached.

### Phase 4 — Know your audience

**4.1 Contact scoring**
- **Scope:**
  - Audience › Scoring holds the rules:
    - points for human opens, clicks, form submissions, API events and field conditions
    - negative points for soft bounces and list unsubscribes
    - decay after weeks without activity
    - caps, and bands (Cold, Warm, Hot)
    - a preview of how contacts spread across the bands
  - The contact profile shows the score with its breakdown.
  - A segment condition, and an automation trigger "Score crosses…".
  - Copy explains how the score differs from engagement tiers: tiers protect deliverability, the score measures interest.
- **States:** no rules yet; recalculating; an edited rule, with a preview of its impact.

**4.2 Address verification services**
- **Scope:**
  - Verification providers in the Integrations hub, for example ZeroBounce, NeverBounce, Kickbox, Bouncer and Emailable, with their credit balance and price per check.
  - Where they're used:
    - the import preview: optional, with the cost shown before it starts
    - signup forms: a real-time check that lets the signup through when the service fails
    - bulk verification of a list or segment
    - re-verifying inactive contacts before a re-engagement campaign
  - Results:
    - valid, invalid, risky (catch-all, role, disposable) or unknown
    - invalid addresses are suppressed for all mail; risky ones are tagged
    - the contact profile shows the result, its date and the service
    - Send Readiness's audience check uses them
  - Before each use, a clear notice that the addresses go to the chosen service. Each use is audited.
- **States:** out of credits; partial results; service down; job running.

**4.3 Placement tests**
- **Scope:**
  - A Placement tests page under Deliverability.
  - Seed mailboxes the workspace owns:
    - Gmail, Google Workspace, Outlook and Microsoft 365, through OAuth or an IMAP app password
    - Yahoo, iCloud, AOL, GMX and Web.de, through an IMAP app password
  - "Run test": pick a campaign or template, a route or provider, and a sender. One copy goes to each seed.
  - A results matrix of mailbox × folder:
    - folders: Inbox, Promotions or another tab, Spam, Not arrived
    - for each copy, the time to arrive and the authentication results read from the headers
  - Comparing providers: the same email sent through SES and through Postal, side by side.
  - History and trends.
  - Entry points:
    - Review & send (optional; a test never blocks a send by itself)
    - the composer's test menu
    - the warm-up planner
  - Test messages are found by their token and can be deleted from the seed mailbox afterwards.
- **Out of scope:** a managed seed network (section 2).
- **States:** a mailbox's sign-in expired; waiting (up to 15 minutes); partial; complete; everything in spam, with guidance.

**4.4 Growth sources**
- **Scope:**
  - Audience › Growth: joins and departures by source. Sources: each form, import, API key and integration; later, referrals and recommendations.
  - UTM parameters and the referring page, captured by forms and hosted pages.
  - Cohort retention: still subscribed after 30 and 90 days.
  - Quality by source: bounce, complaint and human-open rates.
  - A contacts filter by source and UTM.
- **States:** no UTM data yet; too little data to compare.

### Phase 5 — Grow

**5.1 Referral program**
- **Scope:**
  - Grow › Referrals: program settings, and rewards at milestones.
    - Reward types: a download link, a code from an uploaded list, manual fulfillment, a Stripe coupon (from phase 6), or a webhook.
    - Counting rules: confirmed subscribers only, a hold period, and the same IP or a disposable domain excluded.
  - What subscribers see:
    - a Referral block in the composer, with the subscriber's personal link and progress
    - referral tokens for Liquid
    - a public referral hub page per subscriber
    - an optional leaderboard
  - What the publisher sees:
    - a review queue for suspicious referrals
    - analytics: referrals, conversion, top referrers, rewards sent
    - the automation trigger "Reached a milestone"
- **States:** program off; a reward's codes running out; a suspicious referral held; the milestone email.

**5.2 Recommendations**
- **Scope:**
  - Grow › Recommendations: the newsletters you recommend, each with a name, logo, description and subscribe link. They show on the confirmation page and in a welcome-email block.
  - Partner links:
    - Two ReputeCast instances exchange a signed cross-subscribe link.
    - The reader confirms through the partner's own double opt-in.
    - The new subscriber is attributed to that growth source.
  - Click and signup numbers per recommendation.
- **Out of scope:** a central marketplace. It would be a hosted service, not part of the open-source app.
- **States:** none yet; partner link invalid; recommendation paused.

### Phase 6 — Earn

**6.1 Payments**
- Earn › Settings › Payments: a provider catalog, also listed in the Integrations hub. Stripe comes first; Paddle and Lemon Squeezy (merchants of record) later.
- Stripe connects with a restricted key, and its webhook is registered automatically.
- The page shows a test-mode or live badge, the currency, a note on Stripe Tax, and webhook freshness.

**6.2 Paid subscriptions**
- Earn › Plans & offers: tiers with monthly and yearly prices, benefits, a trial and visibility.
- A hosted upgrade page whose tier cards lead to Stripe Checkout, and a "Manage subscription" link in the preference center.
- Paid status on every contact: tier, status, renewal date and lifetime value. Statuses: trialing, active, past due, canceled, complimentary.
- Earn › Paid members, with complimentary access.
- Importing paid members from Substack, Ghost, beehiiv or Kit exports, matched with Stripe customers.
- Paid-only campaigns and paid-only blocks (3.2).
- Payment-failed and renewal emails as transactional templates (phase 1).

**6.3 Offers and gifts**
- Offers: a percent or fixed discount for a duration, by code or link, with a limit and an expiry.
- A retention offer on a hosted cancel page, shown before Stripe's portal.
- Gift subscriptions:
  - a public "Give a gift" page, with a delivery date and a message
  - a one-time Stripe payment
  - redemption without a card
  - reminders before the gift ends

**6.4 Sponsorships**
- Earn › Sponsors: a sponsor library, and bookings by issue and slot.
- A Sponsor block filled from the booking and labelled "Sponsored".
- A sponsor report (human opens, clicks, click rate) shared as a read-only link.
- Payment status recorded by hand; no invoicing.

**6.5 Revenue and attribution**
- Earn › Overview:
  - MRR, ARR and paid members
  - new and churned members, churn rate
  - free-to-paid conversion by source and by campaign
  - gifts and sponsor income
- A Revenue section in the campaign report.
- Revenue data comes from Stripe's events and from a conversion API for SaaS founders. A conversion carries a value and is matched by click token or email.

**States across Earn:** Stripe not connected; webhook missing; test mode; sync error; refund; past due.

### Phase 7 — Publish and sell

**7.1 Publication website** (grows out of the public archive)
- **Pages and posts:**
  - a home page: latest posts, featured posts, about
  - posts from campaigns: "Publish to web", with slug, excerpt, cover and SEO fields
  - web-only posts
  - tags or sections, pages (About and custom pages) and navigation
- **Members:** sign-in by magic link, and an account page (plan, gifts, newsletters).
- **Visibility:**
  - each post is public, for members, or for paid tiers
  - a "Free preview ends here" marker
  - a metered paywall: N free posts per period
- **States:**
  - the same post as seen by an anonymous visitor, a free member and a paid member
  - meter used up
  - sign-in link sent
  - post unpublished
  - domain pending

**7.2 Landing and sales pages**
- Block-based pages on the publisher theme: lead magnet, launch, webinar registration.
- Product blocks, and a storefront page listing products.
- Views, signups and sales per page.

**7.3 Digital products**
- A product is a file plus a price: one-time, or pay what you want.
- Payment goes through Stripe Checkout. The delivery email is a transactional template with an expiring download link.
- Orders, with refunds through a link to Stripe.
- The automation trigger "Bought a product".

**7.4 Shopify and WooCommerce**
- Both are added to the Integrations hub.
- They sync orders, products, and the customers who accepted marketing.
- Triggers: order placed, first order, abandoned checkout.
- Product blocks fed by the catalog.
- Revenue attribution.
- An abandoned-cart starter automation.
- **States:** store disconnected; a customer who didn't accept marketing (not added to lists); webhook missing.

### Phase 8 — More channels and agencies

**8.1 SMS**
- **Providers:**
  - Twilio, Vonage, Plivo, Telnyx and AWS End User Messaging, under Sending providers, with capability badges.
  - Phone numbers and sender IDs, with their registration status: US 10DLC, toll-free verification, alphanumeric IDs by country.
  - Routing by country.
- **Contacts and consent:**
  - Contacts get a phone number (E.164), with SMS consent kept separate from email consent. Forms collect both.
  - STOP and HELP are handled automatically, with an SMS suppression list.
  - Quiet hours follow the recipient's time zone.
- **Sending:**
  - SMS campaigns, and an SMS step in automations.
  - The SMS composer:
    - counts segments (GSM-7 or UCS-2)
    - shortens links on the workspace's own domain
    - personalizes
    - shows a phone preview and the cost per country
  - Pre-send check: consent, registration, quiet hours and the opt-out text.
- **Report:** delivered, failed, clicks, opt-outs and cost.

**8.2 Web push**
- VAPID keys, and a service worker on the publication site (7.1) or a snippet for the publisher's own site.
- A soft opt-in prompt designer.
- Push campaigns and an automation step, with previews for Chrome, Safari and Android, and a report.

**8.3 WhatsApp** (optional, last)
- Meta Cloud API, approved message templates, the 24-hour window rules.

**8.4 Agency accounts**
- Client workspaces grouped under an agency.
- An agency overview: every client's next sends, health, pending approvals and incidents.
- Client roles: Client approver and Client viewer.
- Workspace templates that copy sender defaults, automations and saved sections, and a template library shared across clients.
- Usage per client (sends, provider cost) for rebilling.
- **States:** client invited; a client approving a send; search across clients; a workspace template applied.

---

## 6. Design-system additions, by phase

Add these to the "9 Design system" page before the phase's screens.

| Phase | New patterns |
|---|---|
| 1 | Code samples with language tabs and copy; request and response viewer; variables table with a JSON test-data editor; message status badges (Queued, Sent, Delivered, Deferred, Bounced, Complained, Rejected, Failed, Test); masked sensitive value; phone "More" sheet |
| 2 | Integration catalog card (logo, kind, status, managed-by); language chip with translation status; side-by-side compare; contrast-ratio chip; issue list that locates an element on the canvas |
| 3 | Visibility badge on blocks; email-safe poll buttons; AI suggestion diff card |
| 4 | Score meter and bands; placement matrix; growth-source chart |
| 5–6 | Money (currency, MRR deltas); public tier cards; paywall card; gift card |
| 7 | Public site templates: home, post, paywalled post, page, member account |
| 8 | SMS bubble preview with segment counter; push previews; channel icons |

---

## 7. Decisions made by default

These defaults let design start. Change any of them before its phase begins.

1. **Phase 1:** transactional mail gets its own stream, the fourth after Broadcasts, Automations and System. It has automatic failover, and priority over Broadcasts on a shared provider.
2. **Phase 1:** suppressions for transactional mail.
   - Ignored, because they are marketing-only: unsubscribes and role addresses.
   - Honoured: hard bounces, complaints, spam traps, invalid, manual and imported.
   - Optional notifications also honour the recipient's opt-out for their category.
3. **Phase 1:** the SMTP relay is designed now. Engineering may ship it after the HTTP API.
4. **Phase 4:** placement tests use only the workspace's own seed mailboxes. A managed seed network would be a HostRepute product.
5. **Phase 6:** payments go through a provider abstraction, Stripe first.
6. **Phase 2:** app languages start with English and Vietnamese; right-to-left comes later.
7. **Phase 3:** AI is off by default, uses the workspace's own key and never receives contact data.
8. **Phase 5:** recommendations are partner links between instances, with no central marketplace.
9. **Phase 8:** WhatsApp comes last and is optional.

---

## 8. Engineering notes

- Each phase becomes milestones in `docs/ROADMAP.md` when its design is approved: M15 onward, as in section 3.
- Dependencies outside the current stack need a reason in their PR. Expected ones:
  - an SMTP server library (phase 1)
  - model SDKs, or plain HTTP (phase 3)
  - the Stripe Go SDK (phase 6)
  - SMS provider SDKs, or plain HTTP (phase 8)
  - a Web Push library (phase 8)
- Each new provider kind (payment, verification, AI, SMS, push) gets a shared contract-test suite, like sending adapters.
