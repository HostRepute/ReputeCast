# Review 2026-11: polishing the features that already exist

Source: a detail review of ReputeCast on 31 October 2026, after M40. The October review
([`reputecast-review-2026-10.md`](reputecast-review-2026-10.md)) hardened five core features: Send
Readiness, guarded sending, providers, the composer and reports. This review covers everything
else that already ships: audience, forms and public pages, automations, campaigns, templates,
transactional mail, approvals, polls and surveys, Grow, Earn, the other channels, the platform,
operations and the web app's foundation. It adds no new product areas. It finishes, fixes and
tightens what is there, measured against what competitors ship in 2026.

Every recommendation has an ID, an owner milestone (M41–M50 in `docs/ROADMAP.md`), what "done"
means, a size and the design work it needs. Status: ⏳ planned · 🚧 in progress · ✅ done.

## How the review was done

- **Three code audits**, one per area: audience and growth; create, automate and operate; platform,
  Earn, operations and the frontend foundation. The code has no TODO markers, so the audits read
  behaviour: handlers, queries, jobs, screens and the spec sections and boards for each.
- **A verification pass** re-read every finding in the code before it was kept. 26 findings were
  checked line by line: 24 held as stated, 2 held in part (the approvals inbox does filter by
  workspace; deleting and restoring a list does have a screen). Where a finding was corrected, the
  row below says what is really wrong.
- **Board coverage**: each proposed screen change was looked up in `docs/DESIGN_INVENTORY.md` and
  `design/boards`. The "Board" column says whether a board already draws it, which board to extend,
  or that a new board must be drawn first. Per `CLAUDE.md`, no screen is built before its board.
- **Competitors**: the 2026 releases of beehiiv, Kit, Ghost, Mailchimp, MailerLite, Customer.io,
  Resend, Postmark, listmonk and Loops (see [Competitor benchmark](#competitor-benchmark)). A
  competitor's behaviour is only listed where it improves something ReputeCast already has.

Sizes: **S** about a day, **M** a few days, **L** a week or more. Paths are relative to the
repository root; line numbers are those of 31 October 2026.

## Fix first

These are bugs, not polish. Each was confirmed in the code.

| # | What is wrong | Evidence | Row |
|---|---|---|---|
| 1 | A reader who picks "Weekly roundup" or "Monthly best-of" on the preference center (offered on the unsubscribe page *instead of* unsubscribing) keeps getting every issue: the choice is stored and shown again, and nothing that sends reads it. | `internal/store/queries/audience_signup.sql:98-104` is the only reader and writer of `contact_preferences.frequency` | CA-6 |
| 2 | One failing automation event or run rolls back its whole batch (200 events, 100 runs), and a failed dispatch skips the tick. The same row is claimed first every 10 seconds, so automations stop for every workspace on the instance. | `internal/automations/engine.go:159-182`, `internal/automations/engine.go:336-338`, `internal/automations/jobs.go:44-46` | EN-1 |
| 3 | A date trigger in a workspace east of UTC inserts its event again every minute until UTC midnight: the dedupe compares with a date cast in the database session's time zone. With re-entry "always", contacts get the email again and again. A yearly trigger with the default re-entry "once" fires only in its first year. | `internal/store/queries/automations.sql:505-507`, `internal/automations/engine.go:1044-1059` | EN-2 |
| 4 | Automation steps' Delivered, Bounced and Complained counts are written only by the demo's simulator. On SES, Postmark and every real provider they stay at 0, and the list's open rate shows "–". | `internal/sending/engine/sim.go:126` is the only caller for delivery events | EN-5 |
| 5 | 12 events of the outgoing webhook catalog can be subscribed to but are never sent: `email.sent`, `email.delivered`, `email.deferred`, `email.bounced`, `email.complained`, `email.opened`, `email.clicked`, `provider.down`, `provider.recovered`, `automation.published`, `contact.erased`; `campaign.scheduled` is sent for approved, SMS, push and WhatsApp campaigns but not when an email campaign is scheduled directly. | `internal/webhooks/webhooks.go:72-89`; every emitter is a `webhooks.Emit` call | EN-6 |
| 6 | Erasing a contact leaves their phone number, channel consents, paid membership and its timeline (`EarnErasePaid` and `EarnEraseTimeline` are never called), orders, inbound SMS and WhatsApp messages and push endpoints. The contact's data export leaves out SMS, push, WhatsApp, paid membership and gifts, and stops silently at 10,000 messages and 10,000 events. | `internal/audience/bulk.go:249-279`, `internal/store/queries/earn_members.sql:128-133`, `internal/audience/gdpr.go` | DP-1, DP-2 |
| 7 | Double opt-in confirms on a plain GET, so corporate link scanners confirm addresses that bots submitted; confirmation tokens never expire; a pending address is mailed again on every resubmission with only per-IP and per-form limits (about 120 confirmation emails a minute to one victim from rotating addresses); unconfirmed signups are never purged. | `internal/publicpages/publicpages.go:228`, `internal/audience/signup.go:252-266`, `internal/forms/submit.go:196`, `internal/maintenance/maintenance.go:41-63` | DP-3, DP-4 |
| 8 | Automation emails ignore a contact's preference-center pause, which campaigns honour; "opened/clicked within X" windows start when the contact reaches the branch, not when the email is sent, so mail held by an outage or warm-up misses its window; "skip if they got an email in the last N hours" counts transactional, system and never-sent messages. | `internal/store/queries/sending.sql:210` (campaigns only), `internal/automations/engine.go:527-571`, `internal/automations/engine.go:638-652`, `internal/store/queries/automations.sql:358-359` | DP-5, EN-3, EN-4 |
| 9 | Smaller bugs: the Campaigns list's Duplicate button does nothing; the Contacts Source filter offers the demo's sources; tag search is case-sensitive while tags are stored lower-case; the transactional log lower-cases the search but matches idempotency keys exactly and never matches batch keys (`key#0`, `key#1`…); the approvals inbox stops at 200 rows and filters status afterwards in Go, so tabs lose items; Stripe's `checkout.session.async_payment_failed` and `charge.dispute.updated` are not subscribed, so a failed bank debit never resolves. | `web/src/features/campaigns/CampaignsPage.tsx:464`, `web/src/features/audience/ContactsPage.tsx:336-349`, `internal/audience/filter.go:89`, `internal/transactional/messages.go:308`, `internal/store/queries/transactional.sql:514`, `internal/store/queries/approvals.sql:155`, `internal/approvals/inbox.go:162`, `internal/payments/stripe/stripe.go:66-83` | EN-7 |

## Decisions taken by default

Until the owner decides otherwise, the build follows these.

- **D1 Cadence.** A campaign has an audience cadence. Regular issues reach "Every issue" only. Digest
  issues (M20) are marked weekly or monthly and reach readers who chose that rhythm as well as
  "Every issue" readers who are on the digest's lists. A campaign marked "Announcement" reaches
  everyone on its lists whatever they chose. Readers with no choice are "Every issue".
- **D2 Erasure of money records** (confirmed by the owner, 31 October 2026). A contact who still
  pays through Stripe (trialing, active or past due) can't be erased: the erase is refused with the
  way to Earn › Paid members, and the sunset policy's auto-erase skips them until the subscription
  ends; nothing is canceled behind anyone's back. Ended subscriptions are anonymized, not deleted
  (deleting them would cascade into `earn_mrr_changes` and change past MRR): billing address,
  Stripe ids, cancel reason and notes go, amounts and dates stay. The paid timeline is deleted, as
  the Earn brief says (§3.16), and the dialog says Stripe keeps its own records. Complimentary and
  gift access ends. Orders, gifts and conversions are anonymized the same way.
- **D3 Retention.** The IP address and user agent of marketing opens and clicks are cleared after 12
  months by default (a workspace setting from 30 days to "keep"); counters, scores, tiers and A/B
  decisions are kept. Signups never confirmed are erased after 30 days (a setting, on by default).
  Both settings sit on Settings › General › Data retention (rather than on the forms, since a
  contact can come from several forms).
- **D7 Paused contacts in automations** (owner, 31 October 2026). An automation email to a contact
  who paused email on the preference center is skipped, counted as a skip, and the flow goes on,
  as campaigns leave them out. SMS, push and WhatsApp steps don't read the email pause.
- **D8 Metrics** (owner, 31 October 2026). `/metrics` listens on its own port, `:9464` by default;
  the main listener serves it only to a bearer token (`metrics.token`) and answers 404 otherwise.
- **D4 Deleting campaigns.** Drafts and cancelled campaigns can be deleted (restorable for 30 days,
  like lists). Sent campaigns are archived, never deleted: reports, web posts, retries and approvals
  still point at them.
- **D5 Large counts.** Contacts and Suppressions show exact counts up to 10,000 and "10,000+" above,
  with an exact count on request. This changes the count line ("Showing 1–25 of N") on those two
  screens and is recorded as a deviation in `docs/DESIGN_INVENTORY.md`.
- **D6 Stack.** `docs/TECH_STACK.md` lists recharts, react-hook-form and zod. Recharts has no import
  (charts are drawn as SVG) and react-hook-form is used in one file. Nothing is removed or swapped
  without the owner's answer (`CLAUDE.md`); FE-6 asks the question and records it.

## Consent and data protection (M41) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| DP-1 | An erase registry: every package that stores a contact's personal data registers how to erase it, and `eraseContacts` walks the registry instead of a fixed list. Phone numbers, channel consents and their events, paid status and timeline (removed, D2), orders and gifts (anonymized, D2), inbound SMS and WhatsApp bodies and numbers, push endpoints, unsubscribe feedback text and website member sessions are covered. `contact.erased` is emitted. The erase dialog says Stripe keeps its own records. | A test lists every table with a `contact_id`, email or phone column and fails when one is not in the registry (the pattern of `internal/store/writers_test.go`); an erase test asserts no personal data remains in any of them. | M | Extend the erase dialog (Dialogs) with the Stripe line |
| DP-2 | The contact's data export uses the same registry: SMS, push and WhatsApp messages and consents, paid membership and its timeline, gifts, orders, referrals, poll and survey answers, UTM and referrer, address checks. It streams, and says so when a source was cut instead of stopping silently at 10,000. | Export test covers every registered source and the truncation marker. | M | — |
| DP-3 | Double opt-in that link scanners can't trigger: `/confirm/{token}` shows a page with one Confirm button that POSTs (as the recommendations page does); tokens carry their issue date, expire after 30 days and confirm only the lists of the submission they were sent for; a pending address gets at most one confirmation email per 10 minutes and 5 a day, while the page still says "check your inbox"; the form limiter keys IPv6 by /64 and turns Turnstile on when a form's bucket trips. | Tests: a GET never confirms; an expired link offers a new one; the 2nd submission within 10 minutes sends nothing; a /64 counts as one client. | M | PublicConfirm needs a "Confirm" state and an expired state (reuse PublicSiteStates' "This link has expired · Send a new link") |
| DP-4 | Signups never confirmed are erased after 30 days (D3) by the maintenance job, through DP-1. | Test with a virtual clock. | S | Settings line on the forms' double opt-in settings |
| DP-5 | Automation emails honour the preference center: a paused contact waits (or is skipped, by the step's skip rule) until the pause ends; unsubscribing from the trigger list ends the run when the flow says "exit when they leave the trigger list" (AT-8). | Tests: paused contact gets nothing until the pause ends; unsubscribed contact leaves. | S | — |
| DP-6 | Event detail retention (D3): a nightly job clears `ip` and `user_agent` on marketing `delivery_events` and `messages` older than the workspace's setting; counters, scores, tiers and A/B decisions are untouched. | Test: a report's numbers are identical before and after the job. | M | Settings › General gains the retention setting (extend WorkspaceSettings) |
| DP-7 | `/metrics` is served on its own address by default, or on the main listener only with a bearer token; the reverse-proxy guide says so. Migrations take goose's Postgres session lock (`WithSessionLocker`) so replicas started with auto-migrate can't run them twice. | Tests: unauthenticated `/metrics` on the main listener is 404; two concurrent migrators run each migration once. | S | — |

Riskiest part: DP-1's money records and DP-6's retention must not change any number a report or
revenue page shows beyond what D2 removes. Both are tested by comparing those numbers before and after.

What M41 built (31 October 2026):

- **DP-1** `internal/erasure`: a registry of 21 sources (one per package that keeps personal data
  about contacts, the audience's own last), wired in `internal/app/wire_erasure.go`. Erasing walks it
  for the API, the sunset auto-erase and the unconfirmed-signup purge; Earn's guard refuses live
  Stripe payers (409 `paid_members`). `contact.erased` goes out through the contact-event queue only
  when an endpoint listens, without the address. `TestErasureCoversPersonalData` fails when a table
  with personal-data columns is in no source and not exempted with a reason (it caught
  `optin_sends` while M41 was built); `TestEraseLeavesNothingPersonal` erases demo contacts drawn
  from every registered table and searches every table of the workspace for their addresses and
  phone numbers; `TestErasureKeepsRevenueNumbers` compares the revenue overview before and after.
  The erase dialog and its refusal are drawn on `Dialogs.dc.html`.
- **DP-2** the contact export streams with no cap, a section per source (omitted with the reason
  when the caller lacks the section's permission), `cut` and `complete`
  (`TestContactExportIsComplete`, 10,050 emails).
- **DP-3** `/confirm/{token}` asks with one button; only the POST confirms. Links carry their issue
  day (30 days) and the signup they were sent for, whose lists alone they confirm; links without a
  day work while the signup is recent. `optin_sends` and `audience.ClaimOptInSend` hold every send
  path (forms, retries, reminders, the team, imports, partner signups) to one email every 10
  minutes and five a day. IPv6 is keyed by /64; a form flooded past 120 submissions a minute asks
  for Turnstile while the workspace has keys. Board: `PublicConfirmStates.dc.html`.
- **DP-4, DP-6** Settings › General › Data retention (`update-workspace-retention`) and the hourly
  `audience.retention` job: event detail cleared in batches (`TestRetentionClearsDetailKeepsReports`
  compares a campaign report before and after), unconfirmed signups erased through the registry
  (`TestRetentionErasesUnconfirmedSignups`). Board: `WorkspaceSettings.dc.html`.
- **DP-5** `AutomationContactPaused` in the email step (`TestEmailStepsSkipPausedContacts`).
- **DP-7** `metrics.addr` `:9464`, `metrics.token`, `withMigrateLock` around goose and River
  (`TestConcurrentMigratorsRunEachMigrationOnce`).
- Migrations 03360–03363.

Found while building M41 (added to the backlog):

| ID | What | Status |
|---|---|---|
| DP-8 | An import keeps its uploaded file (`imports.file_data`) and its report and skipped-row CSVs, which hold every address it read, until the import is deleted; erasing a contact doesn't reach them. Strip erased addresses from kept files and reports, or expire them after the import finishes. | ✅ M50 |
| DP-9 | Earn › Paid members lists an erased member's ended subscription under the placeholder address; show "Erased contact" instead. | ✅ M50 |

## Engines and events, correct (M42) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| EN-1 | Each automation event and run is processed in its own savepoint. A failing one gets an attempt count and a backoff, is set aside after 5 attempts with a `step_failing` alert, and never blocks the rest; the tick runs even when dispatch fails. | Test: one poisoned run in a batch of 100, the other 99 advance and the alert opens. | M | — |
| EN-2 | Date triggers dedupe on the workspace's local date (or the contact's, AT-7). A yearly trigger defaults to "once per year" and the editor warns when "once" is chosen with "every year". | Tests in Auckland and Sydney: one event per contact per day; a yearly flow fires in its second year. | S | AutomationFlow trigger panel shows the warning (existing problem list) |
| EN-3 | "Opened/clicked within X" starts when the email is sent; while it is queued or held the branch keeps waiting. | Test: a message held for 2 days by a warm-up cap still gets its full window. | S | — |
| EN-4 | "Skip if they got an email in the last N hours" counts only broadcast and automation messages that were actually sent. | Test: a password reset doesn't skip a welcome email. | S | — |
| EN-5 | Automation step statistics from real providers: the ingest normalizer calls `RecordMessageEvent` for automation messages (delivered, bounced, complained) and the unsubscribe path records unsubscribes. Steps report Queued, Sent (on outbox success), Failed and Cancelled separately. Emails that fail to compose (no sender, deleted version, Liquid error) open a `step_failing` alert, and a new notification kind, "An automation needs attention", reaches the bell and email. | A `sendtest` run through a webhook-based provider shows per-step Delivered and Bounced; a compose failure notifies. | M | AutomationsList and AutomationFlow already draw the step numbers; Notifications preferences gain one row |
| EN-6 | Every catalog event is emitted: `email.*` from webhook normalization and the tracking recorder (only human, settled opens and clicks; batched per endpoint so a 1M send doesn't make 1M HTTP calls), `provider.down` and `provider.recovered` from provider health, `automation.published` from publish, `contact.erased` from erase, `campaign.scheduled` when an email campaign is scheduled without approval. | A test fails when a catalog event has no emitter (like `internal/store/writers_test.go`); a send through `sendtest` delivers each `email.*` event to an endpoint. | M | — |
| EN-7 | The small bugs of Fix first #9: Duplicate on sent rows links to `/campaigns/new?from=` like the report; the Source filter reads `/growth/sources`; tag search and filters lower-case their input; referrer search escapes LIKE wildcards (`internal/referrals/referrers.go:91`); starting an import twice is refused with 409 (`internal/store/queries/imports.sql:53-55`); the transactional log matches idempotency keys case-insensitively and finds a batch by its key; the approvals inbox filters status and counts in SQL with keyset paging; Stripe's failed async payment and dispute update are subscribed, recorded, and the buyer is emailed. | One test per bug. | S each | — |

What M42 built (31 October 2026):

- **EN-1** the automation engine dispatches each event and advances each run in its own savepoint;
  a failure counts an attempt and waits (1, 2, 4, 8, 16 minutes), and the fifth sets it aside with a
  `step_failing` alert. The tick job runs dispatch and advance even when one of them fails
  (`TestFailingRunDoesNotStallOthers`; migration 03370).
- **EN-2** date triggers dedupe on the workspace's local date (`TestDateTriggersEastOfUTC`); a
  yearly trigger fires once a year per contact, which the trigger panel now says ("Once a year per
  contact"). Contact-local times stay AT-7.
- **EN-3** "Opened or clicked within" counts from when the email left; while it is queued or held,
  the branch checks again every 15 minutes (`TestBranchWindowStartsWhenTheEmailLeaves`).
- **EN-4** the skip rule counts campaign and automation email that was sent
  (`TestSkipRuleIgnoresTransactionalMail`).
- **EN-5** delivered, bounced, complained and unsubscribed reach a step's numbers from real
  providers' webhooks and the unsubscribe page, once (`TestProviderEventsCountOnAutomationSteps`).
  An email that can't be composed opens a `step_failing` alert, and alerts notify the people who
  edit automations once ("An automation needs attention", bell and email; migration 03371,
  `TestAlertNotifiesOnce`). The per-step Queued and Sent split moved to AT-13, with the step
  analytics it feeds.
- **EN-6** `email.sent`, `delivered`, `deferred`, `bounced`, `complained`, `opened` (with `machine`)
  and `clicked` go to webhooks for campaign and automation mail; `provider.down` and
  `provider.recovered` from provider health; `automation.published` from publishing.
  `campaign.scheduled` was already sent by Review & send. Each emitter first asks whether any
  endpoint listens, so a workspace without webhooks reads nothing more.
  `TestEveryWebhookEventHasAnEmitter` fails when the catalog offers an event no code sends;
  `TestEmailEventsReachWebhooks` follows a bounce to its delivery. Batching per endpoint is EN-8.
- **EN-7** one test each: Duplicate on sent rows (`CampaignsPage.test.tsx`); the Source filter
  reads the workspace's sources, and the two source fields under More filters became one
  (`ContactsScore.test.tsx`); tag search, the tag filter and segments lower-case their input; one
  LIKE escaper (`db.EscapeLike`) for referrer search and the four packages that had their own;
  concurrent import starts (one 200, the rest 409); the transactional log's search ignores case and
  finds a batch by its key; the approvals inbox filters, counts and pages in SQL with a cursor and
  "Show older requests"; Stripe's `checkout.session.async_payment_failed` and
  `charge.dispute.updated` are subscribed, and webhooks registered earlier get them from the hourly
  check (`payments.webhook_events_added`, audited). A failed bank debit marks the product checkout
  failed and emails the buyer through `product-payment-failed`, a starter made beside the delivery
  template and sent once published; dispute updates go on the order's timeline; the payments log
  names product outcomes.

Found while building M42 (added to the backlog):

| ID | What | Status |
|---|---|---|
| EN-8 | `email.*` webhooks are one delivery per event, like the catalog's other events. A 1M send to an endpoint that takes `email.delivered` queues 1M deliveries; batch them per endpoint (one POST with up to 100 events) behind a setting on the endpoint. | ✅ M51 |
| EN-9 | A gift paid by bank debit was never marked paid: Earn didn't route `checkout.session.async_payment_succeeded` to gifts. Fixed with EN-7 (paid when it clears, abandoned when it fails); the gift's buyer isn't emailed about a failed debit, since gifts have no such template yet. | ✅ M51 |
| EN-10 | `product-payment-failed` is made as a draft. Products › a product's publish checks list the delivery template only; add a non-blocking line for the notice so it gets published. | ✅ M50 |

## Audience at 1M contacts (M43) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| AU-1 | Contacts and Suppressions page with keyset cursors instead of OFFSET and count up to 10,000 (D5). | Benchmarks on the 1M seed (OP-9): page 1 and page 400 both under 200 ms. | M | Deviation for the count line |
| AU-2 | Search: `pg_trgm` GIN indexes on email and name, built CONCURRENTLY outside a goose transaction; an exact, indexed match when the query contains "@". | Benchmark on the 1M seed under 150 ms. | S | — |
| AU-3 | Segment filters run as a subquery of the segment's SQL instead of loading member ids into Go; saved-view and health-view counts are cached like segment counts; a `(workspace_id, source)` index; the activity index is rebuilt with `id DESC` to match the query. | `EXPLAIN` tests in CI for each query plan's index use. | M | — |
| AU-4 | Contacts export and bulk "Resend opt-in" run as background jobs with progress in the jobs tray and a download link; bulk resend skips suppressed and already-confirmed contacts and respects DP-3's cooldown. Suppression export honours the filters on screen. | Export of 1M contacts in constant memory; resend reports the real count it sent. | M | Drawn: Feedback (Background jobs) and SystemStates ("Export is taking longer… we'll email the link") |
| AU-5 | "Select all N matching" after selecting a page, and "Export this view"; bulk actions send `allMatching` (the API already accepts it, `internal/audience/bulk.go:27-32`). | UI test: an action on all matching reaches contacts on later pages. | S | Extend Contacts' bulk bar with the banner |
| AU-6 | Bulk and profile actions: remove from list, remove tag (recorded with source "team", not as the contact's own choice), and the same on the public API. | Tests, audited. | M | Extend Contacts' bulk bar and ContactProfile's lists card |
| AU-7 | A contact's email address can be corrected: uniqueness check, suppression check of the new address, a consent event, an audit entry, the mailbox provider looked up again. | Test: history and consent stay with the contact. | M | New state on ContactProfile (edit email) |
| AU-8 | Saved views: rename, update to the current filters, delete (a ⋯ menu per view; a PATCH endpoint). | Tests. | S | New: menu on Contacts' view chips |
| AU-9 | Columns and sorting: sort by joined, last open, email, score; a column chooser including custom fields, saved with the view. | Tests for each sort key's index use. | M | New: column chooser on Contacts |
| AU-10 | Filters: joined between two dates in the workspace's time zone; contacts from one import, with "View these contacts" on the import report; imports can tag every row. | Tests. | S | Extend Contacts' filter row and MigrationDone |
| AU-11 | Lists and fields management: create and rename lists, set a list's opt-in mode, create, edit and delete custom fields (the APIs exist, `internal/audience/api.go:235-274`). Deleting a field or a segment first shows what uses it (segments, forms, scoring rules, Liquid tokens, scheduled campaigns, automations, digests, saved views); fields are soft-deleted and their values stripped in a background job. | Impact tests for fields and segments; restore within 30 days. | M | New: a Lists & fields tab under Audience; reuse Dialogs' delete pattern and MigrationFields' "New field" |
| AU-12 | List health: the sunset policy can be switched off and its numbers edited (the API accepts them); turning on "delete after N months in Sunset" first shows a dry run (how many, from which date) and asks for a typed confirmation. | Tests for the dry run. | S | Extend ListHealth's policy card |
| AU-13 | Verify addresses from Contacts (selection or view), the segment builder and a flagged growth source (audience insights brief §3.2, Bulk; deferred by inventory #255). | UI tests open the existing verify dialog with each target. | S | Extend Contacts, SegmentBuilder and GrowthStates with the button |
| AU-14 | Suppressions: remove a whole upload group (with a reason, audited). | Test: undoing an accidental CSV upload. | M | Extend Suppressions' group row |
| AU-15 | Opt-in emails sent from the app (Add contact, Resend opt-in, bulk resend, import re-permission) use the forms' branded, localized confirmation email (`forms.ConfirmationEmailIn`) with the names of the pending lists. | Test in English and Vietnamese. | S | — |
| AU-16 | Segments keep the time their members entered and left (a diff when `segment_members` is refreshed), the groundwork for AT-10. | Test: enter and leave events on a refresh. | M | — |

Riskiest part: AU-1 and AU-2 on large tables (index builds must not lock; the count line changes,
D5). AU-5 depends on AU-4's jobs.

What M43 built (31 October 2026):

- **AU-1** Contacts and Suppressions page with keyset cursors in every order (last activity, score,
  joined, last open, email); each order has an index ending in the id. Counts stop at 10,000 and
  the line reads "1 to 50 of 10,000+ · Count all"; Count all asks for the exact number
  (`TestContactsPageWithCursors`, `TestContactCountStopsAtTenThousand`; migration 03372). The 1M
  seed benchmarks wait for OP-9's seed (AU-17); `TestContactQueriesUseIndexes` checks each plan's
  index use in CI.
- **AU-2** trigram GIN indexes on email and name, built concurrently; a whole address matches
  through the unique index (`TestContactQueriesUseIndexes`).
- **AU-3** a segment filters Contacts as a subquery; SQL segments, which run as their own role,
  still resolve first (`TestSegmentFilterIsASubquery`). Saved-view counts are cached for ten
  minutes on Contacts and List health (`TestSavedViewsKeepCountsAndChange`; migration 03373).
- **AU-4** exports of any size and bulk Resend opt-in run as background jobs in batches, followed
  in the jobs tray; an export is kept 7 days for whoever started it, and a resend skips the
  confirmed, the suppressed and DP-3's cooldown and says how many of each
  (`TestExportRunsInTheBackground`, `TestBulkResendReportsWhatItDid`; migration 03374).
- **AU-5** after a whole page is selected, "Select all N matching" reaches every contact the view
  matches, counted exactly first; "Export this view" (`ContactsPage.test.tsx`).
- **AU-6** Remove from list and Remove tag, in bulk and on the profile, recorded as the team's
  change (consent action `removed`), never as an unsubscribe (`TestRemoveFromListAndTag`;
  migration 03375).
- **AU-7** a contact's address can be corrected unless another contact has it or it is
  suppressed; the history keeps the old one (`TestCorrectAnAddress`).
- **AU-8** saved view chips have a menu: rename, save the current filters and columns, delete.
- **AU-9** a column chooser with custom fields, saved with the view; Score, Last open, Joined and
  Contact sort.
- **AU-10** joined between two days in the workspace's time zone, the contacts of one import
  ("View these contacts" on the import report), and a tag the import puts on every row
  (`TestJoinedBetweenAndImportFilters`).
- **AU-11** Audience › Lists & fields: lists (create, rename, opt-in mode, delete) and custom
  fields with how many contacts have a value and what uses each (segments, forms, scoring rules,
  automations, unsent campaigns, digests, saved views, Liquid; `internal/dependents`). Deleting a
  field or a segment lists those uses first. A deleted field keeps its values for 30 days and can be
  restored as it was; then the hourly pass strips them (`TestFieldsAndSegmentsShowWhatUsesThem`,
  `ListsFieldsPage.test.tsx`; migration 03376).
- **AU-12** List health's policy card has a switch and takes the days before Broadcasts stop and the
  months before erasing. Turning erasing on, or erasing sooner than before, first shows the dry run
  (how many contacts, from which night, how many paid members are kept) and takes the typed word
  ERASE (`TestSunsetErasingNeedsTheDryRun`, `SunsetPolicy.test.tsx`).
- **AU-13** Verify addresses from a Contacts selection or view, a saved segment and a flagged growth
  source, all through the one verify dialog (`TargetVerify.test.tsx`).
- **AU-14** a CSV upload of suppressions is one group on the list, with who uploaded it, and comes
  off whole with a reason, as removing each address would; one audit entry
  (`TestRemoveASuppressionUpload`, `RemoveUpload.test.tsx`; migration 03377).
- **AU-15** the team's opt-in emails (Add contact, Resend, bulk resend, import re-permission) are the
  forms' branded, localized email naming the pending lists (`TestTeamOptInEmailIsTheBrandedOne`).
- **AU-16** a static segment's refresh is a diff: members keep the time they entered, and each entry
  and exit is recorded (`TestSegmentsKeepWhenMembersEnteredAndLeft`; migration 03378).

Found while building M43 (added to the backlog):

| ID | What | Status |
|---|---|---|
| AU-17 | The 1M-seed benchmarks of AU-1 and AU-2 (page 1 and page 400 under 200 ms, search under 150 ms) need OP-9's seed; run them with M47. | ✅ M47: 6 ms, 7 ms and 23 ms (`TestAudienceAtOneMillion`) |
| AU-18 | What-uses-it checks read segments, forms, scoring, automations, unsent campaigns, digests and saved views, but not transactional templates, SMS and WhatsApp templates or a form's own confirmation email; a field used only there reads "Nothing". Add them to `internal/dependents`. | ✅ M50 |
| AU-19 | A saved view has no address of its own, so "Saved view · Name" in a what-uses-it list can't open it; Contacts could take `?view=`. | ✅ M50 |

## Automations, complete (M44) ✅

Depends on M42.

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| AT-1 | Delete and duplicate a flow from the list, and discard a draft in the editor (the endpoints exist, `internal/automations/api.go:269-299`, the web never calls them). | UI tests; deletion of a live flow asks what happens to contacts in it. | S | New: AutomationsList row menu; AutomationFlow header gains Discard draft |
| AT-2 | Who is at a step: a paged list from each step, "Remove from flow" (audited), and an Automations card on the contact's profile with the flows they are in and where. | Tests, audited. | M | New: contacts-at-step drawer on AutomationFlow; Automations card on ContactProfile |
| AT-3 | Test send and preview as a contact for an automation email (reusing the campaign test-send path). | Test send through the System route. | S | Extend AutomationFlow's email panel (Preview is drawn; add Send test) |
| AT-4 | Resuming a paused flow asks what to do with overdue delays: send now, skip overdue emails, or spread over N hours, within warm-up caps and quiet hours. | Test for each choice. | M | New: resume dialog (reuse Dialogs' "Resume sending" and AutomationPublish's waiting-contacts copy) |
| AT-5 | Publishing with "move contacts at their current step" updates runs set-based per node mapping (or in a background job over 10,000 runs), not row by row inside the request (`internal/automations/publish.go:729-772`). | Publish with 200,000 waiting contacts under 5 s. | M | — |
| AT-6 | Re-entry "once every N days" for every trigger (the score trigger already has `OnceInDays`). | Tests. | S | Extend the trigger panel |
| AT-7 | Date triggers can fire at the contact's local time when the contact has a time zone. | Test across two time zones. | S | Extend the trigger panel |
| AT-8 | "Exit when they leave the trigger list", and a finished run's state (trigger payloads such as order details) is cleared after 90 days. | Tests. | S | Extend the trigger panel |
| AT-9 | A workspace marketing cap: at most N emails in X days from campaigns and automations together; automation emails over the cap wait or skip per step; campaigns show the excluded count in Review & send. | Tests; the Review & send waterfall shows the line. | M | Settings › Sending rules gains the cap; SendReadiness waterfall gains a line |
| AT-10 | New triggers built on existing data: "enters or leaves a segment" (AU-16) and "clicked a link" in a chosen campaign or email (settled human clicks only). | Tests through `sendtest`. | M | Extend the trigger picker |
| AT-11 | Up to 3 triggers per flow, any of which enrols (OR), and an "Exit here" step on a branch that ends the run there while other branches continue. | Tests. | M | Extend AutomationFlow (trigger stack, exit step) |
| AT-12 | A live flow's trigger can be changed by publishing a new version; the publish diff shows the change and who is affected. | Test. | S | Extend AutomationPublish's diff |
| AT-13 | Sequence analytics: each email step shows delivered, human opens, clicks, unsubscribes and drop-off to the next email, on the numbers EN-5 makes real. | UI test against production-shape data. | M | Extend AutomationFlow's step stats |

Riskiest part: AT-4 (overdue delays must respect warm-up caps and quiet hours) and AT-9 (one cap
read by two senders without double counting).

What M44 built (31 October 2026):

- **AT-1** the Automations list has a row menu (Open, Duplicate, Delete…); deleting a flow that went
  live says how many contacts are in it, asks for its name and offers Pause instead. The editor's
  header has Discard draft, back to the live version (`complete.test.tsx`).
- **AT-2** a step's Stats panel opens the contacts at the step, the latest first, 25 at a time with
  Show more; each can be removed from the flow with a reason, logged as
  `automation.contact_remove` and counted as "left early" at the step. The contact profile has an
  Automations card with the flows the contact is in or went through and where
  (`TestContactsAtAStepAndRemovingOne`; migration 03380 indexes the runs by step and by contact).
- **AT-3** an email step can be previewed as a contact and sent as a test through the System route,
  "[Test]" in the subject, no stats; without the send permission only to one's own address, as
  campaigns (`TestPreviewAndTestSendAnAutomationEmail`).
- **AT-4** resuming a flow whose contacts' waits ended during the pause asks: send now (warm-up caps
  and quiet hours apply as always), skip the email they missed, or spread them over 1 to 72 hours
  (`TestResumeWithOverdueContacts`).
- **AT-5** publishing with "move" moves contacts a step at a time, one statement per version and
  step: 20,000 waiting contacts move in well under five seconds (`TestPublishMovesManyContactsQuickly`).
- **AT-6** re-entry "once every N days" for every trigger (`TestMoreTriggersAndReentryAfterDays`).
- **AT-7** a date trigger can fire at each contact's own time, the workspace's for contacts without
  a time zone (`TestDateTriggersAtTheContactsTime`).
- **AT-8** a Joined list flow can let contacts go when they leave the list: the database wakes their
  runs and the engine ends them with the step's numbers; a finished run's trigger data is cleared
  after 90 days (`TestLeavingTheListEndsTheRun`).
- **AT-9** Settings › Sending rules › Marketing cap: at most N campaign and automation emails in X
  days per contact. Campaigns leave out contacts over it (Review & send's audience shows the line;
  the send excludes them as "capped"); an automation email waits until the contact is under it, or
  is skipped, as its step says (`TestMarketingCap`; migration 03379).
- **AT-10** "Enters a segment" and "Leaves a segment": dynamic segments a live flow listens to are
  evaluated every 10 minutes, the first time only recording members; static ones on refresh.
  "Clicked a link" starts on a person's settled click in a campaign or an automation email, the link
  address optionally containing some text (`TestSegmentAndLinkTriggers`).
- **AT-11** a flow starts on up to three triggers, any of which enrols: the trigger card holds the
  stack. The second and third are the event kinds (list, form, tag, API event, segment, link); date,
  score, store, product and milestone triggers keep their own entry rules and stay first. The Exit
  step already ended one path while the others go on. `automations.triggers` lists every trigger so
  "does a live flow listen to this?" reads one indexed array.
- **AT-12** a live flow's trigger changes by publishing; the publish diff says new contacts enter on
  the new trigger from the moment the version is live and how many already in the flow aren't
  affected.
- **AT-13** the flow's Stats panel lists the sequence: each email's delivered, human opens, clicks,
  unsubscribes and the share that went on to the next email (`TestSequenceFollowsTheEmails`).

Also fixed: `TestClickConditionsCountPeopleOnly` never marked its emails sent, so since EN-3 (M42)
its branch waited forever; it now does.

Found while building M44 (added to the backlog):

| ID | What | Status |
|---|---|---|
| AT-14 | The welcome prompt, Scoring's list of flows, the products and pages helpers and score ingest still read a flow's first trigger: a flow whose second trigger joins a list isn't that list's welcome series. Read `automations.triggers` there too. | ✅ M50 |
| AT-15 | Watched segments are all evaluated every 10 minutes; a slow SQL segment delays the others. Time each one and back off the slow ones, with a note on the trigger. | ✅ M51 |

## Campaigns and everyday sending (M45) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| CA-1 | Delete drafts and cancelled campaigns (restorable for 30 days, audited, open approvals closed) and archive sent ones (D4), on every channel. | Tests; reports, web posts and retries still open an archived campaign. | M | New: Campaigns row menu and confirmation (Dialogs pattern) |
| CA-2 | Campaigns list for many campaigns: select and act in bulk (archive, delete, tag), tags, filters by date range, sender and audience, search in name and subject, Unschedule from the row. | Tests. | M | New: Campaigns filters, tags and bulk bar |
| CA-3 | A calendar of scheduled and sent campaigns and digest issues by week and month, with drag to reschedule a scheduled campaign (re-running Send Readiness's send-time recheck). | Tests; rescheduling is audited. | L | New board: Campaigns › Calendar |
| CA-4 | Follow-ups from a report: "Resend to non-openers" (human opens only, after a chosen delay, new subject) and "Send to clickers", as a draft with that audience (the `opened_campaign` condition exists, `internal/segments/definition.go:88`). | Test: the follow-up's audience excludes privacy-protected opens per the report's buckets. | M | New: follow-up dialog on Report (nearest pattern: ReportStates' "Retry after fixing") |
| CA-5 | Send at the same local time in each recipient's time zone (contacts without one get the workspace's), as SMS already does (`LocalMinute` in `internal/texting/campaigns.go`); "Best time" stays. | Tests across time zones; the estimate covers 24 hours. | M | Extend SendReadiness's schedule control (pattern: SmsReview's local time) |
| CA-6 | Cadence (D1): campaigns carry an audience cadence; the audience build, estimates, approval snapshots and reports exclude readers whose choice doesn't match; Review & send's waterfall shows the count. | Tests: a "Monthly best-of" reader receives only monthly digests and announcements. | M | Extend the composer's Sending-to drawer and SendReadiness's waterfall; PublicPreferences already draws the choices |
| CA-7 | The campaigns list selects the page of ids first and joins afterwards; Overview's "Recent" uses a light query instead of the full list twice (`internal/overview/recent.go:19-25`). | Benchmark with 10,000 campaigns. | M | — |
| CA-8 | Saved templates and sections: rename, replace with the current email, delete, and search in the gallery (`internal/templates/api.go:95-101` exists, unused). | UI tests. | S | Extend ComposerEmpty's gallery (search, tile menu; BrandKit's Edit/Delete pattern) |

Riskiest part: CA-6 changes audience counts everywhere they appear; CA-1 must leave every reference
to a sent campaign working.

What M45 built (1 November 2026):

- **CA-1** a campaign row's ⋯ menu deletes a draft, or a cancelled or failed campaign that sent
  nothing: it goes to a Deleted tab, restorable for 30 days (a toast offers Undo), its open approval
  request closes and an approval-waiting draft is a plain draft again; maintenance removes it 30
  days on. A sent campaign is archived instead, never deleted: it leaves the list for an Archived tab
  and its report, web post and retries stay. Every channel, each change audited
  (`campaign.delete`, `campaign.restore`, `campaign.archive`, `campaign.unarchive`;
  `TestDeleteArchiveRestore`; migrations 03381–03382).
- **CA-2** rows can be selected (page by page) for a bulk bar: add or remove a tag, archive, delete
  or unarchive; each campaign is done on its own and those that can't are skipped with why. Tags
  are lower case and show after the name. The list filters by when it went or goes out (7 or 30
  days, this quarter, custom), sender, a list of its audience and tag; search reaches subjects, with
  LIKE's wildcards taken literally. A scheduled row has Unschedule (`TestBulkTagsAndFilters`,
  `manage.test.tsx`).
- **CA-3** Campaigns › Calendar shows scheduled and sent campaigns and digest issues to come by
  month or week (a list on phones). A scheduled email can be dragged to another day or moved with
  Reschedule (the keyboard's way); the move is audited as `campaign.rescheduled` and Send Readiness
  checks it again before it goes out. An approved time is part of its approval and isn't moved
  (`TestCalendar`, `TestReschedule`).
- **CA-4** a sent campaign's report has Follow up: resend to people who didn't open (opens through
  privacy proxies alone count as unknown and are left out, as the report counts them) or send to
  people who clicked (scanners' clicks don't count), as a draft with the same content, a subject of
  its own and a suggested time one or two days after the original. Its audience is a dynamic segment
  on two new conditions, Received campaign and Clicked campaign, and Opened campaign gains "didn't
  open at all" (`TestFollowUp`).
- **CA-5** Review & send can send at the same time of day in each recipient's time zone, the chosen
  zone for contacts without one. The send starts when that time first comes (UTC+14) and each
  recipient waits for theirs; the estimate and its timeline cover about 26 hours
  (`TestSending/LocalTime`, `TestEstimateLocalTime`, `TestScheduleLocalAt`).
- **CA-6** a campaign is a regular issue or an announcement (the composer's Sending to drawer, with
  how many readers chose another rhythm). A regular issue reaches "Every issue" readers; a weekly or
  monthly digest issue also reaches readers who chose that rhythm; an announcement reaches everyone.
  The audience build, Send Readiness's counts (a line in the waterfall), the approval snapshot and
  so the reports follow (`TestSending/Cadence`, `TestAssembleCadenceLine`, `internal/cadence`).
- **CA-7** the campaigns list chooses the page of ids first and joins its numbers to them; Overview's
  Recent is one page of everything that went out, without the list's totals
  (`TestOverviewRecentIsOneLightPage`, `TestListTenThousandCampaigns`).
- **CA-8** the composer's Templates tab searches templates and sections by name; a saved one's ⋯ menu
  renames it, replaces a template with the email open now, or deletes it.

Also fixed: two operations never received their campaign id, because huma reads path parameters
only through exported embedded fields (`reschedule-campaign` embedded an alias, the follow-up input
an unexported type); the composer's audience drawer grew taller than the screen with many lists and
segments, and now scrolls with Apply in view.

Found while building M45 (added to the backlog):

| ID | What | Status |
|---|---|---|
| CA-9 | The calendar moves email campaigns only: texts, push and WhatsApp keep their own schedule. Give them the same reschedule. | ✅ M51 |
| CA-10 | Calendar items don't say whether an approval covers the time, so Reschedule on an approved campaign is refused only after the dialog. Carry the approval state and offer "Unschedule and ask again". | ✅ M50 |
| CA-11 | A test that every operation's path parameters reach its handler, so an input embedding an unexported or aliased type fails CI instead of answering 404. | ✅ M50 |
| CA-12 | The status tabs count the whole workspace while filters are on; count within the filters, or say so. | ✅ M50 |

## Transactional, approvals, polls and integrations (M46) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| TX-1 | Cancel or reschedule a message scheduled with `send_at`, by API and on the message page, audited, with a `message.cancelled` webhook event in the Transactional messages group. | Tests. | S | Extend TransactionalMessageStates with scheduled and cancelled |
| TX-2 | "Remove from the suppression list, with a reason" on the message detail (spec 5.6; already drawn). | Test, audited. | S | Drawn: TransactionalMessageStates |
| TX-3 | The 10-second live poll of the message log asks only for newer messages and reads status counts from a per-minute cache. | Benchmark at 1M messages a day. | S | — |
| TX-4 | The per-key send rate limit is shared by every web node (a Postgres token bucket) instead of per process (`internal/transactional/ratelimit.go:12-16`). | Test with two nodes. | M | — |
| TX-5 | "Send test" for a template draft, with a chosen test data set (inventory #151 is revisited: the board draws it). | Test through the System route. | S | Drawn: TransactionalTemplate |
| TX-6 | The SMTP relay throttles authentication failures per IP and caps connections. | Tests. | S | — |
| AP-1 | The approvals inbox re-checks only stale requests on load and leaves the rest to the background job; search by campaign name. | Benchmark with 1,000 open requests. | M | — |
| AP-2 | Survey results and text-answer search aggregate and page in SQL; poll comments, text answers and responses show an error state when the request fails instead of "No comments". | Tests. | M | Extend PollResults and survey states with the error |
| IN-1 | Rotating an outgoing webhook's secret signs with both secrets for 24 hours (as community adapters do). | Test: a receiver checking the old secret keeps working for 24 hours. | S | Extend the webhook endpoint page with the overlap note |
| IN-2 | Endpoints that start failing or dead-letter deliveries notify the workspace ("A webhook endpoint is failing"). | Test. | S | Notifications preferences gain one row |
| IN-3 | MCP read tools: `get_contact`, lists, forms, automation detail and report, the approvals queue, offers; `propose_send` for SMS, push and WhatsApp like email's `schedule_campaign`. | Tests per tool; permissions checked like the REST operations. | M | — |
| IN-4 | The in-app API reference links to the instance's own `/api/openapi` and a self-hosted reference page, not GitHub's main branch. | Test: the link carries the running version. | S | — |

What M46 built (1 November 2026):

- **TX-1** a message sent with `send_at` can be cancelled or moved until it goes, by API
  (`cancel-transactional-message`, `reschedule-transactional-message`, keys that can send) and on
  its detail page (Reschedule…, Cancel send…). A cancel removes the sealed payload, keeps the message
  in the log as Cancelled with who did it, and sends `message.cancelled` to webhooks; both changes
  are audited. The log shows Scheduled and Cancelled with their own counts. Failover no longer moves
  a message before its send time (`TestCancelAndRescheduleScheduled`; migration 03383).
- **TX-2** a message rejected for a suppressed address offers "Remove from suppression list…" with
  a reason (audited) and "Open suppression" where the rejection is explained (`schedule.test.tsx`).
- **TX-3** the log's live updates ask every 10 seconds for newer messages only and put them on top;
  the status counts are kept for a minute (`TestLogLivePoll`).
- **TX-4** each key's send rate is a token bucket in Postgres, so every web node takes from the same
  tokens; idle buckets go with the transactional retention job (`TestTakeTokensSharedAcrossNodes`).
- **TX-5** the template editor's Send test sends the draft, filled with a test data set, through the
  System route with "[Test]" in the subject; it never reaches the log. Without transactional:send,
  only to one's own address (`TestSendTemplateTest`).
- **TX-6** the SMTP relay refuses connections over 500 per node and 20 per address with 421, and an
  address with five failed logins in ten minutes is refused for fifteen (`TestGuardBlocksRepeatedFailedLogins`,
  `TestLimitListenerCapsConnections`).
- **AP-1** the approvals inbox re-checks on load only the requests that may be stale (never checked,
  their campaign changed since, or checked more than ten minutes ago); the background job keeps the
  rest. It searches by name (`TestInboxSearchAndStaleChecks`; `approval_requests.revalidated_at`).
- **AP-2** survey results and text answers are counted, searched and paged in SQL; poll comments,
  text answers and responses say when they failed to load, with Try again.
- **IN-1** a rotated webhook secret keeps signing beside the new one for 24 hours:
  `X-ReputeCast-Signature` holds both, comma-separated, the new first (`TestRotationKeepsTheOldSecretSigningADay`).
- **IN-2** an endpoint failing for an hour, or one whose delivery was given up on, tells the people
  who manage webhooks once per failing spell (notification "A webhook endpoint is failing").
- **IN-3** MCP tools `get_contact`, `list_lists`, `list_forms`, `get_automation` (with the Stats
  panel's numbers), `list_approvals`, `list_offers` and `propose_send` for texts, push and WhatsApp,
  each behind its REST operation's permission (`TestM46Tools`).
- **IN-4** the instance serves its own API reference at `/api/reference`, rendered from its OpenAPI
  document with no outside assets; the app's links open it with the running version (`TestAPIReference`).

Also fixed: Send Readiness's address checks matched the audience's addresses with `= ANY`, which made
Postgres probe the trigram index once per contact: minutes for a 40,000-recipient audience (the
approvals suite timed out on it). The addresses now drive a join through `(workspace_id, email)`:
45 ms for 10,000. CA-6's waterfall line now sits before the sunset lines, in the order the send
excludes.

Found while building M46 (added to the backlog):

| ID | What | Status |
|---|---|---|
| TX-7 | Scheduled messages can be moved one at a time; a batch sent with one send_at can't be cancelled together. Cancel or move a batch by its idempotency key. | ✅ M51 |
| AP-3 | The approvals inbox's 1,000-request benchmark: the stale-only re-check is tested for correctness, not timed. | ✅ M51 |
| OP-x | `TestUsageAndRebill` counts a workspace added in the first hours of a month (in the agency's time zone) in the month before; pin its clock. | ✅ M50 |

## Operations (M47) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| OP-1 | Metrics: River queue depth, lag and failures by kind; sends by provider and result; webhook ingest lag; outgoing webhook deliveries; database pool. | A test scrapes each metric name. | M | — |
| OP-2 | `reputecast healthcheck` for containers; a worker-only process answers health on the metrics listener; `/readyz` checks the schema version; the deploy compose file uses them. | Tests. | S | — |
| OP-3 | The binary refuses to serve (or serves read-only with the maintenance banner) when migrations are pending. | Test. | S | Drawn: SystemStates' read-only banner |
| OP-4 | Backups: a restore runbook (including the `reputecast_segments` role, which `pg_dump` doesn't carry), a downgrade policy, and `reputecast doctor` that checks the secrets key, the files folder and the roles. | Runbook tested once on a fresh host. | S | — |
| OP-5 | `reputecast secrets reseal` re-encrypts every stored secret with the current key so old keys can be dropped (`NeedsRotation` has no caller, `internal/secrets/secrets.go:121`). | Test. | S | — |
| OP-6 | Rate limits for signed-in browsers by session, not by IP (a team behind one office NAT shares 600 requests a minute today); per-replica limits documented. | Test. | S | — |
| OP-7 | Logs carry the route pattern, workspace, caller (user, key or agent) and the request id in every service log line of a request. | Test on the log handler. | S | — |
| OP-8 | A plan, then a migration, for monthly partitions of `delivery_events`, with DP-6 dropping detail by partition. | Migration tested on the 1M seed. | L | — |
| OP-9 | Load testing: a 1M-contact seed, k6 scripts (contacts list, segment counts, send throughput, webhook ingest) and Go benchmarks for the hot queries; numbers recorded in `docs/deploy`. | Run in a manual CI job. | M | — |
| OP-10 | Instance admin › System: failed and discarded jobs (retry, discard), queue lag, database and largest tables' sizes, the files folder's size. | Tests. | M | New board: InstanceSettings › System |

What M47 built (1 November 2026):

- **OP-1** every process exports job queue depth, due-job lag and failures by kind
  (`reputecast_jobs`, `reputecast_job_queue_lag_seconds`, `reputecast_job_failures_total`), messages
  and calls to providers by adapter and result, webhook ingest requests and lag, outgoing webhook
  deliveries and lag, and the database pool. Queue readings come from the database at most every
  15 seconds (`TestOperationalMetricsAreScraped`).
- **OP-2** `reputecast healthcheck` asks the local `/readyz` (the HTTP listener, then the metrics
  listener); `/readyz` checks the database, the schema and, in a worker, that workers run; a
  worker-only process answers health on the metrics listener. The Dockerfiles and
  `deploy/docker-compose.yml` use it (`TestHealthcheck`).
- **OP-3** a binary whose migrations are pending serves read-only: reads carry
  `X-ReputeCast-Read-Only`, from which the web app shows the maintenance banner before any write;
  writes answer 503 `maintenance`; provider webhooks are still stored; workers wait; `/readyz` is 503.
  It carries on by itself once the migrations land (`TestPendingMigrationsServeReadOnly`).
- **OP-4** `reputecast doctor` checks the database, the schema, that the key opens a sample of every
  kind of sealed value, the files folder and the segments role; `reputecast migrate segments-role`
  hands the segments function back to its role after a restore. The runbook
  (`docs/deploy/backup-restore.md`) was tested on a fresh Postgres 16 cluster (`TestDoctor`).
- **OP-5** `internal/sealed` lists all 47 sealed values with their associated data;
  `reputecast secrets reseal` re-encrypts them with the current key, only where the value is still
  the one read, and is audited. A test fails on any bytea column that is neither listed nor exempt
  (`TestEverySealedColumnIsKnown`, `TestResealMovesEveryValueToTheCurrentKey`). Links in sent emails
  were signed with the current key only, so a rotation broke every unsubscribe link already sent:
  they now verify with the previous keys too (`TestPreviousKeysStillVerify`).
- **OP-6** signed-in browsers are limited per session (600 a minute) after sign-in is checked, and
  together per address at 6,000 (`http.rate_limit_ip`); limits per replica are documented
  (`TestAPILimitsEachSession`, `TestRateLimitsBrowsersShareAHigherCap`).
- **OP-7** a request's log lines carry `request_id`, `route`, `workspace` and `caller`
  (`user:<id>`, `key:<id>`, `agent:<id>`), the access line included; a job's carry `job`
  (`TestRequestLinesCarryTheRequest`).
- **OP-8** `reputecast migrate partition-events` splits `delivery_events` by month on `received_at`
  without copying it; dedupe keys move to `delivery_event_keys`; months are made ahead hourly; DP-6
  clears a month's detail in one pass once it is past a cutoff. The plan is
  `docs/deploy/partitioning.md`; on the 1M seed the conversion took 5 seconds and every count read the
  same (`TestPartitionDeliveryEvents`; migration 03384).
- **OP-9** `reputecast seed --sample load` (a million contacts, 3.2 million events), Go tests and
  benchmarks of the hot calls, k6 scripts for contacts, segment counts, sends and ingest, and a manual
  "Load test" workflow. Numbers in `docs/deploy/performance.md`.
- **OP-10** Instance admin › System (new board InstanceSystem): failed jobs with their last error,
  retried, cancelled or deleted (audited), queues and their lag, the largest tables and the files
  folder (`TestSystemTab`, `SystemPage.test.tsx`).

Found while building M47 (added to the backlog):

| ID | What | Status |
|---|---|---|
| OP-11 | Convert `delivery_events` at upgrade in a later release, once instances have run the opt-in command; then drop the legacy unique dedupe index. | ⏳ later release |
| OP-12 | A segment count (about a second for two conditions on a million contacts) and the audience header counts (about 300 ms) read every contact: a count cache. | ✅ M51 |
| OP-13 | Record the k6 numbers from a run of the Load test workflow in `docs/deploy/performance.md` (k6 couldn't run where these numbers were taken). | ✅ M51 (run 41353963642; search under load and the send test followed up) |

## Frontend foundation (M48) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| FE-1 | A command palette on ⌘K, from every page: campaigns, contacts, automations, lists, segments, forms, pages and settings, filtered by the person's permissions, with recent items. Overview's search says what it really searches. | UI tests; results never show what the role can't open. | M | New board: command palette (Feedback's shortcut sheet already promises "Search everything") |
| FE-2 | After an upgrade, a tab that fails to load a lazy page (`vite:preloadError`, old chunks are 404, `internal/server/spa.go:64-66`) reloads once; a banner offers "Reload" when the server's version changes. | Test. | S | Extend SystemStates with a reload-after-upgrade banner |
| FE-3 | Shared components where features rebuilt them: `ui/Stepper` (8 wizards), `ui/Pager` (about 12 lists), `lib/useDebounced` (5 copies), switches through `ui/Toggle` (41 files), `NoResults` for every search and filter (3 of 31 screens today), one bulk-selection pattern for lists. | No duplicates left (a lint check for the known patterns). | M | Design system already defines them |
| FE-4 | Tables: sticky headers inside `TableScroll`; card-level error boundaries in `Card`, `StatTile` and chart wrappers so one failing card doesn't take down a page. | UI tests. | S | — |
| FE-5 | A lint rule against hex colours in TSX outside the brand kit and email previews; the 54 copies of the accent and the status colours move to tokens. | Lint passes with no exceptions added. | M | Tokens exist in `design/design-system/tokens.json` |
| FE-6 | One validation pattern for forms and the unused charts dependency, after the owner's answer (D6). | Decision recorded in `docs/TECH_STACK.md`. | S | — |
| FE-7 | Optimistic updates for toggles, marking notifications read and reordering. | UI tests with a failing request roll back. | S | — |
| FE-8 | CI runs a Playwright smoke test and the axe sweep (`web/scripts/a11y.ts`) against a seeded server, and enforces a bundle size budget. | CI job green. | M | — |

What M48 built (1 November 2026):

- **FE-1** ⌘K (Ctrl K) opens the command palette from every page (new board `CommandPalette`). One call,
  `search-workspace`, searches campaigns, contacts, automations, lists, segments, forms and pages by
  name, five a kind, only the kinds the role can open and only a list-scoped member's own contacts and
  lists. Places to go are filtered by permission; Recent (the last eight opened, in the browser) leaves
  out kinds the role can no longer open. It is an ARIA combobox with a listbox, ↑ ↓ ↵ esc and a polite
  results count. Overview's search box says it searches campaigns and provider events
  (`TestSearch`, `CommandPalette.test.tsx`, the smoke test).
- **FE-2** every answer carries `X-ReputeCast-Version`; a tab that hears a new one shows
  "Updated · 1.6 · Reload" (SystemStates extended), and a tab whose lazy chunk is gone after an upgrade
  reloads itself, at most once a minute (`systemStatus.test.ts`).
- **FE-3** in part: `SwitchInput` replaces 42 hand-made switches and the five copies of `useDebounced`
  import `lib/hooks`; `pnpm lint` now runs `scripts/patterns-check.ts`, which refuses both patterns
  coming back. Stepper, Pager, NoResults everywhere and one bulk-selection pattern are FE-3b below.
- **FE-4** `Card`, `StatTile` and the growth, revenue, placement and score charts fail on their own: a
  render error shows "This part couldn't show · Retry" in place and the page works. `TableScroll`
  takes `maxHeight`, and then its header row stays in view (`ErrorBoundary.test.tsx`).
- **FE-5** in part: the same check counts hex colours per file against `scripts/hex-baseline.json`
  (1,166 in 206 files today) and refuses any increase; moving them to tokens is FE-5b.
- **FE-6** the question is recorded in `docs/TECH_STACK.md` with the measurements (recharts imported
  nowhere, react-hook-form in one file, zod in 67 for route search) and a recommendation; nothing is
  removed until the owner answers.
- **FE-7** in part: marking a notification read shows at once in every list and the bell, and a
  refused request puts it back (`NotificationsPopover.test.tsx`). Toggles and reordering are FE-7b.
- **FE-8** a CI job builds the app, checks the bundle budget (`pnpm budget`: first load, each chunk,
  all scripts, gzipped), seeds a server and runs the Playwright smoke test (sign-in, seven screens on
  desktop and phone, ⌘K) and the axe sweep (`web/e2e/smoke.spec.ts`; passed locally).

Found while building M48 (added to the backlog):

| ID | What | Status |
|---|---|---|
| FE-3b | `ui/Stepper` for the eight wizards, `ui/Pager` for the paged lists, `NoResults` on every search and filter, one bulk-selection pattern; the patterns check learns each as it lands. | ✅ M51 |
| FE-5b | Move the 1,166 hex colours in TSX to tokens, file by file, lowering `hex-baseline.json` to zero outside the brand kit and email previews. | ✅ M51, with the exceptions listed below |
| FE-7b | Optimistic switches (with a shared hook) and reordering, rolled back on a refused request. | ✅ M51 |
| FE-9 | The first load is 433 KB gzipped: every route's module (its search schema) is preloaded. Split the route tree so a page loads only its own. | ✅ M51 |
| FE-10 | The command palette searches settings by tab name only; the board shows single settings ("Send weekdays only"). An index of settings rows. | ✅ M51 |

## Earn and other channels (M49) ✅

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| EA-1 | Unsaved-changes guards on the tier, offer and upgrade page editors (`UnsavedChangesGuard`, used by about 25 other editors). | UI tests. | S | — |
| EA-2 | Audited CSV exports for gifts, sponsor bookings and reports, conversions and the payment event log, like paid members and orders. | Tests. | M | Extend each list's header with Export |
| EA-3 | Every Earn chart has a table view (design system rule; Growth's toggle is the pattern); the paid members list has the page-size picker. | UI tests. | S | — |
| EA-4 | Growth of paid members by tier and billing period on Earn › Overview, where it is missing. | UI test. | S | Extend EarnOverview |
| EA-5 | Money paths tested against the in-process Stripe fake: offer limits and end dates, gift link transfer and its 12-month expiry, "end access at once", legacy prices. | Table tests. | M | — |
| CH-1 | SMS, push and WhatsApp campaigns get CA-1 and CA-2 (delete, archive, bulk, tags). | Tests. | M | Reuse CA-1/CA-2's boards |
| CH-2 | Behaviour tests for SMS auto-pause, push and WhatsApp retries, expired push subscriptions and the retention sweeps. | Tests. | M | — |

What M49 built (1 November 2026):

- **EA-1** the tier, offer and upgrade page editors ask before leaving with unsaved changes (the shared
  `UnsavedChangesGuard`); a save marks the form saved before it navigates, so saving never asks.
- **EA-2** audited CSV exports, behind subscribers:export: the gifts a filter shows (`gift.exported`),
  the conversions log (`conversion.exported`) and the payment event log a filter shows
  (`payment_events.exported`), with Export CSV on each through one download helper (`TestEarnExports`).
  Sponsor bookings and reports are EA-2b.
- **EA-3** the MRR chart has the chart/table toggle the movement chart already had, and the paid
  members list a rows-per-page picker (8, 25, 50, 100) kept in the address.
- **CH-1** text, push and WhatsApp campaigns are deleted, restored, archived and tagged like email ones:
  the unified Campaigns list gives every channel the row menu and bulk bar since M45; a test pins the
  server side (`TestChannelCampaignsDeleteArchiveAndTag`).

Found while building M49 (added to the backlog):

| ID | What | Status |
|---|---|---|
| EA-2b | CSV exports of sponsor bookings and sponsor reports, like the other Earn lists. | ✅ M50 |
| EA-4 | Growth of paid members by tier and billing period on Earn › Overview. | ✅ M51 |
| EA-5 | Money paths against the in-process Stripe fake: offer limits and end dates, gift link transfer and expiry, "end access at once", legacy prices. | ✅ M51 |
| CH-2 | Behaviour tests for SMS auto-pause, push and WhatsApp retries, expired push subscriptions and the channel retention sweeps. | ✅ M51 |
| EA-6 | UI tests for the Earn editors' leave guards and the exports' buttons (Earn has no web tests yet). | ✅ M50 |

## Integration and audit (M50) ✅

| ID | What | Done when |
|---|---|---|
| X-1 | No permission keys in copy; refusals say who can help (carried from October). | Visual QA. |
| X-2 | Every new or changed string in English and Vietnamese; `pnpm i18n:check` passes. | CI. |
| X-3 | Roadmap, `docs/DESIGN_INVENTORY.md` (new boards, deviations such as D5) and README claims updated per milestone. | Per milestone. |
| X-4 | Every state change writes an audit entry; every row names the test that pins it. | Review of each PR. |
| X-5 | A traceability page like [`reputecast-review-2026-10-traceability.md`](reputecast-review-2026-10-traceability.md): each row read against the code, with its tests, at the end. | Page written; anything done in part finished. |

What M50 built (1 November 2026):

- **X-1** the Earn exports' locked buttons named `subscribers:export`; they say who can export, like
  every other refusal (`earnUi.test.tsx` checks the copy).
- **X-2** every new string is in English and Vietnamese; `pnpm i18n:check` reports 100% at each commit.
- **X-3, X-4** the roadmap, `docs/DESIGN_INVENTORY.md` and the boards follow each change below; the
  new state changes are audited (`sponsor.bookings_exported`, `sponsor.report_exported`), and the
  traceability page marks the rows whose tests check the audit entry.
- **X-5** [`reputecast-review-2026-11-traceability.md`](reputecast-review-2026-11-traceability.md)
  reads every row against the code, with its tests.

From the backlog the milestones found, M50 built:

- **DP-8** a done import drops its uploaded file; finished imports keep their reports 30 days (the
  report step says until when, downloads answer 410 after, maintenance clears them); an erase strips
  the erased addresses from the reports still kept and removes an uploaded file that names one, with
  a line in the import's log (`TestEraseStripsImportFiles`). Board: MigrationDone.
- **DP-9** Earn › Paid members shows an erased member's kept subscription as "Erased contact", with no
  address and no contact link (`TestErasureKeepsRevenueNumbers`).
- **EN-10** the product's publish checks list the payment-failed notice as a recommendation that
  doesn't block publishing, with a link to publish it (`TestDelayedPaymentThatFails`,
  `products/checks.test.ts`). Boards: ProductEditor, ProductEditorStates.
- **AU-18** what-uses-it reads transactional templates, texts' bodies, push messages and WhatsApp
  campaigns' variables; a form's confirmation email names no custom field
  (`TestFieldsAndSegmentsShowWhatUsesThem`).
- **AU-19** Contacts takes `?view=`: a saved view opens with its filters and columns, or says it was
  deleted; what-uses-it and List health link there (`ContactsPage.test.tsx`, `UsesList.test.tsx`).
- **AT-14** the welcome prompt, score ingest, the integrations hub, a flow's push audience and a held
  campaign's "add to automation" read `automations.triggers`; scoring, products, store and date-trigger
  queries read the array too (`TestWelcomePromptReadsEveryTrigger`,
  `TestAutomationEmailListsReadEveryTrigger`).
- **CA-10** calendar items say when an approval covers the time; moving one offers "Unschedule and
  ask again" (`TestReschedule`, `manage.test.tsx`). Board: CampaignsCalendar.
- **CA-11** `api.Register` refuses an input whose parameter Huma can't set (an unexported field, or
  one reached through an unexported or aliased embedded type) at start-up and in tests
  (`TestParametersReachTheHandler`).
- **CA-12** the campaigns tabs count within the filters, in one query (`TestListCampaigns`).
- **OP-x** the Northwind sample's newest client stays in the current month, so `TestUsageAndRebill`
  holds in a month's first hours (it failed on 1 November before).
- **EA-2b** sponsor bookings and a booking's report export as CSV, audited
  (`TestSponsorReportCountsOnlyBlockRecipients`). Boards: EarnSponsors, EarnBooking.
- **EA-6** Earn's first UI tests: the offer editor's leave guard and the gifts export
  (`earnUi.test.tsx`).

Still open after M50 went to M51 below.

## Review backlog (M51)

What M51 built (1 November 2026):

- **AP-3** the approvals inbox timed at 1,000 open requests: 118 ms when each was checked recently, as it
  re-checks only stale requests, and 460 ms after ten campaigns changed (2.3 s before): stale requests are
  checked four at a time and share audience counts (`TestInboxAtAThousandRequests`,
  `docs/deploy/performance.md`).
- **FE-10** single settings in the command palette, found by the words their page shows.
- **FE-7b** switches and the order of your recommendations change at once and go back when the server
  refuses, through one hook (`useOptimistic`, `optimistic.test.tsx`). Other reorderable lists edit a
  draft that is saved as a whole.
- **TX-7** a batch of scheduled messages moves or is cancelled together, by its Idempotency-Key
  (`TestBatchMovedAndCancelledTogether`). Board: TransactionalComplete.
- **EN-9** a gift's failed bank payment tells the buyer: a ninth member email, Gift payment failed
  (`TestGiftRemindersExpiryAndRefunds`). Boards: EarnSettingsTabs, EarnGifts.
- **AT-15** a slow watched segment is checked every twenty times its evaluation time (10 minutes to
  2 hours), after the quick ones (`TestSlowWatchedSegmentsBackOff`). Board: AutomationsComplete.
- **CH-2** behaviour tests: SMS auto-pause, push and WhatsApp retries, the channels' retention sweeps.
- **EA-5** money paths against the Stripe fake: cancel now or at period end, a legacy price, a forwarded
  gift link.
- **EA-4** growth of paid members by tier and period on Earn › Overview ("+38 · −9"). Board: EarnOverview.
- **EN-8** an endpoint can take its `email.*` events in batches of up to 100 (`email.batch`)
  (`TestEmailEventsInBatches`). Board: TransactionalComplete.
- **CA-9** the calendar moves texts, push and WhatsApp campaigns too: texts are re-planned per recipient
  zone inside quiet hours; push takes one time for everyone (`TestCalendarMovesTexts`,
  `TestCalendarMovesPush`, `TestCalendarMovesWhatsApp`). Board: CampaignsCalendar.
- **OP-12** the slow counts that read every contact (a quarter of a second or more) are kept for a
  minute and dropped by this instance's audited changes (`TestCountsAreKeptUntilTheAudienceChanges`,
  `internal/countcache`).
- **FE-9** the first load is 390 KB gzipped (439 KB before): routes' checks import only small modules,
  and an agency's menu loads for its staff only. The budget is 400 KB.
- **FE-3b** `ui/Stepper` for the six setup wizards with the numbered card (the import wizard too), and
  its one-line form `StepTrail` for the four wizards inside dialogs (integrations, store connect, Earn
  import, transactional import); `ui/Pager` for the eight lists read page by page; one bulk-selection
  pattern (`useSelection` with `ui/SelectAll`: Campaigns, Grow › Review, Team; Contacts keeps its
  select-all-matching table); `NoResults` on every page list that a search or filter can empty (fifteen
  pages), `EmptyRow` inside tables. The patterns check refuses new hand-made steppers, pagers and
  select-alls; only first sign-in's vertical list and Earn's setup checklist, which aren't step bars,
  mark a step themselves.
- **FE-5b** every app colour that has a token uses it (1,166 → 323 hex strings in TSX): classes and
  styles, SVG and chart attributes (browsers resolve `var()` there), status palettes, and ten stronger
  and darker steps of the status families the boards use for alerts and tinted panels, now named in
  `app.css` (`warn-line-strong`, `bad-wash`, `warn-ink`…). What keeps a literal, and why:
  - 248 by design: the brand kit, email and page content (composers, MJML defaults, previews).
  - 45 that must stay text: a canvas (the score spread), colours parsed or given an alpha, HTML built
    as a string, and the push prompt's default accent, which publishers' own sites show.
  - 39 with no token, 32 colours used once or twice: third-party logos (Google, Microsoft), the avatar
    and workspace identity palettes, and single shades. A token each would only grow the palette.
  - 18 that aren't colours ("Weekly Digest #142" in sample names).
  `hex-baseline.json` holds each file at its count; the patterns check refuses any increase.

Checked at the end of M51:
- **Accessibility.** An axe sweep (WCAG 2.0, 2.1 and 2.2 A and AA) of every screen at 1440 and 390 px
  (`web/scripts/a11y.ts`). Its one violation, the saved views' 22px menu buttons, is fixed. The design
  system's stricter 44px rule on phones went from 1,451 controls under it to 730, as buttons and fields
  became 44px tall there. The rest are switches, filter chips and links inside rows.
- **Security.** `pnpm audit` found two advisories in seroval, which TanStack Router pulls in; it is
  pinned to the patched 1.6.11. A CI job now runs govulncheck and `pnpm audit` (it couldn't run here).

- **OP-13** the Load test workflow ran on a pull request labelled `load-test` (GitHub runs a hand-started
  workflow only from the default branch). Every call met its target one at a time; pages (p95 50 ms) and
  segment counts (p95 10.6 ms) held under load. Search under ten people missed (p95 513 ms on four shared
  cores): the first page now counts its own rows when they are all the matches, which halves a narrow
  search's reads. The send test had gone through a person's session (50 a second, so 52% were 429s by
  design) and now sends with a key; the workflow fails when a threshold is crossed. Numbers in
  `docs/deploy/performance.md`.

Still open: OP-11 (a later release, by design).

## Competitor benchmark

What competitors shipped in 2026 that improves a feature ReputeCast already has, and where it lands.
Most facts come from third-party coverage of the releases; plan details may differ by tier.

| Competitor | Behaviour | Here |
|---|---|---|
| beehiiv Automations v4 | Trigger on entering or leaving a segment; up to 3 triggers as OR; early exit on a branch; multi-split 2–5 paths; A/B on an email step | AT-10, AT-11 |
| beehiiv (Aug 2026) | Click triggers for automations; a send API | AT-10 |
| beehiiv segments | Static, dynamic and manual segments; recalculate now; dynamic segments go inactive after 30 days without change | AU-16 |
| Kit (Jun 2026) | Engagement analytics (cohorts, source attribution); subscriber profiles with engagement and a suggested next action | AT-2, AU-9 |
| Ghost (Jan–Aug 2026) | Growth by tier and billing period; traffic filters and UTM; welcome emails and their sequence analytics | EA-4, AT-13 |
| Mailchimp (2026) | Segment performance side by side; resend to non-openers; send in the recipient's time zone; a unified automation view | CA-4, CA-5 |
| MailerLite (2026) | A preference center meant to cut unsubscribes; an MCP server | CA-6, IN-3 |
| Customer.io (2026) | Change a campaign's trigger without rebuilding it; archive unused segments | AT-12, AU-11 |
| Resend | Cancel or reschedule scheduled emails; only drafts can be deleted | TX-1, CA-1 |
| Postmark | Send test from a template; suppression actions from a message | TX-5, TX-2 |
| listmonk v5–v6 | Unschedule; edit a paused campaign; Ctrl+S; global tracking privacy | CA-2 |
| Kit, beehiiv, MailerLite | MCP servers with read and write tools | IN-3 |

Sources:

- [beehiiv webinars and paywalls (TechCrunch, April 2026)](https://techcrunch.com/2026/04/23/beehiiv-rolls-out-new-creator-tools-including-webinars-and-customizable-paywalls/)
- [beehiiv summer release (TechCrunch, July 2026)](https://techcrunch.com/2026/07/16/newsletter-platform-beehiiv-now-lets-subscribers-chat-with-each-other-adds-ai/)
- [beehiiv Automations v4](https://product.beehiiv.com/p/automations-v4) and [triggers and actions](https://www.beehiiv.com/support/article/13080928484887)
- [beehiiv segments](https://www.beehiiv.com/support/article/9366426644503-How-to-create-a-segment)
- [Kit release notes](https://releasebot.io/updates/convertkit) and [subscriber analytics](https://help.kit.com/en/articles/14085468-understanding-the-subscriber-analytics-dashboard-in-kit)
- [Ghost 6.0](https://ghost.org/changelog/6/), [traffic filters](https://ghost.org/changelog/filters-for-traffic-analytics/), [growth tab](https://alternativeto.net/news/2026/1/ghost-improves-growth-tab-with-detailed-subscription-insights-and-new-graph-views/), [sequence analytics](https://alternativeto.net/news/2026/8/ghost-adds-email-sequence-analytics-for-automated-welcome-flows/)
- [Mailchimp product updates](https://mailchimp.com/product-updates/)
- [MailerLite 2026 updates](https://www.stork.ai/en/mailerlite)
- [Customer.io LLM actions](https://docs.customer.io/journeys/llm-actions)
- [Resend broadcasts](https://www.resend.com/docs/dashboard/broadcasts/introduction)
- [listmonk v5.0.1](https://nadh.in/releases/listmonk-v5.0.1)

## Out of scope

New product areas competitors added in 2026 (podcasts, subscriber chat, an ad network, paid trials,
webinars beyond M29's pages) and dark mode, which the design system leaves out of this version
(`design/design-system/README.md`). They need their own brief and boards first.
