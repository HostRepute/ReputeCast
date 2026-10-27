# Review 2026-10: fixes and additions for the five core features

Source: the October 2026 product review of ReputeCast's five core features (Send Readiness,
guarded sending, bring-your-own providers, the composer, honest analytics), its code audits and
competitor research. Every recommendation in that review has an ID below, an owner milestone and
what "done" means. Milestones M33–M39 in `docs/ROADMAP.md` implement them; the design for every
screen is on the canvas (`design/boards`, updated for this review).

Status: ⏳ planned · 🚧 in progress · ✅ done

## Shared foundations (M33) ✅

| ID | What | Done when |
|---|---|---|
| F-1 | Mailbox provider from MX records: Google Workspace counts as Gmail, Microsoft 365 as Outlook, Yahoo-hosted domains as Yahoo, iCloud custom domains as iCloud. One classifier with a domain cache and a background job, used by every contact insert path, routing, the sending engine, Send Readiness, transactional sends, reports and warm-up. Existing contacts are backfilled. | A Workspace-hosted address is routed, throttled, guarded and reported as Gmail; the detail ("Google Workspace") is kept; tests cover the classifier and one end-to-end route. |
| F-2 | Complaint reporting semantics: Gmail and iCloud don't report complaints per message. Complaint rates use only delivered mail at mailbox providers that report complaints; per-mailbox complaint values for the others are "not reported". | Guard, live monitor, report and provider cards agree; APIs return `null` + a reason, never 0, for unreported values. |
| F-3 | Unknown is not zero: rates without data (no delivery reports, SMTP without DSN, silent webhooks) are `null` with a reason in the API and "n/a" with the reason in the UI. | Covered by the API types and UI tests of M34–M37. |
| F-4 | Every table production code reads has a production writer (demo seeds don't count). | A test fails when a queried table is only written by `internal/demo` or tests; the known violations are fixed. |
| F-5 | Production-shape end-to-end tests: a campaign goes through the real API, audience build, sending engine, the simulator transport (delivery events through webhook normalization, opens and clicks through the tracking recorder) on a virtual clock, so features are tested on the data production writes. | A reusable harness, `internal/sending/sendtest`, plus tests in M34–M37 that use it. |
| F-6 | Every claim the UI makes is backed by behaviour (idempotency, events verified, setup time, previews, A/B significance, blocklist checks). | Copy corrected; a test pins each claim's behaviour. |

What M33 built for the others to use:

- **F-1** `internal/mailbox`: `Classifier` (domain cache, `mailbox.sweep` every 2 minutes, `mailbox.lookup` for one domain, a bounded `ClassifyPending` before a send's audience and a Readiness run), `FromHosting`, `MXChecker` (moved from imports). Contacts get `mailbox_checked_at`; the class's detail ("Google Workspace") stays in `mailbox_detail`. Routing, the engine, transactional sends, Readiness and imports use it; reports, warm-up and the guard read the classified contacts and messages. Sample contacts are never looked up. Test: `TestSending/MailboxFromMX`.
- **F-2** `mailbox.ReportsComplaints(provider)` and `mailbox.NotReported`, for the guard (M34), live monitor (M34), report (M37) and provider cards (M35).
- **F-4** `internal/store/writers_test.go`; the known gaps it lists are SR-1 and EXT-1.
- **F-5** `internal/sending/sendtest`: `Suite`, `Fixture`, `ContactSpec` (with `Unclassified` for MX classification), `StartSend`, `Drive`, `Dump`.
- **GS-5 / PR-7 groundwork** `state.Vet`: one target vetting (available, bulk terms, sandbox, sender domain, quota, warm-up) used by a person's move, the guard's automatic move and the failover while an audience is built, and returned with its sentences (EN, VI) on the provider outage card's alternatives (`checks`, `allowed`). M34 and M35 build their screens and the health job's checks on it.

## Send Readiness (M36) ✅

| ID | What | Done when |
|---|---|---|
| SR-1 | Link domains are looked up live on Spamhaus DBL and SURBL (DNS), cached in `link_domain_listings`; a failed lookup is Unknown. Shared with the composer's inline checks. | A listed link domain blocks in Review and warns in the composer on a real install. |
| SR-2 | Sender reputation includes the parent (organizational) domain, return-path domains and the tracking domain. | `acme.io` listed → `news.acme.io` readiness shows it. |
| SR-3 | Send-time recheck holds the send on new warnings or unknowns nobody acknowledged, on audience growth over 10%, and on content, sender or route edits after scheduling; the typed confirmation uses the fresh count; a held send expires (default 3 h after its time). | Tests for each hold reason and the expiry. |
| SR-4 | One-click fixes keep HTML-only campaigns intact (they edit the HTML or are unavailable); saving an empty body is refused. | Test: Minify / Use suggestion on an HTML-only campaign keeps its body. |
| SR-5 | Estimates match the engine: warm-up caps over days, default rates when a provider publishes none, quotas, the A/B wait, best-time spreading, per-mailbox throttles, parallel routes. A provider's bounce rate blocks only after a minimum volume. | Estimate tests against the engine's own planner. |
| SR-6 | A DNS lookup error is Unknown (looked up again during the run), never a Block that says the record is invalid. | Test with a failing resolver. |
| SR-7 | Missing checks: A/B variants B/C, Liquid errors (Block), spam heuristics on subject, preheader and body without the known false positives, one image-ratio threshold, size measured with tracking, honest link counts ("40 of 60 checked"), bot-blocking responses (999) as Unknown, links of every language version, Gmail Postmaster spam rate and compliance, DMARC alignment per provider, one-click unsubscribe per provider capability, PTR for dedicated IPs, tracking domain, invalid BIMI as Warn. | One test per check. |
| SR-8 | Runs are background jobs with progress; polling is cheap; one run at a time per campaign. | No readiness work inside a request beyond reading the stored run. |
| SR-9 | Placement test from Review (seed mailboxes) with its result card. | Starts a placement test for the campaign and shows Inbox / Promotions / Spam per mailbox provider. |
| SR-10 | Workspace sending rules (link query parameters, link domains, subject length, required text) with Warn or Block, in Settings › Sending rules. | Rules appear as a Send Readiness group. |
| SR-11 | Unengaged contacts (sunset policy) are excluded by default, shown as a line of the audience waterfall with "Include them". | Audience math and estimates use it. |
| SR-12 | Gmail Postmaster compliance status is a compliance check when Postmaster data is connected. | Check shows Compliant / Needs work / not connected. |
| SR-13 | Review & send matches the new boards: issues first, passes collapsed by group, technical details, fix review drawer with consequences and undo through versions, sending locked while checks re-run with real progress, the schedule in the main action, sticky action bar on phones, typed confirmation only for unusual sends, auto-pause summary, "blocklist checks skipped" wording, test sends while blocked, emergency send in a menu. | Visual QA at 1440/834/390 against SendReadiness*, SendReadinessFixes, SettingsSendingRules. |

## Guarded sending and the live monitor (M34) ✅

| ID | What | Done when |
|---|---|---|
| GS-1 | No double sends: results are recorded even when the job is cancelled; River stops gracefully; a stale claim becomes "in doubt" and is not resent without an idempotency key; stuck jobs are rescued within minutes; the copy says what really happens. | Tests for SIGTERM, DB error and timeout paths. |
| GS-2 | Resuming a campaign- or route-scope pause re-arms from the current counts; "resume with changes" works for those scopes; thresholds can be changed while paused (audited). | Resume doesn't re-trip on old events (test). |
| GS-3 | Outages found by the provider health job go through the engine's hold with its options; Retry asks again after 15 min even while the provider is down; an automatic probe (1 message every 60 s, resume after 3 successes). | Outage incident always has options (test). |
| GS-4 | The provider-errors threshold is enforced; throttling and an exhausted quota are outages for that route. | Tests. |
| GS-5 | Moves check the target (domain verified, quota, warm-up, sandbox, bulk terms); A/B holdouts stay held; decisions on a share don't undo a person's pause. | Tests. |
| GS-6 | Complaint guard per F-2, a 72-hour complaint watch after the send with alerts, evidence (counts, sample needed) in the API. | Tests + live UI. |
| GS-7 | SMTP: delivery is inferred from acceptance (DSN success requested when the server supports it) so delivered-based rates work. | SMTP send shows delivered (inferred). |
| GS-8 | ETA per route and mailbox provider; throttle buckets shared across a workspace's concurrent sends. | Tests. |
| GS-9 | Deferrals count once per message; a warm-up cap reached mid-send raises a hold with choices; alert preference labels name the right permission. | Tests. |
| GS-10 | Canary phase: automatic (new sending domain or IP, contacts never mailed from this domain, audience over 2× usual) or chosen in Review; 5% most engaged plus seed mailboxes; watch window; gates; continue, extend, stop. | API + LiveMonitorCanary board. |
| GS-11 | Thresholds by bounce type (block/policy bounces per mailbox provider) and deferral thresholds (slow down, then pause). | Engine + Traffic rules. |
| GS-12 | Mailbox provider response rules (reply text → pause or slow a mailbox provider for a time) configurable in Traffic rules. | Engine applies them; tests. |
| GS-13 | Rates per mailbox provider adjustable during a send. | API + live UI. |
| GS-14 | Live monitor matches the new boards: collecting-evidence states, n/a for unreported complaints, people instead of permission keys, stale-data detection with a polling fallback, live regions only where they belong, Pause primary on phones. | Visual QA against LiveMonitor*, ProvidersTrafficRules. |

## Bring-your-own providers, routing and warm-up (M35) ✅

| ID | What | Done when |
|---|---|---|
| PR-1 | Suppression push: only hard bounces and complaints that apply to all mail; SES gets them on the workspace's tenant or configuration-set list, not the account; removals sync; pushes track an insert sequence; the pending count is real. | Unsubscribed contact still gets a transactional message through SES (test). |
| PR-2 | Splits are order-independent and sticky (rendezvous hashing); routing saves only changed streams with a version check; the editor keeps unsaved edits. | Reordering a split moves nobody (test). |
| PR-3 | SES onboarding finishes only after the SNS subscription is confirmed and a real event from the SES mailbox simulator arrives; "mail sent, no event after 30 min" is flagged for every provider. | Wizard tests. |
| PR-4 | SES events through an SQS queue that ReputeCast polls (works behind NAT). | Adapter + wizard option. |
| PR-5 | IAM keys are the default when ReputeCast doesn't run on AWS; the CloudFormation path explains that it needs base credentials off AWS. | Wizard tests. |
| PR-6 | "Webhooks silent" needs mail sent more than 30 min ago, a minimum count and no event for 6 h; polling and bounce mailboxes update freshness. | No false alarm after a quiet period (test). |
| PR-7 | Failover targets are vetted (domain verified, quota, sandbox, warm, bulk terms) in the health job and the engine; identity errors hold instead of dropping mail; rejections are watched. | Tests. |
| PR-8 | Warm-up is enforced in the send loop for every stream (cap, engaged-first order, pause), in one time zone; pausing freezes the plan; a slowdown holds until 2 healthy days; a day is judged after 500 messages; shortening a plan never jumps the cap; days advance by volume (80% of the cap); caps per mailbox provider; overflow to a warm provider. | Tests. |
| PR-9 | Webhook reachability is checked from outside (provider test events, SNS confirmation), not by the server calling itself; one env var name everywhere. | Wizard tests. |
| PR-10 | Unavailable providers raise an issue; Send Readiness checks fallbacks; bounce-mailbox poll errors reach provider health. | Tests. |
| PR-11 | SES tenants per workspace (own suppression list and reputation policy). | Adapter + settings. |
| PR-12 | Conditions can split by weight; "spread across mailbox providers" is applied by warm-up. | Tests. |
| PR-13 | Help me choose in the provider catalog (cost at your volume); campaigns can be written before a provider is connected; a demo workspace can be loaded from the first-run screen. | UI + API. |
| PR-14 | Routing UI per the new boards: priority list with keyboard moves, impact preview before saving, unsaved-changes bar, campaign simulation in the route tester, failover triggers and target checks. | Visual QA against ProvidersRouting. |
| PR-15 | Provider screens per the new boards: channel menu, Traffic rules tab, honest setup copy (time, events verified), suppression scope, webhook grace, SES tenant. | Visual QA against Providers*, Setup*, OverviewEmpty. |

## Composer (M38) ✅

| ID | What | Done when |
|---|---|---|
| CO-1 | Autosave while text is being typed; a local journal restored on reopen (Restore / Discard). | Tests: a crash mid-typing loses at most a few seconds. |
| CO-2 | Conflicts and versions: "Load theirs" saves yours as a version first; the language switch respects the save result and the leave guard; no save race; restore with unsaved edits works; history pages with a named-versions filter. | Tests. |
| CO-3 | The server compiles MJML itself (or validates the client's HTML) and checks the HTML it stores; conditional markers are enforced; gomjml and mjml-browser parity tests. | API test: mismatched HTML is refused or recompiled. |
| CO-4 | Liquid: syntax errors block in Readiness; unknown fields are flagged even with a fallback; a per-recipient render error raises an incident. | Tests. |
| CO-5 | Preview as a contact and test sends use the contact's own first and last name fields; the missing-fields count is real. | Tests. |
| CO-6 | A/B in the composer: each variant checked and previewed; "+ variant" starts from A's subject; test length and winner metric controls. | Tests. |
| CO-7 | Editing an approved campaign asks first; Send test waits for a successful save; failed checks are shown as stale. | Tests. |
| CO-8 | Undo covers subject, preheader and A/B settings; Enter starts text editing, Esc selects the parent, Layers has arrow keys; the text editor has headings and lists. | Tests. |
| CO-9 | Write mode: a document editor over the same email (slash menu, markdown shortcuts, headings and lists, paste cleanup) that maps to MJML blocks. | ComposerWrite board. |
| CO-10 | Collaboration: comments with @mentions on blocks, presence, reviewer links without sign-in (expiring, revocable), section-level merge on conflicts. | Tests + UI. |
| CO-11 | Brand kit (logo, colours with contrast, fonts, buttons, import from a website) and synced blocks. | BrandKit board. |
| CO-12 | Uploaded images are compressed and resized on the server, with an alt-text prompt. | Tests. |
| CO-13 | Optional real rendering service (Litmus or Mailgun Inspect, the workspace's own key); built-in previews say they are approximations. | Integration + copy. |
| CO-14 | Composer matches the new boards: compact chrome, Inbox strip, Sending-to drawer, Liquid chips with a fallback editor, writer-friendly style panel, template thumbnails, brand kit banner, dismissing warnings with a reason, quick-review mode on phones. | Visual QA against Composer*, ComposerWrite, BrandKit. |

## Honest analytics (M37) ✅

| ID | What | Done when |
|---|---|---|
| AN-1 | A/B end to end: decision snapshot, rollout kept apart from the test sample, minimum sample and significance (95%, at least 500 per variant), no clear winner → the default variant, a chosen metric, test length, send-time tests decided after the last variant's time in recipients' local time; the report reads production data. | Production-shape test from send to report. |
| AN-2 | Bot clicks: honeypot link, bursts reclassified afterwards (human open, unique click and automation trigger undone or delayed until settled), windows from delivery, updatable network lists, filtered clicks shown. | Tests. |
| AN-3 | Counters change only on status transitions; "undelivered" is right; tracking writes retry; one lock order; a rebuild job from stored events. | Tests. |
| AN-4 | Provider tracking mode counts provider opens and clicks; SMTP delivery inferred (GS-7). | Tests. |
| AN-5 | Click map per variant; one rule for which links are tracked, shared by engine and UI. | Tests. |
| AN-6 | AI summary: each number checked against its metric, locale number formats, A/B significance included, edits re-checked, removal audited. | Tests. |
| AN-7 | Bounces blocked for sender reputation are never suppressed from the report; "Investigate" instead. | Test + UI. |
| AN-8 | Requests through an untrusted proxy are detected and the admin is told to list it in `http.trusted_proxies` (`REPUTECAST_HTTP__TRUSTED_PROXIES`). | Test + admin notice. |
| AN-9 | Classifier fixes: no "bot" substring false positives; Apple privacy prefetches counted apart from scanners. | Tests. |
| AN-10 | Opens in three buckets (human, privacy-protected, machine) and the measurable open rate. | API + UI. |
| AN-11 | Comparison with the last issues on every tile. | API + UI. |
| AN-12 | Growth and revenue per issue (subscribers from forwards, referrals and the web post; upgrades; attributed revenue; web views). | API + UI. |
| AN-13 | First-party, cookie-free page views for web posts. | Tests. |
| AN-14 | Ask about this report (the workspace's own model, numbers checked) and MCP tools for reports. | Tests. |
| AN-15 | Report matches the new boards: tile comparisons, y-axis, human/bot click map, insights with actions, honest A/B card, share options. | Visual QA against Report*, ReportStates. |
| AN-16 | Share links: what is shared is chosen, links expire (30 days by default), several links can be listed and revoked one by one. | Tests. |

## Cross-cutting (every milestone) ✅

| ID | What | Done when |
|---|---|---|
| X-1 | No permission keys in UI copy on these screens; people or roles instead. | Visual QA. |
| X-2 | Every new or changed string in English and Vietnamese; `pnpm i18n:check` passes. | CI. |
| X-3 | Docs: roadmap, design inventory, README claims. | Updated per milestone. |

M39 checked them across the review's screens: refused actions name who can help (`authz` and each screen's own copy), `pnpm i18n:check` passes with every new string in English and Vietnamese, and the roadmap, `docs/DESIGN_INVENTORY.md` (#517–#554, the deliberate differences from the boards) and the README describe what was built. Screens outside the review (team, verification, Earn, SMS and a few others) still name permission keys in places; an API key's scopes stay named, since that is what the key holds.

## Found while building the review (outside the five features)

| ID | What | Status |
|---|---|---|
| EXT-1 | Referral coupon rewards are never issued: `referralCoupons` in `internal/app/wire_referrals.go` is still the stub written before Earn's offers ("offers come with Earn's offers"), and nothing replaced it, so `earn_offer_codes` is only filled by the demo. Found by F-4's writer check. | ✅ Earn's "Referral reward" offers now issue single-use promotion codes (`offers.IssueReferralCode`), the upgrade link applies them once until they expire, and a subscription that uses one marks it redeemed (`TestReferralRewardCodes`). |
