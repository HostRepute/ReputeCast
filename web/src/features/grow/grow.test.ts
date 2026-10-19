import { describe, expect, it } from 'vitest'
import { growBlockDefs, growMarker } from './composer/blocks'
import { growEvents, withGrow, type GrowEvent } from './contact/ContactGrow'
import { trackPos } from './MilestoneTrack'
import { tileInitials } from './shared'

describe('the milestone track', () => {
  // Milestones sit evenly: 1, 3 and 10 at a third, two thirds and the end.
  const counts = [1, 3, 10]
  it('puts a count between its milestones', () => {
    expect(trackPos(counts, 0)).toBe(0)
    expect(trackPos(counts, 1)).toBeCloseTo(33.33, 1)
    // Sam: 2 counted is half way from 1 to 3.
    expect(trackPos(counts, 2)).toBeCloseTo(50, 1)
    expect(trackPos(counts, 3)).toBeCloseTo(66.67, 1)
    expect(trackPos(counts, 10)).toBe(100)
    expect(trackPos(counts, 23)).toBe(100)
  })
  it('handles no milestones', () => {
    expect(trackPos([], 5)).toBe(0)
  })
})

describe('newsletter tiles', () => {
  it('uses two initials without a leading "The"', () => {
    expect(tileInitials('Ops Weekly')).toBe('OW')
    expect(tileInitials('The Deliverability Desk')).toBe('DD')
    expect(tileInitials('Fabrikam Engineering Letter')).toBe('FE')
  })
})

describe("the composer's Grow blocks", () => {
  it('keeps "--" out of the marker, so the comment stays one comment', () => {
    const m = growMarker('referral', { texts: { many: 'You referred {{ referral.count }} -- thanks -->' } })
    expect(m.startsWith('<mj-raw><!-- rc:referral {')).toBe(true)
    expect(m.endsWith(' --></mj-raw>')).toBe(true)
    const json = m.slice('<mj-raw><!-- rc:referral '.length, -' --></mj-raw>'.length)
    expect(json.includes('--')).toBe(false)
    expect(JSON.parse(json)).toEqual({ texts: { many: 'You referred {{ referral.count }} -- thanks -->' } })
  })
  it('offers the Reward block in milestone emails only', () => {
    expect(growBlockDefs().map((d) => d.id)).toEqual(['rc-referral', 'rc-recommendations'])
    expect(growBlockDefs(true).map((d) => d.id)).toEqual(['rc-reward', 'rc-referral', 'rc-recommendations'])
    expect(growBlockDefs().every((d) => d.group === 'grow' && d.kind === 'section')).toBe(true)
  })
})

const ev = (kind: GrowEvent['kind'], at: string, more: Partial<GrowEvent> = {}): GrowEvent => ({
  kind,
  at,
  counted: 0,
  milestone: 0,
  partner: '',
  referrer: '',
  referrerId: '',
  reward: '',
  ...more,
})

describe("Grow on a contact's timeline", () => {
  const events = [
    ev('reward_sent', '2026-10-06T18:41:00Z', { milestone: 1, reward: 'Sticker pack' }),
    ev('milestone', '2026-10-06T18:40:00Z', { milestone: 1, reward: 'Sticker pack' }),
    ev('milestone', '2026-10-12T09:00:00Z', { milestone: 3, reward: 'The Acme Field Guide' }),
    ev('joined', '2026-09-29T18:02:00Z', { referrer: 'Sam Rivera' }),
  ]
  const g = { events }
  it('makes a reward sent with its milestone one event', () => {
    const all = growEvents(g, 'all')
    expect(all.map((e) => e.kind)).toEqual(['milestone', 'milestone', 'joined'])
    // Milestone 1's reward went out with it; milestone 3's didn't.
    expect(all[0]!.reward).toBe('Sticker pack')
    expect(all[1]!.reward).toBe('')
  })
  it('lists joins under Consent, and nothing under Deliveries', () => {
    expect(growEvents(g, 'consent').map((e) => e.kind)).toEqual(['joined'])
    expect(growEvents(g, 'deliveries')).toEqual([])
    expect(growEvents(undefined, 'all')).toEqual([])
  })
  it('puts the events among the others by date, within what is loaded', () => {
    const items = [
      { kind: 'item', at: '2026-10-13T09:00:00Z' },
      { kind: 'item', at: '2026-10-01T09:00:00Z' },
    ]
    const merged = withGrow(items, growEvents(g, 'all'), false, Date.parse('2026-10-01T09:00:00Z'))
    expect(merged.map((x) => `${x.kind} ${x.at.slice(5, 10)}`)).toEqual(['item 10-13', 'grow 10-12', 'grow 10-06', 'item 10-01'])
    const complete = withGrow(items, growEvents(g, 'all'), true, Date.parse('2026-10-01T09:00:00Z'))
    expect(complete.at(-1)).toMatchObject({ kind: 'grow', at: '2026-09-29T18:02:00Z' })
  })
})
