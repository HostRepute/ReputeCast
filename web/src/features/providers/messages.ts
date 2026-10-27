import { defineMessages } from '@/i18n'

/**
 * Sending providers & routing: the words every section shares (common.tsx):
 * the title and tabs, stream names, failover policies, provider statuses
 * and the limit labels. Each section's own text is in its
 * `<section>.messages.ts`.
 */
export const m = defineMessages('providers', {
  title: 'Sending providers & routing',
  sectionsLabel: 'Sections',
  tabProviders: 'Providers',
  tabRouting: 'Routing & failover',
  tabWarmup: 'Warm-up',
  tabTraffic: 'Traffic rules',
  tabEvents: 'Events & sync',
  tabAdapters: 'Adapters',
  /** The channel button; {channel} is "Email", "SMS"… */
  channelButton: 'Channel: {channel}. Change channel',
  channelLabel: 'Channel',

  // Streams, by their key.
  streamBroadcast: 'Broadcasts',
  streamAutomation: 'Automations',
  streamSystem: 'System',
  streamTransactional: 'Transactional',

  // Failover policies, in a sentence ("failover: pause and ask").
  failoverAutomatic: 'automatic',
  failoverPauseAndAsk: 'pause and ask',
  failoverWarmOnly: 'automatic, warmed-up only',

  // A provider's status badge.
  statusHealthy: 'Healthy',
  statusWarmingUp: 'Warming up',
  statusWarmingUpDay: 'Warming up · day {day} of {days}',
  statusDown: 'Down',
  /** {since} is how long: "14 min", "6 h 2 min". */
  statusDownSince: 'Down · {since}',
  statusDownAuth: 'Down · auth failed',
  statusDegraded: 'Degraded',
  /** {since} is how long: "14 min", "6 h 2 min". */
  statusDegradedSilent: 'Degraded · webhooks silent {since}',
  statusSandbox: 'Sandbox · verified addresses only',
  statusPaused: 'Paused',

  /** A capability the provider lacks: "Not supported:" is read by screen readers only. */
  notSupported: '<vh>Not supported:</vh> {capability}',
  secondsAgo: '{n} s ago',

  // Rate limits on a provider card. {limit}, {review} and {pause} are
  // percentages; {provider} names whose levels they are ("SES").
  limitGuard: '{kind, select, bounce {Bounce rate · warm-up guard {limit}} other {Complaint rate · warm-up guard {limit}}}',
  limit: '{kind, select, bounce {Bounce rate · limit {limit}} other {Complaint rate · limit {limit}}}',
  limitNamed: '{kind, select, bounce {Bounce rate · limit {limit} ({provider})} other {Complaint rate · limit {limit} ({provider})}}',
  limitReview: '{kind, select, bounce {Bounce rate · limit {limit} ({provider} review at {review})} other {Complaint rate · limit {limit} ({provider} review at {review})}}',
  limitPause: '{kind, select, bounce {Bounce rate · limit {limit} ({provider} pause at {pause})} other {Complaint rate · limit {limit} ({provider} pause at {pause})}}',
  limitReviewPause:
    '{kind, select, bounce {Bounce rate · limit {limit} ({provider} review at {review}, pause at {pause})} other {Complaint rate · limit {limit} ({provider} review at {review}, pause at {pause})}}',
})
