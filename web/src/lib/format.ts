import { formatList, getI18n, t } from '@/i18n'
import {
  formatDate,
  formatDecimal,
  formatInt,
  formatMoney,
  formatPercent,
  formatTime,
  monthLabel,
  weekdayLabel,
  type DateParts,
  type RegionFormat,
  type WeekStart,
} from '@/i18n/regions'
import { m } from './format.messages'

/**
 * Formatting rules from the design system: counts use thousands separators
 * (48,213), percentages keep one decimal (99.1%), rates show their unit
 * (14 /s), money in dollars ($14.62), dates like "Thu 9 Oct".
 *
 * Those are the "ReputeCast standard" region format. A member who picks
 * another (Account & security › Language & region) gets its CLDR formats
 * (web/src/i18n/regions.ts), and words such as "2 min ago" follow the app
 * language either way.
 */
const nf = new Intl.NumberFormat('en-US')

function region(): RegionFormat {
  return getI18n().regionFormat
}

/** ReputeCast standard with its own 24-hour clock: today's formats, untouched. */
function standard(): boolean {
  return getI18n().regionFormat === 'standard'
}

function standardClock(): boolean {
  return standard() && getI18n().hourCycle === 'h23'
}

export function count(n: number | null | undefined): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  if (!standard()) return formatInt(region(), Math.round(n))
  return nf.format(Math.round(n))
}

/** 99.1% (one decimal unless `digits` says otherwise). */
export function pct(v: number | null | undefined, digits = 1): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (!standard()) return formatPercent(region(), v, digits)
  return `${v.toFixed(digits)}%`
}

/** Small rates keep two decimals: 0.04%, 0.18%. */
export function rate(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (!standard()) return formatPercent(region(), v, v < 1 ? 2 : 1)
  return v < 1 ? `${v.toFixed(2)}%` : `${v.toFixed(1)}%`
}

export function money(v: number | null | undefined, digits = 2): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (!standard()) return formatMoney(region(), v, digits)
  return `$${v.toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits })}`
}

export function perSecond(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : t(m.perSecond, { n: count(v) })
}

/**
 * Dates and times are shown in the workspace's timezone (so a team in
 * several countries sees the same "09:00"), set by the workspace layout.
 * Outside a workspace the browser's timezone is used.
 */
let displayTimeZone: string | undefined

export function setDisplayTimeZone(tz: string | undefined) {
  if (tz === displayTimeZone) return
  displayTimeZone = tz
  formatters.clear()
}

export function getDisplayTimeZone(): string | undefined {
  return displayTimeZone
}

const formatters = new Map<string, Intl.DateTimeFormat>()
function fmt(key: string, locale: string, opts: Intl.DateTimeFormatOptions) {
  let f = formatters.get(key)
  if (!f) {
    try {
      f = new Intl.DateTimeFormat(locale, { ...opts, timeZone: displayTimeZone })
    } catch {
      f = new Intl.DateTimeFormat(locale, opts)
    }
    formatters.set(key, f)
  }
  return f
}
/** Day-first dates with US month abbreviations, as in the design: "Tue 30 Sep". */
function parts(key: string, d: Date, opts: Intl.DateTimeFormatOptions) {
  const out: Record<string, string> = {}
  for (const p of fmt(key, 'en-US', opts).formatToParts(d)) out[p.type] = p.value
  return out
}
const short = {
  format: (d: Date) => {
    const p = parts('short', d, { weekday: 'short', day: 'numeric', month: 'short' })
    return `${p.weekday} ${p.day} ${p.month}`
  },
}
const long = {
  format: (d: Date) => {
    const p = parts('long', d, { weekday: 'long', day: 'numeric', month: 'short', year: 'numeric' })
    return `${p.weekday} ${p.day} ${p.month} ${p.year}`
  },
}
const dayMonth = {
  format: (d: Date) => {
    const p = parts('dayMonth', d, { day: 'numeric', month: 'short' })
    return `${p.day} ${p.month}`
  },
}
const time = { format: (d: Date) => fmt('time', 'en-GB', { hour: '2-digit', minute: '2-digit', hour12: false }).format(d) }
const monthDayYear = { format: (d: Date) => fmt('mdy', 'en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(d) }

const toDate = (d: string | number | Date) => (d instanceof Date ? d : new Date(d))

const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** A moment's fields in the display timezone, for the region formats. */
function zoned(d: Date): DateParts {
  const p = parts('zoned', d, { year: 'numeric', month: 'numeric', day: 'numeric', weekday: 'short', hour: 'numeric', minute: 'numeric', second: 'numeric', hourCycle: 'h23' })
  return {
    year: Number(p.year),
    month: Number(p.month),
    day: Number(p.day),
    weekday: Math.max(0, weekdays.indexOf(p.weekday ?? '')),
    hour: Number(p.hour) % 24,
    minute: Number(p.minute),
    second: Number(p.second),
  }
}

/** "Thu 9 Oct" */
export function shortDate(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'shortDate', zoned(toDate(d)))
  return short.format(toDate(d)).replace(',', '')
}

/** "Sunday 12 Oct 2026" */
export function longDate(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'longDate', zoned(toDate(d)))
  return long.format(toDate(d)).replace(',', '')
}

/** "Tue 14 Oct 2026": a day with its weekday and year. */
export function weekdayDate(d: string | number | Date): string {
  return formatDate(region(), 'weekdayDate', zoned(toDate(d)))
}

/** "12 Sep" */
export function dayMonthDate(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'dayMonth', zoned(toDate(d)))
  return dayMonth.format(toDate(d))
}

/** "09:41" */
export function clock(d: string | number | Date): string {
  if (!standardClock()) return formatTime(region(), zoned(toDate(d)), getI18n().hourCycle)
  return time.format(toDate(d))
}

/** A time of day kept as "HH:MM" (quiet hours, a digest's send time), on the reader's clock: "22:00", "10:00 PM". */
export function timeOfDay(hhmm: string): string {
  const [h, m] = hhmm.split(':').map(Number)
  if (standardClock() || h === undefined || m === undefined || !Number.isInteger(h) || !Number.isInteger(m)) return hhmm
  return formatTime(region(), { year: 2000, month: 1, day: 1, weekday: 6, hour: h, minute: m, second: 0 }, getI18n().hourCycle)
}

/** "Jan 1, 2027" */
export function usDate(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'date', zoned(toDate(d)))
  return monthDayYear.format(toDate(d))
}

/** "2 min ago", "12 min ago", "2 h ago", "yesterday", "3 days ago", "Sep 30, 2026". */
export function ago(d: string | number | Date | null | undefined, now: Date = new Date()): string {
  if (!d) return t(m.never)
  const ts = toDate(d).getTime()
  const s = Math.round((now.getTime() - ts) / 1000)
  if (s < 45) return t(m.now)
  const min = Math.round(s / 60)
  if (min < 60) return t(m.minutesAgo, { n: min })
  const h = Math.round(min / 60)
  if (h < 24) return t(m.hoursAgo, { n: h })
  const days = Math.round(h / 24)
  if (days === 1) return t(m.yesterday)
  if (days < 7) return t(m.daysAgo, { n: days })
  return usDate(d)
}

/** "in 1 day 22 h", "in 34 min" */
export function until(d: string | number | Date, now: Date = new Date()): string {
  const min = Math.max(0, Math.round((toDate(d).getTime() - now.getTime()) / 60000))
  if (min < 60) return t(m.inMinutes, { n: min })
  const h = Math.floor(min / 60)
  if (h < 24) return t(m.inHoursMinutes, { h, m: min % 60 })
  const days = Math.floor(h / 24)
  return t(m.inDaysHours, { days, h: h % 24 })
}

/** "34 min", "1 h 12 min" */
export function duration(minutes: number): string {
  if (minutes < 60) return t(m.minutes, { n: Math.max(1, Math.round(minutes)) })
  const h = Math.floor(minutes / 60)
  const min = Math.round(minutes % 60)
  return min ? t(m.hoursMinutes, { h, m: min }) : t(m.hours, { h })
}

/** Joins names as "A, B or C". */
export function orList(names: string[]): string {
  return formatList(names, 'disjunction')
}

/** Joins names as "A, B and C". */
export function andList(names: string[]): string {
  return formatList(names)
}

/** "09:18:51" (clock with seconds, for live streams). */
export function clockSeconds(d: string | number | Date): string {
  if (!standardClock()) return formatTime(region(), zoned(toDate(d)), getI18n().hourCycle, true)
  return fmt('time-s', 'en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(toDate(d))
}

/** "12 hours", "1 hour", "45 minutes", "1 minute": a span in words, rounded down. */
export function spanWords(ms: number): string {
  const minutes = Math.max(0, Math.floor(ms / 60_000))
  if (minutes >= 60) return t(m.spanHours, { n: Math.floor(minutes / 60) })
  return t(m.spanMinutes, { n: Math.max(1, minutes) })
}

/** "in 2 minutes", "in 1 minute", "in 45 seconds": a short countdown. */
export function countdown(ms: number): string {
  const s = Math.max(0, Math.ceil(ms / 1000))
  if (s >= 120) return t(m.countdownMinutes, { n: Math.floor(s / 60) })
  if (s >= 60) return t(m.countdownOneMinute)
  return t(m.countdownSeconds, { n: s })
}

function dayKey(d: Date) {
  const p = parts('daykey', d, { year: 'numeric', month: '2-digit', day: '2-digit' })
  return `${p.year}-${p.month}-${p.day}`
}

function daysBetween(d: Date, now: Date) {
  const a = Date.parse(`${dayKey(d)}T00:00:00Z`)
  const b = Date.parse(`${dayKey(now)}T00:00:00Z`)
  return Math.round((b - a) / 86_400_000)
}

function yearOf(d: Date) {
  return parts('year', d, { year: 'numeric' }).year
}

/**
 * When a notification happened, in the notification center: "Today 07:40",
 * "Thu 9 Oct 09:21" within the week, "Oct 3 11:30" before, with the year
 * when it isn't this year.
 */
export function notificationTime(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  const days = daysBetween(ts, now)
  if (days <= 0) return t(m.todayAt, { time: clock(ts) })
  if (days < 7) return `${shortDate(ts)} ${clock(ts)}`
  return yearOf(ts) === yearOf(now) ? `${monthDay(ts)} ${clock(ts)}` : `${usDate(ts)} ${clock(ts)}`
}

/** The bell popover's shorter form: "2 h ago", "Thu 9 Oct", "Oct 1". */
export function notificationAgo(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  const days = daysBetween(ts, now)
  if (days <= 0) return ago(ts, now)
  if (days < 7) return shortDate(ts)
  return yearOf(ts) === yearOf(now) ? monthDay(ts) : usDate(ts)
}

/** A percentage with up to two decimals and no trailing zeros: 0.6%, 0.04%, 99.6%, 100%. */
export function trimPct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  const n = Number(v.toFixed(2))
  if (!standard()) return formatPercent(region(), n, (String(n).split('.')[1] ?? '').length)
  return `${n}%`
}

/** "Oct 10" */
export function monthDay(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'monthDay', zoned(toDate(d)))
  const p = parts('monthDay', toDate(d), { month: 'short', day: 'numeric' })
  return `${p.month} ${p.day}`
}

/** "Oct 12, 09:41" */
export function monthDayTime(d: string | number | Date): string {
  return `${monthDay(d)}, ${clock(d)}`
}

/** "Oct 12, 08:40:55" */
export function monthDayTimeSeconds(d: string | number | Date): string {
  return `${monthDay(d)}, ${clockSeconds(d)}`
}

function sameDay(a: Date, b: Date): boolean {
  return dayKey(a) === dayKey(b)
}

/** True when d falls on the day before now in the display timezone. */
function isYesterday(d: Date, now: Date): boolean {
  return dayKey(d) === dayKey(new Date(now.getTime() - 86_400_000))
}

/** "in 5 days", "in 3 h", "in 20 min", "expired" */
export function inDays(d: string | number | Date, now: Date = new Date()): string {
  const ms = toDate(d).getTime() - now.getTime()
  if (ms <= 0) return t(m.expired)
  const min = Math.round(ms / 60000)
  if (min < 60) return t(m.inMinutes, { n: min })
  const h = Math.round(min / 60)
  if (h < 24) return t(m.inHours, { n: h })
  return t(m.inDays, { n: Math.floor(h / 24) })
}

/** Last use of an API key: "3 min ago", "2 h ago", "yesterday, 03:05", "Aug 30, 2026". */
export function lastUsed(d: string | number | Date | null | undefined, now: Date = new Date()): string {
  if (!d) return t(m.neverTitle)
  const ts = toDate(d)
  const min = Math.round((now.getTime() - ts.getTime()) / 60000)
  if (min < 1) return t(m.now)
  if (min < 60) return t(m.minutesAgo, { n: min })
  if (sameDay(ts, now)) return t(m.hoursAgo, { n: Math.round(min / 60) })
  if (isYesterday(ts, now)) return t(m.yesterdayAt, { time: clock(ts) })
  return usDate(ts)
}

/** A time in an activity list: "09:36" today, else "Mon 6 Oct". */
export function logTime(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  return sameDay(ts, now) ? clock(ts) : shortDate(ts)
}

/** Audit log time: "09:44:31" today, else "Sat 11 Oct 17:20". */
export function auditTime(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  return sameDay(ts, now) ? clockSeconds(ts) : `${shortDate(ts)} ${clock(ts)}`
}

/** When a key was created: "Oct 12, 09:41" today, else "Sep 22, 2026". */
export function createdTime(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  return sameDay(ts, now) ? monthDayTime(ts) : usDate(ts)
}

/** A call in an activity list: "09:36" today, else "Mon 6 Oct", with what was called. */
export function callTime(d: string | number | Date, what: string | undefined, now: Date = new Date()): string {
  return what ? `${logTime(d, now)} · ${what}` : logTime(d, now)
}

/** "Oct" or "October". */
export function monthName(d: string | number | Date, style: 'short' | 'long' = 'short'): string {
  if (!standard()) return monthLabel(region(), zoned(toDate(d)).month, style)
  return fmt(`month-${style}`, 'en-US', { month: style }).format(toDate(d))
}

/** A month with its year: "Oct 2026", "thg 10 2026"; long: "October 2026", "tháng 10 năm 2026". */
export function monthYear(d: string | number | Date, style: 'short' | 'long' = 'short'): string {
  return formatDate(region(), style === 'long' ? 'monthYearLong' : 'monthYear', zoned(toDate(d)))
}

/** "Sunday" */
export function weekdayName(d: string | number | Date): string {
  if (!standard()) return weekdayLabel(region(), zoned(toDate(d)).weekday)
  return fmt('weekday-long', 'en-US', { weekday: 'long' }).format(toDate(d))
}

/**
 * When an address was suppressed: "Today 09:41", "Sat 11 Oct" within the
 * week, "18 Sep" before, with the year when it isn't this year.
 */
export function suppressedWhen(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  const days = daysBetween(ts, now)
  if (days <= 0) return t(m.todayAt, { time: clock(ts) })
  if (days < 7) return shortDate(ts)
  const year = yearOf(ts)
  if (year === yearOf(now)) return dayMonthDate(ts)
  return standard() ? `${dayMonthDate(ts)} ${year}` : usDate(ts)
}

/** Instance audit log time: "09:12:03" today, "Sat 11 Oct 14:02" this week, else "Sep 30 17:10". */
export function instanceAuditTime(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  if (sameDay(ts, now)) return clockSeconds(ts)
  if (now.getTime() - ts.getTime() < 7 * 86_400_000) return `${shortDate(ts)} ${clock(ts)}`
  return `${monthDay(ts)} ${clock(ts)}`
}

/** When something happened, in a sentence: "this morning", "this afternoon", "this evening", "yesterday", "on Oct 9". */
export function dayPartWords(d: string | number | Date, now: Date = new Date()): string {
  const ts = toDate(d)
  if (sameDay(ts, now)) {
    const hour = zoned(ts).hour
    return hour < 12 ? t(m.thisMorning) : hour < 18 ? t(m.thisAfternoon) : t(m.thisEvening)
  }
  if (isYesterday(ts, now)) return t(m.yesterday)
  return t(m.onDate, { date: monthDay(ts) })
}

/** "18 Sep 2026" */
export function dayMonthYear(d: string | number | Date): string {
  if (!standard()) return formatDate(region(), 'date', zoned(toDate(d)))
  const p = parts('dayMonthYear', toDate(d), { day: 'numeric', month: 'short', year: 'numeric' })
  return `${p.day} ${p.month} ${p.year}`
}

/** A moment's year, month, day, weekday and time in the display timezone. */
export function zonedParts(d: string | number | Date): DateParts {
  return zoned(toDate(d))
}

/**
 * A calendar day ("2026-10-14", in no timezone) in the region format:
 * "Wednesday 14 Oct 2026" (longDate), "Nov 4" (monthDay), "Nov 4, 2026" (date).
 */
export function calendarDate(day: string, kind: 'longDate' | 'monthDay' | 'date' | 'shortDate' = 'longDate'): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day)
  if (!m) return day
  const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])))
  return formatDate(region(), kind, { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate(), weekday: d.getUTCDay(), hour: 0, minute: 0, second: 0 })
}

/** A moment in another timezone than the display one, in the region format: "Nov 4, 2026" and "09:00". */
export function inZone(d: string | number | Date, tz: string | undefined): { date: string; time: string } {
  let p: Record<string, string> = {}
  try {
    const f = new Intl.DateTimeFormat('en-US', { timeZone: tz || undefined, year: 'numeric', month: 'numeric', day: 'numeric', weekday: 'short', hour: 'numeric', minute: 'numeric', hourCycle: 'h23' })
    for (const x of f.formatToParts(toDate(d))) p[x.type] = x.value
  } catch {
    p = {}
  }
  const parts: DateParts = {
    year: Number(p.year),
    month: Number(p.month),
    day: Number(p.day),
    weekday: Math.max(0, weekdays.indexOf(p.weekday ?? '')),
    hour: Number(p.hour) % 24,
    minute: Number(p.minute),
    second: 0,
  }
  if (!p.year) return { date: usDate(d), time: clock(d) }
  return { date: formatDate(region(), 'date', parts), time: formatTime(region(), parts, getI18n().hourCycle) }
}

/** The first day of the week in calendars: 1 is Monday, 0 is Sunday. */
export function weekStartsOn(): 0 | 1 {
  const w: WeekStart = getI18n().weekStart
  return w === 'sunday' ? 0 : 1
}

/** A plain number with fixed decimals by the region format: 2.84 · 2,84. */
export function decimal(v: number | null | undefined, digits = 1): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (!standard()) return formatDecimal(region(), v, digits)
  return v.toFixed(digits)
}
