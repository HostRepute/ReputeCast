import { defineMessages } from '@/i18n'

/**
 * Sending providers in an incident (ProvidersOutage, ProvidersWebhooksSilent,
 * ProvidersStates): the outage banner and its streams, a broadcast held for
 * a decision, the incident timeline, silent webhooks and rejected
 * credentials. {provider} is a provider's short name ("SES", "Postal").
 */
export const mi = defineMessages('providers.incidents', {
  // ── Re-check ─────────────────────────────────────────────────────────
  stillDown: '{name} is still down',
  answered: '{name} answered',
  stillDownDetail: 'The next automatic check runs within 5 minutes.',
  refreshed: 'Health refreshed.',
  checkFailed: 'Check failed',

  // ── Outage banner ────────────────────────────────────────────────────
  /** {time} is when ("09:02"); {since} how long ("14 min"). */
  outageTitle: '{provider} is down: API errors since {time} ({since})',
  /** {op} is the API call ("SendEmail"); {codes} lists error codes; {rate} is a percentage. */
  outageReturns: '{hasOp, select, yes {{op} returns {codes}} other {The API returns {codes}}}{hasRate, select, yes { on {rate} of calls} other {}}.',
  /** {handled} and {waiting} list streams; {verb} is "are" for several streams or a plural name ("Broadcasts"). */
  outageHandled: '{handled} handled themselves.',
  outageWaiting: '{waiting} {verb, select, are {are waiting for you} other {is waiting for you}}.',
  outageHandledWaiting: '{handled} handled themselves; {waiting} {verb, select, are {are waiting for you} other {is waiting for you}}.',
  statusPage: '{provider} status',
  recheckNow: 'Re-check {provider} now',

  // A stream's column.
  failoverActive: 'Failover active',
  fallbackUnavailable: 'Fallback unavailable',
  waitingDecision: 'Waiting for a decision',
  held: 'Held',
  /** {list} lists counts with their sources: "211 welcome series and 97 onboarding". */
  sentDirectly: 'The {list} sent since {time} went through {primary} directly.',
  nothingSent: 'Nothing was sent on {stream} since {time}.',
  fallbackBody: 'Primary {primary} is {unavailable, select, yes {unavailable too} other {healthy}}; its {provider} fallback is unavailable.',
  suggestion: 'Suggestion: add {provider} as a second fallback for {stream}.',
  /** {list} is "welcome series 211, onboarding 97". */
  failoverBody:
    'Primary {provider} → fallback {fallback}, automatically since {time}. {hasList, select, yes {{count, plural, one {# email rerouted so far: {list}.} other {# emails rerouted so far: {list}.}}} other {{count, plural, one {# email rerouted so far.} other {# emails rerouted so far.}}}}',
  switchBack: 'Will switch back to {provider} after {minutes, plural, one {# healthy minute} other {# healthy minutes}}.',
  /** {policy} is a failover policy in a sentence: "pause and ask". */
  heldBody: 'New {stream} mail for {provider} is held until it recovers.',
  heldBodyPolicy: 'New {stream} mail for {provider} is held until it recovers: failover is {policy}.',

  // ── A broadcast held for a decision ──────────────────────────────────
  waitingFor: 'Waiting for {provider}',
  waitingForDetail: '"{name}" resumes on its own once {provider} is healthy.',
  moved: '{count, plural, one {Moved # recipient} other {Moved # recipients}}',
  movedDetail: '"{name}" continues on {provider}.',
  movedDetailFallback: '"{name}" continues on the fallback.',
  cancelled: '"{name}" cancelled',
  cancelledDetail: "{count, plural, one {# held recipient won't be sent.} other {# held recipients won't be sent.}}",
  decisionFailed: 'The decision was not saved',
  policyWarmOnly: 'Broadcast failover only moves to warmed-up providers, and none is available right now.',
  policyPause: 'Broadcast failover is pause-and-ask because moving a bulk send to a cold provider mid-send can hurt reputation.',
  decidedWait: '"{name}": waiting for {provider}; it resumes on its own after {minutes, plural, one {# healthy minute} other {# healthy minutes}}.',
  decidedMove: '"{name}": the held share was moved.',
  decidedCancel: '"{name}": the send was cancelled.',
  openLive: 'Open the live monitor',
  /** {through} is a percentage; {sent} and {held} are counts. */
  heldLine: '"{name}" was {through} through its {provider} share when {provider} failed: {sent} sent, {held} held.',
  /** {others} lists providers with their shares: "Postmark (30%)". */
  othersKeep: '{count, plural, one {{others} keeps sending its share.} other {{others} keep sending their shares.}}',
  waitAutoResume: 'Wait for {provider}, auto-resume',
  moveTo: 'Move to {provider}',
  moveToOverCap: 'Move to {provider} · exceeds warm-up cap, not recommended',
  moveToWarming: 'Move to {provider} · warming up',
  cancelSend: 'Cancel send',
  moveTitle: '{count, plural, one {Move # recipient to {provider}?} other {Move # recipients to {provider}?}}',
  moveOverCap: '{provider} is warming up: this is more than its cap for today, and the warm-up guard may pause it within minutes. Waiting for {current} is the safer choice.',
  moveBody: 'The held share of "{name}" continues on {provider} right away.',
  keepWaiting: 'Keep waiting',
  cancelTitle: 'Cancel "{name}"?',
  /** {held} and {sent} are counts. */
  cancelBody: "{held} held recipients won't get this campaign. The {sent} already sent stay sent, and the rest of the send stops. This can't be undone.",

  // ── Incident timeline ────────────────────────────────────────────────
  timelineTitle: 'Incident timeline',
  allIncidents: 'All incidents',
  /** An entry's kind as its badge says it; {raw} is the kind as written ("quota reached") when it's new. */
  incidentKind:
    '{kind, select, broadcast_held {waiting} failover {failover} failover_ended {switched back} provider_down {down} provider_recovered {recovered} campaign_started {started} webhooks_silent {silent} webhooks_resumed {resumed} other {{raw}}}',

  // ── Silent webhooks ──────────────────────────────────────────────────
  /** PR-9: the provider was asked to send a test event; its arrival is what proves the endpoint reachable. */
  testRequested: 'Test event requested',
  testRequestedDetail: '{provider} sends it to the webhook; it shows in the event log, and the banner clears once events arrive.',
  testFailed: "Couldn't ask for a test event",
  registered: 'Webhook registered again',
  registeredDetail: '{provider} now posts events to {url}.',
  registerFailed: "Couldn't re-register the webhook",
  backfilling: '{count, plural, one {Backfilling # message} other {Backfilling # messages}}',
  backfillingDetail: 'Status from {provider} is normalized as it arrives.',
  backfillFailed: "Couldn't start the backfill",
  manualTitle: 'Re-register the webhook in {provider}',
  manualDescription: "{provider} has no API for webhooks, so add ReputeCast's endpoint again in {provider} itself.",
  manualStep1: "In {provider}, open the mail server, then Webhooks, and delete the old ReputeCast webhook if it's still there.",
  manualStep2: 'Add a webhook with this URL, for all events:',
  manualStep3: 'Back here, choose Send test event: the banner clears as soon as events arrive.',
  silentForHours: '{provider} webhooks have been silent for {hours, plural, one {# hour} other {# hours}}',
  /** {duration} is "45 min". */
  silentFor: '{provider} webhooks have been silent for {duration}',
  /** {count} is a count of messages. */
  silentBody:
    'No delivery, bounce or complaint events since {time}. {provider} accepted {count} messages in that time, so sending works, but their status is unknown and bounce protection for this route is blind. The {provider} route is marked Degraded{held, select, yes { and its share of Broadcasts is held until events resume.} other {.}}',
  sendTestEvent: 'Send test event',
  reregister: 'Re-register webhook',
  backfillFrom: 'Backfill from {provider} API',
  checkedTitle: 'What ReputeCast checked',
  causesTitle: 'Likely causes',
  impactTitle: 'Impact so far',
  impactUnknown: 'Messages with unknown status',
  /** PR-6: messages past the 30-minute grace and those still inside it. */
  impactOverdue: 'overdue · sent more than 30 min ago',
  impactInGrace: 'still inside the 30-min grace period',
  eventsOverdue: 'Events overdue',
  /** {hours} is how long no event arrived. */
  overdueLine: '{count, plural, one {# message} other {# messages}} sent more than 30 min ago and no event for {hours} h',
  impactCampaigns: 'Campaigns affected',
  impactBounces: 'Bounces not yet suppressed',
  unknown: 'unknown',
  impactShare: '{provider} share of Broadcasts',
  shareHeld: 'held',
  shareNotUsed: 'not used',
  backfillNote: "Backfill pulls message status from {provider}'s API and normalizes it, so reports and the suppression list catch up.",
  noBackfillNote: "{provider} can't be asked for past events; they arrive once its webhooks resume.",

  // What ReputeCast checked. {ok} is yes or no; {detail} is the server's own words.
  checkApi: '{provider} API reachable: {ok, select, no {no} other {yes}}{hasAt, select, yes {, {time}} other {}}{hasDetail, select, yes { ({detail})} other {}}.',
  checkRegisteredNo: 'Webhook registered in {provider}: no{hasDetail, select, yes {, {detail}} other {}}.',
  checkRegisteredYes: 'Webhook registered in {provider}: yes{hasId, select, yes {, id {id}} other {}}, pointing at <mono>{url}</mono>.',
  /** PR-9: ReputeCast never calls its own URL; it checks the address, and the provider's deliveries prove the rest. */
  checkEndpoint: 'Our webhook address: {ok, select, no {no provider can reach it} other {a public address}}{hasDetail, select, yes {; {detail}} other {}}.',
  checkProviderLog:
    "Events delivered by {provider} in the last 6 h: {count}{hasQueued, select, yes { ({provider}'s own webhook log shows {queued} queued, {count} sent)} other {}}.",

  // Likely causes; the first one is the most likely and starts in bold. {host} is ReputeCast's host.
  causeWorkerStuckFirst:
    "<b>{provider}'s webhook worker is stuck.</b> Queued events with none sent points at the {postal, select, yes {postal-worker process} other {worker that delivers webhooks}}, which usually restarts cleanly.",
  causeWorkerStuck: "{provider}'s webhook worker is stuck: events are queued but none are sent.",
  causeFirewallFirst: '<b>A firewall change on the {provider} host</b> blocking outbound HTTPS to {host}.',
  causeFirewall: 'A firewall change on the {provider} host blocking outbound HTTPS to {host}.',
  causeSecretFirst: "the endpoint secret rotated in ReputeCast but not in {provider} (signatures would fail with 401, and we've received no requests at all).",
  causeSecret: "Less likely: the endpoint secret rotated in ReputeCast but not in {provider} (signatures would fail with 401, and we've received no requests at all).",
  causeRemovedFirst: '<b>The webhook was removed or disabled in {provider}.</b> Re-register it to point {provider} at ReputeCast again.',
  causeRemoved: 'The webhook was removed or disabled in {provider}.',
  causeUnreachableFirst: "<b>ReputeCast's endpoint isn't reachable from the internet.</b> Check DNS, TLS and the reverse proxy for {host}.",
  causeUnreachable: "ReputeCast's endpoint isn't reachable from the internet (DNS, TLS or the reverse proxy for {host}).",

  // Messages accepted while the webhooks were silent.
  awaitingTitle: 'Messages awaiting status from {provider}',
  awaitingCount: '{count} · newest first',
  colAccepted: 'Accepted',
  colRecipient: 'Recipient',
  colCampaign: 'Campaign',
  colMessageId: '{provider} message id',
  colStatus: 'Status',
  acceptedNoEvent: 'accepted · no event',
  /** A message's status, with the time it was delivered when known. */
  awaitingStatus:
    '{status, select, delivered {delivered} bounced {bounced} deferred {deferred} complained {complained} rejected {rejected} other {accepted}}{hasTime, select, yes { {time}} other {}}',

  // ── Rejected credentials ─────────────────────────────────────────────
  /** {status} is an HTTP status and {statusText} its English name ("401 Unauthorized"); {credential} the credential ("server API token"). */
  authTitle: '{provider} returned {status} {statusText}: the {credential} is no longer valid',
  authEveryCall: 'Every call since {time} has failed.',
  /** One stream that fails over; the last one also says how many messages moved. */
  authFailingOver: 'The {stream} stream ({mail}) is failing over to {fallback} automatically;',
  authFailingOverRerouted: 'The {stream} stream ({mail}) is failing over to {fallback} automatically; {count} messages rerouted so far.',
  /** What a stream carries, in the rejected-credentials banner. */
  streamMail:
    '{stream, select, system {double opt-in confirmations, preference-center links} automation {emails sent by flows} broadcast {campaigns} other {password resets, receipts and alerts}}',
  authRotated: 'Someone may have rotated the {credential} in {provider}.',
  update: 'Update {credential}',
  viewFailed: 'View failed calls',
  auditLog: 'Audit log',
  lastSuccessCall: 'Last successful call',
  noneRecorded: 'none recorded',
  errorFact: 'Error',
  credentialsRejected: 'Credentials rejected',
  /** {time} is when the last event arrived; {credential} is "token", "key"… */
  authWebhooks: '{hasEvent, select, yes {last event {time}} other {no events yet}} · unaffected until {credential} is fixed',
  streamsAffected: 'Streams affected',
  streamRole: '{stream} · {role, select, split {split} primary {primary} fallback {fallback} other {condition}}',
  failoverTo: 'Failover to {provider} active',
})
