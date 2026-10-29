# Review 2026-10: every report item, traced to code and tests

The October 2026 review of ReputeCast's five core features came as a report (Vietnamese, kept
private for the owner; "ReputeCast — Phân tích 5 tính năng lõi & đối thủ", 25 October 2026).
[`reputecast-review-2026-10.md`](reputecast-review-2026-10.md) filed its recommendations as spec
rows (F-, SR-, GS-, PR-, CO-, AN-, X-, EXT-) and milestones M33–M39 built them.

This page goes the other way, from the report to the code. It numbers every item the way the
report lists it: section, then L for a logic fix ("Sửa logic"), A for an addition ("Bổ sung tính
năng") or U for UI/UX, then its position. R1.L3 is the third logic fix of section 1 (Send
Readiness); X.1–X.5 are the cross-cutting problems. Each item gives the spec IDs it was filed
under, its status, the code and the tests that pin it.

On 28 October 2026 the code was audited against the report line by line, reading the code rather
than the spec rows. The audit found 47 places where an item was done only in part or differently
from what the report asked, besides a few wording fixes made straight away. M40 finished all of
them; those items say what M40 added.

Statuses:

- **Done**: built as the report asks, pinned by the tests named.
- **Done in M40**: the audit found it partly done, and M40 finished it.
- **Changed on purpose**: built differently, for the reason given (also a row in
  `docs/DESIGN_INVENTORY.md` where it concerns a screen).

Test names are Go test functions (`TestGuardedSending/InDoubtDecision` is a subtest) or web test
files under `web/src`. "Visual QA" means the screen was checked against its board at 390, 768 and
1440 px in English and Vietnamese, with axe. How everything was checked is at the end.

## The report's priority order

The report asked for P0 first (data loss or wrong sends), then P1 (honest numbers), then P2 (UX).

| Priority | What the report listed | Items | Status |
|---|---|---|---|
| P0 | No double sends: results written with a context that isn't cancelled, a stale claim becomes "in doubt", stuck jobs rescued quickly; honest idempotency wording | R2.L1, R2.L2, R2.L3 | Done; M40 added the in-doubt decision and SparkPost's keys |
| P0 | Stop pushing unsubscribers to SES's suppression list | R3.L1 | Done |
| P0 | Routing splits independent of order; save only the streams that changed | R3.L2 | Changed on purpose (a version per stream) |
| P0 | Resume with a baseline; a provider incident always comes with choices | R2.L4, R2.L5 | Done |
| P0 | The send-time check holds when the list grows or content is edited after scheduling; a held send expires | R1.L3 | Done in M40 (audience edits) |
| P0 | Autosave while typing and restore the local copy | R4.L1 | Done |
| P0 | One-click fixes never wipe HTML; Save refuses an empty body | R1.L4 | Done |
| P0 | Never suppress bounces caused by the sender's reputation | R5.L5 | Done in M40 (Retry after fixing) |
| P0 | SES passes the connect step only after a real event | R3.L3, R3.A1 | Done in M40 (the Delivery event) |
| P1 | Real link blocklist lookups; parent, return-path and tracking domains | R1.L1, R1.L2 | Done in M40 (unwatched parent domains) |
| P1 | A/B with a significance test, a minimum sample and the right report data | R5.L1, R4.L6 | Done in M40 (unrecorded decisions) |
| P1 | Bot-click filtering: honeypot, bursts reclassified, window from delivery | R5.L2 | Done in M40 (human clicks everywhere, editable lists) |
| P1 | Opens and clicks with the provider's tracking; delivered state for SMTP | R5.L3 | Done |
| P1 | Mailbox provider by MX; Gmail and iCloud complaints n/a | R2.L8, R2.U2, X.3, X.4 | Done |
| P1 | Warm-up in the send loop for every stream, one time zone | R3.L6 | Done in M40 (automations, transactional, holds) |
| P1 | Check the failover target; a wait before "webhooks silent" | R3.L5, R3.L4, R3.A4 | Done; the triggers changed on purpose |
| P1 | Duration and cost estimates match the engine | R1.L5, R2.L9 | Done in M40 (one volume rule) |
| P2 | Readiness: issues first, friction by risk, a fix's consequences before it applies | R1.U1, R1.U6, R1.U3 | Done in M40 (no guessed links, Undo) |
| P2 | Canary and thresholds by bounce type | R2.A1, R2.A2 | Done in M40 (new IPs read from providers, block bursts) |
| P2 | Write mode, collaboration, brand kit in the composer | R4.A1, R4.A2, R4.A3 | Done in M40 (synced blocks) |
| P2 | Analytics with trends, attribution, three open groups | R5.A2, R5.A1, R5.U1 | Done in M40 (trend lines) |
| P2 | "Help me choose", writing before a provider, SQS polling | R3.U1, R3.U2, R3.A1 | Done in M40 |
| P2 | Mobile as a review and monitoring tool | R4.U6, R2.U4, R1.U5 | Done in M40 (approve on the phone) |
| P2 | Seed test in Review; workspace rules | R1.A1, R1.A2 | Done in M40 (per mailbox provider) |

## 1. Send Readiness

### Logic fixes

**R1.L1 · Link blocklists never looked up on a real install.** `link_domain_listings` was written
only by the demo seed, so the check always passed, in the composer too. Look up DBL and SURBL for
real with a cache; a failed lookup is Unknown.
Spec SR-1, F-4 · **Done**
- Code: `internal/contentcheck/blocklist.go` looks link domains up on Spamhaus DBL and SURBL and
  caches answers for an hour in `link_domain_listings`; a failed lookup is Unknown. The composer's
  inline checks and Review & send (`internal/readiness/assemble.go`) share it.
- Tests: `TestLiveBlocklist`, `TestLinkChecksUseLiveLookups`, `TestLinkDomainsLookedUpLive`,
  `TestUnknownLinkDomains`; `TestQueriedTablesHaveProductionWriters` fails when a table production
  code reads is written only by the demo or tests.

**R1.L2 · The parent domain's listing is skipped.** Only `news.acme.io` was matched, so with
`acme.io` on Spamhaus DBL the screen still said clean. Also check the return-path and tracking
domains.
Spec SR-2 · **Done in M40**
- Code: `internal/readiness/gather.go` checks the sending domain, its parent (organizational)
  domain, the return-path domains and the tracking domain. M40: a parent domain HostRepute doesn't
  watch is looked up live on DBL and SURBL; when nothing can look it up the row is Unknown and says
  why. It is never counted clean.
- Tests: `TestReputationCoversEveryDomain`, `TestParentDomainListingShows`,
  `TestUnwatchedParentDomainIsLookedUp`, `TestUncheckedParentDomainIsLookedUpLive`.

**R1.L3 · At send time only a Block stops the send.** (a) New warnings, Unknowns and a list grown
from 900 to 40,900 pass; (b) the typed number comes from the old review; (c) a scheduled campaign
can be edited without anyone confirming again; (d) a held send runs by itself whenever the recheck
passes.
Spec SR-3 · **Done in M40**
- Code: `internal/readiness/sendtime.go` holds the send on a new warning, an Unknown nobody
  accepted, audience growth over 10%, and content, sender or route edits after scheduling;
  `internal/approvals/send.go` takes the typed count from the fresh run; a held send expires
  3 hours after its time and goes back to Draft (`internal/approvals/jobs.go`). M40: an audience
  edited after scheduling holds the send too (an order-independent digest of its lists, segments
  and exclusions is stored with the confirmation, migration 03300; older confirmations are judged
  by count), and the hold's notification says when it expires.
- Tests: `TestSendTimeRecheckHolds`, `TestAudienceDigest`, `TestSendTimeHoldsOnEdits`.

**R1.L4 · One-click fixes wipe an HTML-only campaign.**
Spec SR-4 · **Done**
- Code: `internal/readiness/fixes.go` edits an HTML-only campaign's HTML, or doesn't offer the fix;
  saving an empty body over a body is refused.
- Tests: `TestFixesKeepHTMLOnlyCampaigns` (both), `TestHTMLOnlyFixes`.

**R1.L5 · Wrong estimates.** (a) A new provider's warm-up cap is ignored ("83 minutes" for
~20 days); (b) a provider that publishes no rate shows "1 minute"; (c) a bounce rate has no
minimum volume, so 1 bounce in 3 messages Blocks.
Spec SR-5 · **Done in M40**
- Code: `internal/sending/engine/estimate.go` plans with the engine's own rules: warm-up caps over
  days, quotas, default rates when a provider publishes none, the A/B wait, best-time spreading
  and routes in parallel. M40: one rule, `manage.TooFewToJudge` (500 messages), judges a
  provider's rates on Review & send and on Deliverability, whose card says "Not enough mail yet".
- Tests: `TestEstimateWarmupCapsOverDays`, `TestEstimateQuota`, `TestEstimateMatchesTheSend`,
  `TestEstimatesFromTheEngine`, `TestRatesNeedVolume`, `TestTooFewToJudge`,
  `TestProviderRatesNeedEnoughMail`, `TestOverviewDemo`, `ProviderHealth.test.tsx`.

**R1.L6 · A DNS timeout becomes a Block saying "record invalid".**
Spec SR-6 · **Done**
- Code: `internal/readiness/gather.go` looks a failed lookup up again during the run; if it still
  fails the row is Unknown, never a Block about an invalid record. M40 puts the DNS answer
  (SERVFAIL, REFUSED, timeout) in the row's technical details.
- Tests: `TestDNSLookupErrorIsUnknown`, `TestTechnicalDetailsOutOfSentences`.

**R1.L7 · Missing checks.** (a) A/B variants B and C; (b) Liquid errors; (c) spam heuristics only
on the subject, with false positives ("gluten-free", "Members only"); (d) the 102 KB limit
measured before tracking; (e) 40 links checked while saying "All 60 respond"; (f) Gmail
complaints not taken from Postmaster.
Spec SR-7, SR-12, CO-4 · **Done**
- Code: (a) `internal/readiness/abtest.go` checks every variant; (b) Liquid errors Block
  (`internal/contentcheck/contentcheck.go`, `internal/readiness/personalization.go`), and a render
  error for one recipient is recorded and shown on the live monitor
  (`internal/sending/engine/render_errors.go`); (c) the heuristics read subject, preheader and body
  without the known false positives; (d) size is measured with tracking inserted
  (`internal/readiness/delivery.go`); (e) link counts say "40 of 60 checked"; (f) Gmail
  Postmaster's spam rate and compliance are checks (`internal/readiness/assemble.go`).
- Tests: `checks_test.go` (readiness), `TestHeuristicsFalsePositives`,
  `TestReviewGathersTheNewChecks`, `TestPersonalizationErrorsBlock`, `render_errors_test.go`,
  `TestPostmasterCompliant`.

### Additions

**R1.A1 · Seed test inside Review.** Inbox, Promotions and Spam for Gmail, Outlook and Yahoo next to
the estimate.
Spec SR-9 · **Done in M40**
- Code: Review & send's placement card (`web/src/features/placement/EntryPoints.tsx`) shows the
  campaign's latest seed test and starts one. M40: it counts Inbox, Promotions (or the provider's
  own tab, such as Outlook's Other) and Spam per mailbox provider ("Gmail · Inbox 0 · Promotions 4
  · Spam 0", `internal/placement/results.go`), with a "Through SES: …" line when routes are
  compared. The board drew a row per sending provider; SR-9 asks per mailbox provider, so the spec
  was followed and the board redrawn.
- Tests: `TestCompareFindsCopiesLateAndDeletesThem`, `ReviewCard.test.tsx`.

**R1.A2 · Workspace rules at Warn or Block.**
Spec SR-10 · **Done**
- Code: `internal/readiness/rules.go`; Settings › Sending rules
  (`web/src/features/sending-rules/RulesPage.tsx`).
- Tests: `TestRuleLinkParam`, `TestRuleShortenersBlock`, `TestRuleSubjectAndText`,
  `TestSendingRules`, `RulesPage.test.tsx`.

**R1.A3 · Leave unengaged contacts out by default, as one line in the audience waterfall.**
Spec SR-11 · **Done**
- Code: `internal/readiness/audience.go`, `internal/sending/engine/audience.go`, migration 03152;
  the waterfall's line with "Include them" (`web/src/features/review/Aside.tsx`).
- Tests: `TestUnengagedExcludedByDefault`, `TestUnengagedLeftOutUntilIncluded`.

**R1.A4 · Gmail Postmaster "Compliant / Needs work" as a check.**
Spec SR-12 · **Done**
- Code: `internal/readiness/assemble.go`.
- Tests: `TestPostmasterCompliant`, `TestPostmasterNeedsWork`, `TestPostmasterNotConnected`.

### UI/UX

**R1.U1 · Issues first in every state.** "Needs attention (2)" on top, "Passed (27)" folded into
chips by group.
Spec SR-13 · **Done in M40**
- Code: `web/src/features/review/Issues.tsx`. M40: the Vietnamese uses the report's words, "Cần xử
  lý · 2" and "Đã đạt · 27".
- Tests: `review.test.ts`; visual QA.

**R1.U2 · One plain sentence per row, "Technical details" when needed.**
Spec SR-13 · **Done in M40**
- Code: `web/src/features/review/Issues.tsx`, `internal/readiness/technical.go`. M40: every record,
  header and code left the sentences for Technical details: SPF and DMARC as `dig` prints them,
  DKIM keys and MAIL FROM per provider, BIMI, List-Unsubscribe headers as sent, HTTP codes of
  links and images, DNS answers and provider error codes. Passed rows have details too.
- Tests: `TestListingTechnicalDetails`, `TestTechnicalDetailsOutOfSentences`,
  `technical.test.tsx`.

**R1.U3 · Fixes preview their consequences and can be undone.** "Apply all 3 fixes" changed a link
to a guessed URL and the routing without a preview; use a drawer listing each change with a
checkbox.
Spec SR-13 · **Done in M40**
- Code: the fixes drawer (`web/src/features/review/FixReview.tsx`) lists each change with a
  checkbox and its consequence ("Leave out 412 people → 31,192 left", "Send only through SES →
  +3 min", `internal/readiness/effects.go`). M40: nothing is guessed. "Replace with destination
  URL" is offered only when the link redirects to a destination that is clean on the live URI
  blocklists, and the fixes endpoint refuses a listed or unknown destination. Applied fixes are
  kept as a batch (migration 03301, `internal/readiness/undo.go`); Undo puts content, route and
  audience back from the toast, under the re-running banner or under the verdict, and is audited
  (`readiness.fixes_undone`).
- Tests: `TestFixNotesAndEffects`, `TestReplaceOnlyWithKnownCleanDestination`,
  `TestListedLinkReplacedOnlyWithKnownDestination`, `TestUndoFixes`, `undo.test.tsx`.

**R1.U4 · Lock Send while fixes apply, and show real progress.**
Spec SR-13, SR-8 · **Done in M40**
- Code: runs are background jobs reporting each group's progress (`internal/readiness/api.go`,
  `RunningBanner` in `web/src/features/review/Verdict.tsx`); the blocklist note says when
  HostRepute's hourly check ran instead of a fixed "8–12 seconds". M40: Send stays locked while any
  fix is being saved, and says why (`sendLockReason` in `web/src/features/review/logic.ts`).
- Tests: `TestRunsAreBackgroundJobs`, `review.test.ts`.

**R1.U5 · The main action states the schedule; a sticky bar on phones.**
Spec SR-13 · **Done**
- Code: `ctaLabel` in `web/src/features/review/logic.ts` ("Schedule for Wed 14 Oct, 09:00 ET");
  `web/src/features/review/issues.css` puts fix buttons under the description and keeps the action
  bar at the bottom on phones.
- Tests: `review.test.ts`; visual QA at 390 px.

**R1.U6 · Friction by risk.** Type the number only for a list ~1.5× its usual size, a first send to
a list, domain or provider, or reputation warnings.
Spec SR-13 · **Done**
- Code: `internal/readiness/confirm.go`, `internal/approvals/send.go`,
  `web/src/features/review/ConfirmSend.tsx`.
- Tests: `TestTypedConfirmationOnlyWhenUnusual`, `TestSendTimeRecheckHolds`.

**R1.U7 · Auto-pause thresholds as a summary.**
Spec SR-13 · **Done**
- Code: `web/src/features/review/Aside.tsx` ("Workspace defaults · Change for this send").
- Tests: visual QA.

**R1.U8 · After accepting an Unknown, say "Ready · blocklist checks skipped".**
Spec SR-13 · **Done in M40**
- Code: `verdictHeading` in `web/src/features/review/logic.ts` names what was skipped ("Ready ·
  blocklist checks skipped", "Ready with warnings · 2 checks skipped") and never calls a blocked
  send ready; the phone's sticky bar uses it too. The button reads "Continue with provider data
  only" only when every skipped check is sender-reputation data, else "Continue without the
  skipped checks". The approval card and the confirmation dialog name the verdict the same way
  (`verdictName`), found by the QA of the final build.
- Tests: `review.test.ts`, `TestAssembleIncompleteAndEmpty`.

**R1.U9 · Test sends while Blocked.**
Spec SR-13 · **Done**
- Code: Review & send keeps the test card while Blocked
  (`web/src/features/review/ReviewPage.tsx`); a test email goes through the System route whatever
  the verdict (`internal/campaigns/composer_api.go`).
- Tests: visual QA of `SendReadinessBlocked`.

**R1.U10 · "Send without approval" in an "Emergency…" menu, with a reason.**
Spec SR-13 · **Done**
- Code: `web/src/features/review/Actions.tsx` (More › Emergency send without approval…);
  `internal/approvals/send.go` (Owners only, a reason and the password).
- Tests: `TestReviewAndApprovals`.

## 2. Guarded sending and the live monitor

### Logic fixes

**R2.L1 · Double sends.** Results were written with the job's context; on SIGTERM, a database error
or the 30 s timeout after the provider accepted, the stale claim was taken back and sent again.
Spec GS-1 · **Done**
- Code: `internal/sending/engine/loop.go` writes results with `context.WithoutCancel`; the provider
  call has its own context; a batch whose outcome is unknown is marked in doubt, never resent
  blindly (`internal/sending/engine/doubt.go`).
- Tests: `TestGuardedSending/TimeoutLeavesMessagesInDoubt`, `/StoppedWorkerRecordsItsBatch`,
  `/DatabaseErrorLeavesMessagesInDoubt`, `/DatabaseErrorWithIdempotentProvider`.

**R2.L2 · Few adapters send an idempotency key while the UI claims "No message was sent twice".**
Spec GS-1, F-6 · **Done in M40**
- Code: `internal/providers/idempotency.go` lets an adapter say whether it recognizes a resend and
  for how long; the live monitor says so per provider (`web/src/features/live/Decision.tsx`)
  instead of the blanket claim. M40: SparkPost recognizes a message it already took for 24 hours
  (`internal/providers/sparkpost/idempotency.go`); Resend stays out because its keys cover a whole
  batch request, not a message; the adapter contract suite checks every adapter that claims it.
- Tests: `/TimeoutLeavesMessagesInDoubt`, `/DatabaseErrorWithIdempotentProvider` (above);
  `TestRetriedSendIsNotSentTwice`, `TestBatchKeysAreNotPerMessage`, the contract suite's
  idempotency check (`internal/providers/contracttest`).

**R2.L3 · The fix: context.WithoutCancel, a graceful stop, a stale claim is "not known whether
sent", stuck jobs rescued within minutes.**
Spec GS-1 · **Done in M40**
- Code: the worker stops River gracefully (`SoftStopTimeout`, `internal/jobs/jobs.go`); a stale
  claim is in doubt (`internal/sending/engine/doubt.go`); a sending job whose worker is gone is
  rescued within minutes (`internal/sending/engine/rescue.go`). M40: the live monitor shows the
  in-doubt decision per provider: whether it recognizes a resend and until when, the two choices
  (send them again or leave them out), the recommended one and who can decide (migrations 03310,
  03311). Resuming with only in-doubt messages left asks for that decision first; cancelling
  leaves them out; the report counts them.
- Tests: `TestGuardedSending/StuckJobsRescued`, `/SendInDoubtAgain`, `/EventSettlesInDoubt`,
  `/InDoubtDecision`, `/InDoubtAfterTheWindow`, `/CancelLeavesInDoubtOut`;
  `InDoubtPanel.test.tsx`, `model.test.ts`.

**R2.L4 · Resuming a campaign- or route-level pause re-trips 2 s later; "Resume with changes" is
ignored there; thresholds can't change while sending.**
Spec GS-2 · **Done**
- Code: `internal/sending/engine/rearm.go` gives every resume a new baseline; resume with changes
  works at every scope; thresholds can be set while sending or paused.
- Tests: `TestGuardedSending/ResumeDoesNotRetripOnOldEvents`, `/CampaignResumeRearms`,
  `/CampaignResumeWithChanges` (added in M40), `/RouteResumeWithChanges`, `/ThresholdsWhilePaused`.

**R2.L5 · Outages found by the health job have no options; Retry never asks again while Down.**
Spec GS-3 · **Done**
- Code: `internal/providers/manage/health.go` hands an outage to
  `internal/sending/engine/outages.go`, which holds the shares with Retry, Move and Cancel share;
  Retry asks the provider again.
- Tests: `TestGuardedSending/HealthJobOutageHasOptions`, `/RetryAsksAgainWhileDown`.

**R2.L6 · Behaviour vs UI.** (a) The API-errors threshold is display-only; (b) throttling or a
used-up quota isn't an outage; (c) Move ignores the target's quota and sends the A/B holdout early;
(d) a decision on a share silently undoes a person's Pause.
Spec GS-4, GS-5 · **Done**
- Code: (a) `internal/sending/engine/routeerrors.go` holds the route; (b) lasting throttling and a
  used-up quota hold it too; (c) Move vets the target with `state.Vet`
  (`internal/providers/state/vet.go`) and keeps the holdout for the winner; (d) a person's Pause
  stays.
- Tests: `TestGuardedSending/APIErrorThresholdHoldsRoute`, `/LastingThrottlingHoldsRoute`,
  `/UsedUpQuotaHoldsRoute`, `/MoveKeepsHoldoutsAndPause`, `TestVet`.

**R2.L7 · Complaint gaps.** (a) Gmail and iCloud rows always show "0.00% OK"; (b) the guard stops
watching when the send ends; (c) no complaint protection under 500 per mailbox provider and 1,000
in all; (d) with SMTP mail is never "delivered", so the complaint guard is off.
Spec F-2, GS-6, GS-7 · **Done in M40**
- Code: (a) complaint rates use only mail delivered where complaints are reported
  (`internal/mailbox`); Gmail and iCloud read n/a; (b) the watch after the send
  (`internal/sending/engine/watch.go`); (d) plain SMTP's deliveries are inferred from its
  acceptance and DSNs. (c) M40: complaints trip a send of any size once the lower bound of a
  one-sided 95% Wilson interval is over the limit (from 3 complaints), per mailbox provider and
  campaign-wide; the watch uses the same rule.
- Tests: `TestGuardedSending/ComplaintWatchAfterTheSend`, `/SMTPInfersDelivery`,
  `/SmallSendComplaintsPause`, `/SmallSendWatchAlerts`, `TestComplaintsCountWhereReported`,
  `TestClearlyOverAtAnyVolume`, `TestThresholdState`, `TestWatchAlertsOncePerComplaint`.

**R2.L8 · Mailbox providers from a fixed domain list.** Google Workspace and Microsoft 365
recipients fall into "Other".
Spec F-1 · **Done**
- Code: `internal/mailbox` classifies by MX with a domain cache, for contacts, routing, the engine,
  Readiness, transactional mail, reports and warm-up.
- Tests: `TestSending/MailboxFromMX`, `classifier_test.go`.

**R2.L9 · ETA wrong when one slow route is left** (1,000 on Postal at 2/s shows ~1 minute).
Spec GS-8 · **Done**
- Code: `internal/sending/engine/live.go` times each route by its own rate.
- Tests: `TestGuardedSending/SendsShareThrottles`, `TestETAOfTheSlowRouteLeft` (added in M40:
  about 8 minutes), `model.test.ts`.

### Additions

**R2.A1 · Canary.** The 5% most engaged plus seeds, wait 20–30 minutes, continue if fine; its own
segment on the progress bar.
Spec GS-10 · **Done in M40**
- Code: `internal/sending/engine/canary.go`; the live monitor's canary segment
  (`web/src/features/live/Canary.tsx`). M40: a dedicated IP that has never sent for the workspace
  starts a canary. Each send records the IPs of a dedicated provider once per run
  (`workspace_sending_ips`, migration 03313), and Review & send and the live monitor name the IP
  and its provider. A provider's dedicated IPs come from its own API (SES, SendGrid, Mailgun,
  SparkPost and Brevo, each in its `ips.go`) when it is connected, when its connection step is
  repeated and every day (`internal/providers/manage/ips.go`); for the others a person sets them
  on Sending providers (`web/src/features/providers/SendingIPs.tsx`, migration 03350). Before M40
  only the demo seed wrote them, so the trigger never fired on a real install.
- Tests: `TestGuardedSending/CanaryPassesThenEveryone`, `/CanaryGateHoldsThenStop`,
  `/CanaryChoices`, `/CanaryForNewSendingIP`, `/CanaryForReportedIP` (a connected provider's
  reported IP starts a canary on the first send, not the second), `TestSeedGateNeedsArrivals`,
  `TestCanaryChoice`, `TestSendingIPsBackfill`, `canary.test.ts`; `TestCleanIPs`,
  `TestSendingIPsFollowsPages`, `TestSendingIPs` (each adapter), `TestRefreshIPs`, `TestIPsWorker`,
  `TestConnectReadsSendingIPs`, `TestSetProviderIPs`, `TestCheckProviderIPs`,
  `TestReviewReadsRefreshedIPs`, `TestWarmupUsesReportedIPs`, `SendingIPs.test.tsx`.

**R2.A2 · Thresholds by bounce type.** Policy and reputation blocks (5.7.x, Outlook S3150, Yahoo
TSS04) stop early; unknown users reflect list quality; deferral rate per mailbox provider.
Spec GS-11 · **Done in M40**
- Code: `internal/sending/engine/bytype.go` and `internal/providers/bounce` keep block/policy
  bounces, unknown users and deferrals apart, per mailbox provider. M40: block and policy bounces
  stop a send early on their own: at any volume once clearly over the limit (the Wilson bound),
  and on a burst at one named mailbox provider: 3 or more within 10 minutes, clearly over the limit
  among what it was sent in those minutes (not in the first 10 minutes after a resume).
- Tests: `TestGuardedSending/TypedLevels`, `/BlockPolicyFloor`, `/BlockPolicyBurst`,
  `TestTypedThresholds`, `TestCheckTyped`, `TestBlockBurst`, `TestClearlyOverAtAnyVolume`.

**R2.A3 · SMTP-reply rules with a UI** ("Yahoo paused 2 hours, everywhere else keeps sending").
Spec GS-12 · **Done in M40**
- Code: `internal/sending/engine/replies.go`; Traffic rules
  (`web/src/features/traffic/TrafficRulesPage.tsx`). M40: every workspace starts with the default
  reply rules (`internal/sending/rules/reply.go`; migration 03312 gives them to workspaces that had
  none).
- Tests: `TestGuardedSending/ReplyRules`, `/TrafficRulesAPI`, `traffic-model.test.ts`,
  `TestNewWorkspaceStartsWithDefaultReplyRules`, `TestDefaultReplyRulesBackfill`.

**R2.A4 · Watch complaints for 72 hours after the send, with Gmail Postmaster.**
Spec GS-6 · **Done in M40**
- Code: `internal/sending/engine/watch.go`. M40: with Traffic rules › After the send › Gmail
  Postmaster on, the watch alerts once when a day's Postmaster spam rate (Pacific time) passes the
  limit; nobody is suppressed for it.
- Tests: `TestGuardedSending/ComplaintWatchAfterTheSend`, `/PostmasterAfterTheSend`.

**R2.A5 · Rates per mailbox provider during a send; after an outage, retry by itself with one
test message.**
Spec GS-13, GS-3 · **Done in M40**
- Code: rates per mailbox provider during the send (`internal/sending/engine/rates.go`,
  `web/src/features/live/RatesPanel.tsx`); Retry tests the provider with one message. M40: a held
  share nobody has decided on tests its provider by itself every 60 seconds (a seed mailbox, or
  one of its own messages), spends no attempts on a failed test and resumes after 3 tests in a row
  go through; the share reads "Testing". A person's Move or Cancel ends the tests; their Retry
  keeps them going.
- Tests: `TestGuardedSending/RatesDuringSend`, `/RetryResumesAfterThreeTests`,
  `/HeldShareTestsItsProviderOnItsOwn`, `/HeldShareTestsWithItsOwnMessages`,
  `/DecisionEndsTheTests`.

### UI/UX

**R2.U1 · "Collecting evidence".** The bar turns red at the limit while the engine waits for enough
sample, and says so.
Spec GS-14, GS-6 · **Done**
- Code: `internal/sending/engine/thresholds_live.go`, `web/src/features/live/model.ts`.
- Tests: `TestThresholdState`, `model.test.ts`.

**R2.U2 · Gmail and iCloud complaints read "n/a — see Postmaster (daily)".**
Spec GS-14, F-2 · **Done**
- Code: `internal/sending/engine/live.go`, `web/src/features/live/LiveParts.tsx`, `Mobile.tsx`.
- Tests: `TestComplaintsCountWhereReported`, `TestGuardedSending/ComplaintWatchAfterTheSend`.

**R2.U3 · No permission jargon:** "Linh and Marcus can continue", not "anyone with
campaigns:pause".
Spec GS-14, X-1 · **Done** (fixed by the audit)
- Code: `whoLine` and `pausedText` in `web/src/features/live/model.ts` name the people, or "someone
  who can send" when nobody is named.
- Tests: `model.test.ts`.

**R2.U4 · On phones Pause is the main button; Cancel in a menu.**
Spec GS-14 · **Done**
- Code: `web/src/features/live/Mobile.tsx` (Adjust rates and Cancel under More actions).
- Tests: visual QA at 390 px.

**R2.U5 · Notice frozen data when a proxy buffers the stream, fall back to polling; role="alert"
only when the state changes.**
Spec GS-14 · **Done** (fixed by the audit)
- Code: `web/src/features/live/useLiveSend.ts` (stale after 90 s → polls every 10 s); the stream
  sends a heartbeat every 15 s; role="alert" is on the incident's title only (`Decision.tsx`).
  While the stream is down only the progress bar fades, as on LiveMonitorStates; the counts keep
  their contrast (`live.css`, found by the QA of the final build). An outage's title says what the
  provider returned in the reader's language ("Postal trả về lỗi kết nối trong 3 phút").
- Tests: `useLiveSend.test.ts`, `TestGuardedSending/StreamHeartbeats`.

**R2.U6 · Keep "What tripped / Likely causes / Recommended / Choose how to continue".**
Spec GS-14 · **Done**
- Code: `web/src/features/live/Decision.tsx`.
- Tests: `model.test.ts`.

## 3. Your own providers: routing, failover, warm-up

### Logic fixes

**R3.L1 · Unsubscribers pushed to SES's account-level suppression list as BOUNCE.**
Spec PR-1 · **Done**
- Code: `internal/store/queries/suppression_sync.sql`, `internal/suppressions/sync.go`: only hard
  bounces and complaints with scope "all" go to SES, to the configuration set (tenant) when there
  is one; removals sync too.
- Tests: `TestSuppressionPushKeepsUnsubscribesOffSES`,
  `TestSuppressionSyncOnlyReadsTheSESAccountList`.

**R3.L2 · Saving reorders splits and moves addresses.** Use rendezvous hashing; save only the
streams that changed, with If-Match.
Spec PR-2 · **Changed on purpose**
- Code: rendezvous hashing (`internal/routing/routing.go`): reordering a split moves nobody. The
  editor saves only the streams it changed, each with the version it started from
  (`internal/routing/version.go`); a stale version is refused (409 `routing_changed`). A version
  per stream in the request replaces one If-Match header, so saving one stream never trips over
  someone's change to another.
- Tests: `TestReorderingASplitMovesNobody`, `TestRoutingSavesChangedStreamsWithVersions`,
  `routing-editing.test.ts`.

**R3.L3 · SES onboarding can finish without an event stream** ("Healthy · no events yet"
forever).
Spec PR-3 · **Done in M40**
- Code: `internal/connect/eventcheck.go` lets setup finish only after a real event. M40: the SES
  check waits for the simulator's Delivery event, not any event; a provider that never reported an
  event is measured from its connection and flagged once mail is 30 minutes overdue
  (`internal/providers/manage/health.go`).
- Tests: `TestSESEventCheckGatesFinish`, `TestNeverAnEventIsFlagged`.

**R3.L4 · False "webhooks silent"** after a quiet night; polling never updates `last_event_at`.
Spec PR-6 · **Done**
- Code: `internal/providers/manage/health.go` (silence needs overdue messages, not a quiet clock);
  polling updates event freshness.
- Tests: `TestNoSilentAlarmAfterAQuietPeriod`, `TestPollingUpdatesEventFreshness`.

**R3.L5 · Failover and Move don't check the target**; a MessageRejected drops the whole share.
Spec PR-7, GS-5 · **Done**
- Code: `state.Vet` (`internal/providers/state/vet.go`) checks domain, quota and sandbox for failover
  and Move; an identity rejection holds the route instead of dropping the share.
- Tests: `TestFailoverVetsTheFallback`, `TestRejectionsAreWatched`, `TestVet`,
  `TestGuardedSending/IdentityErrorHoldsRoute`.

**R3.L6 · Warm-up only partly applied.** (a) With one provider the first 200 go to random
contacts; (b) Automations and Transactional aren't capped; (c) Pause doesn't freeze the plan;
(d) a slowdown holds one day, then the cap jumps; (e) no minimum sample; (f) caps counted per UTC
day but read in local time.
Spec PR-8 · **Done in M40**
- Code: (a) the most engaged go first (`internal/sending/engine/audience.go`); (c) Pause freezes
  the plan (`internal/warmup/service.go`); (e) a minimum sample before a day is judged and (f) one
  workspace day (`internal/warmup/days.go`). M40: (b) automations wait at the cap and while the
  plan is paused, or overflow to a warm provider, per mailbox part; transactional mail never waits:
  it goes to a warm provider of its route, or is sent and counted over the cap (migration 03320);
  (d) a hold keeps the plan on its day, and the ramp resumes from the held cap one step at a time
  (`internal/warmup/job.go`).
- Tests: `TestWarmupInTheSendLoop`, `TestPauseFreezesThePlan`, `TestJudge`,
  `TestPlanMovesByCountedDays`, `TestWarmupOtherStreams`, `TestHoldKeepsThePlanOnItsDay`,
  `TestWarmupEndpoints`.

**R3.L7 · The CloudFormation path only works on a machine in AWS.**
Spec PR-5 · **Done**
- Code: `internal/connect/host.go` detects where the server runs; off AWS, setup uses IAM keys.
- Tests: `TestHostDetection`.

### Additions

**R3.A1 · SQS polling for SES; pass the connect step only after a real delivery event.**
Spec PR-4, PR-3 · **Done in M40**
- Code: `internal/providers/ses/queue.go`, `internal/providers/eventqueue/eventqueue.go`. M40: the
  check waits for the Delivery event of a message to success@simulator.amazonses.com.
- Tests: `TestPollStoresEventsThenAcks`, `TestEventsModeChangesWithItsAuditEntry`,
  `TestSESEventCheckGatesFinish`, `TestQueueReceivesAndAcks`, `TestQueueOffMakesNoCall`,
  `TestQueueErrors`, `TestQueueProvisioning`.

**R3.A2 · SES tenants per agency client and workspace.**
Spec PR-11 · **Done**
- Code: `internal/providers/ses/tenant.go`, `internal/connect/provision.go`.
- Tests: `TestTenantProvisioning`.

**R3.A3 · Warm-up by volume and signals; caps per mailbox provider; overflow to a warm provider.**
Spec PR-8 · **Done**
- Code: `internal/warmup/days.go` (the next day only after 80% of the cap went out with good
  numbers), per-mailbox caps, overflow (`internal/sending/engine/warmup_send.go`).
- Tests: `TestPlanMovesByCountedDays`, `TestMailboxCaps`,
  `TestWarmupInTheSendLoop/OverflowToWarmProvider`.

**R3.A4 · Precise failover:** state the triggers (408/429/5xx within X minutes, no DSN after N
seconds) and check the target automatically.
Spec PR-7, PR-14 · **Changed on purpose** (inventory #517)
- Code: the target is checked automatically (`state.Vet`); each stream states the trigger the
  health check really uses (an error rate or two failed probes), since F-6 forbids naming
  triggers the code doesn't act on.
- Tests: `TestFailoverVetsTheFallback`, `TestCampaignSimulation`.

**R3.A5 · Weighted splits for conditions; apply "spread across mailbox providers".**
Spec PR-12 · **Done**
- Code: `internal/routing/service.go`, `web/src/features/providers/RoutingPage.tsx`; warm-up reads
  the spread setting (`internal/warmup/service.go`).
- Tests: `TestConditionSplitsByWeight`, `TestMailboxCaps`.

### UI/UX

**R3.U1 · "Help me choose" in the catalog.**
Spec PR-13 · **Done**
- Code: `web/src/features/connect/catalog-model.ts`, `CatalogPage.tsx`.
- Tests: `catalog-model.test.ts`, `TestCatalog`.

**R3.U2 · Write campaigns before connecting a provider; only Send is locked; a demo mode.**
Spec PR-13 · **Done in M40** (test)
- Code: New campaign and the composer work without a provider; Explore with demo data.
- Tests: `TestExploreWithDemoData`, `TestWriteBeforeAProvider` (added in M40).

**R3.U3 · SES setup: IAM keys by default off AWS; check the endpoint from outside.**
Spec PR-5, PR-9 · **Done**
- Code: `internal/connect/host.go`, `internal/providers/manage/reach.go` (no call to its own URL).
- Tests: `TestWebhookTestView`, `TestPublicURLName`, `TestHostDetection`.

**R3.U4 · Routing editor.** (a) A drag-and-drop priority list with its impact before saving;
(b) a warning before leaving unsaved changes (the editor remounted and lost them); (c) the route
tester simulates a whole campaign, warm-up and failover included.
Spec PR-14, PR-2 · **Done in M40**
- Code: (a) `web/src/features/providers/RoutingPage.tsx` with drag, Move up and Move down. M40: the
  impact sentence for every stream, from its last 30 days' recipients or the newest contacts
  (migration 03321). (b) M40: a failed background refresh keeps the editor and its edits under a
  banner, and the bulk-terms view has the leave warning and the conflict note. (c)
  `internal/routing/simulate.go`; M40: engagement tiers, per-mailbox caps, a hold without overflow,
  and where a down provider's mail goes under the failover policy. The tester and the condition
  chips name lists, categories and mailbox providers, never their ids (`valueNames`, found by
  the QA of the final build).
- Tests: `routing-order.test.ts`, `TestCampaignSimulation`, `TestRoutingImpactForEveryStream`,
  `RoutingPage.test.tsx`, `routing-editing.test.ts`, `TestCampaignSimulationWarmupAndFailover`.

**R3.U5 · The two-level Email/SMS/Push/WhatsApp header is too dense.**
Spec PR-15 · **Done in M40**
- Code: one channel menu heads Email, SMS, Push and WhatsApp
  (`web/src/features/channels/ChannelMenu.tsx`); the old switch is gone.
- Tests: `ChannelMenu.test.tsx`.

**R3.U6 · No over-promises:** "about 10 minutes" (really 10–15 on AWS, longer off AWS, up to 24 h
for production access) and "Events ✓ subscribed".
Spec PR-15, F-6 · **Done in M40**
- Code: setup time is said the same way in the connect screens, instance setup, the server's
  catalog and the site; "Events" is ticked only after a real event (R3.L3).
- Tests: `setup-time.test.ts`, `TestCatalog`, `TestDemoSetupAndSettings`.

## 4. Composer

### Logic fixes

**R4.L1 · No autosave while typing inside a text block; the offline copy is never read back.**
Spec CO-1 · **Done**
- Code: `web/src/features/composer/Composer.tsx` (typing marks the draft changed and saves every
  30 s), `journal.ts` (the browser copy), `Banners.tsx` (recovered edits).
- Tests: `journal.test.ts`, `Banners.test.tsx`, `ComposerPage.test.tsx` (the 30-second autosave,
  added in M40).

**R4.L2 · Other data-loss paths.** (a) "Load theirs" drops your changes; (b) switching language
skips the leave warning; (c) a subject typed during a slow save; (d) restore with unsaved changes
fails; (e) only 50 versions.
Spec CO-2 · **Done**
- Code: `Banners.tsx`, `web/src/features/languages/composerParts.tsx`, `saveQueue.ts`,
  `Dialogs.tsx`; `internal/campaigns/content.go` keeps 300 autosaves and every named version, paged.
- Tests: `TestComposerLoadTheirsKeepsYours`, `languageNav.test.tsx`, `saveQueue.test.ts`,
  `TestComposerRestoreAfterSavingEdits`, `TestComposerNamedVersionsAndPages`.

**R4.L3 · The server stores the client's HTML but checks the MJML.** Recompile on the server, with
parity tests against mjml-browser.
Spec CO-3 · **Done in M40**
- Code: `internal/mjml/verify.go`, `internal/campaigns/content.go`. M40: an agent's drafts (MCP
  `create_campaign`, `update_campaign_content`) go through the composer's save
  (`internal/campaigns/agent.go`): compiled, checked, versioned as changed by an agent, audited.
- Tests: `TestParityWithMJMLBrowser`, `TestVerifyRefusesMismatches`, `parity.test.ts`,
  `TestComposerChecksStoredHTML`, `TestAgentDraftsAreCompiledAndVersioned`.

**R4.L4 · Liquid errors block in the composer but not in Readiness; one recipient's render error
fails silently.**
Spec CO-4 · **Done**
- Code: `internal/readiness/personalization.go`, `internal/sending/engine/render_errors.go`.
- Tests: `TestPersonalizationErrorsBlock`, `TestGuardedSending/RenderErrorNotice`, `model.test.ts`.

**R4.L5 · "Preview as contact" gets names wrong** ("Hi Nguyen", "Mary" for "Mary Ann", test sends
too; `frist_name` passes).
Spec CO-5, CO-4 · **Done in M40**
- Code: `internal/campaigns/compose.go` renders with the contact's own fields; unknown fields are
  flagged. M40: a test send is personalized as the contact being previewed (or a sample without a
  first name), and the audit entry names it.
- Tests: `TestComposerPreviewUsesTheContactsOwnFields`, `TestCheckUnknownFields`,
  `Personalization.test.tsx`, `PreviewSendTest.test.tsx`.

**R4.L6 · A/B.** (a) B and C unchecked and preview B shows A; (b) a send-time test picks the winner
before the variants went out, in the workspace's time zone; (c) "+ variant" fails autosave.
Spec CO-6, AN-1 · **Done**
- Code: `internal/campaigns/personalize.go`, `web/src/features/composer/PreviewView.tsx`,
  `internal/sending/engine/ab.go` (recipients' local time).
- Tests: `TestComposerChecksEachVariant`, `ABPanel.test.tsx`, `TestABEndToEnd/SendTimeInLocalTime`,
  `ComposerPage.test.tsx` (variant B's preview, added in M40).

**R4.L7 · Editing an approved campaign revokes the approval silently while the badge still says
"Approved"; "Send test" after a failed save sends old content.**
Spec CO-7 · **Done in M40**
- Code: the editor asks first (409 `approval_would_cancel`, `Approval.tsx`); Send test after a
  failed save sends nothing (`SendTest.tsx`). M40: once the save cancels the approval, the header
  stops saying "Approved".
- Tests: `Approval.test.tsx`, `TestComposerEditingAnApprovedCampaignAsksFirst`.

### Additions

**R4.A1 · Write mode** ("/" menu, markdown, clean paste from Google Docs or Notion).
Spec CO-9 · **Done**
- Code: `web/src/features/composer/write/WriteView.tsx`, `markdown.ts`, `paste.ts`.
- Tests: `write.test.ts` (Notion paste added in M40), `WriteView.test.tsx`.

**R4.A2 · Collaboration:** @mentions, reviewer links, who's editing, merging edits to different
sections.
Spec CO-10 · **Done**
- Code: `internal/campaigns/comments.go`, `review.go`, `presence.go`, `internal/mjml/merge.go`.
- Tests: `TestComposerCommentsAndMentions`, `TestComposerReviewLinks`, `TestComposerPresence`,
  `TestComposerMergesDifferentSections`, `TestMergeSections`, `Collab.test.tsx`,
  `Comments.test.tsx`.

**R4.A3 · A shared brand kit, created from a website; synced blocks like universal content.**
Spec CO-11 · **Done in M40**
- Code: `internal/brandkit` (importer, contrast). M40: any section can be saved as a named synced
  block (migration 03330), listed under "Your blocks", unlinked in one campaign, renamed or
  deleted from the brand kit; deleting one leaves an ordinary section in drafts and scheduled
  campaigns, while campaigns sent, sending or awaiting approval keep theirs.
- Tests: `TestBrandKit`, `TestImportFromWebsite`, `TestContrastAndApply`, `TestSyncedBlocks`,
  `TestNamedSyncedBlocks`, `TestUnmarkSynced`, `BrandKitPage.test.tsx`, `BrandKit.test.tsx`.

**R4.A4 · Compress and resize large images.**
Spec CO-12 · **Done**
- Code: `internal/media/optimize.go`.
- Tests: `TestUploadIsResizedAndStripped`, `Dialogs.test.tsx`.

**R4.A5 · Optional real-render previews through Litmus or Mailgun Inspect.**
Spec CO-13 · **Done**
- Code: `internal/renders`.
- Tests: `TestLitmusRenders`, `TestMailgunInspectRenders`, `RealRenders.test.tsx`.

### UI/UX

**R4.U1 · Less chrome above the canvas.**
Spec CO-14 · **Done**
- Code: `web/src/features/composer/Header.tsx`, `InboxStrip.tsx`, the "Sending to" drawer.
- Tests: `Chrome.test.tsx`.

**R4.U2 · Liquid tokens as chips with a fallback editor.**
Spec CO-14 · **Done**
- Code: `web/src/features/composer/editor/liquid.ts`, `LiquidChips.tsx`.
- Tests: `liquid.test.ts`, `LiquidChips.test.tsx`.

**R4.U3 · The Style panel in the writer's words; an advanced mode for designers.**
Spec CO-14 · **Done in M40**
- Code: `web/src/features/composer/Inspector.tsx`. M40: `stylePresets.ts` gives text, button,
  image, column, hero and divider plain controls (spacing Small, Medium, Large); the raw attributes
  are under Advanced.
- Tests: `stylePresets.test.ts`, `Chrome.test.tsx`.

**R4.U4 · A template gallery with real previews.**
Spec CO-14 · **Done in M40**
- Code: M40: `GET /templates/{id}/preview` compiles the template with its Liquid fallbacks and
  strips scripts, forms, frames, links and web font imports; `Thumb.tsx` shows it in a sandboxed,
  inert frame, scaled and loaded lazily.
- Tests: `TestTemplateGallery`, `TestPicture`, `Gallery.test.tsx`.

**R4.U5 · Warnings can be dismissed with a reason; a Block can't.**
Spec CO-14 · **Done in M40**
- Code: `web/src/features/composer/Checks.tsx`, `internal/campaigns/dismissals.go`. M40: the API
  refuses to dismiss a blocking check (422).
- Tests: `TestComposerDismissWarnings`, `TestBlockingChecksCantBeDismissed`,
  `TestSponsorBlockInComposerAndReview`.

**R4.U6 · On phones, a quick review:** edit text and subject, preview, send a test, approve.
Spec CO-14 · **Done in M40**
- Code: `web/src/features/composer/QuickReview.tsx`. M40: approvers approve or ask for changes
  there.
- Tests: `Chrome.test.tsx`, `Approval.test.tsx`.

**R4.U7 · Ask before editing an approved campaign.**
Spec CO-7 · **Done**
- Code: `web/src/features/composer/Approval.tsx`.
- Tests: `Approval.test.tsx`.

**R4.U8 · Keyboard:** Enter starts editing text, arrow keys move in Layers.
Spec CO-8 · **Done**
- Code: `web/src/features/composer/editor/keys.ts`, the Layers tree.
- Tests: `keys.test.ts`, `Layers.test.tsx`.

## 5. Honest analytics

### Logic fixes

**R5.L1 · Production A/B wrong at both ends.** (a) The highest raw rate wins, no test, no minimum;
(b) the report reads data only the demo writes; (c) "B chosen · not significant · 99% confidence".
Spec AN-1 · **Done in M40**
- Code: `internal/abstats` (500 per variant, 95%, a z-test), `internal/sending/engine/ab.go`,
  `internal/reports/abtest.go`. M40: a test decided before decisions were recorded claims no
  confidence ("B chosen · confidence not recorded").
- Tests: `TestDecide`, `TestZTest`, `TestABEndToEnd`, `reports.test.ts`,
  `TestUnrecordedABDecision`.

**R5.L2 · Bot-click filtering is partial.** A scanner's first clicks count as human (opens, unique
clicks, the "clicked" trigger); the window starts at hand-off; fixed IP and user-agent lists; no
honeypot.
Spec AN-2 · **Done in M40**
- Code: `internal/tracking/settle.go` (a click counts once it settles as human, timed from
  delivery), `classify.go`, a honeypot link (`internal/sending/engine/honeypot.go`). M40:
  automations, segments, scores and engagement count only settled human clicks; instance admins
  edit the user agents (migration 03340) and networks on Instance admin › Bot filter, audited. An
  order still credits a click that is settling: buying is a person's act.
- Tests: `TestBotClicks`, `TestClickConditionsCountPeopleOnly`, `TestClickedRuleCountsPeopleOnly`,
  `TestUserAgentList`, `TestPatternChecks`, `BotFilterPage.test.tsx`.

**R5.L3 · Two configurations zero the numbers:** the provider's own tracking, and SMTP never
"delivered".
Spec AN-4, GS-7 · **Done**
- Code: `internal/tracking/provider.go`; SMTP deliveries inferred.
- Tests: `TestProviderTracking`, `TestGuardedSending/SMTPInfersDelivery`, `TestDeliveredCountsOnce`,
  `TestSuccessDSN`.

**R5.L4 · Counting events, not status changes;** failed writes dropped; no rebuild job.
Spec AN-3 · **Done**
- Code: `internal/counters` (transitions only, a rebuild job), `internal/tracking/record.go`
  (retries).
- Tests: `TestCounters`, `TestDiffOnlyOnTransitions`, `TestDeferredCountsOnce`.

**R5.L5 · "Add N to suppression" on "Blocked · policy or reputation".** Replace with "See cause /
delisting" and "Retry after fixing".
Spec AN-7 · **Done in M40**
- Code: `internal/reports/bounces.go` never offers suppression for sender-side blocks and
  `suppress.go` refuses it; the Investigate dialog explains the cause and delisting. M40: "Retry
  after fixing" drafts the issue again for the addresses blocked for the sender's reputation that
  never got it, once each (`internal/campaigns/retry.go`, migration 03341). The draft goes through
  Review & send like any campaign, rather than resending in one click.
- Tests: `TestSenderBlocksAreNeverSuppressed`, `TestReportExportAndSuppress`, `TestRetryBlocked`,
  `TestAssembleRetry`, `BlockedRetry.test.tsx`.

**R5.L6 · Click map wrong with A/B content:** B's links drawn on email A.
Spec AN-5 · **Done in M40**
- Code: `internal/reports/clickmaps.go`. M40: a content test's click map shows each variant on its
  own email, without an "Everyone" map; the top-link insight names its variant; exports, MCP and
  AI read per-variant maps.
- Tests: `TestABEndToEnd/ContentClickMaps`, `TestMarkNumbersLikeTheEngine`, `TestTopLinkInsight`.

**R5.L7 · Behind a reverse proxy with no trusted proxies every hit has the proxy's IP.**
Spec AN-8 · **Done**
- Code: `internal/proxycheck` notices it and says what to set.
- Tests: `TestUntrustedProxy`, `TestPublicProxyIsOneToCheck`.

### Additions

**R5.A1 · Growth and revenue per issue.**
Spec AN-12 · **Done**
- Code: `internal/reports/growth.go`, `web/src/features/reports/Insights.tsx`.
- Tests: `TestReportGrowth`, `TestConversionAttribution`.

**R5.A2 · Comparison with earlier issues, and a trend line.**
Spec AN-11 · **Done in M40** (the median changed on purpose, inventory #525)
- Code: `internal/reports/compare.go` compares each tile with the median of the last six issues sent
  to the same lists and segments; the median, so one unusual issue doesn't move the baseline. M40:
  every tile draws its trend over those issues and this one once the results are final.
- Tests: `TestCompareTiles`, `TestCampaignReport`, `ReportParts.test.tsx`.

**R5.A3 · First-party analytics for web posts.**
Spec AN-13 · **Done**
- Code: `internal/website/views.go`.
- Tests: `TestSiteViews`.

**R5.A4 · Ask the data with AI; MCP tools for reports.**
Spec AN-14 · **Done**
- Code: `internal/ai/reportask.go`, `internal/mcpserver`.
- Tests: `TestReportNumbers`, `TestMCPServer`, `TestShareCampaignReportTool`.

### UI/UX

**R5.U1 · Opens in three groups (human, machine, not measurable) with a measurable open rate;**
Apple's privacy protection concerns everyone reading in Apple Mail.
Spec AN-10, AN-9 · **Done**
- Code: `internal/reports/types.go`, `web/src/features/reports/Tiles.tsx`.
- Tests: `TestCampaignReport`, `TestClassifyOpen`.

**R5.U2 · "N bot clicks filtered" under the total; a Human / Bot switch on the click map; unique
vs total clicks said.**
Spec AN-15, AN-2 · **Done**
- Code: `Tiles.tsx`, `ClickMap.tsx`.
- Tests: `TestCampaignReport`, `TestBotClicks`.

**R5.U3 · Insights with actions** ("Outlook bounces 1.0%, 2.5× elsewhere → open SNDS").
Spec AN-15 · **Done** (tests added in M40)
- Code: `internal/reports/insights.go`, `Insights.tsx`.
- Tests: `TestMailboxBouncesInsight`, `ReportParts.test.tsx`.

**R5.U4 · Without significance, "No clear winner → sending A"; the bar shows the metric used.**
Spec AN-15, AN-1 · **Done**
- Code: `web/src/features/reports/ABTestCard.tsx`.
- Tests: `reports.test.ts`, `TestABEndToEnd/NoClearWinner`.

**R5.U5 · A y-axis on the 72-hour chart.**
Spec AN-15 · **Done**
- Code: `web/src/features/reports/EngagementChart.tsx`.
- Tests: `reports.test.ts`.

**R5.U6 · Share links:** choose what's shared, expiry by default, several links.
Spec AN-16 · **Done**
- Code: `internal/reports/shares.go` (30 days by default, 1–90), `ShareCard.tsx`.
- Tests: `TestReportShares`, `TestShareDaysLimit`, `TestShareExpiryBackfill`,
  `TestSharedReportWarmup`.

## Cross-cutting

**X.1 · Demo data hides bugs.** End-to-end tests with the real engine, a fake provider and
Mailpit, and "every table that is read has production code writing it".
Spec F-5, F-4 · **Done in M40**
- Code: `internal/sending/sendtest` runs the real API, audience build and sending engine with a
  simulator transport on a virtual clock; `internal/store/writers_test.go` fails when a queried
  table is written only by the demo or tests. The health job still never calls a demo provider
  (they are sample data), so the outage flow is tested with a provider that isn't one. M40 found
  what the writers test can't see, data that only the demo seeded: the default reply rules (now
  given to every workspace) and providers' sending IPs, a column the new-IP canary, Send
  Readiness and warm-up read (now read from each provider's API or set by hand, R2.A1).
- Tests: `TestQueriedTablesHaveProductionWriters`, `TestGuardedSending/HealthJobOutageHasOptions`,
  `TestNewWorkspaceStartsWithDefaultReplyRules`, `TestGuardedSending/CanaryForReportedIP`.

**X.2 · The UI over-promises.** Every claim needs a test.
Spec F-6 · **Done in M40**
- Idempotency: R2.L2. "Events subscribed": R3.L3, R3.A1. "About 10 minutes": R3.U6. "So the
  winner is real": R5.L1, R5.U4. "No link domains on URI blocklists": R1.L1. Each item names the
  tests that pin its claim.

**X.3 · Mailbox provider by MX, shared by routing, throttling, the guard and reports.**
Spec F-1 · **Done** (R2.L8)

**X.4 · "No data" is not 0:** Gmail and iCloud complaints, SMTP deliveries, silent webhooks.
Spec F-3, F-2 · **Done**
- Code: the APIs return `null` with a reason for unreported rates; the screens say "n/a" and why.
  M40: a sending IP the blocklist checks haven't reached yet is Unknown on Send Readiness, never
  "clean on 0 of 0".
- Tests: `TestComplaintsCountWhereReported`, `TestGuardedSending/SMTPInfersDelivery`,
  `TestNoSilentAlarmAfterAQuietPeriod`, `TestProviderRatesNeedEnoughMail`,
  `TestAssembleUncheckedIP`.

**X.5 · Scope is spread thin:** pause SMS, WhatsApp, shop and the like until the five features'
P0 is done.
Process · **Followed**
- M33–M40 built only the five core features. Changes elsewhere are bugs found on the way (the
  review spec's last section) and the channel menu R3.U5 asked for, which SMS, Push and WhatsApp
  share.

## How it was checked

On 29 October 2026, at the head of `review/m40` (commit c70ac95; this section was added after it):

- **Generated code**: sqlc, the OpenAPI document and the web's API types regenerate with no change.
- **Migrations**: on a scratch database every migration runs up; the twelve M40 migrations then
  run down one by one, which removes their tables, columns and indexes, and up again, which gives
  a schema identical to the first.
- **Go**: `go vet` over all 195 packages finds nothing, and golangci-lint reports 0 issues (the
  packages the last commits changed were checked again at the head); `go test -race -count=1`
  over every package at that one commit, as CI runs it (`-timeout 45m`, four packages at a
  time): all 195 ran once, the 155 with tests passed and 40 have none, with no data race
  (14:16 to 15:55, in two parts: the 64 longest packages, then the other 131).
- **Faults the race run found**: the first run over every package failed in two packages, and
  the checks behind them found four faults, all fixed with tests before the run above:
  - Late in some months (29 to 31 October among them) the demo's Subscribe page launched inside
    Growth's cohort months, 150 days back, so the 148 joiners it had left for before them joined
    inside them and every cohort passed the boards' numbers. Every source but Beta apply now
    launches the day before the cohorts at the latest (`TestFormLaunchesAroundCohorts`, every
    five hours from 2026 to 2029; the demo seeded at 07:30 on 29 October has the boards'
    cohorts).
  - The demo's seeders picked people by joining contacts with a list's members before either
    table had planner statistics: a nested loop read 21,000 members for each of 48,000 contacts,
    a billion rows a pick, and eight picks took 12 of the seed's 13 minutes. The audience seeder
    now analyzes both once it has written them; the seed takes about a minute, and the run's
    demo-seeded tests with it.
  - The block and policy scenarios of guarded sending counted the reputation bounces and
    complaints the demo's simulated providers make at random: the burst failed 8 runs in 16, the
    floor 1 in 16. They now send through a provider that reports nothing of its own and pass ten
    runs in ten.
  - The reported-IP canary scenario saw its second send as over the usual size whenever the
    scenarios before it had left small sends, so it failed with the whole suite and passed alone.
    It now leaves that trigger out, as only the IP is under test; the whole suite passed three
    runs in three.
  The demo package's four tests now share one seed, made at noon in New York on the 15th and
  read as of then (checked on the 15th of ten months, through both changes of summer time),
  instead of seeding four times past go test's 45 minutes.
- **Web**: TypeScript and ESLint report nothing; `pnpm i18n:check` finds every one of the 36,290
  strings in Vietnamese and no hard-coded text; Vitest passes 995 tests in 141 files; the
  production build succeeds.
- **Screens**: the production build, with the demo data, signed in as an Owner, in English and
  Vietnamese at 390, 834 and 1440 px: 28 states of the screens M40 changed (Review & send with its
  technical details, fixes drawer, applied fixes and their Undo, and skipped checks; the live
  monitor with messages in doubt and a held share testing its provider; the report with its trend
  lines and the Investigate dialog's retry; the composer's style panel, synced block dialog,
  template gallery and the phone's approve sheet; providers with their sending IPs and their
  dialog, the channel menu, the route tester's campaign simulation, unsaved routing, warm-up,
  traffic rules and SMS providers; Deliverability; the bot filter and its dialog; the brand kit).
  168 runs: 158 checked, 10 left out by design (the builder's tabs on phones, which use the quick
  review instead, and the quick review's Approve on larger screens). axe (WCAG 2.0, 2.1 and 2.2
  A and AA) found no violation; no page scrolls sideways; no permission key appears in visible
  text; no console error. The controls under 44 px on phones are the sizes their boards draw
  (30 px chips and small buttons, 24 px text links, 20 px tips, 36 px dialog buttons, the 40×24
  switch). The live monitor states the demo data doesn't reach were drawn by the production build
  from the demo's own snapshot with the fields the engine sends for them. The run found seven
  faults, all fixed with tests: list ids in the route tester, web font imports in the gallery's
  pictures, a badge running past a 390 px screen, the quick review sheets' close button and the
  sending IPs dialog's buttons under the 44 px their boards draw, skipped checks not named on the
  approval card and the confirmation, faded counts below 4.5:1 in the live monitor's stale state,
  and an English error kind in a Vietnamese outage title.
- **Canvas**: every board was measured in Chromium once its fonts loaded and its runtime settled;
  65 frames were refitted to their content, boards under them moved down without overlaps, and
  the canvas was published (version 56).
- **Security**: every endpoint M40 added declares its permission (campaigns:edit for synced
  blocks, a retry and undoing fixes, with campaigns:send for undoing a route; templates:view for
  a template's picture; providers:manage for sending IPs; campaigns:send for messages in doubt;
  instance admins for the bot filter) and writes an audit entry when it changes something. A
  template's picture keeps nothing that runs, links, loads from elsewhere or takes focus, and is
  shown in a sandboxed, inert frame. Sending IPs set by hand must be public addresses; an agent's
  drafts go through the same checks as the composer's.
