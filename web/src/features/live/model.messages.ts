import { defineMessages } from '@/i18n'

/**
 * The live monitor's words and numbers (model.ts). Counts, percentages,
 * rates, times and money arrive formatted by lib/format; {who} is a mailbox
 * provider ("Outlook") or a sending provider ("Postal"), {providers} and
 * {names} a list of them, already joined ("SES and Postal").
 */
export const mm = defineMessages('live.model', {
  mailboxOther: 'Other',
  /** A rate of messages per second: {rate} is "14.2". */
  perSecond: '{rate} /s',
  seconds: '{s} s',
  minutesSeconds: '{m} min {s} s',
  minutes: '{m} min',
  noEventsYet: 'no events yet',
  secondsAgo: '{n} s ago',
  minutesAgo: '{n} min ago',
  minutesLong: '{n, plural, one {# minute} other {# minutes}}',

  // The header line under the title. {when} is a date and time ("Oct 10 16:20").
  startedBy: 'Started by {person}',
  scheduledBy: 'Scheduled by {person}',
  approvedBy: 'Approved by {approver}',
  approvedByAt: 'Approved by {approver}, {when}',
  startedApproved: 'Started by {person} · approved by {approver}',
  startedApprovedAt: 'Started by {person} · approved by {approver}, {when}',
  scheduledApproved: 'Scheduled by {person} · approved by {approver}',
  scheduledApprovedAt: 'Scheduled by {person} · approved by {approver}, {when}',

  // Who can help with a permission; {perm} is a permission key.
  needsPause: 'Only people who can pause sends can pause it.',
  needsResume: 'Only people who can send campaigns can resume it.',
  needsCancel: 'Only people who can send campaigns can cancel it.',
  needsDecide: 'Only people who can send campaigns decide how a held send continues.',
  needsRates: "Only people who can send campaigns change a send's rates.",
  askHelpers: 'Ask {helpers}.',

  // Progress.
  held: 'Held',
  sentOf: '{sent} of {total} sent',
  sentOfHeld: '{sent} of {total} sent · {held} held',
  sentOfHeldOn: '{sent} of {total} sent · {held} held on {provider}',
  timingStarted: 'Started {time}',
  timingHeldSince: '{who} held since {time}',
  timingTheRest: 'the rest: {routes}',
  timingEtaRest: 'ETA for the rest {eta}',
  timingProvidersFinish: '{providers} finishes about {time}',
  timingEta: 'ETA {eta}',
  timingFinishes: 'finishes about {time}',
  timingPausedSince: 'paused since {time}',
  /** {rates} compares the variants' rates ("15.1% vs 13.2%"); {sample} is a percentage. */
  abLine:
    '{phase, select, decided {A/B {kind, select, content {content} sender {sender} other {subject}} test: variant {variant} won, {rates} {metric, select, clicks {clicks} other {opens}} on the {sample} sample} other {A/B {kind, select, content {content} sender {sender} other {subject}} test: variant {variant} leading, {rates} {metric, select, clicks {clicks} other {opens}} on the {sample} sample}}',
  abVs: '{a} vs {b}',
  legendProvider: '{name} {sent} · {rate}',
  legendHeldOn: 'Held on {provider} {count}',
  legendHeldWho: 'Held, {who} {count}',
  legendHeld: 'Held {count}',
  legendQueuedOn: 'Queued on {provider} {count}',
  legendQueued: 'Queued {count}',
  /** Messages in doubt (review GS-1): {count} is formatted. */
  legendInDoubt: 'In doubt {count}',

  // Counters.
  tileDelivered: 'Delivered',
  tileBounced: 'Bounced',
  tileComplained: 'Complained',
  tileOpened: 'Opened',
  tileOpenedHuman: 'Opened · human',
  tileClicked: 'Clicked',
  tileCost: 'Cost',
  deliveredSub: '{rate} of sent',
  bouncedSub: '{rate} · {unknown} unknown user, {block} block/policy, {soft} soft',
  bouncedBlock: '{rate} · {block} block/policy',
  bouncedSuppressed: '{rate} · added to suppression',
  /** Complaint rates count only where mailbox providers report them (review F-2). */
  complainedSubReported: '{rate} of {base} delivered where complaints are reported',
  complainedFinal: '{rate} of {base} delivered where reported{counting, select, yes { · still counting} other {}}',
  complainedWhereReported: '{rate} where reported',
  complainedWhereReportedNa: '{rate} where reported · {names} n/a',
  complainedNa: '{names} n/a',
  complainedNone: "n/a · {names} don't report complaints",
  openedSub: '{rate} · {human} human, {machine} machine',
  openedMachine: '+{n} machine',
  clickedSub: '{rate} · {ctor} click-to-open',

  // Auto-pause thresholds (LiveMonitor's card, review GS-6).
  thresholdBounce: 'Bounces',
  thresholdBlock: 'Block/policy bounces',
  thresholdDeferral: 'Deferrals',
  thresholdComplaint: 'Complaints',
  thresholdMailbox: '{mailbox} complaints',
  thresholdReplies: 'Mailbox provider replies',
  thresholdErrors: 'Sending provider API errors',
  thresholdWholeSend: 'whole send',
  thresholdPerMailbox: 'per mailbox provider',
  thresholdPerMailboxWindow: 'per mailbox provider, over {window} min',
  thresholdWhereReported: 'whole send, where reported',
  thresholdHighest: 'highest mailbox provider',
  thresholdRepliesSub: '{count, plural, one {# reply rule} other {# reply rules}}, over 10 min',
  thresholdErrorsSub: 'over 3 min',
  thresholdBounceShort: 'Bounce',
  thresholdBlockShort: 'Block/policy',
  thresholdDeferralShort: 'Deferrals',
  thresholdComplaintShort: 'Complaint',
  thresholdMailboxShort: '{mailbox} complaint',
  thresholdRepliesShort: 'Replies',
  thresholdErrorsShort: 'Provider errors',
  /** A rate against its limit: "0.60% of 2.0%". */
  ofLimit: '{value} of {limit}',
  /** A mailbox provider's rate against its limit: "Yahoo 3.1% of 15%". */
  ofLimitWho: '{who} {value} of {limit}',
  /** A reply rule's count: "Yahoo TSS04: 1 of 10 replies". */
  thresholdRepliesValue: '{who} {code}: {count} of {limit} replies',
  notAvailable: 'n/a',
  // The complaint watch after the send (LiveMonitorStates: complete; review GS-6).
  watchTitle:
    "Complaint watch · {hours} h · {complaints, plural, one {# complaint} other {# complaints}} so far{alert, select, yes { · you'll be alerted if {who} passes {limit}} other {}}",
  aMailboxProvider: 'a mailbox provider',
  /** Under the mailbox table: {names} is "Gmail and iCloud". */
  naComplaints: "n/a: {names} don't report complaints per message.",
  postmasterRate: "Gmail's spam rate from Postmaster Tools: {rate} on {day}.",
  mxNote: 'Gmail includes Google Workspace and Outlook includes Microsoft 365 recipients (found from MX records).',
  thresholdState: '{state, select, ok {Watching} near {Close to the limit} collecting {Collecting evidence} tripped {Tripped} other {n/a}}',
  thresholdNa: "{reason, select, not_reported {Gmail and iCloud don't report complaints per message} other {Nothing to count yet}}",
  /**
   * {sooner} is yes for complaints and block/policy bounces, which also
   * pause from 3 once clearly over the limit (review R2.L7c).
   */
  thresholdNeeds:
    '{events} in {base} {unit, select, delivered {delivered} other {sent}} · needs {minEvents} {key, select, complaint {complaints} mailbox_complaint {complaints} deferral {deferrals} block_bounce {block/policy bounces} other {bounces}} and {minBase} {unit, select, delivered {delivered} other {sent}} to pause{sooner, select, yes {, or 3 clearly over the limit} other {}}',
  thresholdComplaintsTo: '{events} in {base} delivered to {names}',
  thresholdIn: '{unit, select, delivered {{events} in {base} delivered} other {{events} of {base} sent}}',
  thresholdHighestAction:
    '{action, select, slow {highest · at {limit} it slows to half rate} pause_send {highest · at {limit} it pauses the send} warn {highest · at {limit} it warns} other {highest · pauses only that mailbox provider}}',
  thresholdReplyAction:
    '{action, select, slow {at {limit}, {who} slows to half rate} pause_route {at {limit}, {who} pauses on that route} alert {at {limit}, an alert goes out} other {at {limit}, {who} pauses}}{timed, select, yes { for {duration}} other {}}',
  thresholdsFootnote:
    'A threshold trips after at least 5 events and enough evidence that the rate is really above the limit; complaints and block/policy bounces also trip from 3 when the rate is clearly over it, however small the send. Until then it shows “Collecting evidence” with a grey bar.',
  scopeCampaign: 'Scope: pause the whole campaign.',
  scopeRoute: 'Scope: pause only the affected route or mailbox provider.',

  // Mailbox and sending providers.
  mailboxPausedAt: 'Paused {time}',
  mailboxPaused: 'Paused',
  mailboxSending: 'Sending',
  mailboxDone: 'Done',
  mailboxOverLimit: 'Over limit',
  mailboxNearLimit: 'Near limit',
  mailboxOk: 'OK',
  rateHeld: 'held',
  // The header (review GS-14): freshness and who can act, in people's words.
  freshLive: 'Live · updated {ago}',
  freshPolling: 'Checking every 10 s · updated {ago}',
  freshReconnecting: 'Reconnecting · updated {ago}',
  whoPause: 'Anyone who can send can pause',
  whoResume: '{names} can resume',
  whoCanaryStop: 'Anyone who can send can stop it',
  whoCanaryContinue: '{names} can continue early',
  routeDoneFirst: '{provider} done about {time}',
  routeDoneNext: '{provider} about {time}',
  timingRoutes: 'Started {time} · at current rates, {routes}',
  deliveredRetrying: '{rate} of sent{retrying, plural, =0 {} other { · # deferred, retrying}}',
  staleTitle: '{seconds, plural, other {No update for # s}}',
  staleTitleMinutes: '{minutes, plural, one {No update for # minute} other {No update for # minutes}}',
  staleBody:
    'A proxy may be buffering the live connection, so ReputeCast switched to checking every 10 s. Sending continues on the server; the counts below are from {time}.',
  detailCanary: '{seeds, plural, =0 {{size} most engaged · {state}} other {{size} most engaged plus {seeds} seed mailboxes · {state}}}',
  canaryState: '{phase, select, passed {passed {time}} continued {continued {time}} stopped {stopped {time}} held {held} sending {sending} other {watching until {time}}}',
  detailAfter: 'complaint watch for {hours, select, 168 {7 days} other {{hours} h}}',
  colDeferred: 'Deferred · {minutes} min',
  colDoneAbout: 'Done about',
  eventsCaption: 'newest first · from {via}',
  eventsCaptionPlain: 'newest first',
  pauseUpdates: 'Pause updates',
  resumeUpdates: 'Resume updates',
  feedPaused: 'paused',
  phoneProvider: '{rate} · {delivered} delivered{done, select, none {} other { · done about {done}}}',
  phoneMailbox: '{delivered} delivered · deferred {deferred} · bounce {bounce} · complaints {complaints} · {rate}',
  sendActions: 'Send actions',
  moreActions: 'More actions',
  mobileDelivered: '{rate}{retrying, plural, =0 {} other { · # retrying}}',
  noticeTitle: 'Something needs your attention',
  providerNotes: 'Complaint rates count only mail delivered to mailbox providers that report complaints, so Gmail and iCloud are left out. Done about: at current rates, including the per-mailbox limits.',
  deferredNote: 'Deferred is the share of delivery attempts in the last {minutes} minutes that the mailbox provider asked to retry.',
  // The progress card's summary line: the only live region of the live
  // view; it changes when the send's or a threshold's state does, never on
  // every count (LiveMonitor).
  sumSending: 'Sending to everyone; every threshold is inside its limit.',
  sumCollecting: 'Sending to everyone; every threshold with enough evidence is inside its limit.',
  sumNear: '{count, plural, one {Sending to everyone; {names} is close to its limit.} other {Sending to everyone; {names} are close to their limits.}}',
  sumMailboxPaused:
    '{count, plural, =0 {{who} paused at {time} and waits for your decision.} one {{who} paused at {time} and waits for your decision. {names} is still sending.} other {{who} paused at {time} and waits for your decision. {names} are still sending.}}',
  sumRouteHeld: '{count, plural, =0 {{who} share held since {time} and waiting for your decision.} other {{who} share held since {time} and waiting for your decision. {names} keeps sending.}}',
  sumCampaignPaused: 'Every mailbox provider paused at {time} and waits for your decision.',
  sumPaused: 'Paused at {time}. Nothing goes out until someone resumes.',
  /** Everything else went out; only messages in doubt are left (review GS-1). */
  sumInDoubtWaiting: 'Everything else is sent. {count, plural, one {# message in doubt waits} other {# messages in doubt wait}} for your decision.',
  sumPreparing: 'Preparing the audience; sending starts in a moment.',
  sumCanaryLead: '{phase, select, continued {Canary continued at {time}.} other {Canary passed at {time}.}}',
  rateFull: 'full rate',
  providerDownSince: 'Down since {time}',
  providerHeld: 'Held',
  /** A retried share's test messages: {done} of {needed} went through in a row (review GS-3). */
  providerRetrying: 'Retrying · {done} of {needed} tests passed',
  /** A held share testing its provider while nobody has decided yet (review GS-3). */
  providerProbing: 'Testing · {done} of {needed} tests passed',
  providerWarmup: 'Warm-up',
  providerWarmupNote: "warm-up day {day} · {sent} of today's {cap} cap",
  // A share today's warm-up cap (or a paused plan) holds; it goes on by
  // itself with tomorrow's cap (review PR-8).
  providerWarmupHeld: "Waits for tomorrow's cap",
  providerWarmupPaused: 'Waits for the warm-up plan',
  /** {who} is the sending provider ("Postal"); {time} when tomorrow's cap starts ("00:00"). */
  sumWarmupHeld:
    "{paused, select, yes {{who} share waits for its warm-up plan to resume.} other {{who} share waits for tomorrow's warm-up cap from {time}.}}{count, plural, =0 {} one { {names} keeps sending.} other { {names} keep sending.}}",
  badgeWarmupHeld: '{provider} route waits for the warm-up cap',
  /** A time on the next day: {time} is "00:12". */
  tomorrowAt: 'tomorrow {time}',
  needsWarmupCancel: 'Only people who can pause sends can cancel a share the warm-up cap holds.',
  providerQuotaNote: 'quota {free} free after this send',
  providerLine: '{rate} · {delivered} delivered',

  // An incident. {metric} is complaint, bounce, deferral, block_bounce
  // (block/policy bounces) or unknown_user; {Metric} the metric's own
  // name, capitalised, when it's another one.
  incidentTitle:
    '{metric, select, complaint {{who} complaint rate reached {value} (limit {limit})} bounce {{who} bounce rate reached {value} (limit {limit})} deferral {{who} deferral rate reached {value} (limit {limit})} block_bounce {{who} block/policy bounce rate reached {value} (limit {limit})} unknown_user {{who} unknown-user rate reached {value} (limit {limit})} other {{who} {metric} rate reached {value} (limit {limit})}}',
  incidentTitleAll:
    '{metric, select, complaint {Complaint rate reached {value} (limit {limit})} bounce {Bounce rate reached {value} (limit {limit})} deferral {Deferral rate reached {value} (limit {limit})} block_bounce {Block/policy bounce rate reached {value} (limit {limit})} unknown_user {Unknown-user rate reached {value} (limit {limit})} other {{Metric} rate reached {value} (limit {limit})}}',
  outageAuth: '{name} rejected the credentials',
  outageReturned: '{name} returned {errorKind}',
  outageDown: '{name} is down',
  /** Asked again: a retried share still failed its test messages for 15 minutes. */
  outageStillDown: '{name} is still down after 15 minutes of retrying',
  /** {errorKind} comes from the provider's adapter ("connection errors"). */
  outageReturnedFor: '{name} returned {errorKind} for {mins, plural, one {# minute} other {# minutes}}',
  outageErrorsFor: '{name} returned errors for {mins, plural, one {# minute} other {# minutes}}',
  /** The engine's error kinds (sending/engine/analysis.go), as {errorKind} above says them. */
  kindConnection: 'connection errors',
  kindThrottling: 'throttling errors',
  kindApi: 'API errors',
  // Route-level answers that hold a share (review GS-4, PR-7).
  outageIdentity: "{name} refused the sender: it isn't verified there",
  outageQuota: "{name}'s sending quota is used up",
  outageThrottled: '{name} kept throttling the account for {mins, plural, one {# minute} other {# minutes}}',
  /** {rate} is a percentage ("12.5%"). */
  outageApiErrors: '{name} failed {rate} of its calls over 3 minutes',

  summaryTrippedWho: 'Tripped at {time}, after {sent} of {total} {who} recipients were sent.',
  summaryTripped: 'Tripped at {time}, after {sent} of {total} recipients were sent.',
  /**
   * {count} is {events} formatted, for a metric other than these; {window}
   * the minutes a burst counted (0: the whole send).
   */
  summaryEvents:
    '{metric, select, complaint {{events, plural, one {# complaint} other {# complaints}} out of {base} delivered} bounce {{events, plural, one {# bounce} other {# bounces}} out of {base} sent} deferral {{events, plural, one {# deferral} other {# deferrals}} out of {base} sent} block_bounce {{events, plural, one {# block/policy bounce} other {# block/policy bounces}} out of {base} sent} unknown_user {{events, plural, one {# unknown user} other {# unknown users}} out of {base} sent} other {{count} {metric} out of {base} sent}}{window, plural, =0 {} other { within # minutes}}.',
  summaryCampaignPaused: 'ReputeCast paused the campaign with {held} recipients left, {progress} complete.',
  summaryHeldWho: 'ReputeCast held the remaining {held} {who} recipients.',
  summaryHeld: 'ReputeCast held the remaining {held} recipients.',
  summaryContinue: '{count, plural, one {{names} continues, now {progress} complete overall.} other {{names} continue, now {progress} complete overall.}}',
  summaryNow: 'Now {progress} complete overall.',

  outageWindow:
    'From {from} to {to}, {attempts, plural, one {# send attempt} other {# send attempts}} to {host} failed and {succeeded} succeeded, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.',
  outageAuthSince:
    'Since {from}, {provider} refused the credentials on every send, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.',
  outageAuthAlways: '{provider} refused the credentials on every send, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.',
  outageHealth: "ReputeCast's health checks found {provider} down at {time}, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.",
  outageIdentityHold:
    '{provider} refused the sender of this campaign, so its {short} share is on hold: {sent} sent, {remaining} remaining. Nothing was marked failed.',
  outageQuotaHold:
    "{provider} answered that the account's sending quota is used up, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining. Nothing was marked failed.",
  outageThrottledHold:
    '{provider} kept throttling the account for {mins, plural, one {# minute} other {# minutes}}, even at a lower rate, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.',
  /** {rate} and {limit} are percentages. */
  outageApiErrorsHold:
    '{rate} of the calls to {provider} failed over 3 minutes, over the {limit} limit, so the {short} share of this campaign is on hold: {sent} sent, {remaining} remaining.',
  outageKeepSending:
    '{count, plural, one {{names} keeps sending at {rate}, {progress} complete overall.} other {{names} keep sending at {rate}, {progress} complete overall.}}',
  outageNoOther: 'No other provider is sending, {progress} complete overall.',
  policyPauseAsk: 'Nothing moved automatically because Broadcast failover is set to pause and ask.',
  policyWarmOnly: 'Nothing moved automatically because no warmed-up provider can take the share.',
  policyNone: 'Nothing moved automatically because no other provider can take the share.',
  /**
   * Under an outage's log (review R2.L2): {provider} is the sending provider
   * ("SparkPost"); it recognizes a message it already took (idempotency key)
   * for {window} ("24 h") after the first try.
   */
  retriesKeyed:
    'Retries used exponential back-off (5 s to 60 s). Each attempt carries an idempotency key, and {provider} recognizes a message it already took for {window} after the first try, so a retry in that time is never delivered twice. After it, a message whose answer was lost is held as in doubt.',
  /** {provider} is the sending provider ("Postal"); it can't recognize a message sent again. */
  retriesUnkeyed:
    "Retries used exponential back-off (5 s to 60 s). {provider} can't recognize a message it already took, so a message goes again only after {provider} refused it; one whose answer was lost is held as in doubt, not sent twice.",
  /**
   * While nobody has decided (review GS-3): {short} is the sending provider
   * ("Postal"); {needed} test messages in a row resume the share; {done}
   * went through so far.
   */
  outageProbing:
    'Until you decide, ReputeCast tests {short} with {seed, select, yes {1 message to a seed mailbox} other {1 of these messages}} every 60 s and sends the share again after {needed} go through in a row{done, plural, =0 {} other { ({done} so far)}}.',

  // "What tripped".
  factWho:
    '{metric, select, complaint {{who} complaints} bounce {{who} bounces} deferral {{who} deferrals} block_bounce {{who} block/policy bounces} unknown_user {{who} unknown users} other {{who} {metric}}}',
  factAll: '{metric, select, complaint {Complaints} bounce {Bounces} deferral {Deferrals} block_bounce {Block/policy bounces} unknown_user {Unknown users} other {{Metric}}}',
  factValue: '{events} of {base} · {rate}',
  factLimit: '{scope, select, mailbox_provider {Limit, per mailbox provider} route {Limit, per sending provider} other {Limit, whole campaign}}',
  factOthers: '{names} complaints',
  factEvidence:
    '{metric, select, complaint {{events} complaints} bounce {{events} bounces} deferral {{events} deferrals} block_bounce {{events} block/policy bounces} unknown_user {{events} unknown users} other {{events} events}} in {base} {unit, select, delivered {delivered} other {sent}} · enough evidence (needs {minEvents} events and {minBase} {unit, select, delivered {delivered} other {sent}})',
  /**
   * Clearly over the limit before the usual evidence (review R2.L7c): on a
   * small send ({window} 0; {waitFor} is the part of the evidence it didn't
   * wait for: events, base or both), or in a burst of block/policy bounces
   * within {window} minutes.
   */
  factEvidenceClear:
    "{metric, select, complaint {{events} complaints} block_bounce {{events} block/policy bounces} other {{events} events}} in {base} {unit, select, delivered {delivered} other {sent}}{window, plural, =0 { · clearly over the limit, so it didn't wait for {waitFor, select, events {{minEvents} events} base {{minBase} {unit, select, delivered {delivered} other {sent}}} other {{minEvents} events and {minBase} {unit, select, delivered {delivered} other {sent}}}}} other { within # minutes · clearly over the limit in that time}}",
  factWatch: 'Complaints keep arriving for up to {hours} h; ReputeCast keeps watching after the send ends.',
  factCampaignWide:
    '{metric, select, complaint {Campaign-wide complaints} bounce {Campaign-wide bounces} deferral {Campaign-wide deferrals} block_bounce {Campaign-wide block/policy bounces} unknown_user {Campaign-wide unknown users} other {Campaign-wide {metric}}}',

  // "If you move the share to SES".
  moveTitle: 'If you move the share to {provider}',
  moveCost: 'Extra cost',
  moveTimeAt: 'Extra time at {rate}',
  moveTime: 'Extra time',
  moveQuota: '{provider} quota after',
  moveQuotaValue: '{free} free of {max}',

  // Header badges.
  badgeAutoPausedWho: 'Auto-paused · {who}',
  badgeAutoPaused: 'Auto-paused',
  badgeOthersSending: 'Other mailbox providers still sending',
  badgeStillSending: '{providers} still sending',
  badgeRouteRetrying: '{provider} route held · retrying every 60 s',
  badgeRouteWaiting: '{provider} route held · waiting for your decision',
  badgeRouteProbing: '{provider} route held · testing every 60 s until you decide',
  /** {count} is formatted. */
  badgeInDoubt: '{count} in doubt',
  badgeInDoubtWaiting: 'Waiting for your decision · {count} in doubt',
  badgeProviderHeld: '{provider} held',
  badgeStarting: 'Starting',
  badgeSending: 'Sending',
  badgePausedBy: 'Paused by {name}',
  badgePaused: 'Paused',
  badgeScheduled: 'Scheduled',
  badgeSent: 'Sent',
  badgeCancelledPartial: 'Cancelled · partially sent',
  badgeCancelled: 'Cancelled',
  badgeFailed: 'Failed',
  badgeDraft: 'Draft',

  // The states.
  startingTitle: 'Starting send · {time}',
  startingTitleNoTime: 'Starting send',
  startingCanaryTitle: 'Starting the canary · {time}',
  startingCanaryTitleNoTime: 'Starting the canary',
  startingCanary:
    '{seeds, plural, =0 {{size} most-engaged recipients go first; the other {queued} wait until the canary passes.} other {{size} most-engaged recipients and {seeds} seed mailboxes go first; the other {queued} wait until the canary passes.}} Delivery events usually arrive within 10 seconds.',
  startingBuilding: 'Building the queue for {count} recipients: suppressed, unsubscribed and paused contacts are left out.',
  startingBuildingNoCount: 'Building the queue: suppressed, unsubscribed and paused contacts are left out.',
  startingFirstBatches: "The first batches go to {providers} as soon as it's ready.",
  startingBuilt: 'Queue built for {count} recipients. Handing the first batches to {providers}; delivery events usually arrive within 10 seconds.',
  startingBuiltYours: 'Queue built for {count} recipients. Handing the first batches to your providers; delivery events usually arrive within 10 seconds.',
  finishedAt: '{name} finished at {time}',
  finished: '{name} finished',
  finishedIn: '{sent} sent in {took}: {delivered} delivered ({rate}), {bounced} bounced, {complaints, plural, one {# complaint} other {# complaints}}{watching, select, yes { so far} other {}}.',
  finishedNoTime: '{sent} sent: {delivered} delivered ({rate}), {bounced} bounced, {complaints, plural, one {# complaint} other {# complaints}}{watching, select, yes { so far} other {}}.',
  finishedTrips: '{n, plural, =0 {No threshold tripped.} one {One threshold paused part of the send.} other {# thresholds paused parts of the send.}}',
  finishedOutages: '{n, plural, one {A provider outage held part of it.} other {# provider outages held part of it.}}',
  finishedReport: 'Opens and clicks keep updating in the report for 72 hours.',
  cancelledAt: '{name} was cancelled at {time}',
  cancelled: '{name} was cancelled',
  cancelledNothing: 'Nothing was sent. The campaign can be scheduled again from Review & send.',
  cancelledSome: '{sent} of {total} were sent before it stopped; {unsent} weren\'t sent and are marked "not sent" in the report. Opens and clicks keep updating for 72 hours.',
  pausedByAt: 'Paused by {name} at {time}',
  pausedBy: 'Paused by {name}',
  pausedAt: 'Paused at {time}',
  paused: 'Paused',
  /** {reason} is the pause's reason, as the person wrote it. */
  pausedBodyReason:
    '"{reason}" {count} recipients are waiting. Nothing more is sent until {who} resumes it; what was already sent keeps reporting opens and clicks.',
  pausedBody: '{count} recipients are waiting. Nothing more is sent until {who} resumes it; what was already sent keeps reporting opens and clicks.',
  /** {who} in pausedBody when no one is named. */
  pausedWhoAnyone: 'someone who can send',
  lostTitle: 'Lost the live event stream at {time}',
  lostBody:
    "The browser's connection to ReputeCast dropped. Sending continues on the server; the counts below are from {asOf} and may be up to {behind, plural, one {# minute} other {# minutes}} behind.",
  lostStopped: 'Stopped reconnecting after {max} attempts.',
  lostRetrying: 'Reconnecting automatically (attempt {attempt} of {max}).',
  mobileEta: 'ETA {eta} · about {time}',
  mobileEtaPaused: 'Paused',
  mobileEtaInDoubt: 'Waiting for your decision',

  // Messages in doubt (review GS-1): a provider call ended without an
  // answer, so ReputeCast can't tell whether the provider took them.
  inDoubtTitle: '{count, plural, one {# message in doubt} other {# messages in doubt}}',
  inDoubtWaitingTitle: '{count, plural, one {# message in doubt waits for your decision} other {# messages in doubt wait for your decision}}',
  /** {providers} lists the sending providers ("Postal and SparkPost"). */
  inDoubtBody:
    "ReputeCast can't tell whether {providers} took {count, plural, one {it} other {them}}: the call that handed {count, plural, one {it} other {them}} over ended without an answer. {count, plural, one {It isn't} other {They aren't}} sent again unless someone decides to, so nobody gets the email twice.",
  inDoubtWaitingRest: 'Everything else is sent; the send ends once you decide.',
  inDoubtSendingRest: 'The rest of the send goes on meanwhile; decide whenever you are ready.',
  inDoubtWho: 'Where they are in doubt',
  /** {provider} is the sending provider ("Postal"); {time} when the call went out. */
  inDoubtShare: '{provider} · {count, plural, one {# message} other {# messages}} · handed over at {time}',
  /** {window} is how long the provider recognizes a message it took ("24 h"); {until} a time ("tomorrow 09:09"). */
  inDoubtSafe: '{provider} recognizes a message it already took for {window} after the first try, so sending them again before {until} delivers none twice.',
  inDoubtExpired:
    '{provider} recognizes a message it already took for {window} after the first try. For these that time is up, so sending them again can deliver a copy twice.',
  inDoubtNoKey: "{provider} can't recognize a message it already took, so sending them again can deliver a copy twice.",
  inDoubtSafeBadge: 'Safe to send again',
  inDoubtTwiceBadge: 'Could arrive twice',
  inDoubtUnknownProvider: 'The provider',
  inDoubtSettles: 'If the provider reports an event for one (delivered, bounced or a complaint), it counts as sent and leaves this list by itself.',

  complaintHeading: 'Complaint',
  complaintHeadingLimit: 'Complaint · limit {limit}',
  failoverDone: '{streams} already failed over to {providers} on their own',
  whoDelivers: 'who delivers the mail',
  failoverUnused: "Automations and System don't use {providers}",
  failoverShared: '{count, plural, one {{streams} also uses {providers}} other {{streams} also use {providers}}}',
  /** {items} lists each mailbox provider with its share ("Gmail 46%, Outlook 21%"). */
  mixLabel: 'Delivered share: {items}',
  scheduledFor: 'Scheduled for {date} · {time}',
  scheduled: 'Scheduled',

  // A delivery event's word in the feed (the server's English as fallback).
  evDelivered: 'delivered',
  evHardBounce: 'hard bounce',
  evSoftBounce: 'soft bounce',
  evDeferred: 'deferred',
  evComplaint: 'complaint',
  evRejected: 'rejected',
  evOpenedMachine: 'opened · machine',
  evOpenedHuman: 'opened · human',
  evClickedScanner: 'clicked · scanner',
  evClicked: 'clicked',
  evUnsubscribed: 'unsubscribed',
  evBounce: 'bounce',
  evOpen: 'open',
  evScanner: 'scanner',
  evClick: 'click',
})
