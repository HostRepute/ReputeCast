import { ago, auditTime, clock, count, inDays, lastUsed, logTime, longDate, monthDay, monthDayTime, pct, rate, setDisplayTimeZone, shortDate, until, orList, money, notificationAgo, notificationTime } from './format'

describe('format (team and audit)', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))
  it('month day', () => expect(monthDay('2026-10-10T12:00:00Z')).toBe('Oct 10'))
  it('month day time', () => expect(monthDayTime('2026-10-12T09:41:00Z')).toBe('Oct 12, 09:41'))
  it('in days', () => expect(inDays('2026-10-18T13:00:00Z', new Date('2026-10-13T12:00:00Z'))).toBe('in 5 days'))
  it('last used', () => {
    const now = new Date('2026-10-12T09:44:00Z')
    expect(lastUsed('2026-10-12T09:41:00Z', now)).toBe('3 min ago')
    expect(lastUsed('2026-10-11T03:05:00Z', now)).toBe('yesterday, 03:05')
    expect(lastUsed('2026-08-30T10:00:00Z', now)).toBe('Aug 30, 2026')
    expect(lastUsed(null, now)).toBe('Never')
  })
  it('audit time', () => {
    const now = new Date('2026-10-12T09:46:00Z')
    expect(auditTime('2026-10-12T09:44:31Z', now)).toBe('09:44:31')
    expect(auditTime('2026-10-11T17:20:00Z', now)).toBe('Sun 11 Oct 17:20')
    expect(logTime('2026-10-06T08:02:00Z', now)).toBe('Tue 6 Oct')
  })
})

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
  it('notification times: today, this week, older', () => {
    const now = new Date('2026-10-12T09:40:00Z')
    expect(notificationTime('2026-10-12T07:40:00Z', now)).toBe('Today 07:40')
    expect(notificationTime('2026-10-09T09:21:00Z', now)).toBe('Fri 9 Oct 09:21')
    expect(notificationTime('2026-10-03T11:30:00Z', now)).toBe('Oct 3 11:30')
    expect(notificationTime('2025-12-30T22:14:00Z', now)).toBe('Dec 30, 2025 22:14')
    expect(notificationAgo('2026-10-12T07:40:00Z', now)).toBe('2 h ago')
    expect(notificationAgo('2026-10-08T14:05:00Z', now)).toBe('Thu 8 Oct')
    expect(notificationAgo('2026-10-01T09:02:00Z', now)).toBe('Oct 1')
  })
})
