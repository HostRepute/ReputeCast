import { defineMessages } from '@/i18n'

/**
 * 7.8 Deliverability: the page and its states, the domain and IP cards, the
 * blocklist table, provider health, Postmaster, SNDS and the incident
 * timeline. The DMARC card and its dialogs are in dmarc.messages.ts.
 * Source names (Spamhaus, UCEPROTECT), product names (Google Postmaster
 * Tools, Microsoft SNDS) and SPF/DKIM/DMARC results stay as they are.
 */
export const m = defineMessages('deliverability', {
  // ── Page ─────────────────────────────────────────────────────────────
  title: 'Deliverability',
  loadFailed: "Couldn't load Deliverability",
  /** After the server's error message. */
  loadFailedBody: 'Nothing was changed; the checks keep running in the background.',
  loadingLabel: 'Loading',
  loadingStatus: 'Loading deliverability',
  checksFinished: 'Checks finished',
  /** {sources} is a formatted number of blocklist sources. */
  checksFinishedListed: '{listed, plural, one {# listing on {sources} sources} other {# listings on {sources} sources}}',
  checksFinishedClean: 'clean on {sources} sources',

  hrUnreachable: 'HostRepute unreachable · cached results',
  /** {interval} is how often HostRepute checks: hourly or daily. */
  hrConnected: '{interval, select, hourly {HostRepute connected · hourly checks} daily {HostRepute connected · daily checks} other {HostRepute connected · {interval} checks}}',
  hrNotConnected: 'HostRepute not connected · provider data and built-in checks',
  hrNeedsReconnect: 'HostRepute needs reconnecting',
  lastFullCheck: 'Last full check {time}',
  lastFullCheckNext: 'Last full check {time} · next {next}',
  firstCheckRunning: 'First check running',
  lastCheckDaily: 'Last check {time} · daily',
  dailyChecks: 'Daily checks',

  startFailed: "Couldn't start the checks",
  exportFailed: "Couldn't export the report",
  exportReport: 'Export report',
  runAll: 'Run all checks now',
  runChecks: 'Run checks now',
  nothingToCheck: 'Connect a sending provider first: there is nothing to check yet.',

  emptyTitle: 'Deliverability appears once mail can be sent',
  connectProvider: 'Connect a sending provider',
  emptyBody: "Domain and IP health, blocklists, Postmaster, SNDS and DMARC show up here for your sending domains and dedicated IPs, with each provider's account health.",

  /** {assets} lists the domains and IPs: "news.acme.io and 203.0.113.25". */
  checking: 'Checking {assets}',
  /** {eta} is how long is left: "about 6 s left". */
  checkingProgress: '{answered} of {total} sources answered · {eta}',
  etaStarting: 'starting',
  etaFinishing: 'finishing',
  etaSeconds: 'about {n} s left',
  etaMinutes: 'about {n} min left',

  delistAlready: 'Delisting already requested on {date}',
  delistAlreadyDetail: "{source} hasn't answered yet; HostRepute follows up.",
  delistRequested: 'Delisting requested for {asset}',
  /** {sources} lists the blocklists asked: "UCEPROTECT Level 3 and SpamCop". */
  delistRequestedDetail: '{sources} · HostRepute follows up',
  delistFailed: "Couldn't request delisting",

  cachedLabel: 'Cached results',
  staleNote: 'Send Readiness stays usable: blocklist checks are marked Unknown instead of Pass until HostRepute answers.',
  noDataYet: 'No data yet',
  postmasterPlaceholder: 'Domain and IP reputation, spam rate and authentication at Gmail. Connect HostRepute to bring it in; it appears here after the first sync.',
  sndsPlaceholder: 'Filter results, complaints and trap hits for your dedicated IPs at Outlook.com. Connect HostRepute to bring it in; it appears here after the first sync.',
  connectHostRepute: 'Connect HostRepute',

  inviteTitle: 'Everything on this page works without HostRepute. Connecting it adds more.',
  inviteBody:
    'HostRepute is our sibling email-reputation service. Connected, it checks 84 blocklists every hour instead of 8 once a day, adds delisting workflows, collects your DMARC reports, and feeds all of it into Send Readiness. No paywall in ReputeCast: every feature here stays on either way.',
  whatHostReputeChecks: 'What HostRepute checks',

  reconnectLabel: 'HostRepute needs reconnecting',
  reconnectBadge: 'Reconnect',
  reconnectBody:
    'HostRepute rejected the token (revoked from the HostRepute side or expired). Blocklist, Postmaster, SNDS and DMARC data is stale since then; Send Readiness marks the reputation check as “unknown”, not as a Block.',
  reconnectBodyAt:
    'HostRepute rejected the token on {date}, {time} (revoked from the HostRepute side or expired). Blocklist, Postmaster, SNDS and DMARC data is stale since then; Send Readiness marks the reputation check as “unknown”, not as a Block.',
  reconnectWith: 'Reconnect with HostRepute',

  // HostRepute unreachable. {pm} and {snds} say whether Google Postmaster
  // and Microsoft SNDS are among the cached data (yes) or not (no).
  unreachableStale:
    '{pm, select, yes {{snds, select, yes {Blocklist, DMARC-report, Google Postmaster, Microsoft SNDS and incident data below are from the last successful check{hasTime, select, yes { at {time}} other {}}.} other {Blocklist, DMARC-report, Google Postmaster and incident data below are from the last successful check{hasTime, select, yes { at {time}} other {}}.}}} other {{snds, select, yes {Blocklist, DMARC-report, Microsoft SNDS and incident data below are from the last successful check{hasTime, select, yes { at {time}} other {}}.} other {Blocklist, DMARC-report and incident data below are from the last successful check{hasTime, select, yes { at {time}} other {}}.}}}}',
  /** {pm} and {snds}: yes when Google Postmaster or Microsoft SNDS is fetched directly. */
  unreachableCurrent:
    '{pm, select, yes {{snds, select, yes {Provider account health, Google Postmaster and Microsoft SNDS are fetched directly and are current.} other {Provider account health and Google Postmaster are fetched directly and are current.}}} other {{snds, select, yes {Provider account health and Microsoft SNDS are fetched directly and are current.} other {Provider account health is fetched directly and is current.}}}}',
  unreachableFallbackFound: 'The built-in {sources}-source blocklist check ran as a fallback and found {listed, plural, one {# listing} other {# listings}}.',
  unreachableFallbackNothing: 'The built-in {sources}-source blocklist check ran as a fallback and found nothing new.',
  unreachableFallbackRuns: 'The built-in blocklist check runs as a fallback.',
  hrStatus: 'HostRepute status',
  useBuiltin: "Use built-in checks until it's back",

  // ── Domain and IP cards ──────────────────────────────────────────────
  assetsTitle: 'Health per sending domain and IP',
  /** "1 of 84 · UCEPROTECT L3": how many sources list it, and which. */
  listingLine: '{listed} of {total} · {names}',
  listingNone: '0 of {total} listed',
  dedicatedIp: 'Dedicated IP',
  warmingUpDay: 'warming up, day {day} of {days}',
  warmingUp: 'warming up',
  notCheckedYet: 'not checked yet',
  rdnsMissing: 'missing',
  rdnsMatches: '{ptr} · matches',
  rdnsNoMatch: "{ptr} · doesn't resolve back",
  noDataYetLower: 'no data yet',
  /** {filter} is the SNDS filter result: "Green". */
  sndsTraps: '{filter} · {count, plural, one {# trap hit} other {# trap hits}}',
  sndsCurrent: '{filter} · current',
  sndsStale: '{filter} · stale',
  /** A HostRepute score with how many blocklists list the asset: "81 · Listed on 1". */
  badgeScoreListed: '{score} · Listed on {listed}',
  /** A HostRepute score with its verdict: "96 · Good". */
  badgeScore: '{score} · {verdict}',
  badgeListed: 'Listed on {listed}',
  badgeNotChecked: 'Not checked yet',
  badgeCleanOn: 'Clean on {sources}',
  verdictGood: 'Good',
  /** HostRepute's verdict words; another word shows as HostRepute sends it. */
  verdict: '{verdict, select, Good {Good} Fair {Fair} Poor {Poor} Bad {Bad} other {{verdict}}}',
  needsAttention: 'Needs attention',
  notMonitored: 'Not monitored',
  sendingDomain: 'Sending domain',
  /** {names} lists the providers: "SES, Postmark and Postal". */
  usedBy: 'used by {names}',
  notMonitoredYet: 'not monitored by HostRepute yet',
  blocklists: 'Blocklists',
  checkedOnceMonitored: "checked once it's monitored",
  monitoringAdded: '{asset} added to monitoring',
  monitoringAddedDetail: 'HostRepute checks it on its blocklists within the hour.',
  addToMonitoring: 'Add to monitoring',
  builtinChecks: 'built-in checks',
  blocklistsBuiltin: 'Blocklists · built-in, {sources} public DNSBLs',
  domainBlocklists: 'Domain blocklists',
  dmarcAlignment7d: 'DMARC alignment, 7 d',
  gmailDomainReputation: 'Gmail domain reputation',
  dmarcAlignment: 'DMARC alignment',
  needsReports: 'needs reports: add rua to HostRepute or your own mailbox',
  ipBlocklists: 'IP blocklists',
  noneOfDaily: '0 of {total} · daily',
  reverseDns: 'Reverse DNS',
  requestDelisting: 'Request delisting',
  warmupPlan: 'Warm-up plan',
  moreSources: 'HostRepute would also check 76 more sources hourly, including UCEPROTECT.',
  sharedPool: 'Shared pool',
  sharedPoolRegion: 'Shared pool, {region}',
  /** {name} is the stream's name in English, for a stream without its own words. */
  streamOnly: '{stream, select, broadcast {Broadcast stream only} automation {Automation stream only} system {System stream only} other {{name} stream only}}',
  checkedDomainLevel: 'checked at domain level',
  checkedDomainLevelAmazon: 'checked at domain level, Amazon manages IP reputation',
  poolName: '{provider} pool',
  sharedIps: 'Shared IPs',
  accountReputation: 'Account reputation',
  providerStatus: 'Provider status',
  gmailIpReputationPool: 'Gmail IP reputation, pool',
  dedicatedIpOption: 'Dedicated IP option',
  volumeThisMonth: 'Volume this month',

  // While HostRepute can't be refreshed.
  staleAt: 'Stale · {time}',
  stale: 'Stale',
  staleBlocklistsVia: '{kind, select, domain {Domain blocklists · {sources} via HostRepute} other {IP blocklists · {sources} via HostRepute}}',
  staleBlocklistsFallback: '{kind, select, domain {Domain blocklists · built-in fallback} other {IP blocklists · built-in fallback}}',
  fallbackLine: '{listed} of {total} · {time}',
  runningNow: 'running now',
  current: 'Current',
  bounceComplaint: 'Bounce · complaint',
  reputationCurrent: '{reputation} · current',
  reputationStale: '{reputation} · stale',

  // Words of Google Postmaster Tools (reputations) and SNDS (filter results);
  // another value shows as it comes.
  reputation: '{level, select, high {High} medium {Medium} low {Low} bad {Bad} other {{raw}}}',
  filterResult: '{level, select, green {Green} yellow {Yellow} red {Red} other {{raw}}}',
  /** HostRepute's check interval in a caption. */
  interval: '{interval, select, hourly {hourly} daily {daily} other {{interval}}}',
  streamBroadcast: 'Broadcast',
  streamAutomation: 'Automation',
  streamSystem: 'System',

  // ── Blocklist table ──────────────────────────────────────────────────
  blocklistTitleVia: 'Blacklist status · {sources} sources via HostRepute',
  blocklistTitleBuiltin: 'Blacklist status · built-in, {sources} public sources',
  /** {assets} lists the checked domains and IPs; {interval} is "hourly". */
  blocklistCaption: '{assets} · {interval}',
  blocklistCaptionStale: '{assets} · {interval} · stale',
  blocklistCaptionBuiltin: 'checked once a day · delisting links included',
  cleanCount: '{count} clean',
  listedCount: '{count} listed',
  colSource: 'Source',
  colChecks: 'Checks',
  colStatus: 'Status',
  colSince: 'Since',
  colDelisting: 'Delisting',
  listed: 'Listed',
  clean: 'Clean',
  delistingPageLabel: 'Delisting page of {source} (opens in a new tab)',
  delistingPage: 'Delisting page',
  noBlocklistCheck: 'No blocklist check has run yet. Run the checks to see where your domains and IPs stand.',
  showingSorted: 'Showing {shown} of {total} · sorted by impact',
  showTop: 'Show top {n}',
  showAll: 'Show all {n}',
  withHostRepute: 'With HostRepute: 84 sources every hour, listing history, and one-click delisting requests. Free to connect.',

  // ── Provider health ──────────────────────────────────────────────────
  providerHealthTitle: 'Account health per sending provider',
  providerHealthConnected: "each provider's own thresholds",
  providerHealthDirect: "straight from each provider's API, unaffected by HostRepute",
  /** A rate bar's label for screen readers: "Bounce rate · review 5%, pause 10%: 0.6%". */
  rateBarLabel: '{label}: {value}',

  // ── Postmaster and SNDS ──────────────────────────────────────────────
  connectedDirectly: 'Connected directly',
  connected: 'Connected',
  spamChartLabel: 'Gmail spam rate over 30 days, between {min} and {max} percent, last value {last} percent',
  spamChartNoData: 'Gmail spam rate over 30 days: no data yet',
  spamTableCaption: 'Gmail spam rate by two-day period',
  colDays: 'Days',
  colSpamRate: 'Spam rate',
  dayRange: '{from} to {to}',
  noData: 'no data',
  peak: 'peak {rate} on {date}',
  /** {note} is HostRepute's note on the peak: "first send after migration". */
  peakNote: 'peak {rate} on {date}, {note}',
  today: 'today',
  domainReputation: 'Domain reputation',
  ipReputation: 'IP reputation',
  ipReputationPool: 'IP reputation, {pool} pool',
  spamRate30d: 'Spam rate, 30 d',
  authSuccess: 'SPF · DKIM · DMARC success',
  tlsInbound: 'TLS inbound',
  feedbackLoop: 'Feedback loop',
  noFlagged: 'no flagged campaigns',
  flagged: '{count} flagged',
  directNote: "Postmaster and SNDS connect through their own APIs; HostRepute isn't needed for either.",
  filterResultLabel: 'Filter result',
  messages7d: 'Messages, 7 d',
  complaintRate: 'Complaint rate',
  underRate: 'under {rate}',
  spamTrapHits: 'Spam trap hits',
  jmrpFeedback: 'JMRP feedback',
  jmrpReports: '{count, plural, one {# report} other {# reports}}',
  sharedPools: 'Shared pools',
  notCoveredBySnds: 'not covered by SNDS',

  // ── Incident timeline ────────────────────────────────────────────────
  incidentsTitle: 'Incident timeline',
  incidentsLinked: 'linked to campaigns and providers',
  incidentsFromEvents: 'from provider events and auto-pause history',
  noIncidents: 'No incidents in the last 45 days. Listings, auto-pauses, outages and warm-up slowdowns appear here.',
  incidentsShowing: 'Showing the latest {shown} of {total} in the last 45 days',
})
