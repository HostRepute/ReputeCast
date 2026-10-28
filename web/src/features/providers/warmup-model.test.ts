import { setDisplayTimeZone } from '@/lib/format'
import { audienceText, groupDays, rampColumns, rampHeight, tableRows, tiersText, type WarmupPlan, type WarmupRow } from './warmup-model'

beforeAll(() => setDisplayTimeZone('UTC'))

const row = (day: number, date: number, status: WarmupRow['status'], extra: Partial<WarmupRow> = {}): WarmupRow => ({
  day,
  date: `2026-10-${String(date).padStart(2, '0')}T00:00:00Z`,
  cap: 100 * day,
  sent: status === 'planned' ? undefined : 100 * day,
  tiers: ['active'],
  share: 10,
  shareChange: false,
  uses: [],
  status,
  judged: true,
  need: 80 * day,
  held: false,
  repeat: false,
  eligible: false,
  bounced: day,
  complained: 0,
  delivered: 50 * day,
  ...extra,
})

// The demo plan as ProvidersWarmup draws it: Sunday sent nothing (day 5
// repeated on Monday), day 6 tripped the bounce guard and the cap held at
// 2,600 for two healthy days, the plan staying on day 6 until the second
// one completed it (R3.L6d); today is day 9 and nothing is scheduled.
const demo: WarmupRow[] = [
  row(1, 1, 'counted', { cap: 200, sent: 200, bounced: 1, delivered: 100 }),
  row(2, 2, 'counted', { cap: 400, sent: 400, bounced: 2, delivered: 200 }),
  row(3, 3, 'counted', { cap: 700, sent: 700, bounced: 4, delivered: 350 }),
  row(4, 4, 'counted', { cap: 1200, sent: 1200, bounced: 6, delivered: 600 }),
  row(5, 5, 'not_counted', { cap: 1800, sent: 0, bounced: 0, delivered: 0, reason: 'short', judged: false }),
  row(5, 6, 'counted', { cap: 1800, sent: 1800, bounced: 7, delivered: 1667 }),
  row(6, 7, 'not_counted', { cap: 2600, sent: 2600, reason: 'guard', metric: 'bounce' }),
  row(6, 8, 'not_counted', { cap: 2600, sent: 2600, reason: 'held', held: true, healthy: 1 }),
  row(6, 9, 'counted', { cap: 2600, sent: 2600, held: true, healthy: 2 }),
  row(7, 10, 'counted', { cap: 3600, sent: 3600 }),
  row(8, 11, 'counted', { cap: 4800, sent: 4798 }),
  row(9, 12, 'today', { cap: 6000, sent: 0, judged: false }),
  row(9, 13, 'planned', { cap: 6000, repeat: true }),
  ...Array.from({ length: 12 }, (_, i) => row(10 + i, 14 + i, 'planned', { cap: 7500 + 1000 * i, eligible: i === 11 })),
]

describe('warm-up days table', () => {
  it('folds the plain first days and keeps the last five, today and the next days', () => {
    const rows = tableRows(demo)
    expect(rows.map((r) => (r.kind === 'group' ? `${r.group.firstDay}-${r.group.lastDay}` : `${r.row.day}${r.row.status[0]}`))).toEqual([
      '1-5',
      '6n',
      '6n',
      '6c',
      '7c',
      '8c',
      '9t',
      '9p',
      '10p',
      '11p',
      '12p',
      '21p',
    ])
  })

  it('adds the folded days up as the board does', () => {
    const g = groupDays(demo.slice(0, 6))
    expect(g).toMatchObject({ firstDay: 1, lastDay: 5, cap: 4300, sent: 4300, bounce: 0.5, complaint: 0, counted: true, tiers: ['active'] })
    expect(g.missed.map((r) => r.date)).toEqual(['2026-10-05T00:00:00Z'])
    // Without deliveries at mailbox providers that report complaints, the
    // complaint rate is unknown, not zero.
    expect(groupDays([row(1, 2, 'counted', { delivered: 0 }), row(2, 3, 'counted', { delivered: 0 })]).complaint).toBeUndefined()
  })

  it("never folds a guard trip or a held cap, and doesn't fold a single day", () => {
    const tripEarly = [row(1, 2, 'counted'), row(2, 3, 'not_counted', { reason: 'guard' }), ...demo.slice(2, 12)]
    expect(tableRows(tripEarly)[0]).toMatchObject({ kind: 'day' })
    const held = [row(1, 2, 'counted'), row(2, 3, 'not_counted', { reason: 'held', held: true, healthy: 1 }), ...demo.slice(2, 12)]
    expect(tableRows(held)[0]).toMatchObject({ kind: 'day' })
    const short = demo.slice(5, 12)
    expect(tableRows(short).every((r) => r.kind === 'day')).toBe(true)
  })
})

describe('warm-up ramp', () => {
  it('draws to scale and keeps a day that sent visible', () => {
    expect(rampHeight(48000, 48000)).toBe(100)
    expect(rampHeight(6000, 48000)).toBe(12.5)
    expect(rampHeight(200, 48000)).toBe(0.4)
    expect(rampHeight(10, 48000)).toBe(0.4)
    expect(rampHeight(0, 48000)).toBe(0)
  })

  it('shows what went out, the trip and the days ahead', () => {
    const plan = {
      rows: demo,
      ramp: [
        { day: 6, plannedCap: 2600, sent: 2600, slowed: true, today: false },
        { day: 7, plannedCap: 3600, sent: 3600, slowed: false, today: false },
        { day: 9, plannedCap: 6000, sent: 0, slowed: false, today: true },
        { day: 21, plannedCap: 48000, slowed: false, today: false },
      ],
    } as Pick<WarmupPlan, 'ramp' | 'rows'>
    const cols = rampColumns(plan)
    // Day 6 tripped the guard, then counted on its third date: the hold
    // kept day 6's own cap, so nothing of the schedule was held back.
    expect(cols[0]).toMatchObject({ day: 6, sent: 5.4, cap: 0, tripped: true })
    expect(cols[1]).toMatchObject({ day: 7, sent: 7.5, cap: 0, tripped: false })
    expect(cols[2]).toMatchObject({ day: 9, sent: 0, cap: 12.5, today: true })
    expect(cols[3]).toMatchObject({ day: 21, sent: 0, cap: 100 })
  })

  it('draws the part of the schedule a step back held back', () => {
    // Day 6 tripped with "step back": the cap held at day 5's 1,800 while
    // the plan stayed on day 6 (planned 2,600).
    const rows = [
      row(6, 7, 'not_counted', { cap: 2600, sent: 2600, reason: 'guard', metric: 'bounce' }),
      row(6, 8, 'not_counted', { cap: 1800, plannedCap: 2600, sent: 1800, reason: 'held', held: true, healthy: 1 }),
    ]
    const cols = rampColumns({ rows, ramp: [{ day: 6, plannedCap: 2600, sent: 1800, slowed: true, today: false }] } as Pick<WarmupPlan, 'ramp' | 'rows'>)
    expect(cols[0]).toMatchObject({ day: 6, sent: 69.2, cap: 30.8, tripped: true })
  })
})

describe('warm-up audience', () => {
  it('describes the audience of a day', () => {
    expect(tiersText(['active'])).toBe('Active tier')
    expect(tiersText(['active', 'cooling'])).toBe('Active + Cooling')
    expect(tiersText(['active', 'cooling', 'at_risk'])).toBe('All tiers except Sunset')
    expect(tiersText(['active', 'cooling', 'at_risk', 'sunset'])).toBe('All')
    expect(audienceText(row(10, 14, 'planned', { tiers: ['active', 'cooling'], uses: [{ name: 'Weekly Digest #142', count: 3119 }] }))).toBe(
      'Active + Cooling · Weekly Digest #142 uses 3,119',
    )
    expect(audienceText(row(12, 16, 'planned', { tiers: ['active', 'cooling', 'at_risk'], share: 15, shareChange: true }))).toBe('All tiers except Sunset · share rises to 15%')
    expect(audienceText(row(21, 27, 'planned', { tiers: ['active', 'cooling', 'at_risk', 'sunset'], share: 40, eligible: true }))).toBe('All · share 40%, eligible as failover')
  })
})
