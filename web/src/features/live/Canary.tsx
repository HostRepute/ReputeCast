import { Link } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { T, useT } from '@/i18n'
import { clock, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import type { LiveActions, LivePermissions } from './actions'
import { canaryProgress, canarySentLine, canarySummary, canaryTiles, canaryTitle, canaryWhy, etaFrom, gateRow, seedRows, untilPlus, type Canary } from './canary'
import { mc } from './canary.messages'
import { DetailsCard, EventsCard, MailboxTable, ProviderTable, Tiles, type RatesControl } from './LiveParts'
import { RatesPanel } from './RatesPanel'
import type { Snapshot } from './model'

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
  streaming,
}: {
  s: Snapshot
  c: Canary
  actions: LiveActions
  can: LivePermissions
  now: Date
  streaming: boolean
}) {
  const t = useT()
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
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-col gap-5" style={{ flex: '2 1 560px' }}>
          <MailboxTable s={s} decision={false} rates={rates} caption={t(mc.mailboxCaption)} />
          <EventsCard s={s} streaming={streaming} />
        </div>
        <div className="flex flex-col gap-4 max-md:min-w-0" style={{ flex: '1 1 320px', maxWidth: 400 }}>
          <DetailsCard s={s} />
        </div>
      </div>
      <ProviderTable s={s} />
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
        <Icon name="shield" size={36} className="flex-none text-ok-fg" />
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
          {t(mc.queuedLegend, { count: count(c.queued) })}
        </span>
        {s.ab ? <span className="k ml-auto">{t(mc.abNote)}</span> : null}
      </div>
    </Card>
  )
}
