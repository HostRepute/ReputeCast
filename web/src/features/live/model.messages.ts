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
  needsPause: 'Pausing a send needs the {perm} permission.',
  needsResume: 'Resuming a send needs the {perm} permission.',
  needsCancel: 'Cancelling a send needs the {perm} permission.',
  needsDecide: 'Deciding how a held send continues needs the {perm} permission.',
  askHelpers: 'Ask {helpers}.',

  // Progress.
  held: 'Held',
  sentOf: '{sent} of {total} sent',
  sentOfHeld: '{sent} of {total} sent · {held} held',
  sentOfHeldOn: '{sent} of {total} sent · {held} held on {provider}',
  timingStarted: 'Started {time}',
  timingPausedRoute: 'paused route since {time}',
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

  // Counters.
  tileDelivered: 'Delivered',
  tileBounced: 'Bounced',
  tileComplained: 'Complained',
  tileOpened: 'Opened',
  tileOpenedHuman: 'Opened · human',
  tileClicked: 'Clicked',
  tileCost: 'Cost',
  deliveredSub: '{rate} of sent',
  bouncedSub: '{rate} · {hard} hard, {soft} soft',
  bouncedSuppressed: '{rate} · added to suppression',
  complainedSub: '{rate} of delivered',
  complainedOverall: '{rate} overall · {mailbox} {mailboxRate}',
  openedSub: '{rate} · {human} human, {machine} machine',
  openedMachine: '+{n} machine',
  clickedSub: '{rate} · {ctor} click-to-open',

  // Auto-pause thresholds.
  thresholdBounce: 'Bounce rate',
  thresholdComplaint: 'Complaint rate',
  thresholdMailbox: '{mailbox} complaints (highest mailbox provider)',
  thresholdErrors: 'Provider API errors, 3 min',
  thresholdBounceShort: 'Bounce',
  thresholdComplaintShort: 'Complaint',
  thresholdMailboxShort: '{mailbox} complaint',
  thresholdErrorsShort: 'Provider errors',
  /** A rate against its limit: "0.60% of 2.0%". */
  ofLimit: '{value} of {limit}',
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
  rateFull: 'full',
  providerDownSince: 'Down since {time}',
  providerHeld: 'Held',
  providerWarmup: 'Warm-up',
  providerWarmupNote: "warm-up day {day} · {sent} of today's {cap} cap",
  providerQuotaNote: 'quota {free} free after this send',
  providerLine: '{rate} · {delivered} delivered',

  // An incident. {metric} is complaint, bounce or deferral; {Metric} the
  // metric's own name, capitalised, when it's another one.
  incidentTitle:
    '{metric, select, complaint {{who} complaint rate reached {value} (limit {limit})} bounce {{who} bounce rate reached {value} (limit {limit})} deferral {{who} deferral rate reached {value} (limit {limit})} other {{who} {metric} rate reached {value} (limit {limit})}}',
  incidentTitleAll:
    '{metric, select, complaint {Complaint rate reached {value} (limit {limit})} bounce {Bounce rate reached {value} (limit {limit})} deferral {Deferral rate reached {value} (limit {limit})} other {{Metric} rate reached {value} (limit {limit})}}',
  outageAuth: '{name} rejected the credentials',
  outageReturned: '{name} returned {errorKind}',
  outageDown: '{name} is down',
  /** {errorKind} comes from the provider's adapter ("connection errors"). */
  outageReturnedFor: '{name} returned {errorKind} for {mins, plural, one {# minute} other {# minutes}}',
  outageErrorsFor: '{name} returned errors for {mins, plural, one {# minute} other {# minutes}}',
  // Route-level answers that hold a share (review GS-4, PR-7).
  outageIdentity: "{name} refused the sender: it isn't verified there",
  outageQuota: "{name}'s sending quota is used up",
  outageThrottled: '{name} kept throttling the account for {mins, plural, one {# minute} other {# minutes}}',
  /** {rate} is a percentage ("12.5%"). */
  outageApiErrors: '{name} failed {rate} of its calls over 3 minutes',

  summaryTrippedWho: 'Tripped at {time}, after {sent} of {total} {who} recipients were sent.',
  summaryTripped: 'Tripped at {time}, after {sent} of {total} recipients were sent.',
  /** {count} is {events} formatted, for a metric other than the three. */
  summaryEvents:
    '{metric, select, complaint {{events, plural, one {# complaint} other {# complaints}} out of {base} delivered.} bounce {{events, plural, one {# bounce} other {# bounces}} out of {base} sent.} deferral {{events, plural, one {# deferral} other {# deferrals}} out of {base} sent.} other {{count} {metric} out of {base} sent.}}',
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

  // "What tripped".
  factWho: '{metric, select, complaint {{who} complaints} bounce {{who} bounces} deferral {{who} deferrals} other {{who} {metric}}}',
  factAll: '{metric, select, complaint {Complaints} bounce {Bounces} deferral {Deferrals} other {{Metric}}}',
  factValue: '{events} of {base} · {rate}',
  factLimit: '{scope, select, mailbox_provider {Limit, per mailbox provider} route {Limit, per sending provider} other {Limit, whole campaign}}',
  factCampaignWide: '{metric, select, complaint {Campaign-wide complaints} bounce {Campaign-wide bounces} deferral {Campaign-wide deferrals} other {Campaign-wide {metric}}}',

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
  badgeProviderHeld: '{provider} held',
  badgeStarting: 'Starting',
  badgeSendingLive: 'Sending · live',
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
  startingBuilding: 'Building the queue for {count} recipients: suppressed, unsubscribed and paused contacts are left out.',
  startingBuildingNoCount: 'Building the queue: suppressed, unsubscribed and paused contacts are left out.',
  startingFirstBatches: "The first batches go to {providers} as soon as it's ready.",
  startingBuilt: 'Queue built for {count} recipients. Handing the first batches to {providers}; delivery events usually arrive within 10 seconds.',
  startingBuiltYours: 'Queue built for {count} recipients. Handing the first batches to your providers; delivery events usually arrive within 10 seconds.',
  finishedAt: '{name} finished at {time}',
  finished: '{name} finished',
  finishedIn: '{sent} sent in {took}: {delivered} delivered ({rate}), {bounced} bounced, {complaints, plural, one {# complaint} other {# complaints}}.',
  finishedNoTime: '{sent} sent: {delivered} delivered ({rate}), {bounced} bounced, {complaints, plural, one {# complaint} other {# complaints}}.',
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
    '"{reason}" {count} recipients are waiting. Nothing more is sent until someone with {perm} resumes it; what was already sent keeps reporting opens and clicks.',
  pausedBody: '{count} recipients are waiting. Nothing more is sent until someone with {perm} resumes it; what was already sent keeps reporting opens and clicks.',
  lostTitle: 'Lost the live event stream at {time}',
  lostBody:
    "The browser's connection to ReputeCast dropped. Sending continues on the server; the counts below are from {asOf} and may be up to {behind, plural, one {# minute} other {# minutes}} behind.",
  lostStopped: 'Stopped reconnecting after {max} attempts.',
  lostRetrying: 'Reconnecting automatically (attempt {attempt} of {max}).',
  mobileEta: 'ETA {eta} · about {time}',
  mobileEtaPaused: 'Paused',

  // The events line in the top bar. {pause} and {send} are permission keys.
  eventsReconnecting: 'Reconnecting to the event stream',
  /** {via} names how events arrive ("SNS and Postal webhooks"). */
  eventsVia: 'Events via {via}',
  eventsLive: 'Events live',
  eventsLast: 'last {ago}',
  eventsWaiting: 'waiting for the first events',
  eventsThreshold: 'anyone with {pause} can stop a send; resume needs {send}',
  eventsPerms: 'pause needs {pause}, resume needs {send}',
  mobileReconnecting: 'Reconnecting…',
  mobileLive: 'Live · events {ago}',
  mobileWaiting: 'Live · waiting for the first events',

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
