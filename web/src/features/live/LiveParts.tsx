import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Grid, Row } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { andList, clock, clockSeconds, count, money, pct, shortDate } from '@/lib/format'
import { useReducedMotion } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { LiveActions, LivePermissions } from './actions'
import { canaryPassedLine } from './canary'
import { m } from './messages'
import { RatesPanel } from './RatesPanel'
import {
  abLine,
  barSegments,
  complaintHeading,
  deliveredMix,
  errorRate,
  eventWord,
  failoverCaption,
  mailboxColor,
  mailboxName,
  mailboxRate,
  mailboxStatus,
  mixLabel,
  orderedMailboxes,
  pct1,
  pct2,
  perSec,
  progressLegend,
  providerColor,
  providerNote,
  providerStatus,
  scopeLine,
  sentLine,
  sentOfPlanned,
  summaryLine,
  thresholdRows,
  tiles,
  timingLine,
  whenText,
  type Incident,
  type LiveEvent,
  type Snapshot,
  type Tile,
} from './model'
import { mm } from './model.messages'

/**
 * Counts up to `value` (design system › Motion: counters count up over
 * duration-count on first paint, then ease to each new value). Reduced
 * motion shows the value at once.
 */
export function useCountUp(value: number): number {
  const reduced = useReducedMotion()
  const [shown, setShown] = useState(0)
  const from = useRef(0)
  const first = useRef(true)
  useEffect(() => {
    if (reduced) return
    const start = performance.now()
    const a = from.current
    const dur = first.current ? 1600 : 700
    first.current = false
    let raf = requestAnimationFrame(function tick(t) {
      const k = Math.min(1, (t - start) / dur)
      const v = a + (value - a) * (1 - (1 - k) ** 3)
      from.current = v
      setShown(v)
      if (k < 1) raf = requestAnimationFrame(tick)
    })
    return () => cancelAnimationFrame(raf)
  }, [value, reduced])
  return reduced ? value : shown
}

/** A whole number that counts up. */
export function CountUp({ value, format = count }: { value: number; format?: (n: number) => string }) {
  const v = useCountUp(value)
  return <>{format(Math.round(v))}</>
}

/** A card's heading bar above a table or a list (14px 18px, line below). */
export function CardBar({ title, caption }: { title: string; caption?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
      <h2 className="h2">{title}</h2>
      {caption}
    </div>
  )
}

/** The progress card: percentage, sent line, timing, the per-provider bar and its legend. */
export function ProgressCard({
  s,
  incident,
  stale,
  asOf,
  note,
  polling,
}: {
  s: Snapshot
  incident?: Incident
  stale?: boolean
  asOf?: string
  /** What a person just changed ("Rates changed: …"), shown until the state changes. */
  note?: string
  /** The stream went quiet: "Checking every 10 s · updated 90 s ago" (LiveMonitorStates). */
  polling?: string
}) {
  const t = useT()
  const sentPct = Math.min(99, s.recipients > 0 ? Math.round((s.sent / s.recipients) * 100) : 0)
  const shownPct = useCountUp(s.sent >= s.recipients && s.recipients > 0 ? 100 : sentPct)
  const segments = barSegments(s, incident)
  const legend = progressLegend(s, incident)
  const ab = incident ? '' : abLine(s.ab)
  return (
    <Card className={cn('flex flex-col gap-3 px-[22px] py-5', stale && 'lv-stale')} aria-label={t(m.progress)}>
      <div className="flex flex-wrap items-baseline gap-3.5">
        <span className="n text-[34px] tracking-[-0.02em]">{pct(Math.round(shownPct), 0)}</span>
        <span className="text-[15px]">{sentLine(s, incident)}</span>
        {polling ? (
          <span className="lv-fresh stale ml-auto">
            <span className="dot" aria-hidden="true" />
            {polling}
          </span>
        ) : stale ? (
          <Badge tone="neutral" className="ml-auto">
            {t(m.asOf, { time: asOf ?? '' })}
          </Badge>
        ) : (
          <span className="k ml-auto">{timingLine(s, incident)}</span>
        )}
      </div>
      {stale || polling ? null : <SummaryLine s={s} incident={incident} note={note} />}
      {s.canary && canaryPassedLine(s.canary) ? (
        <div className="flex min-h-[18px] items-end gap-2">
          <span aria-hidden="true" className="lv-canary-mark" style={{ flex: `0 0 ${Math.max(1, s.canary.percent)}%` }} />
          <span className="k inline-flex items-center gap-[5px]">
            <Icon name="check" size={12} strokeWidth={2.4} className="text-ok-fg" />
            {canaryPassedLine(s.canary)}
          </span>
        </div>
      ) : null}
      <div className="bar flex gap-0.5" style={{ height: 12, borderRadius: 6, background: 'var(--line-2)' }}>
        {segments.map((seg, i) => (
          <i
            key={seg.key}
            className={stale ? undefined : 'grow-x'}
            title={seg.title}
            style={{ width: `${seg.width}%`, background: seg.color, borderRadius: stale ? undefined : i === 0 ? '6px 0 0 6px' : 0 }}
          />
        ))}
      </div>
      {stale ? null : (
        <div className="flex flex-wrap gap-[18px] text-[13px]">
          {legend.map((l) => (
            <span key={l.key} className="flex items-center gap-2">
              <Dot color={l.color} />
              {l.text}
            </span>
          ))}
          {ab ? <span className="k ml-auto">{ab}</span> : null}
        </div>
      )}
    </Card>
  )
}

/**
 * The send's state in one sentence: the screen's only live region, so it
 * changes with the state, never with the counts (LiveMonitor).
 */
export function SummaryLine({ s, incident, note }: { s: Snapshot; incident?: Incident; note?: string }) {
  const line = summaryLine(s, incident)
  const tone = note ? 'ok' : line.tone
  return (
    <p className="lv-sumline" aria-live="polite">
      <Icon name={tone === 'ok' ? 'check' : line.held ? 'pause' : 'info'} size={16} strokeWidth={2.2} className={tone === 'ok' ? 'text-ok-fg' : tone === 'bad' ? 'text-bad-fg' : 'text-warn-fg'} />
      <span>{note || line.text}</span>
    </p>
  )
}

/** The five counters (delivered, bounced, complained, opened, clicked). */
export function Tiles({ tiles, className }: { tiles: Tile[]; className?: string }) {
  return (
    <Grid min={200} gap={16} className={className}>
      {tiles.map((t) => (
        <div key={t.key} className={cn('card tile', t.alert && 'border-bad-line')}>
          <span className="k">{t.label}</span>
          <span className={cn('n text-[26px]', t.alert && 'text-bad-fg')}>
            <TileValue tile={t} />
          </span>
          <span className="k">{t.sub}</span>
        </div>
      ))}
    </Grid>
  )
}

/** Counts up a count ("19,402"); other values ("$2.84") show as they are. */
export function TileValue({ tile }: { tile: Tile }) {
  return tile.num === undefined ? <>{tile.value}</> : <CountUp value={tile.num} />
}

/** The mailbox table's "Adjust rates" button and its panel (review GS-13). */
export type RatesControl = { open: boolean; toggle: () => void; disabledReason?: string; panel: ReactNode }

/**
 * "Adjust rates" while there is mail left to send (running, waiting or
 * held for a decision); onApplied gets what changed, for the summary line.
 */
export function useRatesControl(s: Snapshot, actions: LiveActions | undefined, can: LivePermissions | undefined, onApplied?: (summary: string) => void): RatesControl | undefined {
  const [open, setOpen] = useState(false)
  const adjustable = !!actions && ['sending', 'partial', 'paused'].includes(s.state) && s.mailboxes.some((mb) => mb.remaining > 0)
  if (!adjustable) return undefined
  return {
    open,
    toggle: () => setOpen(!open),
    disabledReason: can?.rates,
    panel: (
      <RatesPanel
        s={s}
        busy={actions.pending}
        onCancel={() => setOpen(false)}
        onApply={(r, summary) =>
          void actions
            .setRates(r)
            .then(() => {
              setOpen(false)
              onApplied?.(summary)
            })
            .catch(() => undefined)
        }
      />
    ),
  }
}

/** Per mailbox provider: health in the live view, activity in the decision view. */
export function MailboxTable({ s, decision, rates, caption }: { s: Snapshot; decision: boolean; rates?: RatesControl; caption?: string }) {
  const t = useT()
  const rows = orderedMailboxes(s, decision)
  return (
    <Card>
      {rates ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
          <div className="flex flex-col gap-0.5">
            <h2 className="h2">{t(m.perMailbox)}</h2>
            <span className="k">{caption ?? (decision ? t(m.perMailboxDecision) : t(m.perMailboxLive))}</span>
          </div>
          <Button size="sm" aria-expanded={rates.open} aria-controls="lv-rates" onClick={rates.toggle} disabledReason={rates.disabledReason}>
            {t(m.adjustRates)}
          </Button>
        </div>
      ) : (
        <CardBar title={t(m.perMailbox)} caption={<span className="k">{decision ? t(m.perMailboxDecision) : t(m.perMailboxLive)}</span>} />
      )}
      {rates?.open ? rates.panel : null}
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>{t(m.colMailbox)}</Th>
            <Th align="right">{t(m.colSent)}</Th>
            {decision ? <Th align="right">{t(m.colRemaining)}</Th> : null}
            <Th align="right">{t(m.colDelivered)}</Th>
            {s.deferralWindow ? <Th align="right">{t(mm.colDeferred, { minutes: s.deferralWindow })}</Th> : null}
            <Th align="right">{t(m.colBounce)}</Th>
            <Th align="right">{decision ? complaintHeading(s) : t(m.colComplaint)}</Th>
            {decision ? null : <Th align="right">{t(m.colOpened)}</Th>}
            <Th align="right">{t(m.colRate)}</Th>
            <Th>{t(m.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((mb) => {
            const st = mailboxStatus(mb, s, decision)
            return (
              <tr key={mb.key} className={mb.held ? 'bg-bad-tint' : undefined}>
                <Td>
                  <span className="flex items-center gap-2">
                    <Dot color={mailboxColor[mb.key]} />
                    {mailboxName(mb.key)}
                  </span>
                </Td>
                <Td align="right" num>
                  {count(mb.sent)}
                </Td>
                {decision ? (
                  <Td align="right" num>
                    {count(mb.remaining)}
                  </Td>
                ) : null}
                <Td align="right" num>
                  {pct1(mb.deliveredRate)}
                </Td>
                {s.deferralWindow ? (
                  <Td align="right" num>
                    {pct1(mb.deferredRate)}
                  </Td>
                ) : null}
                <Td align="right" num>
                  {pct1(mb.bounceRate)}
                </Td>
                {mb.complaintReason ? (
                  // Gmail and iCloud don't report complaints per message (review F-2).
                  <Td align="right" className="k" aria-describedby="lv-fn-cmp">
                    {t(mm.notAvailable)}
                  </Td>
                ) : (
                  <Td align="right" num className={decision && mb.held ? 'text-bad-fg' : undefined}>
                    {mb.complaintRate === undefined ? t(mm.notAvailable) : pct2(mb.complaintRate)}
                  </Td>
                )}
                {decision ? null : (
                  <Td align="right" num>
                    {pct1(mb.openRate)}
                  </Td>
                )}
                <Td align="right">{mailboxRate(mb)}</Td>
                <Td>
                  <Badge tone={st.tone} dot>
                    {st.text}
                  </Badge>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      <MailboxNotes s={s} />
    </Card>
  )
}

/**
 * The mailbox table's notes: why Gmail's and iCloud's complaints read n/a
 * (with Gmail's daily spam rate when Postmaster Tools is connected), and
 * how company domains are counted.
 */
export function MailboxNotes({ s }: { s: Snapshot }) {
  const t = useT()
  const unreported = s.mailboxes.filter((mb) => mb.complaintReason).map((mb) => mailboxName(mb.key))
  return (
    <div className="lv-fns">
      {unreported.length ? (
        <p className="lv-fn" id="lv-fn-cmp">
          {[t(mm.naComplaints, { names: andList(unreported) }), s.postmaster ? t(mm.postmasterRate, { rate: pct2(s.postmaster.spamRate), day: shortDate(s.postmaster.day) }) : '']
            .filter(Boolean)
            .join(' ')}
        </p>
      ) : null}
      <p className="lv-fn">{[t(mm.mxNote), s.deferralWindow ? t(mm.deferredNote, { minutes: s.deferralWindow }) : ''].filter(Boolean).join(' ')}</p>
    </div>
  )
}

/** Per sending provider: share and rates in the live view, what is held in the outage view. */
export function ProviderTable({ s, outage }: { s: Snapshot; outage?: Incident }) {
  const t = useT()
  const decision = Boolean(outage)
  return (
    <Card>
      <CardBar title={t(m.perProvider)} caption={<span className="k">{decision ? failoverCaption(s) : t(mm.whoDelivers)}</span>} />
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>{t(m.colProvider)}</Th>
            <Th align="right">{t(m.colSent)}</Th>
            {decision ? <Th align="right">{t(m.colRemaining)}</Th> : <Th align="right">{t(m.colShare)}</Th>}
            <Th align="right">{t(m.colRate)}</Th>
            <Th align="right">{t(m.colDelivered)}</Th>
            <Th align="right">{t(m.colBounce)}</Th>
            {decision ? null : <Th align="right">{t(m.colComplaint)}</Th>}
            <Th align="right">{decision ? t(m.colApiErrors3) : t(m.colApiErrors)}</Th>
            {decision ? null : <Th align="right">{t(mm.colDoneAbout)}</Th>}
            <Th>{t(m.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {s.providers.map((p) => {
            const st = providerStatus(p)
            const held = decision && (p.routeStatus === 'held' || p.id === outage?.providerId)
            const note = providerNote(p, decision)
            return (
              <tr key={p.id} className={held ? 'bg-[#FFFBEB]' : undefined}>
                <Td>
                  <span className="flex items-center gap-2">
                    <Dot color={providerColor[p.color]} />
                    {p.name}
                  </span>
                  {note ? <div className="k">{note}</div> : null}
                </Td>
                <Td align="right" num>
                  {decision ? count(p.sent) : sentOfPlanned(p)}
                </Td>
                <Td align="right" num>
                  {decision ? count(p.remaining) : pct(Math.round(p.share), 0)}
                </Td>
                <Td align="right" num>
                  {perSec(p.rate)}
                </Td>
                <Td align="right" num>
                  {pct1(p.deliveredRate)}
                  {p.infersDelivery ? <div className="k">{t(m.deliveredInferred)}</div> : null}
                </Td>
                <Td align="right" num>
                  {pct1(p.bounceRate)}
                </Td>
                {decision ? null : (
                  <Td align="right" num>
                    {pct2(p.complaintRate ?? 0)}
                  </Td>
                )}
                <Td align="right" num className={held && (p.apiErrorRate ?? 0) >= 50 ? 'text-bad-fg' : undefined}>
                  {errorRate(p.apiErrorRate)}
                </Td>
                {decision ? null : (
                  <Td align="right" num>
                    {p.doneAt && p.remaining > 0 ? whenText(p.doneAt, s.at) : '—'}
                  </Td>
                )}
                <Td>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      <div className="lv-fns">
        {decision ? null : <p className="lv-fn">{t(mm.providerNotes)}</p>}
        {s.providers.some((p) => p.infersDelivery) ? <p className="lv-fn">{t(m.inferredNote)}</p> : null}
      </div>
    </Card>
  )
}

/** The live event feed: newest first, addresses masked. */
export function EventsCard({ s, streaming, limit = 7 }: { s: Snapshot; streaming: boolean; limit?: number }) {
  const t = useT()
  const [held, setHeld] = useState<Snapshot['events'] | null>(null)
  const events = (held ?? s.events).slice(0, limit)
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <div className="flex flex-col gap-0.5">
          <h2 className="h2">{t(m.liveEvents)}</h2>
          <span className="k">{s.eventsVia ? t(mm.eventsCaption, { via: s.eventsVia }) : t(mm.eventsCaptionPlain)}</span>
        </div>
        <div className="flex items-center gap-2">
          {held ? (
            <Badge tone="neutral" dot>
              {t(mm.feedPaused)}
            </Badge>
          ) : streaming ? (
            <span className="badge ok">
              <span className="dot pulse-live" aria-hidden="true" />
              {t(m.streaming)}
            </span>
          ) : (
            <Badge tone="neutral" dot>
              {t(m.reconnecting)}
            </Badge>
          )}
          <Button size="sm" aria-pressed={held !== null} onClick={() => setHeld(held ? null : s.events)}>
            {held ? t(mm.resumeUpdates) : t(mm.pauseUpdates)}
          </Button>
        </div>
      </div>
      {events.length === 0 ? (
        <div className="lv-ev first k">{t(m.eventsEmpty)}</div>
      ) : (
        <ol className="m-0 list-none p-0" aria-label={t(m.eventsList)}>
          {events.map((ev, i) => (
            <EventLine key={ev.id} ev={ev} first={i === 0} />
          ))}
        </ol>
      )}
    </Card>
  )
}

function EventLine({ ev, first }: { ev: LiveEvent; first: boolean }) {
  return (
    <li className={cn('lv-ev', first && 'first')}>
      <span className="mono k">{clockSeconds(ev.at)}</span>
      <Badge tone={ev.tone}>{eventWord(ev.label)}</Badge>
      <span className="mono">{ev.address}</span>
      {ev.detail ? <span className="k">{ev.detail}</span> : null}
    </li>
  )
}

/** "Edit for this send": opens the thresholds dialog, or says who can. */
export type ThresholdsEdit = { onEdit: () => void; disabledReason?: string }

/**
 * Auto-pause thresholds (LiveMonitor, review GS-6): each rate against its
 * limit with its state and evidence; "Edit for this send" changes them
 * while the send runs or waits (GS-2).
 */
export function ThresholdsCard({ s, campaignId, edit }: { s: Snapshot; campaignId: string; edit?: ThresholdsEdit }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Card className="flex flex-col p-0" aria-labelledby="lv-thr-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <div className="flex flex-col gap-0.5">
          <h2 className="h2" id="lv-thr-h">
            {t(m.thresholds)}
          </h2>
          <span className="k">{scopeLine(s)}</span>
        </div>
        <div className="lv-thr-links flex flex-wrap items-center gap-4 text-[13px]">
          <Link to="/w/$workspace/providers/traffic-rules" params={{ workspace: slug }}>
            {t(m.workspaceDefaults)}
          </Link>
          {edit ? (
            <Button variant="ghost" size="sm" className="lv-linkbtn" onClick={edit.onEdit} disabledReason={edit.disabledReason}>
              {t(m.editThresholdsLive)}
            </Button>
          ) : (
            <Link to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
              {t(m.editThresholds)}
            </Link>
          )}
        </div>
      </div>
      <div className="grid gap-x-7 gap-y-[18px] px-[18px] pt-4 pb-[18px]" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))' }}>
        {thresholdRows(s).map((r) => (
          <div key={r.key} className="lv-thr">
            <div className="lv-thr-h">
              <span>
                <b>{r.label}</b>
                <span className="k">{r.sub}</span>
              </span>
              <Badge tone={r.state.tone} dot>
                {r.state.text}
              </Badge>
            </div>
            <div className="bar" aria-hidden="true">
              <i style={{ width: `${r.width}%`, background: r.color }} />
            </div>
            <div className="lv-thr-v">
              <span className="n">{r.value}</span>
              {r.evidence ? <span>{r.evidence}</span> : null}
            </div>
          </div>
        ))}
        <div className="flex items-center rounded-lg border border-line bg-[#F9FAFB] px-3 py-2.5">
          <p className="lv-fn">{t(mm.thresholdsFootnote)}</p>
        </div>
      </div>
    </Card>
  )
}

/** One "label … value" line of a side card. */
export function Fact({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
      <span className="k">{label}</span>
      <span className="n">{children}</span>
    </Row>
  )
}

/** Send details: sender, audience, route, cost, who scheduled it. */
export function DetailsCard({ s }: { s: Snapshot }) {
  const t = useT()
  const d = s.details
  return (
    <Card className="flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">{t(m.sendDetails)}</h2>
      <Fact label={t(m.detailFrom)}>{d.from || '—'}</Fact>
      <Fact label={t(m.detailAudience)}>{d.audience ? `${d.audience} · ${count(d.audienceCount)}` : count(d.audienceCount)}</Fact>
      <Fact label={t(m.detailRoute)}>{d.route || '—'}</Fact>
      {s.canary ? (
        <Fact label={t(m.detailCanaryLabel)}>
          {t(mm.detailCanary, {
            size: count(s.canary.size),
            seeds: s.canary.seeds,
            state: t(mm.canaryState, { phase: s.canary.phase, time: clock(s.canary.decidedAt ?? s.canary.until ?? s.at) }),
          })}
        </Fact>
      ) : null}
      <Fact label={t(m.detailCost)}>{t(m.detailCostValue, { cost: money(d.cost), planned: money(d.costPlanned) })}</Fact>
      {d.watchHours ? <Fact label={t(m.detailAfterLabel)}>{t(mm.detailAfter, { hours: d.watchHours })}</Fact> : null}
      {d.scheduledBy ? <Fact label={t(m.detailScheduledBy)}>{d.scheduledBy}</Fact> : null}
    </Card>
  )
}

/** Delivered by mailbox provider: a stacked bar and its legend. */
export function DeliveredMixCard({ s }: { s: Snapshot }) {
  const t = useT()
  const mix = deliveredMix(s)
  return (
    <Card className="flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">{t(m.deliveredBy)}</h2>
      {mix.length === 0 ? (
        <span className="k">{t(m.sharesEmpty)}</span>
      ) : (
        <>
          <div className="flex gap-0.5 overflow-hidden" style={{ height: 14, borderRadius: 7 }} role="img" aria-label={mixLabel(mix)}>
            {mix.map((x) => (
              <div key={x.key} style={{ width: `${x.percent}%`, background: mailboxColor[x.key] }} />
            ))}
          </div>
          <div className="flex flex-wrap gap-x-3.5 gap-y-2 text-[12px]">
            {mix.map((x) => (
              <span key={x.key} className="flex items-center gap-1.5">
                <Dot color={mailboxColor[x.key]} />
                {`${x.name} ${pct(x.percent, 0)}`}
              </span>
            ))}
          </div>
        </>
      )}
    </Card>
  )
}

/** The live view's body (LiveMonitor): progress, counters, breakdowns, feed and the side cards. */
export function LiveBody({
  s,
  campaignId,
  streaming,
  stale,
  asOf,
  incident,
  edit,
  actions,
  can,
  polling,
}: {
  s: Snapshot
  campaignId: string
  streaming: boolean
  stale?: boolean
  asOf?: string
  incident?: Incident
  edit?: ThresholdsEdit
  actions?: LiveActions
  can?: LivePermissions
  /** The freshness label while the monitor checks every 10 s. */
  polling?: string
}) {
  // A note lasts while the state it was written in does.
  const state = `${s.state}|${incident?.id ?? ''}`
  const [note, setNote] = useState<{ state: string; text: string }>()
  const rates = useRatesControl(s, stale ? undefined : actions, can, (text) => setNote({ state, text }))
  return (
    <>
      <ProgressCard s={s} incident={incident} stale={stale} asOf={asOf} polling={polling} note={note?.state === state ? note.text : undefined} />
      <div className={cn('flex flex-col gap-5 max-md:gap-3', stale && 'lv-stale')}>
        <Tiles tiles={tiles(s)} />
        <ThresholdsCard s={s} campaignId={campaignId} edit={edit} />
        <MailboxTable s={s} decision={false} rates={rates} />
        <div className="lv-split">
          <div className="lv-split-main">
            <EventsCard s={s} streaming={streaming} />
          </div>
          <div className="lv-split-side">
            <DetailsCard s={s} />
            <DeliveredMixCard s={s} />
          </div>
        </div>
        {/* Full width below, so all nine columns fit (LiveMonitor). */}
        <ProviderTable s={s} />
      </div>
    </>
  )
}
