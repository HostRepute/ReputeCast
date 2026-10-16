import vectors from '../../../internal/i18n/testdata/regions.json'
import { REGION_FORMATS, formatDate, formatInt, formatMoney, formatPercent, formatTime, regionHourCycle, regionName, regionWeekStart, type DateParts, type RegionFormat } from './regions'

// The same vectors internal/i18n's Go tests check: the server and the
// browser write dates and numbers the same way.
function partsIn(iso: string, timeZone: string): DateParts {
  const p: Record<string, string> = {}
  const f = new Intl.DateTimeFormat('en-US', { timeZone, year: 'numeric', month: 'numeric', day: 'numeric', weekday: 'short', hour: 'numeric', minute: 'numeric', second: 'numeric', hourCycle: 'h23' })
  for (const x of f.formatToParts(new Date(iso))) p[x.type] = x.value
  return {
    year: Number(p.year),
    month: Number(p.month),
    day: Number(p.day),
    weekday: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(p.weekday ?? ''),
    hour: Number(p.hour) % 24,
    minute: Number(p.minute),
    second: Number(p.second),
  }
}

describe('region formats', () => {
  const at = partsIn(vectors.at, vectors.timeZone)
  const pm = partsIn(vectors.pm, vectors.timeZone)
  const regions = vectors.regions as Record<string, Record<string, string>>

  it('has a vector for every region format', () => {
    expect(Object.keys(regions).sort()).toEqual([...REGION_FORMATS].sort())
  })

  for (const code of REGION_FORMATS) {
    it(`writes ${code}`, () => {
      const r: RegionFormat = code
      const got: Record<string, string> = {
        name: regionName(r),
        date: formatDate(r, 'date', at),
        shortDate: formatDate(r, 'shortDate', at),
        weekdayDate: formatDate(r, 'weekdayDate', at),
        longDate: formatDate(r, 'longDate', at),
        dayMonth: formatDate(r, 'dayMonth', at),
        time: formatTime(r, at),
        time12: formatTime(r, at, 'h12'),
        timePM: formatTime(r, pm),
        timePM12: formatTime(r, pm, 'h12'),
        int: formatInt(r, 48213),
        int4: formatInt(r, 4821),
        int7: formatInt(r, 1234567),
        percent: formatPercent(r, 99.1, 1),
        money: formatMoney(r, 2.84, 2),
        money2: formatMoney(r, 18.21, 2),
        hourCycle: regionHourCycle(r),
        weekStart: regionWeekStart(r),
      }
      expect(got).toEqual(regions[code])
    })
  }

  it('writes the preview line of the design', () => {
    // Language & region: "Th 3, 14 thg 10, 2026 · 09:00 · 48.213 · 99,1% · 2,84 US$".
    const tue14: DateParts = { year: 2026, month: 10, day: 14, weekday: 2, hour: 9, minute: 0, second: 0 }
    const line = (r: RegionFormat) => [formatDate(r, 'weekdayDate', tue14), formatTime(r, tue14), formatInt(r, 48213), formatPercent(r, 99.1), formatMoney(r, 2.84)].join(' · ')
    expect(line('standard')).toBe('Tue 14 Oct 2026 · 09:00 · 48,213 · 99.1% · $2.84')
    expect(line('vi-VN')).toBe('Th 3, 14 thg 10, 2026 · 09:00 · 48.213 · 99,1% · 2,84 US$')
    expect(line('en-US')).toBe('Tue, Oct 14, 2026 · 9:00 AM · 48,213 · 99.1% · $2.84')
    expect(line('de-DE')).toBe('Di., 14. Okt. 2026 · 09:00 · 48.213 · 99,1 % · 2,84 $')
  })
})
