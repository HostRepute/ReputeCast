import { Link } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Table, Td, Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { andList, clock, count, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import type { LiveActions, LivePermissions } from './actions'
import {
  canaryMailboxRows,
  canaryMinNote,
  canaryProgress,
  canarySentLine,
  canarySummary,
  canaryTiles,
  canaryTitle,
  canaryWhy,
  etaFrom,
  gateRow,
  seedRows,
  untilPlus,
  type Canary,
} from './canary'
import { mc } from './canary.messages'
import { Tiles, type RatesControl } from './LiveParts'
import { m } from './messages'
import { RatesPanel } from './RatesPanel'
import { errorRate, mailboxColor, pct1, pct2, providerColor, providerNote, providerStatus, type Snapshot } from './model'
import { mm } from './model.messages'

/**
 * The live monitor during a canary (LiveMonitorCanary; review GS-10): why
 * it runs, its gates and seed placement, the decision (continue now, watch
 * 10 more minutes, stop), then the canary's own numbers. The main send's
 * rates can be set before it starts.
 */
export function CanaryView({
  s,
  c,
  actions,
  can,
  now,
}: {
  s: Snapshot
  c: Canary
  actions: LiveActions
  can: LivePermissions
  now: Date
}) {
  const [ratesOpen, setRatesOpen] = useState(false)
  const [note, setNote] = useState<{ key: string; text: string }>()
  const key = `${c.phase}|${c.extended}`
  const live = c.phase === 'sending' || c.phase === 'watching'
  const rates: RatesControl | undefined = live
    ? {
        open: ratesOpen,
        toggle: () => setRatesOpen(!ratesOpen),
        disabledReason: can.rates,
        panel: (
          <RatesPanel
            s={s}
            startsAt={c.until ?? undefined}
            busy={actions.pending}
            onCancel={() => setRatesOpen(false)}
            onApply={(r, summary) =>
              void actions
                .setRates(r)
                .then(() => {
                  setRatesOpen(false)
                  setNote({ key, text: summary })
                })
                .catch(() => undefined)
            }
          />
        ),
      }
    : undefined
  return (
    <>
      <CanaryCard s={s} c={c} actions={actions} can={can} now={now} note={note?.key === key ? note.text : undefined} />
      <CanaryProgress s={s} c={c} now={now} />
      <Tiles tiles={canaryTiles(s)} />
      <CanaryMailboxTable s={s} rates={rates} />
      <CanaryProviderTable s={s} />
    </>
  )
}

/** The canary card: title, why, gates, seeds and the decision. */
export function CanaryCard({ s, c, actions, can, now, note }: { s: Snapshot; c: Canary; actions: LiveActions; can: LivePermissions; now: Date; note?: string }) {
  const t = useT()
  const id = useId()
  const { slug } = useWorkspace()
  const why = canaryWhy(c)
  return (
    <Card className="lv-canary" aria-labelledby={`${id}-t`}>
      <div className="lv-cnr-top">
        <CanaryIcon phase={c.phase} />
        <div className="flex min-w-0 flex-col gap-1.5" style={{ flex: '1 1 320px' }}>
          <h2 id={`${id}-t`} className="lv-cnr-title">
            {canaryTitle(c, now)}
          </h2>
          <span className="text-ink-3">{canarySentLine(c)}</span>
          <p className="lv-sumline" aria-live="polite">
            <Icon name={c.gates.some((g) => g.state === 'failing') ? 'info' : 'check'} size={16} strokeWidth={2.2} />
            <span>{note || canarySummary(c)}</span>
          </p>
        </div>
      </div>
      <div className="flex flex-col gap-1.5 border-b border-line-2 px-[22px] py-3.5">
        <p className="m-0 text-[13px]">
          <b>{t(mc.why)}</b> {why}{' '}
          <Link to="/w/$workspace/providers/traffic-rules" params={{ workspace: slug }}>
            {t(mc.rulesLink)}
          </Link>
        </p>
        <p className="lv-fn">{t(mc.honest, { hours: s.watch?.hours ?? 72 })}</p>
      </div>
      <div className="flex flex-wrap">
        <div className="flex min-w-0 flex-col px-[22px] pt-4 pb-[18px]" style={{ flex: '3 1 460px' }}>
          <h3 className="h2 mb-1.5">{t(mc.gates)}</h3>
          {c.gates.map((g) => {
            const r = gateRow(g, s)
            return (
              <div key={r.key} className="lv-gate">
                <span className="lv-gn">{r.name}</span>
                <span className="lv-gv">
                  {r.value}
                  {r.note ? <span className="k mt-1 block">{r.note}</span> : null}
                  {r.bar !== undefined ? (
                    <span className="bar mt-[5px] block max-w-[240px]" aria-hidden="true">
                      <i style={{ width: `${r.bar}%`, background: 'var(--ink-4)' }} />
                    </span>
                  ) : null}
                </span>
                <span className="lv-gl k">{r.limit}</span>
                <Badge tone={r.state.tone} dot className="lv-gs">
                  {r.state.text}
                </Badge>
              </div>
            )
          })}
          {c.placement && c.placement.byMailbox.length ? <SeedDetails c={c} /> : null}
        </div>
        <CanaryDecision s={s} c={c} actions={actions} can={can} now={now} />
      </div>
    </Card>
  )
}

/** Where the seed copies landed, per mailbox provider. */
function SeedDetails({ c }: { c: Canary }) {
  const t = useT()
  const { slug } = useWorkspace()
  const p = c.placement!
  const total = p.inbox + p.promotions + p.spam
  return (
    <details className="lv-seeds" open>
      <summary>{t(mc.seedsTitle)}</summary>
      <div className="flex flex-col gap-2 pt-2">
        <div className="lv-seedbar" role="img" aria-label={t(mc.seedsBar, { seeds: c.seeds, inbox: p.inbox, promotions: p.promotions, spam: p.spam })}>
          {total > 0 ? (
            <>
              <i style={{ flex: `${p.inbox} 1 0%`, background: 'var(--ok-strong, #15803d)' }} />
              <i style={{ flex: `${p.promotions} 1 0%`, background: '#93c5fd' }} />
              <i style={{ flex: `${p.spam} 1 0%`, background: 'var(--bad-strong)' }} />
            </>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1.5 text-[12px] text-ink-3">
          <span className="flex items-center gap-1.5">
            <Dot color="var(--ok-strong, #15803d)" />
            {t(mc.seedLegendInbox, { count: p.inbox })}
          </span>
          <span className="flex items-center gap-1.5">
            <Dot color="#93c5fd" />
            {t(mc.seedLegendPromotions, { count: p.promotions })}
          </span>
          <span className="flex items-center gap-1.5">
            <Dot color="var(--bad-strong)" />
            {t(mc.seedLegendSpam, { count: p.spam })}
          </span>
        </div>
        <div>
          {seedRows(p).map((r) => (
            <div key={r.key} className="lv-seedrow">
              <span>
                <b>{r.name}</b> <span className="k">{r.note}</span>
              </span>
              <span>{r.value}</span>
            </div>
          ))}
        </div>
        <p className="lv-fn">
          <T
            msg={mc.seedsNote}
            values={{
              a: (x) => (
                <Link to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }}>
                  {x}
                </Link>
              ),
            }}
          />
        </p>
      </div>
    </details>
  )
}

/** Continue now, watch 10 more minutes, or stop (with a confirmation). */
function CanaryDecision({ s, c, actions, can, now }: { s: Snapshot; c: Canary; actions: LiveActions; can: LivePermissions; now: Date }) {
  const t = useT()
  const id = useId()
  const [confirm, setConfirm] = useState(false)
  if (c.phase === 'stopped') {
    return (
      <div className="lv-cnr-dec">
        <div className="flex flex-col gap-1.5 rounded-[10px] border border-line bg-bg-2 p-3">
          <span className="font-semibold">{c.decidedBy ? t(mc.stoppedBy, { time: clock(c.decidedAt ?? now), name: c.decidedBy }) : t(mc.stoppedAt, { time: clock(c.decidedAt ?? now) })}</span>
          <span className="k">{t(mc.stoppedLine, { queued: count(c.queued) })}</span>
        </div>
      </div>
    )
  }
  const held = c.phase === 'held'
  const until = c.until ? clock(c.until) : clock(now)
  const plus = untilPlus(c, now)
  return (
    <div className="lv-cnr-dec">
      <h3 className="h2 mb-2">{t(mc.choose)}</h3>
      <div className="lv-act">
        <Button variant="primary" onClick={() => void actions.continueCanary().catch(() => undefined)} disabledReason={can.approve} loading={actions.pending}>
          {t(mc.continueNow)}
        </Button>
        <span className="k">
          {held
            ? t(mc.continueHeldBody, { queued: count(c.queued), eta: etaFrom(c, now) })
            : t(mc.continueNowBody, { until, eta: etaFrom(c, now) })}
        </span>
      </div>
      {held ? null : (
        <div className="lv-act">
          <Button onClick={() => void actions.extendCanary().catch(() => undefined)} disabledReason={can.decide} loading={actions.pending}>
            {t(mc.watchMore)}
          </Button>
          <span className="k">{t(mc.watchMoreBody, { queued: count(c.queued), until: clock(plus), eta: etaFrom(c, plus) })}</span>
        </div>
      )}
      <div className="lv-act">
        <Button variant="danger" aria-expanded={confirm} aria-controls={`${id}-stop`} onClick={() => setConfirm(!confirm)} disabledReason={can.decide}>
          {t(mc.stop)}
        </Button>
        <span className="k">{t(mc.stopBody, { queued: count(c.queued), size: count(c.size) })}</span>
      </div>
      {confirm ? (
        <div id={`${id}-stop`} className="lv-stop" role="group" aria-label={t(mc.stop)}>
          <span className="font-semibold text-bad-fg">{t(mc.stopConfirm, { name: s.name, queued: count(c.queued) })}</span>
          <span className="text-[13px] text-bad-fg">{t(mc.stopConfirmBody)}</span>
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => setConfirm(false)}>{t(mc.keepWatching)}</Button>
            <Button variant="danger-primary" loading={actions.pending} onClick={() => void actions.stopCanary().then(() => setConfirm(false)).catch(() => undefined)}>
              {t(mc.stop)}
            </Button>
          </div>
        </div>
      ) : null}
      {held ? null : <p className="lv-fn mt-2 rounded-lg border border-line bg-bg-2 px-3 py-2.5">{t(mc.autoNote, { until })}</p>}
    </div>
  )
}

/** The canary's progress: who's waiting, what went through which provider. */
function CanaryProgress({ s, c, now }: { s: Snapshot; c: Canary; now: Date }) {
  const t = useT()
  const p = canaryProgress(c, s, now)
  const total = Math.max(s.recipients, 1)
  return (
    <Card className="flex flex-col gap-3 px-[22px] py-5" aria-label={t(mc.progressLabel)}>
      <div className="flex flex-wrap items-baseline gap-3.5">
        <span className="n text-[34px] tracking-[-0.02em]">{count(c.queued)}</span>
        <span className="text-[15px]">{t(mc.queuedBig)}</span>
        <span className="k ml-auto">{p.line}</span>
      </div>
      <div
        className="bar flex gap-0.5"
        style={{ height: 12, borderRadius: 6, background: 'var(--line-2)' }}
        role="img"
        aria-label={t(mc.barLabel, { parts: p.parts.map((x) => x.text).join(', '), queued: count(c.queued) })}
      >
        {p.parts.map((x, i) => (
          <i key={x.key} style={{ width: `${(x.value / total) * 100}%`, minWidth: 3, background: x.color, borderRadius: i === 0 ? '6px 0 0 6px' : 0 }} />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-[18px] gap-y-2 text-[13px]">
        {p.parts.map((x) => (
          <span key={x.key} className="flex items-center gap-2">
            <Dot color={x.color} />
            {x.text}
          </span>
        ))}
        <span className="flex items-center gap-2">
          <Dot color="#c9ced6" />
          {p.queued}
        </span>
        {s.ab ? <span className="k ml-auto">{t(mc.abNote)}</span> : null}
      </div>
    </Card>
  )
}

/** The canary card's mark: an eye while it's watched, a warning when held, grey once stopped. */
function CanaryIcon({ phase }: { phase: Canary['phase'] }) {
  if (phase === 'held') {
    return (
      <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
        <circle cx="20" cy="20" r="19" fill="#FECACA" />
        <path d="M20 11v11M20 27v1" stroke="#991B1B" strokeWidth="3" strokeLinecap="round" />
      </svg>
    )
  }
  const stopped = phase === 'stopped'
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
      <circle cx="20" cy="20" r="19" fill={stopped ? '#E5E7EB' : '#CCFBF1'} />
      <path d="M10 20s3.6-6 10-6 10 6 10 6-3.6 6-10 6-10-6-10-6z" stroke={stopped ? '#4B5563' : '#0F766E'} strokeWidth="2.2" fill="none" strokeLinejoin="round" />
      <circle cx="20" cy="20" r="2.6" fill={stopped ? '#4B5563' : '#0F766E'} />
    </svg>
  )
}

/** Per mailbox provider during the canary: what it did, the main send's rates, whether thresholds can trip yet. */
function CanaryMailboxTable({ s, rates }: { s: Snapshot; rates?: RatesControl }) {
  const t = useT()
  const rows = canaryMailboxRows(s)
  const unreported = s.mailboxes.filter((mb) => mb.complaintReason).map((mb) => mb.name)
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <div className="flex flex-col gap-0.5">
          <h2 className="h2">{t(m.perMailbox)}</h2>
          <span className="k">{t(mc.mailboxCaption)}</span>
        </div>
        {rates ? (
          <Button size="sm" aria-expanded={rates.open} aria-controls="lv-rates" onClick={rates.toggle} disabledReason={rates.disabledReason}>
            {t(m.adjustRates)}
          </Button>
        ) : null}
      </div>
      {rates?.open ? rates.panel : null}
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>{t(m.colMailbox)}</Th>
            <Th align="right">{t(mc.colCanarySent)}</Th>
            <Th align="right">{t(m.colDelivered)}</Th>
            <Th align="right">{t(mc.colDeferredOnce)}</Th>
            <Th align="right">{t(m.colBounce)}</Th>
            <Th align="right">{t(m.colComplaint)}</Th>
            <Th align="right">{t(mc.colMainRate)}</Th>
            <Th>{t(mc.colThresholds)}</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.key}>
              <Td>
                <span className="flex items-center gap-2">
                  <Dot color={mailboxColor[r.key]} />
                  {r.name}
                </span>
              </Td>
              <Td align="right" num>
                {r.sent}
              </Td>
              <Td align="right" num>
                {r.delivered}
              </Td>
              <Td align="right" num>
                {r.deferred}
              </Td>
              <Td align="right" num>
                {r.bounce}
              </Td>
              {r.na ? (
                <Td align="right" className="k" aria-describedby="lv-cnr-na">
                  {r.complaint}
                </Td>
              ) : (
                <Td align="right" num>
                  {r.complaint}
                </Td>
              )}
              <Td align="right">{r.rate}</Td>
              <Td>
                <Badge tone={r.state.tone} dot>
                  {r.state.text}
                </Badge>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
      <div className="lv-fns">
        {unreported.length ? (
          <p className="lv-fn" id="lv-cnr-na">
            {[t(mm.naComplaints, { names: andList(unreported) }), s.postmaster ? t(mm.postmasterRate, { rate: pct2(s.postmaster.spamRate), day: shortDate(s.postmaster.day) }) : '']
              .filter(Boolean)
              .join(' ')}
          </p>
        ) : null}
        <p className="lv-fn">{`${t(mm.mxNote)} ${canaryMinNote(s)}`}</p>
      </div>
    </Card>
  )
}

/** Per sending provider during the canary: what each sent and what waits on it. */
function CanaryProviderTable({ s }: { s: Snapshot }) {
  const t = useT()
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <h2 className="h2">{t(m.perProvider)}</h2>
        <span className="k">{t(mm.whoDelivers)}</span>
      </div>
      <Table density="compact" className="lv-tbl" phoneWrap>
        <thead>
          <tr>
            <Th>{t(m.colProvider)}</Th>
            <Th align="right">{t(mc.colCanarySent)}</Th>
            <Th align="right">{t(mc.colQueued)}</Th>
            <Th align="right">{t(m.colDelivered)}</Th>
            <Th align="right">{t(m.colBounce)}</Th>
            <Th align="right">{t(m.colApiErrors)}</Th>
            <Th>{t(m.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {s.providers.map((p) => {
            const st = providerStatus(p)
            const note = providerNote(p, false)
            return (
              <tr key={p.id}>
                <Td>
                  <span className="flex items-center gap-2">
                    <Dot color={providerColor[p.color]} />
                    {p.name}
                  </span>
                  {note ? <div className="k">{note}</div> : null}
                </Td>
                <Td align="right" num>
                  {count(p.sent)}
                </Td>
                <Td align="right" num>
                  {count(p.remaining)}
                </Td>
                <Td align="right" num>
                  {pct1(p.deliveredRate)}
                  {p.infersDelivery ? <div className="k">{t(m.deliveredInferred)}</div> : null}
                </Td>
                <Td align="right" num>
                  {pct1(p.bounceRate)}
                </Td>
                <Td align="right" num>
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
