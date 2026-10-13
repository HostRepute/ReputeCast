import { setDisplayTimeZone } from '@/lib/format'
import {
  abLine,
  deliveredMix,
  errorRate,
  eventsCaption,
  failoverCaption,
  finalTiles,
  finishedText,
  headerBadges,
  incidentTitle,
  lostText,
  mailboxStatus,
  mobileCaption,
  mobileEta,
  mobileTiles,
  moveFacts,
  optionAction,
  outageSummary,
  outageTitle,
  progressLegend,
  progressText,
  providerLine,
  providerNote,
  providerResultLines,
  providerStatus,
  sentLine,
  startedLine,
  startingText,
  thresholdRows,
  thresholdSummary,
  tiles,
  timingLine,
  trippedFacts,
  type Incident,
  type Mailbox,
  type Provider,
  type Snapshot,
} from './model'

/*
 * The live monitor's words and numbers, pinned against the boards
 * (LiveMonitor, LiveMonitorPaused, LiveMonitorOutage, LiveMonitorStates,
 * LiveMonitorMobile) with their sample data. Times are UTC here.
 */

const T = (hms: string) => `2026-10-14T${hms}Z`

function provider(p: Partial<Provider>): Provider {
  return {
    id: 'p',
    name: 'Amazon SES',
    short: 'SES',
    adapter: 'ses',
    color: 'ses',
    sent: 0,
    planned: 0,
    remaining: 0,
    share: 0,
    rate: 0,
    status: 'healthy',
    routeStatus: 'sending',
    host: '',
    ip: '',
    cost: 0,
    plannedCost: 0,
    retrying: false,
    ...p,
  }
}

function mailbox(m: Partial<Mailbox>): Mailbox {
  return { key: 'gmail', name: 'Gmail', sent: 0, remaining: 0, delivered: 0, bounced: 0, complained: 0, opened: 0, held: false, status: 'ok', ...m }
}

function snapshot(s: Partial<Snapshot>): Snapshot {
  return {
    campaignId: 'c',
    name: 'Weekly Digest #142',
    status: 'sending',
    state: 'sending',
    at: T('09:21:45'),
    version: 1,
    startedBy: 'Sofia Rossi',
    approvedBy: 'Marcus Chen',
    approvedAt: '2026-10-10T16:20:00Z',
    scheduledBy: 'Linh Nguyen',
    pausedBy: '',
    pauseReason: '',
    recipients: 31604,
    sent: 0,
    queued: 0,
    held: 0,
    waiting: 0,
    unsent: 0,
    percent: 0,
    rate: 0,
    totals: { delivered: 0, bounced: 0, hardBounced: 0, softBounced: 0, complained: 0, deferred: 0, rejected: 0, opened: 0, openedHuman: 0, openedMachine: 0, clicked: 0, unsubscribed: 0 },
    mailboxes: [],
    providers: [],
    thresholds: [],
    thresholdScope: 'route',
    details: { from: 'digest@news.acme.io', audience: 'Weekly Digest', audienceCount: 31604, route: 'SES 90% · Postal 10%', cost: 1.76, costPlanned: 2.84, scheduledBy: 'Linh Nguyen' },
    deliveredMix: [],
    incidents: [],
    events: [],
    eventsVia: 'SNS and Postal webhooks',
    lastEventAt: T('09:21:44'),
    failovers: [],
    sharedStreams: [],
    excluded: { suppressed: 0, unsubscribed: 0, bounced: 0, complained: 0, paused: 0, role: 0, sunset: 0, excluded: 0, tiers: 0 },
    error: '',
    resumed: false,
    cost: { total: 1.76, planned: 2.84 },
    providersHeld: [],
    mailboxesHeld: [],
    continuing: [],
    pauseScopeHint: '',
    throttles: [],
    options: { canPause: true, canResume: false, canCancel: true },
    routes: [],
    ...s,
  }
}

beforeAll(() => setDisplayTimeZone('UTC'))

// LiveMonitor: 62% of the way through, SES 90% and Postal 10%.
const ses = provider({ id: 'ses', sent: 17635, planned: 28444, remaining: 10809, share: 90, rate: 14.2, deliveredRate: 99.1, bounceRate: 0.6, complaintRate: 0.03, apiErrorRate: 0.1, quotaFree: 19416, quotaMax: 50000 })
const postal = provider({
  id: 'postal',
  name: 'Postal',
  short: 'Postal',
  adapter: 'postal',
  color: 'postal',
  status: 'warming_up',
  sent: 1959,
  planned: 3160,
  remaining: 1201,
  share: 10,
  rate: 2,
  deliveredRate: 98.6,
  bounceRate: 0.9,
  complaintRate: 0.05,
  apiErrorRate: 0.4,
  host: 'mta.acme.io',
  ip: '203.0.113.25',
  warmupDay: 9,
  warmupDays: 21,
  warmupCap: 6000,
  warmupSent: 1959,
})
const live = snapshot({
  sent: 19594,
  queued: 12010,
  rate: 16.2,
  startedAt: T('09:00:04'),
  etaSeconds: 780,
  finishesAt: T('09:34:44'),
  providers: [ses, postal],
  totals: { delivered: 19402, bounced: 118, hardBounced: 94, softBounced: 24, complained: 6, deferred: 40, rejected: 0, opened: 2811, openedHuman: 1920, openedMachine: 891, clicked: 402, unsubscribed: 0 },
  thresholds: [
    { key: 'bounce', value: 0.6, limit: 2 },
    { key: 'complaint', value: 0.03, limit: 0.1 },
    { key: 'mailbox_complaint', value: 0.05, limit: 0.1, mailbox: 'outlook' },
    { key: 'api_errors', value: 0.2, limit: 5 },
  ],
  ab: { kind: 'subject', metric: 'human_opens', phase: 'testing', sample: 20, leader: 'B', winner: '', variants: [ { key: 'A', rate: 13.2, sent: 3120 }, { key: 'B', rate: 15.1, sent: 3110 } ] },
  deliveredMix: [
    { key: 'gmail', name: 'Gmail', percent: 46.4 },
    { key: 'outlook', name: 'Outlook', percent: 20.6 },
    { key: 'yahoo', name: 'Yahoo', percent: 9.2 },
    { key: 'icloud', name: 'iCloud', percent: 8.1 },
    { key: 'other', name: 'Other', percent: 15.7 },
  ],
})

describe('LiveMonitor', () => {
  it('writes the header line', () => {
    expect(startedLine(live)).toBe('Started by Sofia Rossi · approved by Marcus Chen, Oct 10 16:20')
    expect(headerBadges(live, undefined)).toEqual([{ tone: 'ok', text: 'Sending · live', live: true }])
    expect(eventsCaption(live, new Date(T('09:21:45')), 'live')).toBe(
      'Events via SNS and Postal webhooks · last 1 s ago · pause needs campaigns:pause, resume needs campaigns:send',
    )
  })

  it('writes the progress card', () => {
    expect(progressText(live)).toBe('62%')
    expect(sentLine(live)).toBe('19,594 of 31,604 sent')
    expect(timingLine(live)).toBe('Started 09:00:04 · ETA 13 min · finishes about 09:34')
    expect(progressLegend(live).map((l) => l.text)).toEqual(['Amazon SES 17,635 · 14.2 /s', 'Postal 1,959 · 2.0 /s', 'Queued 12,010'])
    expect(abLine(live.ab)).toBe('A/B subject test: variant B leading, 15.1% vs 13.2% opens on the 20% sample')
  })

  it('writes the five counters', () => {
    expect(tiles(live).map((t) => `${t.label} ${t.value} ${t.sub}`)).toEqual([
      'Delivered 19,402 99.0% of sent',
      'Bounced 118 0.60% · 94 hard, 24 soft',
      'Complained 6 0.03% of delivered',
      'Opened 2,811 14.5% · 1,920 human, 891 machine',
      'Clicked 402 2.1% · 14.3% click-to-open',
    ])
  })

  it('measures each threshold against its limit', () => {
    expect(thresholdRows(live).map((r) => [r.label, r.value, Math.round(r.width), r.color])).toEqual([
      ['Bounce rate', '0.60% of 2.0%', 30, 'var(--ok-bar)'],
      ['Complaint rate', '0.03% of 0.10%', 30, 'var(--ok-bar)'],
      ['Outlook complaints (highest mailbox provider)', '0.05% of 0.10%', 50, 'var(--warn-bar)'],
      ['Provider API errors, 3 min', '0.2% of 5%', 4, 'var(--ok-bar)'],
    ])
  })

  it('describes each sending provider', () => {
    expect(providerNote(ses, false)).toBe('quota 19,416 free after this send')
    expect(providerNote(postal, false)).toBe("warm-up day 9 · 1,959 of today's 6,000 cap")
    expect(providerStatus(ses)).toEqual({ tone: 'ok', text: 'Healthy' })
    expect(providerStatus(postal)).toEqual({ tone: 'info', text: 'Warming up' })
    expect([errorRate(0.1), errorRate(0.4), errorRate(100), errorRate(undefined)]).toEqual(['0.1%', '0.4%', '100%', '—'])
  })

  it('rates mailbox providers OK until they near the limit', () => {
    expect(mailboxStatus(mailbox({ key: 'outlook', name: 'Outlook', delivered: 4049, complaintRate: 0.05 }), live, false)).toEqual({ tone: 'ok', text: 'OK' })
    expect(mailboxStatus(mailbox({ key: 'outlook', name: 'Outlook', delivered: 4049, complaintRate: 0.09 }), live, false)).toEqual({ tone: 'warn', text: 'Near limit' })
    expect(mailboxStatus(mailbox({ key: 'outlook', name: 'Outlook', delivered: 1923, complaintRate: 0.16 }), live, false)).toEqual({ tone: 'warn', text: 'Over limit' })
  })

  it('rounds the delivered mix to whole percents that add up to 100', () => {
    const mix = deliveredMix(live)
    expect(mix.map((m) => `${m.name} ${m.percent}%`)).toEqual(['Gmail 46%', 'Outlook 21%', 'Yahoo 9%', 'iCloud 8%', 'Other 16%'])
    expect(mix.reduce((a, m) => a + m.percent, 0)).toBe(100)
  })
})

describe('LiveMonitorMobile', () => {
  it('writes the phone layout', () => {
    expect(mobileTiles(live).map((t) => `${t.label} ${t.value} ${t.sub}`)).toEqual([
      'Delivered 19,402 99.0%',
      'Bounced 118 0.60%',
      'Complained 6 0.03%',
      'Opened · human 1,920 +891 machine',
      'Clicked 402 2.1% · 14.3% click-to-open',
    ])
    expect(mobileEta(live)).toBe('ETA 13 min · about 09:34')
    expect(progressLegend(live, undefined, true).map((l) => l.text)).toEqual(['SES 17,635', 'Postal 1,959'])
    expect(providerLine(ses)).toBe('14.2 /s · 99.1% delivered')
    expect(providerStatus(postal, true).text).toBe('Warm-up')
    expect(mobileCaption(live, new Date(T('09:21:45')))).toBe('Live · events 1 s ago')
  })
})

// LiveMonitorPaused: Outlook complaints tripped at 09:21:14.
const outlook: Incident = {
  id: 'i1',
  kind: 'threshold',
  scope: 'mailbox_provider',
  mailbox: 'outlook',
  mailboxName: 'Outlook',
  providerId: '',
  provider: '',
  providerShort: '',
  metric: 'complaint',
  value: 0.18,
  limit: 0.1,
  held: 2522,
  trippedAt: T('09:21:14'),
  resolution: '',
  detail: {
    sent: 4115,
    total: 6637,
    events: 7,
    base: 3980,
    succeeded: 0,
    note: "Microsoft's SmartScreen uses complaint rate directly; staying under 0.1% protects placement for every later send.",
    options: [
      { key: 'resume', title: 'Resume as is', body: '', recommended: false, action: 'Resume as is' },
      { key: 'resume_with_changes', title: 'Resume with changes (recommended)', body: '', recommended: true, action: 'Resume with changes' },
      { key: 'cancel_share', title: 'Cancel the Outlook share', body: '', recommended: false, action: 'Cancel the Outlook share' },
    ],
  },
}
const paused = snapshot({
  state: 'partial',
  sent: 22440,
  held: 2522,
  queued: 6642,
  startedAt: T('09:00:04'),
  etaSeconds: 540,
  providers: [ses, postal].map((p) => ({ ...p, sent: p.id === 'ses' ? 20196 : 2244, rate: p.id === 'ses' ? 14.1 : 2 })),
  mailboxes: [mailbox({ key: 'outlook', name: 'Outlook', sent: 4115, remaining: 2522, delivered: 3980, complained: 7, complaintRate: 0.18, bounceRate: 1.1, held: true, heldSince: T('09:21:14') })],
  continuing: ['Gmail', 'Yahoo', 'iCloud', 'Other'],
  totals: { delivered: 22071, bounced: 154, hardBounced: 121, softBounced: 33, complained: 9, deferred: 0, rejected: 0, opened: 3604, openedHuman: 2480, openedMachine: 1124, clicked: 521, unsubscribed: 0 },
  thresholds: [
    { key: 'bounce', value: 0.69, limit: 2 },
    { key: 'complaint', value: 0.04, limit: 0.1 },
    { key: 'mailbox_complaint', value: 0.18, limit: 0.1, mailbox: 'outlook' },
  ],
  incident: outlook,
})

describe('LiveMonitorPaused', () => {
  it('says what tripped and what ReputeCast did', () => {
    expect(incidentTitle(outlook)).toBe('Outlook complaint rate reached 0.18% (limit 0.10%)')
    expect(thresholdSummary(outlook, paused)).toBe(
      'Tripped at 09:21:14, after 4,115 of 6,637 Outlook recipients were sent. 7 complaints out of 3,980 delivered. ReputeCast held the remaining 2,522 Outlook recipients. Gmail, Yahoo, iCloud and Other continue, now 71% complete overall.',
    )
  })

  it('lists the evidence', () => {
    const f = trippedFacts(outlook, paused)
    expect(`${f.head.label} ${f.head.value}`).toBe('Outlook complaints 7 of 3,980 · 0.18%')
    expect(f.bar.width).toBe(100)
    expect(f.rows.map((r) => `${r.label} ${r.value}`)).toEqual(['Limit, per mailbox provider 0.10%', 'Campaign-wide complaints 9 of 22,071 · 0.04%', 'Outlook bounces 1.1% of 2.0%'])
  })

  it('writes progress, counters and badges', () => {
    expect(sentLine(paused, outlook)).toBe('22,440 of 31,604 sent · 2,522 held')
    expect(timingLine(paused, outlook)).toBe('Started 09:00:04 · paused route since 09:21:14 · ETA for the rest 9 min')
    expect(progressLegend(paused, outlook).map((l) => l.text)).toEqual(['Amazon SES 20,196 · 14.1 /s', 'Postal 2,244 · 2.0 /s', 'Held, Outlook 2,522', 'Queued 6,642'])
    const complained = tiles(paused, 'threshold', outlook)[2]!
    expect(`${complained.sub} ${complained.alert}`).toBe('0.04% overall · Outlook 0.18% true')
    expect(tiles(paused, 'threshold', outlook)[4]!.sub).toBe('2.4%')
    expect(headerBadges(paused, outlook).map((b) => b.text)).toEqual(['Auto-paused · Outlook', 'Other mailbox providers still sending'])
    expect(eventsCaption(paused, new Date(T('09:21:45')), 'threshold')).toBe(
      'Events live · last 1 s ago · anyone with campaigns:pause can stop a send; resume needs campaigns:send',
    )
  })

  it('maps each choice to the API', () => {
    expect(optionAction(outlook, 'resume_with_changes')).toEqual({ kind: 'resume', body: { mailbox: 'outlook', withChanges: true } })
    expect(optionAction(outlook, 'cancel_share')).toEqual({ kind: 'cancel', body: { mailbox: 'outlook' } })
  })
})

// LiveMonitorOutage: Postal refused connections from 09:14:02.
const outage: Incident = {
  id: 'i2',
  kind: 'provider_outage',
  scope: 'route',
  mailbox: '',
  mailboxName: '',
  providerId: 'postal',
  provider: 'Postal',
  providerShort: 'Postal',
  metric: 'api_errors',
  value: 100,
  limit: 5,
  held: 1204,
  trippedAt: T('09:17:10'),
  resolution: '',
  detail: {
    reason: 'outage',
    errorKind: 'connection errors',
    from: T('09:14:02'),
    to: T('09:17:10'),
    attempts: 37,
    succeeded: 0,
    host: 'mta.acme.io',
    policy: 'pause_and_ask',
    log: [
      { at: T('09:14:02'), text: 'ECONNREFUSED mta.acme.io:443', bad: true },
      { at: T('09:17:40'), text: 'health check · still failing', bad: false },
    ],
    move: { targetId: 'ses', target: 'Amazon SES', targetShort: 'SES', extraCost: 0.12, extraSeconds: 86, rate: 14, quotaFree: 18212, quotaMax: 50000, risk: 'low', riskText: 'Low · SES is warm' },
    warmupNote: "The Postal warm-up schedule shifts by a day so the IP doesn't miss its ramp.",
  },
}
const down = snapshot({
  state: 'partial',
  sent: 23060,
  held: 1204,
  queued: 7340,
  startedAt: T('09:00:04'),
  finishesAt: T('09:27:30'),
  providers: [
    { ...ses, sent: 21104, remaining: 7340, rate: 14 },
    { ...postal, sent: 1956, remaining: 1204, rate: 0, routeStatus: 'held', downSince: T('09:14:02'), apiErrorRate: 100 },
  ],
  totals: { delivered: 22834, bounced: 139, hardBounced: 110, softBounced: 29, complained: 7, deferred: 0, rejected: 0, opened: 3781, openedHuman: 2590, openedMachine: 1191, clicked: 548, unsubscribed: 0 },
  failovers: [
    { stream: 'Automations', to: 'Postmark' },
    { stream: 'System', to: 'SES' },
  ],
  incident: outage,
})

describe('LiveMonitorOutage', () => {
  it('says what Postal returned and what is on hold', () => {
    expect(outageTitle(outage)).toBe('Postal returned connection errors for 3 minutes')
    expect(outageSummary(outage, down)).toBe(
      'From 09:14:02 to 09:17:10, 37 send attempts to mta.acme.io failed and 0 succeeded, so the Postal share of this campaign is on hold: 1,956 sent, 1,204 remaining. Amazon SES keeps sending at 14 /s, 73% complete overall. Nothing moved automatically because Broadcast failover is set to pause and ask.',
    )
  })

  it('prices the move to SES', () => {
    const m = moveFacts(outage)!
    expect(m.title).toBe('If you move the share to SES')
    expect(m.rows.map((r) => `${r.label} ${r.value}`)).toEqual(['Extra cost $0.12', 'Extra time at 14 /s 1 min 26 s', 'SES quota after 18,212 free of 50,000'])
    expect(m.risk).toEqual({ tone: 'ok', text: 'Low · SES is warm' })
  })

  it('writes progress, counters, badges and the provider rows', () => {
    expect(sentLine(down, outage)).toBe('23,060 of 31,604 sent · 1,204 held on Postal')
    expect(timingLine(down, outage)).toBe('Started 09:00:04 · SES finishes about 09:27')
    expect(progressLegend(down, outage).map((l) => l.text)).toEqual(['Amazon SES 21,104 · 14.0 /s', 'Postal 1,956 · 0 /s', 'Held on Postal 1,204', 'Queued on SES 7,340'])
    expect(tiles(down, 'outage', outage).map((t) => t.sub)).toEqual(['99.0% of sent', '0.60%', '0.03%', '16.6% · 2,590 human, 1,191 machine', '2.4%'])
    expect(headerBadges(down, outage).map((b) => b.text)).toEqual(['Postal route held · waiting for your decision', 'SES still sending'])
    expect(providerNote(down.providers[1]!, true)).toBe('mta.acme.io · 203.0.113.25')
    expect(providerStatus(down.providers[1]!)).toEqual({ tone: 'bad', text: 'Down since 09:14' })
    expect(failoverCaption(down)).toBe('Automations and System already failed over to Postmark and SES on their own')
    expect(failoverCaption({ ...down, failovers: [] })).toBe("Automations and System don't use Postal")
    expect(failoverCaption({ ...down, failovers: [], sharedStreams: ['Automations'] })).toBe('Automations also uses Postal')
    expect(optionAction(outage, 'move')).toEqual({ kind: 'move', body: { fromProviderId: 'postal', toProviderId: 'ses' } })
  })
})

describe('LiveMonitorStates', () => {
  it('writes the connecting state', () => {
    const s = snapshot({ startedAt: T('09:00:04'), providers: [ses, postal], lastEventAt: undefined })
    expect(startingText(s)).toEqual({
      title: 'Starting send · 09:00:04',
      body: 'Queue built for 31,604 recipients. Handing the first batches to SES and Postal; delivery events usually arrive within 10 seconds.',
    })
  })

  it('writes the stream-lost state', () => {
    expect(lostText(new Date(T('09:18:51')), T('09:18:51'), 3, 10, false, new Date(T('09:20:30')))).toEqual({
      title: 'Lost the live event stream at 09:18:51',
      body: "The browser's connection to ReputeCast dropped. Sending continues on the server; the counts below are from 09:18:51 and may be up to 2 minutes behind. Reconnecting automatically (attempt 3 of 10).",
    })
  })

  it('writes the complete state', () => {
    const s = snapshot({
      state: 'finished',
      status: 'sent',
      sent: 31604,
      completedAt: T('09:34:12'),
      durationSeconds: 2048,
      providers: [
        { ...ses, sent: 28444, deliveredRate: 99.2, cost: 2.84 },
        { ...postal, sent: 3160, deliveredRate: 98.7, cost: 0 },
      ],
      cost: { total: 2.84, planned: 2.84 },
      totals: { delivered: 31320, bounced: 190, hardBounced: 150, softBounced: 40, complained: 12, deferred: 0, rejected: 0, opened: 0, openedHuman: 0, openedMachine: 0, clicked: 0, unsubscribed: 0 },
    })
    expect(finishedText(s)).toEqual({
      title: 'Weekly Digest #142 finished at 09:34:12',
      body: '31,604 sent in 34 minutes: 31,320 delivered (99.1%), 190 bounced, 12 complaints. No threshold tripped. Opens and clicks keep updating in the report for 72 hours.',
    })
    expect(finalTiles(s).map((t) => `${t.label} ${t.value} ${t.sub}`)).toEqual([
      'Delivered 31,320 99.1%',
      'Bounced 190 0.60% · added to suppression',
      'Complained 12 0.04%',
      'Cost $2.84 SES $2.84 · Postal $0',
    ])
    expect(providerResultLines(s)).toEqual(['SES 28,444 · 99.2%', 'Postal 3,160 · 98.7%'])
  })
})
