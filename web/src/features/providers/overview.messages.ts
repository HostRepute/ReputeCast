import { defineMessages } from '@/i18n'

/**
 * Sending providers › Providers (7.3): the page, the provider rows in every
 * state, a provider's settings dialog and removing a provider. The words
 * the sections share (title, tabs, streams, statuses) are in messages.ts.
 * {thousand} is the number 1,000 in the region format ("per 1,000 emails").
 */
export const mo = defineMessages('providers.overview', {
  // ── Page ─────────────────────────────────────────────────────────────
  loadFailed: "Couldn't load your sending providers",
  loadFailedBody: "Sending isn't affected: provider health is checked in the background.",
  providersDown: '{count, plural, one {# provider down} other {# providers down}}',
  providersDegraded: '{count, plural, one {# provider degraded} other {# providers degraded}}',
  /** {ago} is how long ago: "2 min ago". */
  checkedOne: 'Checked {ago}',
  checkedAll: 'All {count} providers checked {ago}',
  addProvider: 'Add provider',
  loading: 'Fetching health from your sending providers',

  // The summary tiles.
  tileProviders: 'Providers',
  tileConnected: '{count} connected',
  /** "2 Healthy · 1 Warming up": one status with its count. */
  statusCount:
    '{count} {status, select, healthy {Healthy} warming_up {Warming up} degraded {Degraded} down {Down} sandbox {Sandbox} other {Paused}}',
  tileSent: 'Sent this month, all providers',
  tileCost: 'Cost to date, {month}',
  costBlended: '{amount} per {thousand} blended',
  costNoMail: 'no mail sent yet this month',
  tileSuppression: 'Shared suppression list',
  syncedAll: 'synced to all providers {time}',
  syncing: 'syncing · last complete {time}',
  enforced: 'enforced before every send',

  // Routing per stream.
  routingTitle: 'Routing per stream',
  routingEdit: 'Edit rules, failover and route tester',
  legendPrimary: '{provider} primary',
  /** {share} is a percentage: "SES 90%". */
  legendShare: '{provider} {share}',
  legendFallback: '{provider} fallback',
  /** The split bar's label: "Broadcasts: SES 90%, Postal 10%, Postmark fallback". */
  routeLabel: '{stream}: {legend}',
  noProviderYet: 'No provider yet',
  /** {policy} is a failover policy in a sentence: "pause and ask". */
  routeFailover: 'failover: {policy}',
  routeConditionsFailover: '{count, plural, one {# condition} other {# conditions}} · failover: {policy}',

  // No provider yet.
  emptyTitle: 'No sending providers yet',
  emptyBody:
    'Connect Amazon SES, Postmark, Mailgun, SendGrid, Resend, SparkPost, Mailjet, Brevo, Azure Communication Services, a Postal server or any SMTP server. Each one takes about 10 minutes, and you can run several at once.',
  connectProvider: 'Connect a provider',
  browseAdapters: 'Browse adapters',
  emptyFoot: 'Routing, failover and warm-up become available once a provider is connected.',

  // ── A provider's row ─────────────────────────────────────────────────
  /** A stream it backs up and is carrying now. */
  roleActiveFallback: '{stream} · active fallback',
  roleFallback: '{stream} fallback',
  /**
   * A stream it carries: its share ({share} is a percentage), as primary or
   * through conditions; then whether that share is held or still sending.
   */
  roleCarries:
    '{role, select, split {{stream} {share}} primary {{stream} primary} other {{stream} · conditions}}{state, select, held { · held} sending { · still sending} other {}}',
  ipsDedicated: '{ips} · dedicated',
  ipsShared: 'Shared pool · checked at domain level',
  /** {kind} is how events arrive ("SNS"); {ago} how long ago: "20 s ago". */
  webhookLine: '{hasKind, select, yes {{kind} · } other {}}{hasEvent, select, yes {last event {ago}} other {no events yet}}',
  factIps: 'IPs',
  factWebhooks: 'Webhooks',
  factErrorRate7d: 'API error rate · 7 d',
  factErrorRate3m: 'API error rate · 3 min',
  factServer: 'Server',
  factAccountStatus: 'Account status',
  factLastSuccess: 'Last successful send',
  noneRecorded: 'none recorded',
  /** {webhook} is the webhook line ("SNS · last event 2 min ago"). */
  webhookBefore: '{webhook} (deliveries from before {time})',
  warmupPlan: 'Warm-up plan',
  domains: 'Domains',
  events: 'Events',
  settings: 'Settings',
  settingsLabel: '{name} settings',
  capabilities: 'Capabilities',
  recentErrors: 'Recent errors',
  noErrorDetails: 'No error details recorded.',

  // Quota and rate meters. {used}, {sent}, {max} and {cap} are counts;
  // {share} a percentage; {rate} and {peak} messages per second.
  warmupCap: "Today's warm-up cap · {cap}",
  warmupSentToday: '{sent} sent · {weekday}',
  warmupSentShare: '{sent} sent · {share}',
  acceptedUnknown: '{count} accepted · status unknown',
  quota24h: '24 h quota · {max}',
  quotaPlan: 'Plan quota · {max} per month',
  quotaUsed: '{used} used · {share}',
  quotaUsedFailover: '{used} used · {share} · +{failover} from failover',
  rateRamp: 'Rate · ramp limit {rate} /s',
  rateMax: 'Max send rate · {rate} /s',
  rateNone: 'Send rate · no published limit',
  rateThrottled: '{rate} /s · throttled',
  rateOnly: '{rate} /s',
  rateNow: '{rate} /s now',
  rateNowPeak: '{rate} /s now · peak today {peak} /s',
  /** A rate that stopped updating: "0.6% · stale". */
  staleValue: '{value} · stale',

  // Cost.
  costToDate: 'Cost to date · {month}',
  costPlan: '{amount} plan',
  costMetered: '{amount} metered',
  per1000Usage: 'Per {thousand} · at current usage',
  per1000Server: 'Per {thousand} · you entered {amount} per month server',
  per1000SelfHosted: 'Per {thousand} · self-hosted',
  per1000From: 'Per {thousand} · from {source}',
  per1000FromPricing: 'Per {thousand} · from the provider’s pricing',

  // During an outage, on the other providers.
  takingFailover: 'Taking {streams} failover',
  forecastTitle: 'Quota forecast while {provider} is down',
  /** {perDay} and {adds} are counts, {price} money, {date} a day ("24 Oct"). */
  forecastBody:
    '{stream} send about {perDay} per day. {runsOut, select, yes {At this rate the {provider} plan runs out on {date}} other {The {provider} plan has room for now}}{nextTier, select, yes {; the next plan tier adds {adds} for {price}.} other {.}}',
  whyNotTitle: 'Why not move the Broadcast to {provider}',
  /**
   * {held} is a count; {times} "3" (as in 3×) or {share} a percentage of the
   * cap; {days} the IP's age in days and {rate} messages per second.
   */
  whyNotBody:
    "{held} held recipients would be {over, select, yes {{times}× today's cap} other {{share} of today's cap}}{push, select, yes { and would push a {days}-day-old IP to {rate} /s} other {}}. The warm-up guard would pause it within minutes. Failover to a warmed-up provider only is the safe setting.",

  // While its webhooks are silent.
  warmingUpDayPaused: 'Warming up · day {day} of {days}, paused',
  silentLastEvent: 'last event {time} · {since} ago',
  freshnessTitle: 'Webhook freshness, last 24 h',
  freshnessLabel: '{count, plural, one {Events per hour for the last 24 hours; none in the last # hour} other {Events per hour for the last 24 hours; none in the last # hours}}',
  freshnessYesterday: '{time} yesterday',
  freshnessNow: 'now',
  /** A healthy provider's line while another is degraded. */
  compactLine: '{hasKind, select, yes {{kind} · } other {}}{hasEvent, select, yes {last event {ago}} other {no events yet}} · error rate {rate}',
  compactRates:
    'Bounce {bounce} of {bounceLimit} · complaint {complaint} of {complaintLimit}{quota, select, rolling { · quota {used} of {max}} plan { · plan {used} of {max}} other {}}',
  takingHeld: 'Taking the held {provider} share',
  takingHeldNo:
    '{policy, select, warm_only {Taking the held {provider} share? No: Broadcasts failover is warmed-up only} other {Taking the held {provider} share? No: Broadcasts failover is pause-and-ask}}',

  // ── Settings dialog ──────────────────────────────────────────────────
  dialogTitle: '{name} settings',
  dialogDescription: 'connected {date}',
  dialogDescriptionLabel: '{label} · connected {date}',
  removeProvider: 'Remove provider',
  label: 'Label',
  labelHint: 'Shown beside the name, like the region or server.',
  credentials: 'Credentials',
  credentialsLine: 'Stored encrypted and never shown again. New values are checked with {name} before they replace the old ones.',
  credentialSet: 'Set · write-only',
  credentialNotSet: 'Not set',
  replace: 'Replace',
  /** {credential} is the field's name ("API key"). */
  newCredential: 'New {credential}',
  newCredentialPlaceholder: 'New {credential}',
  checkAndSave: 'Check and save credentials',
  keepCurrent: 'Keep current',
  pricing: 'Pricing',
  howYouPay: 'How you pay',
  perEmail: 'Per email sent',
  monthlyPlan: 'Monthly plan',
  selfHostedServer: 'Self-hosted server',
  pricePer1000: 'Price per {thousand} emails',
  planPrice: 'Plan price per month',
  serverCost: 'Server cost per month',
  limitsTitle: 'Limits shown on the card',
  bounceLimit: 'Bounce rate limit (%)',
  complaintLimit: 'Complaint rate limit (%)',
  settingsSaved: '{name} settings saved',
  credentialsAccepted: '{name} accepted the new credentials',
  credentialsAcceptedDetail: 'They are stored encrypted and never shown again.',
  errBounce: 'Enter a bounce limit between 0 and 100%.',
  errComplaint: 'Enter a complaint limit between 0 and 100%.',
  /** {example} is a price in the region format ("0.10"). */
  errPer1000: 'Enter the price per {thousand} emails, like {example}.',
  errMonthly: 'Enter the price per month, like 15.',
  errLabel: 'Give the provider a label, like the region or server name.',
  errCredential: 'Enter the new value for the credential you are replacing.',

  // ── Remove provider ──────────────────────────────────────────────────
  removeTitle: 'Remove {name} from this workspace?',
  removeChecking: 'Checking which routes use {name}…',
  removeNoUses: 'No route uses {name}, so nothing else changes.',
  /** {roles} lists what it is: "the primary route for Broadcasts and the failover for System". */
  removeUses:
    '{name} is {roles}. {orphans, select, yes {Removing it leaves those routes with no provider until you pick another.} other {Its share goes back to the primary.}}',
  rolePrimaryRoute: 'the primary route for {stream, select, broadcast {Broadcasts} automation {Automations} system {System mail} other {Transactional mail}}',
  /** {share} is a percentage. */
  roleShareOf: '{share} of {stream}',
  roleFailoverFor: 'the failover for {stream}',
  roleCondition: 'a {stream} condition',
  reassignAndRemove: 'Reassign routes and remove',
  removing: 'Removing…',
  stillChecking: 'Still checking which routes use {name}. Try again in a moment.',
  connectAnother: 'Connect another provider that can carry {streams} first.',
  replacementProvider: 'Replacement provider',
  replacementFailover: 'Replacement failover',
  nonePauseAsk: 'None, pause and ask',
  rowPrimary: 'Primary: {name} → choose a replacement',
  rowPrimaryNone: 'Primary: {name} → no other provider can carry {stream}',
  rowFailover: 'Failover: {name} → choose a replacement',
  /** {share} is a percentage; {target} the provider that gets it, or "the primary". */
  rowSplit: 'Split: {name} {share} → goes to {target}',
  rowCondition: 'Condition: {values} → {name} only, removed',
  thePrimary: 'the primary',
  keeps: 'Keeps',
  keepsBody: 'The {count} suppressions in the shared list stay, and so do its past events and reports.',
  keepsWebhooks: 'Webhooks are unregistered for you.',
  removedTitle: '{name} removed',
  removedRoutes: 'Its routes now use the providers you picked; its credentials were deleted.',
  removedCredentials: 'Its credentials were deleted.',
  removedWebhook: 'Its webhook was removed from {name}.',
  removedWebhookManual: 'Remove the webhook pointing at {url} in {name}.',
  removedWebhookFailed: "{name} didn't remove its webhook ({detail}); remove the one pointing at {url} there.",
})
