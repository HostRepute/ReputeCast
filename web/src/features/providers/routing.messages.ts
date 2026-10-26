import { defineMessages } from '@/i18n'

/**
 * Sending providers › Routing & failover (ProvidersRouting): the stream
 * cards, the condition dialog, the shared settings, the route tester and the
 * tester's explanations (routing-model).
 */
export const mr = defineMessages('providers.routing', {
  loadFailed: "Couldn't load the routing",
  loadFailedBody: 'Mail keeps going out on the saved routing.',
  loading: 'Loading the routing',
  saved: 'Routing saved',
  savedDetail: 'New sends use it right away; sends in progress keep their route.',
  saveFailed: "Routing wasn't saved",
  unsaved: '{changes, plural, one {Unsaved change} other {Unsaved changes}}',
  unsavedWarnings: '{changes, plural, one {Unsaved change} other {Unsaved changes}} · {warnings, plural, one {# warning} other {# warnings}}',
  /** {ago} is a relative time ("2 h ago"). */
  savedAgo: 'Saved {ago}',
  savedAgoBy: 'Saved {ago} by {name}',
  discard: 'Discard',
  upToDate: 'Routing is up to date',
  upToDateDetail: 'Change a route, a split or the failover policy first.',
  saveRouting: 'Save routing',
  asideLabel: 'Route tester and shared settings',
  /** Leaving the routing screen with unsaved edits. {streams} is a list of stream names ("Broadcasts and System"). */
  leaveTitle: 'Leave without saving the routing?',
  leaveBody: 'You changed {streams}. If you leave now, {count, plural, one {that change is} other {those changes are}} lost and mail keeps the saved routing.',
  saveAndLeave: 'Save routing and leave',
  /** A save refused because the routing changed meanwhile. {streams} is a list of stream names. */
  conflictTitle: '{streams} changed while you were editing',
  conflictBody: 'Someone saved the routing, or the warm-up planner raised a share. Nothing was saved, and your edits are still here.',
  conflictShowLatest: 'Show the latest routing',
  conflictKeepMine: 'Keep my edits and replace it',

  // ── A stream's card ──────────────────────────────────────────────────
  tagBroadcast: 'campaigns',
  tagAutomation: 'emails sent by flows',
  tagSystem: 'opt-in confirmations, preference links, invites',
  tagTransactional: 'password resets, receipts and alerts from your product',
  /** {product} is the product's name from the transactional settings ("Acme Cloud"). */
  tagTransactionalNamed: 'password resets, receipts and alerts from {product}',
  volume: '{per, select, week {about {count} per week} other {about {count} per day}}',
  noMailYet: 'no mail yet',
  addConditionBroadcast: 'Add condition · list, sender domain or mailbox provider',
  addCondition: 'Add condition',
  primary: 'Primary',
  chooseProvider: 'Choose a provider',
  fallback: 'Fallback',
  noneAsk: 'None · pause and ask',
  none: 'None',
  failoverBadge: '{policy, select, pause_and_ask {Failover: pause and ask} warm_only {Failover: warmed-up only} other {Failover: automatic}}',
  systemNote: 'System mail must always go out; it never waits for a decision',
  automationNote: 'switches after 3 consecutive failed sends or 2 failed health checks; returns to the primary after 10 healthy minutes',

  // Transactional.
  firstInLine: 'First in line on shared providers',
  firstInLineShared: 'When {provider} also sends a campaign, transactional messages go first',
  /** {rate} is messages per second ("14"). */
  firstInLineSharedRate: 'When {provider} also sends a campaign, transactional messages go first within its {rate} per second',
  firstInLineAny: 'When a provider also sends a campaign, transactional messages go first within its rate',
  txConditions:
    "{count, plural, =0 {Conditions by category or mailbox provider: none · bulk-marketing terms don't apply to this stream} other {Conditions by category or mailbox provider: # · bulk-marketing terms don't apply to this stream}}",
  templatesAndLog: 'Templates and log',

  // Weighted split.
  weightedSplit: 'Weighted split',
  splitUse: 'used for migration and warm-up',
  splitUseWarming: 'used for migration and warm-up · adjusts daily with the {provider} plan',
  /** {split} lists each provider with its share ("SES 90%, Postal 10%"). */
  splitLabel: 'Weighted split: {split}',
  unknown: 'Unknown',
  /** {steps} lists the planned shares ("15% on day 12, 25% on day 15"). */
  plannerRaises: 'The warm-up planner raises this automatically: {steps}.',
  plannerStep: '{share} on day {day}',
  shareLabel: '{provider} share',
  sharePercent: '{provider} percentage',

  // Conditions.
  conditionsBeforeSplit: 'Conditions · evaluated before the split',
  conditionsBeforePrimary: 'Conditions · evaluated before the primary',
  condIf: 'IF',
  condThen: 'THEN',
  condIsOneOf: 'is one of',
  condIs: 'is',
  condOnly: '{provider} only',
  /** One provider of a condition's split ("Amazon SES 70%"); {share} is a percentage. */
  condShare: '{provider} {share}',
  /** After a split condition's note: its split keeps each address on one provider too. */
  condSticky: 'sticky too',
  condUnknownProvider: 'Unknown provider',
  /** {field} is the condition's field ("recipient mailbox provider") and {values} its values. */
  removeCondition: 'Remove condition: {field} {values}',
  remove: 'Remove',
  fieldMailboxChip: 'recipient mailbox provider',
  fieldListChip: 'list',
  fieldDomainChip: 'sender domain',
  fieldCategoryChip: 'category',
  mailboxOther: 'Other',

  // Broadcasts failover policy.
  failoverPolicy: 'Failover policy',
  pauseAndAsk: '<b>Pause and ask</b> <k>(default for Broadcasts)</k>',
  pauseDetail: 'If the primary fails mid-send, hold the remaining recipients and notify you. Moving a bulk send to a cold provider can hurt reputation.',
  warmOnly: 'Fail over automatically, only to a warmed-up provider',
  warmOnlyDetail: '{provider} qualifies on day {day} of its warm-up. Until then this behaves like pause and ask.',
  warmOnlyNone: 'Providers still warming up are skipped; with none left, this behaves like pause and ask.',
  automatic: 'Fail over automatically to any fallback',

  // Shared across providers.
  sharedTitle: 'Shared across providers',
  sharedSuppression: 'Suppression list',
  /** {count} is the number of addresses, formatted; {time} when it synced. */
  sharedSuppressionSynced: '{count} · synced {time}',
  sharedThrottles: 'Per-mailbox-provider throttles',
  sharedThrottlesValue: 'apply on every route',
  sharedThresholds: 'Auto-pause thresholds',
  sharedThresholdsValue: 'per campaign',
  openEvents: 'Open events & sync',

  // Condition dialog.
  /** {stream} is the stream's name ("Broadcasts"). */
  conditionTitle: 'Add a {stream} condition',
  conditionBody: 'Conditions are checked in order before the split; the first match decides the provider.',
  pickValue: 'Pick at least one value.',
  labelIf: 'If',
  fieldCategory: 'Category',
  fieldMailbox: 'Recipient mailbox provider',
  fieldList: 'List',
  fieldDomain: 'Sender domain',
  legendIsOneOf: 'is one of',
  nothingToPick: 'Nothing to pick yet.',
  thenSend: 'Then send only through',
  thenLegend: 'Then send through',
  thenOne: 'One provider',
  thenSplit: 'A split by weight',
  thenSplitHint: 'Each address keeps its provider, as in the stream\'s split.',
  /** {provider} is a provider's name. */
  splitShareLabel: "{provider}'s share",
  splitProviderLabel: 'Provider {n}',
  splitAddProvider: 'Add a provider',
  splitRemoveProvider: 'Remove {provider} from the split',
  /** {total} is a percentage ("90%"). */
  splitTotal: 'Adds up to {total}',
  splitMust100: 'The split must add up to 100%.',
  splitTwoProviders: 'Pick two different providers for the split.',
  why: 'Why',
  whyHint: 'Shown beside the condition, like “Outlook is strict with new IPs”.',
  notBulk: "This provider's terms don't allow bulk marketing on Broadcasts.",

  // Provider choices (routing-model).
  optionWarming: '{name} · warming up',

  // ── Route tester ─────────────────────────────────────────────────────
  testerTitle: 'Route tester',
  testerLine: 'Which provider would send this email?',
  testerStream: 'Stream',
  testerTemplate: 'Template',
  noTemplates: 'No templates yet',
  testerList: 'List',
  noLists: 'No lists yet',
  testerDomain: 'Sender domain',
  noDomains: 'No sending domain yet',
  testerRecipient: 'Recipient',
  enterAddress: 'Enter an address, like name@example.com, to see which provider would send to it.',
  working: 'Working out the route…',
  mailboxProvider: 'Mailbox provider',
  dkimSelector: 'DKIM selector',
  notSetUp: 'not set up yet',
  cost: 'Cost',
  ifItFails: 'If it fails:',

  // The tester's steps (routing-model).
  stepStreamRules: 'Stream is {stream}, so the {stream} rule set applies.',
  stepStreamPrimary: 'Stream is {stream}: primary {provider}, no conditions.',
  stepStreamNoPrimary: 'Stream is {stream}: primary not set, no conditions.',
  stepCondition:
    '{field, select, list {Condition matched: list is {value}, so {provider} only.} sender_domain {Condition matched: sender domain is {value}, so {provider} only.} category {Condition matched: category is {value}, so {provider} only.} other {Condition matched: recipient mailbox provider is {value}, so {provider} only.}}',
  stepStillWarming: '{provider} is still warming up.',
  /** {split} is "SES 70 / Postal 30". */
  stepConditionSplit:
    '{field, select, list {Condition matched: list is {value}; its {split} split sends this address to {provider}.} sender_domain {Condition matched: sender domain is {value}; its {split} split sends this address to {provider}.} category {Condition matched: category is {value}; its {split} split sends this address to {provider}.} other {Condition matched: recipient mailbox provider is {value}; its {split} split sends this address to {provider}.}}',
  stepNoMatch: 'No condition matched.',
  /** {split} is the split ("SES 90 / Postal 10"). */
  stepSplit: 'Weighted split {split}: this address hashes to bucket {bucket} of 100.',
  stepPrimary: 'Primary {provider}.',
  stepSuppressed: 'Suppression list checked: {email} is suppressed, so nothing is sent to it.',
  stepNotSuppressed: 'Suppression list checked: {email} is not suppressed.',
  stepSilent: "{provider}'s webhooks are silent, so its share waits until events resume.",
  stepDown: '{provider} is down right now, so this email would wait for it or its fallback.',

  // What happens if the provider fails (after "If it fails:").
  fallbackPauseBroadcast: 'pause and ask. Broadcast failover to a cold provider is never automatic.',
  fallbackPause: 'pause and ask: the send holds until you decide.',
  fallbackNone: 'no fallback is set, so the send holds until the provider recovers.',
  fallbackWarmOnly: 'automatic failover to {provider} once it is warmed up; until then, pause and ask.',
  fallbackAutomation: 'automatic failover to {provider} after 3 failed sends or 2 failed health checks.',
  fallbackAutomatic: 'automatic failover to {provider}.',

  // The cost of one message. Amounts are formatted ("$0.0001").
  costSelfHosted: '{amount} metered · self-hosted',
  costPlanAbout: 'plan · about {amount}',
  costPlan: 'plan',
  /** {thousand} is 1,000 in the region format. */
  costMetered: '{each} · {per1000} per {thousand}',
})
