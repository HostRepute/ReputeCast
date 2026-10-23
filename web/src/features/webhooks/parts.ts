import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { t } from '@/i18n'
import { clock, count, pct } from '@/lib/format'
import { m } from './messages'

export type Webhook = Schemas['WebhookView']
export type Delivery = Schemas['WebhookDeliveryView']

/** "99.8%", "100%", "61%": one decimal where it matters (90% and up). */
export function successPct(v: number): string {
  if (v >= 99.95) return pct(100, 0)
  return v >= 90 ? pct(v, 1) : pct(Math.round(v), 0)
}

/** "99.8% · 2,114", or "–" before the first delivery. */
export function successText(w: Webhook): string {
  if (w.total24h === 0 || w.success24h === undefined) return '–'
  return `${successPct(w.success24h)} · ${count(w.total24h)}`
}

export function statusBadge(w: Webhook): { tone: Tone; label: string } {
  switch (w.status) {
    case 'healthy':
      return { tone: 'ok', label: t(m.statusHealthy) }
    case 'failing': {
      const values = { hasStatus: w.lastStatus ? 'yes' : 'no', status: String(w.lastStatus ?? '') }
      return { tone: 'bad', label: w.failingSince ? t(m.statusFailingSince, { ...values, time: clock(w.failingSince) }) : t(m.statusFailing, values) }
    }
    case 'paused':
      return { tone: 'neutral', label: t(m.statusPaused) }
  }
  return { tone: 'info', label: t(m.statusNew) }
}

/** The host and path of a delivery's endpoint; masked paths (Slack) show the host only. */
export function endpointText(d: Pick<Delivery, 'endpoint'>): string {
  let s = d.endpoint
  try {
    s = decodeURIComponent(d.endpoint)
  } catch {
    // keep the raw value
  }
  return s.includes('…') ? s.split('/')[0]! : s
}

/** "200 · 84 ms", "502 · Bad Gateway", "DNS lookup failed" (the endpoint's own words). */
export function responseBadge(d: Delivery): { tone: Tone; label: string } | undefined {
  if (d.state === 'delivered') {
    const code = String(d.statusCode ?? 200)
    return { tone: 'ok', label: d.durationMs !== undefined ? t(m.responseTime, { code, ms: count(d.durationMs) }) : code }
  }
  if (d.statusCode) return { tone: 'bad', label: `${d.statusCode} · ${d.statusText || t(m.responseError)}` }
  if (d.error) return { tone: 'bad', label: d.error.replace(/ for [^ ]+$/, '') }
  return undefined
}

/** "1", "4 of 8". */
export function attemptsText(d: Delivery): string {
  if (d.state === 'delivered' && d.attempts <= 1) return count(Math.max(1, d.attempts))
  if (d.attempts === 0) return '–'
  return t(m.attemptsOf, { attempts: count(d.attempts), max: count(d.maxAttempts) })
}

/** "event · subtype" as on the delivery log. */
export function eventText(d: Pick<Delivery, 'event' | 'subtype'>): string {
  return d.subtype ? `${d.event} · ${d.subtype}` : d.event
}

/** Host of a webhook URL ("warehouse.acme.io"). */
export function hostOf(url: string): string {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}
