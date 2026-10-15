# ReputeCast — Add-on spec: SMS, push, WhatsApp and agencies

Phase 8 of `reputecast-design-expansion-plan.md`: SMS, web push, WhatsApp (optional, last) and agency accounts.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and the earlier phases already exists.**
  - Add the screens in section 5 as a new canvas page, "P8 SMS, push & agencies".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **This phase builds on:**
  - The main spec: providers, routing, the shared suppression list, consent per list, campaigns, Send Readiness and automations.
  - The team add-on: roles, approvals, Instance admin and workspaces.
  - Phase 1: the message log's retention rules, and the phone "More" sheet.
  - Phase 7: the publication website serves the push service worker; Pages › Built-in pages lists the text message terms page; the Summit's registration page uses the Webinar registration template.
  - Where an earlier phase's brief names something differently, follow that brief and note the difference.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and the earlier phases'.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.3 and 5.14. Phone boards for 5.5, 5.11 and 5.18.
  - Public pages at 720 px, plus phone: 5.7.
- **Build order:**
  1. 5.1 SMS providers and senders
  2. 5.3 SMS composer
  3. 5.4 Review & send for SMS
  4. 5.5 SMS report
  5. 5.2 SMS routing and rules
  6. 5.6 Connecting an SMS provider, and the SMS states
  7. 5.8 Steps in automations
  8. 5.7 Text message terms
  9. 5.9 to 5.12 Web push
  10. 5.14 to 5.18 Agencies
  11. 5.13 WhatsApp, last
  12. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.**
   - ReputeCast charges nothing per message, number, client or seat.
   - Texts and WhatsApp messages cost what the workspace's own provider charges, and every screen shows that cost before anything is sent.
   - An agency's rebilling is the agency's own business tool. ReputeCast takes nothing from it.
2. **Consent per channel, per list.**
   - Being on a list by email says nothing about texting the same person.
   - SMS, WhatsApp and push each need their own consent, kept with the page, time, IP address and exact text, as email consent is.
3. **Sender reputation first, on every channel.**
   - SMS gets routing by country, a shared SMS suppression list, guarded sending, a monthly budget and a pre-send check of its own.
   - Push gets a frequency cap and a pre-send check.
4. **Quiet by default.** Texts and notifications respect quiet hours in the recipient's time zone.
5. **The cost comes before the send.** Every SMS and WhatsApp send shows its cost per country and against the month's budget.
6. **Pluggable, like sending providers.**
   - SMS providers get a catalog with capability badges, one connection pattern, health, write-only credentials and a shared contract-test suite.
   - Web push is built in and needs no outside account.
7. **One client's data never reaches another.**
   - Agency staff see only the clients they're assigned to.
   - Client people see only their own workspace.
8. **Same access rules as the team add-on.**
   - Every action has a permission key, and nobody grants what they don't hold.
   - Every change is audited.
   - MCP agents draft and propose. They never send.

---

## 2. Terms

- **Channel:** email, SMS, push or WhatsApp.
- **SMS provider:** the service that delivers texts: Twilio, Vonage, Plivo, Telnyx or AWS End User Messaging.
- **Sender:** what a text appears to come from.
  - **Toll-free number:** a US and Canada number. It needs toll-free verification before it can send.
  - **10DLC number:** a US local number for business texting. It needs a registered brand and campaign (A2P 10DLC).
  - **Alphanumeric sender ID:** a name such as "ACME", up to 11 letters and digits. Allowed in some countries, registered in others, not supported in the US and Canada. People can't reply to it.
- **Registration:** the carriers' approval of a sender: toll-free verification, a 10DLC brand and campaign, or a country's sender ID registration.
- **Segment:** the unit texts are delivered and billed in.
  - **GSM-7:** the basic character set. 160 characters fit in one segment; longer texts use 153 per segment.
  - **UCS-2:** used as soon as one character is outside GSM-7, such as an emoji or a curly apostrophe. 70 characters in one segment, 67 per segment after that.
- **Keyword:** a reply ReputeCast acts on: opt-out (STOP), opt-in (START) or help (HELP).
- **SMS consent:** consent to texts, kept per list, separate from email consent.
- **SMS suppression list:** numbers that must get no texts. It's shared by every SMS provider.
- **Quiet hours:** the hours outside the sending window, in the recipient's time zone.
- **Short-link domain:** the workspace's own domain for links in texts (go.acme.io).
- **Delivery receipt:** the carrier's report on a text, through the provider.
- **Push subscriber:** a browser that allowed notifications from a site. Linked to a contact when ReputeCast knows who it is; otherwise anonymous.
- **VAPID keys:** the key pair that identifies the workspace to the browsers' push services.
- **Service worker:** the script on a site that receives notifications. It must be served from that site.
- **Soft prompt:** ReputeCast's own invitation, shown before the browser's permission prompt.
- **WhatsApp template:** a message format approved by Meta. Messages a business starts must use one.
- **Customer service window:** the 24 hours after a contact's last WhatsApp message, when free-form replies are allowed.
- **Agency:** the business that runs the instance for its clients (Northwind Studio).
- **Client workspace:** a workspace that belongs to one client.
- **Agency staff:** the agency's people.
  - **Agency manager:** works across every client and manages the agency.
  - **Account manager:** works on the clients they're assigned to.
- **Client approver and Client viewer:** built-in roles for the client's own people.
- **Workspace template:** a blueprint for setting up a client workspace.
- **Shared library:** sections and templates the agency shares across its clients.
- **Rebill rate:** how the agency charges a client for that client's usage.

---

## 3. Requirements

### 3.1 SMS providers and senders
- **Where:** Sending providers keeps its name and gets a channel switch at the top: Email · SMS · Push · WhatsApp.
  - The switch appears once a second channel is set up. Until then, the header has "Add a channel".
  - Email keeps its tabs: Providers · Routing & failover · Warm-up · Events & sync · Adapters.
  - SMS: Providers · Routing & rules · Events.
  - Push: Setup · Events.
  - WhatsApp: Account · Templates · Events.
- **Catalog:** the connect wizard's catalog gets channel tabs: Email · SMS · WhatsApp.
  - SMS lists Twilio, Vonage, Plivo, Telnyx and AWS End User Messaging.
  - Capability badges come from each adapter's manifest:
    - sender kinds: toll-free, 10DLC, alphanumeric sender IDs
    - delivery receipts
    - inbound messages (keywords)
    - opt-out sync with the provider's own opt-out list
    - registration status through the API
    - pricing through the API
- **Connecting** is the wizard's SMS path, in five steps:
  1. **Provider:** the SMS tab of the catalog.
  2. **Connection:** the provider's credentials, all write-only.
     - Twilio: Account SID with an API key and secret.
     - Vonage: API key, secret and signature secret.
     - AWS End User Messaging: a CloudFormation role or IAM keys, and the region.
     - Delivery-receipt and inbound webhooks are registered automatically where the API allows it. Otherwise ReputeCast shows an address and a secret to paste.
  3. **Senders:** import the numbers already in the account, or add an alphanumeric sender ID, and pick its countries.
  4. **Test message:** to a member's phone. The timeline waits for the delivery receipt, like the email test send.
  5. **Countries:** add the senders to SMS routes.
- **Webhook ingest:** as for email. Verify the signature, store the raw event, answer 200 at once, normalize later in a River job.
- **Health per provider:** Healthy, Degraded, Down or Credentials rejected.
  - Delivery-receipt freshness: "Receipts silent for 2 hours" while texts were being sent.
  - Error rate over 7 days, and spend this month.
- **Senders table** (numbers and sender IDs): kind, provider, countries, registration status, throughput, and the routes that use it.
  - Registration statuses:
    - Not registered
    - Pending
    - Verified (toll-free) or Approved (10DLC, sender IDs)
    - Rejected, with the carrier's reason
    - Allowed without registration
    - Not supported
  - The status is read from the provider's API where it allows (hourly, by the existing provider health job). Otherwise it's set by hand, with a date.
  - A sender can't send to a country where its registration is Pending, Rejected or Not supported. The pre-send check names the recipients this leaves without a sender.
- **Registration helper:** for 10DLC campaigns and toll-free verification, ReputeCast prepares what the provider's form asks for, with copy buttons:
  - the opt-in description: the form, its page and its consent text
  - sample messages from the workspace's texts
  - the help and opt-out replies
  - the address of the text message terms page (5.7)

  Registering happens in the provider's console. ReputeCast only reads the status back.
- **Contract tests:** every SMS adapter passes the shared SMS contract suite:
  - send
  - delivery receipts and inbound messages
  - signature checks
  - opt-out sync and registration status, where the adapter declares them

### 3.2 Routing by country
- **Routes:** rules from countries to a provider and sender, with an optional fallback sender.
  - A fallback must be registered for those countries. Otherwise the fallback is "None: hold and notify".
  - Failing over changes the sender people see, and the rule says so.
- **Every other country:** "Don't send" by default, or a chosen route.
- **The country comes from the phone number,** not from the contact's address.
- **Route tester:** "Which sender would text +44 7700 900456?"
  - It answers with the provider, the sender, the price per segment, and whether quiet hours would hold the text right now.

### 3.3 Phone numbers and SMS consent
- **Phone numbers:**
  - One per contact, in E.164 format, with its country.
  - The line type (mobile, landline, unknown), learned from provider errors.
  - Two contacts can share a number. A STOP from that number applies to both.
- **SMS consent per list:** Subscribed, Not subscribed or Opted out.
  - Every change keeps its evidence: the number, the list, the exact consent text and its version, the page or source, the IP address and the time.
  - Email unsubscribes don't touch SMS consent, and STOP doesn't touch email.
- **Collected by:**
  - **Signup forms:** a phone field with a country picker, and an SMS consent checkbox with its own text, unchecked by default.
    - The default text names the program, the message frequency, "Msg & data rates may apply", HELP and STOP, and links to the terms page and the privacy policy.
  - **The preference center and the account page:** a "Texts" switch per list. It records the same evidence.
  - **Imports:** SMS consent is imported only with evidence columns: date, source and text. Without them, the number is imported and SMS consent stays Not subscribed.
  - **The API:** with subscribers:edit, and the evidence fields required.
- **Opt-in confirmation text,** on by default:
  - After someone gives SMS consent, ReputeCast texts them once: "Acme: You're signed up for Acme Summit 2026 texts. Up to 4 msgs. Msg & data rates may apply. Reply HELP for help, STOP to cancel."
  - If it fails because the number is a landline or invalid, the number is suppressed.
  - Optional, off by default: consent counts only after they reply YES.

### 3.4 Keywords, opt-outs and SMS suppressions
- **Opt-out keywords:** STOP, STOPALL, UNSUBSCRIBE, CANCEL, END, QUIT, OPTOUT and REVOKE.
  - More can be added, including in other languages. A keyword matches when it's the whole reply, in any case.
  - Effect: SMS consent is withdrawn on every list, for every contact with that number. The number goes on the SMS suppression list as "Opted out by text". One confirmation reply is sent.
- **Opt-in keywords:** START and UNSTOP. The suppression is lifted and the consents that STOP removed come back, each recorded as "Opted back in by text".
- **Help keywords:** HELP and INFO, answered with the help reply.
- **Who answers:** ReputeCast by default. For a provider that handles keywords itself, the provider can answer instead; ReputeCast then syncs its opt-out list and sends no reply of its own.
- **Other replies** go on the contact's timeline and into the campaign report. There's no inbox in this phase.
- **SMS suppression list:** Audience › Suppressions, with a channel filter, Email · SMS.
  - Reasons: Opted out by text, Landline, Not in service, Invalid number, Filtered by carriers (3 times in 30 days), Provider opt-out list, Manual.
  - Numbers are shown partly masked: +1 773 ••• 0148.
  - A number that opted out by text can't be removed by hand. Only the person can come back, by texting START.
  - Other reasons can be removed with a reason (subscribers:edit, audited).
  - The list syncs with each provider's opt-out list where the adapter supports it.

### 3.5 Quiet hours
- **Window:** 09:00–20:00 in the recipient's time zone by default. It can be set anywhere between 08:00 and 21:00. ReputeCast never texts outside 08:00–21:00 local time.
- **Time zone:** the contact's time zone field. Otherwise it comes from the phone number: the area code where it maps to one time zone, else the country's main time zone. The pre-send check counts the recipients whose time zone was assumed.
- **Campaigns:**
  - "Now" holds the recipients who are outside their window until it opens.
  - "At a time in each recipient's time zone" offers only times inside the window.
- **Automations:** the SMS step waits for the window (default) or skips.
- **Push and WhatsApp** use the same window, on by default.

### 3.6 SMS campaigns and the SMS step
- **New campaign** asks for the channel first: Email, SMS, Push or WhatsApp. SMS campaigns are Regular only.
- **SMS composer:**
  - **Header:** name; audience (lists and segments); the route line; status; Send test; Review & send.
    - The audience count is "can receive texts": SMS consent on a list in the audience, a number, and not suppressed.
  - **Message:**
    - The brand prefix ("Acme:") and the opt-out text ("Reply STOP to opt out") come from the rules. They show as locked parts and count toward the length.
    - The editable text sits between them, with personalization tokens and fallbacks, Insert link, and, in agencies, Insert from library.
  - **Counter:** characters, segments and encoding: "130 characters · 1 segment · GSM-7". A bar marks each segment boundary.
  - **Encoding:** a character outside GSM-7 is highlighted with a fix: "’ (curly apostrophe) switches this text to UCS-2: 2 segments instead of 1. Replace with '".
  - **Personalization:** "Length depends on first name: 1 segment for 96% of recipients, 2 for 4%."
  - **Links:**
    - Every link is shortened on the short-link domain, with a code per recipient: go.acme.io/k3P9xq, 17 characters. It redirects with UTM parameters.
    - Short links work for 12 months.
    - Public shorteners such as bit.ly are blocked, because carriers filter them.
  - **Preview:** a phone showing the sender as each country sees it ("+1 888 555 0142" or "ACME"), with "Preview as" a contact and a country picker.
  - **Cost:** per country: recipients, sender, segments, price per segment and cost.
    - Prices come from the provider's pricing, or a price the workspace entered.
    - The total, and the month's budget before and after this send.
- **Review & send for SMS:** the same layout as for email, with an SMS check list (Block, then Warn, then Pass):
  - consent: how many people in the audience have SMS consent. People without it are left out, so this check never blocks.
  - SMS suppressions left out
  - each sender's registration, per country
  - a route for every recipient's country
  - quiet hours, and the recipients with an assumed time zone
  - opt-out text: Block when a campaign has none
  - the sender named at the start: Warn when missing
  - links on the short-link domain: Block for public shorteners
  - length: Warn above 3 segments
  - budget: Block when the send doesn't fit the month's remaining budget
  - Then: verdict, recipients, route breakdown, cost, duration from each sender's throughput, schedule, auto-pause thresholds, test send and approvals, as for email.
  - Sends above 1,000 recipients or $100 need the recipient count typed to confirm.
- **Guarded sending:**
  - Auto-pause, per campaign, with workspace defaults in Routing & rules:
    - failed above 5%
    - filtered by carriers above 2%
    - opt-outs above 3%
    - each counted once 50 texts have been sent
  - **Monthly budget** per workspace, with an alert at 80%. At 100%, campaigns can't start and SMS steps wait, until someone with providers:manage raises it.
  - SMS campaigns don't open the live monitor. Their report shows the progress while sending, with Pause and Cancel.
- **SMS step in automations:**
  - The same message editor and counter, inline.
  - "Outside quiet hours: wait for the window" (default) or "skip".
  - The step names the list whose SMS consent it needs. Contacts without that consent are skipped, and the rule can't be removed.
  - "Skip if the contact got a text in the last [N] hours."
  - Step stats for 30 days: sent, delivered, clicks, opt-outs, cost.
  - New branch conditions: has SMS consent, clicked the text's link, replied.
- **Approvals:** the workspace's rules apply to SMS campaigns. "The author lacks the send permission" means sms:send here. A new optional rule: "An SMS or WhatsApp campaign is estimated to cost at least $[N]".
- **Retention:** text content 30 days by default (0 to 90). Delivery events 90 days.

### 3.7 SMS reports and events
- **Report:**
  - sent, delivered, failed (by reason, with the provider's own code), clicks, opt-outs, help requests, other replies and cost
  - clicks by people apart from link previews, which are counted as machine clicks
  - by country and by sender
  - recipients held or skipped, and why
- **Events tab:** each raw provider event next to its normalized one: queued, sent, delivered, undelivered, failed, and inbound keyword or reply. Opt-out sync per provider.
- **Contact timeline:** each text with its status, clicks and replies.
- **Webhooks:** an "SMS" event group: sms.delivered, sms.failed, sms.clicked, sms.opted_out, sms.opted_in, sms.replied.

### 3.8 Web push
- **Built in:** Sending providers › Push › Setup. No outside account is needed.
- **Keys:**
  - VAPID keys are generated once. The public key is shown; the private key is sealed.
  - A contact address for the push services: mailto:support@acme.io.
  - Rotating the keys ends every subscription. A typed confirmation shows how many.
- **Sites:**
  - **The publication site (phase 7):** ReputeCast serves the service worker and the prompt. Nothing to install.
  - **Another site:** a snippet, plus a service-worker file to host at the site's root, because browsers require the same origin.
    - "Check" fetches the file and reports whether it was found, its version and its scope.
    - If the site already has a service worker, one import line goes into that worker instead.
  - Subscriptions from sites that aren't listed are refused.
- **Subscribers:**
  - Each subscription has its site, browser and time zone (from the browser), and a consent record: page, exact prompt text and version, IP address and time.
  - It's linked to a contact when the reader is a signed-in member, arrived from an email or text link that carried a token, or turned notifications on from the account page. Otherwise it's anonymous.
  - Unsubscribing: in the browser, or "Turn off notifications" on the account page and the preference center. A subscription the push service reports as gone is removed.
- **Prompt designer:** a new form type, Forms & pages › Signup forms › New form › "Push prompt".
  - Style: slide-in bar, dialog or bell button.
  - Text: title, message, Allow and Not now labels, icon.
  - When: after N seconds, after scrolling N% of a page, after reading N posts, or when the reader clicks the publisher's own button.
  - Where: sites and page paths.
  - Frequency: after "Not now", ask again in N days (default 14). Never shown in a browser that blocked notifications.
  - The browser's own prompt appears only after "Allow". "Not now" never uses up the browser's one chance to ask.
  - iPhone and iPad support web push only for sites added to the Home Screen. The prompt either shows how to add the site or stays hidden there.
  - Previews per browser. Numbers: shown, Allow, permission granted, Not now, closed.
- **Push campaigns:**
  - title (up to 50 characters recommended), message (up to 120), link, icon (the site's), image (shown by Chrome on Windows and Android), up to 2 buttons (Chrome and Edge), expiry (default 4 hours), and "Replace an earlier notification with the same tag"
  - audience: sites, optionally narrowed to subscribers linked to contacts in chosen lists or segments
  - previews: Chrome on Windows, Chrome on macOS, Safari on macOS, Chrome on Android, Safari on iPhone (Home Screen)
  - "Send to my browser" as the test
  - pre-send check:
    - the keys
    - the service worker found on each site
    - the audience
    - the frequency cap: at most 2 notifications a day per subscriber by default; anyone over it is skipped
    - quiet hours
    - the link works
    - lengths per platform
    - the image's size
  - approvals as for other campaigns
- **Push step in automations:** sends to the contact's linked subscriptions, and is skipped when there are none. Outside quiet hours it waits for the window (default) or skips, as the SMS step does.
- **Report:**
  - sent (accepted by the push services)
  - shown (reported by the service worker)
  - clicks
  - unsubscribed within 24 hours
  - expired subscriptions removed
  - by browser and by site

### 3.9 WhatsApp (optional, last)
- **Provider:** Meta's WhatsApp Cloud API, the only one in this phase. Under Sending providers › WhatsApp.
- **Connecting:**
  - WhatsApp Business Account ID, phone number ID, a permanent access token (system user) and the app secret. All write-only.
  - The webhook address and verify token, to paste in Meta's app settings. ReputeCast shows "Waiting for Meta's verification request" until it arrives.
  - A test message to a member's WhatsApp.
- **Account card:** display name and its approval, the number and its status, quality rating (High, Medium, Low) and messaging limit, as Meta reports them.
- **Templates:**
  - Synced from Meta every hour and on demand: name, category (Marketing, Utility, Authentication), language, body with its variables, buttons.
  - Statuses: Approved, Pending, Rejected (with Meta's reason), Paused, Disabled.
  - Templates are created and edited in Meta's WhatsApp Manager. The tab links there.
- **Consent:** WhatsApp consent per list, like SMS, with its own text. The opt-out keywords work on WhatsApp too.
- **Sending:**
  - **WhatsApp campaigns:** an approved template, its variables mapped to contact fields with fallbacks, an audience with WhatsApp consent, a schedule and quiet hours.
  - **WhatsApp step in automations:** an approved template; or free text, sent only inside the customer service window, otherwise skipped or replaced by a chosen template.
  - The contact profile shows the window: "Can get free-form replies until 15 Oct 18:02".
  - **Pre-send check:** consent, the template's status and category, quality rating, the messaging limit against the audience, and the cost.
- **Cost:** Meta charges per delivered template message, by category and the recipient's country. ReputeCast estimates it from Meta's rate card, shows the card's date, and lets the workspace edit it.
- **Report:** sent, delivered, read (only from people who share read receipts), failed (with Meta's codes), button clicks, replies, opt-outs and cost.
- **Webhooks** are checked with the app secret, stored raw and normalized later.

### 3.10 Agency accounts
- **Agency mode:** a new "Agency" section in Instance admin › Instance & updates. Instance admins turn it on.
  - The agency's name (the instance name by default).
  - Which workspaces are clients, and which is the agency's own.
  - The default workspace template for new clients.
  - One agency per instance.
- **Agency area:** for agency staff, opened from the top of the workspace switcher: "Northwind Studio · Agency".
  - Its own sidebar items: Overview · Staff · Library · Usage. On phones the tab bar shows them.
- **Staff** (instance level):
  - **Agency manager:** every client. Creates and deletes client workspaces, manages staff and assignments, the library, workspace templates, usage and rates. Instance admins are agency managers.
  - **Account manager:** the clients they're assigned to: the overview, search, usage and the library.
  - **Assignments:** staff × client × workspace role.
    - An assignment is an ordinary workspace membership. Team and the member page show its source, "Northwind Studio", as they show an SSO group mapping.
    - Agency managers get Admin in every client workspace this way.
    - Removing an assignment removes the membership, and asks about the person's API keys, as deactivating does.
    - Escalation guard: you can assign only roles whose permissions you hold in that workspace.
- **Clients:**
  - **New client:** name, slug, time zone, workspace template (optional), the Owner (an agency manager), account managers, and optionally the client's first person, invited as Client approver or Client viewer.
  - Each client workspace is an ordinary workspace, with its own providers, domains, contacts, suppression list, consent, API keys and audit log. Nothing is shared between clients: a bounce at one client never suppresses an address at another.
  - Client details for the agency: account manager, rebill rate, client since.
  - Client people see "Managed by Northwind Studio · your contact: Mei Lin" in Settings › General.
  - Deleting a client uses the existing workspace delete: typed name, export first, 7-day restore.
- **Client roles:** built-in in every client workspace, read-only like the others, and duplicable.
  - **Client approver:** sees campaigns, templates, automations, reports, deliverability and contacts (no export). Approves sends (campaigns:approve ⚑) and comments.
  - **Client viewer:** sees campaigns, templates, automations, reports and deliverability. No contacts.
  - Client people never see the Agency area, other clients, or staff who aren't members of their workspace.
- **Client sign-off:** Settings › Approvals in client workspaces gets a rule: "Every send needs one approval from a Client approver".
  - It covers campaigns on every channel, and optionally publishing automations.
  - Approvals from agency staff don't count toward it.
- **Workspace templates:**
  - **Copied:**
    - sender defaults: names and footer text, not domains
    - lists, empty
    - automations, as drafts, with their triggers pointed at the new lists
    - saved sections, campaign templates and SMS texts
    - signup forms, paused until a sender is verified
    - the approval policy and roles
    - branding, with the logo and colors left empty
  - **Never copied:** providers and credentials, domains and senders, contacts, suppressions, consent, API keys, integrations, members.
  - Created from an existing client ("Save as workspace template", needs agency:library) or edited in Library.
  - Applied when a client is created, or later to an existing client. Applying adds what's missing, renames clashes ("Welcome series (template)"), and never overwrites or deletes. The result lists what was added.
- **Shared library:**
  - Items: saved sections, campaign templates, automation templates, transactional templates and SMS texts.
  - Added from any client workspace with "Add to Northwind library": needs agency:library, and the item's view permission in that workspace.
  - Used in a client through a "Northwind library" tab in the composer's template gallery and in the automation starters. Using an item makes a copy, which needs templates:edit or automations:edit there.
  - Copies aren't linked. When the library item changes, the copy shows "Library has a newer version", with the diff and "Update this copy".
  - Items with Liquid tokens list the fields they need.
- **Usage per client,** per month:
  - emails by stream, SMS segments, push notifications and WhatsApp messages
  - provider cost, from each provider's pricing or the price the workspace entered
  - **Rebill rates** per client: cost plus a percentage, a price per 1,000 emails, a price per SMS segment, a monthly fee, or a flat monthly amount. They can be combined.
  - The "To rebill" column is calculated. ReputeCast doesn't send invoices or take payments.
  - Export CSV (one row per client and channel), and the API (agency:usage).
- **Agency overview:** per client:
  - health: the Overview's sender health score
  - the next send
  - approvals waiting, and who they wait for
  - incidents in the last 7 days: auto-pauses, provider outages, webhooks silent, SMS budget reached, a store disconnected
  - the account manager
  - Filters: All, Needs attention (health below 80, an incident, or an approval waiting over 24 hours), My clients. "New client".
- **Search across clients:**
  - Campaigns, automations, templates, forms and library items, by name.
  - Contacts only by exact email or phone, and only in workspaces where the person searching has subscribers:view. Contact searches are audited.
- **Staff notifications:** incidents and approvals in their clients, following their notification preferences.

### 3.11 Permissions
New workspace permissions:
- SMS: send or schedule SMS campaigns, and publish automations with SMS steps (sms:send) ⚑
- Push: send or schedule push campaigns, and publish automations with push steps (push:send) ⚑
- WhatsApp: send or schedule WhatsApp campaigns, and publish automations with WhatsApp steps (whatsapp:send) ⚑

Sending an SMS campaign needs campaigns:send and sms:send; preparing one needs campaigns:edit. The same goes for push and WhatsApp.

Existing keys that apply:
- providers:view and providers:manage ⚑: SMS, push and WhatsApp providers, senders, routing, rules, the budget, VAPID keys and sites.
- forms:edit: push prompts, phone fields and SMS consent text.
- subscribers:view, subscribers:edit and subscribers:export ⚑: phone numbers, consent per channel and SMS suppressions.
- campaigns:approve ⚑: approving campaigns on every channel.

New agency permissions, at instance level, held by agency roles rather than workspace roles:
- view the agency overview and search across your clients (agency:view)
- see usage and costs per client, set rebill rates and export usage (agency:usage)
- manage the shared library and workspace templates (agency:library)
- create and delete client workspaces, and manage staff and assignments (agency:manage) ⚑

Who gets what:
- **Built-in roles:**
  - Owner, Admin and Editor: sms:send, push:send and whatsapp:send.
  - Viewer: none of them.
  - Client approver: campaigns:view, campaigns:approve, templates:view, automations:view, reports:view, deliverability:view, subscribers:view.
  - Client viewer: campaigns:view, templates:view, automations:view, reports:view, deliverability:view.
- **Content Writer**, the sample's custom role: none. Their SMS campaigns go through "Request approval".
- **Agency roles:**
  - Agency manager: all four agency permissions.
  - Account manager: agency:view, agency:usage and agency:library, for their clients.
- **API keys:** the channel permissions, never more than their creator holds. Agency permissions can't be given to a workspace's API keys.
- **MCP agents:** can draft SMS, push and WhatsApp campaigns, and read reports, costs and registration statuses. Sending becomes a proposal for a person. They never send, and never change keywords, quiet hours or the budget, or lift a suppression.

### 3.12 Audit
- **Audited:**
  - SMS, push and WhatsApp providers connected, reconnected or removed; credentials replaced
  - senders added or removed; registration statuses set by hand
  - routes and rules changed: keywords, replies, quiet hours, opt-out text, the budget, auto-pause thresholds
  - the short-link domain added or verified
  - SMS, push and WhatsApp campaigns sent, scheduled, paused, resumed or cancelled, as for email
  - the SMS budget reached (by the system) and raised (by a person)
  - a number taken off the SMS suppression list, with the reason
  - SMS or WhatsApp consent changed by a person
  - VAPID keys created or rotated; push sites added or removed; a push prompt published or paused
  - WhatsApp templates synced (by the system)
  - agency mode turned on or off
  - client workspaces created (with the template used) or deleted
  - staff added or removed; assignments changed
  - workspace templates created, changed or applied; library items added, changed, removed or copied into a client
  - rebill rates changed; usage exported
  - contact searches across clients: who searched, and for which address or number
- **Not in the audit log:**
  - each text, notification or WhatsApp message (the SMS log and the reports keep them for 90 days)
  - each inbound keyword or reply (on the contact's timeline)
  - each push subscription (counted per site)

  See section 8.

### 3.13 HostRepute
- When HostRepute is connected and the short-link domain is verified, offer "Add go.acme.io to HostRepute monitoring", with the existing "Add to monitoring" flow. A link domain on a blocklist gets texts filtered too.
- Agencies: each client workspace connects to HostRepute on its own. The agency overview shows a client's HostRepute status where it's connected.
- No phone numbers, texts, notifications or recipients are ever sent to HostRepute.

### 3.14 Out of scope for this phase
- MMS, RCS and short codes.
- Two-way conversations and a shared inbox.
- Texts through the transactional API.
- Buying numbers, or submitting registrations, from ReputeCast (status and the helper only).
- Number lookups during imports.
- A/B tests of texts and notifications.
- Push for native mobile apps.
- Creating WhatsApp templates in ReputeCast, other WhatsApp providers, and a WhatsApp inbox.
- Agency invoicing and payment collection, reseller pricing, agency-wide API keys and MCP agents, one sending provider shared by several client workspaces, and moving a workspace between instances.

---

## 4. Sample data

All Acme data is in the "Acme Newsletter" workspace at reputecast.acme.io. The agency data is on a separate instance.

### SMS providers
| Provider | Account | Senders | Status | Spend, October |
|---|---|---|---|---|
| Twilio | Acme Inc, live · Account SID AC…3f9 · connected 1 Oct 2026 by Marcus Chen | +1 888 555 0142 (toll-free) · +1 312 555 0187 (10DLC) | Healthy · last receipt 3 min ago | $14.71 |
| Vonage | Acme Inc · API key …7c21 · connected 2 Oct 2026 by Marcus Chen | ACME (sender ID) | Healthy · last receipt 41 min ago | $24.80 |

- **Not connected:** Plivo, Telnyx, AWS End User Messaging.
- **October so far:** $39.51 of a $100 budget.
  - Opt-in confirmations: $19.90.
  - The 14 Oct campaign: $19.54.
  - 6 test texts: $0.07.

### Senders
| Sender | Kind | Provider | Countries | Registration | Throughput | Route |
|---|---|---|---|---|---|---|
| +1 888 555 0142 | Toll-free | Twilio | US, CA | Verified 6 Oct 2026 | 3 /s | US and Canada |
| +1 312 555 0187 | Local, 10DLC | Twilio | US | Brand "Acme Inc" approved 2 Oct; campaign "Acme Summit attendee updates" pending since 9 Oct | – | None yet |
| ACME | Sender ID | Vonage | GB, DE, NL | Allowed without registration | 30 /s | UK, Germany, Netherlands |
| ACME | Sender ID | Vonage | IE | Pending since 9 Oct 2026 | – | Ireland (held) |

- Alphanumeric sender IDs aren't supported in the US and Canada; those countries use the toll-free number.

### Routes
| Countries | Provider · sender | Fallback |
|---|---|---|
| United States, Canada | Twilio · +1 888 555 0142 | None: hold and notify |
| United Kingdom, Germany, Netherlands | Vonage · ACME | None: hold and notify |
| Ireland | Vonage · ACME (pending) | None: skipped until approved |
| Every other country | Don't send | – |

- **Route tester:** +44 7700 900456 → Vonage, ACME, $0.0450 per segment. At 21:40 London time: "Quiet hours: it would wait until 09:00."
- **Prices per segment** (from each provider's pricing on 14 Oct 2026): US and Canada $0.0110 (carrier fees included), UK $0.0450, Germany $0.0880, Netherlands $0.0960, Ireland $0.0760.

### Rules
- **Brand prefix:** "Acme:".
- **Opt-out text:** "Reply STOP to opt out". Always on campaigns; on by default in automation steps.
- **Quiet hours:** 09:00–20:00 in the recipient's time zone.
- **Keywords:** the defaults, answered by ReputeCast.
- **Help reply:** "Acme: Help at acme.io/sms or support@acme.io. Msg & data rates may apply. Reply STOP to cancel."
- **Opt-out reply:** "Acme: You're unsubscribed from Acme texts and won't get more. Reply START to resubscribe."
- **Opt-in confirmation:** "Acme: You're signed up for Acme Summit 2026 texts. Up to 4 msgs. Msg & data rates may apply. Reply HELP for help, STOP to cancel."
- **Short-link domain:** go.acme.io, verified 3 Oct 2026 (a CNAME to reputecast.acme.io), Let's Encrypt certificate.
- **Budget:** $100 a month, alert at 80%.
- **Auto-pause:** failed above 5%, filtered above 2%, opt-outs above 3%.
- **Test numbers:** Marcus Chen +1 312 555 0161, Sofia Rossi +44 7700 900456.

### Acme Summit 2026
- 22 Oct 2026, 09:00–17:00 America/Chicago, at Lakeside Hall in Chicago, and online.
- **Registration page:** news.acme.io/summit, made from the Webinar registration template.
- **The list "Acme Summit 2026":** 1,240 registrants. 902 were already contacts and 338 are new, so the workspace has 49,107 contacts: Phase 7's 48,769 plus 338.
- **Form fields:** email, first name, company, phone (optional), SMS consent (optional, unchecked), WhatsApp consent (optional, unchecked).
- **SMS consent text, v1:** "Text me reminders about Acme Summit 2026. Up to 4 messages. Msg & data rates may apply. Reply HELP for help, STOP to cancel. Terms: news.acme.io/sms-terms"
- **Phones and consent:**
  - 1,032 registrants gave a phone number; 891 ticked SMS consent.
  - 10 replied STOP to the opt-in confirmation.
  - 7 numbers were suppressed when the confirmation failed: 5 landlines, 2 invalid.
  - That leaves 874 who can receive texts: US 611, CA 48, GB 118, DE 52, NL 25, IE 20.
  - The 20 in Ireland haven't had their confirmation text yet: it's held until ACME is approved there.
- **WhatsApp consent:** 214. **Push:** 612 subscriptions linked to registrants.
- **Sam Rivera** registered on 2 Oct with +1 312 555 0123 (Chicago, America/Chicago), and gave SMS and WhatsApp consent.

### SMS campaign: "Summit: 8 days to go"
- Written and sent by Sofia Rossi on 14 Oct, at 10:00 in each recipient's time zone: from 08:00 UTC in Germany and the Netherlands to 17:00 UTC on the US Pacific coast.
- **Text** (130 characters, 1 segment, GSM-7): "Acme: 8 days to Acme Summit 2026! The agenda is live: go.acme.io/k3P9xq See you 22 Oct in Chicago or online. Reply STOP to opt out"
- **Audience:** 874 can receive texts. Ireland's 20 were skipped (sender ID pending), so 854 were sent.
- **Route:** Twilio 659 (US 611, CA 48), Vonage 195 (GB 118, DE 52, NL 25).
- **Before the send:** $19.97 of the October budget used; after it, $39.51.

| Country | Sent | Delivered | Failed | Clicks | Opt-outs | Cost |
|---|---|---|---|---|---|---|
| United States | 611 | 601 | 10 | 158 | 4 | $6.72 |
| Canada | 48 | 47 | 1 | 13 | 0 | $0.53 |
| United Kingdom | 118 | 116 | 2 | 34 | 1 | $5.31 |
| Germany | 52 | 51 | 1 | 17 | 1 | $4.58 |
| Netherlands | 25 | 25 | 0 | 10 | 0 | $2.40 |
| **Total** | **854** | **840 (98.4%)** | **14** | **232 (27.6%)** | **6 (0.7%)** | **$19.54** |

- **Skipped:** Ireland, 20: "ACME isn't approved in Ireland yet."
- **Failed by reason:**
  - unreachable, 9: Twilio 30003, Vonage "absent subscriber". Temporary, so not suppressed.
  - filtered by carriers, 3: Twilio 30007.
  - not in service, 2: Twilio 30005. Both numbers suppressed.
- **Clicks:** 232 by people. 41 link previews were counted as machine clicks.
- **Replies:** 6 STOP, 2 HELP, 11 other replies, such as "Is there parking?".

### SMS draft by Daniel Park: "Summit: doors open"
- Text: "Acme: Doors open at 8:30 tomorrow. You’ll find badges at the north entrance." With the opt-out text, 98 characters.
- The curly apostrophe switches it to UCS-2: 2 segments. Replacing it brings it back to 1.
- Daniel can't send it: "Request approval", for Marcus Chen or Linh Nguyen.

### Automation: "Acme Summit reminders"
- Live since 9 Oct, built by Sofia Rossi. 1,240 contacts waiting.
- **Steps:**
  1. Trigger: date field `summit_date` (22 Oct 2026), 1 day before, at 09:00 America/Chicago: the same moment for every registrant.
  2. Send email "See you tomorrow at Acme Summit".
  3. Branch: has SMS consent for Acme Summit 2026.
  4. Yes: send SMS "Acme: See you tomorrow! Doors open 8:30 at Lakeside Hall. Livestream: go.acme.io/k3P9xq Reply STOP to opt out" (109 characters, 1 segment). Estimated $19.60 when it runs on 21 Oct. Where it's still before 09:00 local time, as on the US west coast, the text waits for the window.
  5. Wait 23 hours 45 minutes, to 22 Oct 08:45 America/Chicago.
  6. Send push "Acme Summit starts in 15 minutes", to linked subscriptions. Outside quiet hours it's skipped, because it's only useful on time.
  7. Send WhatsApp: the template `summit_livestream`, to contacts with WhatsApp consent.
  8. Exit.

### Web push
- **Keys:** created 3 Oct 2026 by Marcus Chen. Contact: mailto:support@acme.io.
- **Sites:**
  - news.acme.io, the publication site: the service worker is built in. 2,946 subscribers.
  - acme.io: snippet installed 5 Oct. The service worker at https://acme.io/rc-push-sw.js, version 1, was found at the 15 Oct 08:00 check. 1,318 subscribers.
- **4,264 subscribers by browser:** Chrome desktop 1,866, Chrome on Android 544, Edge 512, Firefox 401, Safari on macOS 845, Safari on iPhone and iPad (Home Screen) 96.
- **Linked to contacts:** 2,118.
- **Frequency cap:** 2 a day.
- **Prompt on news.acme.io:**
  - A slide-in bar: "Get new posts as notifications" / "One notification when a new post is out. Never more than one a day." / Allow / Not now.
  - Shown after 30 seconds, or after 50% of a post. Asks again after 14 days.
  - On iPhone, it shows how to add the site to the Home Screen.
  - Last 30 days: shown 18,240 times; Allow 4,980 (27.3%), of whom 4,412 granted the browser's permission (88.6%); Not now 9,104; closed 4,156.
- **Prompt on acme.io:** a bell button at the bottom right.
- **Push campaign "New post: Feature flags at scale":** sent by Sofia Rossi on 14 Oct at 09:05 to news.acme.io's 2,931 subscribers.
  - Accepted by the push services: 2,912. Expired and removed: 19.
  - Shown: 2,804 (96.3%). Clicks: 421 (14.5%). Unsubscribed within 24 hours: 7.
- **Scheduled:** "Acme Summit starts in 15 minutes", from the automation, on 22 Oct at 08:45 America/Chicago, to 612 linked subscriptions.

### WhatsApp
- **Account:** Acme Inc (WhatsApp Business Account …4410), +1 312 555 0110, display name "Acme" (approved). Quality: High. Messaging limit: 1,000 people a day, as Meta reports it. Connected 8 Oct 2026 by Marcus Chen.
- **Templates,** synced 15 Oct 08:00:

| Template | Category | Language | Status |
|---|---|---|---|
| summit_day_before | Utility | English (US) | Approved · {{1}} first name, {{2}} doors-open time |
| summit_livestream | Utility | English (US) | Approved · {{1}} livestream link |
| summit_venue_change | Utility | English (US) | Pending |
| digest_highlights | Marketing | English (US) | Rejected: "Add sample values for every variable." |

- **WhatsApp consent:** 214 registrants. Estimated cost of one utility template to all of them: $0.86, from Meta's rate card of 1 Oct 2026.
- **Window:** Sam Rivera wrote "Is there parking at the venue?" on 14 Oct at 18:02, so free-form replies are open until 15 Oct 18:02.

### People at Acme
- **Linh Nguyen** (Owner) set the SMS budget.
- **Marcus Chen** (Admin) connected Twilio, Vonage and WhatsApp, created the push keys, and set routes and rules.
- **Sofia Rossi** (Editor) wrote and sent the Summit texts and the push campaign, and built the reminders automation.
- **Daniel Park** (Content Writer) drafts texts and requests approval.
- **Aisha Bello** (Viewer) sees SMS reports and costs read-only, and never sees credentials.

### Agency: Northwind Studio
- **Instance:** app.northwind.studio, named "Northwind Studio". Agency mode turned on by Nadia Okafor on 1 Aug 2026.
- Northwind Studio has no connection with Acme's instance. "Acme Newsletter" isn't one of its clients.

**Staff**

| Person | Level | Clients and role |
|---|---|---|
| Nadia Okafor | Agency manager, instance admin | Every client · Owner |
| Tomás Herrera | Agency manager | Every client · Admin |
| Mei Lin | Account manager | Contoso Coffee, Fabrikam Outdoors, Coho Winery, Wide World Importers · Admin |
| Ella Fischer | Account manager | Tailspin Toys, Alpine Ski House, Litware Books · Admin |
| Ravi Menon | Account manager (designer) | Every client · Editor |

**Clients** (on 15 Oct 2026)

| Client | Contacts | Health | Next send | Approvals waiting | Incidents, 7 days | Account manager |
|---|---|---|---|---|---|---|
| Contoso Coffee | 64,210 | 94 | Holiday blend pre-order · 3 Nov 08:00 America/Los_Angeles · 58,940 | 1 · Jane Morales | – | Mei Lin |
| Fabrikam Outdoors | 182,406 | 71 | Weekend gear sale, by email and text · 16 Oct 09:00 · 141,220 emails, 12,480 texts | – | Auto-paused 13 Oct: Yahoo deferrals | Mei Lin |
| Tailspin Toys | 41,862 | 88 | – (Holiday catalog is a draft) | – | – | Ella Fischer |
| Alpine Ski House | 23,115 | 90 | Season pass early bird · 15 Oct 17:00 · 19,804 | – | – | Ella Fischer |
| Coho Winery | 12,940 | 97 | Wine club: November box · 19 Oct 10:00 · 3,412 | 1 · Daniela Ortiz | – | Mei Lin |
| Litware Books | 8,320 | 62 | – | – | Mailgun down 12 Oct 14:10–14:52; Automations failed over to Postmark | Ella Fischer |
| Wide World Importers | – | Setting up | – | – | – | Mei Lin |
| Northwind Studio (agency) | 3,410 | 99 | Studio notes #18 · 22 Oct | – | – | Nadia Okafor |

- **The overview's summary:** 7 clients. 2 need attention: Fabrikam Outdoors and Litware Books. 2 approvals waiting. 4 sends in the next 7 days.
- **Incidents:**
  - Fabrikam Outdoors: "Fall clearance" auto-paused on 13 Oct at 07:42, when Yahoo deferrals reached 4.1% (limit 3%). Mei Lin resumed it at 08:30 at a lower rate.
  - Litware Books: Mailgun was down on 12 Oct, 14:10–14:52. Automations failed over to Postmark on their own; no Broadcast was sending.

**Client people**
- **Contoso Coffee:**
  - Jane Morales (jane.morales@contoso.example), Client approver, signs in with a passkey.
  - Owen Price (owen.price@contoso.example), Client viewer, invited by Mei Lin on 13 Oct; the invitation expires on 20 Oct.
- **Coho Winery:** Daniela Ortiz, Client approver.
- **Fabrikam Outdoors:** Rick Alvarez, Client approver, and Hana Sato, Client viewer.
- **Every client workspace's approval policy** has "Client sign-off" on.

**Approval waiting: "Holiday blend pre-order"** (Contoso Coffee)
- By Mei Lin, requested on 14 Oct at 16:20, with the note: "Prices are final. The photo is the new bag."
- 58,940 recipients from the Coffee Club and Newsletter lists, after 1,212 suppressed and 74 role addresses were left out.
- Sent through Contoso's own Amazon SES account in us-west-2, from "Contoso Coffee <hello@mail.contoso.example>". Estimated cost $5.89.
- Send Readiness: Ready, 31 checks passed.
- Scheduled for 3 Nov 2026 at 08:00 America/Los_Angeles. Rule matched: Client sign-off.
- Jane Morales opens it on her phone on 15 Oct at 11:52 and approves it at 11:54. The overview's sample is from 10:00 that morning, while it's still waiting.

**Workspace templates**

| Template | What it sets up | Used by |
|---|---|---|
| Retail starter (Shopify) | 3 lists, 4 automations, 14 saved sections, 6 campaign templates, approval policy with client sign-off | Contoso Coffee, Fabrikam Outdoors, Tailspin Toys, Wide World Importers |
| Hospitality starter | 2 lists, 3 automations (welcome, booking reminder by SMS, win-back), 9 saved sections | Alpine Ski House, Coho Winery |
| Publisher starter | 2 lists, 2 automations, 5 saved sections, approval policy with client sign-off | Litware Books |

- **Template applied:** Tomás Herrera created "Wide World Importers" on 15 Oct at 09:40 from "Retail starter (Shopify)".
  - Added: 3 empty lists (Newsletter, Customers, VIP), 4 draft automations (Welcome series, Abandoned cart, Post-purchase thank-you, Win-back), 14 saved sections, 6 campaign templates, the approval policy, the client roles.
  - Not copied: providers, domains, contacts, keys, integrations.
  - Next steps: connect a sending provider, verify a domain, invite the client, connect Shopify.

**Shared library:** 41 items: 22 saved sections, 9 campaign templates, 5 automation templates, 3 SMS texts and 2 transactional templates.
- "Holiday hero (snow)": a saved section, v3, updated by Ravi Menon on 12 Oct, used in 4 clients. Contoso Coffee's copy is v2 and shows "Library has a newer version".

**Usage, September 2026**

| Client | Emails | SMS segments | Push | Provider cost | Rebill rate | To rebill |
|---|---|---|---|---|---|---|
| Contoso Coffee | 846,210 | – | – | $84.62 (SES) | Cost + 20%, plus $150 a month | $251.54 |
| Fabrikam Outdoors | 2,214,880 | 38,412 | – | $633.59 (SES $221.49, Twilio $412.10) | Cost + 15%, plus $300 a month | $1,028.63 |
| Tailspin Toys | 402,118 | – | 12,040 | $40.21 (SES) | $0.25 per 1,000 emails, plus $100 a month | $200.53 |
| Alpine Ski House | 96,440 | 4,210 | – | $55.95 (SES $9.64, Twilio $46.31) | Cost + 20%, plus $100 a month | $167.14 |
| Coho Winery | 41,320 | – | – | $15.00 (Postmark plan) | $120 a month, flat | $120.00 |
| Litware Books | 33,960 | – | 2,880 | $35.00 (Mailgun plan) | Cost + 25%, plus $80 a month | $123.75 |
| Northwind Studio (agency) | 6,820 | – | – | $0.68 (SES) | Not billed | – |
| **Total** | **3,641,748** | **42,622** | **14,920** | **$865.05** | | **$1,891.59** |

**Search across clients**
- Mei Lin searches "holiday" and gets results from her clients and the library:
  - Contoso Coffee: "Holiday blend pre-order", campaign, waiting for approval
  - Fabrikam Outdoors: "Holiday gift guide", campaign, draft
  - Coho Winery: "Holiday shipping banner", saved section
  - Library: "Holiday hero (snow)", saved section, v3
- Tomás Herrera also sees Tailspin Toys: "Holiday catalog", a draft campaign, and "Holiday countdown", a paused automation.
- Tomás searches the address j.doe@example.com: it's a contact at Contoso Coffee and Fabrikam Outdoors. "This search is recorded in the audit log."

---

## 5. New screens

- Sending providers gets the channel switch: Email · SMS · Push · WhatsApp.
- Campaigns › New campaign asks for the channel.
- The Agency area has Overview · Staff · Library · Usage.

### 5.1 Sending providers › SMS › Providers
- **Header:** the channel switch on SMS, "Both providers checked 1 min ago", "Add SMS provider".
- **Summary tiles:** texts sent this month, delivered rate, October spend against the budget (a meter with the 80% mark), and the SMS suppression list's count.
- **A card per provider:**
  - status
  - account
  - senders and the countries they serve
  - delivery-receipt freshness, error rate, spend this month
  - capability badges
  - links to Senders, Events and Settings
- **Senders:** the table from 3.1. A row opens its registration panel:
  - **10DLC:** brand and campaign status, the helper's text with copy buttons, "Open in Twilio".
  - **Toll-free:** the verification details.
  - **Sender ID:** status per country.

States to show:
- the sample
- the 10DLC campaign's panel, pending, with the helper
- a sender ID pending in Ireland
- a rejected registration: "Twilio rejected 'Acme Summit attendee updates': opt-in proof not reachable. Publish news.acme.io/sms-terms and the registration page, then resubmit in Twilio."
- tablet and phone: the cards stacked, the senders table scrolling inside its card

### 5.2 Sending providers › SMS › Routing & rules
One page, in sections:
- **Routes:** a table of countries, provider and sender, and fallback, with "Add route", and the "every other country" row. Then the route tester.
- **Quiet hours:** the window on a 24-hour bar, the time zone rule, and the number of contacts whose time zone is assumed.
- **Keywords and replies:** the opt-out, opt-in and help words; the replies with their segment counts; who answers.
- **Message rules:** brand prefix, opt-out text, the opt-in confirmation (and "Ask them to reply YES first"), the short-link domain with its status.
- **Budget and guards:** the monthly budget, the alert level, and the auto-pause thresholds.
- **Test numbers.**

States to show:
- the sample
- a route whose sender isn't registered: "ACME isn't approved in Ireland yet. Texts to Ireland are skipped."
- a quiet-hours window outside 08:00–21:00: "Texts can't go out after 21:00 local time."
- a help reply over one segment: a warning with its count
- the short-link domain not verified: "Add a CNAME for go.acme.io pointing to reputecast.acme.io", with the check
- saved, and read-only (Aisha Bello)

### 5.3 SMS composer
- **Main board:** "Summit: 8 days to go", before it was sent.
- **Header:**
  - Campaigns › Summit: 8 days to go, an SMS badge, Draft
  - audience "Acme Summit 2026 · 874 can receive texts"
  - the route line: "US and Canada via Twilio +1 888 555 0142 · UK, Germany, Netherlands via Vonage ACME · Ireland skipped (pending)"
  - Send test, Review & send
- **Left:** the message editor, with the locked prefix and opt-out text, the counter and its segment bar, the link list ("acme.io/summit/agenda → go.acme.io/xxxxxx, 17 characters"), Personalize and Insert link.
- **Middle:** the phone preview, with the sender, "Preview as Sam Rivera · US" and the country picker.
- **Right:** the cost by country, the total ($19.54) and the budget ($19.97 → $39.51 of $100).

States to show:
- Daniel Park's draft with the curly apostrophe: the UCS-2 warning and the Replace fix, then back to 1 segment
- personalization: "Length depends on first name: 1 segment for 96% of recipients, 2 for 4%"
- a public shortener pasted: "bit.ly links are filtered by carriers. Use the original address; ReputeCast shortens it on go.acme.io."
- over 3 segments
- the preview as a UK recipient (sender ACME)
- an empty audience: "Nobody on Acme Summit 2026 can receive texts yet. Add a phone field and SMS consent to the registration form."
- autosave saved, and read-only
- tablet and phone

### 5.4 Review & send for SMS
- **Verdict:** "Ready with warnings".
  - Warn: "20 recipients in Ireland are skipped: ACME is pending there." Fixes: "Skip them" (chosen) or "Wait for approval".
  - Warn: "12 recipients have an assumed time zone, from their phone number."
  - Pass: consent (359 people on the list have no SMS consent and get nothing), suppressions (7 numbers left out), registration, opt-out text, sender named, links, quiet hours, length, budget.
- **Audience:** 1,240 on the list → 874 can receive texts → 854 to send.
- **Route breakdown:** Twilio 659, Vonage 195. Cost $19.54. Duration: Twilio's 659 texts take under 4 minutes in all at 3 /s, spread over the time zones; Vonage's 195 take seconds.
- **Schedule:** "At 10:00 in each recipient's time zone, 14 Oct": from 08:00 UTC (Germany, Netherlands) to 17:00 UTC (US Pacific).
- **Auto-pause thresholds**, test send, and "Schedule texts to 854 recipients".

States to show:
- the sample
- blocked: the budget ("This send costs $19.54; $12.10 is left in October's budget. Raise the budget or send to fewer people.")
- blocked: no opt-out text
- quiet hours: "Now" would reach 118 UK recipients at 21:40, so they're held until 09:00
- approval required (Daniel Park), and waiting for approval
- the typed confirmation for a large send
- read-only

### 5.5 SMS report
- **Header:** "Summit: 8 days to go · sent 14 Oct, at 10:00 in each recipient's time zone".
- **Tiles:** sent 854, delivered 840 (98.4%), failed 14, clicks 232 (27.6%), opt-outs 6 (0.7%), cost $19.54.
- **Delivery over time:** one rise per time zone.
- **By country:** the table from section 4, with Ireland's skipped row.
- **Failures:** by reason, with the provider codes, and what ReputeCast did.
- **Clicks:** by link, with machine clicks shown apart.
- **Replies:** STOP, HELP and other replies, each linked to the contact.

States to show:
- the sample
- sending: progress by time zone, with Pause and Cancel
- auto-paused: "Carriers filtered 2.6% of texts (limit 2%). 412 held." Resume, Resume with changes, Cancel.
- a provider outage: "Twilio isn't answering since 10:12. 214 texts to the US and Canada are held." Retry, Cancel.
- loading, error
- phone

### 5.6 Connecting an SMS provider, and the SMS states
All in SmsStates:
- **The wizard's SMS path:** the catalog's SMS tab, Twilio's connection step, importing senders, the test message timeline, and the countries step.
- **Credentials rejected (Vonage):** "Vonage rejected the API key on 15 Oct at 09:10. Texts to the UK, Germany and the Netherlands are held."
- **Delivery receipts silent:** "No receipts from Twilio for 2 hours while texts were sent. Check the status callback in Twilio."
- **The SMS step waiting for quiet hours:** "38 contacts are waiting for 09:00 local time."
- **The budget reached:** "October's SMS budget is used up ($100.00). Campaigns can't start, and 38 texts in Acme Summit reminders are waiting." Raise the budget.
- **The Events tab:** a delivery receipt and a STOP, each raw next to its normalized event.
- **A STOP received:** "+1 773 ••• 0148 texted STOP: SMS consent withdrawn on Acme Summit 2026. Reply sent."
- **Audience › Suppressions filtered to SMS,** with the reasons; an opted-out number with no Remove action, and the reason why.
- **Sam Rivera's consent card** with the SMS column and its evidence.
- **Empty:** "Text your readers through your own Twilio, Vonage, Plivo, Telnyx or AWS account."
- **Read-only, loading, error.**

### 5.7 Public: text message terms
A built-in page at news.acme.io/sms-terms, generated from the SMS rules. Its text can be edited, and it's versioned: consent records point to the version that was live.
- **Program:** "Acme texts: reminders and updates for Acme Summit 2026."
- **Frequency:** "Up to 4 messages per event."
- **Cost:** "Message and data rates may apply."
- **Help:** "Text HELP to +1 888 555 0142, or email support@acme.io."
- **Opting out:** "Text STOP at any time. We'll confirm once and send nothing more."
- **Senders:** "+1 888 555 0142 in the US and Canada; ACME in the UK, Germany, the Netherlands and Ireland."
- **Privacy:** "We never sell or share your number for marketing." Linked to the privacy policy.
- **Carriers:** "Carriers aren't liable for delayed or undelivered messages."

At 720 px and on phones, in the site's theme.

### 5.8 SMS, push and WhatsApp steps in automations
- **Main board:** "Acme Summit reminders" on the flow canvas, with the SMS step selected.
- **The SMS step's panel:**
  - the message editor and counter
  - "Outside quiet hours: wait for the window"
  - the locked consent rule
  - "Skip if the contact got a text in the last 12 hours"
  - the estimated cost
  - the step's stats
- **The palette** adds Send SMS, Send push and Send WhatsApp.
- **The branch** "has SMS consent", with its Yes and No counts.

States to show (on the same board):
- the push step's panel: notification text and previews, and "Skipped when the contact has no linked subscription"
- the WhatsApp step's panel: the template, its variables mapped to fields, and the window rule for free text
- the publish dialog for Daniel Park: "This flow has an SMS step. Publishing needs sms:send. Request approval."
- a step skipped for a contact without consent, on the step's stats

### 5.9 Sending providers › Push › Setup
- **Keys:** the public key, when it was created and by whom, the contact address, "Rotate keys".
- **Sites:**
  - news.acme.io: "Built in · 2,946 subscribers".
  - acme.io: "Snippet · service worker found 15 Oct 08:00 · version 1 · 1,318 subscribers", with Check.
  - "Add a site": the snippet, the file to download, and the import line for sites that already have a service worker.
- **Subscribers:** by browser and by site; linked and anonymous.
- **Prompts:** the two prompts with their numbers, linked to the designer.
- **Frequency cap.**

States to show (in PushStates):
- push not set up: "Send notifications to readers' browsers. Nothing to sign up for: the keys are generated here."
- the service worker missing on acme.io: "https://acme.io/rc-push-sw.js answered 404. Upload the file to the site's root."
- an old service-worker version
- rotating the keys: the typed confirmation, "All 4,264 subscriptions end. Readers have to allow notifications again."
- read-only

### 5.10 Push prompt designer
The form builder's layout, with the "Push prompt" type.
- **Left:** style, text, when, where, frequency, the iPhone behavior.
- **Right:** a live preview on news.acme.io (desktop and phone), and the browser's own prompt, which appears only after "Allow".
- **Bottom:** the 30-day numbers.

States to show:
- the sample, slide-in bar
- the bell button on acme.io
- iPhone: "To get notifications, add The Acme newsletter to your Home Screen: tap Share, then Add to Home Screen."
- the browser's prompt after Allow, in Chrome and Safari
- a browser that blocked notifications: the prompt stays hidden, and the account page explains how to allow them again
- the site not added in Setup: "news.acme.io isn't a push site yet"

### 5.11 Push composer and review
- **Main board:** "New post: Feature flags at scale".
- **Left:** title, message, link, image, buttons, expiry, tag.
- **Right:** the previews (Chrome on Windows, Chrome on macOS, Safari on macOS, Chrome on Android, Safari on iPhone), each showing where the text is cut.
- **Header:** audience "news.acme.io · 2,931 subscribers", Send to my browser, Review & send.
- **Review & send for push:** the checks from 3.8, the schedule, and "Send to 2,931 subscribers".

States to show (in PushStates):
- title cut on Windows: "Chrome on Windows shows about 40 characters of the title"
- the frequency cap: "126 subscribers already got 2 notifications today. They'll be skipped."
- quiet hours holding subscribers in Europe
- approval required (Daniel Park)
- phone

### 5.12 Push report
In PushStates:
- **Tiles:** sent 2,912, shown 2,804 (96.3%), clicks 421 (14.5%), unsubscribed within 24 hours 7, expired and removed 19.
- **By browser and site.**
- **"Shown" note:** reported by the service worker; some browsers report it late or not at all.
- **Loading, error.**

### 5.13 Sending providers › WhatsApp
One board, WhatsAppChannel:
- **Account:** the card from 3.9, "Send a test message", Disconnect.
- **Templates:** the table from section 4, "Sync now", "Open WhatsApp Manager".
- **A WhatsApp campaign:** the template picker, variables mapped to fields with fallbacks, a chat-bubble preview, the audience with WhatsApp consent, and the cost.
- **The customer service window** on Sam Rivera's profile.

States to show:
- not connected
- waiting for Meta's verification request
- a rejected template, with Meta's reason
- quality dropped to Medium, and the messaging limit lower than the audience: the check says how many people get the message today and how many the next day
- the access token expired: Reconnect
- read-only

### 5.14 Agency overview
- **Shell:** the switcher shows "Northwind Studio · Agency". The sidebar: Overview, Staff, Library, Usage.
- **Header:** "7 clients · 2 need attention", "Search all clients", "New client".
- **Cards:** approvals waiting 2, incidents in the last 7 days 2, sends in the next 7 days 4.
- **Clients table:** the columns from section 4, each row opening that client's workspace. Filters: All, Needs attention, My clients.

States to show:
- the sample, as Tomás Herrera
- My clients, as Mei Lin: 4 clients
- Wide World Importers setting up: "0 of 4 setup steps"
- no clients yet: "Create your first client workspace. Start from a template to set it up in minutes."
- loading: each row on its own; a client that fails to load says so without hiding the others
- tablet and phone

### 5.15 Staff and clients
In AgencyStates:
- **Staff:** the table from section 4, "Add staff" (from the instance's users, or by invitation), and the assignment matrix (clients × staff, a role in each cell).
- **New client:** the dialog from 3.10, with the template's preview of what it adds.
- **Template applied:** the result for Wide World Importers: what was added, what wasn't copied, and the next steps.
- **Client details** in a client's Settings › General: "Managed by Northwind Studio · your contact: Mei Lin", as client people see it.
- **Client invited:** Owen Price, pending, with Resend and Copy link. His invitation page carries the Northwind Studio logo: "Mei Lin invited you to Contoso Coffee as Client viewer. You'll see campaigns and reports."
- **Owen Price as Client viewer:** Campaigns and Reports read-only; Audience says "Contacts aren't part of the Client viewer role. Ask Mei Lin."
- **Search across clients:** Mei Lin's "holiday" results grouped by client, and Tomás Herrera's contact search with the audit note.
- **Instance admin › Instance & updates › Agency:** agency mode, the agency workspace, the default template.
- **Escalation guard:** an account manager trying to assign Admin where they're Editor.

### 5.16 Library
- **Workspace templates:** a card per template, with what it sets up, the clients using it, "Apply to a client" and "Edit".
- **Shared library:** items by kind, with a preview, version, last change, the clients using a copy, and "Copy to a client".
- **An item's detail:** its versions, the fields it needs, and the copies that are behind.

States to show (on the same board):
- "Holiday hero (snow)", with Contoso Coffee's copy behind
- applying a template to an existing client: the list of what will be added, and the clashes that get renamed
- the composer's template gallery in a client workspace, with the "Northwind library" tab
- read-only for staff without agency:library

### 5.17 Usage
- **Header:** month picker (September 2026), "Export CSV".
- **Table:** the usage table from section 4, with totals.
- **A client's row** opens its detail: emails by stream, SMS by country, push, cost by provider, and the rate.
- **Rates:** "Edit rate" per client: cost plus a percentage, per 1,000 emails, per SMS segment, a monthly fee, a flat amount.
- **Note:** "Costs come from each provider's pricing or the price set in the workspace. ReputeCast doesn't send invoices."

States to show (on the same board):
- the current month to date, marked as partial
- a client without a rate: "Not billed"
- a provider without pricing: "Mailgun's cost is the $35.00 monthly plan entered in Litware Books"
- read-only for staff without agency:usage

### 5.18 A client approving a send
The approval review as Jane Morales sees it, in Contoso Coffee's workspace. Its sidebar only shows what her role allows.
- **Header:** "Holiday blend pre-order · waiting for your approval · requested by Mei Lin, Northwind Studio".
- **Preview first:** the email, with "Send me a test".
- **What you're approving:** 58,940 people on the Coffee Club and Newsletter lists, on 3 Nov at 08:00 Pacific time, from "Contoso Coffee <hello@mail.contoso.example>".
- **Readiness:** "Ready · all 31 checks passed".
- **Cost:** "Sent through Contoso's Amazon SES account · estimated $5.89".
- **Mei Lin's note,** the comment thread, then Approve, Request changes, Reject.

States to show:
- the sample on the phone (the primary board for this screen), and on desktop
- approved: "Approved by Jane Morales, 11:54. Mei Lin has been told."
- request changes, with a comment
- invalidated after Mei Lin changed the subject, with the diff
- the approval email Jane received

---

## 6. Changes to existing screens

- **App shell:**
  - The workspace switcher, for agency staff: "Northwind Studio · Agency" at the top, then the clients with a health dot, then the agency's own workspace. A search field once there are more than 8 workspaces.
  - The phone "More" sheet lists the Agency for staff.
- **Overview:** the provider section adds the SMS providers, with October's spend against the budget.
- **Campaigns list:**
  - Channel tabs: All · Email · SMS · Push · WhatsApp.
  - A channel icon on each row.
  - With SMS chosen: delivered, clicks, opt-outs and cost columns.
  - "New campaign" asks for the channel.
- **Approval inbox and review:** a channel icon on each request. SMS, push and WhatsApp previews in the review.
- **Settings › Approvals:**
  - The cost rule for SMS and WhatsApp.
  - "Client sign-off" in client workspaces.
  - "The author lacks the send permission" covers sms:send, push:send and whatsapp:send.
- **Sending providers** (Providers, Routing & failover and the other email tabs): the channel switch in the header. Email content is unchanged.
- **Connect a provider › Catalog:** channel tabs, and the SMS providers with their badges.
- **Settings › Providers:** the SMS, push and WhatsApp providers in the summary, and the SMS budget.
- **Settings › Senders:** the short-link domain go.acme.io, with its record and status.
- **Contacts:** a phone column; filters for has phone, phone country, SMS consent, WhatsApp consent and push subscribed.
- **Contact profile:**
  - the phone with its country, line type and time zone
  - consent per list with Email, SMS and WhatsApp columns and their evidence
  - a push subscriptions card (browser, site, since, Remove)
  - the WhatsApp window
  - timeline events: texts, keywords, replies, notifications
- **Segment builder:** conditions for phone, phone country, SMS consent per list, texts received or clicked, push subscribed (site, browser), WhatsApp consent.
- **Suppressions:** the Email · SMS filter, and the SMS reasons and sync.
- **Migration importer › Map fields and Consent:** the phone column, and SMS consent with its evidence columns.
- **Form builder:**
  - A phone field and the SMS and WhatsApp consent checkboxes, with their default texts.
  - The new form type "Push prompt" (5.10).
- **Public subscribe page:** the phone field and the unchecked SMS consent.
- **Public confirmation page:** "We've also texted +1 312 ••• 0123 to confirm."
- **Public preference center:** a Texts switch per list, the phone number, and "Turn off notifications in this browser".
- **Phase 7 boards:**
  - Pages › Built-in pages adds "Text message terms".
  - The account page adds a Notifications section.
- **Automations:**
  - The palette adds Send SMS, Send push and Send WhatsApp; new branch conditions.
  - The automations list shows channel icons and the SMS cost for 30 days.
  - Publish checks: steps that need sms:send, push:send or whatsapp:send, quiet hours, and sender registration.
- **Roles & permissions:** the three channel permissions (3.11), and the Client approver and Client viewer roles in client workspaces.
- **Team, member page and invite dialog:**
  - Assignments from the agency, marked "via Northwind Studio".
  - The client roles in the role picker.
  - The client invitation with the agency's logo.
- **Accept invitation:** the client variant, which explains the role in one line.
- **Settings › General** in client workspaces: "Managed by Northwind Studio", and "Save as workspace template" for staff.
- **Instance admin:**
  - Instance & updates: the Agency section.
  - Users: an "Agency staff" column with each person's level.
- **API keys:** the new permissions in the picker.
- **Webhooks:** the SMS event group, and push.clicked and whatsapp.* events.
- **Notifications:** new kinds:
  - the SMS budget at 80% and at 100%
  - a sender's registration approved or rejected
  - an SMS provider down, or its receipts silent
  - an SMS campaign auto-paused
  - a push service worker missing
  - a WhatsApp template rejected, or the quality rating dropped
  - for staff: incidents and approvals in their clients
- **Audit log:** the actions from 3.12, and filter categories SMS, Push, WhatsApp and Agency, in the workspace and instance views.
- **Settings › MCP server:** drafting SMS, push and WhatsApp campaigns and reading costs in the agents' permission matrix. Sending stays a proposal.
- **DSCoverage and `docs/DESIGN_INVENTORY.md`:** the new boards, and the conflicts in section 8.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens and badge colors.

- **Channel icons:** email (envelope), SMS (speech bubble), push (bell), WhatsApp (a chat bubble with a handset). The provider's name is always written next to the icon; no brand logos in the chrome.
- **SMS bubble preview:** a neutral phone frame, the sender line, the bubble with the locked parts in a lighter tone and the short link underlined.
- **Segment counter:** characters, segments and encoding, with a bar marked at each segment boundary, and the encoding chip that highlights the character responsible.
- **Push previews:** Chrome on Windows, Chrome on macOS, Safari on macOS, Chrome on Android, Safari on iPhone, each showing where the text is cut.
- **Registration badges:**
  - Verified and Approved: ok
  - Pending: warn
  - Rejected: bad
  - Not registered: neutral
  - Allowed without registration: ok, with that wording
  - Not supported: neutral, with the reason
- **Country chips:** the ISO code and the name ("GB United Kingdom"). No flags.
- **Quiet-hours bar:** a 24-hour track with the window shaded, and a marker for "now" in a chosen time zone.
- **Cost table and budget meter:** cost per country with the price per segment; the month's spend against the budget, with the alert mark at 80% and this send's share shown apart.
- **Consent per channel:** a per-list row with Email, SMS and WhatsApp badges:
  - Subscribed: ok
  - Opted out: bad, with the keyword or the way it happened
  - Not subscribed: neutral
- **Agency:**
  - a client row with initials, health score, next send and attention badges
  - the "Managed by Northwind Studio" banner
  - client role badges: Client approver in info, Client viewer in neutral
  - the usage table with its "To rebill" column
  - search results grouped by client

---

## 8. Conflicts to note

1. **Sending providers by channel.** The plan puts SMS and push providers under Sending providers and keeps its name. This brief adds a channel switch with tabs per channel; the email tabs don't change. The push prompt designer lives in Forms & pages, with the other forms.
2. **SMS consent per list.** The plan asks for SMS consent separate from email consent. This brief keeps it per list, as email consent is, so a text consent for the Summit doesn't cover other lists. That adds a channel dimension to list memberships and consent events.
3. **Two permissions to send a text.** campaigns:send keeps its meaning for every channel; sms:send, push:send and whatsapp:send are needed on top. Built-in Editors get them; custom roles can leave them out.
4. **Message log vs audit log.** Each text, notification and WhatsApp message stays in its own log for 90 days, not in the audit log, as in Phase 1. The owner needs to confirm this.
5. **Anonymous push subscribers.** Push subscribers aren't contacts until they're linked. They have consent records but no profile, and automations reach only linked subscriptions.
6. **The Agency area is a new top-level area,** like Instance admin, with instance-level roles (Agency manager, Account manager). Until now the only instance-level role was instance admin. No Settings tab is added.
7. **One agency per instance.** The plan says client workspaces are grouped under an agency; this brief puts one agency on each instance, the agency's own.
8. **Sample data.** Acme Newsletter lives on Acme's own instance (reputecast.acme.io), so it isn't one of Northwind Studio's clients. The agency's sample uses other companies, so there's only one workspace called Acme Newsletter.
9. **WhatsApp templates are created in Meta's WhatsApp Manager.** ReputeCast syncs them and their status only.
10. **Registration status reads** run in the existing provider health job. They're about ReputeCast's own senders, not continuous host monitoring, so they stay on ReputeCast's side of the HostRepute line.
11. **Quiet hours have hard limits,** 08:00–21:00 local. Some countries are stricter; the window can be narrowed, never widened.
12. **Contact totals change:** the Summit adds 338 contacts, so this phase's boards read 49,107.
13. **Text message terms page:** listed in Phase 7's built-in pages. If Phase 7 isn't built, it lives with the preference center under Forms & pages.

---

## 9. Boards

New boards for the canvas page "P8 SMS, push & agencies". Boards are 1440 px wide, except the public page at 720 px. Tablet boards are 834 px and phone boards 390 px, named `<Board>Tablet.dc.html` and `<Board>Phone.dc.html`.

| # | Board | Canvas title | Implements | Width | Tablet | Phone |
|---|---|---|---|---|---|---|
| 1 | `SmsProviders.dc.html` | Sending providers · SMS providers and senders | 5.1 | 1440 | Yes | Yes |
| 2 | `SmsRouting.dc.html` | Sending providers · SMS routing and rules | 5.2 | 1440 | – | – |
| 3 | `SmsComposer.dc.html` | SMS composer · Summit: 8 days to go | 5.3 | 1440 | Yes | Yes |
| 4 | `SmsReview.dc.html` | Review & send · SMS | 5.4 | 1440 | – | – |
| 5 | `SmsReport.dc.html` | SMS report · Summit: 8 days to go | 5.5 | 1440 | – | Yes |
| 6 | `SmsStates.dc.html` | SMS · connect, registration, keywords, suppressions, budget, outage, states | 5.1–5.6 | 1440 | – | – |
| 7 | `PublicSmsTerms.dc.html` | Public · text message terms | 5.7 | 720 | – | Yes |
| 8 | `AutomationChannels.dc.html` | Automations · SMS, push and WhatsApp steps | 5.8 | 1440 | – | – |
| 9 | `PushSetup.dc.html` | Sending providers · web push setup | 5.9 | 1440 | – | – |
| 10 | `PushPrompt.dc.html` | Signup forms · push prompt designer | 5.10 | 1440 | – | – |
| 11 | `PushComposer.dc.html` | Push composer · previews | 5.11 | 1440 | – | Yes |
| 12 | `PushStates.dc.html` | Push · review, report, keys, service worker, states | 5.9–5.12 | 1440 | – | – |
| 13 | `WhatsAppChannel.dc.html` | Sending providers · WhatsApp account, templates, campaign, states | 5.13 | 1440 | – | – |
| 14 | `AgencyOverview.dc.html` | Agency · overview | 5.14 | 1440 | Yes | Yes |
| 15 | `AgencyLibrary.dc.html` | Agency · workspace templates and shared library | 5.16 | 1440 | – | – |
| 16 | `AgencyUsage.dc.html` | Agency · usage per client | 5.17 | 1440 | – | – |
| 17 | `AgencyClientApproval.dc.html` | Client approver · reviewing a send (Jane Morales) | 5.18 | 1440 | – | Yes |
| 18 | `AgencyStates.dc.html` | Agency · staff, new client, client invited, client viewer, search, agency mode, states | 5.14–5.18 | 1440 | – | – |

That's 18 new boards: 17 at 1440 px and 1 public page at 720 px. Plus 3 tablet and 7 phone boards.

**Existing boards to update (section 6):**

| Board | Change |
|---|---|
| `ShellSwitcher.dc.html` | Agency entry, clients with health dots, search |
| `Main.dc.html` (+ tablet, phone) | SMS providers and budget in the provider section; the phone "More" sheet lists the Agency for staff |
| `Campaigns.dc.html` (+ tablet, phone) | Channel tabs, icons and columns; channel picker on New campaign |
| `ApprovalInbox.dc.html` (+ phone), `ApprovalReview.dc.html` (+ tablet, phone), `ApprovalPolicy.dc.html` | Channel icons and previews; the cost rule; Client sign-off |
| `Providers.dc.html` (+ tablet, phone), `ProvidersRouting.dc.html` (+ tablet, phone), `ProvidersOutage.dc.html`, `ProvidersWebhooksSilent.dc.html`, `ProvidersStates.dc.html`, `ProvidersWarmup.dc.html`, `ProvidersEvents.dc.html`, `ProvidersAdapters.dc.html`, `ProvidersBulkWarning.dc.html` | Channel switch in the header |
| `SetupCatalog.dc.html` (+ tablet, phone) | Channel tabs and SMS providers |
| `SettingsProviders.dc.html`, `SettingsSenders.dc.html` (+ tablet, phone) | SMS, push and WhatsApp in the summary; the short-link domain |
| `Contacts.dc.html` (+ tablet, phone), `ContactProfile.dc.html` (+ phone) | Phone, consent per channel, push subscriptions, WhatsApp window, timeline events |
| `SegmentBuilder.dc.html` (+ tablet) | Phone and channel conditions |
| `Suppressions.dc.html` | Email · SMS filter and SMS reasons |
| `MigrationFields.dc.html`, `MigrationConsent.dc.html` | Phone and SMS consent with evidence |
| `FormBuilder.dc.html` (+ tablet, phone), `FormsStates.dc.html` | Phone field, SMS and WhatsApp consent, the Push prompt type |
| `PublicSubscribe.dc.html` (+ phone), `PublicConfirm.dc.html`, `PublicPreferences.dc.html` (+ phone) | Phone and SMS consent; the confirmation text note; Texts and notifications switches |
| `AutomationFlow.dc.html` (+ tablet, phone), `AutomationsList.dc.html` (+ tablet, phone), `AutomationPublish.dc.html` | New steps and branches; channel icons and SMS cost; publish checks |
| `Roles.dc.html` (+ tablet, phone), `RoleEditor.dc.html` | Channel permissions; client roles |
| `SettingsTeam.dc.html` (+ tablet, phone), `TeamMember.dc.html`, `TeamInvite.dc.html`, `InviteStates.dc.html` | Agency assignments; client roles; the client invitation |
| `WorkspaceSettings.dc.html` (+ phone) | "Managed by Northwind Studio"; Save as workspace template |
| `InstanceSettings.dc.html`, `InstanceUsers.dc.html` (+ tablet, phone) | Agency section; Agency staff column |
| `SettingsKeys.dc.html` (+ tablet, phone), `SettingsAPI.dc.html` (+ phone), `SettingsMCP.dc.html` | New permissions; webhook event groups; MCP tools |
| `Notifications.dc.html` (+ tablet, phone) | New notification kinds |
| `SettingsAudit.dc.html`, `AuditInstance.dc.html` | New actions and filter categories |
| Phase 7 boards: `PagesList.dc.html` (+ tablet, phone), `PublicAccount.dc.html` (+ phone) | Text message terms in Built-in pages; Notifications on the account page |
| `DSCoverage.dc.html`, `DSPatterns.dc.html`, `DSComponents.dc.html` | New boards in the coverage matrix; section 7 patterns |
