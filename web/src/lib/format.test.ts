import { ago, clock, count, longDate, pct, rate, setDisplayTimeZone, shortDate, until, orList, money } from './format'

describe('format', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))
  it('counts with separators', () => expect(count(48213)).toBe('48,213'))
  it('percent with one decimal', () => expect(pct(99.1)).toBe('99.1%'))
  it('small rates with two decimals', () => expect(rate(0.04)).toBe('0.04%'))
  it('money', () => expect(money(14.62)).toBe('$14.62'))
  it('short date', () => expect(shortDate('2026-10-09T12:00:00Z')).toBe('Fri 9 Oct'))
  it('long date', () => expect(longDate('2026-10-12T12:00:00Z')).toBe('Monday 12 Oct 2026'))
  it('ago', () => {
    const now = new Date('2026-10-12T10:00:00Z')
    expect(ago('2026-10-12T09:58:00Z', now)).toBe('2 min ago')
    expect(ago('2026-10-12T08:00:00Z', now)).toBe('2 h ago')
    expect(ago('2026-10-11T08:00:00Z', now)).toBe('yesterday')
  })
  it('until', () => {
    const now = new Date('2026-10-12T11:00:00Z')
    expect(until('2026-10-14T09:00:00Z', now)).toBe('in 1 day 22 h')
  })
  it('uses the display timezone', () => {
    setDisplayTimeZone('America/New_York')
    expect(clock('2026-10-14T13:00:00Z')).toBe('09:00')
    expect(shortDate('2026-10-14T13:00:00Z')).toBe('Wed 14 Oct')
    setDisplayTimeZone('UTC')
  })
  it('uses US month abbreviations', () => {
    expect(shortDate('2026-09-30T12:00:00Z')).toBe('Wed 30 Sep')
  })
  it('orList', () => expect(orList(['Marcus Chen', 'Linh Nguyen'])).toBe('Marcus Chen or Linh Nguyen'))
})
