import { useState, type ReactNode } from 'react'
import { useT } from '@/i18n'
import { clock, clockSeconds, count } from '@/lib/format'
import { probe, useSystemStatus, type Maintenance } from '@/lib/systemStatus'
import { Badge } from '../ui/Badge'
import { Button, ButtonAnchor } from '../ui/Button'
import { m } from './messages'

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
  const t = useT()
  return (
    <SystemBanner
      tone="warn"
      badge={t(m.bannerOffline)}
      actions={
        <Button size="sm" onClick={onRetry} loading={retrying}>
          {t(m.bannerRetryNow)}
        </Button>
      }
    >
      {t(unreachable ? m.bannerUnreachableBody : m.bannerOfflineBody)}
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
  const t = useT()
  return (
    <SystemBanner
      tone="info"
      badge={t(m.bannerReconnecting)}
      actions={
        <Button size="sm" onClick={onReconnect}>
          {t(m.bannerReconnectNow)}
        </Button>
      }
    >
      {t(m.bannerReconnectingBody, { droppedAt: clockSeconds(droppedAt), asOf: clock(dataAsOf), attempt, max: maxAttempts })}
    </SystemBanner>
  )
}

/** The instance is read-only while an upgrade installs. */
export function MaintenanceBanner({ maintenance }: { maintenance: Maintenance }) {
  const t = useT()
  const { version, startedBy, startedAt, etaSeconds, notesUrl } = maintenance
  const minutes = etaSeconds ? Math.max(1, Math.round(etaSeconds / 60)) : undefined
  const when =
    startedAt && minutes
      ? t(m.bannerMaintenanceStartedEta, { time: clock(startedAt), minutes })
      : startedAt
        ? t(m.bannerMaintenanceStarted, { time: clock(startedAt) })
        : minutes
          ? t(m.bannerMaintenanceEta, { minutes })
          : 'none'
  return (
    <SystemBanner
      tone="neutral"
      badge={t(m.bannerMaintenance)}
      actions={
        <ButtonAnchor size="sm" href={notesUrl ?? 'https://github.com/hostrepute/reputecast/releases'}>
          {t(m.bannerWhatsNew)}
        </ButtonAnchor>
      }
    >
      {t(m.bannerMaintenanceBody, { version: version || 'none', by: startedBy || 'none', when })}
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
  period,
  usedUpAt,
  held,
  heldStream,
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
  const t = useT()
  const values = {
    provider,
    period: period ?? t(m.bannerQuotaPeriod),
    quota: count(quota),
    time: clock(usedUpAt),
    stream: heldStream ?? t(m.bannerQuotaStream),
    held: count(held),
    resumesAt,
  }
  return (
    <SystemBanner
      tone="bad"
      badge={t(m.bannerQuota)}
      actions={
        <>
          <ButtonAnchor size="sm" href={requestHref} newTab={false}>
            {t(m.bannerQuotaRequest)}
          </ButtonAnchor>
          {moveTo ? (
            <Button size="sm" onClick={moveTo.onMove}>
              {t(m.bannerQuotaMove, { provider: moveTo.provider })}
            </Button>
          ) : null}
        </>
      }
    >
      {continues ? t(m.bannerQuotaBodyContinues, { ...values, continues }) : t(m.bannerQuotaBody, values)}
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
