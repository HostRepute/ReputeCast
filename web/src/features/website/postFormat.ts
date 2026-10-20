import { clock, dayMonthDate } from '@/lib/format'
import { offsetMinutes } from '@/features/audit/zoned'

/** "13 Oct 08:00": a post's day and time in the workspace's timezone. */
export function dateTime(d: string): string {
  return `${dayMonthDate(d)} ${clock(d)}`
}

/** A moment's day, month and time in a timezone of its own (a scheduled post's). */
export function zonedDayTime(d: string, tz: string | undefined): { day: string; time: string } {
  try {
    const p: Record<string, string> = {}
    for (const x of new Intl.DateTimeFormat('en-US', { timeZone: tz || undefined, day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(new Date(d))) p[x.type] = x.value
    return { day: `${p.day} ${p.month}`, time: `${p.hour}:${p.minute}` }
  } catch {
    return { day: dayMonthDate(d), time: clock(d) }
  }
}

/** "16 Oct, 17:00 America/New_York": when a scheduled post goes live, in its own timezone. */
export function scheduledLine(at: string, tz: string | undefined): string {
  const z = zonedDayTime(at, tz)
  return tz ? `${z.day}, ${z.time} ${tz}` : `${z.day}, ${z.time}`
}

/** The visibility's icon: a globe, a person or a star. */
export function visibilityIcon(v: string): 'globe' | 'user' | 'star' {
  return v === 'paid' ? 'star' : v === 'members' ? 'user' : 'globe'
}

/**
 * The instant a wall-clock time ("2026-10-16T17:00") is in a timezone:
 * scheduling a post "at 17:00 America/New_York". A second pass settles DST
 * edges.
 */
export function zonedToISO(local: string, tz: string): string {
  const [day = '', time = '00:00'] = local.split('T')
  const [y, mo, d] = day.split('-').map(Number) as [number, number, number]
  const [h, mi] = time.split(':').map(Number) as [number, number]
  const wall = Date.UTC(y, mo - 1, d, h, mi)
  let at = wall - offsetMinutes(tz, new Date(wall)) * 60_000
  at = wall - offsetMinutes(tz, new Date(at)) * 60_000
  return new Date(at).toISOString()
}

/** "2026-10-16T17:00" for an instant in a timezone (a datetime-local input's value). */
export function isoToZonedLocal(at: string, tz: string): string {
  try {
    const p: Record<string, string> = {}
    for (const x of new Intl.DateTimeFormat('en-US', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(new Date(at))) p[x.type] = x.value
    return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`
  } catch {
    return at.slice(0, 16)
  }
}
