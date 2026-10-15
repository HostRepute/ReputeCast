# ReputeCast — Add-on spec: know your audience (scoring, address verification, placement tests, growth sources)

Phase 4 of `reputecast-design-expansion-plan.md`. It builds on phases 1–3: transactional email (`reputecast-design-addon-transactional.md`), the Integrations hub of phase 2, and conditional content and polls (`reputecast-design-addon-create.md`).

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and phases 1–3 already exists.**
  - Add the screens in section 5 as a new canvas page, "P4 Audience insights & placement tests".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
  - Phase 2's Integrations hub (catalog and detail page) and phase 3's outbound-data notice are assumed. If they aren't drawn yet, draw the parts this brief touches first.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and phases 1–3's.
  - Desktop boards at 1440 px. Tablet (834) boards for 5.1, 5.6 and 5.8; phone (390) boards for 5.6 and 5.8.
- **Build order:**
  1. 5.1 Audience › Scoring, then the contact profile's score card (section 6)
  2. 5.6 Placement test result
  3. 5.4 Deliverability › Placement tests
  4. 5.5 Run a placement test, and 5.7 Seed mailboxes
  5. 5.8 Audience › Growth
  6. 5.3 Verify addresses, and 5.2 Verification services
  7. Then the rest of section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.** No limits on rules, seed mailboxes, tests or sources. Verification services charge the workspace's own account per check. ReputeCast shows the cost before every use and takes nothing.
2. **Interest isn't deliverability.**
   - Engagement tiers protect sender reputation: they decide who gets Broadcasts and who enters the sunset policy.
   - The score measures interest. It never stops or starts mail by itself; only automations the team builds on it do.
3. **Human signals only.** Scores and quality by source count human opens and real clicks. Machine opens and bot clicks never count.
4. **Addresses leave only on purpose.**
   - Verification sends email addresses to the service the workspace chose, after a notice that names it and says exactly what is sent.
   - Every use is audited.
   - ReputeCast never probes recipients' mail servers itself: that would hurt the sending IPs' reputation.
5. **One-off checks, not monitoring.**
   - A placement test runs when someone starts it, on the workspace's own seed mailboxes, for a campaign, template or provider.
   - No scheduled tests and no alerts about hosts: watching reputation over time is HostRepute's side of the line.
   - A test never blocks or changes a send.
6. **Small samples, honest claims.** Ten seeds are a sample, not an audience. Results read "8 of 10 seeds in the inbox", never a predicted inbox rate. Sources with too little data show "Too few to compare".
7. **Same access rules as the team add-on.**
   - Permission keys, the escalation guard, write-only secrets and audit entries.
   - MCP agents can read scores, growth and placement results. They never run tests or send addresses to a service.

---

## 2. Terms

- **Score:** a number from 0 to 100 per contact that measures recent interest, built from the scoring rules.
- **Scoring rule:** an event or a condition, the points it gives (or takes), and a cap.
- **Decay:** points lost each week after a stretch without activity.
- **Band:** Cold, Warm or Hot, set by two thresholds on the score.
- **Engagement tier:** Active, Cooling, At risk or Sunset, as today. Tiers and bands are separate.
- **Verification service:** a third-party service that checks whether an address can receive mail: ZeroBounce, NeverBounce, Kickbox, Bouncer, Emailable.
- **Verification result:** Valid, Invalid, Risky (catch-all, role, disposable, or the service's own reason) or Unknown, with the service's own code kept beside it.
- **Credits:** what the service charges per check, from the workspace's balance with that service.
- **Verification job:** one bulk use: a list, a segment, a tier, a selection or an import's new addresses.
- **Real-time check:** a signup form checking each address as it is submitted.
- **Seed mailbox:** a mailbox the workspace owns, used only to receive test copies.
- **Placement test:** one copy of an email sent to each seed mailbox through a route or provider, then found and classified.
- **Copy:** one test message to one seed through one provider.
- **Folder:** where a copy landed: Inbox, a tab (Promotions, Updates, Other…), Spam, or Not arrived.
- **Time to arrive:** from the hand-off to the provider to the mailbox's own received time.
- **Test token:** an ID in each copy's Message-ID and in an `X-ReputeCast-Test` header, used to find it.
- **Growth source:** how a contact first joined: a form, the subscribe page, an import, an API key, an integration, or a person.
- **Join:** a contact confirmed onto at least one list for the first time. **Departure:** a contact who unsubscribed from every list, hard-bounced, complained or was erased.
- **Cohort:** the contacts who joined in one month. **Retention:** the share of a cohort still subscribed after 30 and 90 days.
- **UTM parameters:** `utm_source`, `utm_medium`, `utm_campaign`, `utm_content` and `utm_term` on the address of the page where someone signed up.
- **Referring page:** the page a visitor came from, stored without its query string.

---

## 3. Requirements

### 3.1 Contact scoring
- **Where:** Audience › Scoring, a new tab after List health.
- **Rules:**
  - Events: human open (of any email, or of a list's emails), click (any link, or a link whose address contains some text), form submitted (a chosen form), API event (by name), soft bounce (not delivered after the retries), unsubscribed from a list.
  - Field conditions, such as "`plan` is team or business": the points count while the condition is true.
  - Points from −50 to +50 per rule.
  - A cap per rule: at most N points a day, a week or a month, or "once".
  - Each rule can be switched off without deleting it.
- **What counts:**
  - Campaigns and automation emails. Transactional messages (phase 1) never count: they aren't a sign of interest, and their recipients may not be contacts.
  - Human opens only. Machine opens and bot clicks never count.
  - The last 180 days of activity. Older events drop out.
- **Decay:** optional. After N weeks without a human open or click (default 4), the score drops by a share each week (default 10%), never below 0.
- **Maximum score:** 100.
- **Bands:** Cold, Warm and Hot, with two editable thresholds (defaults 20 and 50). The names are fixed so segments and triggers read the same everywhere.
- **Preview:** how the workspace's contacts spread across the score, in buckets of 10, coloured by band, with the count per band. While a rule is being edited, the new spread is drawn over the current one, with the moves: "1,108 Warm → Hot".
- **When scores change:**
  - Within a minute of an open, a click, a form submission or an API event.
  - Every night with the tier job (03:00 workspace time) for decay and field conditions.
  - After a rule change: every contact is recalculated in the background from the last 180 days. Progress shows on Scoring. Until it finishes, contacts keep their previous score, and segments and triggers use it.
- **Triggers after a rule change:** "Score crosses…" automations start on changes caused by activity or decay. For contacts who cross only because a rule changed, the save dialog asks: "Don't start them (recommended)" or "Start them for N contacts".
- **Score and tiers, in the product's words:** "Engagement tiers protect your sender reputation: they decide who gets Broadcasts and who enters the sunset policy. The score measures interest: use it to find your most interested readers and customers. A score never stops or starts mail by itself." This note sits on Scoring, on List health and beside the score on the contact profile.
- **Where the score appears:**
  - the contact profile: the score, its band, a 90-day line and the breakdown by rule (section 6)
  - Contacts: a Score column and a band filter
  - the segment builder: "Score is at least / at most / between" and "Score band is"
  - automations: the trigger "Score crosses…": rises to N or more, or falls below N; at most once per contact in N days
  - conditional content (phase 3): score and band in the Visibility conditions
  - the MCP server: score and band in contact search results
- **No rules:** contacts have no score ("—"). The empty state offers the suggested rules (section 4) or "Add a rule".

### 3.2 Address verification services
- **Where they're set up:** Settings › Integrations, the "Verification" group of the phase 2 hub.
  - Catalog: ZeroBounce, NeverBounce, Kickbox, Bouncer and Emailable. Each declares its capabilities, as sending adapters do: bulk lists, real-time checks, catch-all detection, credit balance, data region.
  - Connection: an API key (write-only). "Test connection" reads the credit balance and sends no address.
  - Price per check: entered from the workspace's plan, used for every estimate.
  - Defaults: one service for bulk and imports, one for signup forms. They can be the same.
  - Health: balance (read at each use and every hour), last used, last error.
  - Each service adapter passes a shared contract-test suite, like sending adapters.
- **Results:**
  - Each service's answers map to four results, and the original code is kept beside it, as bounce codes are: "Invalid · ZeroBounce: invalid / mailbox_not_found".
  - **Valid:** nothing changes except the result.
  - **Invalid:** the address goes on the shared suppression list, reason "Invalid address", source the service, for all mail. Phase 1 honours invalid addresses for transactional mail too. A "possible spam trap" answer uses the reason "Spam trap".
  - **Risky:** a tag says why: `verify:catch-all`, `verify:role`, `verify:disposable`, or the service's own reason. Nothing else changes; the team decides with segments and Send Readiness.
  - **Unknown:** tagged `verify:unknown`. It can be checked again after 72 hours.
  - The contact keeps the latest result, its date and the service.
- **Where they're used:**
  1. **Import preview** (Migration importer, step 4): an optional card, "Verify the new addresses with ZeroBounce", with the count, the cost and the balance. Results join the preview's groups: invalid ones are skipped, risky ones are imported and tagged. The import never needs it.
  2. **Signup forms:** "Check each address with Kickbox as people sign up", in the form's spam protection.
     - Invalid: the form asks to fix it: "This address doesn't seem to exist. Check it and try again."
     - Disposable: follows the form's "Reject disposable domains" setting.
     - Risky and unknown: accepted and tagged.
     - The service fails or takes more than 3 seconds: the signup goes through, and the address is checked once the service answers again.
  3. **Bulk:** from Contacts (a selection or a saved view), a list (List health › Health per list › ⋯), a segment or a tier. The dialog: service, what to check, "Skip addresses checked in the last 90 days", the cost, the notice. The job runs in the jobs tray. Cancelling keeps the results so far.
  4. **Before a re-engagement send:** Send Readiness's audience check warns when at least 10% of the audience is At risk or Sunset and hasn't been checked in 90 days. The fix: "Verify 5,302 addresses first (ZeroBounce, about $42.42)". List health offers the same for the At risk and Sunset tiers.
  5. **One contact:** "Verify now" on the contact profile.
- **The notice before every use** (phase 3's outbound-data notice): "5,302 email addresses will be sent to ZeroBounce (United States). Only the addresses: no names, fields or lists. What ZeroBounce keeps, and for how long, depends on your ZeroBounce account." With the cost, the balance after, and "Verify 5,302 addresses".
- **Pacing:** jobs follow the service's rate limits and retry 429 and 5xx answers for up to an hour, then stop with the results so far.
- **Send Readiness, audience quality:**
  - invalid addresses are already suppressed, and the passing line counts them
  - risky addresses: a Warn when they are 2% or more of the audience, with "Exclude N risky addresses"
  - inactive and unchecked: the Warn in use 4
- **Built-in checks stay** (syntax, MX, role, disposable list, typos, trap-like patterns), for free and on every import. A service adds a mailbox-level check on top.

### 3.3 Placement tests
- **Where:** Deliverability gets two tabs: Health (today's page, unchanged) and Placement tests.
- **Seed mailboxes the workspace owns:**
  - Gmail, Google Workspace, Outlook.com and Microsoft 365: OAuth or an IMAP app password.
  - Yahoo, iCloud, AOL, GMX and Web.de: an IMAP app password. The IMAP host and port are filled in.
  - OAuth uses the instance's own Google and Microsoft OAuth clients (Instance admin › Authentication, as for social sign-in). It asks for read-and-modify mail access, only when a seed is connected.
  - Google limits mail access for apps it hasn't verified: a gmail.com seed connected by OAuth may have to sign in again every 7 days. The add dialog says so and recommends an app password for gmail.com seeds.
  - Passwords and tokens are write-only.
  - A notice when adding one: "Use mailboxes that exist only for tests. ReputeCast only looks for messages that carry a test token, but it can read the whole mailbox."
  - Checked when added: sign-in, and the folders found. Statuses: Connected, Sign-in expired (an OAuth grant that ended), Sign-in failed (a password that was rejected), each with since when, and Not checked yet.
- **Folders, by mailbox provider** (the provider's own name in brackets when it differs):
  - Gmail and Google Workspace: Inbox · Primary, Promotions, Updates, Social or Forums tab; Spam.
  - Outlook.com and Microsoft 365: Inbox · Focused or Other; Spam (Junk Email).
  - Yahoo and AOL: Inbox; Spam (Bulk).
  - iCloud: Inbox; Spam (Junk).
  - GMX and Web.de: Inbox (Posteingang); Spam (Spamverdacht).
  - Gmail tabs are read over IMAP too, through Gmail's search extension. Outlook's Focused and Other need OAuth; over IMAP the copy shows as "Inbox".
- **Run a test:**
  - **Content:** a campaign (draft, scheduled or sent), a campaign template, a digest's next issue (phase 3), or a transactional template's published version (phase 1).
  - **Version:** for conditional content, the version with the most recipients, or another; the language variant (phase 2); the A/B variant.
  - **Personalization:** the fallbacks, or one of a transactional template's test-data sets. Never a real contact's data.
  - **Through:**
    - "The route as configured": each seed gets the provider the route would pick for it, as the route tester shows.
    - "One provider".
    - "Compare providers": one copy per provider for each seed, sent 2 minutes apart, alternating which provider goes first.
  - **From:** a verified sender of the stream.
  - **Seeds:** all connected seeds (default), or some.
  - The dialog ends with the copies and their cost: "20 copies · SES $0.001 · Postal $0".
- **The copies are real sends:**
  - The same headers as a real send. The List-Unsubscribe link opens a page that says "This is a placement test copy. Nobody was unsubscribed."
  - Link tracking stays on, so the tracking domain is part of the test. Opens and clicks of copies are never counted.
  - Copies never appear in campaign reports or contact timelines. They count toward a warming-up provider's daily cap.
- **Waiting:** results fill in live. After 15 minutes, a missing copy is "Not arrived", and the provider's events for it are shown (a deferral or a bounce). A copy that turns up within 60 minutes updates the result, marked "arrived late". "Stop waiting" ends it early.
- **Each copy:** folder, time to arrive, SPF, DKIM and DMARC from the mailbox's Authentication-Results header (with domains and selectors), the server that handed it over, and the provider's message ID.
- **Summary per provider:** copies per folder and the median time to arrive. Never a percentage headline.
- **Guidance, written by ReputeCast's own rules (not AI):** when half or more of a provider's copies land in spam, a card lists what to look at:
  - the authentication results in the headers
  - Send Readiness's content and link checks for this email
  - the same email through another provider
  - the provider's warm-up state, and Deliverability › Health
  - when HostRepute is connected, "Check 203.0.113.25 in HostRepute"
- **History:** every test, with filters (content, provider, started by). The trend is one stacked bar per test, in time order, per provider: Inbox, Tab, Spam, Not arrived. No line, no rate.
- **Deleting copies:** 24 hours after the test (default), when it finishes, or never. "Delete copies now" on each test.
- **Entry points:**
  - Review & send: an optional card. A test never changes the verdict.
  - The composer's Send test menu, including the template editor in transactional mode.
  - The warm-up planner, with the warming-up provider chosen.
- **Out of the picture:** no scheduled tests, no alerts, no managed seed network. A managed network, if HostRepute ever offers one, would show here as a second seed source; nothing is drawn for it now.

### 3.4 Growth sources
- **Where:** Audience › Growth, a new tab after Scoring.
- **Sources, recorded on their own:**
  - each signup form, with its type (embedded, popup, hosted)
  - the subscribe page
  - each import, by date and platform
  - each API key
  - each integration that creates contacts (phase 2), such as a Zapier Zap. Forms that the WordPress plugin places on a site count under their form; the site shows among the referring pages.
  - added by a person
  - later: referrals and recommendations (phase 5)
- **Attribution:**
  - A contact's source is how they first joined. Each list membership also keeps how that list was joined, so with one list selected, someone who added it in the preference center counts under "Preference center".
  - Imported contacts keep their original join date and source when the import carries them (the Brevo import did). Otherwise the import is their source and the import day their join date.
- **The page:**
  - Period: 30 days (default), 90 days, 12 months, or custom. List: all lists (contacts) or one list (its members).
  - A chart of joins above the axis and departures below it, by day, by source.
  - Imports are left out of the chart by default, because moving a list isn't growth: "Include imports" adds them as a marker on their day.
  - A table by source: joins, departures, net, and quality.
  - Departures by kind: unsubscribed from every list, hard-bounced, complained, erased.
- **UTM parameters and the referring page:**
  - Captured by embedded, popup and hosted forms, including those the WordPress plugin places (phase 2), and by the subscribe page.
  - Saved with the consent record, first touch only.
  - The referring page is stored without its query string.
  - Shown as a table of UTM combinations and a table of referring pages, each with joins and human opens.
- **Cohort retention:** monthly cohorts by join month, with the share still subscribed after 30 and 90 days, for all sources or one. A cohort that hasn't reached the mark shows "in progress" and the date it will.
- **Quality by source:** bounce rate, complaint rate and human-open rate over the period's sends.
  - A source is flagged when its bounce or complaint rate is more than twice the workspace's.
  - Under 100 delivered messages or 50 contacts, rates read "Too few to compare".
- **Contacts and segments:** filter and condition by source, UTM source, medium and campaign, and referring domain.
- **Export:** the tables as CSV, aggregate only, audited like report exports.

### 3.5 Permissions
New permissions:
- **Contacts:** verify addresses with a service (subscribers:verify) ⚑. It sends contacts' addresses out of the instance and spends credits, like an export.
- **Audience scoring:** edit scoring rules, decay and bands (scoring:edit).
- **Placement tests:**
  - run a placement test (placement:run). It sends copies through the workspace's providers, to its own seeds only.
  - add, reconnect and remove seed mailboxes (placement:manage) ⚑. Their passwords and tokens can read whole mailboxes.

Who gets what:
- **Built-in roles:**
  - Owner and Admin: all four.
  - Editor: scoring:edit and placement:run.
  - Viewer: none.
- **Content Writer**, the sample's custom role: none. Daniel Park sees scores and growth (subscribers:view) and placement results (deliverability:view).
- **API keys:** can hold any of them, under the escalation guard.
- **MCP agents:** can't hold subscribers:verify, placement:run or placement:manage. They read scores and growth with subscribers:view, and placement results with deliverability:view.

Existing keys used:

| Action | Permission |
|---|---|
| See scores, bands, Growth and verification results | subscribers:view |
| See placement tests and results | deliverability:view |
| See verification services, balances and uses | integrations:view (phase 2) |
| Connect, change or remove a verification service | integrations:manage ⚑ (phase 2) |
| Turn on real-time checks on a form | forms:edit and subscribers:verify ⚑ |
| Use "Score crosses…" in an automation | automations:edit |
| Use score, source or UTM in a segment | segments:edit |
| Export Growth tables | reports:view |

### 3.6 Audit
- **Audited:**
  - scoring rules created, changed, switched off or deleted; decay, cap or bands changed (before and after); a recalculation started and finished; the choice to start, or not start, automations after a rule change
  - a verification service connected, changed or removed, its key replaced, its price or default use changed
  - every use of a verification service: who, which service, what (import, list, segment, tier, selection or one contact), how many addresses and the estimated cost; then the result counts and the addresses suppressed; a job cancelled
  - real-time checks turned on or off on a form
  - a seed mailbox added, reconnected or removed
  - a placement test started (content, providers, seeds) or stopped
  - test copies deleted from seeds, by a person or by the system
  - Growth exports
- **Not in the audit log:** each real-time check on a form. The service's activity counts them by day and by form, as phase 1 keeps each send in the message log. See section 8.

### 3.7 Notifications
New kinds, in the notification center and by email, following each member's preferences:
- **Verification job finished, or stopped** (no credits, service down): the person who started it.
- **Verification credits low:** under 1,000 credits, or under the size of a job that is waiting. Members with integrations:manage.
- **Placement test finished:** the person who started it.
- **Seed mailbox sign-in failed:** members with placement:manage.
- **Score recalculation finished:** the person who changed the rules.

### 3.8 HostRepute
- A placement result's guidance offers "Check 203.0.113.25 in HostRepute" when HostRepute is connected, using the existing link to the monitored asset. When it isn't, the guidance points to Deliverability › Health and nothing more.
- Nothing from this phase is sent to HostRepute: no scores, verification results, seed mailboxes, placement results or growth data.
- A managed seed network is a possible HostRepute product. If it ever exists, it plugs into Placement tests as a second seed source. It isn't designed here.

### 3.9 Out of scope for this phase
- **Scoring:** several scoring models, predicted scores, company scores, scores from transactional mail.
- **Verification:** phone numbers, scheduled re-verification of whole lists, ReputeCast probing mail servers itself.
- **Placement tests:** scheduled or recurring tests, alerts, a managed seed network, spam-filter scoring of the content.
- **Growth:** multi-touch attribution, ad spend, referrals and recommendations (phase 5).

---

## 4. Sample data

### Today
- **Wednesday 15 Oct 2026**, 10:30 America/New_York. Workspace "Acme Newsletter" (48,213 contacts), instance reputecast.acme.io.

### Scoring rules
Set up by Sofia Rossi on Mon 13 Oct 2026.

| # | When | Points | Cap |
|---|---|---|---|
| 1 | Human open of any email | +2 | 6 a week |
| 2 | Click in any email | +5 | 20 a week |
| 3 | Click on a link containing acme.io/pricing | +15 | once a month |
| 4 | Submitted the form "Beta apply" | +20 | once |
| 5 | API event `trial_started` | +25 | once |
| 6 | Field `plan` is team or business | +10 | while true |
| 7 | Soft bounce, not delivered after the retries | −5 | – |
| 8 | Unsubscribed from a list | −20 | – |

- **Decay:** after 4 weeks without a human open or click, −10% a week.
- **Maximum:** 100. **Look-back:** 180 days.
- **Bands:** Cold 0–19, Warm 20–49, Hot 50–100.

### How 48,213 contacts spread
- **Bands:** Cold 29,190 (60.5%) · Warm 14,207 (29.5%) · Hot 4,816 (10.0%).
- **Buckets:** 0–9: 18,406 · 10–19: 10,784 · 20–29: 6,912 · 30–39: 4,401 · 40–49: 2,894 · 50–59: 1,966 · 60–69: 1,207 · 70–79: 806 · 80–89: 497 · 90–100: 340.
- **Bands by tier** (tiers and bands differ):

| Tier | Contacts | Hot | Warm | Cold |
|---|---|---|---|---|
| Active | 29,412 | 4,702 | 12,960 | 11,750 |
| Cooling | 9,884 | 114 | 1,198 | 8,572 |
| At risk | 5,302 | 0 | 49 | 5,253 |
| Sunset | 3,615 | 0 | 0 | 3,615 |

### Contacts with their scores
| Contact | Tier | Score | Band |
|---|---|---|---|
| Sam Rivera | Active | 52 | Hot |
| Priya Natarajan | Active | 78 | Hot |
| Diego Alvarez | Cooling | 12 | Cold |
| Hannah Berg | Active | 47 | Warm |
| Kenji Watanabe | Active | 18 | Cold · opens every week, never clicks |
| Lena Fischer | At risk | 0 | Cold |
| Omar Haddad | Active | 64 | Hot |
| Grace Okafor | Sunset | 0 | Cold |
| Tomasz Nowak | Cooling | 21 | Warm |
| Aisha Rahman | Active | 91 | Hot |

### Sam Rivera's score: 52, Hot
- Human opens: +22 (11 opens since he joined on 18 Sep).
- Clicks: +20 (4 clicks).
- Field `plan` is team: +10.
- Soft bounce on 23 Sep: 0, because the message was delivered after a retry.
- Decay: none; last open Tue 7 Oct.
- Last change: +5 on Thu 9 Oct, "Clicked 'Read the docs' in October Product Update".

### Editing a rule
- Sofia Rossi changes rule 3 from +15 to +25, Tue 14 Oct.
- Impact before saving: "1,312 contacts change band: 1,108 Warm → Hot, 204 Cold → Warm." Hot 4,816 → 5,924 · Warm 14,207 → 13,303 · Cold 29,190 → 28,986.
- "1,108 contacts would cross 50 because of this change. The live automation 'Sales-ready handoff' starts when a score rises to 50 or more." Choices: "Don't start it for them (recommended)" or "Start it for 1,108 contacts".
- **Recalculating:** "Recalculating 48,213 scores · 31,400 done · about 2 minutes left. Until it finishes, contacts show their previous score."

### Automation: "Sales-ready handoff"
- Created by Marcus Chen. Live since Tue 14 Oct; 37 contacts entered.
- Trigger: score rises to 50 or more, at most once per contact in 90 days.
- Steps: Filter, `plan` is team or business → Add tag `sales-ready` → Webhook POST https://api.acme.io/crm/leads (demo endpoint, never called).

### Verification services
| Service | Connected | Balance | Price per check | Default for |
|---|---|---|---|---|
| ZeroBounce | Marcus Chen, Thu 2 Oct · United States | 36,656 credits | $0.008 | Bulk and imports |
| Kickbox | Marcus Chen, Thu 2 Oct | 9,214 credits | $0.010 | Signup forms |
| NeverBounce, Bouncer, Emailable | not connected | – | – | – |

Prices are Acme's plan prices, entered by Marcus Chen.

### Verification uses
| When | Who | Service | What | Addresses | Cost | Results |
|---|---|---|---|---|---|---|
| Wed 15 Oct 09:12 | Linh Nguyen | ZeroBounce | Import preview: Mailchimp "Acme Customers", new addresses | 11,844 | $94.75 | Valid 10,912 · Invalid 403 · Risky 417 · Unknown 112 |
| Tue 14 Oct 15:02 | Marcus Chen | ZeroBounce | Tier: Sunset | 3,615 | $28.92 | Valid 2,904 · Invalid 287, suppressed · Risky 318 · Unknown 106 |
| Since Thu 2 Oct | Signup forms | Kickbox | Real-time checks on Newsletter signup (338), Footer mini form (126), subscribe page (48) | 512 | $5.12 | Valid 478 · Invalid 17 (asked to fix) · Risky 13 · Unknown 2 · No answer in 3 s 2 (checked later) |

- **Risky, Sunset job:** catch-all 251, disposable 55, role 12.
- **Risky, import:** catch-all 352, role 27, disposable 38 (19 found by ReputeCast's own list, 19 more by ZeroBounce).
- The Sunset job finished at 15:09.

### Contacts and their results
- **Grace Okafor** (g.okafor@outlook.com, Sunset): Valid · ZeroBounce · 14 Oct.
- **d.m···@yahoo.com** (Sunset): Invalid · ZeroBounce: invalid / mailbox_not_found · suppressed 14 Oct, 15:09.
- **m.keller@kellerdesign.ch** (Sunset): Risky · catch-all domain · ZeroBounce: catch-all · tagged `verify:catch-all`.
- **Sam Rivera** and **Lena Fischer:** never checked.

### Before a re-engagement send: "Re-engagement wave 3"
- A draft by Sofia Rossi. Audience: the At risk and Sunset tiers, 8,917 contacts. 287 of them are already suppressed after the 14 Oct job.
- Send Readiness, audience quality, Warn: "5,302 At risk contacts haven't been checked in the last 90 days. The 3,615 Sunset contacts were checked on 14 Oct." Fix: "Verify 5,302 addresses first (ZeroBounce, about $42.42; balance 36,656 credits)".

### Verification states
- **Out of credits:** Marcus Chen checks the Product Updates list (14,110) with Kickbox. "Stopped at 9,214 of 14,110: Kickbox has no credits left. Results for 9,214 are kept. Add credits in Kickbox, then Resume."
- **Partial:** the Sunset job's 106 unknown: "ZeroBounce couldn't get an answer from these mail servers. They keep no result; check them again after 72 hours."
- **Service down:** "ZeroBounce answered 503 for 5 minutes. The job is paused and retries every 5 minutes until 16:10."
- **Forms while Kickbox is down:** "Kickbox isn't answering. Signups go through and are checked when it's back (4 waiting)."
- **Cancelled:** Marcus Chen stopped a job at 2,000 of 3,615. Results for 2,000 are kept.
- **No permission:** Sofia Rossi in the import preview: "Verifying addresses sends them to ZeroBounce and needs subscribers:verify. Ask Marcus Chen or Linh Nguyen."
- **Not connected:** the workspace "Acme Engineering Blog" has no verification service.

### Seed mailboxes
Added by Linh Nguyen on Thu 2 Oct 2026, two days before Postal's warm-up started. acmeseed.io is a domain Acme registered for seeds: gws.acmeseed.io is hosted on Google Workspace and m365.acmeseed.io on Microsoft 365.

| Seed | Mailbox provider | Connected with | Status |
|---|---|---|---|
| acme.seed01@gmail.com | Gmail | IMAP app password | Connected |
| acme.seed02@gmail.com | Gmail | IMAP app password | Connected |
| seed@gws.acmeseed.io | Google Workspace | OAuth | Connected |
| acme.seed@outlook.com | Outlook.com | OAuth | Connected |
| seed@m365.acmeseed.io | Microsoft 365 | OAuth | Connected |
| acme.seed@yahoo.com | Yahoo | IMAP app password | Connected |
| acme.seed@aol.com | AOL | IMAP app password | Connected |
| acme.seed@icloud.com | iCloud | IMAP app-specific password | Sign-in failed since Tue 14 Oct, 21:40 |
| acme.seed@gmx.de | GMX | IMAP password | Connected |
| acme.seed@web.de | Web.de | IMAP password | Connected |

- Copies are deleted 24 hours after each test.

### Placement test: "Weekly Digest #142 · SES vs Postal"
- **Started** by Sofia Rossi from Review & send, Sun 12 Oct 2026 at 16:20 America/New_York, on Postal's warm-up day 9 of 21. Finished 16:37, 15 minutes after the last copy went out.
- **Content:** Weekly Digest #142 as scheduled, with the fallbacks ("Hi there"). From: Acme Newsletter <digest@news.acme.io>.
- **Through:** Amazon SES us-east-1 and Postal (mta.acme.io, 203.0.113.25). 20 copies, 2 minutes apart, alternating which provider went first.

| Seed | Amazon SES | Postal |
|---|---|---|
| Gmail · acme.seed01 | Promotions tab · 9 s | Promotions tab · 26 s |
| Gmail · acme.seed02 | Promotions tab · 11 s | Promotions tab · 31 s |
| Google Workspace | Inbox · Primary · 8 s | Inbox · Primary · 19 s |
| Outlook.com | Inbox · Focused · 14 s | Spam (Junk Email) · 44 s |
| Microsoft 365 | Inbox · Other · 12 s | Inbox · Other · 52 s |
| Yahoo | Inbox · 21 s | Spam (Bulk) · 2 min 10 s |
| AOL | Inbox · 24 s | Inbox · 1 min 48 s |
| iCloud | Inbox · 17 s | Inbox · 41 s |
| GMX | Inbox · 15 s | Inbox · 37 s |
| Web.de | Inbox · 16 s | Not arrived · Postal reports 3 deferrals from Web.de: "421 Temporary rejection, try again later" |

- **Amazon SES:** Inbox 8 · Tab 2 · Spam 0 · Not arrived 0. Median 15 s.
- **Postal:** Inbox 5 · Tab 2 · Spam 2 · Not arrived 1. Median 41 s.
- **Authentication, every copy:**
  - SES: SPF pass (bounce.news.acme.io), DKIM pass (news.acme.io, selector 7kq2n3x), DMARC pass.
  - Postal: SPF pass (rp.news.acme.io), DKIM pass (news.acme.io, selector postal-2026), DMARC pass.
- **Guidance card** (Postal, 2 of 10 in spam): "Same email, same authentication results: the difference comes from the sending IP. Postal is on day 9 of a 21-day warm-up, and young IPs often reach Outlook and Yahoo inboxes last. The warm-up plan already sends your most engaged contacts first." Links: Warm-up plan · Deliverability › Health · Check 203.0.113.25 in HostRepute.

### Placement history
| Started | Content | Through | By | Result |
|---|---|---|---|---|
| Wed 15 Oct, 10:05 | Transactional template password-reset v3 | Postmark | Marcus Chen | Inbox 9 · iCloud not checked (sign-in failed) |
| Mon 13 Oct, 15:10 | Flash sale: 48 hours only (draft) | Amazon SES | Linh Nguyen | Spam 9 · Promotions tab 1 |
| Sun 12 Oct, 16:20 | Weekly Digest #142 | SES vs Postal | Sofia Rossi | SES: Inbox 8, Tab 2 · Postal: Inbox 5, Tab 2, Spam 2, Not arrived 1 |
| Wed 8 Oct, 11:30 | Weekly Digest #141 (warm-up day 5) | Postal | Linh Nguyen | Inbox 4 · Tab 2 · Spam 3 · Not arrived 1 |
| Sun 5 Oct, 10:15 | Weekly Digest #140 (warm-up day 2) | Postal | Linh Nguyen | Inbox 3 · Tab 2 · Spam 4 · Not arrived 1 |
| Thu 2 Oct, 14:40 | Feature flags preview | Amazon SES | Sofia Rossi | Inbox 8 · Tab 2 |

### Placement states
- **Waiting** (Wed 15 Oct, 10:08, password-reset through Postmark): "6 of 9 found · 2 min 51 s · keeps looking until 10:20."
- **Sign-in failed:** "acme.seed@icloud.com: Apple rejected the app-specific password on Tue 14 Oct at 21:40. It may have been revoked. Reconnect with a new one."
- **Everything in spam** (Flash sale, SES): authentication passed on every copy. Guidance: "Send Readiness blocks this email: a link to go.acme-promo.link, which is on SURBL and Spamhaus DBL, and no footer with an unsubscribe link and address. Its subject also trips 3 spam heuristics. Fix those first, then test again."
- **Warm-up planner:** "Placement during this warm-up: day 2: Inbox 3, Tab 2, Spam 4, Not arrived 1 · day 5: 4, 2, 3, 1 · day 9: 5, 2, 2, 1." With "Run a test through Postal".
- **No permission:** Daniel Park: "Running a placement test sends copies through your providers and needs placement:run. Ask Sofia Rossi, Marcus Chen or Linh Nguyen."
- **No seeds:** the workspace "Acme Engineering Blog".

### Growth: last 30 days (16 Sep – 15 Oct 2026), all lists
- **Joined 1,756 · Left 472 · Net +1,284 (+2.7%)**, the same net growth the Overview shows.
- **Imports on 18 Sep, counted apart:** Brevo, 7,391 contacts that took the import day as their join date (32,011 more kept their original date and source); Mailchimp, 2,118.
- **Workspace rates over the period:** bounces 0.6%, complaints 0.04%. A source is flagged above 1.2% bounces or 0.08% complaints.

| Source | Joined | Left | Net | Bounces | Complaints | Human opens |
|---|---|---|---|---|---|---|
| Newsletter signup (embedded on acme.io and blog.acme.io) | 731 | 39 | +692 | 0.3% | 0.01% | 52.4% |
| Footer mini form | 612 | 31 | +581 | 0.4% | 0.02% | 47.1% |
| Subscribe page (news.acme.io/subscribe) | 104 | 3 | +101 | 0.3% | 0.00% | 50.8% |
| Beta apply | 43 | 0 | +43 | 0.2% | 0.00% | 63.5% |
| API key "Website signup form" | 214 | 31 | +183 | 2.1% · flagged | 0.03% | 39.6% |
| Zapier · Beta application (Typeform) | 38 | 0 | +38 | Too few to compare | | |
| Added by a person | 14 | 0 | +14 | Too few to compare | | |
| Brevo import (18 Sep) | apart | 301 | – | 0.6% | 0.04% | 41.0% |
| Mailchimp import (18 Sep) | apart | 64 | – | 1.9% · flagged | 0.18% · flagged | 18.6% |
| CSV import (Feb 2024) | – | 3 | – | 0.9% | 0.05% | 24.3% |
| **Total** | **1,756** | **472** | **+1,284** | | | |

- **Departures by kind:** unsubscribed from every list 333 · hard-bounced 104 · complained 27 · erased 8.
- **Flag note on the API key:** "Signups through the 'Website signup form' key bounce 3.5 times as often as the workspace (2.1% against 0.6%). Typos are the usual cause: check addresses on the site before calling the API, or verify these 214 contacts (ZeroBounce, about $1.71)."

### UTM and referring pages (1,490 joins from forms and the subscribe page)
| utm_source / utm_medium / utm_campaign | Joins | Human opens |
|---|---|---|
| (none) | 716 | 49.8% |
| linkedin / social / flags-ga | 214 | 51.2% |
| google / cpc / brand-newsletter | 168 | 44.0% |
| github / readme / – | 121 | 58.3% |
| hackernews / social / boring-infra | 97 | 61.4% |
| devto / social / flags-ga | 58 | 47.5% |
| Other, 14 combinations | 116 | 46.9% |

| Referring page | Joins |
|---|---|
| acme.io/pricing | 241 |
| docs.acme.io/flags | 198 |
| blog.acme.io/2026/10/postgres-partitioning | 176 |
| github.com/acme/flags | 121 |
| news.ycombinator.com | 97 |
| None (typed or bookmarked) | 493 |
| Other, 52 pages | 164 |

### Cohort retention, all sources
| Joined in | Contacts | Still subscribed after 30 days | After 90 days |
|---|---|---|---|
| June 2026 | 1,402 | 95.9% | 91.2% |
| July 2026 | 1,288 | 96.3% | in progress, complete 29 Oct |
| August 2026 | 1,351 | 95.1% | in progress, complete 29 Nov |
| September 2026 | 1,612 | in progress, complete 30 Oct | – |

- Cohorts before 18 Sep use the join and unsubscribe dates the Brevo import carried over.
- **Zapier · Beta application (Typeform):** first join 24 Sep, so "30-day retention from 24 Oct".

### Growth states
- **No UTM data yet:** "Forms and pages capture UTM parameters from today. Joins before 15 Oct show 'Not captured'."
- **Too little data:** "Added by a person: 14 contacts. Too few to compare."
- **Only imports so far** (Acme Engineering Blog): "Moving your list isn't growth. Joins from your forms, API keys and integrations will show here."
- **Contacts filtered by source:** "Source: Zapier · Beta application (Typeform) · 38 contacts", and by `utm_campaign` flags-ga: 272 contacts.

### People
- **Sofia Rossi** (Editor) set up the scoring rules and runs placement tests from Review & send. She can't send addresses to a verification service.
- **Marcus Chen** (Admin) connected ZeroBounce and Kickbox, runs verification jobs and built "Sales-ready handoff".
- **Linh Nguyen** (Owner) added the seed mailboxes, tests Postal during its warm-up and verified the Mailchimp import.
- **Daniel Park** (Content Writer) sees scores, Growth and placement results. He can't run tests.
- **Aisha Bello** (Viewer) sees everything read-only.

---

## 5. New screens

No new sidebar items. Audience gets two tabs (Contacts · Segments · List health · Scoring · Growth · Suppression list), Deliverability gets two (Health · Placement tests), and the verification services join the Integrations hub.

### 5.1 Audience › Scoring
- **At the top:**
  - the note on how the score differs from tiers (3.1), with a link to List health
  - three band tiles: Cold 29,190 (60.5%), Warm 14,207 (29.5%), Hot 4,816 (10.0%)
  - the spread chart: buckets of 10, coloured by band, with the two thresholds marked
- **Rules:** a table with the rule in words, points, cap, a switch, and a row menu (Edit, Duplicate, Delete).
  - "Add rule" opens a side panel: the event or field condition, its details (which form, which API event, which link text, which field and value), points, cap.
- **Decay and limits:** decay on or off, after how many weeks, by how much; the maximum score; the 180-day look-back, stated.
- **Bands:** the two thresholds, as number fields that also move the markers on the chart.
- **Bands by tier:** the table from section 4, so the difference is visible.
- **Used by:** "Sales-ready handoff" (automation). Segments and conditional blocks that use the score are listed here too.
- **Save bar** while something changed: "Unsaved changes · Save". The chart already shows the new spread.
- **Save dialog:** the moves between bands, the automations question when contacts would cross a trigger's threshold, and Save.

States to show:
- the sample
- no rules yet: "Start with suggested rules" (the sample's eight) or "Add a rule"
- editing rule 3 to +25: the new spread over the current one, and "1,108 Warm → Hot, 204 Cold → Warm"
- the save dialog with the "Sales-ready handoff" question
- recalculating: "31,400 of 48,213 · about 2 minutes left"
- recalculation failed: scores unchanged, with Try again
- validation: "A cap of 3 a week is lower than the 5 points one click gives"
- read-only (Daniel Park, Aisha Bello): "Editing scoring needs scoring:edit. Ask Sofia Rossi, Marcus Chen or Linh Nguyen."
- loading, error
- tablet

### 5.2 Settings › Integrations › Verification services
The phase 2 hub's "Verification" group and its detail page.
- **Catalog cards:** the five services, with capability badges and their status.
- **Detail page (ZeroBounce):**
  - status, balance with the time it was read, price per check (editable), default use
  - the key as a masked value with Replace, and Test connection
  - result mapping: the service's codes, the ReputeCast result each one becomes, and what happens (suppressed, tagged, nothing)
  - the uses (section 4), each linking to its job or audit entry
  - what is sent, as the outbound-data notice
  - Disconnect
- **Connect dialog:** key, price per check, default use, Test connection.

States to show:
- not connected: the catalog group, as Marcus Chen
- the connect dialog, and a rejected key
- ZeroBounce connected (sample)
- Kickbox as the forms default
- low credits (Kickbox): "9,214 credits. Checking Product Updates needs 14,110." (a warn note)
- service error: the last error and when it happened
- read-only (Sofia Rossi): status, balance, uses and the notice; no key, no settings
- disconnect confirmation: "Newsletter signup, Footer mini form and the subscribe page stop checking addresses as people sign up."
- loading, error

### 5.3 Verify addresses
One dialog from Contacts (a selection or a saved view), List health (a list or a tier), a segment, or a contact profile; then a job; then its results.
- **Dialog:**
  - what is checked: "Sunset tier · 3,615 contacts"
  - service: ZeroBounce (default) or Kickbox
  - "Skip addresses checked in the last 90 days", with how many it skips
  - cost: "3,615 checks × $0.008 = $28.92 · balance 52,115 credits, 48,500 after"
  - the outbound-data notice
  - what happens to each result
  - "Verify 3,615 addresses"
- **Job:** in the jobs tray and on a progress panel: "1,240 of 3,615 · about 4 minutes left", the results so far, and Cancel.
- **Results:**
  - counts per result as a bar, with "287 suppressed for all mail"
  - the risky ones by reason, with their tags
  - unknown ones, with the date they can be checked again
  - links: "See the 287 suppressed", "See risky contacts" (Contacts, filtered)
  - "Download results (CSV)", which holds addresses, so it needs subscribers:export ⚑ and is audited

States to show:
- the dialog (sample), and with addresses skipped
- the job running
- the results (sample)
- cancelled at 2,000 of 3,615
- out of credits (Kickbox, Product Updates)
- service down, retrying
- partial: 106 unknown
- no service connected: "Connect a verification service in Settings › Integrations", or who can, for people without integrations:manage
- no permission (Sofia Rossi)
- one contact, from the profile: "Verify sam.rivera@outlook.com with ZeroBounce · 1 credit · $0.008"

### 5.4 Deliverability › Placement tests
- **Tabs:** Health (today's page) and Placement tests. Inside Placement tests, pill tabs: Tests and Seed mailboxes (5.7).
- **Header:** "Run placement test" and "Seed mailboxes · 10 · 1 needs attention".
- **Tests table:** started, content, through, by, and the result as a small stacked bar per provider (Inbox, Tab, Spam, Not arrived) with counts. Running tests show live counts.
- **Trend:** one stacked bar per test in time order, filtered by provider. Caption: "Each bar is one test of up to 10 seeds. Tests send different emails, so compare like with like."
- **Notes:** "Copies are deleted 24 hours after each test." and "A test never changes a send."

States to show:
- the sample
- a test running
- seeds connected, no tests yet
- no seeds: "Add mailboxes you own at Gmail, Outlook, Yahoo and others. Each test sends one copy to each."
- a seed's sign-in failed: a warn banner with Reconnect
- Daniel Park: "Run placement test" disabled, with the reason and who can help
- Aisha Bello: read-only
- loading, error

### 5.5 Run a placement test
A dialog, opened from 5.4, Review & send, the composer's Send test menu or the warm-up planner. The entry point fills in what it knows.
- **Content:** a picker with search across campaigns, campaign templates, digests' next issues and transactional templates.
- **Version:** the conditional-content version, language variant or A/B variant, when the email has them.
- **Personalization:** "Fallbacks" (default), or a test-data set for a transactional template.
- **Through:** the route as configured, one provider, or compare providers (a checklist with each provider's health badge).
- **From:** a verified sender of the stream.
- **Seeds:** every connected seed, checked; a seed whose sign-in failed is unchecked and says why.
- **Summary:** "20 copies · SES $0.001 · Postal $0. Copies through Postal count toward its warm-up cap."
- **Start test.**

States to show:
- from Review & send: Weekly Digest #142, comparing SES and Postal (sample)
- from the composer's Send test menu: the route as configured
- from the warm-up planner: Postal chosen, with the last sent campaign
- a transactional template: password-reset v3 with "New user", through the Transactional route (Postmark)
- a provider that is Down: unavailable, with the reason
- no verified sender for the stream: blocked, with a link to Senders
- the iCloud seed unchecked: "Sign-in failed on 14 Oct"

### 5.6 Placement test result
- **Header:** the test's name, who started it and when, status (Waiting, Finished, Stopped), content and sender. Actions: Run again, Stop waiting, Delete copies now.
- **Summary per provider:** a stacked bar with counts per folder and the median time to arrive. "8 of 10 seeds in the inbox", never a percentage.
- **Matrix:** a row per seed, grouped by mailbox provider; a column per provider. Each cell: the folder badge, the time to arrive, and SPF, DKIM and DMARC as small pass or fail marks.
- **Copy detail:** a side panel from any cell:
  - the folder path as the mailbox reports it
  - Authentication-Results and the Received line from the provider
  - the provider's message ID and its events for this copy
  - for Not arrived: the deferrals or bounce the provider reported
- **Guidance card** when half or more of a provider's copies land in spam (3.3).
- **Footnote:** "10 seeds are a sample, not your audience. Use tests to compare emails and providers."

States to show:
- finished (sample)
- waiting: cells fill in as copies arrive, with the elapsed time and "keeps looking until 16:37"
- the copy detail of Postal → Outlook.com, in Junk Email
- partial: Web.de not arrived after 15 minutes, with Postal's deferrals
- everything in spam: the Flash sale, with its guidance
- a seed not checked (iCloud)
- a copy that arrived late
- stopped early
- HostRepute not connected: the guidance without the HostRepute link
- loading, error
- tablet, and phone (the matrix becomes one card per seed, with a row per provider)

### 5.7 Seed mailboxes
The "Seed mailboxes" pill tab in Placement tests.
- **Table:** seed, mailbox provider, connected with, status, last used, added by, and Reconnect or Remove.
- **Add seed dialog:**
  - pick the mailbox provider
  - OAuth: a "Connect with Google" or "Connect with Microsoft" button
  - IMAP: host and port filled in, username, app password, and a link to that provider's help on app passwords
  - the notice about using mailboxes that exist only for tests
  - for gmail.com: the 7-day sign-in note and the app-password recommendation
  - "Check and add", then the result: signed in, and the folders found
- **Settings:** delete copies after 24 hours, when the test finishes, or never.

States to show:
- the sample
- adding a Microsoft 365 seed with OAuth
- adding a Yahoo seed with an app password: check passed, folders found
- IMAP refused: "GMX refused the sign-in. Turn on IMAP access in GMX's settings, then try again."
- OAuth cancelled at the provider
- a seed whose sign-in failed (iCloud), with Reconnect
- an OAuth seed whose sign-in expired: "Google ended ReputeCast's access to seed@gws.acmeseed.io today at 10:20. Reconnect to keep testing it."
- remove confirmation: "Copies of past tests stay in this mailbox until you delete them there."
- no seeds yet
- read-only for people without placement:manage (Sofia Rossi): the list without Add, Reconnect or Remove, and who can help
- loading, error

### 5.8 Audience › Growth
- **Header:** period (30 days, 90 days, 12 months, custom), list (all lists or one), "Include imports", Export.
- **Tiles:** Joined 1,756 · Left 472 · Net +1,284 (+2.7%) · Imports on 18 Sep, counted apart.
- **Chart:** joins above the axis and departures below, by day, in the four biggest sources plus Other; imports as a marker when included; a table view.
- **Sources table:** joined, left, net, bounces, complaints, human opens; flags and "Too few to compare"; each row links to its contacts.
- **Departures by kind.**
- **UTM and referring pages:** pill tabs over the two tables.
- **Cohort retention:** the table, with a source picker.

States to show:
- the sample
- imports included: the 18 Sep marker and the import rows' joins
- one list: Beta Program
- a flagged source: the "Website signup form" API key, with its note and "Verify these 214 contacts"
- no UTM data yet
- too little data: "Added by a person", and Zapier's retention "from 24 Oct"
- only imports so far (Acme Engineering Blog)
- empty: no contacts yet
- loading, error
- tablet, and phone (tiles two by two, the chart scrolls inside its card, the tables wrap their secondary cells)

---

## 6. Changes to existing screens

- **Contact profile:**
  - A score card: "52 · Hot", a 90-day line, the breakdown by rule, the last change, and the note on tiers.
  - An address row: "Never checked · Verify now" (subscribers:verify), or "Valid · ZeroBounce · 14 Oct".
  - The join line gains UTM and the referring page for new signups: "Joined Mon 13 Oct · Newsletter signup · linkedin / social / flags-ga · from docs.acme.io/flags".
- **Contacts:**
  - A Score column (sortable), and filters for band, address check result, source, UTM source, medium and campaign, and referring domain (under More filters).
  - A saved view example: "Beta applicants from Typeform · 38".
- **Segment builder:** conditions for score, band, address check result, source, UTM and referring domain.
- **Automation flow:** the trigger "Score crosses…" in the palette and its side panel, shown with "Sales-ready handoff".
- **List health:**
  - The note on tiers and scores.
  - "Verify addresses" in each list's ⋯ menu, and on the At risk and Sunset tier cards.
  - "Net growth · 30 d" says it counts list members, with a link "By source" to Growth.
- **Overview:** the Subscriber growth card links "By source" to Audience › Growth.
- **Migration importer › Preview:**
  - the optional verification card: count, cost, balance, notice
  - after checking, the results join the preview's groups, with ZeroBounce's share marked
  - Sofia Rossi without subscribers:verify, and no service connected
- **Form builder:**
  - Spam protection: "Check each address with Kickbox as people sign up" (needs subscribers:verify), with "If Kickbox doesn't answer within 3 seconds, the signup goes through and is checked later".
  - A line under Consent: "UTM parameters and the referring page are saved with each signup."
  - The 30-day numbers gain "Asked to fix their address: 17".
- **Review & send:**
  - Audience quality uses the check results (3.2), including "Re-engagement wave 3"'s Warn and its fix.
  - An optional "Placement test" card: "Not run for this campaign · Run placement test", or the latest result per provider with "View results". It is never part of the verdict.
- **Send approvals, review:** the campaign's latest placement test, when there is one, shown as context.
- **Composer:** "Send test" becomes a split button: "Send a test email…" and "Run a placement test…". The template editor in transactional mode gets the same.
- **Warm-up planner:** a "Placement during this warm-up" card with the day 2, 5 and 9 tests as small stacked bars, and "Run a test through Postal".
- **Deliverability** (with and without HostRepute): the tabs Health and Placement tests. Placement tests work the same without HostRepute.
- **Suppression list:** the reasons "Invalid address" and "Spam trap", the source "ZeroBounce" or "Kickbox", and the service's code in the Original code column; both in the filters.
- **Settings › Integrations (phase 2 hub):** the "Verification" group, with ZeroBounce and Kickbox connected.
- **Instance admin › Authentication:** the Google and Microsoft OAuth settings note that seed mailboxes use them for mail access, and Google's limit on unverified apps.
- **Roles & permissions, role editor:** subscribers:verify ⚑ in Contacts; new groups "Audience scoring" (scoring:edit) and "Placement tests" (placement:run, placement:manage ⚑). Built-in roles as in 3.5.
- **API keys & MCP agents:** the new keys. For agents, subscribers:verify, placement:run and placement:manage aren't offered, with the reason.
- **Settings › MCP server:** read-only tools `list_placement_tests` and `get_placement_test`, `get_growth_sources`, and score and band in `search_contacts`.
- **Audit log:** the actions in 3.6, and the filter categories Scoring, Verification and Placement tests.
- **Notifications:** the kinds in 3.7, in the list and in the preferences.
- **Phase 3 screens:** the Visibility condition picker gains Score and Score band.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens, badges and components.

- **Score meter and bands:**
  - a 0–100 track with the two thresholds marked, the score as a marker, and the number with its band in words: "52 · Hot"
  - band colours: a new three-step ordinal ramp, `chart-score-1` to `chart-score-3`, validated like the other chart palettes; never status colours, never the tier palette
  - a breakdown list: the rule in words, its points, and "cap reached" when it applies
- **Placement matrix:**
  - folder badges: Inbox (ok), Tab with its name (info), Spam (bad), Not arrived (neutral), Not checked (neutral, with the reason), Waiting (neutral, pulse)
  - SPF, DKIM and DMARC as small marks with an icon and a word for screen readers
  - times in tabular figures
  - on phones, one card per seed with a row per provider
- **Placement bar:** a stacked bar per provider or per test: Inbox, Tab, Spam, Not arrived, in the same status colours as the badges, each segment with its count.
- **Growth-source chart:** joins above the axis and departures below, by day; the four biggest sources take the categorical palette in its fixed order and the rest `chart-other`; imports as a marker; always with a table view.
- **Verification result badges:** Valid (ok), Invalid (bad), Risky with the reason (warn), Unknown (neutral). "Never checked" is plain text.
- **Flagged value:** a warn badge beside a table value: "2× the workspace".
- **Outbound-data notice:** phase 3's pattern, reused for verification.

---

## 8. Conflicts to note

1. **Audit log vs real-time checks.** CLAUDE.md says every state-changing action writes an audit entry. A real-time check runs for every signup, so this brief audits turning it on or off and records the checks as daily counts per form in the service's activity, as phase 1 does with sends. The owner needs to confirm this.
2. **One-off tests and a trend.** The plan says ReputeCast does no continuous monitoring. The history and its trend only show tests people started; there are no schedules and no alerts. The owner should confirm the trend is acceptable on that basis.
3. **Deliverability gets tabs.** The Deliverability boards have none today. Health is today's page, unchanged.
4. **Seed mailboxes and OAuth.** Gmail and Outlook mail access through OAuth depends on Google's and Microsoft's app rules: Google limits unverified apps, which can force a gmail.com seed to sign in again every 7 days. IMAP app passwords are the dependable path. Engineering should confirm the providers' current rules before building.
5. **Mailbox provider names.** Contacts and reports group mailboxes into Gmail, Outlook, Yahoo, iCloud and Other (Google Workspace shows as "Other · Google Workspace"). Seeds name the exact provider: Google Workspace, Microsoft 365, AOL, GMX, Web.de. The five report categories don't change.
6. **Growth and List health.** List health's "Net growth · 30 d" counts list members (+1,284 in the sample across the three lists); Growth counts contacts by default. In the sample both come to +1,284; each screen says which it counts.
7. **Imported contacts' sources.** The Contacts board shows imported contacts with their original join dates and sources (Priya Natarajan, 2 Mar 2025, "Signup form · docs"). Growth uses them, so an import only counts the contacts that came without one.
8. **Addresses leaving the instance.** Until now only sending providers received contacts' addresses. Verification services are a second kind. This brief makes subscribers:verify sensitive and keeps it from Editors by default; the owner should confirm both.
9. **Scores and automations after a rule change.** Starting "Score crosses…" automations for contacts who cross only because a rule changed would flood them, so the default is not to start them.

---

## 9. Boards

New boards, on the canvas page "P4 Audience insights & placement tests":

| File | Canvas title | Implements | Tablet and phone |
|---|---|---|---|
| `ScoringRules.dc.html` | Audience · Scoring | 5.1 | `ScoringRulesTablet.dc.html` |
| `ScoringStates.dc.html` | Scoring · no rules, impact preview, save dialog, recalculating, read-only | 5.1 states | — |
| `VerificationServices.dc.html` | Integrations · verification services | 5.2 | — |
| `VerificationRun.dc.html` | Verify addresses · dialog, job and results | 5.3 | — |
| `VerificationStates.dc.html` | Verification · out of credits, partial, service down, no permission, import preview, signup form | 5.2–5.3 states, section 6 (import preview, form builder) | — |
| `PlacementTests.dc.html` | Deliverability · placement tests | 5.4 | — |
| `PlacementRun.dc.html` | Placement · run a test | 5.5 | — |
| `PlacementResult.dc.html` | Placement · test result and copy detail | 5.6 | `PlacementResultTablet.dc.html`, `PlacementResultPhone.dc.html` |
| `PlacementSeeds.dc.html` | Placement · seed mailboxes | 5.7 | — |
| `PlacementStates.dc.html` | Placement · waiting, everything in spam, sign-in failed, no seeds, no permission | 5.4–5.7 states | — |
| `GrowthSources.dc.html` | Audience · growth sources | 5.8 | `GrowthSourcesTablet.dc.html`, `GrowthSourcesPhone.dc.html` |
| `GrowthStates.dc.html` | Growth · imports, no UTM yet, too little data, flagged source | 5.8 states | — |

12 desktop boards and 5 tablet and phone boards.

Existing boards that section 6 changes:
- `ContactProfile.dc.html`, `ContactProfilePhone.dc.html`
- `Contacts.dc.html`, `ContactsTablet.dc.html`, `ContactsPhone.dc.html`
- `SegmentBuilder.dc.html`
- `AutomationFlow.dc.html`
- `ListHealth.dc.html`
- `Main.dc.html`
- `MigrationPreview.dc.html`
- `FormBuilder.dc.html`
- `SendReadiness.dc.html`, `ApprovalReview.dc.html`
- `Composer.dc.html`
- `ProvidersWarmup.dc.html`
- `Deliverability.dc.html`, `DeliverabilityTablet.dc.html`, `DeliverabilityPhone.dc.html`, `DeliverabilityNoHostRepute.dc.html`
- `Suppressions.dc.html`
- Phase 2's `IntegrationsCatalog.dc.html` (the Verification group)
- `InstanceAuth.dc.html`
- `Roles.dc.html`, `RoleEditor.dc.html`, `SettingsKeys.dc.html`, `SettingsMCP.dc.html`
- `SettingsAudit.dc.html`
- `Notifications.dc.html`
- `DSPatterns.dc.html`, `DSComponents.dc.html`, `DSCharts.dc.html` (section 7), `DSCoverage.dc.html` (the new screens and states)
- Phase 3's `ComposerVisibility.dc.html`
