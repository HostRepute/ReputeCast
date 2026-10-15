import { useEffect, useState, type ReactNode } from 'react'
import { T, useT } from '@/i18n'
import { countdown, duration, spanWords } from '@/lib/format'
import type { SessionClock } from '@/lib/sessionClock'
import { clearRateLimit, type RateLimit } from '@/lib/systemStatus'
import { Badge, type Tone } from '../ui/Badge'
import { Button } from '../ui/Button'
import { m } from './messages'

/**
 * "Timeouts and limits" (SystemStates board): a card with a title, a state
 * badge, one or two sentences and the actions. They float bottom-left, above
 * the toasts, until resolved.
 */
export function SystemNotice({
  title,
  badge,
  tone,
  children,
  actions,
  alert,
}: {
  title: ReactNode
  badge: ReactNode
  tone: Tone
  children: ReactNode
  actions?: ReactNode
  /** Time-critical notices are announced assertively. */
  alert?: boolean
}) {
  return (
    <section className="card notice" role={alert ? 'alert' : 'status'} aria-label={typeof title === 'string' ? title : undefined}>
      <div className="nh">
        <h2 className="nt">{title}</h2>
        <Badge tone={tone}>{badge}</Badge>
      </div>
      <span className="nb">{children}</span>
      {actions ? <div className="na">{actions}</div> : null}
    </section>
  )
}

/**
 * Two minutes before the session ends: extend it (idle timeout) or know
 * when you'll be asked to sign in again (lifetime).
 */
export function SessionEndingNotice({
  clock,
  onExtend,
  onSignOut,
  extending,
  workNote,
}: {
  clock: SessionClock
  onExtend: () => void
  onSignOut: () => void
  extending?: boolean
  /** What happens to unsaved work, from the screen: "the draft autosaves". */
  workNote?: string
}) {
  const t = useT()
  const minutes = (ms: number) => duration(ms / 60_000)
  const idle = clock.endsBy === 'idle'
  return (
    <SystemNotice
      alert
      title={t(m.sessionEndingTitle, { countdown: countdown(clock.msLeft) })}
      badge={t(m.sessionEndingBadge)}
      tone="warn"
      actions={
        <>
          {clock.extendable ? (
            <Button variant="primary" onClick={onExtend} loading={extending}>
              {t(m.sessionExtend, { span: spanWords(clock.extendMs) })}
            </Button>
          ) : null}
          <Button onClick={onSignOut}>{t(m.sessionSignOutNow)}</Button>
        </>
      }
    >
      {idle
        ? t(m.sessionIdleBody, { inactive: minutes(clock.idleForMs), work: workNote ?? t(m.sessionWorkStays) })
        : t(m.sessionLifetimeBody, { signedIn: minutes(clock.signedInForMs) })}
    </SystemNotice>
  )
}

function useSecondsLeft(until: number): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (until <= now) return
    const h = window.setTimeout(() => setNow(Date.now()), Math.min(1000, until - now))
    return () => window.clearTimeout(h)
  }, [until, now])
  return Math.max(0, Math.ceil((until - now) / 1000))
}

/**
 * The API answered 429. For an API key the screen passes the key and its
 * numbers; in the browser it says when requests resume.
 */
export function RateLimitNotice({
  limit,
  apiKey,
  actions,
}: {
  limit: RateLimit
  /** Key usage for Settings › API keys: rc_live_bf2…, 1,200 sent, 600 allowed. */
  apiKey?: { masked: string; sent: string; allowed: string }
  actions?: ReactNode
}) {
  const t = useT()
  const left = useSecondsLeft(limit.until)
  const resume = left > 0 ? t(m.rateLimitResumeIn, { seconds: left }) : t(m.rateLimitResumeNow)
  return (
    <SystemNotice
      title={t(m.rateLimitTitle)}
      badge="429"
      tone="bad"
      actions={
        actions ?? (
          <Button size="sm" onClick={clearRateLimit} disabledReason={left > 0 ? t(m.rateLimitResumeIn, { seconds: left }) : undefined}>
            {left > 0 ? t(m.rateLimitWaiting) : t(m.rateLimitDismiss)}
          </Button>
        )
      }
    >
      {apiKey ? (
        <T
          msg={m.rateLimitKeyBody}
          values={{ key: apiKey.masked, sent: apiKey.sent, allowed: apiKey.allowed, resume, mono: (chunks) => <span className="mono">{chunks}</span> }}
        />
      ) : (
        t(m.rateLimitBrowserBody, { resume })
      )}
    </SystemNotice>
  )
}

/** A background job that's slower than usual, shown where it was started. */
export function JobNotice({
  title,
  children,
  progress,
  onCancel,
}: {
  title: ReactNode
  children: ReactNode
  /** 0–100 */
  progress: number
  onCancel?: () => void
}) {
  const t = useT()
  const pct = Math.max(0, Math.min(100, Math.round(progress)))
  return (
    <section className="card notice" role="status" aria-label={typeof title === 'string' ? title : undefined}>
      <div className="nh">
        <h2 className="nt">{title}</h2>
        <Badge tone="info">{t(m.jobNoticeBadge)}</Badge>
      </div>
      <span className="nb">{children}</span>
      <div className="progress">
        <span className="track" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={t(m.jobNoticeProgress)}>
          <i style={{ width: `${pct}%` }} />
        </span>
        <span className="k">{t(m.jobPercent, { pct })}</span>
        {onCancel ? (
          <Button size="sm" onClick={onCancel}>
            {t(m.jobNoticeCancel)}
          </Button>
        ) : null}
      </div>
    </section>
  )
}
