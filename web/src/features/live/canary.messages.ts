import { defineMessages } from '@/i18n'

/**
 * The live monitor's canary (LiveMonitorCanary, LiveMonitorCanaryPhone;
 * review GS-10): the most engaged few percent go first, everyone else waits
 * until the gates pass.
 */
export const mc = defineMessages('live.canary', {
  titleWatching:
    '{seeds, plural, =0 {Canary · {size} most-engaged recipients ({percent}) · watching until {until} ({left} min left)} other {Canary · {size} most-engaged recipients ({percent}) plus {seeds} seed mailboxes · watching until {until} ({left} min left)}}',
  titleSending:
    '{seeds, plural, =0 {Canary · sending to {size} most-engaged recipients ({percent})} other {Canary · sending to {size} most-engaged recipients ({percent}) plus {seeds} seed mailboxes}}',
  titleHeld: 'Canary held: {gate} failed its gate',
  titleStopped:
    '{seeds, plural, =0 {Canary · stopped at {time} · {size} most-engaged recipients ({percent}) got it} other {Canary · stopped at {time} · {size} most-engaged recipients ({percent}) plus {seeds} seed mailboxes got it}}',
  sentLine: 'Sent {from} to {to}. {queued} recipients are queued until the canary passes.',
  sendingLine: 'Sending since {from}. {queued} recipients are queued until the canary passes.',
  stoppedLine: 'The {queued} queued recipients were removed from this send. The canary results stay in the report.',

  // The summary line: the screen's one polite live region.
  sumGates:
    '{collecting, plural, =0 {{passing} of {total} gates are passing and none has failed.} one {{passing} of {total} gates are passing and none has failed; {names} {singular, select, yes {is} other {are}} still collecting evidence.} other {{passing} of {total} gates are passing and none has failed; {names} are still collecting evidence.}}',
  sumWatchingUntil: 'Watching until {until}.',
  sumFailed: 'The {gate} gate failed: the send is held until someone continues or stops it.',
  sumStopped: "Stopped at {time}. The {queued} queued recipients won't get this email.",
  sumRates: 'Main-send rates changed: {changes}. If it continues at {at}, everyone has it by about {time}.',

  // Why it runs.
  why: 'Why it runs:',
  whyNeverMailed: '{count, plural, one {# recipient has never had mail from {domain}.} other {# recipients have never had mail from {domain}.}}',
  whyNewDomain: '{domain} is new: this is among its first 30 days of sending.',
  whyOverUsual: 'The audience of {audience} is over twice the usual {usual}.',
  whyChosen: 'A canary was chosen in Review & send.',
  rulesLink: 'Canary rules',
  honest:
    "The canary goes to your most-engaged recipients, so it checks blocks, deferrals and inbox placement, not how new imports will react. Their complaints are watched during the main send and for {hours} h after it.",

  // Gates.
  gates: 'Gates',
  gate: '{key, select, bounce {Bounces} block_bounce {Block/policy bounces} deferral {Deferrals} complaint {Complaints} other {Seed placement}}',
  gateState: '{state, select, passing {Passing} failing {Failing} other {Collecting evidence}}',
  valueOf: '{rate} · {events} of {base}',
  limit: 'limit {limit}',
  limitPerMailbox: 'limit {limit} per mailbox provider',
  limitSeeds: 'limit {limit} in spam',
  complaintValue: '{unreported, select, none {{count} from {reported}} other {n/a for {unreported} · {count} from {reported}}}',
  complaintNeeds: '{base} of {min} delivered to {mailbox} before this can trip',
  seedValue: '{promotions, plural, =0 {Inbox {inbox} · Spam {spam}} other {Inbox {inbox} · Promotions {promotions} · Spam {spam}}}',
  seedPending: '{pending} not found yet',
  seedMissing: '{missing} never arrived',
  seedsTitle: 'Seed placement by mailbox provider',
  seedsBar: '{seeds} seed mailboxes: {inbox} inbox, {promotions} Promotions tab, {spam} spam',
  seedLegendInbox: 'Inbox {count}',
  seedLegendPromotions: 'Promotions tab {count}',
  seedLegendSpam: 'Spam {count}',
  seedsOf: '{count, plural, one {# seed} other {# seeds}}',
  seedsIncl: '{mailbox, select, gmail {incl. Google Workspace} outlook {incl. Microsoft 365} yahoo {incl. AOL} other {GMX, Web.de}}',
  seedsNote: 'Seeds are your own test mailboxes in <a>Seed mailboxes</a>; they show where the mail lands, not how people react.',

  // Decisions.
  choose: 'Choose how to continue',
  continueNow: 'Continue to everyone now',
  continueNowBody:
    'Starts the main send now instead of at {until}. Gates still collecting evidence become thresholds for the main send. Everyone has it by about {eta}.',
  continueHeldBody: 'Sends to the {queued} recipients the canary held. Every threshold stays armed for the main send. Everyone has it by about {eta}.',
  watchMore: 'Watch 10 more minutes',
  watchMoreBody: 'Holds the {queued} queued recipients until {until}. Everyone has it by about {eta}.',
  stop: 'Stop the send',
  stopBody: "The {queued} queued recipients won't get this email. The {size} canary recipients already have it.",
  stopConfirm: 'Stop {name} for the {queued} queued recipients?',
  stopConfirmBody: "This can't be undone; you can send a new copy later.",
  keepWatching: 'Keep watching',
  autoNote:
    "If nobody chooses: at {until}, when every gate with enough evidence passes, ReputeCast continues to everyone by itself. If a gate fails, the send stays held and you're asked.",
  stoppedBy: 'Stopped at {time} by {name}',
  stoppedAt: 'Stopped at {time}',
  continued: 'Continued to everyone',
  watched: 'Watching 10 more minutes',
  stopped: 'Send stopped',
  decideFailed: "That didn't work",
  needsApprove: 'Continuing early needs someone who can approve sends{helpers, select, none {} other {: ask {helpers}}}.',
  needsSend: 'Only people who can send campaigns decide on a canary{helpers, select, none {} other {: ask {helpers}}}.',

  // Progress.
  queuedBig: 'recipients queued until the canary passes',
  progressLine: '{sent} of {total} sent ({percent}) · if it continues at {until}, everyone has it by about {eta}',
  progressLineNow: '{sent} of {total} sent ({percent})',
  barLabel: 'Canary sent: {parts}. {queued} queued.',
  viaProvider: 'Canary via {provider} {count}',
  queuedLegend: 'Queued {count}{routes, select, none {} other { · {routes}}}',
  abNote: 'The A/B subject test runs on the main send.',
  progressLabel: 'Canary progress',

  // Tiles.
  deliveredSub: '{rate} of the canary{retrying, plural, =0 {} other { · # retrying}}',
  deferredSub: '{rate} of the canary',
  sinceDelivered: '{count} since delivered',
  retryingCount: '{count} retrying',
  complainedSub: '{unreported, select, none {from {reported}} other {from {reported} · {unreported} n/a}}',
  tileDeferred: 'Deferred',

  // The badge.
  badge: '{phase, select, sending {Canary · sending} watching {Canary · watching} held {Canary · held} other {Stopped}}',
  passed: 'Canary {percent} · passed {time}',
  passedEarly: 'Canary {percent} · continued {time}',
  mailboxCaption: 'canary results · rates apply to the main send',
  // The canary's tables (LiveMonitorCanary).
  colCanarySent: 'Canary sent',
  colDeferredOnce: 'Deferred once or more',
  colMainRate: 'Main-send rate',
  colThresholds: 'Thresholds',
  colQueued: 'Queued',
  deliveredRetrying: '{rate} · {retrying} retrying',
  noneSoFar: '0 so far',
  minNote:
    "{count, plural, =0 {A mailbox provider's thresholds can trip only after {min} delivered to it; in the canary none has that many yet.} one {A mailbox provider's thresholds can trip only after {min} delivered to it; in the canary only {names} has that many.} other {A mailbox provider's thresholds can trip only after {min} delivered to it; in the canary only {names} have that many.}}",
  ratesTimes: 'Times assume the main send starts at {until}.',
})
