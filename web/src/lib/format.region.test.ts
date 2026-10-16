import { act } from 'react'
import { setCatalogLoader, setLocale, setPreferences, PSEUDO_LOCALE } from '@/i18n'
import vi from './format.messages.vi'
import {
  ago,
  andList,
  clock,
  count,
  dayMonthDate,
  inDays,
  lastUsed,
  longDate,
  money,
  notificationTime,
  orList,
  pct,
  perSecond,
  rate,
  setDisplayTimeZone,
  shortDate,
  trimPct,
  until,
  usDate,
  weekdayDate,
  weekStartsOn,
} from './format'

// format.test.ts keeps checking ReputeCast standard, unchanged. These check
// the other region formats and the words in the app's other languages.
describe('format in other region formats and languages', () => {
  beforeAll(() => {
    setDisplayTimeZone('America/New_York')
    setCatalogLoader('vi', async () => vi)
  })
  afterEach(async () => {
    await act(() => setPreferences({ locale: 'en', regionFormat: 'standard' }))
  })

  it('writes Tiếng Việt (Việt Nam) like the stress-test boards', async () => {
    await setPreferences({ locale: 'vi', regionFormat: 'vi-VN' })
    expect(shortDate('2026-10-14T13:00:00Z')).toBe('Th 4, 14 thg 10')
    expect(longDate('2026-10-13T13:00:00Z')).toBe('Thứ Ba, 13 tháng 10, 2026')
    expect(weekdayDate('2026-10-13T13:00:00Z')).toBe('Th 3, 13 thg 10, 2026')
    expect(usDate('2026-09-30T16:00:00Z')).toBe('30 thg 9, 2026')
    expect(dayMonthDate('2026-10-10T16:00:00Z')).toBe('10 thg 10')
    expect(clock('2026-10-14T13:00:00Z')).toBe('09:00')
    expect(count(187312)).toBe('187.312')
    expect(pct(99.2)).toBe('99,2%')
    expect(rate(0.04)).toBe('0,04%')
    expect(trimPct(0.6)).toBe('0,6%')
    expect(money(18.21)).toBe('18,21 US$')
    expect(perSecond(14)).toBe('14/giây')
    const now = new Date('2026-10-15T18:20:00Z')
    expect(ago('2026-10-15T18:08:00Z', now)).toBe('12 phút trước')
    expect(ago('2026-10-15T16:20:00Z', now)).toBe('2 giờ trước')
    expect(ago('2026-10-14T16:20:00Z', now)).toBe('hôm qua')
    expect(ago('2026-10-15T18:20:10Z', now)).toBe('vừa xong')
    expect(inDays('2026-10-20T19:00:00Z', now)).toBe('sau 5 ngày')
    expect(until('2026-10-17T16:20:00Z', now)).toBe('sau 1 ngày 22 giờ')
    expect(notificationTime('2026-10-15T11:40:00Z', now)).toBe('Hôm nay 07:40')
    expect(lastUsed(null, now)).toBe('Chưa bao giờ')
    expect(orList(['Marcus Chen', 'Linh Nguyen'])).toBe('Marcus Chen hoặc Linh Nguyen')
    expect(andList(['English', 'Spanish', 'German'])).toBe('English, Spanish và German')
    expect(weekStartsOn()).toBe(1)
  })

  it('writes English (United States) with a 12-hour clock', async () => {
    await setPreferences({ locale: 'en', regionFormat: 'en-US' })
    expect(weekdayDate('2026-10-13T13:00:00Z')).toBe('Tue, Oct 13, 2026')
    expect(clock('2026-10-13T13:00:00Z')).toBe('9:00 AM')
    expect(shortDate('2026-10-13T13:00:00Z')).toBe('Tue, Oct 13')
    expect(weekStartsOn()).toBe(0)
    // Words stay English: they follow the language, not the region.
    expect(ago('2026-10-15T18:08:00Z', new Date('2026-10-15T18:20:00Z'))).toBe('12 min ago')
  })

  it('lets the clock and the week differ from the region', async () => {
    await setPreferences({ locale: 'en', regionFormat: 'standard', hourCycle: 'h12', weekStart: 'sunday' })
    expect(clock('2026-10-14T01:05:00Z')).toBe('9:05 PM')
    expect(weekStartsOn()).toBe(0)
    expect(count(48213)).toBe('48,213')
  })

  it('writes Deutsch (Deutschland)', async () => {
    await setPreferences({ locale: 'en', regionFormat: 'de-DE' })
    expect(weekdayDate('2026-10-13T13:00:00Z')).toBe('Di., 13. Okt. 2026')
    expect(pct(99.1)).toBe('99,1 %')
    expect(money(2.84)).toBe('2,84 $')
  })

  it('brackets words in the pseudo-locale and keeps the standard formats', async () => {
    await setPreferences({ locale: PSEUDO_LOCALE, regionFormat: 'standard' })
    expect(ago('2026-10-15T18:08:00Z', new Date('2026-10-15T18:20:00Z'))).toBe('⟦12 mîñ áĝö ···⟧')
    expect(shortDate('2026-10-13T13:00:00Z')).toBe('Tue 13 Oct')
    expect(count(31604)).toBe('31,604')
    await setLocale('en')
  })
})
