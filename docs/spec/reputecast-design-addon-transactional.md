# ReputeCast — Add-on spec: transactional email

Phase 1 of `reputecast-design-expansion-plan.md`.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec and the team add-on already exists.**
  - Add the screens in section 5 as a new canvas page, "P1 Transactional email".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's and the team add-on's.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.5 and 5.6.
- **Build order:**
  1. 5.1 Transactional overview
  2. 5.3 Template editor
  3. 5.4 Publish dialog
  4. 5.5 Message log
  5. 5.6 Message detail
  6. 5.7 Setup: API and SMTP
  7. 5.2 First-send guide
  8. The rest, in order
  9. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.** No limits on templates, messages, keys or SMTP credentials. Rate limits exist only where the workspace sets them.
2. **Separate reputations.** Transactional mail has its own stream, route, sender and suppression rules. A marketing problem never blocks a password reset.
3. **Only published versions are sent.**
   - The API always renders a template's published version. Drafts never reach anyone.
   - Publishing is deliberate: a diff, a test render and the checks.
4. **Never lost, never sent twice.**
   - Idempotency keys, retries and automatic failover.
   - When ReputeCast can't deliver, it says exactly why, both in the API answer and in the log.
5. **Secrets stay out of logs.**
   - Variables marked sensitive are never stored.
   - Message content is kept for a set time, and viewing it needs a permission.
6. **Show the request.** Every message shows the request that produced it, and every template shows the exact call that sends it. A developer can debug without reading code, and a marketer can follow along without writing any.
7. **Same access rules as the team add-on.**
   - Permission keys and send-only keys.
   - Nobody grants a permission they don't hold.
   - Audit entries for templates, keys and settings.
   - MCP agents never send.

---

## 2. Terms

- **Transactional email:** one email to one person, triggered by something they did or need: a password reset, a sign-in link, a receipt, a usage alert. Not marketing.
- **Transactional stream:** the fourth stream, next to Broadcasts, Automations and System.
  - System stays for ReputeCast's own mail: invitations, opt-in confirmations.
  - Transactional carries mail from the workspace's own product.
- **Transactional template:** a template with an ID (for example `password-reset`), declared variables, a category and a published version.
- **Category:** groups templates and decides whether recipients can turn them off.
  - **Essential categories** (Account, Billing): always sent, with no unsubscribe link.
  - **Optional categories** (Reminders, Product updates): listed in the preference center, where recipients can turn each one off. Their emails carry an unsubscribe link and a List-Unsubscribe header for that category.
- **Message:** one transactional email to one recipient, with its own ID, status and timeline.
- **Send-only key:** an API key whose only permission is transactional:send. It can be limited to some templates.
- **Test-mode key:** a key whose messages are rendered and logged but delivered only to the workspace's members and to allowed test addresses. Other recipients are marked "Not delivered (test mode)".
- **SMTP relay:** an SMTP server inside ReputeCast for apps that can't call an HTTP API. Each SMTP credential is a send-only key.
- **Idempotency key:** sent with a request so that retrying the same request never sends twice. Kept for 24 hours.
- **Sensitive variable:** a template variable whose value is never stored. Logs and stored content show it as ••••••.

---

## 3. Requirements

### 3.1 Templates
- **Editor:** the campaign composer in transactional mode.
  - Modes: Visual (GrapesJS + MJML), MJML code and HTML, plus the plain-text version.
  - Saved sections and campaign templates can be reused, so product mail matches the newsletter's brand.
  - The footer block offers a "Transactional footer": company name and address, no unsubscribe link.
- **Fields:**
  - name
  - ID: lowercase letters, digits and hyphens; fixed after the first publish
  - description
  - category
  - sender (a verified sender of the Transactional stream) and reply-to
  - subject and preheader, with Liquid
  - open and click tracking, both off by default
- **Variables:**
  - Each has a name, a type (text, number, yes/no, date, URL, list, object), required or optional, a sample value, and "Sensitive".
  - The editor warns about variables the template uses but doesn't declare, and declared ones it never uses.
  - A sensitive variable in the subject gets a warning, because subjects are stored.
- **Test data:** named JSON samples, such as "New user" or "No first name", used by the preview, the checks and test sends.
- **Starters:** password reset, sign-in link, email verification, welcome, receipt or invoice, trial ending, team invitation, usage alert. Each comes with its variables declared.
- **Versions:**
  - The draft saves itself, like a campaign. Publishing makes the draft the live version.
  - The publish dialog shows the diff since the live version, the checks and a test render.
  - An older version can be published again ("Roll back to v2").
  - The API can't send a template that has never been published.
- **Checks:** the composer's content checks, adapted.
  - Same as for campaigns: size against Gmail's clipping, alt text, broken or blocklisted links, plain-text version.
  - Unsubscribe link and footer address: required for Optional categories, not for Essential ones.
  - Transactional-only:
    - Liquid errors
    - missing or unknown variables
    - a test-data set that fails to render
    - a sensitive variable in the subject
    - tracking turned on in an Essential category (a privacy note, not a block)
    - breaking changes against recent API calls (5.4)
- **Languages** come in phase 2, as language variants per template. Leave room for a language switcher in the editor's header.

### 3.2 Sending API
- **Send one message:** `POST /api/v1/workspaces/{workspace}/transactional/messages`.
  - Required: `template` (the ID), `to` (an email and an optional name), `data` (the variables).
  - Optional:
    - `cc` and `bcc`
    - `subject` and `preheader` overrides
    - `from`: any verified Transactional sender
    - `reply_to`
    - `tags`
    - `metadata`: key–value pairs, returned in webhooks
    - `attachments`: filename, content type and base64 content, with an optional `content_id` for inline images
    - `send_at`: up to 72 hours ahead
  - Header: `Idempotency-Key`.
  - Answer: `202 Accepted`, with the message ID and status `queued`.
- **Send many:** `POST …/transactional/messages/batch` sends up to 100 messages, each with its own recipient and data. Each item gets its own answer.
- **Render without sending:** `POST …/transactional/templates/{id}/render` returns the subject, HTML and text.
- **Read the log:** `GET …/transactional/messages` and `GET …/transactional/messages/{id}`.
- **Errors** follow RFC 9457, each with a fix in `detail`:
  - `template_not_found`
  - `template_not_published`
  - `missing_variables`, listing them
  - `invalid_recipient`
  - `recipient_suppressed`, with the reason and date
  - `sender_not_verified`
  - `attachments_too_large`, naming the route's provider limit
  - `rate_limited`, with Retry-After
  - `no_healthy_route`
- **Recipients don't have to be contacts.**
  - A Transactional setting decides whether a recipient who matches a contact is linked to that contact. It's on by default, so the message shows in the contact's timeline.
  - A recipient who isn't a contact is never added as one.
- **Size:** a message with its attachments can be up to the smallest limit among the route's providers. That is 10 MB on Postmark.

### 3.3 SMTP relay
- **Off until an instance admin turns it on**, in Instance admin › SMTP relay. Settings there:
  - hostname
  - ports: 587 with STARTTLS, 465 with TLS
  - certificate source
  - a status check: DNS, port reachable from the internet, certificate valid
- **Credentials:** each SMTP credential is a send-only key. The username is the key's ID; the password is its secret, shown once.
- **Rendering:** mail arriving by SMTP was already rendered by the app. It's logged as "Raw (SMTP)" instead of with a template, and goes through the same route, suppression rules and log.
- **Optional headers:** `X-ReputeCast-Tags` and `X-ReputeCast-Category`. The category header names an Optional category, which then honours opt-outs.
- **Senders:** the From address must be a verified Transactional sender. Anything else is refused with an SMTP error that says why.

### 3.4 Delivery and routing
- **Route:** the Transactional stream has its own card on Sending providers › Routing.
  - A primary, a fallback, and conditions by category or recipient mailbox provider.
  - Failover is automatic, as for System.
  - Bulk-marketing terms don't apply to this stream.
- **Priority:** when a provider also carries a Broadcast, transactional messages go first within that provider's per-second rate. It's on by default and can be turned off on the route.
- **Suppressions:**
  - Honoured: hard bounce, complaint, spam trap, invalid, manual, imported from a provider.
  - Ignored, because they are marketing-only: unsubscribe, role address.
  - Optional categories also honour the recipient's opt-out for that category.
- **Statuses:**
  - Queued, then Sent (accepted by the provider), then Delivered or Deferred.
  - Bounced (hard or soft) or Complained, from provider events.
  - Rejected: stopped before sending, with the reason: suppressed, opted out, or invalid.
  - Failed: every provider in the route failed after the retries.
  - Test: sent with a test-mode key. Either delivered to an allowed address, or not delivered.
- **Retries:**
  - Temporary errors from a provider are retried with back-off for up to 10 minutes, then the message goes to the fallback.
  - When the health job marks the provider down, new messages go to the fallback at once.
- **Tracking:** off by default. When it's turned on for a template, it uses the workspace's link tracking domain.
- **Delivery alerts:** a notification when, within the last hour, any of these passes its limit:
  - a template's bounce rate (default limit 2%)
  - failed messages (default limit 10)
  - median delivery time (default limit 30 seconds)

  There is no auto-pause, because password resets must keep flowing. Alerts reach the notification center and email, following each member's preferences.
- **Webhooks:** a "Transactional messages" event group.
  - Events: `message.sent`, `message.delivered`, `message.deferred`, `message.bounced`, `message.complained`, `message.rejected`, `message.failed`, `message.opened`, `message.clicked`.
  - Each event carries the message's tags and metadata.
- **Retention:**
  - Message metadata and events: 90 days, like provider events.
  - Rendered content: 30 days by default, adjustable from 0 to 90 days in Transactional › Settings.
  - Sensitive variables: never stored.

### 3.5 Reports
- **Per template:**
  - sent, delivered, bounced and complained, as counts and rates
  - median time to delivery
  - opens and clicks, when tracked
  - a chart of the last 7 or 30 days
- **Per sending provider:** messages, delivery rate and median delivery time, so a failover is visible.
- **Optional categories:** opt-outs per category.
- **Test messages** are left out of every number, unless "Include test messages" is turned on.

### 3.6 Permissions
New permission group, "Transactional":
- view templates, messages and reports (transactional:view)
- edit templates (transactional:edit)
- publish templates (transactional:publish) ⚑
- send through the API or SMTP (transactional:send) ⚑
- view the content of sent messages (transactional:content) ⚑

Who gets what:
- **Built-in roles:**
  - Owner and Admin: all five.
  - Editor: view, edit and publish.
  - Viewer: view only.
- **Content Writer**, the sample's custom role: view and edit.
- **Send-only keys:** transactional:send alone.
- **MCP agents:** can list templates, render previews and read the log without message content. They never send.

**Approvals:** Settings › Approvals gets an optional rule, "Publishing a transactional template needs approval". It's off by default.

### 3.7 Audit
- **Audited:**
  - a template created, published, rolled back, archived or deleted
  - Transactional settings changed
  - keys and SMTP credentials created, rotated or revoked
  - the SMTP relay turned on or off
  - a message resent by a person
  - message content viewed, because it can hold personal data
  - the message log exported
- **Not in the audit log:** each message sent through the API or SMTP. The message log is the record of sends, and it keeps them for 90 days. See section 8.

### 3.8 Importing templates
- **Sources:** Postmark, SendGrid (dynamic templates), Mailgun and Notifuse, each through an API key that is sealed and never returned; or MJML and HTML files.
- **Conversion:**
  - Their syntax (Mustachio, Handlebars, Liquid) becomes ReputeCast Liquid.
  - Variables are declared from what each template uses.
- **Report per template:** converted, converted with notes, or needs a manual fix, naming the line and the construct. Example: "Handlebars helper `formatCurrency` has no Liquid equivalent."
- **Drafts only:** imported templates arrive as drafts. Nothing is published until someone publishes it.

### 3.9 HostRepute
- When HostRepute is connected and a new Transactional sending domain is verified, offer "Add notify.acme.io to HostRepute monitoring", using the existing "Add to monitoring" flow.
- Nothing about transactional messages, recipients or content is ever sent to HostRepute.

### 3.10 Out of scope for this phase
- Language variants of templates (phase 2).
- Inbound email and reply parsing.
- SMS and push notifications (phase 8).
- Scheduling more than 72 hours ahead.

---

## 4. Sample data

### Product
- **Acme Cloud** is Acme's SaaS product, at app.acme.io.
- It sends its product mail through the "Acme Newsletter" workspace. Instance: reputecast.acme.io.

### Sender and domain
- **From:** Acme Cloud <no-reply@notify.acme.io>.
- **Reply-to:** support@acme.io.
- **Postmark:** notify.acme.io is verified, with DKIM selector `pm-2026` and return path `pm-bounces.notify.acme.io`.
- **Amazon SES:** notify.acme.io is verified, with Easy DKIM and MAIL FROM `bounce.notify.acme.io`.

### Route
- **Transactional:** Postmark (primary), then Amazon SES us-east-1 (automatic failover).
- **Priority on shared providers:** on.
- SES also carries Broadcasts and Automations. When it's the fallback, transactional mail goes first within its 14 per second.

### Categories
| Category | Kind | Templates | Opted out |
|---|---|---|---|
| Account | Essential | password-reset, magic-link, welcome | — |
| Billing | Essential | invoice-paid | — |
| Reminders | Optional | trial-ending | 2 |
| Product updates | Optional | usage-summary | 31 |

### Templates (last 7 days)
| Template | ID | Category | Version | Sent | Delivered | Median delivery |
|---|---|---|---|---|---|---|
| Password reset | password-reset | Account | v3 live | 3,212 | 99.9% | 1.4 s |
| Magic sign-in link | magic-link | Account | v2 live | 6,881 | 99.8% | 1.2 s |
| Welcome to Acme Cloud | welcome | Account | v5 live | 1,904 | 99.6% | 1.9 s |
| Invoice paid | invoice-paid | Billing | v4 live, with a PDF attached | 412 | 100% | 2.3 s |
| Trial ends in 3 days | trial-ending | Reminders | v1 live, v2 draft | 71 | 98.6% | 2.0 s |
| Weekly usage summary | usage-summary | Product updates | v6 live | 1,240 | 99.4% | 1.6 s |

Totals for the 7 days:
- 13,720 sent and 13,687 delivered (99.8%).
- 27 bounced (0.20%) and 6 still deferred.
- 1 complaint, on usage-summary.
- 12 rejected before sending: 10 suppressed and 2 opted out.
- Median delivery 1.3 s.

### password-reset v3
- **Subject:** `Reset your Acme Cloud password`
- **Preheader:** `This link works for {{ expires_in }}.`
- **Variables:**
  - `first_name`: text, optional, sample "Sam", default "there"
  - `reset_url`: URL, required, sensitive
  - `expires_in`: text, required, sample "30 minutes"
- **Test data:** "New user" and "No first name".

### trial-ending v2, draft
- Adds the required variable `trial_end_date` and removes `days_left`.
- In the last 7 days, all 71 requests sent `days_left` and none sent `trial_end_date`.

### Keys and SMTP credentials
| Name | Kind | Scope | Created | Last used |
|---|---|---|---|---|
| Acme Cloud · production | Send-only, live | All templates | Marcus Chen, 2 Sep 2026 | 2 min ago from 198.51.100.24 |
| Acme Cloud · staging | Send-only, test mode | All templates. Allowed test addresses: workspace members and qa+*@acme.io | Marcus Chen, 2 Sep 2026 | 1 hour ago |
| Billing service (SMTP) | SMTP credential | Category Billing | Linh Nguyen, 15 Sep 2026 | 38 min ago from 198.51.100.40 |

### SMTP relay
- **Status:** on. Linh Nguyen turned it on on 15 Sep 2026.
- **Host:** reputecast.acme.io. Ports 587 (STARTTLS) and 465 (TLS).
- **Certificate:** Let's Encrypt, renews 2 Dec 2026.

### Latest messages (15 Oct 2026)
| Time | Recipient | Template | Provider | Status |
|---|---|---|---|---|
| 10:42:18 | sam.rivera@outlook.com | Password reset | Postmark | Delivered · 1.3 s |
| 10:41:55 | jordan@example.org | Magic sign-in link | — | Rejected · suppressed: hard bounce on 2 Oct 2026, "550 5.1.1 mailbox does not exist" (Postmark) |
| 10:41:20 | billing@northwind.example | Invoice paid | Postmark | Delivered · 2.4 s · role address (its marketing-only suppression doesn't apply) |
| 10:40:07 | priya.shah@example.net | Welcome to Acme Cloud | Postmark | Delivered after 3 min · deferred once (greylisting) |
| 10:39:31 | ops@contoso.example | Trial ends in 3 days | Postmark | Bounced · soft: "452 4.2.2 mailbox full" |
| 10:38:02 | kim.tran@example.com | Weekly usage summary | — | Rejected · opted out of Product updates on 28 Sep 2026 |
| 10:37:44 | qa+signup@acme.io | Welcome to Acme Cloud | Postmark | Test · delivered to an allowed address |
| 10:37:40 | lena@example.com | Welcome to Acme Cloud | — | Test · not delivered (not an allowed test address) |
| 10:36:12 | finance@northwind.example | Raw (SMTP): "Your receipt #4821" | Postmark | Delivered · 1.8 s |

### Message detail: password reset to Sam Rivera
- **Request:**
  - `POST /api/v1/workspaces/acme-newsletter/transactional/messages`
  - key "Acme Cloud · production", from 198.51.100.24
  - `Idempotency-Key: pwreset-7f3a91`
- **Template:** password-reset v3.
- **Data:** `first_name: "Sam"`, `reset_url: ••••••` (sensitive, not stored), `expires_in: "30 minutes"`.
- **Tags and metadata:** tag `auth`; metadata `user_id: u_48213`.
- **Contact:** linked to Sam Rivera, a Weekly Digest subscriber.
- **Timeline:**
  1. 10:42:18.120 Accepted (202)
  2. 10:42:18.164 Rendered: v3, 18 KB of HTML plus text
  3. 10:42:18.170 Routed: Transactional → Postmark (primary)
  4. 10:42:18.402 Handed to Postmark, with Postmark's message ID
  5. 10:42:19.430 Delivered: accepted by outlook-com.olc.protection.outlook.com. "Delivered in 1.3 s."

### Incident
- **When:** 9 Oct 2026, 14:02–14:06. Postmark answered with 503 errors.
- **What happened:** 37 messages failed over to Amazon SES automatically, and none were lost. Median delivery during the incident was 3.1 s.

### Delivery alert (history)
- **When:** 3 Oct 2026, 09:15.
- **What:** magic-link's bounce rate reached 2.4% in one hour (limit 2%).
- **Cause:** most bounces went to gmial.com and yaho.com, typos made when signing up for Acme Cloud.

### Template import (from Postmark)
- 6 templates in all:
  - 4 converted.
  - 1 converted with a note: `{{#each}}` became `{% for %}`.
  - 1 needs a manual fix: the Mustachio partial `{{> footer}}` has no equivalent; use a saved section instead.

### People
- **Marcus Chen** (Admin) set up the route and the keys.
- **Sofia Rossi** (Editor) edits and publishes templates.
- **Daniel Park** (Content Writer) edits drafts. Publishing needs transactional:publish, so he asks Sofia Rossi, Marcus Chen or Linh Nguyen.
- **Aisha Bello** (Viewer) sees templates, the log and reports, but not message content.

### Welcome-series prompt
- The Product Updates list has no welcome email yet.
- The Weekly Digest list already has one: the Welcome series automation.

---

## 5. New screens

The new sidebar item "Transactional" has four tabs: Templates · Messages · Setup · Settings.

### 5.1 Transactional overview (Templates tab)
- **Header:**
  - The numbers for the period (24 hours, 7 days or 30 days): sent, delivered, bounced, complaints, median delivery and rejected.
  - The route: "Postmark → Amazon SES, automatic failover".
- **Templates table:**
  - Columns:
    - name and ID
    - category, with its kind
    - live version, with a "Draft changes" badge when the draft differs
    - sent, delivery rate, bounce rate and median delivery
    - last sent, and a sparkline
  - Search, and filters by category, kind and status.
  - "New template" and "Import templates".
- **Side card, "Send from your product":** the endpoint and a short snippet, linking to Setup.

States to show:
- the sample data
- first run, with no template and no key, which leads into 5.2
- delivery alert active: "Magic sign-in link: bounce rate 2.4% in the last hour (limit 2%)"
- no Transactional route yet: "Choose which provider delivers transactional mail", leading to Routing
- provider outage with failover active: "Postmark is down. Transactional mail is going through Amazon SES."
- read-only (Aisha Bello)
- loading, error, and search with no results

### 5.2 First-send guide
A five-step guide. Its progress is kept, so a reload resumes the same step, as in the connect wizard.
1. **Sender:** pick or add a sender on a Transactional domain.
   - Suggest a separate subdomain, such as notify.acme.io, with one line on why: it keeps product mail's reputation apart from newsletters.
   - When HostRepute is connected, offer "Add to monitoring" here.
2. **Route:** pick the provider and the fallback. Defaults to the System route.
3. **Template:** start from a starter, or import one.
4. **Key:** create a send-only key, shown once. Show the first call in cURL, Node.js, Python, Go and PHP, filled with the template's ID and sample data.
5. **Watch it arrive:** a live timeline waits for the first call and shows each step: accepted, rendered, handed to Postmark, delivered. It reuses the test-send pattern of the connect wizard.

The guide ends with a summary, "You're sending transactional mail", and links to Messages and the API reference.

States to show:
- waiting for the first call, with "Send it from here instead"
- the call rejected: `template_not_published`. "password-reset has a draft but no published version. Publish it first."
- delivered

### 5.3 Template editor
The composer in transactional mode.
- **Header:**
  - breadcrumb Transactional › Templates
  - name, ID and category
  - live version and draft status
  - Send test, and Publish
- **Left panel:** blocks, layers and templates, as in the composer.
- **Right panel tabs:**
  - **Style and Settings:** as in the composer.
  - **Variables:** a table with name, type, required, sample and sensitive, plus the warnings for undeclared and unused variables.
  - **Test data:** named JSON samples in a code editor, and "Preview with" to pick one.
  - **Checks:** the template's checks (3.1).
  - **API:** the exact request for this template in five languages, with the declared variables filled in.
- **Subject and preheader:** Liquid tokens, with the inbox-line preview.
- **Canvas:** renders with the selected test data and highlights variables. A missing required variable shows as a red token.

States to show:
- password-reset v3 with the Variables tab open (`reset_url` is sensitive)
- Liquid error: line and message, for example "Unclosed {% if %} on line 14"
- undeclared variable: "company_name is used but not declared"
- a test-data set missing a required variable: the render fails and says which variable
- "Draft changes since v3"
- Daniel Park editing: Publish is disabled, and says why and who can help
- Aisha Bello: read-only
- the composer's autosave, saved and conflict states

### 5.4 Publish dialog and versions
- **Diff since the live version:**
  - subject, preheader and content, as a text diff
  - variables added or removed
- **Breaking-change check:** the new version is compared with the requests of the last 7 days. Example for trial-ending v2:
  - "trial_end_date is now required, and none of the 71 requests of the last 7 days sent it. Publishing as it is would make those calls fail with missing_variables."
  - Fixes: "Make trial_end_date optional with a default", or "Update your app first, then publish".
- **Checks summary**, and a test render with each test-data set.
- **"Publish v2"**, or "Request approval" when the workspace's rule requires it.
- **Versions:** a list of versions with author and date, compare, and "Publish again" (roll back).

States to show:
- ready to publish
- breaking change found
- approval required
- publishing
- published: a toast with "Undo", which rolls back

### 5.5 Message log (Messages tab)
- **Search:** recipient, message ID, idempotency key, tag or metadata value.
- **Filters:** status, template, provider, key, and time range. Ranges: last hour, 24 hours, 7 days, 30 days, or custom dates within the last 90 days.
- **Table:**
  - time and recipient
  - template, or "Raw (SMTP)"
  - subject and provider
  - status badge with its detail ("Delivered · 1.3 s")
  - opened and clicked, when tracked
- **Live updates:** new messages appear at the top. Live updates can be paused.
- **Export to CSV:** needs transactional:content, because it holds addresses, and is audited.

States to show:
- live, with the sample rows
- a row expanded to explain a rejection
- test rows
- the 9 Oct outage: rows marked "Failed over to Amazon SES"
- filtered, with no results
- retention notice: "Messages older than 90 days are removed"
- loading, error, and empty (before the first message)

### 5.6 Message detail
A side panel from the log, with its own link.
- **Summary:**
  - the recipient, linked to their contact
  - the template and version, or Raw (SMTP)
  - the status, explained in plain words
  - the provider and its message ID
  - the key and IP, tags, metadata, idempotency key
- **Timeline:** each step with its timestamp.
  - accepted, rendered, routed (with the rule that matched), handed to the provider
  - every provider event: delivered, deferred, bounced, complained, opened, clicked
  - raw provider events link to Providers › Events
- **Content:**
  - the rendered HTML and text, and the headers
  - needs transactional:content
  - sensitive values masked
- **Actions:**
  - **Resend:** the same data, with a new message ID. Needs transactional:send. Unavailable when the template has a sensitive variable: "reset_url is sensitive and wasn't stored. Send it again from your app."
  - Remove from the suppression list, with a reason, where it applies. Needs subscribers:edit, and is audited, as on the Suppressions screen.
  - Copy as cURL.

States to show:
- delivered (Sam Rivera)
- rejected because suppressed, with the suppression's date, reason and source
- rejected because the recipient opted out of the category
- hard bounce, now added to the suppression list
- soft bounce
- deferred, then delivered
- failed: every provider failed, with the error of each attempt
- failed over to SES during the 9 Oct outage
- test mode, not delivered
- content removed after 30 days
- no permission to see content (Aisha): "Viewing message content needs transactional:content. Ask Marcus Chen or Linh Nguyen."
- loading, error

### 5.7 Setup: API and SMTP (Setup tab)
- **API:**
  - the base URL and the endpoints, each with one line about what it does
  - authentication and idempotency
  - limits: message size per route, 100 messages per batch, rate per key
  - an error table
  - a link to the instance's OpenAPI document
  - code samples in cURL, Node.js, Python, Go and PHP
- **Keys that can send:**
  - every key holding transactional:send, with name, mode (live or test), template scope, last used and created by
  - "Create send-only key": name, mode, template scope, allowed test addresses (test mode), expiry
  - the new key is shown once, with a filled-in snippet
- **SMTP relay:**
  - status, host and ports
  - SMTP credentials, listed with an SMTP badge, and "Create SMTP credential". Host, port, username and password are shown once.
  - which From addresses are allowed: the verified Transactional senders
  - the supported headers

States to show:
- relay off: only an instance admin can turn it on, and the screen names who can
- relay on (sample)
- relay unreachable: port 587 isn't reachable from the internet
- certificate expired
- key shown once
- key expired
- no verified Transactional sender: blocked, with a fix that leads to Senders

### 5.8 Settings (Settings tab)
- **Contact linking:** on or off.
- **Content retention:** 30 days, from 0 to 90.
- **Delivery alerts:** the three limits from 3.4. Who receives them: members with transactional:view, following their notification preferences.
- **Categories:**
  - name, kind (Essential or Optional), the description recipients see, templates and opt-outs
  - add, edit, delete
  - a category's kind can't change while it has templates
- **Defaults:** sender, reply-to and tracking for new templates.

States to show:
- the sample
- a validation error
- saved
- read-only

### 5.9 Import templates
A dialog in four steps:
1. **Source:** Postmark, SendGrid, Mailgun, Notifuse, or MJML and HTML files.
2. **Connect or upload:** the API key is sealed and never shown again.
3. **Pick:** choose the templates to import.
4. **Conversion report:** each template converted, converted with notes, or needing a manual fix, with notes by line. Then "Import as drafts".

States to show:
- invalid key
- the Postmark sample, with its partial conversion
- an unsupported construct, with its note
- done

---

## 6. Changes to existing screens

- **App shell:**
  - The sidebar gets "Transactional" between Automations and Audience.
  - The phone tab bar changes to four items plus More. Proposed: Overview, Campaigns, Audience, Deliverability.
  - More opens a sheet: Automations, Transactional, Forms & pages, Sending providers, Settings, plus the workspace switcher and the HostRepute status that the sidebar shows on larger screens. This is the board where the sheet pattern gets designed.
- **Settings › General:** a "Sidebar sections" switch for Transactional, on by default. Turning it off only hides the section from the sidebar; sending through the API keeps working.
- **Overview:** a Transactional card with the last 7 days' sent, delivery rate and median delivery, any active alert, and the route.
- **Sending providers › Routing:**
  - A fourth stream card, Transactional: primary, fallback, automatic failover, conditions by category and mailbox provider, and "Priority over Broadcasts on shared providers".
  - The route tester accepts "Transactional · password-reset".
- **Sending providers list:** each provider's numbers include transactional messages. The outage state says when transactional mail has failed over.
- **Connect a provider › Streams (step 5):** Transactional is a fourth stream to assign.
- **Settings › Senders:**
  - "From per stream" includes Transactional.
  - The notify.acme.io domain row.
  - A hint suggesting a separate subdomain for transactional mail.
- **Settings › API keys:**
  - A "Send-only key" preset in the create dialog: transactional:send, template scope, live or test mode.
  - Badges on key rows: Send-only, Test, SMTP.
  - The empty state's promise, "…send transactional mail from your product", now links to Transactional › Setup.
- **Roles & permissions:** the Transactional group (3.6) in the role editor and the compare matrix.
- **Settings › Approvals:** the optional rule for publishing transactional templates.
- **Contact profile:**
  - The activity timeline lists transactional messages: template, status and provider. Their content is behind transactional:content.
  - The contact's opt-outs of Optional categories.
- **Suppressions:**
  - An "Applies to" column and filter: All mail or Marketing only.
  - Marketing only is explained: "Transactional mail still reaches this address."
  - Unsubscribes and role addresses are Marketing only.
- **Public preference center:** a section "Notifications from Acme Cloud", with a switch for each Optional category and its description.
- **Public unsubscribe page:** the unsubscribe link of an Optional category turns off just that category, and says so.
- **Webhooks:** the "Transactional messages" event group.
- **Notifications:**
  - New kind: transactional delivery alert.
  - New kind, for instance admins: SMTP relay unreachable.
- **Audit log:** the actions from 3.7, and a "Transactional" filter category.
- **Migration importer › Source:** Notifuse, with base URL and API key. It imports contacts, lists and templates. Transactional templates arrive as drafts.
- **Welcome-series prompts:**
  - Where: List health after a list is created, Form builder when a form goes live, Migration done after an import finishes.
  - The card: "New subscribers of Product Updates get no welcome email yet. Start the welcome series". It opens the Welcome series starter with that list as its trigger.
  - It can be dismissed per list. It doesn't show when a live automation already starts on that list or form.
- **Instance admin › Instance settings:** an SMTP relay section: turn on, hostname, ports, certificate, status checks.
- **Deliverability and Integrations › HostRepute:** when HostRepute is connected, notify.acme.io appears among the assets not yet monitored, with "Add to monitoring".
- **Settings › MCP server:** the new tools in the agents' permission matrix: list templates, render a preview, read the log without content.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens and badge colors.

- **Code samples:** language tabs (cURL, Node.js, Python, Go, PHP) with copy, and a request and response viewer.
- **Variables:** a variables table next to a JSON test-data editor.
- **Message status badges:**
  - Queued: neutral
  - Sent: info
  - Delivered: ok
  - Deferred: warn
  - Bounced: bad
  - Complained: bad
  - Rejected: neutral, with the reason
  - Failed: bad
  - Test: purple
- **Masked sensitive value:** "••••••" with an eye-off icon and the label "Sensitive". No lock icon, because lock icons are kept out of the product.
- **Phone "More" sheet:** the tab bar's fifth item.

---

## 8. Conflicts to note

1. **Audit log vs message log.** CLAUDE.md says every state-changing action writes an audit entry. Writing every API send into the audit log would bury what people and keys change. This brief keeps each send in the message log (90 days) and audits everything else (3.7). The owner needs to confirm this.
2. **Streams.** The main spec defines three streams; this brief adds a fourth, Transactional.
3. **MCP agents.** The add-on's rule that MCP agents never send still holds. Agents can't send transactional mail; apps use send-only keys.

---

## 9. Boards

Canvas page "P1 Transactional email". Desktop boards are 1440 px wide unless noted.

| Board | Title on the canvas | Section | Tablet · phone |
|---|---|---|---|
| `Transactional.dc.html` | Transactional · templates, last 7 days | 5.1 | tablet, phone |
| `TransactionalStates.dc.html` | Transactional · first run, alert, no route, outage, read-only, loading | 5.1 | — |
| `TransactionalGuideSteps.dc.html` | First-send guide · steps 1 to 4: sender, route, template, key | 5.2 | — |
| `TransactionalGuide.dc.html` | First-send guide · step 5, delivered | 5.2 | — |
| `TransactionalGuideStates.dc.html` | First-send guide · waiting, rejected, done | 5.2 | — |
| `TransactionalTemplate.dc.html` | Template editor · password-reset v3, Variables | 5.3 | — |
| `TransactionalTemplateStates.dc.html` | Template editor · Liquid error, variables, test data, API, permissions | 5.3 | — |
| `TransactionalPublish.dc.html` | Publish dialog · trial-ending v2, breaking change | 5.4 | — |
| `TransactionalPublishStates.dc.html` | Publish · ready, approval, publishing, published, versions | 5.4 | — |
| `TransactionalMessages.dc.html` | Transactional · message log (live) | 5.5 | tablet, phone |
| `TransactionalMessage.dc.html` | Transactional · message detail (delivered) | 5.6 | phone |
| `TransactionalMessageStates.dc.html` | Message detail · rejected, bounced, failed, failover, test, retention, permission | 5.6 | — |
| `TransactionalSetup.dc.html` | Transactional · setup: API, keys, SMTP relay | 5.7 | — |
| `TransactionalSetupStates.dc.html` | Setup · create key, shown once, relay off, unreachable, certificate, expired key | 5.7 | — |
| `TransactionalSettings.dc.html` | Transactional · settings, categories, alerts | 5.8 | — |
| `TransactionalImport.dc.html` | Import templates · Postmark conversion report | 5.9 | — |
| `TransactionalImportStates.dc.html` | Import templates · source, refused token, pick, done | 5.9 | — |
| `PublicUnsubscribeCategory.dc.html` | Public · unsubscribe from one Optional category (720 px) | 6 | — |
| `ShellMore.dc.html` | App shell · phone tab bar and the More sheet (390 × 844) | 6 | — |
| `DSExpansion.dc.html` | Design system · expansion patterns (page "9 Design system") | 7 | — |

Existing boards changed by section 6: `Main`, `ProvidersRouting`, `SetupStreams`, `SettingsSenders`, `SettingsKeys`, `SettingsKeysStates`, `SettingsStates`, `Roles`, `RoleEditor`, `ApprovalPolicy`, `ContactProfile`, `Suppressions`, `PublicPreferences`, `SettingsAPI`, `Notifications`, `SettingsAudit`, `MigrationSource`, `ListHealth`, `FormBuilder`, `MigrationDone`, `InstanceSettings`, `SettingsIntegrations`, `SettingsMCP`, `WorkspaceSettings`, and the sidebar of every app-shell board.
