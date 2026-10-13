import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { ago, count, longDate, money, weekdayName } from '@/lib/format'

export type ProviderCard = Schemas['ProviderCard']

/** Ring and word colours of the sender health score. */
export const scoreColors: Record<string, { ring: string; word: string }> = {
  ok: { ring: '#0F766E', word: '#166534' },
  warn: { ring: '#B45309', word: '#92400E' },
  bad: { ring: '#DC2626', word: '#991B1B' },
  neutral: { ring: '#9AA3B2', word: '#5B6472' },
}

/** Dot colours of the reasons under the score. */
export const reasonDot: Record<string, string> = { ok: '#15803D', warn: '#D97706', bad: '#DC2626' }

/** "Sunday 12 Oct 2026 · news.acme.io" */
export function overviewSubtitle(now: Date, domain: string | undefined, pending: boolean): string {
  const day = longDate(now)
  if (pending) return `${day} · no sending domain yet`
  return domain ? `${day} · ${domain}` : day
}

/** The readiness badge: "Ready with 2 warnings", "Blocked by 3 checks", "Ready". */
export function readinessBadge(r: Schemas['OverviewReadiness'] | undefined): { tone: Tone; text: string } {
  if (!r) return { tone: 'neutral', text: 'Not checked yet' }
  const warns = r.issues.filter((i) => i.status === 'warn').length
  const blocks = r.issues.filter((i) => i.status === 'block').length
  switch (r.verdict) {
    case 'blocked':
      return { tone: 'bad', text: blocks ? `Blocked by ${blocks} ${blocks === 1 ? 'check' : 'checks'}` : 'Blocked' }
    case 'warnings':
      return { tone: 'warn', text: warns ? `Ready with ${warns} ${warns === 1 ? 'warning' : 'warnings'}` : 'Ready with warnings' }
  }
  return { tone: 'ok', text: 'Ready' }
}

/** "SES 90 / Postal 10" */
export function routeLine(routes: Schemas['OverviewRouteShare'][]): string {
  if (!routes.length) return '—'
  if (routes.length === 1) return routes[0]!.short
  return routes.map((r) => `${r.short} ${Math.round(r.percent)}`).join(' / ')
}

/** "34 min", "1 h 12 min" */
export function minutes(v: number | undefined): string {
  if (v === undefined) return '—'
  if (v < 60) return `${Math.max(1, Math.round(v))} min`
  const h = Math.floor(v / 60)
  const m = Math.round(v % 60)
  return m ? `${h} h ${m} min` : `${h} h`
}

/** "5%", "0.1%": a limit without trailing zeros. */
function limitPct(v: number): string {
  return `${Number(v.toFixed(2))}%`
}

/** "Bounce rate · limit 5% (SES review)", "Bounce rate · warm-up limit 2%". */
export function bounceLabel(l: Schemas['ProviderLimitRate']): string {
  if (l.label === 'warm-up guard') return `Bounce rate · warm-up limit ${limitPct(l.limit)}`
  return `Bounce rate · limit ${limitPct(l.limit)}${l.review && l.label ? ` (${l.label} review)` : ''}`
}

/** "Complaint rate · limit 0.1%" */
export function complaintLabel(l: Schemas['ProviderLimitRate']): string {
  if (l.label === 'warm-up guard') return `Complaint rate · warm-up limit ${limitPct(l.limit)}`
  return `Complaint rate · limit ${limitPct(l.limit)}`
}

/** The quota row: label and value ("24 h quota · 50,000", "2,140 used (4%)"). */
export function quotaRow(c: ProviderCard, now: Date): { label: string; value: string; percent: number } | null {
  const q = c.quota
  if (q.kind === 'none' || q.max <= 0) return null
  const percent = Math.round((q.used / q.max) * 100)
  switch (q.kind) {
    case 'rolling_24h':
      return { label: `24 h quota · ${count(q.max)}`, value: `${count(q.used)} used (${percent}%)`, percent }
    case 'monthly_plan':
      return { label: `Plan · ${count(q.max)} / month`, value: `${count(q.used)} used (${percent}%)`, percent }
    case 'warmup_daily':
      return { label: `Today's warm-up cap · ${count(q.max)}`, value: `${count(q.used)} sent (${weekdayName(now)})`, percent }
  }
  return null
}

/** The rate cell: "Max rate · 14 /s", "Rate (ramp) · 2 /s", "Max rate · No limit set". */
export function rateCell(c: ProviderCard): { label: string; value: string } {
  const r = c.rate
  const value = r.max > 0 ? `${Number.isInteger(r.max) ? r.max : r.max.toFixed(1)} /s` : 'No limit set'
  return { label: r.label === 'ramp limit' ? 'Rate (ramp)' : 'Max rate', value }
}

/** "2 min ago"; a webhook that just arrived reads "just now". */
export function webhookAgo(at: string | undefined, now: Date): string {
  if (!at) return 'no events yet'
  const a = ago(at, now)
  return a === 'now' ? 'just now' : a
}

/** The cost cell: "$14.62", "$15.00 plan", "$0 metered". */
export function costCell(c: ProviderCard): string {
  switch (c.cost.basis) {
    case 'plan':
      return `${money(c.cost.monthly ?? c.cost.toDate)} plan`
    case 'self_hosted':
      return '$0 metered'
  }
  return money(c.cost.toDate)
}

/** "+1,284 net · +2.7%" */
export function netBadge(a: Schemas['OverviewAudience']): { tone: Tone; text: string } {
  const sign = a.net > 0 ? '+' : a.net < 0 ? '−' : ''
  const pct = a.netPercent !== undefined ? ` · ${a.netPercent > 0 ? '+' : a.netPercent < 0 ? '−' : ''}${Math.abs(a.netPercent).toFixed(1)}%` : ''
  return { tone: a.net >= 0 ? 'ok' : 'bad', text: `${sign}${count(Math.abs(a.net))} net${pct}` }
}

/** "+812", "−40", "±0" */
export function delta(n: number): string {
  if (n > 0) return `+${count(n)}`
  if (n < 0) return `−${count(-n)}`
  return '±0'
}

export const tierName: Record<string, string> = { active: 'Active', cooling: 'Cooling', at_risk: 'At risk', sunset: 'Sunset' }
export const tierColor: Record<string, string> = {
  active: 'var(--chart-tier-1)',
  cooling: 'var(--chart-tier-2)',
  at_risk: 'var(--chart-tier-3)',
  sunset: 'var(--chart-tier-4)',
}

/** "Tuesday 14 Oct 2026" in a given timezone (a campaign's schedule). */
export function dateIn(d: string, tz: string): string {
  try {
    const parts: Record<string, string> = {}
    for (const p of new Intl.DateTimeFormat('en-US', { weekday: 'long', day: 'numeric', month: 'short', year: 'numeric', timeZone: tz }).formatToParts(new Date(d))) {
      parts[p.type] = p.value
    }
    return `${parts.weekday} ${parts.day} ${parts.month} ${parts.year}`
  } catch {
    return longDate(d)
  }
}

/** "09:00" in a given timezone. */
export function clockIn(d: string, tz: string): string {
  const opts: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit', hour12: false }
  try {
    return new Intl.DateTimeFormat('en-GB', { ...opts, timeZone: tz }).format(new Date(d))
  } catch {
    return new Intl.DateTimeFormat('en-GB', opts).format(new Date(d))
  }
}

/** "Couldn't refresh Amazon SES account health" alert text. */
export function refreshSentences(card: Schemas['OverviewProviderCard'], clock: (d: string) => string): { title: string; body: string } | null {
  const f = card.refresh
  if (!f) return null
  const name = card.displayName
  const times = f.attempts.map((a) => clock(a))
  const n = f.attempts.length
  const what = [f.status ? String(f.status) : '', f.code].filter(Boolean).join(' ') || f.message || 'an error'
  const many = n === 1 ? 'once' : n === 2 ? 'twice' : `${['three', 'four', 'five', 'six', 'seven', 'eight', 'nine'][n - 3] ?? n} times`
  const short = card.card.short
  let body = `${short} ${f.op} returned ${what} ${many}${times.length ? ` (${times.join(', ')})` : ''}.`
  if (f.lastSuccessAt) body += ` Bounce and complaint rates below are from the last successful fetch at ${clock(f.lastSuccessAt)}.`
  const feed = card.card.webhook.kind
  body += ` Routing and sending continue${feed ? `; ${short} events still arrive through ${feed === 'Registered' ? 'its webhook' : feed}` : ''}.`
  return { title: `Couldn't refresh ${name} account health`, body }
}
