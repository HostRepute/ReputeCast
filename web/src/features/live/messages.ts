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
  /** {perm} is the permission key (campaigns:pause). */
  pauseNeeds: 'Pausing needs {perm}',
  cancelSend: 'Cancel send',
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
  chooseLabel: 'Choose how to continue',
  cancelWhole: 'Cancel whole campaign',
  pickFirst: 'Pick how to continue first.',
  continue: 'Continue',
  whatTripped: 'What tripped',
  /** {provider} is the sending provider ("Postal"). */
  whatReturned: 'What {provider} returned',
  healthDown: "ReputeCast's health checks marked the provider down; no send call to it has succeeded since.",
  retries: 'Retries used exponential back-off (5 s to 60 s). No message was sent twice: each attempt carries an idempotency key.',
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
  /** {who} is a mailbox provider ("Outlook") or a sending provider ("Postal"). */
  heldUntilWho: '{count} {who} recipients are held until someone decides.',
  heldUntil: '{count} recipients are held until someone decides.',
  loading: 'Loading the live monitor…',

  // ── Actions ──────────────────────────────────────────────────────────
  nothingChanged: 'Nothing was changed.',
  pauseFailed: "Couldn't pause the send",
  resumeFailed: "Couldn't resume the send",
  cancelFailed: "Couldn't cancel",
  moveFailed: "Couldn't move the share",
  retryFailed: "Couldn't retry",
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
})
