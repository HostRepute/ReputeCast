import { defineMessages } from '@/i18n'

/**
 * Sending providers › Warm-up (ProvidersWarmup): the plan, its ramp, settings
 * and log, the new plan dialog and the day words of warmup-model. The tiers
 * (Active, Cooling, At risk, Sunset) are Audience › Health's.
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
  asideLabel: 'Plan settings and log',

  // Pausing and resuming.
  pause: 'Pause warm-up',
  resume: 'Resume warm-up',
  paused: 'Warm-up paused',
  pausedDetail: 'Today’s cap stays where it is until you resume.',
  resumed: 'Warm-up resumed',
  resumedDetail: 'The ramp continues from today’s cap.',
  changeFailed: "The warm-up didn't change",
  allHavePlans: 'Every connected provider already has a warm-up plan.',

  // The plan's headline.
  titleIp: '{provider} · dedicated IP {ip}',
  badgePaused: 'Paused · day {day} of {days}',
  badgeDone: '{days, plural, one {Warmed up · # day} other {Warmed up · # days}}',
  /** {date} is a short date ("Thu 9 Oct"). */
  started: 'started {date}',
  finished: 'Finished {date}; {provider} can take Broadcast failover',
  finishes: 'Finishes {date}, then {provider} is eligible as a Broadcast failover target',
  progressLabel: 'Day {day} of {days}',
  todayCap: "Today's cap",
  /** {sent} is formatted; {day} is today's weekday ("Tuesday"). */
  sentOn: '{sent} sent · {day}',
  sentOnQuiet: '{sent} sent · {day}, no broadcasts',
  sentToday: '{sent} sent · today',
  sentDuring: 'Sent during warm-up',
  plannedSoFar: 'of {planned} planned so far',
  bounceGuard: 'Bounce rate · guard {limit}',
  complaintGuard: 'Complaint rate · guard {limit}',
  sevenDay: '7-day',
  slowdowns: 'Slowdowns',
  /** {bounce} is the day's bounce rate ("2.4%"). */
  lastSlowdown:
    '{action, select, step_back {day {day} stepped back, bounces {bounce}} pause {day {day} paused, bounces {bounce}} other {day {day} held, bounces {bounce}}}',
  noneSoFar: 'none so far',
  rateToday: 'Rate limit today',
  /** {rate} is a rate per second ("14 /s"). */
  risesTo: 'rises to {rate} on day {day}',
  staysThere: 'stays there to the end',

  // The ramp.
  rampTitle: 'Daily volume ramp',
  legendPlanned: 'planned cap',
  legendSent: 'sent',
  /** {first} and {last} are daily caps, formatted. */
  rampLabel: 'Daily caps rising from {first} on day 1 to {last} on day {day}',
  rampLabelSent: 'Daily caps rising from {first} on day 1 to {last} on day {day}, with sent volume filled in for days 1 to {done}',
  dayFirst: 'Day 1 · {cap}',
  dayToday: 'Day {day} · today · {cap}',
  dayLast: 'Day {day} · {cap}',
  colDay: 'Day',
  colDate: 'Date',
  colCap: 'Cap',
  colSent: 'Sent',
  colBounce: 'Bounce',
  colComplaints: 'Complaints',
  colAudience: 'Audience',
  colStatus: 'Status',
  /** A day's cap held at the day before's ("6,000 · held"). */
  capHeld: '{cap} · held',

  // A day (warmup-model).
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
  statusDone: 'Done',
  /** {limit} is the bounce guard ("2%"). */
  statusSlowdown: 'Slowdown · bounces over {limit}',
  statusResumed: 'Done · ramp resumed',
  statusToday: 'Today',
  statusPlanned: 'Planned',

  // Plan settings.
  settingsTitle: 'Plan settings',
  length: 'Length',
  length21: '21 days · recommended for a new dedicated IP',
  length14: '14 days · fast',
  length30: '30 days · cautious',
  ordering: 'Ordering',
  orderEngaged: 'Most engaged contacts first (Active, then Cooling, then At risk)',
  orderRandom: 'Random sample',
  slowIfBounces: 'Slow down if bounces exceed',
  orComplaints: 'or complaints exceed',
  whenTrips: 'When a guard trips',
  actionHold: "Hold tomorrow's cap at today's level, then continue",
  actionStepBack: 'Step back one day',
  actionPause: 'Pause until I resume',
  spread: "Spread each day's cap evenly across mailbox providers, so no single provider sees a spike from the new IP",
  savePlan: 'Save plan',
  saved: 'Warm-up plan saved',
  savedDetail: 'Changes apply from tomorrow’s cap.',
  /** {min} and {max} are percentages ("0.5%", "10%"). */
  bounceRange: 'Use a bounce guard between {min} and {max}.',
  complaintRange: 'Use a complaint guard between {min} and {max}.',

  // The log.
  logTitle: 'Warm-up log',
  logEmpty: "Nothing yet: the first day's result shows up here tomorrow.",

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
