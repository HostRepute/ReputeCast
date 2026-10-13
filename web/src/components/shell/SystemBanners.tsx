import { useState, type ReactNode } from 'react'
import { clock, clockSeconds, count } from '@/lib/format'
import { probe, useSystemStatus, type Maintenance } from '@/lib/systemStatus'
import { Badge } from '../ui/Badge'
import { Button, ButtonAnchor } from '../ui/Button'

/**
 * Banners at the top of the content column (SystemStates board): a full-width
 * strip with a state badge, one sentence that says what happened and what
 * still works, and the action. A banner always has an action.
 */
export function SystemBanner({
  tone,
  badge,
  children,
  actions,
}: {
  tone: 'warn' | 'info' | 'neutral' | 'bad'
  badge: ReactNode
  children: ReactNode
  actions: ReactNode
}) {
  return (
    <div className={`sysbanner tone-${tone}`} role={tone === 'bad' ? 'alert' : 'status'}>
      <Badge tone={tone} dot>
        {badge}
      </Badge>
      <span className="msg">{children}</span>
      {actions}
    </div>
  )
}

/** Offline (no network) or unreachable (ReputeCast doesn't answer). */
export function OfflineBanner({ unreachable, onRetry, retrying }: { unreachable?: boolean; onRetry: () => void; retrying?: boolean }) {
  return (
    <SystemBanner
      tone="warn"
      badge="Offline"
      actions={
        <Button size="sm" onClick={onRetry} loading={retrying}>
          Retry now
        </Button>
      }
    >
      {unreachable ? "Can't reach ReputeCast." : "You're offline."} Edits save on this device and sync when the connection is back. Sending and live stats are paused.
    </SystemBanner>
  )
}

/** A live event stream (live monitor) dropped and is reconnecting. */
export function ReconnectingBanner({
  droppedAt,
  dataAsOf,
  attempt,
  maxAttempts,
  onReconnect,
}: {
  droppedAt: string | number | Date
  /** When the numbers on the page were last updated. */
  dataAsOf: string | number | Date
  attempt: number
  maxAttempts: number
  onReconnect: () => void
}) {
  return (
    <SystemBanner
      tone="info"
      badge="Reconnecting"
      actions={
        <Button size="sm" onClick={onReconnect}>
          Reconnect now
        </Button>
      }
    >
      Live event stream dropped at {clockSeconds(droppedAt)}. Counts below are from {clock(dataAsOf)}. Reconnecting automatically, attempt {attempt} of {maxAttempts}.
    </SystemBanner>
  )
}

/** The instance is read-only while an upgrade installs. */
export function MaintenanceBanner({ maintenance }: { maintenance: Maintenance }) {
  const { version, startedBy, startedAt, etaSeconds, notesUrl } = maintenance
  const minutes = etaSeconds ? Math.max(1, Math.round(etaSeconds / 60)) : undefined
  const who = startedBy ? ` by ${startedBy}` : ''
  const when = [startedAt ? `started ${clock(startedAt)}` : '', minutes ? `about ${minutes} minute${minutes === 1 ? '' : 's'}` : ''].filter(Boolean).join(', ')
  return (
    <SystemBanner
      tone="neutral"
      badge="Read-only · upgrade"
      actions={
        <ButtonAnchor size="sm" href={notesUrl ?? 'https://github.com/hostrepute/reputecast/releases'}>
          What&apos;s new
        </ButtonAnchor>
      }
    >
      {version ? `ReputeCast ${version} is being installed${who}` : `ReputeCast is being upgraded${who}`}
      {when ? ` (${when})` : ''}. Reading works; edits and sends resume when it finishes.
    </SystemBanner>
  )
}

/**
 * A sending provider's quota is used up: what is held, until when, what
 * keeps going, and the two ways out.
 */
export function QuotaBanner({
  provider,
  quota,
  period = '24-hour',
  usedUpAt,
  held,
  heldStream = 'Broadcasts',
  resumesAt,
  continues,
  requestHref,
  moveTo,
}: {
  provider: string
  quota: number
  period?: string
  usedUpAt: string | number | Date
  held: number
  heldStream?: string
  /** When the quota resets, already formatted ("00:00 UTC"). */
  resumesAt: string
  /** "automations continue on Postmark" */
  continues?: string
  requestHref: string
  moveTo?: { provider: string; onMove: () => void }
}) {
  return (
    <SystemBanner
      tone="bad"
      badge="Quota reached"
      actions={
        <>
          <ButtonAnchor size="sm" href={requestHref} newTab={false}>
            Request a higher quota
          </ButtonAnchor>
          {moveTo ? (
            <Button size="sm" onClick={moveTo.onMove}>
              Move the rest to {moveTo.provider}
            </Button>
          ) : null}
        </>
      }
    >
      {provider} {period} quota of {count(quota)} is used up at {clock(usedUpAt)}. {heldStream} hold {count(held)} recipients until {resumesAt}
      {continues ? `; ${continues}` : ''}.
    </SystemBanner>
  )
}

/**
 * The banners every signed-in screen gets: offline/unreachable and
 * read-only maintenance. Mounted by the app shell; full-screen editors can
 * render it at the top of their own layout.
 */
export function SystemBanners() {
  const status = useSystemStatus()
  const [retrying, setRetrying] = useState(false)
  const retry = async () => {
    setRetrying(true)
    await probe()
    setRetrying(false)
  }
  return (
    <>
      {status.connection !== 'online' ? <OfflineBanner unreachable={status.connection === 'unreachable'} onRetry={() => void retry()} retrying={retrying} /> : null}
      {status.maintenance ? <MaintenanceBanner maintenance={status.maintenance} /> : null}
    </>
  )
}
