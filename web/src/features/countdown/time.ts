/** Moments of the Countdown block: a local end in its IANA time zone. */

/** The moment a local "YYYY-MM-DDTHH:MM" names in a time zone. */
export function zonedMoment(local: string, zone: string): Date | null {
  const m_ = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(local)
  if (!m_) return null
  const [y, mo, d, h, mi] = m_.slice(1).map(Number) as [number, number, number, number, number]
  let guess = Date.UTC(y, mo - 1, d, h, mi)
  // Two passes settle the zone's offset at that moment (DST included).
  for (let i = 0; i < 2; i++) {
    const p = new Intl.DateTimeFormat('en-US', { timeZone: zone, hourCycle: 'h23', year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric' }).formatToParts(new Date(guess))
    const get = (t: string) => Number(p.find((x) => x.type === t)?.value)
    const shown = Date.UTC(get('year'), get('month') - 1, get('day'), get('hour'), get('minute'))
    guess += Date.UTC(y, mo - 1, d, h, mi) - shown
  }
  return new Date(guess)
}

/** "Fri 27 Nov, 23:59 EST": a moment in a zone, in the app's language (English as the app writes days). */
export function inZone(d: Date, zone: string, lang: string, withZone = true): string {
  try {
    const f = new Intl.DateTimeFormat(lang, { timeZone: zone, weekday: 'short', day: 'numeric', month: 'short' })
    const p = f.formatToParts(d)
    const part = (t: string) => p.find((x) => x.type === t)?.value ?? ''
    const day = lang.startsWith('en') ? `${part('weekday')} ${part('day')} ${part('month')}` : f.format(d)
    const time = new Intl.DateTimeFormat(lang, { timeZone: zone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(d)
    const tz = withZone ? new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'short' }).formatToParts(d).find((p) => p.type === 'timeZoneName')?.value : ''
    return `${day}, ${time}${tz ? ` ${tz}` : ''}`
  } catch {
    return d.toISOString()
  }
}
