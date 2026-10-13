/**
 * Formatting rules from the design system: counts use thousands separators
 * (48,213), percentages keep one decimal (99.1%), rates show their unit
 * (14 /s), money in dollars ($14.62), dates like "Thu 9 Oct".
 */
const nf = new Intl.NumberFormat('en-US')

export function count(n: number | null | undefined): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  return nf.format(Math.round(n))
}

/** 99.1% (one decimal unless `digits` says otherwise). */
export function pct(v: number | null | undefined, digits = 1): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  return `${v.toFixed(digits)}%`
}

/** Small rates keep two decimals: 0.04%, 0.18%. */
export function rate(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  return v < 1 ? `${v.toFixed(2)}%` : `${v.toFixed(1)}%`
}

export function money(v: number | null | undefined, digits = 2): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  return `$${v.toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits })}`
}

export function perSecond(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : `${count(v)} /s`
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

/** "Thu 9 Oct" */
export function shortDate(d: string | number | Date): string {
  return short.format(toDate(d)).replace(',', '')
}

/** "Sunday 12 Oct 2026" */
export function longDate(d: string | number | Date): string {
  return long.format(toDate(d)).replace(',', '')
}

/** "12 Sep" */
export function dayMonthDate(d: string | number | Date): string {
  return dayMonth.format(toDate(d))
}

/** "09:41" */
export function clock(d: string | number | Date): string {
  return time.format(toDate(d))
}

/** "Jan 1, 2027" */
export function usDate(d: string | number | Date): string {
  return monthDayYear.format(toDate(d))
}

/** "2 min ago", "12 min ago", "2 h ago", "yesterday", "3 days ago", "Sep 30, 2026". */
export function ago(d: string | number | Date | null | undefined, now: Date = new Date()): string {
  if (!d) return 'never'
  const t = toDate(d).getTime()
  const s = Math.round((now.getTime() - t) / 1000)
  if (s < 45) return 'now'
  const m = Math.round(s / 60)
  if (m < 60) return `${m} min ago`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} h ago`
  const days = Math.round(h / 24)
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days} days ago`
  return usDate(d)
}

/** "in 1 day 22 h", "in 34 min" */
export function until(d: string | number | Date, now: Date = new Date()): string {
  const m = Math.max(0, Math.round((toDate(d).getTime() - now.getTime()) / 60000))
  if (m < 60) return `in ${m} min`
  const h = Math.floor(m / 60)
  if (h < 24) return `in ${h} h ${m % 60} min`
  const days = Math.floor(h / 24)
  return `in ${days} day${days === 1 ? '' : 's'} ${h % 24} h`
}

/** "34 min", "1 h 12 min" */
export function duration(minutes: number): string {
  if (minutes < 60) return `${Math.max(1, Math.round(minutes))} min`
  const h = Math.floor(minutes / 60)
  const m = Math.round(minutes % 60)
  return m ? `${h} h ${m} min` : `${h} h`
}

/** Joins names as "A, B or C". */
export function orList(names: string[]): string {
  if (names.length <= 1) return names[0] ?? ''
  return `${names.slice(0, -1).join(', ')} or ${names[names.length - 1]}`
}

/** Joins names as "A, B and C". */
export function andList(names: string[]): string {
  if (names.length <= 1) return names[0] ?? ''
  return `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`
}

// --- Team, keys and audit log (W03) ---------------------------------------

/** "Oct 10" */
export function monthDay(d: string | number | Date): string {
  const p = parts('monthDay', toDate(d), { month: 'short', day: 'numeric' })
  return `${p.month} ${p.day}`
}

/** "Oct 12, 09:41" */
export function monthDayTime(d: string | number | Date): string {
  return `${monthDay(d)}, ${clock(d)}`
}

/** "09:44:31" */
export function clockSeconds(d: string | number | Date): string {
  return fmt('timeSeconds', 'en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(toDate(d))
}

/** "Oct 12, 08:40:55" */
export function monthDayTimeSeconds(d: string | number | Date): string {
  return `${monthDay(d)}, ${clockSeconds(d)}`
}

function dayKey(d: Date): string {
  const p = parts('dayKey', d, { year: 'numeric', month: '2-digit', day: '2-digit' })
  return `${p.year}-${p.month}-${p.day}`
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
  if (ms <= 0) return 'expired'
  const m = Math.round(ms / 60000)
  if (m < 60) return `in ${m} min`
  const h = Math.round(m / 60)
  if (h < 24) return `in ${h} h`
  const days = Math.floor(h / 24)
  return `in ${days} day${days === 1 ? '' : 's'}`
}

/** Last use of an API key: "3 min ago", "2 h ago", "yesterday, 03:05", "Aug 30, 2026". */
export function lastUsed(d: string | number | Date | null | undefined, now: Date = new Date()): string {
  if (!d) return 'Never'
  const t = toDate(d)
  const m = Math.round((now.getTime() - t.getTime()) / 60000)
  if (m < 1) return 'now'
  if (m < 60) return `${m} min ago`
  if (sameDay(t, now)) return `${Math.round(m / 60)} h ago`
  if (isYesterday(t, now)) return `yesterday, ${clock(t)}`
  return usDate(t)
}

/** A time in an activity list: "09:36" today, else "Mon 6 Oct". */
export function logTime(d: string | number | Date, now: Date = new Date()): string {
  const t = toDate(d)
  return sameDay(t, now) ? clock(t) : shortDate(t)
}

/** Audit log time: "09:44:31" today, else "Sat 11 Oct 17:20". */
export function auditTime(d: string | number | Date, now: Date = new Date()): string {
  const t = toDate(d)
  return sameDay(t, now) ? clockSeconds(t) : `${shortDate(t)} ${clock(t)}`
}
