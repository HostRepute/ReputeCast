import { defineMessages } from '@/i18n'

/**
 * Sending providers › Warm-up (ProvidersWarmup): the plan, how it moves, its
 * ramp and days, today's caps by mailbox provider, settings, log, shares
 * held at the cap, the new plan dialog and the day words of warmup-model.
 * The tiers (Active, Cooling, At risk, Sunset) are Audience › Health's.
 */
export const mw = defineMessages('providers.warmup', {
  loadFailed: "Couldn't load the warm-up",
  loadFailedBody: 'The ramp keeps running on the saved plan.',
  loading: 'Loading the warm-up plan',
  emptyTitle: 'No warm-up running',
  emptyBody:
    "A new dedicated IP or a new provider needs a few weeks of slowly rising volume before it can carry your Broadcasts. Start a plan and ReputeCast ramps the provider's share daily, sending to your most engaged contacts first.",
  newPlan: 'New warm-up plan',
  planPicker: 'Warm-up plan',
  asideLabel: "Today's caps, plan settings and log",

  // Pausing and resuming.
  pause: 'Pause warm-up',
  resume: 'Resume warm-up',
  paused: 'Warm-up paused',
  pausedDetail: 'The plan stays on its day and sends nothing until you resume.',
  resumed: 'Warm-up resumed',
  resumedDetail: 'The plan goes on from the day it stopped.',
  changeFailed: "The warm-up didn't change",
  allHavePlans: 'Every connected provider already has a warm-up plan.',

  // The plan's headline.
  titleIp: '{provider} · dedicated IP {ip}',
  badgePaused: 'Paused · day {day} of {days}',
  badgeSlowed: 'Cap held · day {day} of {days}',
  badgeDone: '{days, plural, one {Warmed up · # day} other {Warmed up · # days}}',
  /** {date} is a short date ("Thu 2 Oct"). */
  started: 'started {date}',
  finished: 'Finished {date}; {provider} can take Broadcast failover',
  finishes: 'Finishes {date} at the earliest, then {provider} is eligible as a Broadcast failover target',
  counted: '{counted} of {days} days counted',

  // How the plan moves.
  howTitle: 'How the plan moves',
  /** {share} is a percentage ("80%"). */
  howCounts: 'A day counts once at least {share} of its cap was sent and bounces stayed under the guard.',
  /** {weekday} is today's weekday ("Sunday"); {sent} is formatted. */
  howRepeats: '{weekday} sent {sent}, so day {day} repeats tomorrow.',
  /** {need} and {sent} are formatted. */
  howNeeds: 'Today counts once {need} are sent; {sent} so far.',
  howPausedNow: 'The plan is paused on day {day}.',
  howSample: 'Guards judge a day only after {count} messages.',
  howHold:
    '{action, select, step_back {After a guard trips, the cap steps back a day and holds until {days} healthy days in a row.} pause {After a guard trips, the plan pauses until someone resumes it.} other {After a guard trips, the cap holds until {days} healthy days in a row.}}',
  howPause: 'Pausing freezes the plan; it resumes on the day it stopped.',

  // The plan's numbers.
  todayCap: "Today's cap",
  /** {sent} is formatted; {day} is today's weekday ("Sunday"). */
  sentOn: '{sent} sent · {day}',
  sentOnQuiet: '{sent} sent · {day}, no broadcasts',
  sentDuring: 'Sent during warm-up',
  /** {counted} of {days} calendar days so far. */
  daysCounted: '{counted} of {days} days counted',
  bounceGuard: 'Bounce rate · guard {limit}',
  complaintGuard: 'Complaint rate · guard {limit}',
  sevenDay: '7-day',
  notAvailable: 'n/a',
  bounceUnknown: 'nothing sent in the last 7 days',
  complaintUnknown: "no reports yet; Gmail and iCloud don't report complaints",
  slowdowns: 'Slowdowns',
  /** {rate} is the day's rate ("2.3%"). */
  lastSlowdown: '{metric, select, complaint {day {day}, complaints {rate}} other {day {day}, bounces {rate}}}',
  noneSoFar: 'none so far',
  rateToday: 'Rate limit today',
  /** {rate} is a rate per second ("4 /s"). */
  risesTo: 'rises to {rate} on day {day}',
  staysThere: 'stays there to the end',

  // The ramp.
  rampTitle: 'Daily volume ramp',
  legendPlanned: 'planned cap',
  legendSent: 'sent',
  legendTripped: 'guard tripped',
  /** {first} and {last} are daily caps, formatted. */
  rampLabel: 'Planned daily caps by plan day, drawn to scale: {first} on day 1 rising to {last} on day {days}.',
  rampLabelSent: '{days, plural, one {Day 1 is sent.} other {Days 1 to # are sent.}}',
  rampLabelTripped: 'Day {day} tripped a guard.',
  rampLabelToday: 'Day {day}, today, plans {cap}.',
  dayFirst: 'Day 1 · {cap}',
  dayToday: 'Day {day} · today · {cap}',
  dayLast: 'Day {day} · {cap}',

  // The days.
  tableCaption: 'Warm-up days, with whether each one counted',
  colDay: 'Day',
  colDate: 'Date',
  colCap: 'Cap',
  colSent: 'Sent',
  colBounce: 'Bounce',
  colComplaints: 'Complaints',
  colAudience: 'Audience',
  colCounted: 'Counted',
  /** Days folded into one row ("1–4"). */
  dayRange: '{first}–{last}',
  /** The last date of a folded row ("to Mon 6 Oct"). */
  dateTo: 'to {date}',
  todayWord: 'today',
  atTheEarliest: 'at the earliest',
  capHeld: 'held',
  /** {cap} is the schedule's cap, formatted. */
  capHeldPlan: 'held · plan {cap}',
  notJudged: 'not judged',
  badgeCounted: 'Counted',
  badgeNotCounted: 'Not counted',
  badgePlanned: 'Planned',
  badgeToday: 'Today',
  badgePausedDay: 'Paused',
  /** How many plan days a folded row covers. */
  noteDays: '{days, plural, one {# day} other {# days}}',
  /** A day that didn't count inside a folded row; {date} is a short date, {weekday} the next day's ("Monday"). */
  noteMissed: '{date} sent {sent}, so day {day} repeated on {weekday}',
  /** {limit} is the guard ("2%"); {cap} is formatted. */
  noteGuard:
    '{metric, select, complaint {complaints over {limit}} other {bounces over {limit}}} · {action, select, step_back {slowdown, the cap steps back to {cap}} pause {slowdown, the plan paused} other {slowdown, the cap holds at {cap}}}',
  noteShort: '{sent} of {cap} sent · day {day} repeated',
  noteHealthy: 'healthy day {n} of {of}',
  noteHealthyLast: 'healthy day {n} of {of} · ramp resumes',
  noteTodayRepeats: '{sent} of {cap} sent · day {day} repeats tomorrow',
  noteTodayNeeds: '{sent} of {need} sent to count',
  notePaused: 'the plan is paused',
  noteRepeat: 'repeat of day {day}',

  // A day's audience (warmup-model).
  tierActive: 'Active',
  tierCooling: 'Cooling',
  tierAtRisk: 'At risk',
  tierSunset: 'Sunset',
  tiersAll: 'All',
  /** {tier} is a tier's name ("Active"). */
  tierOne: '{tier} tier',
  tiersExceptSunset: 'All tiers except Sunset',
  /** {name} is a campaign's name; {count} how many it sends that day, formatted. */
  dayUses: '{name} uses {count}',
  /** {share} is a percentage ("40%"). */
  dayEligible: 'share {share}, eligible as failover',
  dayShareRises: 'share rises to {share}',

  // Today's caps by mailbox provider.
  capsTitle: "Today's caps by mailbox provider",
  /** {cap} is formatted. */
  capsTotal: 'day {day} · {cap} in total',
  capsCaption: "{provider}'s caps for today, by mailbox provider",
  colMailbox: 'Mailbox provider',
  total: 'Total',
  capsMx: 'Gmail includes Google Workspace and Outlook includes Microsoft 365 recipients (found from MX records).',
  /** {cap} is formatted. */
  capsShared: "Mailbox providers aren't capped separately: today's {cap} can go to any of them.",
  sharedCap: 'shared',
  /** {provider} is a warm provider ("Amazon SES"). */
  overflowLabel: "Send what's over today's cap through {provider} (warm) instead of waiting",
  overflowLabelNone: "Send what's over today's cap through a warm provider instead of waiting",
  /** {count} is formatted. */
  overflowToday: 'Over the cap so far today: {count}',
  overflowNone: "No warm provider shares the Broadcasts route now, so what's over the cap waits for tomorrow's cap.",
  overflowOn: "What's over the cap goes through a warm provider",
  overflowOff: "What's over the cap waits for tomorrow's cap",

  // Plan settings.
  settingsTitle: 'Plan settings',
  length: 'Length',
  length21: '21 counted days · recommended for a new dedicated IP',
  length14: '14 counted days · fast',
  length30: '30 counted days · cautious',
  /** An option the plan already went past. */
  lengthPast: '{days} counted days · the plan is past that',
  /** {cap} and {last} are formatted. */
  lengthHint: "Today's cap stays {cap}; the days ahead ramp to {last} by day {days}.",
  ordering: 'Ordering',
  orderEngaged: 'Most engaged contacts first (Active, then Cooling, then At risk)',
  orderRandom: 'Random sample',
  slowIfBounces: 'Slow down if bounces exceed',
  orComplaints: 'or complaints exceed',
  whenTrips: 'When a guard trips',
  actionHold: 'Hold the cap until {days} healthy days in a row',
  actionStepBack: 'Step back one day, then hold',
  actionPause: 'Pause until I resume',
  spread: "Cap each mailbox provider separately: Gmail 40%, Outlook 25%, Yahoo 15%, iCloud 10% and Other 10% of the day's cap",
  savePlan: 'Save plan',
  saved: 'Warm-up plan saved',
  savedDetail: "Today's cap stays where it is; the changes apply to the days ahead.",
  /** {min} and {max} are percentages ("0.5%", "10%"). */
  bounceRange: 'Use a bounce guard between {min} and {max}.',
  complaintRange: 'Use a complaint guard between {min} and {max}.',

  // The log.
  logTitle: 'Warm-up log',
  logEmpty: "Nothing yet: the first day's result shows up here tomorrow.",

  // Shares held at the cap.
  holdsLabel: 'Campaigns waiting at the warm-up cap',
  /** {campaign} is a campaign's name. */
  holdTitle: '"{campaign}" waits at {provider}\'s warm-up cap',
  /** {time} is when tomorrow's cap starts ("00:00"). */
  holdBody:
    "{count, plural, one {# recipient waits} other {# recipients wait}}. {paused, select, yes {They go out when the warm-up resumes} other {They go out with tomorrow's cap from {time}}}, unless you move them to a warm provider or cancel the rest.",
  holdWaiting: 'Chosen: wait for the cap.',
  holdWait: 'Wait',
  /** {provider} is the provider the held share would move to. */
  holdMove: 'Move to {provider}',
  holdMoveTarget: 'Move the held recipients to',
  holdCancel: 'Cancel the rest',
  holdCancelTitle: 'Cancel the held rest of "{campaign}"?',
  holdCancelBody:
    "{count, plural, one {# recipient on {provider} won't get this campaign.} other {# recipients on {provider} won't get this campaign.}} What already went out, and the rest of the send, stay as they are.",
  holdNoTarget: 'No connected provider can take them now:',
  holdWaitToast: 'The held recipients wait for the cap',
  /** {count} is formatted. */
  holdMovedToast: '{count} recipients move to {provider}',
  holdCancelledToast: '{count} recipients won’t get this campaign',
  holdFailed: "The held share didn't change",

  // New plan dialog.
  newPlanBody: "ReputeCast raises the provider's share of Broadcasts day by day, most engaged contacts first, and slows down if bounces or complaints climb.",
  startWarmup: 'Start warm-up',
  provider: 'Provider',
  dedicatedIp: 'Dedicated IP',
  dedicatedIpHint: "Leave empty to warm the provider's shared pool for this domain.",
  sharedPool: 'Shared pool',
  startedToast: '{provider} warm-up started',
  /** {cap} is formatted; {share} is a percentage ("10%"). */
  startedDetail: 'Day 1 cap {cap}; its Broadcasts share starts at {share}.',
})
