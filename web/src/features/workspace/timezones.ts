/** Timezone options labelled with their current offset: "America/New_York (UTC−4)". */
export function timezoneOptions(current: string): { value: string; label: string }[] {
  let zones: string[]
  try {
    zones = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.('timeZone') ?? []
  } catch {
    zones = []
  }
  if (!zones.includes('UTC')) zones = [...zones, 'UTC']
  if (current && !zones.includes(current)) zones = [current, ...zones]
  return zones.map((z) => ({ value: z, label: `${z} (${offset(z)})` }))
}

/** "UTC−4", "UTC+5:30", "UTC" */
export function offset(zone: string, at: Date = new Date()): string {
  try {
    const part = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortOffset' }).formatToParts(at).find((p) => p.type === 'timeZoneName')?.value ?? ''
    const m = part.match(/GMT([+-])(\d{1,2})(?::(\d{2}))?/)
    if (!m) return 'UTC'
    const sign = m[1] === '-' ? '−' : '+'
    return `UTC${sign}${Number(m[2])}${m[3] ? `:${m[3]}` : ''}`
  } catch {
    return 'UTC'
  }
}
