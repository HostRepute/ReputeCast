import type { ReactNode } from 'react'
import { m as shell } from '@/components/shell/messages'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Row } from '@/components/ui/Layout'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clockSeconds, count, pct } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { LivePermissions } from './actions'
import { CountUp } from './LiveParts'
import { m } from './messages'
import { ConnectingCard } from './States'
import {
  autoPausedText,
  barSegments,
  eventWord,
  mailboxColor,
  mailboxName,
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
  thresholdRows,
  type Incident,
  type Snapshot,
} from './model'
import { mm } from './model.messages'

/** LiveMonitorMobile: the live view on a phone, with its own header. */
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
  const t = useT()
  const { slug, me } = useWorkspace()
  useDocumentTitle(t(shell.docTitleWorkspace, { title: t(m.documentTitle, { name: s.name }), workspace: me.workspace.name }))
  const tiles = mobileTiles(s)
  const paused = s.state === 'paused' && !incident
  return (
    <>
      <header className="lv-mhead">
        <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }} className="lv-back" aria-label={t(m.backToCampaigns)}>
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
            <Card className={cn('flex flex-col gap-2.5 p-4', !streaming && 'lv-stale')} aria-label={t(m.progress)}>
              <Row gap={10} align="baseline">
                <span className="n text-[32px] tracking-[-0.02em]">
                  <CountUp value={wholePercent(s)} format={(n) => pct(n, 0)} />
                </span>
                <span>{t(m.sentOfTotal, { sent: count(s.sent), total: count(s.recipients) })}</span>
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
              {tiles.map((x, i) => (
                <div key={x.key} className={cn('card tile', i === tiles.length - 1 && 'col-span-2')}>
                  <span className="k">{x.label}</span>
                  <span className="n text-[22px]">
                    <CountUp value={x.num ?? 0} />
                  </span>
                  <span className="k">{x.sub}</span>
                </div>
              ))}
            </div>
          </>
        )}
        <Card className="flex flex-col">
          <div className="flex items-center justify-between px-3.5 py-3">
            <h2 className="h2">{t(m.thresholdsShort)}</h2>
            <span className="k">{t(m.autoPause)}</span>
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
            <h2 className="h2">{t(m.perMailbox)}</h2>
          </div>
          <ul className="m-0 list-none p-0">
            {s.mailboxes.map((mb) => {
              const st = mailboxStatus(mb, s, false)
              return (
                <li key={mb.key} className="lv-row">
                  <Dot color={mailboxColor[mb.key]} />
                  <span className="flex-1">{mailboxName(mb.key)}</span>
                  <span className="n">{count(mb.sent)}</span>
                  <span className="k w-[52px] text-right">{pct1(mb.deliveredRate)}</span>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </li>
              )
            })}
          </ul>
        </Card>
        <Card className="flex flex-col">
          <div className="px-3.5 py-3">
            <h2 className="h2">{t(m.perProvider)}</h2>
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
            <h2 className="h2">{t(m.liveEvents)}</h2>
            {streaming ? (
              <span className="badge ok">
                <span className="dot pulse-live" aria-hidden="true" />
                {t(m.streaming)}
              </span>
            ) : (
              <Badge tone="neutral" dot>
                {t(m.reconnecting)}
              </Badge>
            )}
          </div>
          {s.events.length === 0 ? (
            <div className="lv-row k">{t(m.eventsEmpty)}</div>
          ) : (
            <ol className="m-0 list-none p-0" aria-label={t(m.eventsList)}>
              {s.events.slice(0, 4).map((ev) => (
                <li key={ev.id} className="lv-row">
                  <span className="mono k">{clockSeconds(ev.at)}</span>
                  <Badge tone={ev.tone}>{eventWord(ev.short)}</Badge>
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
                {t(m.resume)}
              </Button>
            ) : (
              <Button onClick={onPause} disabledReason={can.pause} loading={busy}>
                {t(m.pause)}
              </Button>
            )}
            <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
              {t(m.cancelShort)}
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
  const t = useT()
  if (incident?.kind === 'threshold') {
    return (
      <Badge tone="bad" pulse>
        {autoPausedText(incident)}
      </Badge>
    )
  }
  if (incident?.kind === 'provider_outage') {
    return (
      <Badge tone="warn" dot>
        {t(mm.badgeProviderHeld, { provider: incident.providerShort })}
      </Badge>
    )
  }
  switch (s.state) {
    case 'preparing':
      return (
        <Badge tone="info" dot>
          {t(mm.badgeStarting)}
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="warn" dot>
          {t(mm.badgePaused)}
        </Badge>
      )
  }
  return (
    <span className="badge ok">
      <span className="dot pulse-live" aria-hidden="true" />
      {t(mm.badgeSending)}
    </span>
  )
}
