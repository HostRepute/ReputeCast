import { useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Segmented } from '@/components/ui/Tabs'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useI18n, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count, decimal, pct, shortDate, trimPct, weekdayName } from '@/lib/format'
import { m as fm } from '@/lib/format.messages'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { limitPct, perSec, ProvidersTabs, throwIfForbidden } from './common'
import { m } from './messages'
import { audienceText, dayStatus, guardIn, rampHeight, visibleDays, type WarmupPlan } from './warmup-model'
import { mw } from './warmup.messages'
import { WarmupPlacementCard } from '@/features/placement/EntryPoints'

const warmupKey = ['get', '/api/v1/workspaces/{workspace}/warmup'] as const
const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** Plan lengths, in days, and how each one is described. */
const lengths = [
  { days: 21, label: mw.length21 },
  { days: 14, label: mw.length14 },
  { days: 30, label: mw.length30 },
] as const

/** 7.3 Warm-up: the ramp of a new IP or provider, its guards and its log. */
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
          <PlanSummary plan={plan} />
          <div className="flex flex-wrap items-start gap-5">
            <RampCard plan={plan} />
            <aside className="flex max-w-[440px] flex-[1_1_340px] flex-col gap-4 max-md:min-w-0" aria-label={t(mw.asideLabel)}>
              <PlanSettings key={plan.id} plan={plan} />
              <WarmupPlacementCard key={`pl-${plan.id}`} providerId={plan.providerId} provider={plan.short} day={plan.day} days={plan.days} />
              <WarmupLog plan={plan} />
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

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div>
      <div className="k">{label}</div>
      <div className="n text-[20px]">{value}</div>
      <div className="k">{sub}</div>
    </div>
  )
}

/** The plan's headline: progress, today's cap, totals, guards, slowdowns and rate. */
function PlanSummary({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const today = plan.schedule.find((d) => d.status === 'today')
  const quietToday = today && today.uses.length === 0
  const rate = (v: number) => t(fm.perSecond, { n: perSec(v) })
  const statusBadge =
    plan.status === 'paused' ? (
      <Badge tone="warn" dot>
        {t(mw.badgePaused, { day: plan.day, days: plan.days })}
      </Badge>
    ) : plan.status === 'done' ? (
      <Badge tone="ok" dot>
        {t(mw.badgeDone, { days: plan.days })}
      </Badge>
    ) : (
      <Badge tone="info" dot>
        {t(m.statusWarmingUpDay, { day: plan.day, days: plan.days })}
      </Badge>
    )
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="pv-wu-title">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="h2 text-[17px]" id="pv-wu-title">
          {plan.ip ? t(mw.titleIp, { provider: plan.short, ip: plan.ip }) : plan.short}
        </h2>
        {statusBadge}
        <Badge>{t(mw.started, { date: shortDate(plan.startDate) })}</Badge>
        <span className="k ml-auto max-md:ml-0">
          {plan.status === 'done'
            ? t(mw.finished, { date: shortDate(plan.endDate), provider: plan.short })
            : t(mw.finishes, { date: shortDate(plan.endDate), provider: plan.short })}
        </span>
      </div>
      {plan.statusDetail ? <span className="text-[13px] text-warn-fg">{plan.statusDetail}</span> : null}
      <div className="flex h-2.5 gap-0.5 overflow-hidden rounded-[5px] bg-line-2" role="img" aria-label={t(mw.progressLabel, { day: plan.day, days: plan.days })}>
        <div style={{ width: `${plan.progress}%`, background: '#1c5cab' }} />
      </div>
      <Grid min={180} gap={14}>
        <Stat
          label={t(mw.todayCap)}
          value={count(plan.todayCap)}
          sub={
            today
              ? quietToday
                ? t(mw.sentOnQuiet, { sent: count(plan.sentToday), day: weekdayName(today.date) })
                : t(mw.sentOn, { sent: count(plan.sentToday), day: weekdayName(today.date) })
              : t(mw.sentToday, { sent: count(plan.sentToday) })
          }
        />
        <Stat label={t(mw.sentDuring)} value={count(plan.sentTotal)} sub={t(mw.plannedSoFar, { planned: count(plan.plannedToDate) })} />
        <Stat
          label={t(mw.bounceGuard, { limit: limitPct(plan.settings.guardBounce) })}
          value={plan.bounceRate !== undefined ? pct(plan.bounceRate, 1) : '—'}
          sub={t(mw.sevenDay)}
        />
        <Stat
          label={t(mw.complaintGuard, { limit: limitPct(plan.settings.guardComplaint) })}
          value={plan.complaintRate !== undefined ? pct(plan.complaintRate, 2) : '—'}
          sub={t(mw.sevenDay)}
        />
        <Stat
          label={t(mw.slowdowns)}
          value={count(plan.slowdowns)}
          sub={
            plan.lastSlowdown
              ? t(mw.lastSlowdown, { action: plan.settings.guardAction, day: plan.lastSlowdown.day, bounce: trimPct(plan.lastSlowdown.bounce) })
              : t(mw.noneSoFar)
          }
        />
        <Stat
          label={t(mw.rateToday)}
          value={rate(plan.rateToday)}
          sub={plan.nextRate ? t(mw.risesTo, { rate: rate(plan.nextRate.value), day: plan.nextRate.day }) : t(mw.staysThere)}
        />
      </Grid>
    </section>
  )
}

/** The daily ramp chart and the table of nearby days. */
function RampCard({ plan }: { plan: WarmupPlan }) {
  const t = useT()
  const days = plan.schedule
  const max = Math.max(...days.map((d) => d.cap), 1)
  const first = days[0]
  const last = days[days.length - 1]
  const today = days.find((d) => d.status === 'today')
  const done = days.filter((d) => d.sent !== undefined && d.status !== 'today' && d.status !== 'planned')
  const rampValues = { first: count(first?.cap ?? 0), last: count(last?.cap ?? 0), day: last?.day ?? plan.days }
  const label = done.length ? t(mw.rampLabelSent, { ...rampValues, done: done[done.length - 1]!.day }) : t(mw.rampLabel, rampValues)
  return (
    <section className="card flex min-w-0 flex-[2_1_600px] flex-col" aria-labelledby="pv-ramp">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-ramp">
          {t(mw.rampTitle)}
        </h2>
        <div className="flex gap-3.5 text-[12px]">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-info-bg" aria-hidden="true" />
            {t(mw.legendPlanned)}
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-[#1c5cab]" aria-hidden="true" />
            {t(mw.legendSent)}
          </span>
        </div>
      </div>
      <div className="flex flex-col gap-2 px-5 py-[18px]">
        <div className="flex h-[180px] items-end gap-1" role="img" aria-label={label}>
          {days.map((d) => {
            const isDone = d.status !== 'today' && d.status !== 'planned'
            return (
              <div key={d.day} className="pv-col" style={d.status === 'today' ? { borderBottom: '3px solid var(--ink)' } : undefined}>
                {isDone ? (
                  <div className="sent" style={{ height: `${rampHeight(d.sent ?? 0, max)}%`, background: d.status === 'slowdown' ? 'var(--warn-bar)' : undefined }} />
                ) : (
                  <div className="cap" style={{ height: `${rampHeight(d.cap, max)}%` }} />
                )}
              </div>
            )
          })}
        </div>
        <div className="k flex justify-between">
          <span>{t(mw.dayFirst, { cap: count(first?.cap ?? 0) })}</span>
          {today ? <span>{t(mw.dayToday, { day: today.day, cap: count(today.cap) })}</span> : null}
          <span>{t(mw.dayLast, { day: last?.day ?? plan.days, cap: count(last?.cap ?? 0) })}</span>
        </div>
      </div>
      <div className="border-t border-line-2">
        <Table className="pv-9 [&_td]:px-2.5 [&_th]:px-2.5" phoneWrap fit>
          <thead>
            <tr>
              <Th>{t(mw.colDay)}</Th>
              <Th>{t(mw.colDate)}</Th>
              <Th align="right">{t(mw.colCap)}</Th>
              <Th align="right">{t(mw.colSent)}</Th>
              <Th align="right">{t(mw.colBounce)}</Th>
              <Th align="right">{t(mw.colComplaints)}</Th>
              <Th>{t(mw.colAudience)}</Th>
              <Th>{t(mw.colStatus)}</Th>
            </tr>
          </thead>
          <tbody>
            {visibleDays(days).map((d) => {
              const st = dayStatus[d.status]
              const isToday = d.status === 'today'
              const future = d.status === 'planned'
              return (
                <tr key={d.day} className={cn(d.status === 'slowdown' && 'bg-warn-tint', isToday && 'bg-accent-tint')}>
                  <Td className={cn(isToday && 'font-bold')}>{d.day}</Td>
                  <Td className={cn(isToday && 'font-bold')}>{shortDate(d.date)}</Td>
                  <Td align="right" num>
                    {d.held ? t(mw.capHeld, { cap: count(d.cap) }) : count(d.cap)}
                  </Td>
                  <Td align="right" num>
                    {future || d.sent === undefined ? '–' : count(d.sent)}
                  </Td>
                  <Td align="right" num className={cn(d.status === 'slowdown' && 'text-warn-fg')}>
                    {d.bounce !== undefined ? pct(d.bounce, 1) : '–'}
                  </Td>
                  <Td align="right" num>
                    {d.complaint !== undefined ? pct(d.complaint, 2) : '–'}
                  </Td>
                  <Td>{audienceText(d)}</Td>
                  <Td>
                    <Badge tone={st.tone}>{st.text(plan.settings.guardBounce)}</Badge>
                  </Td>
                </tr>
              )
            })}
          </tbody>
        </Table>
      </div>
    </section>
  )
}

/** Plan settings: length, ordering, guards and what happens when one trips. */
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
      body: { days, engagedFirst: engaged, guardBounce: b!, guardComplaint: c!, guardAction: action, spreadMailbox: spread },
    })
  }
  const ro = Boolean(reason)
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
        options={lengths.map((l) => ({ value: String(l.days), label: t(l.label) }))}
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
      <Grid cols={2} gap={10}>
        <Field label={t(mw.slowIfBounces)} error={errors.bounce}>
          <Input value={bounce} disabled={ro} inputMode="decimal" onChange={(e) => setBounce(e.target.value)} />
        </Field>
        <Field label={t(mw.orComplaints)} error={errors.complaint}>
          <Input value={complaint} disabled={ro} inputMode="decimal" onChange={(e) => setComplaint(e.target.value)} />
        </Field>
      </Grid>
      <LongChoice
        label={t(mw.whenTrips)}
        value={action}
        disabled={ro}
        onChange={setAction}
        options={[
          { value: 'hold', label: t(mw.actionHold) },
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
}: {
  label: string
  value: V
  options: { value: V; label: string }[]
  onChange: (v: V) => void
  disabled?: boolean
}) {
  const { locale } = useI18n()
  const name = useId()
  if (locale === 'en') {
    return (
      <Field label={label}>
        <Select value={value} disabled={disabled} onChange={(e) => onChange(e.target.value as V)}>
          {options.map((o) => (
            <option key={o.value} value={o.value}>
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
        <OptionCard key={o.value} name={name} value={o.value} checked={value === o.value} disabled={disabled} onChange={(v) => onChange(v as V)} title={o.label} />
      ))}
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
      toast({ kind: 'success', title: t(mw.startedToast, { provider: p.short }), detail: t(mw.startedDetail, { cap: count(p.schedule[0]?.cap ?? 0), share: trimPct(p.schedule[0]?.share ?? 0) }) })
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
                body: { providerId, ip: ip || undefined, days, engagedFirst: true, guardBounce: 2, guardComplaint: 0.1, guardAction: 'hold', spreadMailbox: true },
              })
            }
          >
            {t(mw.startWarmup)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? <span className="err" role="alert">{error}</span> : null}
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
