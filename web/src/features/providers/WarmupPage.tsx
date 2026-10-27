import { useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Banner } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useI18n, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { clock, count, decimal, pct, shortDate, trimPct, weekdayName } from '@/lib/format'
import { m as fm } from '@/lib/format.messages'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { limitPct, perSec, ProvidersTabs, throwIfForbidden } from './common'
import { m } from './messages'
import { mailboxName } from './routing-model'
import {
  audienceText,
  guardIn,
  mailboxColor,
  nextDay,
  rampColumns,
  tableRows,
  tiersText,
  type DayGroup,
  type WarmupCapHold,
  type WarmupPlan,
  type WarmupRow,
} from './warmup-model'
import { mw } from './warmup.messages'
import { WarmupPlacementCard } from '@/features/placement/EntryPoints'

const warmupKey = ['get', '/api/v1/workspaces/{workspace}/warmup'] as const
const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** Plan lengths, in counted days, and how each one is described. */
const lengths = [
  { days: 21, label: mw.length21 },
  { days: 14, label: mw.length14 },
  { days: 30, label: mw.length30 },
] as const

/** 7.3 Warm-up: the ramp of a new IP or provider, how it moves, its guards and its log. */
export function WarmupPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/warmup', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  const [picked, setPicked] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const plans = q.data?.plans ?? []
  const plan = plans.find((p) => p.id === picked) ?? plans.find((p) => p.status !== 'done') ?? plans[0]
  return (
    <Page
      title={t(m.title)}
      filters={<ProvidersTabs />}
      actions={q.data ? <WarmupActions plan={plan} canCreate={q.data.candidates.length > 0} onNew={() => setCreating(true)} /> : null}
    >
      {q.isError ? (
        <ErrorPanel title={t(mw.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(mw.loadFailedBody)}
        </ErrorPanel>
      ) : q.isPending ? (
        <WarmupSkeleton />
      ) : !plan ? (
        <EmptyState
          title={t(mw.emptyTitle)}
          action={
            q.data.candidates.length ? (
              <Button variant="primary" onClick={() => setCreating(true)}>
                {t(mw.newPlan)}
              </Button>
            ) : null
          }
        >
          {t(mw.emptyBody)}
        </EmptyState>
      ) : (
        <>
          {plans.length > 1 ? (
            <Segmented label={t(mw.planPicker)} value={plan.id} onChange={setPicked} items={plans.map((p) => ({ value: p.id, label: `${p.short}${p.ip ? ` · ${p.ip}` : ''}` }))} />
          ) : null}
          {plan.holds.length ? <CapHolds plan={plan} /> : null}
          <PlanSummary plan={plan} />
          <div className="flex flex-wrap items-start gap-5">
            <RampCard plan={plan} />
            <aside className="flex max-w-[440px] flex-[1_1_340px] flex-col gap-4 max-md:max-w-none max-md:min-w-0" aria-label={t(mw.asideLabel)}>
              {plan.status !== 'done' ? <MailboxCaps plan={plan} /> : null}
              {plan.status !== 'done' ? <PlanSettings key={plan.id} plan={plan} /> : null}
              <WarmupLog plan={plan} />
              <WarmupPlacementCard key={`pl-${plan.id}`} providerId={plan.providerId} provider={plan.short} day={plan.day} days={plan.days} />
            </aside>
          </div>
        </>
      )}
      {creating && q.data ? <NewPlanDialog candidates={q.data.candidates} onClose={() => setCreating(false)} /> : null}
    </Page>
  )
}

function useRefreshWarmup() {
  const qc = useQueryClient()
  return () => Promise.all([qc.invalidateQueries({ queryKey: warmupKey }), qc.invalidateQueries({ queryKey: providersKey })])
}

function WarmupActions({ plan, canCreate, onNew }: { plan?: WarmupPlan; canCreate: boolean; onNew: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const reason = usePermissionReason('providers:manage')
  const onDone = (paused: boolean) => () => {
    void refresh()
    toast({ kind: 'success', title: paused ? t(mw.paused) : t(mw.resumed), detail: paused ? t(mw.pausedDetail) : t(mw.resumedDetail) })
  }
  const onError = (e: unknown) => toast({ kind: 'error', title: t(mw.changeFailed), detail: errorMessage(e) })
  const pause = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/pause', { onSuccess: onDone(true), onError })
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/resume', { onSuccess: onDone(false), onError })
  const path = plan ? { workspace: slug, planId: plan.id } : undefined
  return (
    <>
      {plan && plan.status !== 'done' ? (
        plan.status === 'paused' ? (
          <Button disabledReason={reason} loading={resume.isPending} onClick={() => path && resume.mutate({ params: { path } })}>
            {t(mw.resume)}
          </Button>
        ) : (
          <Button disabledReason={reason} loading={pause.isPending} onClick={() => path && pause.mutate({ params: { path } })}>
            {t(mw.pause)}
          </Button>
        )
      ) : null}
      <Button variant="primary" disabledReason={reason ?? (canCreate ? undefined : t(mw.allHavePlans))} onClick={onNew}>
        {t(mw.newPlan)}
      </Button>
    </>
  )
}

/** Campaigns waiting at today's cap (or a paused plan), each with its choices. */
function CapHolds({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-3" role="group" aria-label={t(mw.holdsLabel)}>
      {plan.holds.map((h) => (
        <CapHold key={h.campaignId} plan={plan} hold={h} />
      ))}
    </div>
  )
}

function CapHold({ plan, hold }: { plan: WarmupPlan; hold: WarmupCapHold }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const sendReason = usePermissionReason('campaigns:send')
  const pauseReason = usePermissionReason('campaigns:pause')
  const allowed = hold.alternatives.filter((a) => a.allowed)
  const [target, setTarget] = useState(allowed[0]?.providerId ?? '')
  const [cancelling, setCancelling] = useState(false)
  const decide = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/held/{campaignId}')
  const path = { workspace: slug, planId: plan.id, campaignId: hold.campaignId }
  const targetName = allowed.find((a) => a.providerId === target)?.short ?? ''
  const run = async (body: Schemas['WarmupHoldDecision']) => {
    const res = await decide.mutateAsync({ params: { path }, body })
    void refresh()
    const n = count(res.moved || res.cancelled)
    toast({
      kind: 'success',
      title:
        body.action === 'move'
          ? t(mw.holdMovedToast, { count: n, provider: targetName })
          : body.action === 'cancel'
            ? t(mw.holdCancelledToast, { count: n })
            : t(mw.holdWaitToast),
    })
  }
  const act = (body: Schemas['WarmupHoldDecision']) => () => {
    run(body).catch((e: unknown) => toast({ kind: 'error', title: t(mw.holdFailed), detail: errorMessage(e) }))
  }
  const blocked = hold.alternatives.filter((a) => !a.allowed)
  return (
    <Banner
      tone="warn"
      title={t(mw.holdTitle, { campaign: hold.campaign, provider: plan.short })}
      actions={
        <>
          {hold.decision !== 'wait' ? (
            <Button disabledReason={sendReason} loading={decide.isPending && decide.variables?.body.action === 'wait'} onClick={act({ action: 'wait' })}>
              {t(mw.holdWait)}
            </Button>
          ) : null}
          {allowed.length ? (
            <Button
              variant="primary"
              disabledReason={sendReason}
              loading={decide.isPending && decide.variables?.body.action === 'move'}
              onClick={act({ action: 'move', targetProviderId: target })}
            >
              {t(mw.holdMove, { provider: targetName })}
            </Button>
          ) : null}
          <Button variant="danger" disabledReason={pauseReason} onClick={() => setCancelling(true)}>
            {t(mw.holdCancel)}
          </Button>
        </>
      }
    >
      <span>
        {t(mw.holdBody, { count: hold.held, paused: hold.paused ? 'yes' : 'no', time: hold.resumeAt ? clock(hold.resumeAt) : '' })}
        {hold.decision === 'wait' ? ` ${t(mw.holdWaiting)}` : ''}
      </span>
      {allowed.length > 1 ? (
        <span className="mt-2 flex max-w-[360px] flex-col gap-1">
          <Field label={t(mw.holdMoveTarget)}>
            <Select value={target} onChange={(e) => setTarget(e.target.value)}>
              {allowed.map((a) => (
                <option key={a.providerId} value={a.providerId}>
                  {a.name}
                </option>
              ))}
            </Select>
          </Field>
        </span>
      ) : null}
      {!allowed.length && blocked.length ? (
        <span className="mt-1 block">
          {t(mw.holdNoTarget)}{' '}
          {blocked.map((a) => `${a.short}: ${a.checks.find((c) => c.result === 'fail')?.text ?? ''}`).join(' · ')}
        </span>
      ) : null}
      <ConfirmDialog
        open={cancelling}
        onOpenChange={setCancelling}
        badge={t(mw.holdCancel)}
        title={t(mw.holdCancelTitle, { campaign: hold.campaign })}
        description={t(mw.holdCancelBody, { count: hold.held, provider: plan.short })}
        confirmLabel={t(mw.holdCancel)}
        onConfirm={() => run({ action: 'cancel' })}
      />
    </Banner>
  )
}

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div>
      <div className="k">{label}</div>
      <div className="n text-[20px]">{value}</div>
      <div className="k">{sub}</div>
    </div>
  )
}

/** The plan's headline: progress, how it moves, today's cap, totals, guards, slowdowns and rate. */
function PlanSummary({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const today = plan.rows.find((d) => d.status === 'today')
  const rate = (v: number) => t(fm.perSecond, { n: perSec(v) })
  const s = plan.settings
  const statusBadge =
    plan.status === 'paused' ? (
      <Badge tone="warn" dot>
        {t(mw.badgePaused, { day: plan.day, days: plan.days })}
      </Badge>
    ) : plan.status === 'done' ? (
      <Badge tone="ok" dot>
        {t(mw.badgeDone, { days: plan.days })}
      </Badge>
    ) : plan.status === 'slowed' ? (
      <Badge tone="warn" dot>
        {t(mw.badgeSlowed, { day: plan.day, days: plan.days })}
      </Badge>
    ) : (
      <Badge tone="info" dot>
        {t(m.statusWarmingUpDay, { day: plan.day, days: plan.days })}
      </Badge>
    )
  const todayLine =
    plan.status === 'done'
      ? ''
      : plan.status === 'paused'
        ? t(mw.howPausedNow, { day: plan.day })
        : plan.repeatsTomorrow && today
          ? t(mw.howRepeats, { weekday: weekdayName(today.date), sent: count(plan.sentToday), day: plan.day })
          : t(mw.howNeeds, { need: count(plan.needToday), sent: count(plan.sentToday) })
  const slowdown = plan.lastSlowdown
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="pv-wu-title">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="h2 text-[17px]" id="pv-wu-title">
          {plan.ip ? t(mw.titleIp, { provider: plan.short, ip: plan.ip }) : plan.short}
        </h2>
        {statusBadge}
        <Badge>{t(mw.started, { date: shortDate(plan.startedOn) })}</Badge>
        <span className="k ml-auto max-md:ml-0">
          {plan.status === 'done'
            ? t(mw.finished, { date: shortDate(plan.finishesOn), provider: plan.short })
            : t(mw.finishes, { date: shortDate(plan.finishesOn), provider: plan.short })}
        </span>
      </div>
      {plan.statusDetail ? <span className="text-[13px] text-warn-fg">{plan.statusDetail}</span> : null}
      <div className="flex flex-col gap-1.5">
        <div className="flex h-2.5 gap-0.5 overflow-hidden rounded-[5px] bg-line-2" aria-hidden="true">
          <div style={{ width: `${plan.progress}%`, background: '#1c5cab' }} />
        </div>
        <span className="k">{t(mw.counted, { counted: plan.counted, days: plan.days })}</span>
      </div>
      {plan.status !== 'done' ? (
        <div className="flex flex-col gap-1.5 rounded-lg border border-line bg-ground-3 px-3.5 py-3">
          <h3 className="m-0 text-[13px] font-semibold">{t(mw.howTitle)}</h3>
          <ul className="m-0 flex flex-col gap-1 pl-[18px] text-[13px]">
            <li>
              {t(mw.howCounts, { share: trimPct(plan.countShare * 100) })} {todayLine ? <strong>{todayLine}</strong> : null}
            </li>
            <li>{t(mw.howSample, { count: count(plan.minSample) })}</li>
            <li>{t(mw.howHold, { action: s.guardAction, days: plan.healthyDays })}</li>
            <li>{t(mw.howPause)}</li>
          </ul>
        </div>
      ) : null}
      <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))' }}>
        <Stat
          label={t(mw.todayCap)}
          value={count(plan.todayCap)}
          sub={
            today
              ? plan.scheduledToday === 0
                ? t(mw.sentOnQuiet, { sent: count(plan.sentToday), day: weekdayName(today.date) })
                : t(mw.sentOn, { sent: count(plan.sentToday), day: weekdayName(today.date) })
              : t(mw.sentOn, { sent: count(plan.sentToday), day: weekdayName(new Date()) })
          }
        />
        <Stat label={t(mw.sentDuring)} value={count(plan.sentTotal)} sub={t(mw.daysCounted, { counted: plan.counted, days: plan.calendarDays })} />
        <Stat
          label={t(mw.bounceGuard, { limit: limitPct(s.guardBounce) })}
          value={plan.bounceRate !== undefined ? pct(plan.bounceRate, 1) : t(mw.notAvailable)}
          sub={plan.bounceRate !== undefined ? t(mw.sevenDay) : t(mw.bounceUnknown)}
        />
        <Stat
          label={t(mw.complaintGuard, { limit: limitPct(s.guardComplaint) })}
          value={plan.complaintRate !== undefined ? pct(plan.complaintRate, 2) : t(mw.notAvailable)}
          sub={plan.complaintRate !== undefined ? t(mw.sevenDay) : t(mw.complaintUnknown)}
        />
        <Stat
          label={t(mw.slowdowns)}
          value={count(plan.slowdowns)}
          sub={
            slowdown
              ? t(mw.lastSlowdown, { metric: slowdown.metric, day: slowdown.day, rate: slowdown.rate !== undefined ? trimPct(slowdown.rate) : t(mw.notAvailable) })
              : t(mw.noneSoFar)
          }
        />
        <Stat
          label={t(mw.rateToday)}
          value={rate(plan.rateToday)}
          sub={plan.nextRate ? t(mw.risesTo, { rate: rate(plan.nextRate.value), day: plan.nextRate.day }) : t(mw.staysThere)}
        />
      </div>
    </section>
  )
}

/** The daily ramp chart and the table of the plan's days. */
function RampCard({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const cols = rampColumns(plan)
  const first = plan.ramp[0]
  const last = plan.ramp[plan.ramp.length - 1]
  const today = plan.ramp.find((d) => d.today)
  const sentDays = plan.ramp.filter((d) => d.sent !== undefined && !d.today).length
  const tripped = cols.filter((c) => c.tripped).map((c) => c.day)
  const label = [
    t(mw.rampLabel, { first: count(first?.plannedCap ?? 0), last: count(last?.plannedCap ?? 0), days: plan.days }),
    sentDays ? t(mw.rampLabelSent, { days: sentDays }) : '',
    ...tripped.map((d) => t(mw.rampLabelTripped, { day: d })),
    today && plan.status !== 'done' ? t(mw.rampLabelToday, { day: today.day, cap: count(plan.todayCap) }) : '',
  ]
    .filter(Boolean)
    .join(' ')
  return (
    <section className="card flex min-w-0 flex-[2_1_600px] flex-col" aria-labelledby="pv-ramp">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-ramp">
          {t(mw.rampTitle)}
        </h2>
        <div className="flex flex-wrap gap-3.5 text-[12px]">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-info-bg shadow-[inset_0_0_0_1px_var(--line-input)]" aria-hidden="true" />
            {t(mw.legendPlanned)}
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-[#1c5cab]" aria-hidden="true" />
            {t(mw.legendSent)}
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-warn-bar" aria-hidden="true" />
            {t(mw.legendTripped)}
          </span>
        </div>
      </div>
      <div className="flex flex-col gap-2 px-5 py-[18px]">
        <div className="flex h-[180px] items-end gap-1" role="img" aria-label={label}>
          {cols.map((c) => (
            <div key={c.day} className="pv-col" style={c.today ? { borderBottom: '3px solid var(--ink)' } : undefined}>
              {c.cap ? <div className="cap" style={{ height: `${c.cap}%` }} /> : null}
              {c.sent ? <div className="sent" style={{ height: `${c.sent}%`, background: c.tripped ? 'var(--warn-bar)' : undefined }} /> : null}
            </div>
          ))}
        </div>
        <div className="k flex justify-between gap-2">
          <span>{t(mw.dayFirst, { cap: count(first?.plannedCap ?? 0) })}</span>
          {today && plan.status !== 'done' ? <span>{t(mw.dayToday, { day: today.day, cap: count(plan.todayCap) })}</span> : null}
          <span>{t(mw.dayLast, { day: last?.day ?? plan.days, cap: count(last?.plannedCap ?? 0) })}</span>
        </div>
      </div>
      <div className="border-t border-line-2">
        <Table className="pv-9 min-w-[720px]" caption={t(mw.tableCaption)}>
          <thead>
            <tr>
              <Th>{t(mw.colDay)}</Th>
              <Th>{t(mw.colDate)}</Th>
              <Th align="right">{t(mw.colCap)}</Th>
              <Th align="right">{t(mw.colSent)}</Th>
              <Th align="right">{t(mw.colBounce)}</Th>
              <Th align="right">{t(mw.colComplaints)}</Th>
              <Th>{t(mw.colAudience)}</Th>
              <Th>{t(mw.colCounted)}</Th>
            </tr>
          </thead>
          <tbody>
            {tableRows(plan.rows).map((r) =>
              r.kind === 'group' ? <GroupRow key={r.key} group={r.group} /> : <DayRow key={r.key} plan={plan} row={r.row} />,
            )}
          </tbody>
        </Table>
      </div>
    </section>
  )
}

function CountedBadge({ counted }: { counted: boolean }) {
  const t = useT()
  return counted ? (
    <Badge tone="ok" icon="check">
      {t(mw.badgeCounted)}
    </Badge>
  ) : (
    <Badge tone="warn" icon="close">
      {t(mw.badgeNotCounted)}
    </Badge>
  )
}

/** The first days, folded into one row. */
function GroupRow({ group: g }: { group: DayGroup }) {
  const t = useT()
  const notes = [
    t(mw.noteDays, { days: g.lastDay - g.firstDay + 1 }),
    ...g.missed.map((r) => t(mw.noteMissed, { date: shortDate(r.date), sent: count(r.sent ?? 0), day: r.day, weekday: weekdayName(nextDay(r.date)) })),
  ]
  return (
    <tr>
      <Td className="nw">{t(mw.dayRange, { first: g.firstDay, last: g.lastDay })}</Td>
      <Td className="nw">
        {shortDate(g.from)}
        <div className="k">{t(mw.dateTo, { date: shortDate(g.to) })}</div>
      </Td>
      <Td align="right" num>
        {count(g.cap)}
      </Td>
      <Td align="right" num>
        {count(g.sent)}
      </Td>
      <Td align="right" num>
        {g.bounce !== undefined ? pct(g.bounce, 1) : '–'}
      </Td>
      <Td align="right" num>
        {g.complaint !== undefined ? pct(g.complaint, 2) : t(mw.notAvailable)}
      </Td>
      <Td>{tiersText(g.tiers)}</Td>
      <Td className="wrap">
        <CountedBadge counted={g.counted} />
        <div className="k">{notes.join(' · ')}</div>
      </Td>
    </tr>
  )
}

/** One date of the plan: judged, today or ahead. */
function DayRow({ plan, row: d }: { plan: WarmupPlan; row: WarmupRow }) {
  const t = useT()
  const s = plan.settings
  const isToday = d.status === 'today'
  const ahead = d.status === 'planned'
  const guard = d.reason === 'guard'
  const last = ahead && d.day === plan.days && !d.repeat
  let note = ''
  if (d.status === 'counted' && d.healthy) {
    note = d.healthy >= plan.healthyDays ? t(mw.noteHealthyLast, { n: d.healthy, of: plan.healthyDays }) : t(mw.noteHealthy, { n: d.healthy, of: plan.healthyDays })
  } else if (guard) {
    const limit = d.metric === 'complaint' ? limitPct(s.guardComplaint) : limitPct(s.guardBounce)
    note = t(mw.noteGuard, { metric: d.metric ?? 'bounce', limit, action: s.guardAction, cap: count(plan.held ?? d.cap) })
  } else if (d.status === 'not_counted') {
    note = t(mw.noteShort, { sent: count(d.sent ?? 0), cap: count(d.cap), day: d.day })
  } else if (isToday && plan.status === 'paused') {
    note = t(mw.notePaused)
  } else if (isToday && plan.repeatsTomorrow) {
    note = t(mw.noteTodayRepeats, { sent: count(d.sent ?? 0), cap: count(d.cap), day: d.day })
  } else if (isToday) {
    note = t(mw.noteTodayNeeds, { sent: count(d.sent ?? 0), need: count(d.need) })
  } else if (d.repeat) {
    note = t(mw.noteRepeat, { day: d.day })
  }
  const rate = (v: number | undefined, digits: number) => (ahead ? '–' : isToday && !d.judged ? t(mw.notJudged) : v !== undefined ? pct(v, digits) : '–')
  return (
    <tr className={cn(guard && 'bg-[#FFFBEB]', isToday && 'bg-accent-tint')}>
      <Td className={cn(isToday && 'font-bold')}>{d.day}</Td>
      <Td className={cn('nw', isToday && 'font-bold')}>
        {shortDate(d.date)}
        {isToday ? <div className="k font-medium">{t(mw.todayWord)}</div> : last ? <div className="k">{t(mw.atTheEarliest)}</div> : null}
      </Td>
      <Td align="right" num>
        {count(d.cap)}
        {d.held && !ahead ? <div className="k">{d.plannedCap ? t(mw.capHeldPlan, { cap: count(d.plannedCap) }) : t(mw.capHeld)}</div> : null}
      </Td>
      <Td align="right" num>
        {ahead || d.sent === undefined ? '–' : count(d.sent)}
      </Td>
      <Td align="right" num className={cn(guard && d.metric !== 'complaint' && 'font-semibold text-warn-fg')}>
        {rate(d.bounce, 1)}
      </Td>
      <Td align="right" num className={cn(guard && d.metric === 'complaint' && 'font-semibold text-warn-fg')}>
        {rate(d.complaint, 2)}
      </Td>
      <Td className="wrap">{audienceText(d)}</Td>
      <Td className="wrap">
        {ahead ? (
          <Badge dot>{t(mw.badgePlanned)}</Badge>
        ) : isToday ? (
          plan.status === 'paused' ? (
            <Badge tone="warn">{t(mw.badgePausedDay)}</Badge>
          ) : plan.repeatsTomorrow ? (
            <CountedBadge counted={false} />
          ) : (
            <Badge tone="info" dot>
              {t(mw.badgeToday)}
            </Badge>
          )
        ) : (
          <CountedBadge counted={d.status === 'counted'} />
        )}
        {note ? <div className="k">{note}</div> : null}
      </Td>
    </tr>
  )
}

/** Today's cap per mailbox provider, and what's over the cap. */
function MailboxCaps({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const reason = usePermissionReason('providers:manage')
  const spread = plan.settings.spreadMailbox
  const overflow = plan.settings.overflow ?? true
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/warmup/{planId}', {
    onSuccess: (p) => {
      void refresh()
      toast({ kind: 'success', title: p.settings.overflow ? t(mw.overflowOn) : t(mw.overflowOff) })
    },
    onError: (e) => toast({ kind: 'error', title: t(mw.changeFailed), detail: errorMessage(e) }),
  })
  const sentTotal = plan.mailboxCaps.reduce((n, c) => n + c.sent, 0)
  const toggleId = useId()
  return (
    <section className="card flex flex-col gap-3 px-5 py-[18px]" aria-labelledby="pv-mb-caps">
      <div className="flex flex-wrap items-baseline justify-between gap-x-2.5 gap-y-1">
        <h2 className="h2" id="pv-mb-caps">
          {t(mw.capsTitle)}
        </h2>
        <span className="k">{t(mw.capsTotal, { day: plan.day, cap: count(plan.todayCap) })}</span>
      </div>
      {spread ? (
        <div className="flex h-3 gap-0.5 overflow-hidden rounded-full bg-line-2" aria-hidden="true">
          {plan.mailboxCaps.map((c) => (
            <i key={c.mailbox} className="block h-full" style={{ width: `${c.percent}%`, background: mailboxColor[c.mailbox] }} />
          ))}
        </div>
      ) : (
        <span className="k">{t(mw.capsShared, { cap: count(plan.todayCap) })}</span>
      )}
      <Table className="pv-mtbl" caption={t(mw.capsCaption, { provider: plan.short })} scroll={false}>
        <thead>
          <tr>
            <Th>{t(mw.colMailbox)}</Th>
            <Th align="right">{t(mw.colCap)}</Th>
            <Th align="right">{t(mw.colSent)}</Th>
          </tr>
        </thead>
        <tbody>
          {plan.mailboxCaps.map((c) => (
            <tr key={c.mailbox}>
              <Td>
                <span className="dot mr-2" style={{ color: mailboxColor[c.mailbox] }} aria-hidden="true" />
                {mailboxName[c.mailbox] ?? c.mailbox}
              </Td>
              <Td align="right" num>
                {spread ? count(c.cap) : t(mw.sharedCap)}
              </Td>
              <Td align="right" num>
                {count(c.sent)}
              </Td>
            </tr>
          ))}
          <tr className="tot">
            <Td>{t(mw.total)}</Td>
            <Td align="right" num>
              {count(plan.todayCap)}
            </Td>
            <Td align="right" num>
              {count(sentTotal)}
            </Td>
          </tr>
        </tbody>
      </Table>
      <span className="k">{t(mw.capsMx)}</span>
      <div className="flex items-start justify-between gap-3 rounded-lg border border-line px-3 py-2.5">
        <label htmlFor={toggleId} className="flex min-w-0 cursor-pointer flex-col gap-0.5">
          <span className="text-[13px] font-semibold">
            {plan.overflowTo ? t(mw.overflowLabel, { provider: plan.overflowTo }) : t(mw.overflowLabelNone)}
          </span>
          <span className="k">{plan.overflowTo ? t(mw.overflowToday, { count: count(plan.overflowToday) }) : t(mw.overflowNone)}</span>
          {reason ? <span className="k">{reason}</span> : null}
        </label>
        <input
          id={toggleId}
          type="checkbox"
          role="switch"
          className="tg"
          checked={overflow}
          disabled={Boolean(reason) || save.isPending}
          onChange={(e) => save.mutate({ params: { path: { workspace: slug, planId: plan.id } }, body: { ...plan.settings, overflow: e.target.checked } })}
        />
      </div>
    </section>
  )
}

/** Plan settings: length, ordering, guards, what a trip does and the caps per mailbox provider. */
function PlanSettings({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const reason = usePermissionReason('providers:manage')
  const s = plan.settings
  const [days, setDays] = useState<14 | 21 | 30>(s.days)
  const [engaged, setEngaged] = useState(s.engagedFirst)
  const [bounce, setBounce] = useState(`${decimal(s.guardBounce, 1)} %`)
  const [complaint, setComplaint] = useState(`${decimal(s.guardComplaint, 2)} %`)
  const [action, setAction] = useState(s.guardAction)
  const [spread, setSpread] = useState(s.spreadMailbox)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/warmup/{planId}', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(mw.saved), detail: t(mw.savedDetail) })
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const submit = () => {
    const b = guardIn(bounce)
    const c = guardIn(complaint)
    const next: Record<string, string> = {}
    if (b === undefined || b < 0.5 || b > 10) next.bounce = t(mw.bounceRange, { min: trimPct(0.5), max: trimPct(10) })
    if (c === undefined || c < 0.01 || c > 1) next.complaint = t(mw.complaintRange, { min: trimPct(0.01), max: trimPct(1) })
    setErrors(next)
    if (Object.keys(next).length) return
    save.mutate({
      params: { path: { workspace: slug, planId: plan.id } },
      body: { days, engagedFirst: engaged, guardBounce: b!, guardComplaint: c!, guardAction: action, spreadMailbox: spread, overflow: s.overflow },
    })
  }
  const ro = Boolean(reason)
  const lastCap = plan.ramp[plan.ramp.length - 1]?.plannedCap ?? 0
  const sampleId = useId()
  return (
    <section className="card flex flex-col gap-3 px-5 py-[18px]" aria-labelledby="pv-plan-settings">
      <h2 className="h2" id="pv-plan-settings">
        {t(mw.settingsTitle)}
      </h2>
      {errors.form ? (
        <span className="err" role="alert">
          {errors.form}
        </span>
      ) : null}
      <LongChoice
        label={t(mw.length)}
        value={String(days)}
        disabled={ro}
        onChange={(v) => setDays(Number(v) as 14 | 21 | 30)}
        hint={days !== s.days ? t(mw.lengthHint, { cap: count(plan.todayCap), last: count(lastCap), days }) : undefined}
        options={lengths.map((l) => ({
          value: String(l.days),
          label: l.days < plan.day ? t(mw.lengthPast, { days: l.days }) : t(l.label),
          disabled: l.days < plan.day,
        }))}
      />
      <LongChoice
        label={t(mw.ordering)}
        value={engaged ? 'engaged' : 'random'}
        disabled={ro}
        onChange={(v) => setEngaged(v === 'engaged')}
        options={[
          { value: 'engaged', label: t(mw.orderEngaged) },
          { value: 'random', label: t(mw.orderRandom) },
        ]}
      />
      <Grid cols={2} gap={10} className="items-end">
        <Field label={t(mw.slowIfBounces)} error={errors.bounce}>
          <Input value={bounce} disabled={ro} inputMode="decimal" aria-describedby={sampleId} onChange={(e) => setBounce(e.target.value)} />
        </Field>
        <Field label={t(mw.orComplaints)} error={errors.complaint}>
          <Input value={complaint} disabled={ro} inputMode="decimal" aria-describedby={sampleId} onChange={(e) => setComplaint(e.target.value)} />
        </Field>
      </Grid>
      <span className="k -mt-1.5" id={sampleId}>
        {t(mw.howSample, { count: count(plan.minSample) })}
      </span>
      <LongChoice
        label={t(mw.whenTrips)}
        value={action}
        disabled={ro}
        onChange={setAction}
        options={[
          { value: 'hold', label: t(mw.actionHold, { days: plan.healthyDays }) },
          { value: 'step_back', label: t(mw.actionStepBack) },
          { value: 'pause', label: t(mw.actionPause) },
        ]}
      />
      <label className="flex cursor-pointer items-start gap-2.5 text-[13px]">
        <input type="checkbox" className="pv-check" checked={spread} disabled={ro} onChange={(e) => setSpread(e.target.checked)} />
        <span>{t(mw.spread)}</span>
      </label>
      <Button variant="primary" disabledReason={reason} loading={save.isPending} onClick={submit}>
        {t(mw.savePlan)}
      </Button>
    </section>
  )
}

/**
 * A choice whose labels run long: a select in English, as on the board, and
 * radio cards in other languages, where a select would cut the longer labels
 * off (Text expansion, rule 1).
 */
function LongChoice<V extends string>({
  label,
  value,
  options,
  onChange,
  disabled,
  hint,
}: {
  label: string
  value: V
  options: { value: V; label: string; disabled?: boolean }[]
  onChange: (v: V) => void
  disabled?: boolean
  hint?: string
}) {
  const { locale } = useI18n()
  const name = useId()
  if (locale === 'en') {
    return (
      <Field label={label} hint={hint}>
        <Select value={value} disabled={disabled} onChange={(e) => onChange(e.target.value as V)}>
          {options.map((o) => (
            <option key={o.value} value={o.value} disabled={o.disabled}>
              {o.label}
            </option>
          ))}
        </Select>
      </Field>
    )
  }
  return (
    <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
      <legend className="mb-1 p-0 text-[12px] font-semibold text-ink-2">{label}</legend>
      {options.map((o) => (
        <OptionCard key={o.value} name={name} value={o.value} checked={value === o.value} disabled={disabled || o.disabled} onChange={(v) => onChange(v as V)} title={o.label} />
      ))}
      {hint ? <span className="k">{hint}</span> : null}
    </fieldset>
  )
}

/** What happened during the warm-up, newest first. */
function WarmupLog({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-[18px]" aria-labelledby="pv-wu-log">
      <h2 className="h2" id="pv-wu-log">
        {t(mw.logTitle)}
      </h2>
      {plan.log.length ? (
        plan.log.map((e, i) => (
          <div key={i} className="flex gap-2.5 text-[13px] max-md:flex-wrap">
            <span className="pv-log-date mono k">{shortDate(e.at)}</span>
            <span>
              {e.title ? <strong>{e.title}</strong> : null}
              {e.title ? ' ' : ''}
              {e.body}
            </span>
          </div>
        ))
      ) : (
        <span className="k">{t(mw.logEmpty)}</span>
      )}
    </section>
  )
}

/** Starts a warm-up for a provider without one. */
function NewPlanDialog({ candidates, onClose }: { candidates: Schemas['WarmupCandidate'][]; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const [providerId, setProviderId] = useState(candidates[0]?.providerId ?? '')
  const cand = candidates.find((c) => c.providerId === providerId)
  const [ip, setIp] = useState(cand?.ips[0] ?? '')
  const [days, setDays] = useState<14 | 21 | 30>(21)
  const [error, setError] = useState('')
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup', {
    onSuccess: (p) => {
      void refresh()
      onClose()
      const first = p.rows[0]
      toast({ kind: 'success', title: t(mw.startedToast, { provider: p.short }), detail: t(mw.startedDetail, { cap: count(first?.cap ?? 0), share: trimPct(first?.share ?? 0) }) })
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mw.newPlan)}
      description={t(mw.newPlanBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button
            variant="primary"
            loading={create.isPending}
            onClick={() =>
              create.mutate({
                params: { path: { workspace: slug } },
                body: { providerId, ip: ip || undefined, days, engagedFirst: true, guardBounce: 2, guardComplaint: 0.1, guardAction: 'hold', spreadMailbox: true, overflow: true },
              })
            }
          >
            {t(mw.startWarmup)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        <Field label={t(mw.provider)}>
          <Select
            value={providerId}
            onChange={(e) => {
              setProviderId(e.target.value)
              setIp(candidates.find((c) => c.providerId === e.target.value)?.ips[0] ?? '')
            }}
          >
            {candidates.map((c) => (
              <option key={c.providerId} value={c.providerId}>
                {c.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(mw.dedicatedIp)} optional hint={t(mw.dedicatedIpHint)}>
          {cand?.ips.length ? (
            <Select value={ip} onChange={(e) => setIp(e.target.value)}>
              <option value="">{t(mw.sharedPool)}</option>
              {cand.ips.map((x) => (
                <option key={x} value={x}>
                  {x}
                </option>
              ))}
            </Select>
          ) : (
            <Input value={ip} onChange={(e) => setIp(e.target.value)} placeholder={'203.0.113.25'} inputMode="decimal" />
          )}
        </Field>
        <LongChoice
          label={t(mw.length)}
          value={String(days)}
          onChange={(v) => setDays(Number(v) as 14 | 21 | 30)}
          options={lengths.map((l) => ({ value: String(l.days), label: t(l.label) }))}
        />
      </div>
    </Dialog>
  )
}

function WarmupSkeleton() {
  const t = useT()
  return (
    <Loading label={t(mw.loading)}>
      <div className="card flex flex-col gap-3.5 px-[22px] py-5">
        <Skeleton w={320} h={20} />
        <Skeleton h={10} />
        <Grid min={180} gap={14}>
          {Array.from({ length: 6 }, (_, i) => (
            <div key={i} className="flex flex-col gap-1.5">
              <Skeleton w={110} h={12} />
              <Skeleton w={80} h={20} />
              <Skeleton w={140} h={12} />
            </div>
          ))}
        </Grid>
      </div>
    </Loading>
  )
}
