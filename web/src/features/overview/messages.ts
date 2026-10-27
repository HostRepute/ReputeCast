import { defineMessages } from '@/i18n'

/**
 * 7.1 Overview (Main, OverviewEmpty and OverviewStates boards): sender
 * health, the next send, the provider cards, growth, tiers, recent campaigns
 * and the first-run steps. Numbers, dates and times arrive formatted.
 */
export const m = defineMessages('overview', {
  // ── Page ─────────────────────────────────────────────────────────────
  title: 'Overview',
  subtitleNoDomain: '{day} · no sending domain yet',
  newCampaign: 'New campaign',
  retryNow: 'Retry now',
  viewEventLog: 'View event log',
  restCurrent: 'The rest of the Overview is current.',
  loadFailed:
    "{card, select, health {Couldn't load sender health} next {Couldn't load next scheduled campaign} providers {Couldn't load sending providers} growth {Couldn't load subscriber growth} tiers {Couldn't load engagement tiers} recent {Couldn't load recent campaigns} other {Couldn't load this card}}",
  checking: 'Checking blocklists and provider APIs… usually under 5 seconds.',
  manageProviders: 'Manage providers & routing',

  // ── Card titles ──────────────────────────────────────────────────────
  senderHealth: 'Sender health',
  nextCampaign: 'Next scheduled campaign',
  sendingProviders: 'Sending providers',
  subscriberGrowth: 'Subscriber growth',
  engagementTiers: 'Engagement tiers',
  recentCampaigns: 'Recent campaigns',

  // ── Sender health ────────────────────────────────────────────────────
  deliverabilityDetails: 'Deliverability details',
  scoreLabel: 'Sender health score {score} out of 100, {verdict}',
  /** The score's verdict, one of the API's labels (OverviewSenderHealth.label). */
  verdict: '{label, select, Good {Good} Fair {Fair} Poor {Poor} other {{label}}}',
  reasonGood: 'Good:',
  reasonProblem: 'Problem:',
  reasonWarning: 'Warning:',
  healthPending: 'Appears once a sending domain is verified. Blocklists, authentication and provider limits are checked every hour.',

  // ── Next scheduled campaign ──────────────────────────────────────────
  nothingScheduled: 'Nothing is scheduled.',
  draftsWaiting:
    '{n, plural, one {{count} draft is waiting; schedule one and its readiness, route and cost show here.} other {{count} drafts are waiting; schedule one and its readiness, route and cost show here.}}',
  scheduleOne: 'Schedule a campaign and its readiness, route and cost show here.',
  openDrafts: 'Open drafts',
  recipients: 'Recipients',
  route: 'Route',
  estCost: 'Est. cost',
  estDuration: 'Est. duration',
  issueBlock: 'Block',
  issueWarn: 'Warn',
  openReadiness: 'Open Send Readiness',
  viewCampaign: 'View campaign',
  editCampaign: 'Edit campaign',
  nextPending: 'No campaigns yet. Write one now; scheduling and sending unlock when a provider is assigned to the Broadcasts stream.',
  notChecked: 'Not checked yet',
  blockedBy: '{n, plural, one {Blocked by # check} other {Blocked by # checks}}',
  blocked: 'Blocked',
  readyWith: '{n, plural, one {Ready with # warning} other {Ready with # warnings}}',
  readyWarnings: 'Ready with warnings',
  ready: 'Ready',

  // ── Provider cards ───────────────────────────────────────────────────
  stale: 'Stale',
  staleSince: 'Stale · {time}',
  bounceLimit: 'Bounce rate · limit {limit}',
  bounceLimitReview: 'Bounce rate · limit {limit} ({provider} review)',
  bounceWarmup: 'Bounce rate · warm-up limit {limit}',
  complaintLimit: 'Complaint rate · limit {limit}',
  complaintWarmup: 'Complaint rate · warm-up limit {limit}',
  quota24h: '24 h quota · {max}',
  quotaPlan: 'Plan · {max} / month',
  quotaWarmup: "Today's warm-up cap · {max}",
  quotaUsed: '{used} used ({percent})',
  quotaSent: '{sent} sent ({weekday})',
  maxRate: 'Max rate',
  rampRate: 'Rate (ramp)',
  noLimit: 'No limit set',
  lastWebhook: 'Last webhook',
  costMonth: 'Cost · {month}',
  costPlan: '{cost} plan',
  costMetered: '{cost} metered',
  noEvents: 'no events yet',
  justNow: 'just now',
  eventsVia:
    '{webhook, select, yes {{unaffected, select, yes {Events via webhook: last received {ago}, so delivery tracking is unaffected.} other {Events via webhook: last received {ago}.}}} other {{unaffected, select, yes {Events via {feed}: last received {ago}, so delivery tracking is unaffected.} other {Events via {feed}: last received {ago}.}}}}',
  noFeed: '{unaffected, select, yes {No event feed, so delivery tracking is unaffected.} other {No event feed.}}',
  /** The event feed of SMTP providers; SNS and Event Grid are product names. */
  feedBounceMailbox: 'Bounce mailbox',

  // ── A failing account-health refresh (OverviewStates) ────────────────
  refreshTitle: "Couldn't refresh {name} account health",
  refreshReturned:
    '{known, select, yes {{provider} {op} returned {what} {n, plural, =1 {once} =2 {twice} =3 {three times} =4 {four times} =5 {five times} =6 {six times} =7 {seven times} =8 {eight times} =9 {nine times} other {# times}}} other {{provider} {op} returned an error {n, plural, =1 {once} =2 {twice} =3 {three times} =4 {four times} =5 {five times} =6 {six times} =7 {seven times} =8 {eight times} =9 {nine times} other {# times}}}}{shown, select, yes { ({times}).} other {.}}',
  refreshLastGood: 'Bounce and complaint rates below are from the last successful fetch at {time}.',
  refreshContinue: 'Routing and sending continue.',
  refreshContinueWebhook: 'Routing and sending continue; {provider} events still arrive through its webhook.',
  refreshContinueFeed: 'Routing and sending continue; {provider} events still arrive through {feed}.',

  // ── Subscriber growth and engagement tiers ───────────────────────────
  lastDays: 'Last {days} days · {link}',
  listsUnique: '{lists, plural, one {# list} other {# lists}} · {n, plural, one {{total} unique contact} other {{total} unique contacts}}',
  noListsUnique: 'No lists yet · {n, plural, one {{total} unique contact} other {{total} unique contacts}}',
  sparkRising: 'Subscribers over the last {days} days, rising from {from} to {to}',
  sparkFalling: 'Subscribers over the last {days} days, falling from {from} to {to}',
  net: '{net} net',
  netPercent: '{net} net · {percent}',
  tiersLabel: 'Engagement tiers: {tiers}',
  tierShare: '{tier} {percent}',
  listHealthLink: 'List health & sunset policy',
  noContacts: 'No contacts yet',
  sunsetPolicy: 'Sunset policy',

  // ── Recent campaigns ─────────────────────────────────────────────────
  allCampaigns: 'All campaigns',
  colCampaign: 'Campaign',
  colSent: 'Sent',
  colRecipients: 'Recipients',
  colDelivered: 'Delivered',
  colHumanOpens: 'Human opens',
  colClicks: 'Clicks',
  colProviders: 'Sending providers',
  colStatus: 'Status',
  importedFrom: 'imported from {origin}',
  imported: '{origin} · imported',
  sent: 'Sent',
  recentEmpty: 'Campaigns appear here once they go out, with their delivery, human opens and clicks.',

  // ── Search ───────────────────────────────────────────────────────────
  searchPlaceholder: 'Search campaigns, contacts, events',
  searchResults: 'Search results',
  groupCampaigns: 'Campaigns',
  groupEvents: 'Events',
  scheduledOn: 'scheduled {date}',
  campaignStatus:
    '{status, select, draft {draft} approval {approval} approved {approved} scheduled {scheduled} sending {sending} paused {paused} sent {sent} cancelled {cancelled} failed {failed} other {{status}}}',
  eventType:
    '{type, select, delivered {delivered} deferred {deferred} hard_bounce {hard bounce} soft_bounce {soft bounce} complaint {complaint} rejected {rejected} other {{label}}}',
  searching: 'Searching…',
  searchNothing: 'Nothing matches “{term}”. Campaign names, and recipient addresses in provider events, are searched.',

  // ── First run (OverviewEmpty) ────────────────────────────────────────
  step: '{current, select, yes {Step {n}, current:} other {Step {n}:}}',
  selfHosted: 'Self-hosted · {version} · all features included',
  welcome: 'Welcome to {name}',
  welcomeLead:
    "ReputeCast sends through a sending provider you control. Connect one now, and this page fills up with sender health, provider limits and your first campaign's readiness.",
  stepConnect: 'Connect a sending provider',
  stepConnectLine: 'Amazon SES, Postmark, Mailgun, SendGrid, Resend, Brevo, your own Postal server or any SMTP server. Most take 10–15 minutes.',
  /** PR-13: campaigns can be written before a provider is connected. */
  writeFirst: 'Write your first campaign now; sending unlocks once a provider is connected.',
  /** PR-13: the first-run screen loads the demo into its own workspace. */
  exploreDemo: 'Explore with demo data',
  exploreDemoLine: 'Loads Acme sample data into a separate demo workspace. Nothing is sent.',
  exploreDemoBusy: 'Loading the demo…',
  exploreDemoFailed: "Couldn't load the demo workspace",
  chooseProvider: 'Choose a provider',
  browseAdapters: 'Browse community adapters',
  stepVerify: 'Verify your sending domain',
  stepVerifyLine: 'DKIM, return-path, SPF and DMARC records, verified live. Unlocked after step 1.',
  stepImport: 'Import contacts',
  importNow: 'Import now',
  stepImportLine: 'From Brevo, Mailchimp, MailerLite, listmonk, Keila or CSV. Unsubscribed and bounced contacts stay suppressed.',
  subscribers: 'Subscribers',
  subscribersNoLists: 'Create a list to collect contacts, or import them with their lists.',
  subscribersWaiting:
    '{n, plural, one {On 1 list: {lists}. Nothing is sent to them until a sending provider is connected.} other {Across # lists: {lists}. Nothing is sent to them until a sending provider is connected.}}',
  listsReady: '{n, plural, one {# list is ready to receive contacts: {lists}.} other {# lists are ready to receive contacts: {lists}.}}',

  // ── Loading (OverviewStates) ─────────────────────────────────────────
  loadingHealth: 'Loading sender health',
  loadingNext: 'Loading the next campaign',
  loadingProviders: 'Loading sending providers',
  loading: 'Loading',
})
