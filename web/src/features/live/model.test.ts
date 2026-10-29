import { setDisplayTimeZone } from '@/lib/format'
import {
  abLine,
  deliveredMix,
  errorRate,
  failoverCaption,
  finalTiles,
  finishedText,
  freshness,
  headerBadges,
  inDoubtIncident,
  inDoubtRows,
  inDoubtText,
  inDoubtWaiting,
  incidentTitle,
  lostText,
  mailboxStatus,
  mobileEta,
  mobileTiles,
  moveFacts,
  optionAction,
  outageSummary,
  outageTitle,
  pausedText,
  progressLegend,
  progressText,
  providerLine,
  providerNote,
  providerResultLines,
  providerStatus,
  retriesNote,
  sentLine,
  startedLine,
  startingText,
  routeDoneLine,
  noticeText,
  openNotices,
  summaryLine,
  thresholdEvidence,
  thresholdRows,
  thresholdSummary,
  tiles,
  timingLine,
  trippedFacts,
  type Incident,
  type Mailbox,
  type Provider,
  type Snapshot,
  type Threshold,
  queuedCount,
  warmupHeld,
  whenText,
  whoLine,
} from './model'
import { currentRates, finishSeconds, nearestLevel } from './RatesPanel'

/*
 * The live monitor's words and numbers, pinned against the LiveMonitor boards
 * with their sample data. Times are UTC here.
 */

const T = (hms: string) => `2026-10-14T${hms}Z`

/** A thresholds card row with its evidence, as the API sends it. */
function thr(key: Threshold['key'], value: number | undefined, limit: number, extra: Partial<Threshold> = {}): Threshold {
  // Complaints and block/policy bounces also trip clearly over the limit at any volume (review R2.L7c).
  const anyVolume = key === 'complaint' || key === 'mailbox_complaint' || key === 'block_bounce'
  return { key, value, limit, events: 0, base: 0, minEvents: 5, minBase: 500, anyVolume, state: 'ok', ...extra }
}

/** Totals, with what complaint rates are counted on (F-2) and bounces by type (GS-11). */
function totals(
  t: Omit<Snapshot['totals'], 'complaintBase' | 'notReported' | 'blockBounced' | 'unknownBounced'> & Partial<Snapshot['totals']>,
): Snapshot['totals'] {
  return { complaintBase: 0, notReported: [], blockBounced: 0, unknownBounced: t.hardBounced, ...t }
}

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
    infersDelivery: false,
    host: '',
    ip: '',
    cost: 0,
    plannedCost: 0,
    retrying: false,
    probing: false,
    probes: 0,
    probesNeeded: 3,
    probeSeed: false,
    idempotent: false,
    ...p,
  }
}

function mailbox(m: Partial<Mailbox>): Mailbox {
  return { key: 'gmail', name: 'Gmail', sent: 0, remaining: 0, delivered: 0, bounced: 0, complained: 0, opened: 0, deferred: 0, retrying: 0, held: false, status: 'ok', ...m }
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
    retrying: 0,
    unsent: 0,
    percent: 0,
    rate: 0,
    totals: totals({ delivered: 0, bounced: 0, hardBounced: 0, softBounced: 0, complained: 0, deferred: 0, rejected: 0, opened: 0, openedHuman: 0, openedMachine: 0, clicked: 0, unsubscribed: 0 }),
    mailboxes: [],
    providers: [],
    thresholds: [],
    thresholdScope: 'route',
    limits: { bounce: 2, complaint: 0.1, mailboxComplaint: 0.1, deferral: 40, apiErrors: 5 },
    details: { from: 'digest@news.acme.io', audience: 'Weekly Digest', audienceCount: 31604, route: 'SES 90% · Postal 10%', cost: 1.76, costPlanned: 2.84, scheduledBy: 'Linh Nguyen', watchHours: 72 },
    deliveredMix: [],
    incidents: [],
    events: [],
    eventsVia: 'SNS and Postal webhooks',
    lastEventAt: T('09:21:44'),
    failovers: [],
    sharedStreams: [],
    excluded: { suppressed: 0, unsubscribed: 0, bounced: 0, complained: 0, paused: 0, role: 0, duplicates: 0, sunset: 0, excluded: 0, tiers: 0 },
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
  totals: totals({ delivered: 19402, bounced: 118, hardBounced: 94, unknownBounced: 86, blockBounced: 8, softBounced: 24, complained: 4, deferred: 40, rejected: 0, opened: 2811, openedHuman: 1920, openedMachine: 891, clicked: 402, unsubscribed: 0, complaintBase: 8895, notReported: ['gmail', 'icloud'] }),
  thresholds: [
    thr('bounce', 0.6, 2),
    thr('complaint', 0.03, 0.1),
    thr('mailbox_complaint', 0.05, 0.1, { mailbox: 'outlook' }),
    thr('api_errors', 0.2, 5),
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
    expect(headerBadges(live, undefined)).toEqual([{ tone: 'ok', text: 'Sending', live: true }])
    // People, not permission keys (review GS-14).
    expect(whoLine({ ...live, helpers: { pause: [], send: ['Linh Nguyen', 'Marcus Chen'], approve: [] } })).toBe(
      'Anyone who can send can pause · Linh Nguyen and Marcus Chen can resume',
    )
    expect(whoLine({ ...live, helpers: { pause: [], send: [], approve: ['Linh Nguyen', 'Marcus Chen'] } }, 'canary')).toBe(
      'Anyone who can send can stop it · Linh Nguyen and Marcus Chen can continue early',
    )
    expect(freshness(live, new Date(T('09:21:46')), 'live')).toEqual({ text: 'Live · updated 1 s ago', stale: false })
    expect(freshness(live, new Date(T('09:23:15')), 'polling')).toEqual({ text: 'Checking every 10 s · updated 90 s ago', stale: true })
  })

  it("starts a canary's send with who goes first (LiveMonitorStates)", () => {
    const starting = { ...live, canary: { phase: 'sending', size: 1560, seeds: 10, queued: 29632 } as Snapshot['canary'] }
    expect(startingText(starting)).toEqual({
      title: 'Starting the canary · 09:00:04',
      body: '1,560 most-engaged recipients and 10 seed mailboxes go first; the other 29,632 wait until the canary passes. Delivery events usually arrive within 10 seconds.',
    })
  })

  it('says when each route is done at current rates (review GS-8)', () => {
    const s = { ...live, providers: [{ ...ses, doneAt: T('09:51:10') }, { ...postal, doneAt: T('09:48:00') }] }
    expect(routeDoneLine(s)).toBe('SES done about 09:51 · Postal about 09:48')
    expect(timingLine(s)).toBe('Started 09:00:04 · at current rates, SES done about 09:51 · Postal about 09:48')
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
      'Bounced 118 0.60% · 86 unknown user, 8 block/policy, 24 soft',
      'Complained 4 0.04% of 8,895 delivered where complaints are reported',
      'Opened 2,811 14.5% · 1,920 human, 891 machine',
      'Clicked 402 2.1% · 14.3% click-to-open',
    ])
  })

  it('measures each threshold against its limit', () => {
    expect(thresholdRows(live).map((r) => [r.label, r.value, Math.round(r.width), r.color])).toEqual([
      ['Bounces', '0.60% of 2.0%', 30, 'var(--ok-bar)'],
      ['Complaints', '0.03% of 0.10%', 30, 'var(--ok-bar)'],
      ['Outlook complaints', '0.05% of 0.10%', 50, 'var(--warn-bar)'],
      ['Sending provider API errors', '0.2% of 5%', 4, 'var(--ok-bar)'],
    ])
    const failing = thresholdRows({ ...live, thresholds: [thr('api_errors', 100, 5)] })[0]!
    expect([failing.value, failing.color]).toEqual(['100% of 5%', 'var(--bad-strong)'])
  })

  // Review R2.L7c: complaints and block/policy bounces also trip from 3
  // once clearly over the limit, so "Collecting evidence" says so; other
  // rates wait for the floors alone.
  it('says what a rate still needs before it can pause', () => {
    const complaints = thr('mailbox_complaint', 0.67, 0.1, { mailbox: 'outlook', events: 2, base: 300, state: 'collecting' })
    expect(thresholdEvidence(complaints)).toBe('2 in 300 delivered · needs 5 complaints and 500 delivered to pause, or 3 clearly over the limit')
    const blocks = thr('block_bounce', 0.5, 0.5, { mailbox: 'outlook', events: 2, base: 400, state: 'collecting' })
    expect(thresholdEvidence(blocks)).toBe('2 in 400 sent · needs 5 block/policy bounces and 500 sent to pause, or 3 clearly over the limit')
    const bounces = thr('bounce', 2.5, 2, { events: 10, base: 400, minBase: 1000, state: 'collecting' })
    expect(thresholdEvidence(bounces)).toBe('10 in 400 sent · needs 5 bounces and 1,000 sent to pause')
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
    expect(mailboxStatus(mailbox({ key: 'outlook', name: 'Outlook', delivered: 4049, remaining: 2436, complaintRate: 0.05 }), live, false)).toEqual({ tone: 'ok', text: 'Sending' })
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
      'Bounced 118 0.60% · 8 block/policy',
      'Complained 4 0.04% where reported',
      'Opened · human 1,920 +891 machine',
      'Clicked 402 2.1% · 14.3% click-to-open',
    ])
    expect(mobileEta(live)).toBe('ETA 13 min · about 09:34')
    expect(progressLegend(live, undefined, true).map((l) => l.text)).toEqual(['SES 17,635', 'Postal 1,959', 'Queued 12,010'])
    expect(providerLine(ses)).toBe('14.2 /s · 99.1% delivered')
    expect(providerStatus(postal, true).text).toBe('Warm-up')
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
  mailboxes: [
    mailbox({ key: 'gmail', name: 'Gmail', sent: 9810, delivered: 9741, complaintReason: 'not_reported' }),
    mailbox({ key: 'outlook', name: 'Outlook', sent: 4115, remaining: 2522, delivered: 3980, complained: 7, complaintRate: 0.18, bounceRate: 1.1, held: true, heldSince: T('09:21:14') }),
    mailbox({ key: 'yahoo', name: 'Yahoo', sent: 1771, delivered: 1750, complained: 1 }),
    mailbox({ key: 'icloud', name: 'iCloud', sent: 2312, delivered: 2304, complaintReason: 'not_reported' }),
    mailbox({ key: 'other', name: 'Other', sent: 4432, delivered: 4296, complained: 1 }),
  ],
  continuing: ['Gmail', 'Yahoo', 'iCloud', 'Other'],
  totals: totals({ delivered: 22071, bounced: 154, hardBounced: 121, softBounced: 33, complained: 9, deferred: 0, rejected: 0, opened: 3604, openedHuman: 2480, openedMachine: 1124, clicked: 521, unsubscribed: 0, complaintBase: 10026, notReported: ['gmail', 'icloud'] }),
  thresholds: [
    thr('bounce', 0.69, 2),
    thr('complaint', 0.04, 0.1),
    thr('mailbox_complaint', 0.18, 0.1, { mailbox: 'outlook' }),
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
    expect(f.evidence).toEqual({ state: 'Tripped', text: '7 complaints in 3,980 delivered · enough evidence (needs 5 events and 500 delivered)' })
    expect(f.rows.map((r) => `${r.label} ${r.value}`)).toEqual(['Limit, per mailbox provider 0.10%', 'Yahoo and Other complaints 2 of 6,046 · 0.03%', 'Outlook bounces 1.1% of 2.0%'])
    expect(f.watch).toBe('Complaints keep arriving for up to 72 h; ReputeCast keeps watching after the send ends.')
  })

  it('writes progress, counters and badges', () => {
    expect(sentLine(paused, outlook)).toBe('22,440 of 31,604 sent · 2,522 held')
    expect(timingLine(paused, outlook)).toBe('Started 09:00:04 · Outlook held since 09:21:14 · ETA for the rest 9 min')
    expect(timingLine({ ...paused, providers: paused.providers.map((p) => ({ ...p, doneAt: p.id === 'ses' ? T('09:28:10') : T('09:29:30') })) }, outlook)).toBe(
      'Started 09:00:04 · Outlook held since 09:21:14 · the rest: SES done about 09:28 · Postal about 09:29',
    )
    expect(progressLegend(paused, outlook).map((l) => l.text)).toEqual(['Amazon SES 20,196 · 14.1 /s', 'Postal 2,244 · 2.0 /s', 'Held, Outlook 2,522', 'Queued 6,642'])
    const complained = tiles(paused, 'threshold', outlook)[2]!
    expect(`${complained.sub} ${complained.alert}`).toBe('Outlook 7 · Yahoo 1 · Other 1 · Gmail and iCloud n/a true')
    expect(tiles(paused, 'threshold', outlook)[4]!.sub).toBe('2.4%')
    expect(headerBadges(paused, outlook).map((b) => b.text)).toEqual(['Auto-paused · Outlook', 'Other mailbox providers still sending'])
  })

  it('maps each choice to the API', () => {
    expect(optionAction(outlook, 'resume_with_changes')).toEqual({ kind: 'resume', body: { mailbox: 'outlook', withChanges: true } })
    expect(optionAction(outlook, 'cancel_share')).toEqual({ kind: 'cancel', body: { mailbox: 'outlook' } })
  })

  // Review R2.L7c: a small send's complaints, clearly over the limit,
  // trip before the usual evidence; the decision view says so.
  it('says when a rate tripped clearly over the limit on a small send', () => {
    const small: Incident = { ...outlook, value: 3.33, detail: { ...outlook.detail, events: 10, base: 300, clear: true } }
    expect(trippedFacts(small, paused).evidence?.text).toBe("10 complaints in 300 delivered · clearly over the limit, so it didn't wait for 500 delivered")
    // It names only the evidence it didn't have.
    const fewer: Incident = { ...small, value: 0.5, detail: { ...small.detail, events: 3, base: 600 } }
    expect(trippedFacts(fewer, paused).evidence?.text).toBe("3 complaints in 600 delivered · clearly over the limit, so it didn't wait for 5 events")
    const both: Incident = { ...small, value: 1.5, detail: { ...small.detail, events: 3, base: 200 } }
    expect(trippedFacts(both, paused).evidence?.text).toBe("3 complaints in 200 delivered · clearly over the limit, so it didn't wait for 5 events and 500 delivered")
  })

  // Review GS-11: block/policy bounces read in words, and a burst says
  // the minutes it counted.
  it('names block/policy bounces and their burst', () => {
    const burst: Incident = {
      ...outlook,
      metric: 'block_bounce',
      value: 1.33,
      limit: 0.5,
      detail: { ...outlook.detail, events: 8, base: 600, window: 10, clear: true },
    }
    const blocks = snapshot({ ...paused, thresholds: [...paused.thresholds, thr('block_bounce', 0.4, 0.5, { mailbox: 'outlook' })] })
    expect(incidentTitle(burst)).toBe('Outlook block/policy bounce rate reached 1.3% (limit 0.5%)')
    expect(thresholdSummary(burst, blocks)).toContain('8 block/policy bounces out of 600 sent within 10 minutes.')
    const f = trippedFacts(burst, blocks)
    expect(f.head.label).toBe('Outlook block/policy bounces')
    // A burst can have the usual evidence within its window; it says the time instead.
    expect(f.evidence?.text).toBe('8 block/policy bounces in 600 sent within 10 minutes · clearly over the limit in that time')
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
  totals: totals({ delivered: 22834, bounced: 139, hardBounced: 110, softBounced: 29, complained: 7, deferred: 0, rejected: 0, opened: 3781, openedHuman: 2590, openedMachine: 1191, clicked: 548, unsubscribed: 0, complaintBase: 10350, notReported: ['gmail', 'icloud'] }),
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
    expect(timingLine(down, outage)).toBe('Started 09:00:04 · SES finishes about 09:27 · Postal held since 09:17:10')
    expect(progressLegend(down, outage).map((l) => l.text)).toEqual(['Amazon SES 21,104 · 14.0 /s', 'Postal 1,956 · 0 /s', 'Held on Postal 1,204', 'Queued on SES 7,340'])
    expect(tiles(down, 'outage', outage).map((t) => t.sub)).toEqual(['99.0% of sent', '0.60%', '0.07% where reported · Gmail and iCloud n/a', '16.6% · 2,590 human, 1,191 machine', '2.4%'])
    expect(headerBadges(down, outage).map((b) => b.text)).toEqual(['Postal route held · waiting for your decision', 'SES still sending'])
    expect(providerNote(down.providers[1]!, true)).toBe('mta.acme.io · 203.0.113.25')
    expect(providerStatus(down.providers[1]!)).toEqual({ tone: 'bad', text: 'Down since 09:14' })
    expect(failoverCaption(down)).toBe('Automations and System already failed over to Postmark and SES on their own')
    expect(failoverCaption({ ...down, failovers: [] })).toBe("Automations and System don't use Postal")
    expect(failoverCaption({ ...down, failovers: [], sharedStreams: ['Automations'] })).toBe('Automations also uses Postal')
    expect(optionAction(outage, 'move')).toEqual({ kind: 'move', body: { fromProviderId: 'postal', toProviderId: 'ses' } })
  })

  // Review GS-3: Retry keeps the share held and sends a test message a
  // minute; three in a row resume it, 15 minutes of failures ask again.
  it('shows a retried share testing the provider', () => {
    const retrying = snapshot({
      ...down,
      incident: undefined,
      providers: [down.providers[0]!, { ...down.providers[1]!, retrying: true, probes: 1, probesNeeded: 3, probeSeed: true }],
    })
    expect(headerBadges(retrying, undefined).map((b) => b.text)).toEqual(['Postal route held · retrying every 60 s', 'SES still sending'])
    expect(providerStatus(retrying.providers[1]!)).toEqual({ tone: 'warn', text: 'Retrying · 1 of 3 tests passed' })
    const again: Incident = { ...outage, detail: { ...outage.detail, retried: true } }
    expect(outageTitle(again)).toBe('Postal is still down after 15 minutes of retrying')
  })

  // Review GS-3 (R2.A5b): until someone decides, the held share tests its
  // provider by itself; a decision still wins.
  it('says the held share tests its provider until someone decides', () => {
    const probing = snapshot({
      ...down,
      providers: [down.providers[0]!, { ...down.providers[1]!, probing: true, probes: 2, probesNeeded: 3, probeSeed: true }],
    })
    expect(headerBadges(probing, outage).map((b) => b.text)).toEqual(['Postal route held · testing every 60 s until you decide', 'SES still sending'])
    expect(providerStatus(probing.providers[1]!)).toEqual({ tone: 'warn', text: 'Testing · 2 of 3 tests passed' })
    expect(outageSummary(outage, probing)).toBe(
      'From 09:14:02 to 09:17:10, 37 send attempts to mta.acme.io failed and 0 succeeded, so the Postal share of this campaign is on hold: 1,956 sent, 1,204 remaining. Amazon SES keeps sending at 14 /s, 73% complete overall. Nothing moved automatically because Broadcast failover is set to pause and ask. Until you decide, ReputeCast tests Postal with 1 message to a seed mailbox every 60 s and sends the share again after 3 go through in a row (2 so far).',
    )
    const own = snapshot({ ...probing, providers: [probing.providers[0]!, { ...probing.providers[1]!, probes: 0, probeSeed: false }] })
    expect(outageSummary(outage, own)).toContain('Until you decide, ReputeCast tests Postal with 1 of these messages every 60 s and sends the share again after 3 go through in a row.')
  })

  // Review R2.L2: what a provider's idempotency really guarantees, and for how long.
  it('says how long the provider recognizes a retry, or that it cannot', () => {
    expect(retriesNote(outage, down)).toBe(
      "Retries used exponential back-off (5 s to 60 s). Postal can't recognize a message it already took, so a message goes again only after Postal refused it; one whose answer was lost is held as in doubt, not sent twice.",
    )
    const keyed = snapshot({ ...down, providers: [down.providers[0]!, { ...down.providers[1]!, idempotent: true, idempotencySeconds: 86400 }] })
    expect(retriesNote(outage, keyed)).toBe(
      'Retries used exponential back-off (5 s to 60 s). Each attempt carries an idempotency key, and Postal recognizes a message it already took for 24 h after the first try, so a retry in that time is never delivered twice. After it, a message whose answer was lost is held as in doubt.',
    )
  })
})

// Review GS-1 (R2.L3): messages whose provider call ended without an
// answer. Nobody knows whether the provider took them, so a person decides.
describe('Messages in doubt', () => {
  const doubt: Incident = {
    id: 'i9',
    kind: 'in_doubt',
    scope: 'campaign',
    mailbox: '',
    mailboxName: '',
    providerId: '',
    provider: '',
    providerShort: '',
    metric: 'in_doubt',
    held: 15,
    trippedAt: T('09:20:02'),
    resolution: '',
    detail: {
      succeeded: 0,
      remaining: 15,
      options: [
        { key: 'resend', title: 'Send them again', body: 'Where the provider no longer recognizes a message it already took, anyone whose copy it did take gets the email twice.', action: 'Send 15 messages again', recommended: false },
        { key: 'leave_out', title: 'Leave them out (recommended)', body: 'They\'re marked "not sent, in doubt" in the report.', action: 'Leave 15 messages out', recommended: true },
      ],
    },
  }
  const shares: NonNullable<Snapshot['inDoubt']>['providers'] = [
    { providerId: 'postal', provider: 'Postal', short: 'Postal', count: 12, since: T('09:14:02'), idempotent: false, safe: false },
    { providerId: 'sp', provider: 'SparkPost', short: 'SparkPost', count: 3, since: T('09:14:02'), idempotent: true, windowSeconds: 86400, safeUntil: '2026-10-15T09:09:02Z', safe: true },
  ]
  const sending = snapshot({
    sent: 23060,
    queued: 7340,
    startedAt: T('09:00:04'),
    providers: [{ ...ses, sent: 21104, remaining: 7340, rate: 14 }],
    inDoubt: { count: 15, since: T('09:14:02'), providers: shares, safe: false, waiting: false, incidentId: 'i9' },
    incidents: [doubt],
  })
  const waiting = snapshot({
    ...sending,
    status: 'paused',
    state: 'paused',
    sent: 31589,
    queued: 0,
    options: { canPause: false, canResume: false, canCancel: true },
    inDoubt: { ...sending.inDoubt!, waiting: true },
  })

  it('says what is in doubt and what each provider recognizes', () => {
    expect(inDoubtIncident(sending)?.id).toBe('i9')
    expect(inDoubtText(sending)).toEqual({
      title: '15 messages in doubt',
      body: "ReputeCast can't tell whether Postal and SparkPost took them: the call that handed them over ended without an answer. They aren't sent again unless someone decides to, so nobody gets the email twice. The rest of the send goes on meanwhile; decide whenever you are ready.",
    })
    expect(inDoubtRows(sending)).toEqual([
      {
        key: 'postal',
        title: 'Postal · 12 messages · handed over at 09:14:02',
        text: "Postal can't recognize a message it already took, so sending them again can deliver a copy twice.",
        badge: { tone: 'warn', text: 'Could arrive twice' },
      },
      {
        key: 'sp',
        title: 'SparkPost · 3 messages · handed over at 09:14:02',
        text: 'SparkPost recognizes a message it already took for 24 h after the first try, so sending them again before tomorrow 09:09 delivers none twice.',
        badge: { tone: 'ok', text: 'Safe to send again' },
      },
    ])
    // Once the provider's window is over, it says so.
    const late = { ...sending, inDoubt: { ...sending.inDoubt!, providers: [{ ...shares[1]!, safe: false }] } }
    expect(inDoubtRows(late)[0]!.text).toBe(
      'SparkPost recognizes a message it already took for 24 h after the first try. For these that time is up, so sending them again can deliver a copy twice.',
    )
    // A resolved incident, or a finished send, has nothing to decide.
    expect(inDoubtIncident({ ...sending, incidents: [{ ...doubt, resolvedAt: T('09:30:00') }] })).toBeUndefined()
    expect(inDoubtIncident({ ...sending, state: 'finished' })).toBeUndefined()
  })

  it('maps the two choices to the API', () => {
    expect(optionAction(doubt, 'resend')).toEqual({ kind: 'inDoubt', body: { action: 'resend' } })
    expect(optionAction(doubt, 'leave_out')).toEqual({ kind: 'inDoubt', body: { action: 'leave_out' } })
    expect(optionAction(doubt, 'resume')).toBeNull()
  })

  it('shows them beside what keeps sending', () => {
    expect(inDoubtWaiting(sending)).toBe(false)
    expect(headerBadges(sending, undefined).map((b) => b.text)).toEqual(['Sending', '15 in doubt'])
    expect(progressLegend(sending).map((l) => l.text)).toEqual(['Amazon SES 21,104 · 14.0 /s', 'In doubt 15', 'Queued 7,340'])
  })

  it('waits for the decision once nothing else is left', () => {
    expect(inDoubtWaiting(waiting)).toBe(true)
    expect(headerBadges(waiting, undefined).map((b) => b.text)).toEqual(['Waiting for your decision · 15 in doubt'])
    expect(summaryLine(waiting).text).toBe('Everything else is sent. 15 messages in doubt wait for your decision.')
    expect(inDoubtText(waiting).title).toBe('15 messages in doubt wait for your decision')
    expect(inDoubtText(waiting).body).toContain('Everything else is sent; the send ends once you decide.')
    expect(mobileEta(waiting)).toBe('Waiting for your decision')
    // A send a person paused isn't waiting for it: Resume still works there.
    const paused = { ...waiting, pausedBy: 'Linh Nguyen', inDoubt: { ...waiting.inDoubt!, waiting: false } }
    expect(inDoubtWaiting(paused)).toBe(false)
    expect(headerBadges(paused, undefined).map((b) => b.text)).toEqual(['Paused by Linh Nguyen', '15 in doubt'])
  })
})

// Review PR-8 with GS-14: a share today's warm-up cap holds shows like the
// other holds, and it goes on with tomorrow's cap, which the ETA counts.
describe('Warm-up hold', () => {
  const waiting = provider({
    ...postal,
    routeStatus: 'held',
    rate: 0,
    remaining: 1201,
    doneAt: '2026-10-15T00:01:00Z',
    warmupSent: 6000,
    warmupHold: { planId: 'plan', held: 1201, since: T('09:20:00'), until: '2026-10-15T00:00:00Z', paused: false, cap: 6000, targets: [] },
  })
  const s = snapshot({
    ...live,
    state: 'partial',
    queued: 10809,
    held: 1201,
    finishesAt: '2026-10-15T00:01:00Z',
    providers: [{ ...ses, doneAt: T('09:34:00') }, waiting],
    providersHeld: ['Postal'],
  })

  it('says which route waits, until when, and that the rest keeps sending', () => {
    expect(summaryLine(s)).toEqual({ tone: 'warn', held: true, text: "Postal share waits for tomorrow's warm-up cap from 00:00. Amazon SES keeps sending." })
    expect(headerBadges(s, undefined).map((b) => b.text)).toEqual(['Postal route waits for the warm-up cap', 'SES still sending'])
    expect(providerStatus(waiting)).toEqual({ tone: 'warn', text: "Waits for tomorrow's cap" })
    const paused = { ...waiting, warmupHold: { ...waiting.warmupHold!, paused: true, until: undefined } }
    expect(providerStatus(paused)).toEqual({ tone: 'warn', text: 'Waits for the warm-up plan' })
    expect(summaryLine({ ...s, providers: [paused] }).text).toBe('Postal share waits for its warm-up plan to resume.')
  })

  it('counts the wait for tomorrow in when the route is done', () => {
    expect(routeDoneLine(s)).toBe('SES done about 09:34 · Postal about tomorrow 00:01')
    expect(whenText('2026-10-14T21:05:00Z', s.at)).toBe('21:05')
    expect(whenText('2026-10-16T00:01:00Z', s.at)).toBe('Fri 16 Oct 00:01')
    expect(mobileEta({ ...s, etaSeconds: 52_000 })).toContain('tomorrow 00:01')
  })

  it('keeps the held recipients out of what is queued now', () => {
    expect(warmupHeld(s)).toBe(1201)
    expect(queuedCount(s)).toBe(10809)
    expect(sentLine(s)).toBe('19,594 of 31,604 sent · 1,201 held on Postal')
    expect(progressLegend(s).map((l) => l.text)).toEqual(['Amazon SES 17,635 · 14.2 /s', 'Postal 1,959 · 0 /s', 'Held on Postal 1,201', 'Queued 10,809'])
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

  it('names who can resume a paused send, never a permission key (GS-14, X-1)', () => {
    const s = snapshot({ state: 'paused', pausedBy: 'Sofia Rossi', pausedAt: T('09:21:10'), held: 0, queued: 18000, waiting: 0 })
    expect(pausedText({ ...s, helpers: { send: ['Linh Nguyen', 'Marcus Chen'], approve: [], pause: [] } })).toEqual({
      title: 'Paused by Sofia Rossi at 09:21:10',
      body: '18,000 recipients are waiting. Nothing more is sent until Linh Nguyen or Marcus Chen resumes it; what was already sent keeps reporting opens and clicks.',
    })
    const anyone = pausedText({ ...s, helpers: undefined, pauseReason: 'Checking a broken link.' })
    expect(anyone.body).toBe(
      '"Checking a broken link." 18,000 recipients are waiting. Nothing more is sent until someone who can send resumes it; what was already sent keeps reporting opens and clicks.',
    )
    expect(anyone.body).not.toMatch(/campaigns:/)
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
      totals: totals({ delivered: 31320, bounced: 190, hardBounced: 150, softBounced: 40, complained: 12, deferred: 0, rejected: 0, opened: 0, openedHuman: 0, openedMachine: 0, clicked: 0, unsubscribed: 0, complaintBase: 14161, notReported: ['gmail', 'icloud'] }),
      watch: { active: true, alert: false, hours: 72, limit: 0.1, rows: [], until: '2026-10-17T09:34:12Z' },
    })
    expect(finishedText(s)).toEqual({
      title: 'Weekly Digest #142 finished at 09:34:12',
      body: '31,604 sent in 34 minutes: 31,320 delivered (99.1%), 190 bounced, 12 complaints so far. No threshold tripped. Opens and clicks keep updating in the report for 72 hours.',
    })
    expect(finalTiles(s).map((t) => `${t.label} ${t.value} ${t.sub}`)).toEqual([
      'Delivered 31,320 99.1%',
      'Bounced 190 0.60% · added to suppression',
      'Complained 12 0.08% of 14,161 delivered where reported · still counting',
      'Cost $2.84 SES $2.84 · Postal $0',
    ])
    expect(providerResultLines(s)).toEqual(['SES 28,444 · 99.2%', 'Postal 3,160 · 98.7%'])
  })
})

// Adjust rates (LiveMonitorStates, review GS-13): live view at 09:38:44.
describe('Adjust rates', () => {
  const s = snapshot({
    at: T('09:38:44'),
    providers: [
      { ...ses, remaining: 10438, rate: 14.2 },
      { ...postal, remaining: 1160, rate: 2 },
    ],
    mailboxes: [
      mailbox({ key: 'gmail', name: 'Gmail', remaining: 5335 }),
      mailbox({ key: 'outlook', name: 'Outlook', remaining: 2436, throttle: 5 }),
      mailbox({ key: 'yahoo', name: 'Yahoo', remaining: 1044, throttle: 4 }),
      mailbox({ key: 'icloud', name: 'iCloud', remaining: 928, throttle: 3 }),
      mailbox({ key: 'other', name: 'Other', remaining: 1855 }),
    ],
  })

  it('works out when the whole send finishes at given rates', () => {
    expect(currentRates(s)).toEqual({ gmail: 0, outlook: 5, yahoo: 4, icloud: 3, other: 0 })
    // SES's 10,438 at 14.2 /s outlast Outlook's 2,436 at 5 /s: done about 09:51.
    expect(Math.round(finishSeconds(s, currentRates(s)))).toBe(735)
    // At 2 /s Outlook is last: 2,436 / 2 = 1,218 s, about 09:59.
    expect(Math.round(finishSeconds(s, { ...currentRates(s), outlook: 2 }))).toBe(1218)
    expect([nearestLevel(2.5), nearestLevel(7), nearestLevel(0), nearestLevel(100)]).toEqual([2, 6, 0, 12])
  })

  it('says the send state in one line', () => {
    expect(summaryLine(live).text).toBe('Sending to everyone; every threshold is inside its limit.')
    const passed = { ...live, canary: { phase: 'passed', decidedAt: T('09:20:04') } as Snapshot['canary'] }
    expect(summaryLine(passed).text).toBe('Canary passed at 09:20. Sending to everyone; every threshold is inside its limit.')
    expect(summaryLine(paused, outlook).text).toBe(
      'Outlook paused at 09:21:14 and waits for your decision. Gmail, Yahoo, iCloud and Other are still sending.',
    )
    expect(summaryLine(down, outage).text).toBe('Postal share held since 09:17:10 and waiting for your decision. Amazon SES keeps sending.')
  })
})

// Notices (review GS-11, GS-12, CO-4): acknowledged, not decided.
describe('Notices', () => {
  const slowed: Incident = {
    id: 'n1',
    kind: 'threshold',
    scope: 'mailbox_provider',
    mailbox: 'yahoo',
    mailboxName: 'Yahoo',
    metric: 'deferral',
    provider: '',
    providerId: '',
    providerShort: '',
    resolution: '',
    held: 0,
    trippedAt: T('09:30:00'),
    detail: {
      succeeded: 0,
      action: 'slowed',
      causes: [{ lead: 'Yahoo deferred 18% over 10 minutes', body: 'ReputeCast slowed Yahoo to half rate (2 /s).' }],
      recommendations: ['Acknowledge once you have seen it.'],
    },
  }
  const render: Incident = {
    ...slowed,
    id: 'n2',
    kind: 'render_error',
    scope: 'campaign',
    mailbox: '',
    mailboxName: '',
    metric: 'render',
    detail: {
      succeeded: 0,
      action: 'alerted',
      causes: [{ lead: 'The template failed for 1 recipient', body: 'Their message wasn\'t sent; the others keep getting it.' }],
      log: [{ at: T('09:31:00'), bad: true, text: 'a···@gmail.com: output over 4 MB' }],
    },
  }

  it('lists the open notices with their words', () => {
    const s = { ...live, incidents: [slowed, render, { ...slowed, id: 'n3', resolvedAt: T('09:40:00') }, { ...slowed, id: 'n4', detail: { succeeded: 0 } }] }
    expect(openNotices(s).map((i) => i.id)).toEqual(['n1', 'n2'])
    expect(noticeText(slowed)).toEqual({
      title: 'Yahoo deferred 18% over 10 minutes',
      body: 'ReputeCast slowed Yahoo to half rate (2 /s). Acknowledge once you have seen it.',
      samples: [],
    })
    expect(noticeText(render).samples).toEqual(['a···@gmail.com: output over 4 MB'])
  })
})

// The thresholds card's reply rule row (LiveMonitor, review GS-12).
describe('Reply rules on the thresholds card', () => {
  it('says what the closest rule does at its count', () => {
    const row = thr('reply_rules', 1, 10, { events: 1, minEvents: 10, minBase: 3, mailbox: 'yahoo', action: 'pause', duration: 120, parts: [{ name: 'TSS04', key: 'yahoo' }] })
    expect(thresholdRows({ ...live, thresholds: [row] })[0]?.value).toBe('Yahoo TSS04: 1 of 10 replies')
    expect(thresholdEvidence(row)).toBe('at 10, Yahoo pauses for 2 h')
    expect(thresholdEvidence({ ...row, action: 'slow', duration: 30 })).toBe('at 10, Yahoo slows to half rate for 30 min')
    expect(thresholdEvidence({ ...row, action: 'pause_route', duration: 0 })).toBe('at 10, Yahoo pauses on that route')
  })
})
