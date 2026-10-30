import { ApiError, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs } from '@/components/ui/Tabs'
import { ChannelMenu } from '@/features/channels/ChannelMenu'
import { T, t as tt, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { ago, decimal, duration, pct, trimPct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './providers.css'

export type ProviderCard = Schemas['ProviderCard']
export type ColorKey = ProviderCard['color']
export type Stream = 'broadcast' | 'automation' | 'system' | 'transactional'
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

/** A stream's name in the app language (read when shown, so it follows the language). */
export const streamName: Record<Stream, string> = {
  get broadcast() {
    return tt(m.streamBroadcast)
  },
  get automation() {
    return tt(m.streamAutomation)
  },
  get system() {
    return tt(m.streamSystem)
  },
  get transactional() {
    return tt(m.streamTransactional)
  },
}

/** A failover policy in a sentence ("failover: pause and ask"). */
export const failoverText: Record<FailoverPolicy, string> = {
  get automatic() {
    return tt(m.failoverAutomatic)
  },
  get pause_and_ask() {
    return tt(m.failoverPauseAndAsk)
  },
  get warm_only() {
    return tt(m.failoverWarmOnly)
  },
}

/**
 * A role without access to this screen gets the shell's 403 state (the
 * route's error component); other load errors stay inline with a retry.
 */
export function throwIfForbidden(error: unknown) {
  if (error instanceof ApiError && error.status === 403) throw error
}

/** The five sections of Sending providers & routing, inline after the title. */
export function ProvidersTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  return (
    <>
      {/* One channel button with a listbox: Email, SMS, Push, WhatsApp (Providers board). */}
      <ChannelMenu current="email" />
      <NavTabs
        label={t(m.sectionsLabel)}
        className="ml-3 max-md:ml-0"
        tabs={[
          { label: t(m.tabProviders), to: '/w/$workspace/providers', params, exact: true },
          { label: t(m.tabRouting), to: '/w/$workspace/providers/routing', params },
          { label: t(m.tabWarmup), to: '/w/$workspace/providers/warmup', params },
          { label: t(m.tabTraffic), to: '/w/$workspace/providers/traffic-rules', params },
          { label: t(m.tabEvents), to: '/w/$workspace/providers/events', params },
          { label: t(m.tabAdapters), to: '/w/$workspace/providers/adapters', params },
        ]}
      />
    </>
  )
}

/** "5%", "0.1%", "2%": a limit without trailing zeros (in the region format). */
export function limitPct(v: number): string {
  return trimPct(v)
}

/** Bounce rates keep one decimal (0.6%), complaint rates two (0.04%). */
export function bouncePct(v: number | null | undefined): string {
  return pct(v, 1)
}
export function complaintPct(v: number | null | undefined): string {
  return pct(v, 2)
}

/** Messages per second: "0", "14", "0.5"; `decimals` for a live rate ("2.0"). */
export function perSec(v: number, decimals = false): string {
  if (v === 0) return '0'
  if (decimals) return decimal(v, 1)
  return decimal(v, Number.isInteger(v) ? 0 : 1)
}

/** The status word of a provider card. */
export function statusBadge(c: Pick<ProviderCard, 'status' | 'warmup' | 'down' | 'webhook'>, now = new Date()): { tone: Tone; text: string } {
  switch (c.status) {
    case 'healthy':
      return { tone: 'ok', text: tt(m.statusHealthy) }
    case 'warming_up':
      return { tone: 'info', text: c.warmup ? tt(m.statusWarmingUpDay, { day: c.warmup.day, days: c.warmup.days }) : tt(m.statusWarmingUp) }
    case 'down':
      if (c.down?.reason === 'auth') return { tone: 'bad', text: tt(m.statusDownAuth) }
      return { tone: 'bad', text: c.down ? tt(m.statusDownSince, { since: sinceText(c.down.since, now) }) : tt(m.statusDown) }
    case 'degraded':
      return { tone: 'warn', text: c.webhook.silentSince ? tt(m.statusDegradedSilent, { since: sinceText(c.webhook.silentSince, now) }) : tt(m.statusDegraded) }
    case 'sandbox':
      return { tone: 'warn', text: tt(m.statusSandbox) }
    case 'paused':
      return { tone: 'neutral', text: tt(m.statusPaused) }
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
      {cap.kind === 'no' ? <T msg={m.notSupported} values={{ capability: cap.label, vh: (c) => <span className="vh">{c}</span> }} /> : cap.label}
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
  note,
}: {
  label: string
  value: string
  percent: number
  color: string
  valueClassName?: string
  valueStyle?: React.CSSProperties
  /** Why there is no value ("Gmail and iCloud don't report complaints"). */
  note?: string
}) {
  const w = Math.max(0, Math.min(100, percent))
  return (
    <div className="pv-m">
      <div className="l">
        <span className="k">{label}</span>
        <span className={cn('n whitespace-nowrap', valueClassName)} style={valueStyle}>
          {value}
        </span>
      </div>
      <div className="bar" aria-hidden="true">
        <i style={{ width: `${w}%`, background: color }} />
      </div>
      {note ? <span className="k text-[11.5px]">{note}</span> : null}
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
export function agoSeconds(at: string, now = new Date()): string {
  const s = Math.max(0, Math.round((now.getTime() - new Date(at).getTime()) / 1000))
  return s < 60 ? tt(m.secondsAgo, { n: s }) : ago(at, now)
}

/** "Bounce rate · limit 5% (SES review at 5%, pause at 10%)"; `short` drops the provider's levels. */
export function limitLabel(kind: 'Bounce' | 'Complaint', l: Schemas['ProviderLimitRate'], short = false): string {
  const v = { kind: kind === 'Bounce' ? 'bounce' : 'complaint', limit: limitPct(l.limit), provider: l.label ?? '' }
  if (l.label === 'warm-up guard') return tt(m.limitGuard, v)
  if (short || !l.label) return tt(m.limit, v)
  if (l.review && l.pause) return tt(m.limitReviewPause, { ...v, review: limitPct(l.review), pause: limitPct(l.pause) })
  if (l.review) return tt(m.limitReview, { ...v, review: limitPct(l.review) })
  if (l.pause) return tt(m.limitPause, { ...v, pause: limitPct(l.pause) })
  return tt(m.limitNamed, v)
}

/** A key on the left, its value on the right (provider card facts). */
export function Fact({ k, children, className }: { k: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn('flex justify-between gap-x-3 max-md:flex-wrap', className)}>
      <span className="k">{k}</span>
      {children}
    </div>
  )
}
