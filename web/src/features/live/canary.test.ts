import { setDisplayTimeZone } from '@/lib/format'
import {
  canaryActive,
  canaryMailboxRows,
  canaryMinNote,
  canaryProgress,
  canarySentLine,
  canarySummary,
  canaryTiles,
  canaryTitle,
  canaryWhy,
  gateRow,
  minutesLeft,
  seedRows,
  untilPlus,
  type Canary,
  type Gate,
} from './canary'
import type { Mailbox, Snapshot } from './model'

/*
 * The canary's words, pinned against LiveMonitorCanary with its sample
 * data: 1,560 most-engaged recipients (5%) plus 10 seed mailboxes, sent
 * 09:00:04 to 09:05:12, watched until 09:25:12. Times are UTC here.
 */

const T = (hms: string) => `2026-10-14T${hms}Z`

function gate(g: Partial<Gate> & Pick<Gate, 'key'>): Gate {
  return { limit: 1, events: 0, base: 0, minBase: 0, state: 'passing', ...g }
}

const placement = {
  inbox: 8,
  promotions: 2,
  spam: 0,
  pending: 0,
  missing: 0,
  byMailbox: [
    { mailbox: 'gmail' as const, seeds: 3, inbox: 1, promotions: 2, spam: 0 },
    { mailbox: 'outlook' as const, seeds: 2, inbox: 2, promotions: 0, spam: 0 },
    { mailbox: 'icloud' as const, seeds: 1, inbox: 1, promotions: 0, spam: 0 },
  ],
}

const watching: Canary = {
  phase: 'watching',
  chosen: false,
  reasons: ['never_mailed'],
  percent: 5,
  size: 1560,
  sent: 1560,
  seeds: 10,
  queued: 29632,
  neverMailed: 2118,
  usual: 30800,
  domain: 'news.acme.io',
  startedAt: T('09:00:04'),
  sentAt: T('09:05:12'),
  until: T('09:25:12'),
  watchMinutes: 20,
  extended: 0,
  gates: [
    gate({ key: 'bounce', value: 0.32, limit: 1, events: 5, base: 1560 }),
    gate({ key: 'block_bounce', value: 0, limit: 0.2 }),
    gate({ key: 'deferral', value: 3.97, limit: 15, events: 62, base: 1560 }),
    gate({
      key: 'complaint',
      limit: 0.1,
      state: 'collecting',
      mailbox: 'outlook',
      base: 312,
      minBase: 500,
      reported: ['outlook', 'yahoo', 'other'],
      notReported: ['gmail', 'icloud'],
    }),
    gate({ key: 'seeds', limit: 2, base: 10, seeds: placement }),
  ],
  placement,
  mainSeconds: 1500,
}

const s = {} as Snapshot

beforeAll(() => setDisplayTimeZone('UTC'))

describe('LiveMonitorCanary', () => {
  it('knows when the canary is still deciding', () => {
    expect(canaryActive(watching)).toBe(true)
    expect(canaryActive({ ...watching, phase: 'held' })).toBe(true)
    expect(canaryActive({ ...watching, phase: 'passed' })).toBe(false)
    expect(canaryActive(undefined)).toBe(false)
  })

  it('writes the heading and the sent line', () => {
    const now = new Date(T('09:13:00'))
    expect(minutesLeft(watching.until, now)).toBe(13)
    expect(canaryTitle(watching, now)).toBe('Canary · 1,560 most-engaged recipients (5%) plus 10 seed mailboxes · watching until 09:25 (13 min left)')
    expect(canarySentLine(watching)).toBe('Sent 09:00:04 to 09:05:12. 29,632 recipients are queued until the canary passes.')
    expect(canaryWhy(watching)).toBe('2,118 recipients have never had mail from news.acme.io.')
  })

  it('says how the gates stand in one line', () => {
    expect(canarySummary(watching)).toBe('4 of 5 gates are passing and none has failed; complaints are still collecting evidence.')
    const seedsOnly = { ...watching, gates: [gate({ key: 'bounce' }), gate({ key: 'seeds', state: 'collecting', base: 10 })] }
    expect(canarySummary(seedsOnly)).toBe('1 of 2 gates are passing and none has failed; seed placement is still collecting evidence.')
    const held = { ...watching, phase: 'held' as const, gates: [gate({ key: 'bounce', value: 2.4, events: 38, base: 1560, state: 'failing' })] }
    expect(canarySummary(held)).toBe('The bounces gate failed: the send is held until someone continues or stops it.')
    expect(canaryTitle(held, new Date())).toBe('Canary held: Bounces failed its gate')
  })

  it('writes each gate with its evidence', () => {
    const rows = watching.gates.map((g) => gateRow(g, s))
    const row = (key: string) => {
      const r = rows.find((x) => x.key === key)
      if (!r) throw new Error(`no ${key} gate`)
      return r
    }
    const bounce = row('bounce')
    const block = row('block_bounce')
    const deferral = row('deferral')
    const complaint = row('complaint')
    const seeds = row('seeds')
    expect([bounce.name, bounce.value, bounce.limit, bounce.state.text]).toEqual(['Bounces', '0.3% · 5 of 1,560', 'limit 1.0%', 'Passing'])
    expect([block.name, block.value]).toEqual(['Block/policy bounces', '0'])
    expect([deferral.value, deferral.limit]).toEqual(['4.0% · 62 of 1,560', 'limit 15%'])
    expect(complaint.value).toBe('n/a for Gmail and iCloud · 0 from Outlook, Yahoo and Other')
    expect(complaint.note).toBe('312 of 500 delivered to Outlook before this can trip')
    expect([complaint.limit, complaint.state.text, complaint.state.tone]).toEqual(['limit 0.10% per mailbox provider', 'Collecting evidence', 'neutral'])
    expect([seeds.name, seeds.value, seeds.limit]).toEqual(['Seed placement', 'Inbox 8 · Promotions 2 · Spam 0', 'limit 2 in spam'])
  })

  it('lists the seeds per mailbox provider', () => {
    expect(seedRows(placement)).toEqual([
      { key: 'gmail', name: 'Gmail', note: 'incl. Google Workspace · 3 seeds', value: 'Inbox 1 · Promotions 2 · Spam 0' },
      { key: 'outlook', name: 'Outlook', note: 'incl. Microsoft 365 · 2 seeds', value: 'Inbox 2 · Spam 0' },
      { key: 'icloud', name: 'iCloud', note: '1 seed', value: 'Inbox 1 · Spam 0' },
    ])
  })

  it('watches 10 more minutes from the end of the watch', () => {
    expect(untilPlus(watching, new Date(T('09:13:00'))).toISOString()).toBe('2026-10-14T09:35:12.000Z')
    // After the watch ended, from now.
    expect(untilPlus(watching, new Date(T('09:30:00'))).toISOString()).toBe('2026-10-14T09:40:00.000Z')
  })

  it("writes the canary's numbers as the board does", () => {
    const mb = (key: Mailbox['key'], m: Partial<Mailbox>): Mailbox => ({
      key,
      name: key,
      sent: 0,
      remaining: 0,
      delivered: 0,
      bounced: 0,
      complained: 0,
      opened: 0,
      deferred: 0,
      retrying: 0,
      held: false,
      status: 'sending',
      ...m,
    })
    const snap = {
      sent: 1560,
      recipients: 31192,
      retrying: 16,
      thresholds: [],
      providers: [
        { id: 'ses', name: 'Amazon SES', short: 'SES', color: 'ses', sent: 1404, remaining: 26669 },
        { id: 'postal', name: 'Postal', short: 'Postal', color: 'postal', sent: 156, remaining: 2963 },
      ],
      mailboxes: [
        mb('gmail', { sent: 718, delivered: 717, deliveredRate: 99.9, deferred: 7, bounceRate: 0.1, complaintReason: 'not_reported' }),
        mb('outlook', { sent: 328, delivered: 312, deliveredRate: 95.1, retrying: 14, deferred: 21, bounceRate: 0.6, throttle: 5 }),
      ],
      totals: {
        delivered: 1539,
        bounced: 5,
        hardBounced: 3,
        unknownBounced: 3,
        blockBounced: 0,
        softBounced: 2,
        complained: 0,
        deferred: 62,
        rejected: 0,
        opened: 498,
        openedHuman: 301,
        openedMachine: 197,
        clicked: 0,
        unsubscribed: 0,
        complaintBase: 0,
        notReported: ['gmail', 'icloud'],
      },
    } as unknown as Snapshot
    const tiles = canaryTiles(snap)
    expect(tiles.map((x) => x.sub)).toContain('98.7% of the canary · 16 retrying')
    expect(tiles.find((x) => x.key === 'deferred')?.sub).toBe('4.0% · 46 since delivered, 16 retrying')
    expect(canaryProgress(watching, snap, new Date(T('09:13:00'))).queued).toBe('Queued 29,632 · SES 26,669 · Postal 2,963')
    const [gmail, outlook] = canaryMailboxRows(snap)
    expect(gmail).toMatchObject({ sent: '718', delivered: '99.9%', deferred: '1.0%', bounce: '0.1%', complaint: 'n/a', na: true, rate: 'full rate' })
    expect(gmail?.state.text).toBe('Watching')
    expect(outlook).toMatchObject({ delivered: '95.1% · 14 retrying', deferred: '6.4%', complaint: '0 so far', rate: '5 /s' })
    expect(outlook?.state.text).toBe('Collecting evidence')
    expect(canaryMinNote(snap)).toBe("A mailbox provider's thresholds can trip only after 500 delivered to it; in the canary only Gmail has that many.")
  })
})
