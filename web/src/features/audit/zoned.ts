/**
 * The instant a calendar day starts in an IANA timezone: zonedDayStart("2026-10-06",
 * "America/New_York") is 2026-10-06T04:00:00Z. `plusDays` moves to a later day
 * (1 gives the end of a custom range).
 */
export function zonedDayStart(day: string, zone: string, plusDays = 0): Date {
  const [y, m, d] = day.split('-').map(Number) as [number, number, number]
  const utcMidnight = Date.UTC(y, m - 1, d + plusDays)
  // The zone's offset near that moment; a second pass settles DST edges.
  let t = utcMidnight - offsetMinutes(zone, new Date(utcMidnight)) * 60_000
  t = utcMidnight - offsetMinutes(zone, new Date(t)) * 60_000
  return new Date(t)
}

/** Minutes east of UTC for a zone at an instant (New York in October: -240). */
export function offsetMinutes(zone: string, at: Date): number {
  try {
    const name =
      new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortOffset' }).formatToParts(at).find((p) => p.type === 'timeZoneName')?.value ?? ''
    const m = name.match(/GMT([+-])(\d{1,2})(?::(\d{2}))?/)
    if (!m) return 0
    const v = Number(m[2]) * 60 + Number(m[3] ?? 0)
    return m[1] === '-' ? -v : v
  } catch {
    return 0
  }
}
