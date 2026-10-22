import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { tierLabel } from '@/features/audience/labels'
import { getI18n, t } from '@/i18n'
import { formatDate, formatTime, type DateParts } from '@/i18n/regions'
import { ago, clock as displayClock, count, decimal, duration, longDate, money, pct, trimPct, weekdayName } from '@/lib/format'
import { m as fm } from '@/lib/format.messages'
import { m } from './messages'

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
  if (pending) return t(m.subtitleNoDomain, { day })
  return domain ? `${day} · ${domain}` : day
}

/** The readiness badge: "Ready with 2 warnings", "Blocked by 3 checks", "Ready". */
export function readinessBadge(r: Schemas['OverviewReadiness'] | undefined): { tone: Tone; text: string } {
  if (!r) return { tone: 'neutral', text: t(m.notChecked) }
  const warns = r.issues.filter((i) => i.status === 'warn').length
  const blocks = r.issues.filter((i) => i.status === 'block').length
  switch (r.verdict) {
    case 'blocked':
      return { tone: 'bad', text: blocks ? t(m.blockedBy, { n: blocks }) : t(m.blocked) }
    case 'warnings':
      return { tone: 'warn', text: warns ? t(m.readyWith, { n: warns }) : t(m.readyWarnings) }
  }
  return { tone: 'ok', text: t(m.ready) }
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
  return duration(v)
}

/** "Bounce rate · limit 5% (SES review)", "Bounce rate · warm-up limit 2%"; `short` drops whose level it is. */
export function bounceLabel(l: Schemas['ProviderLimitRate'], short = false): string {
  const limit = trimPct(l.limit)
  if (l.label === 'warm-up guard') return t(m.bounceWarmup, { limit })
  return l.review && l.label && !short ? t(m.bounceLimitReview, { limit, provider: l.label }) : t(m.bounceLimit, { limit })
}

/** "Complaint rate · limit 0.1%" */
export function complaintLabel(l: Schemas['ProviderLimitRate']): string {
  const limit = trimPct(l.limit)
  if (l.label === 'warm-up guard') return t(m.complaintWarmup, { limit })
  return t(m.complaintLimit, { limit })
}

/** The quota row: label and value ("24 h quota · 50,000", "2,140 used (4%)"). */
export function quotaRow(c: ProviderCard, now: Date): { label: string; value: string; percent: number } | null {
  const q = c.quota
  if (q.kind === 'none' || q.max <= 0) return null
  const percent = Math.round((q.used / q.max) * 100)
  const used = t(m.quotaUsed, { used: count(q.used), percent: pct(percent, 0) })
  switch (q.kind) {
    case 'rolling_24h':
      return { label: t(m.quota24h, { max: count(q.max) }), value: used, percent }
    case 'monthly_plan':
      return { label: t(m.quotaPlan, { max: count(q.max) }), value: used, percent }
    case 'warmup_daily':
      return { label: t(m.quotaWarmup, { max: count(q.max) }), value: t(m.quotaSent, { sent: count(q.used), weekday: weekdayName(now) }), percent }
  }
  return null
}

/** The rate cell: "Max rate · 14 /s", "Rate (ramp) · 2 /s", "Max rate · No limit set". */
export function rateCell(c: ProviderCard): { label: string; value: string } {
  const r = c.rate
  const value = r.max > 0 ? t(fm.perSecond, { n: Number.isInteger(r.max) ? count(r.max) : decimal(r.max, 1) }) : t(m.noLimit)
  return { label: r.label === 'ramp limit' ? t(m.rampRate) : t(m.maxRate), value }
}

/** "2 min ago"; a webhook that just arrived reads "just now". */
export function webhookAgo(at: string | undefined, now: Date): string {
  if (!at) return t(m.noEvents)
  // ago() says "now" under 45 seconds; the card says "just now".
  if (Math.round((now.getTime() - new Date(at).getTime()) / 1000) < 45) return t(m.justNow)
  return ago(at, now)
}

/** The cost cell: "$14.62", "$15.00 plan", "$0 metered". */
export function costCell(c: ProviderCard): string {
  switch (c.cost.basis) {
    case 'plan':
      return t(m.costPlan, { cost: money(c.cost.monthly ?? c.cost.toDate) })
    case 'self_hosted':
      return t(m.costMetered, { cost: money(0, 0) })
  }
  return money(c.cost.toDate)
}

/** "+1,284 net · +2.7%" */
export function netBadge(a: Schemas['OverviewAudience']): { tone: Tone; text: string } {
  const sign = (n: number) => (n > 0 ? '+' : n < 0 ? '−' : '')
  const net = `${sign(a.net)}${count(Math.abs(a.net))}`
  const text = a.netPercent !== undefined ? t(m.netPercent, { net, percent: `${sign(a.netPercent)}${pct(Math.abs(a.netPercent), 1)}` }) : t(m.net, { net })
  return { tone: a.net >= 0 ? 'ok' : 'bad', text }
}

/** "+812", "−40", "±0" */
export function delta(n: number): string {
  if (n > 0) return `+${count(n)}`
  if (n < 0) return `−${count(-n)}`
  return '±0'
}

/** The tier names, as on List health. */
export const tierName: Record<string, string> = tierLabel
export const tierColor: Record<string, string> = {
  active: 'var(--chart-tier-1)',
  cooling: 'var(--chart-tier-2)',
  at_risk: 'var(--chart-tier-3)',
  sunset: 'var(--chart-tier-4)',
}

const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** A moment's fields in a given timezone, for the region formats. */
function partsIn(d: string, tz: string): DateParts | undefined {
  try {
    const p: Record<string, string> = {}
    const f = new Intl.DateTimeFormat('en-US', { timeZone: tz, year: 'numeric', month: 'numeric', day: 'numeric', weekday: 'short', hour: 'numeric', minute: 'numeric', hourCycle: 'h23' })
    for (const x of f.formatToParts(new Date(d))) p[x.type] = x.value
    return {
      year: Number(p.year),
      month: Number(p.month),
      day: Number(p.day),
      weekday: Math.max(0, weekdays.indexOf(p.weekday ?? '')),
      hour: Number(p.hour) % 24,
      minute: Number(p.minute),
      second: 0,
    }
  } catch {
    return undefined
  }
}

/** "Tuesday 14 Oct 2026" in a given timezone (a campaign's schedule). */
export function dateIn(d: string, tz: string): string {
  const p = partsIn(d, tz)
  return p ? formatDate(getI18n().regionFormat, 'longDate', p) : longDate(d)
}

/** "09:00" in a given timezone. */
export function clockIn(d: string, tz: string): string {
  const p = partsIn(d, tz)
  return p ? formatTime(getI18n().regionFormat, p, getI18n().hourCycle) : displayClock(d)
}

/** "13 Sep" from a calendar date of a series ("2026-09-13"), whatever the timezone. */
export function dayLabel(day: string): string {
  const [y, mo, dd] = day.split('-').map(Number)
  return formatDate(getI18n().regionFormat, 'dayMonth', { year: y ?? 0, month: mo ?? 1, day: dd ?? 1, weekday: 0, hour: 0, minute: 0, second: 0 })
}

/** "Couldn't refresh Amazon SES account health" alert text. */
export function refreshSentences(card: Schemas['OverviewProviderCard'], clock: (d: string) => string): { title: string; body: string } | null {
  const f = card.refresh
  if (!f) return null
  const times = f.attempts.map((a) => clock(a))
  const what = [f.status ? String(f.status) : '', f.code].filter(Boolean).join(' ') || f.message
  const short = card.card.short
  const sentences = [
    t(m.refreshReturned, { known: what ? 'yes' : 'no', provider: short, op: f.op, what: what ?? '', n: f.attempts.length, shown: times.length ? 'yes' : 'no', times: times.join(', ') }),
  ]
  if (f.lastSuccessAt) sentences.push(t(m.refreshLastGood, { time: clock(f.lastSuccessAt) }))
  const feed = card.card.webhook.kind
  sentences.push(!feed ? t(m.refreshContinue) : feed === 'Registered' ? t(m.refreshContinueWebhook, { provider: short }) : t(m.refreshContinueFeed, { provider: short, feed }))
  return { title: t(m.refreshTitle, { name: card.displayName }), body: sentences.join(' ') }
}
