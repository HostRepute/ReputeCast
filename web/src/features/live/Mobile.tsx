import { useRef, useState, type ReactNode } from 'react'
import { m as shell } from '@/components/shell/messages'
import { Icon } from '@/components/icons/Icon'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Row } from '@/components/ui/Layout'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, clockSeconds, count, pct } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { LiveActions, LivePermissions } from './actions'
import { canaryPassedLine } from './canary'
import { CountUp, MailboxNotes, SummaryLine, type ThresholdsEdit } from './LiveParts'
import { m } from './messages'
import { RatesPanel } from './RatesPanel'
import { ConnectingCard } from './States'
import {
  autoPausedText,
  barSegments,
  eventWord,
  mailboxColor,
  mailboxName,
  mailboxRate,
  mailboxStatus,
  mobileTiles,
  pct1,
  pct2,
  perSec,
  progressLegend,
  providerColor,
  providerStatus,
  routeDoneLine,
  thresholdRows,
  whoLine,
  type Incident,
  type Snapshot,
} from './model'
import { mm } from './model.messages'

/**
 * LiveMonitorMobile: the live view on a phone, with its own header. Pause
 * is the primary action in a bar that stays above the tab bar; Adjust rates
 * and Cancel send are in its menu (review GS-14).
 */
export function MobileLive({
  s,
  streaming,
  freshness,
  incident,
  banner,
  can,
  actions,
  edit,
  onPause,
  onResume,
  onCancel,
  busy,
  connecting,
}: {
  s: Snapshot
  streaming: boolean
  /** "Live · updated 1 s ago", or the polling label when the stream is stale. */
  freshness: { text: string; stale: boolean }
  /** The first events haven't arrived: the connecting card replaces progress and counters. */
  connecting?: boolean
  incident?: Incident
  /** Shown above the progress card: a lost stream, a pause, an open incident. */
  banner?: ReactNode
  can: LivePermissions
  actions: LiveActions
  /** "Edit" on the thresholds card: this send's thresholds. */
  edit?: ThresholdsEdit
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
  const [ratesOpen, setRatesOpen] = useState(false)
  const [note, setNote] = useState<{ state: string; text: string }>()
  const state = `${s.state}|${incident?.id ?? ''}`
  const mailboxes = useRef<HTMLDivElement>(null)
  const adjustable = ['sending', 'partial', 'paused'].includes(s.state) && s.mailboxes.some((mb) => mb.remaining > 0)
  const openRates = () => {
    setRatesOpen(true)
    mailboxes.current?.scrollIntoView({ block: 'start' })
  }
  const done = routeDoneLine(s)
  // "Pause updates" keeps the feed as it is while the send goes on.
  const [held, setHeld] = useState<Snapshot['events'] | null>(null)
  const events = held ?? s.events
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
          <span className={cn('lv-fresh', freshness.stale && 'stale')}>
            <span className="dot" aria-hidden="true" />
            {freshness.text}
          </span>
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
              {streaming ? <SummaryLine s={s} incident={incident} note={note?.state === state ? note.text : undefined} /> : null}
              {s.canary && canaryPassedLine(s.canary) ? (
                <div className="flex min-h-[16px] items-end gap-1.5">
                  <span aria-hidden="true" className="lv-canary-mark" style={{ flex: `0 0 ${Math.max(1, s.canary.percent)}%` }} />
                  <span className="k inline-flex items-center gap-[5px] text-[12px]">
                    <Icon name="check" size={12} strokeWidth={2.4} className="text-ok-fg" />
                    {canaryPassedLine(s.canary)}
                  </span>
                </div>
              ) : null}
              <div className="bar flex gap-0.5" style={{ height: 10, borderRadius: 5 }}>
                {barSegments(s, incident).map((seg) => (
                  <i key={seg.key} className="grow-x" style={{ width: `${seg.width}%`, background: seg.color }} />
                ))}
              </div>
              <Row justify="space-between" className="k">
                <span>{done}</span>
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
          <div className="flex items-center justify-between px-3.5 py-1.5">
            <h2 className="h2">{t(m.thresholdsShort)}</h2>
            {edit ? (
              <Button variant="ghost" className="lv-linkbtn" onClick={edit.onEdit} disabledReason={edit.disabledReason}>
                {t(m.editThresholds)}
              </Button>
            ) : (
              <span className="k">{t(m.autoPause)}</span>
            )}
          </div>
          <div className="flex flex-col gap-3 px-3.5 pb-3">
            {thresholdRows(s).map((r) => (
              <div key={r.key} className="flex flex-col gap-[5px]">
                <Row justify="space-between" gap={8} align="normal" className="text-[13px]">
                  <span className="flex min-w-0 flex-col">
                    <b className="font-semibold">{r.label}</b>
                    <span className="k">{r.sub}</span>
                  </span>
                  <Badge tone={r.state.tone} dot>
                    {r.state.text}
                  </Badge>
                </Row>
                <div className="bar" aria-hidden="true">
                  <i style={{ width: `${r.width}%`, background: r.color }} />
                </div>
                <Row justify="space-between" gap={8} align="normal" className="text-[12.5px]">
                  <span className="n">{r.value}</span>
                  {r.evidence ? <span className="k text-right">{r.evidence}</span> : null}
                </Row>
              </div>
            ))}
            <p className="lv-fn">{t(mm.thresholdsFootnote)}</p>
          </div>
        </Card>
        <div ref={mailboxes} className="scroll-mt-4">
        <Card className="flex flex-col">
          <div className="flex items-center justify-between gap-2 px-3.5 py-3">
            <h2 className="h2">{t(m.perMailbox)}</h2>
            {adjustable ? (
              <Button size="sm" aria-expanded={ratesOpen} aria-controls="lv-rates" onClick={() => setRatesOpen(!ratesOpen)} disabledReason={can.rates}>
                {t(m.adjustRates)}
              </Button>
            ) : null}
          </div>
          {ratesOpen && adjustable ? (
            <RatesPanel
              s={s}
              busy={actions.pending}
              onCancel={() => setRatesOpen(false)}
              onApply={(r, summary) =>
                void actions
                  .setRates(r)
                  .then(() => {
                    setRatesOpen(false)
                    setNote({ state, text: summary })
                  })
                  .catch(() => undefined)
              }
            />
          ) : null}
          <ul className="m-0 list-none p-0">
            {s.mailboxes.map((mb) => {
              const st = mailboxStatus(mb, s, false)
              return (
                <li key={mb.key} className="lv-row lv-mrow">
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span className="flex items-center gap-2">
                      <Dot color={mailboxColor[mb.key]} />
                      <b className="font-semibold">{mailboxName(mb.key)}</b>
                      <span className="k">· {t(m.mobileSent, { count: count(mb.sent) })}</span>
                    </span>
                    <span className="k">
                      {t(mm.phoneMailbox, {
                        delivered: pct1(mb.deliveredRate),
                        deferred: mb.deferredRate === undefined ? '—' : pct1(mb.deferredRate),
                        bounce: pct1(mb.bounceRate),
                        complaints: mb.complaintReason || mb.complaintRate === undefined ? t(mm.notAvailable) : pct2(mb.complaintRate),
                        rate: mailboxRate(mb),
                      })}
                    </span>
                  </div>
                  <Badge tone={st.tone} dot>
                    {st.text}
                  </Badge>
                </li>
              )
            })}
          </ul>
          <MailboxNotes s={s} />
        </Card>
        </div>
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
                    <span className="k">
                      {t(mm.phoneProvider, { rate: perSec(p.rate), delivered: pct1(p.deliveredRate ?? 0), done: p.doneAt && p.remaining > 0 ? clock(p.doneAt) : 'none' })}
                    </span>
                  </div>
                  <span className="n">{count(p.sent)}</span>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </li>
              )
            })}
          </ul>
        </Card>
        <Card className="flex flex-col">
          <div className="flex items-center justify-between gap-2 px-3.5 py-1.5">
            <h2 className="h2">{t(m.liveEvents)}</h2>
            <Button aria-pressed={held !== null} onClick={() => setHeld(held ? null : s.events)}>
              {held ? t(mm.resumeUpdates) : t(mm.pauseUpdates)}
            </Button>
          </div>
          {!streaming && !held ? (
            <div className="px-3.5 pb-2">
              <Badge tone="neutral" dot>
                {t(m.reconnecting)}
              </Badge>
            </div>
          ) : null}
          {events.length === 0 ? (
            <div className="lv-row k">{t(m.eventsEmpty)}</div>
          ) : (
            <ol className="m-0 list-none p-0" aria-label={t(m.eventsList)}>
              {events.slice(0, 4).map((ev) => (
                <li key={ev.id} className="lv-row">
                  <span className="mono k">{clockSeconds(ev.at)}</span>
                  <Badge tone={ev.tone}>{eventWord(ev.short)}</Badge>
                  <span className="mono text-[12px]">{ev.address}</span>
                </li>
              ))}
            </ol>
          )}
        </Card>
        <p className="k m-0 px-2 text-center text-[12px]">{whoLine(s)}</p>
        {s.options.canPause || s.options.canCancel || paused ? (
          <div className="lv-pbar" role="group" aria-label={t(mm.sendActions)}>
            {paused ? (
              <Button variant="primary" className="flex-1" onClick={onResume} disabledReason={can.resume} loading={busy}>
                {t(m.resume)}
              </Button>
            ) : (
              <Button variant="primary" className="flex-1" onClick={onPause} disabledReason={can.pause} loading={busy}>
                {t(m.pause)}
              </Button>
            )}
            <Menu label={t(mm.moreActions)} trigger={<Button iconOnly icon="more" aria-label={t(mm.moreActions)} />}>
              {adjustable ? <MenuItem onSelect={openRates}>{t(m.adjustRates)}</MenuItem> : null}
              {adjustable && s.options.canCancel ? <MenuSeparator /> : null}
              {s.options.canCancel ? (
                <MenuItem onSelect={onCancel} className="text-bad-fg" disabled={!!can.cancel}>
                  {t(m.cancelSend)}
                </MenuItem>
              ) : null}
            </Menu>
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
