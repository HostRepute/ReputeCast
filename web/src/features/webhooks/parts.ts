import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { clock, count } from '@/lib/format'

export type Webhook = Schemas['WebhookView']
export type Delivery = Schemas['WebhookDeliveryView']

/** "99.8%", "100%", "61%": one decimal where it matters (90% and up). */
export function successPct(v: number): string {
  if (v >= 99.95) return '100%'
  return v >= 90 ? `${v.toFixed(1)}%` : `${Math.round(v)}%`
}

/** "99.8% · 2,114", or "–" before the first delivery. */
export function successText(w: Webhook): string {
  if (w.total24h === 0 || w.success24h === undefined) return '–'
  return `${successPct(w.success24h)} · ${count(w.total24h)}`
}

export function statusBadge(w: Webhook): { tone: Tone; label: string } {
  switch (w.status) {
    case 'healthy':
      return { tone: 'ok', label: 'Healthy' }
    case 'failing':
      return { tone: 'bad', label: `Failing · ${w.lastStatus ? w.lastStatus : 'no response'}${w.failingSince ? ` since ${clock(w.failingSince)}` : ''}` }
    case 'paused':
      return { tone: 'neutral', label: 'Paused' }
  }
  return { tone: 'info', label: 'New' }
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

/** "200 · 84 ms", "502 · Bad Gateway", "DNS lookup failed". */
export function responseBadge(d: Delivery): { tone: Tone; label: string } | undefined {
  if (d.state === 'delivered') return { tone: 'ok', label: `${d.statusCode ?? 200}${d.durationMs !== undefined ? ` · ${d.durationMs} ms` : ''}` }
  if (d.statusCode) return { tone: 'bad', label: `${d.statusCode} · ${d.statusText || 'Error'}` }
  if (d.error) return { tone: 'bad', label: d.error.replace(/ for [^ ]+$/, '') }
  return undefined
}

/** "1", "4 of 8". */
export function attemptsText(d: Delivery): string {
  if (d.state === 'delivered' && d.attempts <= 1) return String(Math.max(1, d.attempts))
  if (d.attempts === 0) return '–'
  return `${d.attempts} of ${d.maxAttempts}`
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
