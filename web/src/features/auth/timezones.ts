/** "UTC−4", "UTC+5:30", "UTC" for a timezone right now. */
export function utcOffset(tz: string, at = new Date()): string {
  try {
    const part = new Intl.DateTimeFormat('en-US', { timeZone: tz, timeZoneName: 'shortOffset' }).formatToParts(at).find((p) => p.type === 'timeZoneName')?.value ?? ''
    const m = /GMT([+-])(\d{1,2})(?::(\d{2}))?/.exec(part)
    if (!m) return 'UTC'
    return `UTC${m[1] === '-' ? '−' : '+'}${Number(m[2])}${m[3] && m[3] !== '00' ? `:${m[3]}` : ''}`
  } catch {
    return 'UTC'
  }
}

let cached: string[] | undefined

/** Every IANA timezone the browser knows, UTC first. */
export function timezones(): string[] {
  if (!cached) {
    const all = typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : []
    cached = ['UTC', ...all.filter((z) => z !== 'UTC')]
  }
  return cached
}

const labels = new Map<string, string>()

/** Option label: "America/New_York (UTC−4)". */
export function timezoneLabel(tz: string): string {
  let label = labels.get(tz)
  if (!label) {
    const off = utcOffset(tz)
    label = tz === 'UTC' || off === 'UTC' ? tz : `${tz} (${off})`
    labels.set(tz, label)
  }
  return label
}

/** The timezones to offer, making sure the current value is one of them. */
export function timezoneOptions(current?: string): string[] {
  const list = timezones()
  return current && !list.includes(current) ? [current, ...list] : list
}
