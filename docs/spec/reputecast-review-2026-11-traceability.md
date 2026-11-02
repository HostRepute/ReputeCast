# Review 2026-11: every row, traced to code and tests

[`reputecast-review-2026-11.md`](reputecast-review-2026-11.md) filed the November 2026 detail review
as rows (DP-, EN-, AU-, AT-, CA-, TX-, AP-, IN-, OP-, FE-, EA-, CH-, X-), and milestones M41–M51 built
them. This page reads each row against the code at the end of M51 (1 November 2026): its status,
where it lives and the tests that pin it. It is the review's X-5.

Statuses:

- **Done (M4x)**: built as the row asks, in that milestone, pinned by the tests named.
- **Done in M50**: found while building M41–M49 and put on the backlog; M50 built it.
- **Done (M51)**: left open at the end of M50; M51, the review backlog, built it.
- **In part**: what was built, and the "b" row that holds the rest.
- **Open**: not built; why, and what it needs. These rows stay ⏳ in the review and move to the
  roadmap's next milestone.

Test names are Go test functions or web test files under `web/src`. Every state change these rows
added writes an audit entry; where the row's test checks the entry, it says so ("audited").

## Summary

| Area | Rows | Done | Done in M50 | Done in M51 | In part | Open |
|---|---|---|---|---|---|---|
| Consent and data protection (DP) | 9 | 7 | 2 | 0 | 0 | 0 |
| Engines and events (EN) | 10 | 7 | 1 | 2 (EN-8, EN-9) | 0 | 0 |
| Audience (AU) | 19 | 17 | 2 | 0 | 0 | 0 |
| Automations (AT) | 15 | 13 | 1 | 1 (AT-15) | 0 | 0 |
| Campaigns (CA) | 12 | 8 | 3 | 1 (CA-9) | 0 | 0 |
| Transactional (TX) | 7 | 6 | 0 | 1 (TX-7) | 0 | 0 |
| Approvals, polls (AP) | 3 | 2 | 0 | 1 (AP-3) | 0 | 0 |
| Integrations (IN) | 4 | 4 | 0 | 0 | 0 | 0 |
| Operations (OP) | 14 | 10 | 1 (OP-x) | 3 (OP-11, OP-12, OP-13) | 0 | 0 |
| Frontend (FE) | 8 + 5 | 5 | 0 | 8 (FE-3 with FE-3b, FE-5 with FE-5b, FE-7 with FE-7b, FE-9, FE-10) | 0 | 0 |
| Earn (EA) | 6 + 1 | 3 | 2 (EA-2b, EA-6) | 2 (EA-4, EA-5) | 0 | 0 |
| Other channels (CH) | 2 | 1 | 0 | 1 (CH-2) | 0 | 0 |
| Integration and audit (X) | 5 | 5 | — | — | — | — |

## Fix first

The nine bugs the review led with are all fixed:

| # | Bug | Rows | Status |
|---|---|---|---|
| 1 | Digest and monthly choices on the preference center were ignored | CA-6 | Done (M45) |
| 2 | One failing automation event or run stalled every workspace | EN-1 | Done (M42) |
| 3 | Date triggers east of UTC fired every minute; yearly ones only once | EN-2 | Done (M42) |
| 4 | Automation steps' delivered, bounced and complained stayed at 0 | EN-5 | Done (M42) |
| 5 | 12 webhook events were never sent | EN-6 | Done (M42) |
| 6 | Erasing left personal data in about 40 tables; the export stopped at 10,000 | DP-1, DP-2 | Done (M41) |
| 7 | Double opt-in confirmed on a GET; tokens never expired; no per-address throttle | DP-3, DP-4 | Done (M41) |
| 8 | Paused contacts got automation email; branch windows and the skip rule counted wrongly | DP-5, EN-3, EN-4 | Done (M41, M42) |
| 9 | Smaller bugs (Duplicate, Source filter, tag case, transactional search, approvals inbox, Stripe events) | EN-7 | Done (M42) |

## Consent and data protection (M41)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| DP-1 | Erasing a contact reaches every table | Done (M41) | `internal/erasure`, `internal/app/wire_erasure.go`, each package's `erasure.go` | `TestErasureCoversPersonalData`, `TestEraseLeavesNothingPersonal`, `TestEraseRefusesLiveStripePayers`, `TestErasureKeepsRevenueNumbers`, `TestContactErasedEvent`, `EraseDialog.test.tsx` (audited) |
| DP-2 | A complete, streamed data export | Done (M41) | `internal/audience/gdpr.go` | `TestContactExportIsComplete` (audited) |
| DP-3 | Scanner-safe double opt-in, expiring links, per-address cooldown | Done (M41) | `internal/publicpages/confirm.go`, `internal/audience/optin.go`, `internal/linktoken` | `TestConfirmationLinkLifetime`, `TestOptInCooldown`, `TestFloodedFormAsksForTurnstile`, `TestClientKeyGroupsIPv6`, `TestExpired` |
| DP-4 | Unconfirmed signups erased after a setting | Done (M41) | `internal/audience/retention.go`, `internal/workspaces/settings.go` | `TestRetentionErasesUnconfirmedSignups`, `TestRetentionSettings` (audited) |
| DP-5 | Paused contacts skip automation email | Done (M41) | `internal/automations/engine.go` | `TestEmailStepsSkipPausedContacts` |
| DP-6 | Open and click IP and user agent cleared after a setting | Done (M41) | `internal/audience/retention.go`; by partition since OP-8 | `TestRetentionClearsDetailKeepsReports`, `TestRetentionSettings`, `TestPartitionDeliveryEvents` |
| DP-7 | `/metrics` on its own port or behind a token; migration lock | Done (M41) | `internal/app/metrics.go`, `internal/db/db.go` | `TestMetricsDefaults`, `TestConcurrentMigratorsRunEachMigrationOnce` |
| DP-8 | Imports' kept files and reports leave with an erased contact | Done in M50 | `internal/imports/erasure.go`, `FinishImport` and `ImportFilesExpire` (`internal/store/queries/imports.sql`), maintenance; report step note | `TestEraseStripsImportFiles`, `TestCSVImport` |
| DP-9 | Paid members list shows "Erased contact" | Done in M50 | `internal/earn/members.go`, `web/src/features/earn/format.ts` (`memberName`) | `TestErasureKeepsRevenueNumbers` |

DP-8 as built: a done import drops its uploaded file; finished imports keep their report and
skipped-row files 30 days (the report step says until when; downloads answer 410 after), and an
erase strips the erased addresses from the reports still kept and removes an uploaded file that
names one, with a line in the import's log. Board: MigrationDone.

## Engines and events (M42)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| EN-1 | A failing event or run doesn't stall the rest | Done (M42) | `internal/automations/engine.go`, `jobs.go` | `TestFailingRunDoesNotStallOthers` |
| EN-2 | Date triggers dedupe on the local date | Done (M42) | `internal/store/queries/automations.sql` | `TestDateTriggersEastOfUTC` |
| EN-3 | Branch windows start when the email leaves | Done (M42) | `internal/automations/engine.go` | `TestBranchWindowStartsWhenTheEmailLeaves`, `TestClickConditionsCountPeopleOnly` |
| EN-4 | The skip rule counts marketing mail that was sent | Done (M42) | `internal/store/queries/automations.sql` | `TestSkipRuleIgnoresTransactionalMail` |
| EN-5 | Real providers' events reach automation steps | Done (M42) | `internal/sending/engine`, `internal/automations` | `TestProviderEventsCountOnAutomationSteps`, `TestAlertNotifiesOnce` |
| EN-6 | Every webhook event in the catalog is sent | Done (M42) | `internal/webhooks`, emitters | `TestEveryWebhookEventHasAnEmitter`, `TestEmailEventsReachWebhooks` |
| EN-7 | The smaller bugs | Done (M42) | as listed in the review | `CampaignsPage.test.tsx`, `ContactsScore.test.tsx`, `TestCompileQueryBindsValues`, `TestReviewAndApprovals`, `InboxPage.test.tsx`, `TestDelayedPaymentThatFails`, `TestDisputes`, `TestGiftPaidByBankDebit` |
| EN-8 | Batched `email.*` webhook deliveries | Done (M51) | `internal/webhooks/batch.go`, migrations 03387–03388, `WebhookDialog` | `TestEmailEventsInBatches` (250 events → 100/100/50, held while paused, audited), `WebhookDialog.test.tsx` |
| EN-9 | A gift paid by bank debit | Done (M51) | `internal/gifts/hooks.go`, `internal/gifts/deliver.go`, `internal/earn/memberemails.go` (gift-payment-failed), migration 03385 | `TestGiftPaidByBankDebit`, `TestGiftRemindersExpiryAndRefunds` |
| EN-10 | Products' publish checks name the payment-failed notice | Done in M50 | `internal/products/catalog.go` (`failed_notice`, advisory), `web/src/features/products/EditorAside.tsx` | `TestDelayedPaymentThatFails`, `products/checks.test.ts`. Boards: ProductEditor, ProductEditorStates |

## Audience at 1M contacts (M43)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| AU-1 | Keyset pages; counts stop at 10,000 | Done (M43) | `internal/audience/contacts.go`, `ContactsPage.tsx` | `TestContactsPageWithCursors`, `TestContactCountStopsAtTenThousand`, `TestAudienceAtOneMillion`, `ContactsPage.test.tsx` |
| AU-2 | Trigram search indexes | Done (M43) | migrations 0337x | `TestContactQueriesUseIndexes` |
| AU-3 | Segment filter as a subquery; cached view counts | Done (M43) | `internal/audience/filter.go` | `TestSegmentFilterIsASubquery`, `TestSavedViewsKeepCountsAndChange` |
| AU-4 | Background exports and bulk resend | Done (M43) | `internal/audience/bulk.go`, the jobs tray | `TestExportRunsInTheBackground`, `TestBulkResendReportsWhatItDid` (audited) |
| AU-5 | Select all matching | Done (M43) | `ContactsPage.tsx` | `ContactsPage.test.tsx` |
| AU-6 | Remove from list and tag | Done (M43) | `internal/audience/bulk.go` | `TestRemoveFromListAndTag` (audited) |
| AU-7 | Correct a contact's address | Done (M43) | `internal/audience/contacts.go` | `TestCorrectAnAddress` (audited) |
| AU-8 | Saved view menu | Done (M43) | `ViewChips.tsx` | `TestSavedViewsKeepCountsAndChange` |
| AU-9 | Column chooser | Done (M43) | `ContactColumns.tsx` | `TestContactsPageWithCursors`, `TestSavedViewsKeepCountsAndChange` |
| AU-10 | Joined between, one import's contacts | Done (M43) | `internal/audience/filter.go`, imports | `TestJoinedBetweenAndImportFilters`, `TestCSVImport` |
| AU-11 | Lists & fields; what uses a field or segment | Done (M43) | `internal/dependents`, `ListsFieldsPage.tsx` | `TestFieldsAndSegmentsShowWhatUsesThem`, `ListsFieldsPage.test.tsx` (audited) |
| AU-12 | Sunset policy with a dry run | Done (M43) | `internal/audience/tiers.go` | `TestSunsetErasingNeedsTheDryRun`, `SunsetPolicy.test.tsx` (audited) |
| AU-13 | Verify from a selection, segment or source | Done (M43) | `features/verification` | `TargetVerify.test.tsx` |
| AU-14 | Remove a suppression upload whole | Done (M43) | `internal/audience/suppressions.go` | `TestRemoveASuppressionUpload`, `RemoveUpload.test.tsx` (audited) |
| AU-15 | The team's opt-in email is the branded one | Done (M43) | `internal/audience/optin.go` | `TestTeamOptInEmailIsTheBrandedOne` |
| AU-16 | Static segments keep entry times | Done (M43) | `internal/segments` | `TestSegmentsKeepWhenMembersEnteredAndLeft` |
| AU-17 | Numbers at a million contacts | Done (M47) | `internal/loadtest` | `TestAudienceAtOneMillion` (6 ms, 7 ms, 23 ms) |
| AU-18 | What-uses-it reads every channel | Done in M50 | `internal/dependents` (transactional templates, texts, push, WhatsApp variables), `UsesList.tsx` | `TestFieldsAndSegmentsShowWhatUsesThem`, `UsesList.test.tsx` |
| AU-19 | A saved view has an address | Done in M50 | Contacts `?view=`, `UsesList.tsx`, List health | `ContactsPage.test.tsx`, `UsesList.test.tsx` |

AU-18 as built: a form's confirmation email is built from the form's branding and the contact's
first name, so it can't name a custom field and isn't read.

## Automations (M44)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| AT-1 | Row menu, delete with care, discard draft | Done (M44) | `features/automations` | `complete.test.tsx` (audited) |
| AT-2 | Contacts at a step; remove one | Done (M44) | `internal/automations/detail.go` | `TestContactsAtAStepAndRemovingOne`, `complete.test.tsx` (audited) |
| AT-3 | Preview and test send | Done (M44) | `internal/automations/testsend.go` | `TestPreviewAndTestSendAnAutomationEmail` |
| AT-4 | Resume with overdue contacts | Done (M44) | `internal/automations/publish.go` | `TestResumeWithOverdueContacts`, `complete.test.tsx` (audited) |
| AT-5 | Publish moves contacts quickly | Done (M44) | `internal/automations/publish.go` | `TestPublishMovesManyContactsQuickly` |
| AT-6 | Re-entry after N days | Done (M44) | engine | `TestMoreTriggersAndReentryAfterDays` |
| AT-7 | Date triggers at the contact's time | Done (M44) | engine | `TestDateTriggersAtTheContactsTime` |
| AT-8 | Leave the list, leave the flow | Done (M44) | migration 03379, engine | `TestLeavingTheListEndsTheRun` |
| AT-9 | Marketing cap | Done (M44) | sending rules, engine | `TestMarketingCap`, `complete.test.tsx` |
| AT-10 | Segment and link triggers | Done (M44) | `watchSegments`, link clicks | `TestSegmentAndLinkTriggers` |
| AT-11 | Up to three triggers | Done (M44) | `automations.triggers` (generated) | `TestMoreTriggersAndReentryAfterDays`, `complete.test.tsx` |
| AT-12 | Change a live flow's trigger by publishing | Done (M44) | publish diff | `complete.test.tsx` |
| AT-13 | Sequence stats | Done (M44) | `internal/automations/detail.go` | `TestSequenceFollowsTheEmails`, `complete.test.tsx` |
| AT-14 | Helpers read every trigger | Done in M50 | welcome prompt, score ingest, integrations hub, push audience, held-campaign action; scoring, products, store and date queries | `TestWelcomePromptReadsEveryTrigger`, `TestAutomationEmailListsReadEveryTrigger`, the scoring, commerce, products, push, integrations and automations suites |
| AT-15 | Back off slow watched segments | Done (M51) | `internal/automations/engine.go` (`WatchInterval`), migration 03386, the trigger panel's note | `TestSlowWatchedSegmentsBackOff`, `triggers.test.tsx` |

## Campaigns (M45)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| CA-1 | Delete, restore, archive | Done (M45) | `internal/campaigns/manage.go` | `TestDeleteArchiveRestore`, `CampaignsPage.test.tsx`, `manage.test.tsx` (audited) |
| CA-2 | Filters, tags, bulk | Done (M45) | `internal/campaigns/list.go`, `manage.go` | `TestBulkTagsAndFilters`, `manage.test.tsx` (audited) |
| CA-3 | Calendar and reschedule | Done (M45) | `CalendarPage.tsx`, `approvals/send.go` | `TestCalendar`, `TestReschedule`, `manage.test.tsx` (audited) |
| CA-4 | Follow-ups | Done (M45) | `internal/campaigns/followup.go` | `TestFollowUp`, `manage.test.tsx` |
| CA-5 | Send in each recipient's time zone | Done (M45) | readiness schedule, sending engine | `TestSending/LocalTime`, `TestEstimateLocalTime`, `TestReschedule` |
| CA-6 | Cadence | Done (M45) | sending audience | `TestSending/Cadence`, `TestAssembleCadenceLine` |
| CA-7 | A light list at scale | Done (M45) | `internal/campaigns/list.go` | `TestListTenThousandCampaigns`, `TestOverviewRecentIsOneLightPage` |
| CA-8 | Template gallery search and menu | Done (M45) | composer Templates tab | composer tests (visual QA) |
| CA-9 | Calendar moves texts, push and WhatsApp | Done (M51) | `internal/approvals/send.go` (`ChannelMove`), `internal/texting/move.go`, `internal/push/move.go`, `internal/whatsapp/move.go`, `CalendarPage` | `TestCalendarMovesTexts`, `TestCalendarMovesPush`, `TestCalendarMovesWhatsApp`, `manage.test.tsx` (audited) |
| CA-10 | Calendar knows an approval covers the time | Done in M50 | `CampaignCalendar` (`approved`), `ApprovedMoveDialog` | `TestReschedule`, `manage.test.tsx`. Board: CampaignsCalendar |
| CA-11 | Parameters Huma can't set fail CI | Done in M50 | `internal/api/params.go` (checked at registration) | `TestParametersReachTheHandler`, `TestRegisterOperationsWithoutConfig` |
| CA-12 | Tabs count within the filters | Done in M50 | `CountCampaignTabs` | `TestListCampaigns` |

## Transactional, approvals, polls, integrations (M46)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| TX-1 | Cancel or move a scheduled message | Done (M46) | `internal/transactional/schedule.go` | `TestCancelAndRescheduleScheduled`, `schedule.test.tsx` (audited) |
| TX-2 | Suppression actions from a message | Done (M46) | message page | `schedule.test.tsx` (audited) |
| TX-3 | Light live log | Done (M46) | `internal/transactional/messages.go` | `TestLogLivePoll` |
| TX-4 | Shared rate buckets | Done (M46) | token buckets in Postgres | `TestTakeTokensSharedAcrossNodes` |
| TX-5 | Send test from a template | Done (M46) | template editor | `TestSendTemplateTest` |
| TX-6 | SMTP relay limits | Done (M46) | `internal/smtprelay` | `TestGuardBlocksRepeatedFailedLogins`, `TestLimitListenerCapsConnections` |
| TX-7 | Cancel or move a batch by its key | Done (M51) | `internal/transactional/schedule.go` (`CancelBatch`, `RescheduleBatch`), the message page's "Which messages" | `TestBatchMovedAndCancelledTogether` (audited) |
| AP-1 | Approvals inbox re-checks only stale requests | Done (M46) | `internal/approvals/inbox.go` | `TestInboxSearchAndStaleChecks` |
| AP-2 | Survey results in SQL; load failures said | Done (M46) | `internal/polls` | polls suite (visual QA for the failure states) |
| AP-3 | Time the inbox at 1,000 requests | Done (M51) | `docs/deploy/performance.md` | `TestInboxAtAThousandRequests` |
| IN-1 | Old webhook secret signs a day | Done (M46) | `internal/webhooks` | `TestRotationKeepsTheOldSecretSigningADay` |
| IN-2 | Failing endpoint notification | Done (M46) | `internal/webhooks` | `TestRotationKeepsTheOldSecretSigningADay` |
| IN-3 | More MCP tools | Done (M46) | `internal/mcpserver` | `TestM46Tools` |
| IN-4 | API reference on the instance | Done (M46) | `internal/app/reference.go` | `TestAPIReference` |

## Operations (M47)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| OP-1 | Operational metrics | Done (M47) | `internal/opsmetrics` | `TestOperationalMetricsAreScraped` |
| OP-2 | `healthcheck`, `/readyz` | Done (M47) | `internal/cli/healthcheck.go`, `internal/app` | `TestHealthcheck`, `TestPendingMigrationsServeReadOnly` |
| OP-3 | Read-only while migrations are pending | Done (M47) | `internal/app/schema.go` | `TestPendingMigrationsServeReadOnly` |
| OP-4 | `doctor` and the restore runbook | Done (M47) | `internal/cli/doctor.go`, `docs/deploy/backup-restore.md` | `TestDoctor` |
| OP-5 | Reseal secrets | Done (M47) | `internal/sealed` | `TestEverySealedColumnIsKnown`, `TestResealMovesEveryValueToTheCurrentKey`, `TestPreviousKeysStillVerify` (audited) |
| OP-6 | Session rate limits | Done (M47) | `internal/server/sessionlimit.go` | `TestAPILimitsEachSession`, `TestRateLimitsBrowsersShareAHigherCap` |
| OP-7 | Request context in log lines | Done (M47) | `internal/logging` | `TestRequestLinesCarryTheRequest` |
| OP-8 | Partitioned delivery events | Done (M47) | `internal/db/partition.go` | `TestPartitionDeliveryEvents` |
| OP-9 | Load tests | Done (M47) | `internal/loadtest`, `loadtest/k6` | `TestAudienceAtOneMillion`, benchmarks |
| OP-10 | Instance admin › System | Done (M47) | `internal/instance/system.go` | `TestSystemTab`, `SystemPage.test.tsx` (audited) |
| OP-x | Rebill test at a month's start | Done in M50 | `internal/demo/seed_agency.go` | `TestUsageAndRebill` (failed on 1 November before the fix) |
| OP-11 | Convert `delivery_events` at upgrade | Done (M51, decision D9) | `internal/db/partition.go` (`upgradeEvents`, run by `MigrateUp` after the migrations: converts when not partitioned, drops `delivery_events_dedupe_key` concurrently), `internal/testdb/testdb.go` (test databases made the same way), `docs/deploy/partitioning.md` (rollback in full) | `TestMigrateUpConvertsEvents`, `TestMigrateUpAfterTheOptInCommand`, `TestPartitionDeliveryEvents` (from the documented rollback) |
| OP-12 | Count cache for segments and header counts | Done (M51) | `internal/countcache`, the audit log's observer, `audience.Summary`, `segments.Preview` | `TestCountsAreKeptUntilTheAudienceChanges`, `countcache_test.go` |
| OP-13 | Record k6 numbers | Done (M51) | `.github/workflows/load.yml` (a `load-test` label runs it; it fails on a crossed threshold), `loadtest/sendkey.sh`, `docs/deploy/performance.md`; the contacts list counts a first page that holds every match from its rows and reads a search's matches once (`internal/audience/contacts.go`); a send takes its rate tokens in one statement (`internal/transactional/ratelimit.go`) and wakes the outbox after it commits (`internal/sending/engine/transactional.go`); tier runs go in ordered batches and the classifier skips held contacts (`internal/audience/tiers.go`, `internal/mailbox/classifier.go`) | Load test runs 41353963642 to 41390919040 (the sixth meets every target); `TestTakeTokensConcurrently`, `TestClassifierSkipsLockedContacts`, `TestClassifierAlongsideTierRun`, `TestContactCountStopsAtTenThousand` |

## Frontend foundation (M48)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| FE-1 | Command palette | Done (M48) | `internal/search`, `CommandPalette.tsx` | `TestSearch`, `CommandPalette.test.tsx`, `Shortcuts.test.tsx` |
| FE-2 | Updated-version banner, chunk recovery | Done (M48) | `web/src/lib/systemStatus.ts` | `systemStatus.test.ts` |
| FE-3 | Shared parts | Done (M48; the rest in M51 as FE-3b) | `SwitchInput`, `lib/hooks`, `web/scripts/patterns-check.ts` | `pnpm lint` |
| FE-4 | Error boundaries per card and chart | Done (M48) | `ErrorBoundary.tsx` | `ErrorBoundary.test.tsx` |
| FE-5 | Colours from tokens | Done (M48; the rest in M51 as FE-5b) | `web/scripts/hex-baseline.json` (no increase allowed) | `pnpm lint` |
| FE-6 | Stack question recorded | Done (M48) | `docs/TECH_STACK.md` note 12 | — (owner's answer pending) |
| FE-7 | Optimistic updates | Done (M48; the rest in M51 as FE-7b) | notifications | `NotificationsPopover.test.tsx` |
| FE-8 | CI smoke, axe and bundle budget | Done (M48) | `web/e2e/smoke.spec.ts`, `web/scripts/bundle-budget.ts` | CI `smoke` job |
| FE-3b | Stepper, Pager, NoResults, one bulk pattern | Done (M51) | `components/ui/Stepper.tsx`: `Stepper` (six wizards, the import wizard included) and `StepTrail` (four dialogs); `components/ui/Pager.tsx` (eight lists); `lib/selection.ts` with `components/ui/SelectAll.tsx` (Campaigns, Grow › Review, Team); `NoResults` on fifteen page lists, `EmptyRow` in tables; `web/scripts/patterns-check.ts` | `Stepper.test.tsx`, `Pager.test.tsx`, `selection.test.tsx`, `eventLog.test.tsx`, `InboxPage.test.tsx`, `pnpm lint` |
| FE-5b | 1,166 hex colours to tokens | Done (M51) | Classes and styles (87 files), SVG and chart attributes (60), status palettes (45), ten status steps named in `web/src/styles/app.css`; `web/scripts/hex-baseline.json` | `pnpm lint` (1,166 → 323: by design, kept as text, logos and identity palettes, not colours) |
| FE-7b | Optimistic switches and reordering | Done (M51) | `useOptimistic` in `web/src/lib/hooks.ts`: sending rules' switches, recommendations' order | `optimistic.test.tsx` |
| FE-9 | Split the route tree | Done (M51) | `integrations/platforms.ts`, `pages/templateKeys.ts`, the lazy agency menu, `web/scripts/bundle-budget.ts` (400 KB) | `pnpm budget`: first load 390 KB gzipped (439 KB before) |
| FE-10 | Settings rows in the palette | Done (M51) | `web/src/features/settings/settingsIndex.ts`, `CommandPalette` | `CommandPalette.test.tsx` |

## Earn and other channels (M49)

| ID | What | Status | Code | Tests |
|---|---|---|---|---|
| EA-1 | Leave guards on Earn editors | Done (M49) | `TierEditor.tsx`, `OfferEditor.tsx`, `UpgradePageEditor.tsx` | `earnUi.test.tsx` (since M50) |
| EA-2 | Audited exports: gifts, conversions, payment events | Done (M49) | `internal/gifts/export.go`, `internal/revenue/export.go`, `internal/earn/eventexport.go` | `TestEarnExports`, `earnUi.test.tsx` (audited) |
| EA-2b | Sponsor bookings and reports as CSV | Done in M50 | `internal/sponsors/export.go` | `TestSponsorReportCountsOnlyBlockRecipients` (audited). Boards: EarnSponsors, EarnBooking |
| EA-3 | Table views and page sizes | Done (M49) | `RevenueOverview.tsx`, `MembersPage.tsx` | visual QA |
| EA-4 | Paid growth by tier and period | Done (M51) | `RevenueTierGrowth`, `RevenueOverview`'s "Joined · left" | `TestMRRMovementAndChurnFromTheLedger` (growth by tier and period), `earnUi.test.tsx` |
| EA-5 | Money paths against the Stripe fake | Done (M51) | `internal/earn/flow_test.go`, `internal/gifts/gifts_test.go` | `TestCancelNowOrAtThePeriodEnd`, `TestLegacyPriceStaysForItsMembers`, `TestForwardedGiftLinkGoesToWhoeverRedeems` |
| EA-6 | Earn UI tests | Done in M50 | — | `earnUi.test.tsx` |
| CH-1 | Channel campaigns deleted, archived, tagged | Done (M49) | unified Campaigns list | `TestChannelCampaignsDeleteArchiveAndTag` |
| CH-2 | Channel behaviour tests | Done (M51) | `internal/texting/behaviour_test.go`, `internal/push/push_test.go`, `internal/whatsapp/whatsapp_test.go`, `internal/app/channel_sweeps_test.go` | `TestSMSAutoPause`, `TestSMSRetentionSweep`, `TestPushRetriesABusyService`, `TestWhatsAppRetriesWhenMetaSlowsDown`, `TestChannelRetentionSweeps` |

## Integration and audit (M50)

| ID | What | Status | How |
|---|---|---|---|
| X-1 | No permission keys in copy | Done | The Earn exports' locked buttons said `subscribers:export`; they use `deniedReason` now (`earnUi.test.tsx` checks the copy). The other refusals already named the action and who can help. |
| X-2 | Every string in English and Vietnamese | Done | `pnpm i18n:check`: 100%, no hard-coded text, at every M50 commit. |
| X-3 | Roadmap, design inventory and README per milestone | Done | Each milestone updated `docs/ROADMAP.md` and `docs/DESIGN_INVENTORY.md`; M50's board changes are listed there. |
| X-4 | Every state change audited, with its test | Done | The tables above mark the rows whose tests check the audit entry. M50 added `sponsor.bookings_exported` and `sponsor.report_exported`; DP-8's stripping is part of `contacts.erase`. |
| X-5 | This page | Done | Written at the end of M50. |

## How this was checked

- Go: each touched package's suite against PostgreSQL 16, at every commit (`go test` with
  `REPUTECAST_TEST_DATABASE_URL`); CI runs the whole suite on 16, 17 and 18.
- Web: `pnpm typecheck`, `pnpm lint` (eslint, the patterns check), `pnpm i18n:check` and the vitest
  files named above.
- The tests named here were found by reading each row against the code, then searching the test
  files for the row's ID.
