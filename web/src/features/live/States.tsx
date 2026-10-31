import { useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Banner } from '@/components/ui/Note'
import { mw } from '@/features/providers/warmup.messages'
import { t as tr, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { andList, clock, clockSeconds, count, pct, weekdayDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import type { LiveActions, LivePermissions } from './actions'
import { TileValue } from './LiveParts'
import { m } from './messages'
import {
  autoPausedText,
  cancelledText,
  continuingProviders,
  finalTiles,
  finishedText,
  incidentTitle,
  noticeText,
  openNotices,
  pausedText,
  providerColor,
  providerResultLines,
  scheduledLine,
  scopeName,
  startingText,
  warmupHolds,
  watchRows,
  watchTitle,
  type Incident,
  type Provider,
  type Snapshot,
} from './model'
import { mm } from './model.messages'

/** LiveMonitorStates › connecting: the send started, the first events haven't arrived. */
export function ConnectingCard({ s }: { s: Snapshot }) {
  const tr = useT()
  const t = startingText(s)
  const first = s.providers[0]
  const pct = s.recipients > 0 ? Math.max(1, (s.sent / s.recipients) * 100) : 1
  return (
    <Card className="flex flex-col gap-3 px-[22px] py-5" aria-busy="true">
      <div className="flex flex-wrap items-center gap-3.5">
        <span className="spin lv-spin" aria-hidden="true" />
        <div className="flex flex-col" role="status">
          <span className="text-[18px] font-bold">{t.title}</span>
          <span className="k">{t.body}</span>
        </div>
      </div>
      <div className="bar" style={{ height: 12, borderRadius: 6 }}>
        <i style={{ width: `${Math.min(100, pct)}%`, background: first ? providerColor[first.color] : 'var(--chart-ses)' }} />
      </div>
      <Grid min={180} gap={12}>
        {(
          [
            ['delivered', mm.tileDelivered, 80],
            ['bounced', mm.tileBounced, 60],
            ['complained', mm.tileComplained, 50],
            ['opened', mm.tileOpened, 70],
            ['clicked', mm.tileClicked, 60],
          ] as const
        ).map(([key, label, w]) => (
          <div key={key} className="flex flex-col gap-1.5">
            <span className="k">{tr(label)}</span>
            <Skeleton w={w} h={26} />
          </div>
        ))}
      </Grid>
    </Card>
  )
}

/** LiveMonitorStates › error: the browser lost the event stream. */
export function StreamLostBanner({ title, body, onReconnect, busy }: { title: string; body: string; onReconnect: () => void; busy?: boolean }) {
  const t = useT()
  return (
    <div className="flex flex-wrap items-center gap-4 rounded-[12px] border border-bad-line bg-[#FEF2F2] px-5 py-4">
      <svg width="32" height="32" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
        <circle cx="20" cy="20" r="19" fill="#FECACA" />
        <path d="M20 11v11M20 27v1" stroke="#991B1B" strokeWidth="3" strokeLinecap="round" />
      </svg>
      <div role="alert" className="flex flex-col gap-0.5 max-md:min-w-[120px]" style={{ flex: '1 1 300px' }}>
        <span className="text-[16px] font-bold text-[#7F1D1D]">{title}</span>
        <span className="text-[13px] text-[#7F1D1D]">{body}</span>
      </div>
      <Button onClick={onReconnect} loading={busy}>
        {t(m.reconnectNow)}
      </Button>
    </div>
  )
}

/**
 * LiveMonitorStates › stale: the stream says nothing although it looks
 * open (a buffering proxy); the monitor checks every 10 s meanwhile.
 */
export function StaleBanner({ title, body, onReconnect }: { title: string; body: string; onReconnect: () => void }) {
  const t = useT()
  return (
    <div className="flex flex-wrap items-center gap-4 rounded-[12px] border border-[#FCD34D] bg-[#FFFBEB] px-5 py-4">
      <svg width="32" height="32" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
        <circle cx="20" cy="20" r="19" fill="#FEF3C7" />
        <path d="M20 12v8l5 3" stroke="#92400E" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      </svg>
      <div role="status" className="flex flex-col gap-0.5 max-md:min-w-[120px]" style={{ flex: '1 1 300px' }}>
        <span className="text-[16px] font-bold text-[#78350F]">{title}</span>
        <span className="text-[13px] text-[#78350F]">{body}</span>
      </div>
      <Button onClick={onReconnect}>{t(m.reconnect)}</Button>
    </div>
  )
}

/** "No update for 90 s", and what the monitor does about it. */
export function staleText(s: Snapshot, heardAt: Date | undefined, now: Date): { title: string; body: string } {
  const quiet = Math.max(0, Math.round((now.getTime() - (heardAt ?? new Date(s.at)).getTime()) / 1000))
  const title = quiet < 120 ? tr(mm.staleTitle, { seconds: quiet }) : tr(mm.staleTitleMinutes, { minutes: Math.round(quiet / 60) })
  return { title, body: tr(mm.staleBody, { time: clockSeconds(s.at) }) }
}

/** LiveMonitorStates › complete: what the send did, and the way to the report. */
export function FinishedView({ s, campaignId, actions, can }: { s: Snapshot; campaignId: string; actions?: LiveActions; can?: LivePermissions }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const t = finishedText(s)
  const shareReason = usePermissionReason('reports:share')
  const params = { workspace: slug, campaignId }
  return (
    <>
      <div className="flex flex-wrap items-center gap-[18px] rounded-[12px] border border-[#86EFAC] bg-ok-tint px-[22px] py-[18px]" role="status">
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
          <circle cx="20" cy="20" r="19" fill="#BBF7D0" />
          <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
        </svg>
        <div className="flex flex-col gap-0.5 max-md:min-w-[120px]" style={{ flex: '1 1 300px' }}>
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#14532D]">{t.title}</span>
          <span className="text-[#14532D]">{t.body}</span>
        </div>
        <div className="flex flex-wrap gap-2.5">
          {shareReason ? (
            <Button disabledReason={shareReason}>{tr(m.shareLink)}</Button>
          ) : (
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={params} hash="share">
              {tr(m.shareLink)}
            </ButtonLink>
          )}
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
            {tr(m.viewReport)}
          </ButtonLink>
        </div>
      </div>
      {actions ? <ComplaintAlerts s={s} actions={actions} can={can} /> : null}
      <ComplaintWatchCard s={s} />
      <FinalTiles s={s} />
    </>
  )
}

/**
 * What the guard did without stopping the send, and the recipients the
 * template failed for (review GS-11, GS-12, CO-4): one banner each, with
 * Acknowledge for the people who can send. Only the heading is an alert.
 */
export function NoticeBanners({ s, actions, can }: { s: Snapshot; actions: LiveActions; can?: LivePermissions }) {
  const t = useT()
  const notices = openNotices(s)
  if (!notices.length) return null
  return (
    <>
      {notices.map((i) => {
        const n = noticeText(i)
        return (
          <Banner
            key={i.id}
            tone="warn"
            title={<span role="alert">{n.title}</span>}
            actions={
              <Button onClick={() => void actions.acknowledge({ incidentId: i.id }).catch(() => undefined)} disabledReason={can?.resume} loading={actions.pending}>
                {t(m.acknowledge)}
              </Button>
            }
          >
            {n.body}
            {n.samples.length ? (
              <ul className="lv-samples">
                {n.samples.map((x) => (
                  <li key={x} className="mono">
                    {x}
                  </li>
                ))}
              </ul>
            ) : null}
          </Banner>
        )
      })}
    </>
  )
}

/**
 * The shares today's warm-up cap holds (review PR-8), each like the other
 * holds: the route, how many recipients wait and until when, the choices
 * (wait, move or cancel the rest, through the warm-up plan: waiting and
 * moving need campaigns:send, cancelling campaigns:pause) and who can
 * decide. Each goes on by itself with tomorrow's cap, so the send keeps
 * running instead of waiting for a decision.
 */
export function WarmupHoldBanners({ s, actions, can }: { s: Snapshot; actions: LiveActions; can?: LivePermissions }) {
  const holds = warmupHolds(s)
  if (!holds.length) return null
  return (
    <>
      {holds.map((p) => (
        <WarmupHoldBanner key={p.id} s={s} p={p} actions={actions} can={can} />
      ))}
    </>
  )
}

function WarmupHoldBanner({ s, p, actions, can }: { s: Snapshot; p: Provider; actions: LiveActions; can?: LivePermissions }) {
  const t = useT()
  const [target, setTarget] = useState('')
  const [cancelling, setCancelling] = useState(false)
  const h = p.warmupHold
  if (!h) return null
  const allowed = h.targets.filter((x) => x.allowed)
  const blocked = h.targets.filter((x) => !x.allowed)
  const chosen = allowed.find((x) => x.providerId === target) ?? allowed[0]
  const send = s.helpers?.send ?? []
  const pause = s.helpers?.pause ?? []
  const same = send.length > 0 && send.length === pause.length && send.every((n) => pause.includes(n))
  const who = same
    ? t(m.warmupWhoAll, { names: andList(send) })
    : [send.length ? t(m.warmupWhoMove, { names: andList(send) }) : '', pause.length ? t(m.warmupWhoCancel, { names: andList(pause) }) : ''].filter(Boolean).join(' · ')
  const decide = (action: 'wait' | 'move') => () =>
    void actions
      .decideWarmupHold(h.planId, action === 'move' && chosen ? { action, targetProviderId: chosen.providerId } : { action }, chosen?.short)
      .catch(() => undefined)
  return (
    <Banner
      tone="warn"
      title={t(m.warmupHoldTitle, { provider: p.name })}
      actions={
        h.planId ? (
          <>
            {h.decision !== 'wait' ? (
              <Button disabledReason={can?.decide} loading={actions.pending} onClick={decide('wait')}>
                {t(mw.holdWait)}
              </Button>
            ) : null}
            {chosen ? (
              <Button variant="primary" disabledReason={can?.decide} loading={actions.pending} onClick={decide('move')}>
                {t(mw.holdMove, { provider: chosen.short })}
              </Button>
            ) : null}
            <Button variant="danger" disabledReason={can?.warmupCancel} onClick={() => setCancelling(true)}>
              {t(mw.holdCancel)}
            </Button>
          </>
        ) : null
      }
    >
      <span>
        {t(mw.holdBody, { count: h.held, paused: h.paused ? 'yes' : 'no', time: h.until ? clock(h.until) : '' })}
        {h.decision === 'wait' ? ` ${t(m.warmupWaiting)}` : ''}
      </span>
      {allowed.length > 1 ? (
        <span className="mt-2 flex max-w-[360px] flex-col gap-1">
          <Field label={t(mw.holdMoveTarget)}>
            <Select className="lv-warm-target" value={chosen?.providerId ?? ''} onChange={(e) => setTarget(e.target.value)}>
              {allowed.map((x) => (
                <option key={x.providerId} value={x.providerId}>
                  {x.name}
                </option>
              ))}
            </Select>
          </Field>
        </span>
      ) : null}
      {!allowed.length && blocked.length ? (
        <span className="mt-1 block">
          {t(mw.holdNoTarget)} {blocked.map((x) => (x.reason ? `${x.short}: ${x.reason}` : x.short)).join(' · ')}
        </span>
      ) : null}
      {who ? <span className="k mt-1 block">{who}</span> : null}
      <ConfirmDialog
        open={cancelling}
        onOpenChange={setCancelling}
        badge={t(mw.holdCancel)}
        title={t(m.warmupCancelTitle, { count: count(h.held), provider: p.short })}
        description={t(mw.holdCancelBody, { count: h.held, provider: p.short })}
        confirmLabel={t(mw.holdCancel)}
        cancelLabel={t(m.warmupKeep)}
        onConfirm={() => actions.decideWarmupHold(h.planId, { action: 'cancel' })}
      />
    </Banner>
  )
}

/**
 * A complaint alert raised after the send (review GS-6): the mailbox
 * provider passed its complaint limit during the watch. Nothing is
 * sending; the people who can resume sends acknowledge it.
 */
function ComplaintAlerts({ s, actions, can }: { s: Snapshot; actions: LiveActions; can?: LivePermissions }) {
  const tr = useT()
  const open = s.incidents.filter((i) => i.kind === 'complaint_watch' && !i.resolvedAt)
  return (
    <>
      {open.map((i) => (
        <Banner
          key={i.id}
          tone="warn"
          title={
            <span role="alert">
              {i.detail.reason === 'postmaster'
                ? // Gmail's daily spam rate from Postmaster Tools (review GS-6, R2.A4).
                  tr(m.watchAlertPostmaster, { value: pct(i.value ?? undefined, 2), limit: pct(i.limit ?? undefined, 2) })
                : i.scope === 'mailbox_provider'
                  ? tr(m.watchAlertWho, { who: scopeName(i), value: pct(i.value ?? undefined, 2), limit: pct(i.limit ?? undefined, 2) })
                  : tr(m.watchAlertAll, { value: pct(i.value ?? undefined, 2), limit: pct(i.limit ?? undefined, 2) })}
            </span>
          }
          actions={
            <Button onClick={() => void actions.acknowledge({ incidentId: i.id }).catch(() => undefined)} disabledReason={can?.resume} loading={actions.pending}>
              {tr(m.acknowledge)}
            </Button>
          }
        >
          {i.detail.recommendations?.join(' ')}
        </Banner>
      ))}
    </>
  )
}

/** The complaint watch after the send (LiveMonitorStates: complete). */
export function ComplaintWatchCard({ s }: { s: Snapshot }) {
  const tr = useT()
  const w = s.watch
  if (!w) return null
  const rows = watchRows(s)
  const unreported = s.totals.notReported
  return (
    <section className="card flex flex-col gap-3 px-5 py-4" aria-labelledby="lv-cw-h">
      <div className="flex flex-wrap items-center gap-x-3.5 gap-y-2.5">
        <Badge tone={w.active ? 'ok' : 'neutral'} dot>
          {tr(w.active ? m.watchWatching : m.watchEnded)}
        </Badge>
        <h2 className="h2" id="lv-cw-h">
          {watchTitle(s)}
        </h2>
        <span className="k ml-auto">{tr(m.watchUntil, { until: `${weekdayDate(w.until)}, ${clock(w.until)}` })}</span>
      </div>
      <div className="lv-cw">
        {rows.map((r) => (
          <div key={r.key} className="lv-thr">
            <div className="lv-thr-h">
              <span>
                <b>{r.name}</b>
                <span className="k">{r.counts}</span>
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
            </div>
          </div>
        ))}
        {unreported.length ? (
          <div className="lv-thr">
            <div className="lv-thr-h">
              <span>
                <b>{andList(unreported)}</b>
                <span className="k">{tr(m.watchNoPerMessage)}</span>
              </span>
              <Badge tone="neutral" dot>
                {tr(m.watchNotReported)}
              </Badge>
            </div>
            {s.postmaster ? (
              <div className="lv-thr-v">
                <span>{tr(m.watchPostmaster, { rate: pct(s.postmaster.spamRate, 2) })}</span>
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
      <p className="lv-fn">{tr(m.watchFootnote)}</p>
    </section>
  )
}

/** The complete state's counters, cost and the per-provider results. */
export function FinalTiles({ s }: { s: Snapshot }) {
  const tr = useT()
  const lines = providerResultLines(s)
  return (
    <Grid min={200} gap={16}>
      {finalTiles(s).map((t) => (
        <div key={t.key} className="card tile">
          <span className="k">{t.label}</span>
          <span className="n text-[26px]">
            <TileValue tile={t} />
          </span>
          <span className="k">{t.sub}</span>
        </div>
      ))}
      {lines.length ? (
        <div className="card tile">
          <span className="k">{tr(m.perProvider)}</span>
          {lines.map((l) => (
            <span key={l} className="n text-[16px]">
              {l}
            </span>
          ))}
        </div>
      ) : null}
    </Grid>
  )
}

/** A cancelled send: what went out before it stopped. */
export function CancelledView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const t = cancelledText(s)
  const params = { workspace: slug, campaignId }
  return (
    <>
      <Banner
        tone="info"
        title={t.title}
        actions={
          s.sent > 0 ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
              {tr(m.viewReport)}
            </ButtonLink>
          ) : (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
              {tr(m.backToCampaigns)}
            </ButtonLink>
          )
        }
      >
        {t.body}
      </Banner>
      {s.sent > 0 ? <FinalTiles s={s} /> : null}
    </>
  )
}

/** A send that stopped on an error. */
export function FailedView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      <Banner
        tone="bad"
        title={t(m.failedTitle, { name: s.name })}
        actions={
          <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId }}>
            {t(m.viewReport)}
          </ButtonLink>
        }
      >
        {s.error ? `${s.error} ` : ''}
        {t(m.failedBody, { sent: count(s.sent), total: count(s.recipients) })}
      </Banner>
      {s.sent > 0 ? <FinalTiles s={s} /> : null}
    </>
  )
}

/** A scheduled campaign: the monitor comes alive when it starts. */
export function ScheduledView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Banner
      tone="info"
      title={scheduledLine(s.scheduledAt)}
      actions={
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          {t(m.reviewSend)}
        </ButtonLink>
      }
    >
      {t(m.scheduledBody, { name: s.name, count: count(s.recipients) })}
    </Banner>
  )
}

/** A draft or a campaign waiting for approval: nothing is sending. */
export function IdleView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <EmptyState
      title={t(m.idleTitle)}
      action={
        <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          {t(m.reviewSend)}
        </ButtonLink>
      }
    >
      {t(m.idleBody, { name: s.name })}
    </EmptyState>
  )
}

/** A send paused by a person: who, when, why, and the way back. */
export function PausedBanner({ s, can, onResume, onCancel, busy }: { s: Snapshot; can: LivePermissions; onResume: () => void; onCancel: () => void; busy?: boolean }) {
  const tr = useT()
  const t = pausedText(s)
  return (
    <Banner
      tone="warn"
      title={t.title}
      actions={
        <>
          <Button variant="primary" onClick={onResume} disabledReason={can.resume} loading={busy}>
            {tr(m.resumeSending)}
          </Button>
          <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
            {tr(m.cancelSend)}
          </Button>
        </>
      }
    >
      {t.body}
    </Banner>
  )
}

/** An open incident seen from the live view: what happened and the way to decide. */
export function IncidentBanner({ s, incident, onDecide }: { s: Snapshot; incident: Incident; onDecide: () => void }) {
  const t = useT()
  const outage = incident.kind === 'provider_outage'
  const who = scopeName(incident)
  const others = continuingProviders(s, incident).map((p) => p.short)
  const rest = outage ? (others.length ? t(m.othersKeepSending, { providers: andList(others) }) : t(m.nothingElseSending)) : t(m.everythingElseSending)
  const held = who ? t(m.heldUntilWho, { count: count(incident.held), who }) : t(m.heldUntil, { count: count(incident.held) })
  // While nobody has decided, the held share tests its provider by itself (review GS-3).
  const probing = outage && s.providers.some((p) => p.id === incident.providerId && p.probing && p.routeStatus === 'held')
  return (
    <Banner
      tone={outage ? 'warn' : 'bad'}
      title={incidentTitle(incident)}
      actions={
        <Button variant="primary" onClick={onDecide}>
          {t(m.chooseTitle)}
        </Button>
      }
    >
      {[held, rest, probing ? t(m.bannerProbing, { provider: incident.providerShort || incident.provider }) : ''].filter(Boolean).join(' ')}
    </Banner>
  )
}

/** First load: the live view's shape in grey. */
export function LiveSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="flex flex-col gap-5 max-md:gap-3">
      <Card className="flex flex-col gap-3 px-[22px] py-5">
        <div className="flex items-baseline gap-3.5">
          <Skeleton w={80} h={34} />
          <Skeleton w={180} h={16} />
        </div>
        <Skeleton h={12} r={6} className="w-full" />
        <div className="flex gap-[18px]">
          <Skeleton w={160} h={14} />
          <Skeleton w={130} h={14} />
          <Skeleton w={100} h={14} />
        </div>
      </Card>
      <Grid min={200} gap={16}>
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="card tile">
            <Skeleton w={70} h={12} />
            <Skeleton w={90} h={26} />
            <Skeleton w={110} h={12} />
          </div>
        ))}
      </Grid>
      <Card className={cn('flex flex-col gap-3 px-[18px] py-3.5')}>
        <Skeleton w={160} h={16} />
        {[0, 1, 2, 3, 4].map((i) => (
          <Skeleton key={i} h={14} className="w-full" />
        ))}
      </Card>
    </Loading>
  )
}

/** The decision views' badge for a whole campaign paused by a threshold. */
export function HeldBadge({ incident }: { incident: Incident }) {
  return (
    <Badge tone="bad" pulse>
      {autoPausedText(incident)}
    </Badge>
  )
}
