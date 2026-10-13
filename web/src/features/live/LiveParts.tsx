import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Badge, Dot } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { Grid, Row } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { cn } from '@/lib/cn'
import { clockSeconds, count, money } from '@/lib/format'
import { useReducedMotion } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import {
  abLine,
  barSegments,
  complaintHeading,
  deliveredMix,
  errorRate,
  failoverCaption,
  mailboxColor,
  mailboxRate,
  mailboxStatus,
  mixLabel,
  orderedMailboxes,
  pct1,
  pct2,
  perSec,
  progressLegend,
  progressText,
  providerColor,
  providerNote,
  providerStatus,
  scopeLine,
  sentLine,
  sentOfPlanned,
  thresholdRows,
  tiles,
  timingLine,
  type Incident,
  type LiveEvent,
  type Snapshot,
  type Tile,
} from './model'

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
export function ProgressCard({ s, incident, stale, asOf }: { s: Snapshot; incident?: Incident; stale?: boolean; asOf?: string }) {
  const pct = Math.min(99, s.recipients > 0 ? Math.round((s.sent / s.recipients) * 100) : 0)
  const shownPct = useCountUp(s.sent >= s.recipients && s.recipients > 0 ? 100 : pct)
  const segments = barSegments(s, incident)
  const legend = progressLegend(s, incident)
  const ab = incident ? '' : abLine(s.ab)
  return (
    <Card className={cn('flex flex-col gap-3 px-[22px] py-5', stale && 'lv-stale')} aria-label="Progress">
      <div className="flex flex-wrap items-baseline gap-3.5">
        <span className="n text-[34px] tracking-[-0.02em]">{Math.round(shownPct)}%</span>
        <span className="vh" aria-live="polite">
          {progressText(s)} sent
        </span>
        <span className="text-[15px]">{sentLine(s, incident)}</span>
        {stale ? (
          <Badge tone="neutral" className="ml-auto">
            as of {asOf}
          </Badge>
        ) : (
          <span className="k ml-auto">{timingLine(s, incident)}</span>
        )}
      </div>
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

/** The five counters (delivered, bounced, complained, opened, clicked). */
export function Tiles({ tiles, className }: { tiles: Tile[]; className?: string }) {
  return (
    <Grid min={200} gap={16} className={className}>
      {tiles.map((t) => (
        <div key={t.key} className={cn('card tile', t.alert && 'border-bad-line')}>
          <span className="k">{t.label}</span>
          <span className={cn('n text-[26px]', t.alert && 'text-bad-fg')}>
            <TileValue value={t.value} />
          </span>
          <span className="k">{t.sub}</span>
        </div>
      ))}
    </Grid>
  )
}

/** Counts up a formatted count ("19,402"); other values ("$2.84") show as they are. */
function TileValue({ value }: { value: string }) {
  const n = /^[\d,]+$/.test(value) ? Number(value.replaceAll(',', '')) : null
  return n === null ? <>{value}</> : <CountUp value={n} />
}

/** Per mailbox provider: health in the live view, activity in the decision view. */
export function MailboxTable({ s, decision }: { s: Snapshot; decision: boolean }) {
  const rows = orderedMailboxes(s, decision)
  return (
    <Card>
      <CardBar title="Per mailbox provider" caption={<span className="k">{decision ? 'thresholds apply per mailbox provider' : 'where recipients read the mail'}</span>} />
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>Mailbox provider</Th>
            <Th align="right">Sent</Th>
            {decision ? <Th align="right">Remaining</Th> : null}
            <Th align="right">Delivered</Th>
            <Th align="right">Bounce</Th>
            <Th align="right">{decision ? complaintHeading(s) : 'Complaint'}</Th>
            {decision ? null : <Th align="right">Opened</Th>}
            <Th align="right">Rate</Th>
            <Th>Status</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((m) => {
            const st = mailboxStatus(m, s, decision)
            return (
              <tr key={m.key} className={m.held ? 'bg-bad-tint' : undefined}>
                <Td>
                  <span className="flex items-center gap-2">
                    <Dot color={mailboxColor[m.key]} />
                    {m.name}
                  </span>
                </Td>
                <Td align="right" num>
                  {count(m.sent)}
                </Td>
                {decision ? (
                  <Td align="right" num>
                    {count(m.remaining)}
                  </Td>
                ) : null}
                <Td align="right" num>
                  {pct1(m.deliveredRate)}
                </Td>
                <Td align="right" num>
                  {pct1(m.bounceRate)}
                </Td>
                <Td align="right" num className={decision && m.held ? 'text-bad-fg' : undefined}>
                  {pct2(m.complaintRate ?? 0)}
                </Td>
                {decision ? null : (
                  <Td align="right" num>
                    {pct1(m.openRate)}
                  </Td>
                )}
                <Td align="right">{mailboxRate(m)}</Td>
                <Td>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
    </Card>
  )
}

/** Per sending provider: share and rates in the live view, what is held in the outage view. */
export function ProviderTable({ s, outage }: { s: Snapshot; outage?: Incident }) {
  const decision = Boolean(outage)
  return (
    <Card>
      <CardBar title="Per sending provider" caption={<span className="k">{decision ? failoverCaption(s) : 'who delivers the mail'}</span>} />
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>Sending provider</Th>
            <Th align="right">Sent</Th>
            {decision ? <Th align="right">Remaining</Th> : <Th align="right">Share</Th>}
            <Th align="right">Rate</Th>
            <Th align="right">Delivered</Th>
            <Th align="right">Bounce</Th>
            {decision ? null : <Th align="right">Complaint</Th>}
            <Th align="right">{decision ? 'API errors · 3 min' : 'API errors'}</Th>
            <Th>Status</Th>
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
                  {decision ? count(p.remaining) : `${Math.round(p.share)}%`}
                </Td>
                <Td align="right" num>
                  {perSec(p.rate)}
                </Td>
                <Td align="right" num>
                  {pct1(p.deliveredRate)}
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
                <Td>
                  <Badge tone={st.tone}>{st.text}</Badge>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
    </Card>
  )
}

/** The live event feed: newest first, addresses masked. */
export function EventsCard({ s, streaming, limit = 7 }: { s: Snapshot; streaming: boolean; limit?: number }) {
  const events = s.events.slice(0, limit)
  return (
    <Card>
      <CardBar
        title="Live events"
        caption={
          streaming ? (
            <span className="badge ok">
              <span className="dot pulse-live" aria-hidden="true" />
              streaming
            </span>
          ) : (
            <Badge tone="neutral" dot>
              reconnecting
            </Badge>
          )
        }
      />
      {events.length === 0 ? (
        <div className="lv-ev first k">Delivery events appear here as providers report them.</div>
      ) : (
        <ol className="m-0 list-none p-0" aria-label="Newest events first">
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
      <Badge tone={ev.tone}>{ev.label}</Badge>
      <span className="mono">{ev.address}</span>
      {ev.detail ? <span className="k">{ev.detail}</span> : null}
    </li>
  )
}

/** Auto-pause thresholds: each rate against its limit. */
export function ThresholdsCard({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  return (
    <Card className="flex flex-col gap-3.5 px-5 py-[18px]">
      <div className="flex items-center justify-between">
        <h2 className="h2">Auto-pause thresholds</h2>
        <Link to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }} className="k">
          Edit
        </Link>
      </div>
      {thresholdRows(s).map((r) => (
        <div key={r.key} className="flex flex-col gap-1.5">
          <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
            <span>{r.label}</span>
            <span className="n">{r.value}</span>
          </Row>
          <div className="bar">
            <i style={{ width: `${r.width}%`, background: r.color }} />
          </div>
        </div>
      ))}
      <span className="k">{scopeLine(s)}</span>
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
  const d = s.details
  return (
    <Card className="flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">Send details</h2>
      <Fact label="From">{d.from || '—'}</Fact>
      <Fact label="Audience">{d.audience ? `${d.audience} · ${count(d.audienceCount)}` : count(d.audienceCount)}</Fact>
      <Fact label="Route">{d.route || '—'}</Fact>
      <Fact label="Cost so far">{`${money(d.cost)} of ${money(d.costPlanned)}`}</Fact>
      {d.scheduledBy ? <Fact label="Scheduled by">{d.scheduledBy}</Fact> : null}
    </Card>
  )
}

/** Delivered by mailbox provider: a stacked bar and its legend. */
export function DeliveredMixCard({ s }: { s: Snapshot }) {
  const mix = deliveredMix(s)
  return (
    <Card className="flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">Delivered by mailbox provider</h2>
      {mix.length === 0 ? (
        <span className="k">Shares appear once the first deliveries are reported.</span>
      ) : (
        <>
          <div className="flex gap-0.5 overflow-hidden" style={{ height: 14, borderRadius: 7 }} role="img" aria-label={mixLabel(mix)}>
            {mix.map((m) => (
              <div key={m.key} style={{ width: `${m.percent}%`, background: mailboxColor[m.key] }} />
            ))}
          </div>
          <div className="flex flex-wrap gap-x-3.5 gap-y-2 text-[12px]">
            {mix.map((m) => (
              <span key={m.key} className="flex items-center gap-1.5">
                <Dot color={mailboxColor[m.key]} />
                {m.name} {m.percent}%
              </span>
            ))}
          </div>
        </>
      )}
    </Card>
  )
}

/** The live view's body (LiveMonitor): progress, counters, breakdowns, feed and the side cards. */
export function LiveBody({ s, campaignId, streaming, stale, asOf, incident }: { s: Snapshot; campaignId: string; streaming: boolean; stale?: boolean; asOf?: string; incident?: Incident }) {
  return (
    <>
      <ProgressCard s={s} incident={incident} stale={stale} asOf={asOf} />
      <div className={cn('flex flex-col gap-5 max-md:gap-3', stale && 'lv-stale')}>
        <Tiles tiles={tiles(s)} />
        <div className="flex flex-wrap items-start gap-5">
          <div className="flex min-w-0 flex-col gap-5" style={{ flex: '2 1 560px' }}>
            <MailboxTable s={s} decision={false} />
            <ProviderTable s={s} />
            <EventsCard s={s} streaming={streaming} />
          </div>
          <div className="flex flex-col gap-4 max-md:min-w-0" style={{ flex: '1 1 320px', maxWidth: 400 }}>
            <ThresholdsCard s={s} campaignId={campaignId} />
            <DetailsCard s={s} />
            <DeliveredMixCard s={s} />
          </div>
        </div>
      </div>
    </>
  )
}
