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
  savedAgo: 'Last saved {ago}',
  savedAgoBy: 'Last saved {ago} by {name}',
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
  systemNote: 'System mail must always go out; it never waits for a decision. Same trigger as Automations.',
  automationNote: 'switches after 3 consecutive failed sends or 2 failed health checks; returns to the primary after 10 healthy minutes',

  // Transactional.
  firstInLine: 'First in line on shared providers',
  firstInLineShared: 'When {provider} also sends a campaign, transactional messages go first',
  /** {rate} is messages per second ("14"). */
  firstInLineSharedRate: 'When {provider} also sends a campaign, transactional messages go first within its {rate} per second',
  firstInLineAny: 'When a provider also sends a campaign, transactional messages go first within its rate',
  /** {auto} is no when the stream pauses and asks instead of failing over. */
  txConditions:
    "{auto, select, no {Pauses and asks on the same trigger as Automations} other {Same trigger as Automations}} · {count, plural, =0 {no conditions by category or mailbox provider} one {# condition by category or mailbox provider} other {# conditions by category or mailbox provider}} · bulk-marketing terms don't apply to this stream",
  templatesAndLog: 'Templates and log',

  // Weighted split.
  weightedSplit: 'Weighted split',
  splitUse: 'used for migration and warm-up',
  splitUseWarming: 'used for migration and warm-up · adjusts daily with the {provider} plan',
  unknown: 'Unknown',
  /** {steps} lists the planned shares ("15% on day 12, 25% on day 15"). */
  plannerRaises: 'The warm-up planner raises this automatically: {steps}.',
  plannerStep: '{share} on day {day}',
  shareLabel: '{provider} share',
  sharePercent: '{provider} percentage',

  // Conditions.
  conditionsBeforeSplit: 'Conditions · checked before the split',
  conditionsBeforePrimary: 'Conditions · checked before the primary',
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
  automatic: 'Fail over automatically to any fallback that passes its checks',

  // Shared across providers.
  sharedTitle: 'Shared across providers',
  sharedSuppression: 'Suppression list',
  /** {count} is the number of addresses, formatted; {time} when it synced. */
  sharedSuppressionSynced: '{count} · {pushed} pushed {time}',
  /** {pushed}: hard bounces and complaints that apply to all mail, which providers get. */
  sharedSuppressionPushed: '{count} · {pushed} pushed',
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

  // ── The ordered split, the unsaved bar and failover checks (PR-14) ────
  stickyLine: 'Sticky: each recipient keeps the same provider from send to send',
  /** Announced after a keyboard move; {position} of {count}. */
  movedTo: '{provider} moved to position {position} of {count}. Nobody changes provider.',
  moveUp: 'Move {provider} up',
  moveDown: 'Move {provider} down',
  removeFromSplit: 'Remove {provider} from the split',
  dragHint: 'Drag a row or use its arrows to reorder; the order only changes how the bar reads. Changing a weight moves only the difference, and nobody else changes provider.',
  totalOk: 'Adds up to 100%',
  /** {total} is a number of percent. */
  totalBad: 'Adds up to {total}% · needs 100%',
  addToSplit: 'Add a provider to the split',
  allInSplit: 'Every connected provider is already in the split.',
  weightLabel: "{provider}'s share of {stream}, in percent",
  plannerRaisesProvider: 'The warm-up planner raises {provider} on its own: {steps}.',
  unsavedTitle: 'Unsaved changes',
  unsavedBadge: 'Unsaved change',
  weightsNot100: 'Weights add up to {total}%. Make them 100% to save.',
  /** {order} lists short provider names ("SES, Postal"). */
  orderChanged: 'Broadcasts order changed to {order} · nobody changes provider',
  /** {before} and {after} are splits like "90/10". */
  splitChanged: 'Broadcasts split {before} → {after}',
  onlyDifferenceMoves: 'only the difference moves; nobody else changes provider',
  nobodyMoves: 'nobody changes provider',
  /** {about} is "about 1,600"; {from} and {to} are short provider names. */
  recipientsMove: '{about} Broadcast recipients move from {from} to {to}; nobody else changes provider',
  streamsChanged: '{count, plural, one {# stream changed} other {# streams changed}}: {streams}',
  fewerThan100: 'fewer than 100',
  aboutN: 'about {n}',
  failWhen: 'Fail over when:',
  /** The trigger the outage check uses (manage/health.go). */
  failWhenRule: 'at least half of the calls to the provider fail over 3 minutes (5 calls or more), or 2 health checks in a row fail.',
  returnsAfter: 'Returns to the primary after {minutes} healthy minutes.',
  /** Automations: the trigger beside the failover badge (the outage check's rule, manage/health.go). */
  failsOverWhenInline: 'Fails over when at least half of the calls to the provider fail over 3 minutes (5 calls or more), or 2 health checks in a row fail. Returns to the primary after {minutes} healthy minutes.',
  pausesWhenInline: 'Pauses and asks when at least half of the calls to the provider fail over 3 minutes (5 calls or more), or 2 health checks in a row fail.',
  silentNoFailover: "Silent webhooks alone raise an alarm, not a failover: ReputeCast asks the provider's API about messages before counting them as lost.",
  whenTriggerFires: 'What happens when the trigger fires',
  /** {provider} is the provider whose checks follow. */
  checkedBeforeAny: 'Checked before any failover · {provider}',
  checkedBeforeTakes: 'Checked before {provider} takes over',
  /** {domain} is the stream's sending domain (notify.acme.io). */
  checkedBeforeTakesOn: 'Checked before {provider} takes over · {domain}',
  checksFailBroadcast: 'If any check fails, Broadcasts pause and ask instead.',
  /** {provider} is the stream's primary. */
  checksFailOther: 'If a check fails, ReputeCast keeps retrying {provider} and alerts you instead of switching.',
  /** A failover target's checks as chips. */
  checkDomain: '{result, select, pass {sending domain verified} unknown {sending domain not checked} other {sending domain not verified}}',
  /** {left} is what the quota (shape left, per {period}) or today's warm-up cap (shape today) has left, formatted. */
  checkQuota:
    '{result, select, pass {{shape, select, left {quota free · {left} left{period, select, day { today} month { this month} other {}}} today {quota free · {left} today} other {no quota limit}}} unknown {quota not read yet} other {quota too small · {left} left}}',
  checkWarm: '{result, select, pass {warm} unknown {warming up} other {{hasDay, select, yes {not warm yet · day {day} of {days}} other {not warm yet}}}}',
  checkSandbox: '{result, select, pass {not in sandbox} other {in the sandbox}}',
  checkAvailable: '{result, select, pass {available} other {not available now}}',
  checkBulk: "{result, select, pass {terms allow Broadcasts} other {terms don't allow Broadcasts}}",
  noTarget: 'No fallback to check.',
  mxNote: 'Gmail includes Google Workspace and Outlook includes Microsoft 365 recipients (found from MX records). Strict mailbox providers get their own lower caps while a provider warms up.',

  // The route tester's tabs.
  testerSaved: 'Which provider would send this? Tests the saved routing.',
  tabRecipient: 'Recipient',
  tabCampaign: 'Campaign',
  testerTabs: 'Test by',
  /** The MX host the mailbox provider was found from. */
  mxLine: 'MX {host}',
  mxOther: 'MX is not Gmail, Outlook, Yahoo or iCloud',
  mxUnknown: 'MX not found',
  /** {pair} is the unsaved split ("85/15"). */
  draftLine: 'With your unsaved {pair} split:',
  draftStays: 'this address stays on {provider}.',
  draftMoves: 'this address moves to {provider}.',
  draftSame: 'the route is the same.',

  // The Campaign tab.
  campaignLoading: 'Working out where the campaign goes…',
  campaignNone: 'No broadcast to simulate yet. Write or schedule a campaign and it shows here.',
  campaignTitle: 'Campaign',
  /** {total} recipients, formatted. */
  campaignRecipients: '{total} recipients',
  /** {lists} names the lists; {sunset} is how many the sunset policy excludes. */
  campaignAudience: '{lists} · {domain}{hasSunset, select, yes { · {sunset} unengaged excluded by the sunset policy} other {}}',
  /** A large audience: {simulated} recipients went through the routing and the counts were scaled to the whole audience. */
  campaignEstimated: 'Estimated: the first {simulated} recipients went through the routing, and the counts are scaled to the whole audience.',
  conditionsMatched: 'Conditions matched',
  conditionsNone: 'none',
  /** {count} recipients a condition matched. */
  conditionMatchedLine: '{field} is {values} · {count}',
  overflowTo: 'Warm-up overflow to {provider}',
  /** {provider}'s cap on {date} (day {day}) is {cap}. */
  overflowCap: "{provider}'s cap on {date} (day {day}) is {cap}, split by mailbox provider.",
  ifProviderFails: 'If {provider} fails',
  policyText: '{policy, select, pause_and_ask {pause and ask} warm_only {fail over to a warmed-up provider} other {{hasFallback, select, yes {fail over to {fallback}} other {fail over}}}}',
  /** {shares} lists the unsaved split's counts ("Amazon SES 26,513 · Postal 4,679"). */
  campaignDraft: '{shares}. {moved} recipients move; everyone else keeps their provider.',
  discarded: 'Changes discarded.',
  /** A split row's note after the provider's region or host. */
  noteWarm: 'warm',
  noteWarming: 'warming up',
  noteWarmingDay: 'warming up, day {day} of {days}',
})
