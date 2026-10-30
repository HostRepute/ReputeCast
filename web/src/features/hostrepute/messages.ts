import { defineMessages } from '@/i18n'

/**
 * HostRepute: the card before connecting (ConnectHostRepute), Deliverability
 * › Connect HostRepute with the OAuth handoff, its return and the API key,
 * and Settings › Integrations › HostRepute (the connected view, the revoked
 * token, the plan's monitoring limit, disconnecting).
 */
export const m = defineMessages('hostrepute', {
  // ── Shared ───────────────────────────────────────────────────────────
  oauthFailed: "Couldn't start connecting HostRepute",
  roleSendingDomain: 'Sending domain',
  roleOrgDomain: 'Organization domain',
  roleDedicatedIp: 'Dedicated IP',
  /** {provider} is the sending provider's short name ("SES"). */
  roleDedicatedIpOf: 'Dedicated IP · {provider}',
  roleServerHostname: 'Server hostname',
  /** {provider} is the sending provider's short name ("Postal"). */
  roleServerHostnameOf: '{provider} server hostname',
  roleIp: 'IP',
  roleDomain: 'Domain',
  addToMonitoring: 'Add to monitoring',
  added: '{asset} added to monitoring',
  addedDetail: 'HostRepute checks it within the hour',
  addFailed: "Couldn't add {asset}",
  useApiKey: 'Use an API key instead',
  reconnectWith: 'Reconnect with HostRepute',
  statusConnected: 'Connected',
  statusNeedsReconnect: 'Needs reconnecting',
  statusNotConnected: 'Not connected',
  methodApiKey: 'API key',
  nothingChanged: 'Nothing was changed.',

  // ── Before connecting (ConnectCard) ──────────────────────────────────
  connectTitle: 'Connect HostRepute',
  connectLine: 'Optional · no paywall in ReputeCast · HostRepute is a separate paid service by the same team',
  benefitReadinessTitle: 'Send Readiness gets reputation checks',
  benefitReadinessText: 'Blocklist status of your domains and IPs becomes a Pass / Warn / Block check before every send.',
  benefitDeliverabilityTitle: 'Deliverability gets the full picture',
  benefitDeliverabilityText: 'Postmaster, SNDS and DMARC panels fill in next to the provider data you already see.',
  benefitAlertsTitle: 'Alerts land where you work',
  benefitAlertsText: 'A new listing or a reputation drop shows up in ReputeCast notifications, with the delisting link.',
  sharedTitle: 'Shared with HostRepute',
  /** {domains} lists the domains, comma-separated. */
  sharedDomains: 'Sending domains: {domains}',
  /** {ips} lists the IPs, comma-separated. */
  sharedIps: 'Dedicated IPs: {ips}',
  sharedNothingYet: 'Nothing yet: your sending domains and dedicated IPs, once you add them',
  sharedNever: 'Never contacts, email content or reports',
  sharedNothingElse: 'Nothing else: no contacts, no email content, no reports',
  receivedTitle: 'Received by ReputeCast',
  receivedBlocklists: 'Blocklist status across 80+ sources, with delisting links',
  receivedPostmaster: 'Google Postmaster Tools and Microsoft SNDS data',
  receivedDmarc: 'DMARC report summaries',
  receivedAlerts: 'Alerts, delivered as ReputeCast notifications',
  openSettings: 'Open Settings › Integrations',
  connectedTeam: 'Connected to team “{team}”',
  connectWith: 'Connect with HostRepute',
  /** {role} is the member's role. */
  permHave: 'You can connect HostRepute as {role}',
  permAsk: "Your role can't connect HostRepute · ask an Owner or Admin of this workspace",

  // ── Deliverability › Connect HostRepute ──────────────────────────────
  pageTitle: 'Connect HostRepute',
  loadFailed: "Couldn't load the HostRepute connection",
  loading: 'Loading',
  redirectTitle: 'Redirecting to HostRepute…',
  approveOn: 'Approve the request on {host}',
  leaving: "You're leaving ReputeCast to sign in to HostRepute and confirm the scopes above. You'll come straight back here. Nothing is connected until you approve.",
  notRedirected: "Didn't get redirected? <link>Open HostRepute</link>",
  keyTitle: 'Use an API key instead',
  keyLine: "For installs that can't use the browser handoff",
  keyLabel: 'HostRepute API key',
  keyEmpty: 'Paste the API key from HostRepute → Settings → API.',
  /** {scopes} lists the scope keys ("monitor:read, alerts:read and assets:write"). */
  keyNote: 'Create a key in HostRepute → Settings → API with the {scopes} scopes. The key is write-only here: it can be replaced but never read back.',
  keyConnect: 'Connect',
  connectFailed: "Couldn't connect HostRepute",
  backTitle: 'Back from HostRepute',
  backSuccess: 'Success',
  backConnected: 'Connected',
  /** {scopes} lists the granted scope keys, comma-separated. */
  backLine: 'Team “{team}” · scopes {scopes} · {syncing, select, yes {first sync running (about 1 minute)} other {first sync done}}',
  monitoredNow: 'Monitored now',
  monitoredNone: 'nothing yet: HostRepute adds them on the first sync',
  suggested: 'Suggested',
  suggestedLine: '{asset} is not monitored yet ·',
  stillUsable: 'ReputeCast stays fully usable.',
  finish: 'Finish',
  stillConnected: 'Still connected',
  // Why the handoff came back without a connection. {kept} is "connected"
  // when the workspace keeps its connection to team {team}.
  reasonDeclined: '{kept, select, connected {You declined the request on HostRepute. Your connection to team “{team}” is unchanged.} other {You declined the request on HostRepute. Nothing was connected.}}',
  reasonWrongUser: 'This handoff was started by someone else. Start it again from your own account.',
  reasonForbidden:
    "{kept, select, connected {Your role can't connect HostRepute. Your connection to team “{team}” is unchanged.} other {Your role can't connect HostRepute. Nothing was connected.}}",
  reasonExchange:
    "{kept, select, connected {HostRepute didn't accept the approval (it expired or was already used). Your connection to team “{team}” is unchanged; try again.} other {HostRepute didn't accept the approval (it expired or was already used). Nothing was connected; try again.}}",
  reasonUnreachable:
    "{kept, select, connected {Couldn't reach HostRepute to finish connecting. Your connection to team “{team}” is unchanged; try again in a minute.} other {Couldn't reach HostRepute to finish connecting. Nothing was connected; try again in a minute.}}",
  reasonHostReputeError:
    '{kept, select, connected {HostRepute answered with an error. Your connection to team “{team}” is unchanged; try again.} other {HostRepute answered with an error. Nothing was connected; try again.}}',
  reasonSaveFailed:
    "{kept, select, connected {ReputeCast couldn't save the connection. Your connection to team “{team}” is unchanged; try again.} other {ReputeCast couldn't save the connection. Nothing was connected; try again.}}",
  reasonOther:
    '{kept, select, connected {HostRepute sent you back without connecting. Your connection to team “{team}” is unchanged; try again.} other {HostRepute sent you back without connecting. Nothing was connected; try again.}}',

  // ── Settings › Integrations › HostRepute ─────────────────────────────
  documentTitle: 'HostRepute · Integrations',
  integrationsLoadFailed: "Couldn't load integrations",
  integrationsLoading: 'Loading integrations',
  listed: 'Listed',
  /** {sources} lists the blocklists, comma-separated. */
  listings: '{count, plural, one {# listing} other {# listings}} ({sources}) · delisting link in Deliverability',
  firstCheck: 'First check running',
  clean: 'Clean',
  dmarcPolicy: 'DMARC p={policy}',
  /** {aligned} is a percentage. */
  dmarcPolicyAligned: 'DMARC p={policy}, {aligned} aligned',
  cleanOn: 'Clean on {count} blocklists',
  /** {level} is Google Postmaster's domain reputation; another value shows as {raw}. */
  postmasterReputation: 'Postmaster {level, select, high {high} medium {medium} low {low} bad {bad} other {{raw}}} reputation',
  sectionLine: 'Reputation monitoring that feeds Send Readiness and Deliverability',
  connectedTo: 'Connected to HostRepute team “{team}”',
  metaConnected: 'connected {date}',
  metaConnectedBy: 'connected {date} by {name}',
  metaTokenValid: 'token valid',
  metaUnreachable: 'HostRepute unreachable',
  metaLastSync: 'last sync {ago}',
  metaFirstSync: 'first sync running',
  unreachable: 'Unreachable',
  /** {error} is the server's sentence, without its full stop. */
  syncError: '{error}{hasTime, select, yes { at {time}} other {}}. ReputeCast retries every hour and runs its built-in blocklist check meanwhile.',
  grantedScopes: 'Granted scopes',
  sentToHostRepute: 'Sent to HostRepute',
  sentToHostReputeValue: 'Sending domains and dedicated IPs to monitor. Never contacts, never email content.',
  alerts: 'Alerts',
  alertsOn: 'Delivered into ReputeCast notifications (in-app and email per member preference)',
  alertsOff: 'Not delivered into ReputeCast notifications',
  assetsCaption: 'Assets monitored by HostRepute',
  colAsset: 'Asset',
  colType: 'Type',
  colMonitoring: 'Monitoring',
  colLastSync: 'Last sync',
  colStatus: 'Status',
  monitored: 'Monitored',
  notMonitored: 'Not monitored',
  syncNow: 'Sync now',
  syncing: 'Syncing with HostRepute',
  syncFailed: "Couldn't sync",
  openDeliverability: 'Open Deliverability',
  disconnect: 'Disconnect',

  // Token revoked or expired.
  revokedTitle: 'Token revoked or expired',
  /** {method} is "API key" or "OAuth". */
  revokedLine: 'HostRepute team “{team}” · {method}',
  staleSince: 'Stale since {date}',
  staleSinceLastSync: 'Stale since the last sync',
  reconnectLabel: 'Reconnect',
  stillWorking: 'Still working',
  stillWorkingValue: 'Everything in ReputeCast · provider-side health, bounce and complaint rates, authentication checks',
  paused: 'Paused',
  pausedValue: 'Blocklist checks, Postmaster and SNDS panels, DMARC summary, HostRepute alerts',

  // The plan's monitoring limit.
  limitTitle: 'HostRepute monitoring limit reached',
  limitAdding: 'Adding {asset} to monitoring',
  /** {used} and {limit} are counts. */
  limitUsed: '{used} of {limit} used',
  /** {used} and {limit} are formatted counts; {usedCount} is {used} as a number. */
  limitNote:
    "Your HostRepute team “{team}” monitors {used} of {limit} assets allowed by its plan, so {asset} can't be added right now. This is a HostRepute limit, not a ReputeCast one: ReputeCast stays fully usable and the {usedCount, plural, =0 {zero} =1 {one} =2 {two} =3 {three} =4 {four} =5 {five} =6 {six} =7 {seven} =8 {eight} =9 {nine} =10 {ten} other {#}} monitored assets keep syncing.",
  options: 'Options',
  /** {role} is what the asset to swap is ("organization domain"). */
  optionsSwap: 'Raise the limit in HostRepute, or swap an asset: {swap} ({role}) could make room for {asset}, which actually sends mail.',
  optionsNoSwap: 'Raise the limit in HostRepute, or stop monitoring an asset there to make room.',
  /** An asset's role in a sentence; another role shows as {raw}. */
  swapRole: '{role, select, organization_domain {organization domain} sending_domain {sending domain} dedicated_ip {dedicated IP} server_hostname {server hostname} other {{raw}}}',
  managePlan: 'Manage the plan in HostRepute',
  swapButton: 'Swap {swap} for {asset}',
  notNow: 'Not now',
  swapped: '{asset} monitored instead of {swap}',
  swapFailed: "Couldn't swap the assets",

  // Disconnecting.
  disconnected: 'HostRepute disconnected',
  disconnectedDetail: 'team “{team}” keeps monitoring on hostrepute.com',
  disconnectFailed: "Couldn't disconnect HostRepute",
  disconnectEyebrow: 'Dialog · disconnect',
  disconnectTitle: 'Disconnect HostRepute?',
  disconnectBody: 'ReputeCast forgets the token. Your HostRepute team and its monitoring keep running on hostrepute.com; nothing is deleted there.',
  stopsWorking: 'Stops working in ReputeCast',
  stopsReadiness: 'Blocklist check in Send Readiness (shown as “not checked”)',
  stopsPanels: 'Postmaster, SNDS and DMARC panels in Deliverability',
  stopsAlerts: 'HostRepute alerts in notifications',
  stopsAddToMonitoring: '“Add to monitoring” for new domains and IPs',
  keepsWorking: 'Keeps working',
  keepsEverything: 'Every other feature, including provider health, routing, guarded sending and reports',
})
