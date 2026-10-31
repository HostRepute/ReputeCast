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
| DP-8 | An import keeps its uploaded file (`imports.file_data`) and its report and skipped-row CSVs, which hold every address it read, until the import is deleted; erasing a contact doesn't reach them. Strip erased addresses from kept files and reports, or expire them after the import finishes. | ⏳ |
| DP-9 | Earn › Paid members lists an erased member's ended subscription under the placeholder address; show "Erased contact" instead. | ⏳ |

## Engines and events, correct (M42) ⏳

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| EN-1 | Each automation event and run is processed in its own savepoint. A failing one gets an attempt count and a backoff, is set aside after 5 attempts with a `step_failing` alert, and never blocks the rest; the tick runs even when dispatch fails. | Test: one poisoned run in a batch of 100, the other 99 advance and the alert opens. | M | — |
| EN-2 | Date triggers dedupe on the workspace's local date (or the contact's, AT-7). A yearly trigger defaults to "once per year" and the editor warns when "once" is chosen with "every year". | Tests in Auckland and Sydney: one event per contact per day; a yearly flow fires in its second year. | S | AutomationFlow trigger panel shows the warning (existing problem list) |
| EN-3 | "Opened/clicked within X" starts when the email is sent; while it is queued or held the branch keeps waiting. | Test: a message held for 2 days by a warm-up cap still gets its full window. | S | — |
| EN-4 | "Skip if they got an email in the last N hours" counts only broadcast and automation messages that were actually sent. | Test: a password reset doesn't skip a welcome email. | S | — |
| EN-5 | Automation step statistics from real providers: the ingest normalizer calls `RecordMessageEvent` for automation messages (delivered, bounced, complained) and the unsubscribe path records unsubscribes. Steps report Queued, Sent (on outbox success), Failed and Cancelled separately. Emails that fail to compose (no sender, deleted version, Liquid error) open a `step_failing` alert, and a new notification kind, "An automation needs attention", reaches the bell and email. | A `sendtest` run through a webhook-based provider shows per-step Delivered and Bounced; a compose failure notifies. | M | AutomationsList and AutomationFlow already draw the step numbers; Notifications preferences gain one row |
| EN-6 | Every catalog event is emitted: `email.*` from webhook normalization and the tracking recorder (only human, settled opens and clicks; batched per endpoint so a 1M send doesn't make 1M HTTP calls), `provider.down` and `provider.recovered` from provider health, `automation.published` from publish, `contact.erased` from erase, `campaign.scheduled` when an email campaign is scheduled without approval. | A test fails when a catalog event has no emitter (like `internal/store/writers_test.go`); a send through `sendtest` delivers each `email.*` event to an endpoint. | M | — |
| EN-7 | The small bugs of Fix first #9: Duplicate on sent rows links to `/campaigns/new?from=` like the report; the Source filter reads `/growth/sources`; tag search and filters lower-case their input; referrer search escapes LIKE wildcards (`internal/referrals/referrers.go:91`); starting an import twice is refused with 409 (`internal/store/queries/imports.sql:53-55`); the transactional log matches idempotency keys case-insensitively and finds a batch by its key; the approvals inbox filters status and counts in SQL with keyset paging; Stripe's failed async payment and dispute update are subscribed, recorded, and the buyer is emailed. | One test per bug. | S each | — |

## Audience at 1M contacts (M43) ⏳

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

## Automations, complete (M44) ⏳

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

## Campaigns and everyday sending (M45) ⏳

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

## Transactional, approvals, polls and integrations (M46) ⏳

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

## Operations (M47) ⏳

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

## Frontend foundation (M48) ⏳

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

## Earn and other channels (M49) ⏳

| ID | What | Done when | Size | Board |
|---|---|---|---|---|
| EA-1 | Unsaved-changes guards on the tier, offer and upgrade page editors (`UnsavedChangesGuard`, used by about 25 other editors). | UI tests. | S | — |
| EA-2 | Audited CSV exports for gifts, sponsor bookings and reports, conversions and the payment event log, like paid members and orders. | Tests. | M | Extend each list's header with Export |
| EA-3 | Every Earn chart has a table view (design system rule; Growth's toggle is the pattern); the paid members list has the page-size picker. | UI tests. | S | — |
| EA-4 | Growth of paid members by tier and billing period on Earn › Overview, where it is missing. | UI test. | S | Extend EarnOverview |
| EA-5 | Money paths tested against the in-process Stripe fake: offer limits and end dates, gift link transfer and its 12-month expiry, "end access at once", legacy prices. | Table tests. | M | — |
| CH-1 | SMS, push and WhatsApp campaigns get CA-1 and CA-2 (delete, archive, bulk, tags). | Tests. | M | Reuse CA-1/CA-2's boards |
| CH-2 | Behaviour tests for SMS auto-pause, push and WhatsApp retries, expired push subscriptions and the retention sweeps. | Tests. | M | — |

## Integration and audit (M50) ⏳

| ID | What | Done when |
|---|---|---|
| X-1 | No permission keys in copy; refusals say who can help (carried from October). | Visual QA. |
| X-2 | Every new or changed string in English and Vietnamese; `pnpm i18n:check` passes. | CI. |
| X-3 | Roadmap, `docs/DESIGN_INVENTORY.md` (new boards, deviations such as D5) and README claims updated per milestone. | Per milestone. |
| X-4 | Every state change writes an audit entry; every row names the test that pins it. | Review of each PR. |
| X-5 | A traceability page like [`reputecast-review-2026-10-traceability.md`](reputecast-review-2026-10-traceability.md): each row read against the code, with its tests, at the end. | Page written; anything done in part finished. |

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
