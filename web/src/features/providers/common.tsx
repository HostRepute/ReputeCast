import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs } from '@/components/ui/Tabs'
import { cn } from '@/lib/cn'
import { ago, duration } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import './providers.css'

export type ProviderCard = Schemas['ProviderCard']
export type ColorKey = ProviderCard['color']
export type Stream = 'broadcast' | 'automation' | 'system'
export type FailoverPolicy = 'automatic' | 'pause_and_ask' | 'warm_only'

/** Sending-provider series colours from the design system. */
export const providerColor: Record<ColorKey, string> = {
  ses: 'var(--chart-ses)',
  postal: 'var(--chart-postal)',
  postmark: 'var(--chart-postmark)',
  other: 'var(--chart-other)',
}

/** Text colour on a provider-coloured segment (the routing split bar). */
export const onProviderColor: Record<ColorKey, string> = { ses: '#fff', postal: '#3F0A1F', postmark: '#fff', other: 'var(--ink)' }

export const streamName: Record<Stream, string> = { broadcast: 'Broadcasts', automation: 'Automations', system: 'System' }

export const failoverText: Record<FailoverPolicy, string> = {
  automatic: 'automatic',
  pause_and_ask: 'pause and ask',
  warm_only: 'automatic, warmed-up only',
}

/** The five sections of Sending providers & routing, inline after the title. */
export function ProvidersTabs() {
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  return (
    <NavTabs
      label="Sections"
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: 'Providers', to: '/w/$workspace/providers', params, exact: true },
        { label: 'Routing & failover', to: '/w/$workspace/providers/routing', params },
        { label: 'Warm-up', to: '/w/$workspace/providers/warmup', params },
        { label: 'Events & sync', to: '/w/$workspace/providers/events', params },
        { label: 'Adapters', to: '/w/$workspace/providers/adapters', params },
      ]}
    />
  )
}

/** "5%", "0.1%", "2%": a limit without trailing zeros. */
export function limitPct(v: number): string {
  return `${Number(v.toFixed(2))}%`
}

/** Bounce rates keep one decimal (0.6%), complaint rates two (0.04%). */
export function bouncePct(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : `${v.toFixed(1)}%`
}
export function complaintPct(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : `${v.toFixed(2)}%`
}

/** Messages per second: "0", "14", "0.5"; `decimals` for a live rate ("2.0"). */
export function perSec(v: number, decimals = false): string {
  if (v === 0) return '0'
  if (decimals) return v.toFixed(1)
  return Number.isInteger(v) ? String(v) : v.toFixed(1)
}

/** The status word of a provider card. */
export function statusBadge(c: Pick<ProviderCard, 'status' | 'warmup' | 'down' | 'webhook'>, now = new Date()): { tone: Tone; text: string } {
  switch (c.status) {
    case 'healthy':
      return { tone: 'ok', text: 'Healthy' }
    case 'warming_up':
      return { tone: 'info', text: c.warmup ? `Warming up · day ${c.warmup.day} of ${c.warmup.days}` : 'Warming up' }
    case 'down':
      if (c.down?.reason === 'auth') return { tone: 'bad', text: 'Down · auth failed' }
      return { tone: 'bad', text: c.down ? `Down · ${sinceText(c.down.since, now)}` : 'Down' }
    case 'degraded':
      return { tone: 'warn', text: c.webhook.silentSince ? `Degraded · webhooks silent ${sinceText(c.webhook.silentSince, now)}` : 'Degraded' }
    case 'sandbox':
      return { tone: 'warn', text: 'Sandbox · verified addresses only' }
    case 'paused':
      return { tone: 'neutral', text: 'Paused' }
  }
}

/** "14 min", "6 h 2 min" since a time. */
export function sinceText(t: string, now = new Date()): string {
  return duration(Math.max(1, (now.getTime() - new Date(t).getTime()) / 60000))
}

export function StatusBadge({ card, className }: { card: Pick<ProviderCard, 'status' | 'warmup' | 'down' | 'webhook'>; className?: string }) {
  const b = statusBadge(card)
  return (
    <Badge tone={b.tone} dot className={className}>
      {b.text}
    </Badge>
  )
}

/** A capability chip (`.cap`): yes, partial, no (struck) or plain info. */
export function CapabilityChip({ cap }: { cap: Schemas['ProviderCapabilityBadge'] }) {
  const cls = { yes: 'y', partial: 'p', no: 'n', info: '' }[cap.kind]
  return (
    <span className={cn('pv-cap', cls)}>
      {cap.kind === 'no' ? <span className="vh">Not supported: </span> : null}
      {cap.label}
    </span>
  )
}

/** Labelled metric with a thin bar (`.m`). */
export function Meter({
  label,
  value,
  percent,
  color,
  valueClassName,
  valueStyle,
}: {
  label: string
  value: string
  percent: number
  color: string
  valueClassName?: string
  valueStyle?: React.CSSProperties
}) {
  const w = Math.max(0, Math.min(100, percent))
  return (
    <div className="pv-m">
      <div className="l">
        <span className="k">{label}</span>
        <span className={cn('n', valueClassName)} style={valueStyle}>
          {value}
        </span>
      </div>
      <div className="bar" aria-hidden="true">
        <i style={{ width: `${w}%`, background: color }} />
      </div>
    </div>
  )
}

/** Green below half of the limit, amber from half, red over it; grey when stale. */
export function rateColor(value: number | undefined, limit: number, stale = false): string {
  if (stale) return 'var(--color-muted-line)'
  if (value === undefined || limit <= 0) return 'var(--ok-bar)'
  const share = value / limit
  if (share >= 1) return 'var(--bad-bar)'
  if (share >= 0.5) return 'var(--warn-bar)'
  return 'var(--ok-bar)'
}

/** "20 s ago" under a minute, then as `ago` ("2 min ago"). */
export function agoSeconds(t: string, now = new Date()): string {
  const s = Math.max(0, Math.round((now.getTime() - new Date(t).getTime()) / 1000))
  return s < 60 ? `${s} s ago` : ago(t, now)
}

/** "Bounce rate · limit 5% (SES review at 5%, pause at 10%)"; `short` drops the provider's levels. */
export function limitLabel(kind: 'Bounce' | 'Complaint', l: Schemas['ProviderLimitRate'], short = false): string {
  if (l.label === 'warm-up guard') return `${kind} rate · warm-up guard ${limitPct(l.limit)}`
  let s = `${kind} rate · limit ${limitPct(l.limit)}`
  if (short) return s
  if (l.label && (l.review || l.pause)) {
    const parts = [l.review ? `review at ${limitPct(l.review)}` : '', l.pause ? `pause at ${limitPct(l.pause)}` : ''].filter(Boolean)
    s += ` (${l.label} ${parts.join(', ')})`
  } else if (l.label) {
    s += ` (${l.label})`
  }
  return s
}

/** A key on the left, its value on the right (provider card facts). */
export function Fact({ k, children, className }: { k: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn('flex justify-between max-md:flex-wrap', className)}>
      <span className="k">{k}</span>
      {children}
    </div>
  )
}
