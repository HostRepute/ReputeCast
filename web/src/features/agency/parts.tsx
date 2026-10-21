import type { CSSProperties } from 'react'
import { Avatar } from '@/components/ui/Avatar'
import { getI18n, t } from '@/i18n'
import { formatDate, formatDecimal, formatTime, monthLabel as regionMonth, type DateParts } from '@/i18n/regions'
import { cn } from '@/lib/cn'
import { getDisplayTimeZone } from '@/lib/format'
import type { AgencyHealth, AgencyWorkspaceRef } from './api'
import { m } from './messages'
import './agency.css'

/** A workspace's initials tile in the client's colour (24 px by default). */
export function ClientTile({ ws, size = 24, className, style }: { ws: Pick<AgencyWorkspaceRef, 'initials' | 'color'>; size?: number; className?: string; style?: CSSProperties }) {
  const radius = size >= 28 ? 7 : size >= 24 ? 6 : 5
  const font = size >= 28 ? 11 : 10
  return (
    <span
      className={cn('ag-wst', className)}
      style={{ width: size, height: size, borderRadius: radius, fontSize: font, background: ws.color, ...style }}
      aria-hidden="true"
      data-initials={ws.initials}
    />
  )
}

/** The health colours of the boards: the dot and the trend's line. */
export function healthColors(tone: string): { dot: string; line: string } {
  switch (tone) {
    case 'warn':
      return { dot: '#B45309', line: '#B45309' }
    case 'bad':
      return { dot: '#B91C1C', line: '#B91C1C' }
    case 'ok':
      return { dot: '#15803D', line: '#0F766E' }
    default:
      return { dot: '#9AA3B2', line: '#9AA3B2' }
  }
}

function toneWord(tone: string) {
  return tone === 'ok' ? t(m.healthGood) : tone === 'warn' ? t(m.healthFair) : t(m.healthPoor)
}

/** The score with its dot ("● 71"), titled "Sender health 71, fair". */
export function HealthScore({ health }: { health: AgencyHealth }) {
  const c = healthColors(health.tone)
  return (
    <span className="ag-hsc" title={t(m.healthTitle, { score: health.score, tone: toneWord(health.tone) })}>
      <span className="dot" style={{ color: c.dot }} aria-hidden="true" />
      {health.score}
    </span>
  )
}

/**
 * A client's 30-day health as the overview draws it: seven points on a
 * fixed scale (so 90 sits near the top and 62 near the bottom), with a
 * light fill.
 */
export function HealthSpark({ trend, tone, width = 60, height = 20 }: { trend: number[]; tone: string; width?: number; height?: number }) {
  if (trend.length < 2) return null
  const n = 7
  const pts: number[] = []
  for (let i = 0; i < n; i++) pts.push(trend[Math.round((i * (trend.length - 1)) / (n - 1))]!)
  const y = (v: number) => Math.min(height - 1, Math.max(1, ((34.1 - 0.3214 * v) * height) / 20))
  const xs = pts.map((_, i) => 1 + (i * (width - 2)) / (n - 1))
  const line = pts.map((v, i) => `${xs[i]!.toFixed(1)},${y(v).toFixed(1)}`).join(' ')
  const area = `M1,${height} ` + pts.map((v, i) => `L${xs[i]!.toFixed(1)},${y(v).toFixed(1)}`).join(' ') + ` L${width - 1},${height} Z`
  const c = healthColors(tone)
  return (
    <svg className="ag-spark" width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <path d={area} fill={c.line} opacity=".10" />
      <polyline points={line} fill="none" stroke={c.line} strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}

/** The health cell: score, trend, and HostRepute's view where it's connected. */
export function HealthCell({ health, spark = true }: { health: AgencyHealth | undefined; spark?: boolean }) {
  if (!health || health.state === 'unknown') return <span className="k">{t(m.dash)}</span>
  return (
    <span className="flex flex-col gap-[3px]">
      <span className="flex items-center gap-2.5">
        <HealthScore health={health} />
        {spark ? <HealthSpark trend={health.trend} tone={health.tone} /> : null}
      </span>
      {health.hostRepute ? <span className="k whitespace-nowrap">{t(m.hostRepute, { state: hostReputeWord(health.hostRepute, health.listings ?? 0) })}</span> : null}
    </span>
  )
}

function hostReputeWord(state: string, listings: number) {
  switch (state) {
    case 'clean':
      return t(m.hrClean)
    case 'listed':
      return t(m.hrListed, { n: listings })
    default:
      return t(m.hrStale)
  }
}

/** A person with their avatar ("ML Mei Lin"). */
export function PersonChip({ name, color, size = 22 }: { name: string; color?: string; size?: number }) {
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap">
      <Avatar name={name} color={color} size={size} fontSize={11} />
      <span>{name}</span>
    </span>
  )
}

// Times. The Agency area shows the agency's time zone (its workspace's);
// a client's send in another zone says so ("3 Nov, 08:00 Pacific"). With
// ReputeCast standard they're written as the boards write them; another
// region format (Account & security › Language & region) writes them its
// own way, as lib/format does.

/** ReputeCast standard: the boards' formats, untouched. */
function standard() {
  return getI18n().regionFormat === 'standard'
}

const fmts = new Map<string, Intl.DateTimeFormat>()
function fmt(zone: string | undefined, opts: Intl.DateTimeFormatOptions) {
  // Without a zone, the agency's (the display time zone).
  const tz = zone ?? getDisplayTimeZone()
  const key = `${tz ?? ''}|${JSON.stringify(opts)}`
  let f = fmts.get(key)
  if (!f) {
    try {
      // US month names, written day first ("30 Sep"), as the boards do.
      f = new Intl.DateTimeFormat('en-US', { ...opts, timeZone: tz })
    } catch {
      f = new Intl.DateTimeFormat('en-US', opts)
    }
    fmts.set(key, f)
  }
  return f
}

function partsOf(d: Date, tz: string | undefined, opts: Intl.DateTimeFormatOptions) {
  const out: Record<string, string> = {}
  for (const p of fmt(tz, opts).formatToParts(d)) out[p.type] = p.value
  return out
}

const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** A moment's fields in the zone, for the region formats. */
function zoned(d: Date, tz: string | undefined): DateParts {
  const p = partsOf(d, tz, { year: 'numeric', month: 'numeric', day: 'numeric', weekday: 'short', hour: 'numeric', minute: 'numeric', second: 'numeric', hourCycle: 'h23' })
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

function offsetOf(d: Date, tz: string | undefined) {
  const p = partsOf(d, tz, { year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric', hourCycle: 'h23' })
  return Date.UTC(Number(p.year), Number(p.month) - 1, Number(p.day), Number(p.hour) % 24, Number(p.minute)) - Math.floor(d.getTime() / 60000) * 60000
}

/** The zone's name when it isn't the agency's: "Pacific"; in the app language otherwise ("Giờ Thái Bình Dương"). */
export function zoneSuffix(d: Date, tz: string | undefined): string {
  const here = getDisplayTimeZone()
  if (!tz || tz === here || offsetOf(d, tz) === offsetOf(d, here)) return ''
  const { locale } = getI18n()
  const english = locale.startsWith('en')
  try {
    const name = new Intl.DateTimeFormat(english ? 'en-US' : locale, { timeZone: tz, timeZoneName: 'longGeneric' }).formatToParts(d).find((p) => p.type === 'timeZoneName')?.value ?? ''
    return english ? name.replace(/ Time$/, '') : name
  } catch {
    return tz
  }
}

const sameDay = (a: Date, b: Date, tz: string | undefined) => {
  const o = { year: 'numeric', month: 'numeric', day: 'numeric' } as const
  return fmt(tz, o).format(a) === fmt(tz, o).format(b)
}

/** The moment falls on today in the agency's zone. */
export function isToday(d: Date | string, now = new Date()) {
  return sameDay(new Date(d), now, getDisplayTimeZone())
}

/** "17:00" in the zone. */
export function hhmm(d: Date | string, tz?: string) {
  const { regionFormat, hourCycle } = getI18n()
  if (regionFormat !== 'standard' || hourCycle !== 'h23') return formatTime(regionFormat, zoned(new Date(d), tz), hourCycle)
  const p = partsOf(new Date(d), tz, { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  return `${p.hour}:${p.minute}`
}

/** "16 Oct" in the zone. */
export function dayMonth(d: Date | string, tz?: string) {
  if (!standard()) return formatDate(getI18n().regionFormat, 'dayMonth', zoned(new Date(d), tz))
  const p = partsOf(new Date(d), tz, { day: 'numeric', month: 'short' })
  return `${p.day} ${p.month}`
}

/** "4 Mar 2024": a calendar date, stored at midnight UTC. */
export function calendarDate(d: Date | string) {
  if (!standard()) return formatDate(getI18n().regionFormat, 'date', zoned(new Date(d), 'UTC'))
  return `${dayMonth(d, 'UTC')} ${new Date(d).getUTCFullYear()}`
}

/** "16 Oct 2026": a day with its year, in the agency's zone. */
export function dayMonthYear(d: Date | string) {
  if (!standard()) return formatDate(getI18n().regionFormat, 'date', zoned(new Date(d), undefined))
  return `${dayMonth(d)} ${new Date(d).getFullYear()}`
}

/** "Thu 16 Oct" in the zone. */
export function weekdayDayMonth(d: Date | string, tz?: string) {
  if (!standard()) return formatDate(getI18n().regionFormat, 'shortDate', zoned(new Date(d), tz))
  const p = partsOf(new Date(d), tz, { weekday: 'short', day: 'numeric', month: 'short' })
  return `${p.weekday} ${p.day} ${p.month}`
}

/** "October": a month ("2026-10") on its own. */
export function monthOf(month: string) {
  const [y, mo] = month.split('-').map(Number)
  if (!standard()) return regionMonth(getI18n().regionFormat, mo!, 'long')
  return new Intl.DateTimeFormat('en-GB', { month: 'long', timeZone: 'UTC' }).format(new Date(Date.UTC(y!, mo! - 1, 1)))
}

/** "October 2026": a month ("2026-10") with its year. */
export function monthYear(month: string, style: 'long' | 'short' = 'long') {
  const [y, mo] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y!, mo! - 1, 1))
  const { regionFormat } = getI18n()
  // ReputeCast standard writes "October 2026"; other region formats as CLDR does ("tháng 10 năm 2026").
  const locale = regionFormat === 'standard' ? 'en-GB' : regionFormat
  return new Intl.DateTimeFormat(locale, { month: style, year: 'numeric', timeZone: 'UTC' }).format(d)
}

/**
 * A send's day as the sends strip heads it: "Today", else "Thu 16 Oct"
 * (in the agency's zone).
 */
export function sendDay(d: Date | string, now = new Date()) {
  const date = new Date(d)
  return sameDay(date, now, getDisplayTimeZone()) ? t(m.today) : weekdayDayMonth(date, getDisplayTimeZone())
}

/**
 * When a send goes out, in the send's own zone: "Today, 17:00",
 * "Sun 19 Oct, 10:00" within a week, "3 Nov, 08:00 Pacific" later.
 */
export function sendWhen(d: Date | string, tz: string | undefined, now = new Date(), weekday = false) {
  const date = new Date(d)
  const zone = tz || getDisplayTimeZone()
  const suffix = zoneSuffix(date, zone)
  const day = sameDay(date, now, zone)
    ? t(m.today)
    : weekday && date.getTime() - now.getTime() < 7 * 86400_000
      ? weekdayDayMonth(date, zone)
      : dayMonth(date, zone)
  return suffix ? t(m.whenZone, { day, time: hhmm(date, zone), zone: suffix }) : t(m.when, { day, time: hhmm(date, zone) })
}

/** Whole hours since a moment ("19"). */
export function hoursSince(d: string | Date, now = new Date()) {
  return Math.max(0, Math.round((now.getTime() - new Date(d).getTime()) / 3_600_000))
}

/** Whole days until a moment, at least 1. */
export function daysUntil(d: string | Date, now = new Date()) {
  return Math.max(1, Math.round((new Date(d).getTime() - now.getTime()) / 86_400_000))
}

/** "12 Oct, 14:10–14:52": an incident's span in the agency's zone. */
export function span(from: string, to?: string) {
  const tz = getDisplayTimeZone()
  const start = `${dayMonth(from, tz)}, ${hhmm(from, tz)}`
  if (!to) return start
  return sameDay(new Date(from), new Date(to), tz) ? `${start}–${hhmm(to, tz)}` : `${start}–${dayMonth(to, tz)}, ${hhmm(to, tz)}`
}

/** "1,451.74" money without the currency sign, as the board writes amounts. */
export function amount(v: number) {
  if (!standard()) return formatDecimal(getI18n().regionFormat, v, 2)
  return v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
