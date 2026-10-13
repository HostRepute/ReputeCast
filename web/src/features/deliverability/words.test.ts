import { monthDay, setDisplayTimeZone, trimPct } from '@/lib/format'
import { authLine, cap, dayMonthOf, eta, firstRun, shortSource, usedBy, type CheckRun, type Overview } from './words'

describe('deliverability words', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))

  it('formats calendar dates without a timezone shift', () => {
    expect(dayMonthOf('2026-09-12')).toBe('12 Sep')
    expect(dayMonthOf('2026-10-01T00:00:00Z')).toBe('1 Oct')
  })

  it('capitalizes reputations and filter results', () => {
    expect(cap('high')).toBe('High')
    expect(cap('')).toBe('—')
    expect(cap(undefined)).toBe('—')
  })

  it('shortens blocklist levels', () => expect(shortSource('UCEPROTECT Level 3')).toBe('UCEPROTECT L3'))

  it('writes the authentication line as the board does', () => {
    expect(authLine({ spf: 'pass', dkim: 'pass', dmarc: 'quarantine' } as Parameters<typeof authLine>[0])).toBe('pass · pass · p=quarantine')
    expect(authLine({ spf: 'fail', dkim: 'pass', dmarc: 'missing' } as Parameters<typeof authLine>[0])).toBe('fail · pass · missing')
  })

  it('lists who uses a domain', () => {
    expect(usedBy(['SES', 'Postmark', 'Postal'])).toBe('used by SES, Postmark and Postal')
    expect(usedBy([])).toBe('')
  })

  it('estimates the time left of a check run', () => {
    const started = Date.parse('2026-10-12T10:00:00Z')
    const run = (answered: number) => ({ startedAt: '2026-10-12T10:00:00Z', total: 84, answered }) as CheckRun
    expect(eta(run(0), started + 2000)).toBe('starting')
    expect(eta(run(61), started + 14_000)).toBe('about 6 s left')
    expect(eta(run(84), started + 20_000)).toBe('finishing')
    expect(eta(run(1), started + 10_000)).toBe('about 14 min left')
  })

  it('knows the first run from a later one', () => {
    const overview = (checked: boolean) =>
      ({
        checks: { running: { id: 'r1' } },
        assets: [{ blocklists: { checkedAt: checked ? '2026-10-12T09:40:00Z' : undefined }, auth: undefined }],
      }) as unknown as Overview
    expect(firstRun(overview(false))).toBe(true)
    expect(firstRun(overview(true))).toBe(false)
  })

  it('trims rates and writes short dates', () => {
    expect(trimPct(0.6)).toBe('0.6%')
    expect(trimPct(0.04)).toBe('0.04%')
    expect(trimPct(1.1789)).toBe('1.18%')
    expect(monthDay('2026-10-10')).toBe('Oct 10')
  })
})
