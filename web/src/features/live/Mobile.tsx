import type { ReactNode } from 'react'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Row } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clockSeconds, count } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { LivePermissions } from './actions'
import { CountUp } from './LiveParts'
import { ConnectingCard } from './States'
import {
  barSegments,
  mailboxColor,
  mailboxStatus,
  mobileCaption,
  mobileEta,
  mobileTiles,
  pct1,
  perSec,
  progressLegend,
  providerColor,
  providerLine,
  providerStatus,
  scopeName,
  thresholdRows,
  type Incident,
  type Snapshot,
} from './model'

/**
 * LiveMonitorMobile: the live view on a phone. Its own header (back to the
 * campaigns, the name, how fresh the events are, the state), then progress,
 * counters, thresholds, both breakdowns, the feed and the two actions.
 */
export function MobileLive({
  s,
  now,
  streaming,
  incident,
  banner,
  can,
  onPause,
  onResume,
  onCancel,
  busy,
  connecting,
}: {
  s: Snapshot
  now: Date
  streaming: boolean
  /** The first events haven't arrived: the connecting card replaces progress and counters. */
  connecting?: boolean
  incident?: Incident
  /** Shown above the progress card: a lost stream, a pause, an open incident. */
  banner?: ReactNode
  can: LivePermissions
  onPause: () => void
  onResume: () => void
  onCancel: () => void
  busy?: boolean
}) {
  const { slug, me } = useWorkspace()
  useDocumentTitle(`${s.name} · Live · ${me.workspace.name} · ReputeCast`)
  const t = mobileTiles(s)
  const paused = s.state === 'paused' && !incident
  return (
    <>
      <header className="lv-mhead">
        <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }} className="lv-back" aria-label="Back to campaigns">
          <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
            <path d="m10 3-5 5 5 5" />
          </svg>
        </ButtonLink>
        <div className="flex min-w-0 flex-1 flex-col">
          <h1 className="m-0 truncate text-[14px] font-semibold">{s.name}</h1>
          <span className="k">{mobileCaption(s, now, streaming)}</span>
        </div>
        <StateBadge s={s} incident={incident} />
      </header>
      <main className="content lv-m" id="main" tabIndex={-1}>
        {banner}
        {connecting ? (
          <ConnectingCard s={s} />
        ) : (
          <>
            <Card className={cn('flex flex-col gap-2.5 p-4', !streaming && 'lv-stale')} aria-label="Progress">
              <Row gap={10} align="baseline">
                <span className="n text-[32px] tracking-[-0.02em]">
                  <CountUp value={wholePercent(s)} />%
                </span>
                <span>
                  {count(s.sent)} of {count(s.recipients)}
                </span>
              </Row>
              <div className="bar flex gap-0.5" style={{ height: 10, borderRadius: 5 }}>
                {barSegments(s, incident).map((seg) => (
                  <i key={seg.key} className="grow-x" style={{ width: `${seg.width}%`, background: seg.color }} />
                ))}
              </div>
              <Row justify="space-between" className="k">
                <span>{mobileEta(s)}</span>
                <span>{perSec(s.rate)}</span>
              </Row>
              <div className="flex flex-wrap gap-3.5 text-[12px]">
                {progressLegend(s, incident, true).map((l) => (
                  <span key={l.key} className="flex items-center gap-1.5">
                    <Dot color={l.color} />
                    {l.text}
                  </span>
                ))}
              </div>
            </Card>
            <div className="grid grid-cols-2 gap-2.5">
              {t.map((x, i) => (
                <div key={x.key} className={cn('card tile', i === t.length - 1 && 'col-span-2')}>
                  <span className="k">{x.label}</span>
                  <span className="n text-[22px]">
                    <CountUp value={Number(x.value.replaceAll(',', '')) || 0} />
                  </span>
                  <span className="k">{x.sub}</span>
                </div>
              ))}
            </div>
          </>
        )}
        <Card className="flex flex-col">
          <div className="flex items-center justify-between px-3.5 py-3">
            <h2 className="h2">Thresholds</h2>
            <span className="k">auto-pause</span>
          </div>
          <div className="flex flex-col gap-2.5 px-3.5 pb-3">
            {thresholdRows(s).map((r) => (
              <div key={r.key} className="flex flex-col gap-[5px]">
                <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
                  <span>{r.short}</span>
                  <span className="n">{r.value}</span>
                </Row>
                <div className="bar">
                  <i style={{ width: `${r.width}%`, background: r.color }} />
                </div>
              </div>
            ))}
          </div>
        </Card>
        <Card className="flex flex-col">
          <div className="px-3.5 py-3">
            <h2 className="h2">Per mailbox provider</h2>
          </div>
          <ul className="m-0 list-none p-0">
            {s.mailboxes.map((m) => {
              const st = mailboxStatus(m, s, false)
              return (
                <li key={m.key} className="lv-row">
                  <Dot color={mailboxColor[m.key]} />
                  <span className="flex-1">{m.name}</span>
                  <span className="n">{count(m.sent)}</span>
                  <span className="k w-[52px] text-right">{pct1(m.deliveredRate)}</span>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </li>
              )
            })}
          </ul>
        </Card>
        <Card className="flex flex-col">
          <div className="px-3.5 py-3">
            <h2 className="h2">Per sending provider</h2>
          </div>
          <ul className="m-0 list-none p-0">
            {s.providers.map((p) => {
              const st = providerStatus(p, true)
              return (
                <li key={p.id} className="lv-row">
                  <Dot color={providerColor[p.color]} />
                  <div className="flex flex-1 flex-col">
                    <span>{p.name}</span>
                    <span className="k">{providerLine(p)}</span>
                  </div>
                  <span className="n">{count(p.sent)}</span>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </li>
              )
            })}
          </ul>
        </Card>
        <Card className="flex flex-col">
          <div className="flex items-center justify-between px-3.5 py-3">
            <h2 className="h2">Live events</h2>
            {streaming ? (
              <span className="badge ok">
                <span className="dot pulse-live" aria-hidden="true" />
                streaming
              </span>
            ) : (
              <Badge tone="neutral" dot>
                reconnecting
              </Badge>
            )}
          </div>
          {s.events.length === 0 ? (
            <div className="lv-row k">Delivery events appear here as providers report them.</div>
          ) : (
            <ol className="m-0 list-none p-0" aria-label="Newest events first">
              {s.events.slice(0, 4).map((ev) => (
                <li key={ev.id} className="lv-row">
                  <span className="mono k">{clockSeconds(ev.at)}</span>
                  <Badge tone={ev.tone}>{ev.short}</Badge>
                  <span className="mono text-[12px]">{ev.address}</span>
                </li>
              ))}
            </ol>
          )}
        </Card>
        {s.options.canPause || s.options.canCancel || paused ? (
          <div className="mt-1 grid grid-cols-2 gap-2.5">
            {paused ? (
              <Button variant="primary" onClick={onResume} disabledReason={can.resume} loading={busy}>
                Resume
              </Button>
            ) : (
              <Button onClick={onPause} disabledReason={can.pause} loading={busy}>
                Pause
              </Button>
            )}
            <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
              Cancel
            </Button>
          </div>
        ) : null}
      </main>
    </>
  )
}

/** The whole percent sent: never 100 before everything went out. */
function wholePercent(s: Snapshot): number {
  if (s.recipients <= 0) return 0
  if (s.sent >= s.recipients) return 100
  return Math.min(99, Math.round((s.sent / s.recipients) * 100))
}

/** The header's state word on a phone: "Sending", "Paused", "Auto-paused · Outlook", "Postal held". */
function StateBadge({ s, incident }: { s: Snapshot; incident?: Incident }) {
  if (incident?.kind === 'threshold') {
    const who = scopeName(incident)
    return (
      <Badge tone="bad" pulse>
        {who ? `Auto-paused · ${who}` : 'Auto-paused'}
      </Badge>
    )
  }
  if (incident?.kind === 'provider_outage') return <Badge tone="warn" dot>{`${incident.providerShort} held`}</Badge>
  switch (s.state) {
    case 'preparing':
      return (
        <Badge tone="info" dot>
          Starting
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="warn" dot>
          Paused
        </Badge>
      )
  }
  return (
    <span className="badge ok">
      <span className="dot pulse-live" aria-hidden="true" />
      Sending
    </span>
  )
}
