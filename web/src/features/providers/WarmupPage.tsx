import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Segmented } from '@/components/ui/Tabs'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { count, shortDate, weekdayName } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { limitPct, ProvidersTabs, throwIfForbidden } from './common'
import { parseNumber } from './ProviderSettingsDialog'
import { audienceText, dayStatus, rampHeight, visibleDays, type WarmupPlan } from './warmup-model'

const warmupKey = ['get', '/api/v1/workspaces/{workspace}/warmup'] as const
const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** 7.3 Warm-up: the ramp of a new IP or provider, its guards and its log. */
export function WarmupPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/warmup', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  const [picked, setPicked] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const plans = q.data?.plans ?? []
  const plan = plans.find((p) => p.id === picked) ?? plans.find((p) => p.status !== 'done') ?? plans[0]
  return (
    <Page
      title="Sending providers & routing"
      filters={<ProvidersTabs />}
      actions={q.data ? <WarmupActions plan={plan} canCreate={q.data.candidates.length > 0} onNew={() => setCreating(true)} /> : null}
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the warm-up" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} The ramp keeps running on the saved plan.
        </ErrorPanel>
      ) : q.isPending ? (
        <WarmupSkeleton />
      ) : !plan ? (
        <EmptyState
          title="No warm-up running"
          action={
            q.data.candidates.length ? (
              <Button variant="primary" onClick={() => setCreating(true)}>
                New warm-up plan
              </Button>
            ) : null
          }
        >
          A new dedicated IP or a new provider needs a few weeks of slowly rising volume before it can carry your Broadcasts. Start a plan and ReputeCast ramps the
          provider&apos;s share daily, sending to your most engaged contacts first.
        </EmptyState>
      ) : (
        <>
          {plans.length > 1 ? (
            <Segmented label="Warm-up plan" value={plan.id} onChange={setPicked} items={plans.map((p) => ({ value: p.id, label: `${p.short}${p.ip ? ` · ${p.ip}` : ''}` }))} />
          ) : null}
          <PlanSummary plan={plan} />
          <div className="flex flex-wrap items-start gap-5">
            <RampCard plan={plan} />
            <aside className="flex max-w-[440px] flex-[1_1_340px] flex-col gap-4 max-md:min-w-0" aria-label="Plan settings and log">
              <PlanSettings key={plan.id} plan={plan} />
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
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const reason = usePermissionReason('providers:manage')
  const onDone = (paused: boolean) => () => {
    void refresh()
    toast({ kind: 'success', title: paused ? 'Warm-up paused' : 'Warm-up resumed', detail: paused ? 'Today’s cap stays where it is until you resume.' : 'The ramp continues from today’s cap.' })
  }
  const onError = (e: unknown) => toast({ kind: 'error', title: "The warm-up didn't change", detail: errorMessage(e) })
  const pause = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/pause', { onSuccess: onDone(true), onError })
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/resume', { onSuccess: onDone(false), onError })
  const path = plan ? { workspace: slug, planId: plan.id } : undefined
  return (
    <>
      {plan && plan.status !== 'done' ? (
        plan.status === 'paused' ? (
          <Button disabledReason={reason} loading={resume.isPending} onClick={() => path && resume.mutate({ params: { path } })}>
            Resume warm-up
          </Button>
        ) : (
          <Button disabledReason={reason} loading={pause.isPending} onClick={() => path && pause.mutate({ params: { path } })}>
            Pause warm-up
          </Button>
        )
      ) : null}
      <Button variant="primary" disabledReason={reason ?? (canCreate ? undefined : 'Every connected provider already has a warm-up plan.')} onClick={onNew}>
        New warm-up plan
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
  const today = plan.schedule.find((d) => d.status === 'today')
  const quietToday = today && today.uses.length === 0
  const guardWord = plan.settings.guardAction === 'step_back' ? 'stepped back' : plan.settings.guardAction === 'pause' ? 'paused' : 'held'
  const statusBadge =
    plan.status === 'paused' ? (
      <Badge tone="warn" dot>
        Paused · day {plan.day} of {plan.days}
      </Badge>
    ) : plan.status === 'done' ? (
      <Badge tone="ok" dot>
        Warmed up · {plan.days} days
      </Badge>
    ) : (
      <Badge tone="info" dot>
        Warming up · day {plan.day} of {plan.days}
      </Badge>
    )
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="pv-wu-title">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="h2 text-[17px]" id="pv-wu-title">
          {plan.short}
          {plan.ip ? ` · dedicated IP ${plan.ip}` : ''}
        </h2>
        {statusBadge}
        <Badge>started {shortDate(plan.startDate)}</Badge>
        <span className="k ml-auto max-md:ml-0">
          {plan.status === 'done'
            ? `Finished ${shortDate(plan.endDate)}; ${plan.short} can take Broadcast failover`
            : `Finishes ${shortDate(plan.endDate)}, then ${plan.short} is eligible as a Broadcast failover target`}
        </span>
      </div>
      {plan.statusDetail ? <span className="text-[13px] text-warn-fg">{plan.statusDetail}</span> : null}
      <div className="flex h-2.5 gap-0.5 overflow-hidden rounded-[5px] bg-line-2" role="img" aria-label={`Day ${plan.day} of ${plan.days}`}>
        <div style={{ width: `${plan.progress}%`, background: '#1c5cab' }} />
      </div>
      <Grid min={180} gap={14}>
        <Stat
          label="Today's cap"
          value={count(plan.todayCap)}
          sub={`${count(plan.sentToday)} sent · ${today ? weekdayName(today.date) : 'today'}${quietToday ? ', no broadcasts' : ''}`}
        />
        <Stat label="Sent during warm-up" value={count(plan.sentTotal)} sub={`of ${count(plan.plannedToDate)} planned so far`} />
        <Stat label={`Bounce rate · guard ${limitPct(plan.settings.guardBounce)}`} value={plan.bounceRate !== undefined ? `${plan.bounceRate.toFixed(1)}%` : '—'} sub="7-day" />
        <Stat
          label={`Complaint rate · guard ${limitPct(plan.settings.guardComplaint)}`}
          value={plan.complaintRate !== undefined ? `${plan.complaintRate.toFixed(2)}%` : '—'}
          sub="7-day"
        />
        <Stat
          label="Slowdowns"
          value={count(plan.slowdowns)}
          sub={plan.lastSlowdown ? `day ${plan.lastSlowdown.day} ${guardWord}, bounces ${plan.lastSlowdown.bounce}%` : 'none so far'}
        />
        <Stat
          label="Rate limit today"
          value={`${plan.rateToday} /s`}
          sub={plan.nextRate ? `rises to ${plan.nextRate.value} /s on day ${plan.nextRate.day}` : 'stays there to the end'}
        />
      </Grid>
    </section>
  )
}

/** The daily ramp chart and the table of nearby days. */
function RampCard({ plan }: { plan: WarmupPlan }) {
  const days = plan.schedule
  const max = Math.max(...days.map((d) => d.cap), 1)
  const first = days[0]
  const last = days[days.length - 1]
  const today = days.find((d) => d.status === 'today')
  const done = days.filter((d) => d.sent !== undefined && d.status !== 'today' && d.status !== 'planned')
  const label = `Daily caps rising from ${count(first?.cap ?? 0)} on day 1 to ${count(last?.cap ?? 0)} on day ${last?.day ?? plan.days}${
    done.length ? `, with sent volume filled in for days 1 to ${done[done.length - 1]!.day}` : ''
  }`
  return (
    <section className="card flex min-w-0 flex-[2_1_600px] flex-col" aria-labelledby="pv-ramp">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-ramp">
          Daily volume ramp
        </h2>
        <div className="flex gap-3.5 text-[12px]">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-info-bg" aria-hidden="true" />
            planned cap
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-[2px] bg-[#1c5cab]" aria-hidden="true" />
            sent
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
          <span>
            Day 1 · {count(first?.cap ?? 0)}
          </span>
          {today ? (
            <span>
              Day {today.day} · today · {count(today.cap)}
            </span>
          ) : null}
          <span>
            Day {last?.day} · {count(last?.cap ?? 0)}
          </span>
        </div>
      </div>
      <div className="border-t border-line-2">
        <Table className="pv-9" phoneWrap>
          <thead>
            <tr>
              <Th>Day</Th>
              <Th>Date</Th>
              <Th align="right">Cap</Th>
              <Th align="right">Sent</Th>
              <Th align="right">Bounce</Th>
              <Th align="right">Complaints</Th>
              <Th>Audience</Th>
              <Th>Status</Th>
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
                    {count(d.cap)}
                    {d.held ? ' · held' : ''}
                  </Td>
                  <Td align="right" num>
                    {future || d.sent === undefined ? '–' : count(d.sent)}
                  </Td>
                  <Td align="right" num className={cn(d.status === 'slowdown' && 'text-warn-fg')}>
                    {d.bounce !== undefined ? `${d.bounce.toFixed(1)}%` : '–'}
                  </Td>
                  <Td align="right" num>
                    {d.complaint !== undefined ? `${d.complaint.toFixed(2)}%` : '–'}
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
  const { slug } = useWorkspace()
  const refresh = useRefreshWarmup()
  const reason = usePermissionReason('providers:manage')
  const s = plan.settings
  const [days, setDays] = useState<14 | 21 | 30>(s.days)
  const [engaged, setEngaged] = useState(s.engagedFirst)
  const [bounce, setBounce] = useState(`${s.guardBounce.toFixed(1)} %`)
  const [complaint, setComplaint] = useState(`${s.guardComplaint.toFixed(2)} %`)
  const [action, setAction] = useState(s.guardAction)
  const [spread, setSpread] = useState(s.spreadMailbox)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/warmup/{planId}', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: 'Warm-up plan saved', detail: 'Changes apply from tomorrow’s cap.' })
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const submit = () => {
    const b = parseNumber(bounce)
    const c = parseNumber(complaint)
    const next: Record<string, string> = {}
    if (b === undefined || b < 0.5 || b > 10) next.bounce = 'Use a bounce guard between 0.5% and 10%.'
    if (c === undefined || c < 0.01 || c > 1) next.complaint = 'Use a complaint guard between 0.01% and 1%.'
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
        Plan settings
      </h2>
      {errors.form ? (
        <span className="err" role="alert">
          {errors.form}
        </span>
      ) : null}
      <Field label="Length">
        <Select value={days} disabled={ro} onChange={(e) => setDays(Number(e.target.value) as 14 | 21 | 30)}>
          <option value={21}>21 days · recommended for a new dedicated IP</option>
          <option value={14}>14 days · fast</option>
          <option value={30}>30 days · cautious</option>
        </Select>
      </Field>
      <Field label="Ordering">
        <Select value={engaged ? 'engaged' : 'random'} disabled={ro} onChange={(e) => setEngaged(e.target.value === 'engaged')}>
          <option value="engaged">Most engaged contacts first (Active, then Cooling, then At risk)</option>
          <option value="random">Random sample</option>
        </Select>
      </Field>
      <Grid cols={2} gap={10}>
        <Field label="Slow down if bounces exceed" error={errors.bounce}>
          <Input value={bounce} disabled={ro} inputMode="decimal" onChange={(e) => setBounce(e.target.value)} />
        </Field>
        <Field label="or complaints exceed" error={errors.complaint}>
          <Input value={complaint} disabled={ro} inputMode="decimal" onChange={(e) => setComplaint(e.target.value)} />
        </Field>
      </Grid>
      <Field label="When a guard trips">
        <Select value={action} disabled={ro} onChange={(e) => setAction(e.target.value as typeof action)}>
          <option value="hold">Hold tomorrow&apos;s cap at today&apos;s level, then continue</option>
          <option value="step_back">Step back one day</option>
          <option value="pause">Pause until I resume</option>
        </Select>
      </Field>
      <label className="flex cursor-pointer items-start gap-2.5 text-[13px]">
        <input type="checkbox" className="pv-check" checked={spread} disabled={ro} onChange={(e) => setSpread(e.target.checked)} />
        <span>Spread each day&apos;s cap evenly across mailbox providers, so no single provider sees a spike from the new IP</span>
      </label>
      <Button variant="primary" disabledReason={reason} loading={save.isPending} onClick={submit}>
        Save plan
      </Button>
    </section>
  )
}

/** What happened during the warm-up, newest first. */
function WarmupLog({ plan }: { plan: WarmupPlan }) {
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-[18px]" aria-labelledby="pv-wu-log">
      <h2 className="h2" id="pv-wu-log">
        Warm-up log
      </h2>
      {plan.log.length ? (
        plan.log.map((e, i) => (
          <div key={i} className="flex gap-2.5 text-[13px] max-md:flex-wrap">
            <span className="mono k">{shortDate(e.at)}</span>
            <span>
              {e.title ? <strong>{e.title}</strong> : null}
              {e.title ? ' ' : ''}
              {e.body}
            </span>
          </div>
        ))
      ) : (
        <span className="k">Nothing yet: the first day&apos;s result shows up here tomorrow.</span>
      )}
    </section>
  )
}

/** Starts a warm-up for a provider without one. */
function NewPlanDialog({ candidates, onClose }: { candidates: Schemas['WarmupCandidate'][]; onClose: () => void }) {
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
      toast({ kind: 'success', title: `${p.short} warm-up started`, detail: `Day 1 cap ${count(p.schedule[0]?.cap ?? 0)}; its Broadcasts share starts at ${p.schedule[0]?.share ?? 0}%.` })
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="New warm-up plan"
      description="ReputeCast raises the provider's share of Broadcasts day by day, most engaged contacts first, and slows down if bounces or complaints climb."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
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
            Start warm-up
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? <span className="err" role="alert">{error}</span> : null}
        <Field label="Provider">
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
        <Field label="Dedicated IP" optional hint="Leave empty to warm the provider's shared pool for this domain.">
          {cand?.ips.length ? (
            <Select value={ip} onChange={(e) => setIp(e.target.value)}>
              <option value="">Shared pool</option>
              {cand.ips.map((x) => (
                <option key={x} value={x}>
                  {x}
                </option>
              ))}
            </Select>
          ) : (
            <Input value={ip} onChange={(e) => setIp(e.target.value)} placeholder="203.0.113.25" inputMode="decimal" />
          )}
        </Field>
        <Field label="Length">
          <Select value={days} onChange={(e) => setDays(Number(e.target.value) as 14 | 21 | 30)}>
            <option value={21}>21 days · recommended for a new dedicated IP</option>
            <option value={14}>14 days · fast</option>
            <option value={30}>30 days · cautious</option>
          </Select>
        </Field>
      </div>
    </Dialog>
  )
}

function WarmupSkeleton() {
  return (
    <Loading label="Loading the warm-up plan">
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
