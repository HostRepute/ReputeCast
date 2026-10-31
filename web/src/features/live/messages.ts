import { defineMessages } from '@/i18n'

/**
 * The live sending monitor's screens (LiveMonitor boards): the page, its
 * cards and tables, the decision view, the phone layout, the states and
 * the actions. The words computed from a snapshot are in model.messages.ts.
 */
export const m = defineMessages('live', {
  // ── Page ─────────────────────────────────────────────────────────────
  title: 'Live sending monitor',
  campaigns: 'Campaigns',
  /** The browser tab: {name} is the campaign. */
  documentTitle: '{name} · Live',
  backToLive: 'Back to live view',
  pause: 'Pause',
  mobileSent: '{count} sent',
  detailCanaryLabel: 'Canary',
  detailAfterLabel: 'After the send',
  reconnect: 'Reconnect',
  cancelSend: 'Cancel send…',
  cancelBadge: 'Cancel send',
  backToCampaigns: 'Back to campaigns',
  forbiddenTitle: "Your role can't watch this send",
  notFoundBody: 'The campaign may have been deleted, or the link is from another workspace.',
  serverErrorNote: 'Sending continues on the server.',

  // ── Progress, counters and tables ────────────────────────────────────
  progress: 'Progress',
  /** Read by screen readers as the percentage changes: {percent} is "62%". */
  percentSent: '{percent} sent',
  asOf: 'as of {time}',
  perMailbox: 'Per mailbox provider',
  perMailboxDecision: 'thresholds apply per mailbox provider',
  perMailboxLive: 'where recipients read the mail',
  colMailbox: 'Mailbox provider',
  colSent: 'Sent',
  colRemaining: 'Remaining',
  colDelivered: 'Delivered',
  colBounce: 'Bounce',
  colComplaint: 'Complaint',
  colOpened: 'Opened',
  colRate: 'Rate',
  colStatus: 'Status',
  perProvider: 'Per sending provider',
  // Adjust rates (review GS-13; LiveMonitor, LiveMonitorStates, LiveMonitorCanary).
  adjustRates: 'Adjust rates',
  ratesTitle: 'Rates per mailbox provider',
  ratesCaption: 'For the rest of this send, on every route. Lower a rate to ease off one mailbox provider; the others keep their pace.',
  ratesCaptionCanary: 'For the main send after the canary, on every route. Lower a rate to ease off one mailbox provider; the others keep their pace.',
  ratePerSecond: '{rate} /s',
  rateFull: 'full rate',
  rateNow: 'now {rate}',
  ratePlanned: 'planned {rate}',
  rateWas: 'was {rate}',
  rateEffect: 'finishes about {to}',
  rateEffectInstead: 'finishes about {to} instead of {from}',
  rateEffectFull: 'limited only by the sending provider',
  ratesWhole: 'Whole send:',
  ratesWholeSame: 'about {time} (no change yet)',
  ratesWholeChange: 'about {from} → about {to} ({way, select, later {{diff} min later} sooner {{diff} min sooner} other {same minute}})',
  ratesWholeSameCanary: 'everyone by about {time} (no change yet)',
  ratesWholeChangeCanary:
    'everyone by about {from} → about {to} ({way, select, later {{diff} min later} sooner {{diff} min sooner} other {same minute}})',
  ratesApplied: 'Rates changed: {changes}. The whole send now finishes about {time}.',
  ratesAppliedCanary: 'Main-send rates changed: {changes}. If it continues at {at}, everyone has it by about {time}.',
  ratesHeld: '{who} is on hold.',
  ratesHeldNote: 'Its rate applies when it resumes.',
  rateGroup: '{who} rate',
  rateLower: 'Lower {who} rate',
  rateRaise: 'Raise {who} rate',
  ratesFailed: "The rates didn't change",
  apply: 'Apply',
  workspaceDefaults: 'Workspace defaults',
  // Plain SMTP reports no deliveries: they're inferred from the server's acceptance (review GS-7).
  deliveredInferred: 'inferred',
  inferredNote: 'Delivered (inferred): plain SMTP reports no deliveries, so a message counts as delivered once the server accepts it. Bounces and complaints still arrive as reports.',
  colProvider: 'Sending provider',
  colShare: 'Share',
  colApiErrors: 'API errors',
  colApiErrors3: 'API errors · 3 min',
  liveEvents: 'Live events',
  streaming: 'streaming',
  reconnecting: 'reconnecting',
  eventsEmpty: 'Delivery events appear here as providers report them.',
  eventsList: 'Newest events first',
  thresholds: 'Auto-pause thresholds',
  editThresholds: 'Edit',
  /** Opens the thresholds dialog of a running or waiting send (LiveMonitor). */
  editThresholdsLive: 'Edit for this send',
  sendDetails: 'Send details',
  detailFrom: 'From',
  detailAudience: 'Audience',
  detailRoute: 'Route',
  detailCost: 'Cost so far',
  /** {cost} so far of {planned}, both money. */
  detailCostValue: '{cost} of {planned}',
  detailScheduledBy: 'Scheduled by',
  deliveredBy: 'Delivered by mailbox provider',
  sharesEmpty: 'Shares appear once the first deliveries are reported.',

  // ── Decision view ────────────────────────────────────────────────────
  chooseTitle: 'Choose how to continue',
  /** Above the list of checks run on a move's target (review GS-5). */
  checksTitle: 'Checks ReputeCast ran on {provider}',
  chooseLabel: 'Choose how to continue',
  cancelWhole: 'Cancel whole campaign…',
  pickFirst: 'Pick how to continue first.',
  continue: 'Continue',
  whatTripped: 'What tripped',
  /** {provider} is the sending provider ("Postal"). */
  whatReturned: 'What {provider} returned',
  healthDown: "ReputeCast's health checks marked the provider down; no send call to it has succeeded since.",
  likelyCauses: 'Likely causes',
  noCauses: 'Nothing in this send explains it yet; the evidence is on the left.',
  recommended: 'Recommended',
  recommendedDefault: "Resume when you're confident, or cancel this share; the rest of the campaign keeps going.",
  reputationRisk: 'Reputation risk',

  // ── Phone layout ─────────────────────────────────────────────────────
  /** {sent} of {total} recipients, both counts. */
  sentOfTotal: '{sent} of {total}',
  thresholdsShort: 'Thresholds',
  autoPause: 'auto-pause',
  resume: 'Resume',
  cancelShort: 'Cancel',

  // ── States ───────────────────────────────────────────────────────────
  reconnectNow: 'Reconnect now',
  shareLink: 'Share read-only link',
  viewReport: 'View campaign report',
  failedTitle: '{name} stopped before it finished',
  failedBody: '{sent} of {total} were sent; the rest is marked “not sent”. Nothing more goes out, and what was sent keeps reporting opens and clicks.',
  reviewSend: 'Review & send',
  scheduledBody:
    '{name} goes to {count} recipients. When it starts, Send Readiness runs once more; then progress, counts per mailbox provider and sending provider and the auto-pause thresholds appear here, live.',
  idleTitle: 'Nothing is sending yet',
  idleBody:
    "{name} hasn't started. Once it does, this page shows its progress, the counts per mailbox provider and sending provider, the live events and the auto-pause thresholds.",
  resumeSending: 'Resume sending',
  /** {providers} lists the sending providers still sending ("SES and Postmark"). */
  othersKeepSending: '{providers} keeps sending.',
  nothingElseSending: 'Nothing else is sending.',
  everythingElseSending: 'Everything else keeps sending.',
  /** A held share testing its provider while nobody has decided (review GS-3); {provider} is "Postal". */
  bannerProbing: 'Meanwhile ReputeCast tests {provider} with 1 message every 60 s.',
  /** {who} is a mailbox provider ("Outlook") or a sending provider ("Postal"). */
  heldUntilWho: '{count} {who} recipients are held until someone decides.',
  heldUntil: '{count} recipients are held until someone decides.',
  loading: 'Loading the live monitor…',

  // ── Actions ──────────────────────────────────────────────────────────
  nothingChanged: 'Nothing was changed.',
  pauseFailed: "Couldn't pause the send",
  resumeFailed: "Couldn't resume the send",
  // Resume sending (Dialogs board; review GS-2). {metric} is complaint, bounce or deferral.
  resumeBadge: 'Resume sending',
  resumeTitleWho: 'Resume the {held} held {who} recipients?',
  resumeTitleAll: 'Resume the {held} held recipients?',
  resumeWasWho:
    'The {who} {metric, select, complaint {complaint rate} bounce {bounce rate} deferral {deferral rate} other {rate}} was {value} when ReputeCast paused at {time}.',
  resumeWasAll:
    'The {metric, select, complaint {complaint rate} bounce {bounce rate} deferral {deferral rate} other {rate}} was {value} when ReputeCast paused at {time}.',
  resumeSince: 'Since then: {value} on the {base} {metric, select, complaint {delivered} other {sent}}.',
  resumeLimit: 'The threshold is {limit}.',
  resumeRearmAt:
    'Resume at {rate} and re-pause if {metric, select, complaint {complaints pass} bounce {bounces pass} deferral {deferrals pass} other {the rate passes}} {limit} again',
  resumeRearm:
    'Resume and re-pause if {metric, select, complaint {complaints pass} bounce {bounces pass} deferral {deferrals pass} other {the rate passes}} {limit} again',
  resumeRaise: 'Resume with the threshold raised to {limit} for this campaign only',
  resumeRaiseNote: 'Logged in the audit log with your name.',
  keepPaused: 'Keep paused',
  resumeWho: 'Resume {who}',
  // Edit for this send: the send's thresholds while it sends or waits.
  thresholdsEditTitle: 'Auto-pause thresholds for this send',
  thresholdsEditBody: 'They apply to the rest of {name} only; the workspace defaults stay as they are. Logged in the audit log with your name.',
  thresholdsBounce: 'Bounces, whole send (%)',
  thresholdsComplaint: 'Complaints, whole send, where reported (%)',
  thresholdsMailboxComplaint: 'Complaints per mailbox provider (%)',
  thresholdsDeferral: 'Deferrals per mailbox provider (%)',
  thresholdsApiErrors: 'Sending provider API errors, over 3 min (%)',
  thresholdsScope: 'When one trips',
  thresholdsScopeRoute: 'Pause only the affected route or mailbox provider',
  thresholdsScopeCampaign: 'Pause the whole campaign',
  thresholdRange: 'Between {min} and {max}.',
  thresholdsFix: 'Fix the values first',
  thresholdsSave: 'Save thresholds',
  thresholdsSaved: 'Thresholds saved',
  thresholdsSavedDetail: 'The guard uses them within seconds.',
  thresholdsFailed: "Couldn't save the thresholds",
  // The complaint watch after the send (LiveMonitorStates: complete; review GS-6).
  watchWatching: 'Watching',
  watchEnded: 'Watch ended',
  watchUntil: 'until {until}',
  watchNoPerMessage: 'no complaints per message',
  watchNotReported: 'Not reported',
  watchPostmaster: "Gmail's Postmaster Tools spam rate arrives once a day: {rate} on its latest day.",
  watchFootnote: "Late complaints add the contact to the suppression list and count in this campaign's report. Most arrive in the first 24 hours.",
  watchAlertWho: '{who} complaints reached {value} after the send (limit {limit})',
  watchAlertAll: 'Complaints reached {value} after the send (limit {limit})',
  /** From Gmail Postmaster Tools: one spam rate a day for the sending domain. */
  watchAlertPostmaster: "Gmail's spam rate reached {value} after the send (limit {limit}), from Postmaster Tools",
  acknowledge: 'Acknowledge',
  acknowledgeFailed: "Couldn't acknowledge the alert",
  cancelFailed: "Couldn't cancel",
  moveFailed: "Couldn't move the share",
  retryFailed: "Couldn't retry",
  // Messages in doubt (review GS-1). {n} is {count} formatted.
  inDoubtResent: '{count, plural, one {# message in doubt goes} other {{n} messages in doubt go}} out again',
  inDoubtLeftOut: '{count, plural, one {# message in doubt left out} other {{n} messages in doubt left out}}',
  inDoubtLeftOutDetail: 'The report shows them as "not sent, in doubt".',
  inDoubtFailed: "Couldn't decide on the messages in doubt",
  /** On the in-doubt panel's button while one of the people below decides. */
  inDoubtWho: '{names} can decide',
  cancelInDoubt: '{count, plural, one {The # message in doubt is} other {The # messages in doubt are}} left out too; the report shows {count, plural, one {it} other {them}} as "not sent, in doubt".',
  resumedWithChanges: 'Resumed with changes',
  /** {who} is a mailbox provider ("Outlook"). */
  resumedWhoWithChanges: '{who} resumed with changes',
  excludedDetail: '{count} contacts left out of the rest of this send.',
  shareCancelled: 'Share cancelled',
  shareCancelledDetail: '{count} recipients won\'t be sent; they\'re marked "not sent" in the report.',
  /** {target} is the sending provider the recipients moved to ("SES"). */
  moved: 'Moved {count} recipients to {target}',
  movedToOther: 'Moved {count} recipients to the other provider',
  movedDetail: 'They joined its queue; what was already sent keeps its tracking.',
  cancelTitle: 'Cancel the rest of {name}?',
  cancelBody: '{sent} of {total} are already delivered and cannot be recalled. {remaining} are queued and will not be sent. The report keeps everything sent so far.',
  pauseInstead: 'Pause instead',
  cancelRemaining: 'Cancel remaining sends',
  pausing: 'Pausing…',
  cancelling: 'Cancelling…',
  keepSending: 'Keep sending',
  cancelRemainingCount: 'Cancel the remaining {count}',
  cancelRemainingBody: 'Marks the campaign as partially sent. You can send the remainder later as a new campaign.',
  pauseInsteadBody: 'Keeps the queue; resume from the monitor at any time.',
  // A share today's warm-up cap holds (review PR-8): it waits for
  // tomorrow's cap by itself; the choices are R03's (warm-up holds).
  /** {provider} is the sending provider ("Postal"). */
  warmupHoldTitle: '{provider} share waits at its warm-up cap',
  warmupWaiting: 'Chosen: wait for the cap.',
  /** {names} are up to two people, joined ("Linh Nguyen and Marcus Chen"). */
  warmupWhoMove: '{names} can wait or move it',
  warmupWhoCancel: '{names} can cancel the rest',
  warmupWhoAll: '{names} can wait, move it or cancel the rest',
  warmupCancelTitle: 'Cancel the {count} recipients waiting on {provider}?',
  warmupKeep: 'Keep waiting',
})
