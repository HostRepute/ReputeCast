import { useState } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Select, Textarea } from '@/components/ui/Field'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useScrollRegion } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { AgentBadge, AgentBudget, AgentPrivacy, agentRunPath, agentRunning, useAgentMutation, type AgentAction, type AgentRun } from './agent'
import { ag } from './agent.messages'
import { AgentPoolReview } from './AgentPool'
import { AgentReportReview } from './AgentReport'
import { AgentReview } from './AgentReview'
import { AgentLimitsDialog } from './AgentSettings'
import { dollars } from './format'
import { useSharingReview } from './SharingReview'
import './agent.css'

export function AgentRunPage({ id, selectedTab, onTabChange: setTab }: { id: string; selectedTab?: 'plan' | 'steps' | 'review'; onTabChange: (tab: 'plan' | 'steps' | 'review') => void }) {
  const t = useT()
  const { slug, me: workspace } = useWorkspace()
  const me = useMe()
  const can = useCan('ai:use')
  const manager = useCan('integrations:manage')
  const q = $api.useQuery('get', agentRunPath, { params: { path: { workspace: slug, id } } }, {
    refetchInterval: (query) => agentRunning(query.state.data?.state ?? '') ? 2000 : 15_000,
  })
  const mutations = useAgentMutation(id)
  const [confirmation, setConfirmation] = useState<{ run: AgentRun; action: 'build' | 'cancel' }>()
  const [limits, setLimits] = useState<AgentRun>()
  const [error, setError] = useState('')
  const run = q.data
  const tab = selectedTab ?? (run && ['review', 'completed', 'cancelled'].includes(run.state) ? 'review' : 'plan')
  if (q.error instanceof ApiError && [403, 404].includes(q.error.status)) throw q.error
  const own = run?.ownerId === me.data?.user.id
  const blocked = q.isError || q.isPaused
  const act = async (body: AgentAction) => {
    setError('')
    try { await mutations.action(body) } catch (err) { setError(errorMessage(err)); throw err }
  }
  const active = run && agentRunning(run.state)
  const modelActions = Boolean(own && can && !blocked && !mutations.busy)
  const missing = run?.plan?.items.filter((item) => !run.artifacts.some((a) => a.handle === item.handle || (a.bundle === item.bundle && a.decision === 'discarded'))) ?? []
  return <Page title={t(ag.title)} badges={run ? <span role="status"><AgentBadge state={run.state} /></span> : undefined} actions={<ButtonLink to="/w/$workspace/agent" params={{ workspace: slug }}>{t(ag.runs)}</ButtonLink>}>
    {blocked ? <ErrorPanel title={t(ag.offline)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>{errorMessage(q.error)}</ErrorPanel> : null}
    {!run ? !q.isError ? <Skeleton h={250} /> : null : <>
      <div className="flex flex-wrap justify-between gap-3"><p className="k m-0">{run.model} · {t(ag.expires, { date: shortDate(run.expiresAt) })}</p>{own && !['completed', 'cancelled', 'expired'].includes(run.state) ? <Button disabled={blocked || mutations.busy} onClick={() => setConfirmation({ run, action: 'cancel' })}>{t(ag.cancel)}</Button> : null}</div>
      {run.state === 'expired' || run.state === 'cancelled' || run.state === 'completed' ? <p className="note i m-0" role="status">{t(run.state === 'expired' ? ag.expired : run.state === 'cancelled' ? ag.cancelled : ag.completed)}</p> : null}
      {active ? <p role="status" className="note i m-0">{run.queuePosition ? <>{t(ag.queue, { position: run.queuePosition })}<br /></> : null}{t(ag.running)}</p> : null}
      {run.state === 'paused' ? <Card className="flex flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(ag.paused)} /><p className="m-0 text-[13px]">{t(run.pauseReason === 'validation_exhausted' ? ag.validation : run.pauseReason === 'incomplete_plan' ? ag.incompletePlan : ag.pausedReason, { code: run.pauseReason })}</p>
        {missing.length ? <div><strong className="text-[13px]">{t(ag.missingDrafts)}</strong><ul className="mb-0 mt-2 pl-5 text-[13px]">{missing.map((item) => <li key={item.handle}>{item.name}</li>)}</ul></div> : null}
        <div className="flex flex-wrap gap-2"><Button variant="primary" disabled={!modelActions || run.uncertain || run.pauseReason === 'validation_exhausted'} onClick={() => void act({ action: 'continue', version: run.version, planRevision: run.planRevision }).catch(() => {})}>{t(ag.continue)}</Button>{manager ? <Button disabled={blocked || mutations.busy} onClick={() => setLimits(run)}>{t(ag.raise)}</Button> : null}<Button onClick={() => setTab('review')}>{t(ag.review)}</Button></div>{!manager ? <p className="k m-0">{t(ag.manager)}</p> : null}
      </Card> : null}
      {error ? <p className="note b m-0" role="alert">{error}</p> : null}
      <div className="agent-columns"><div className="agent-main">
        <PanelTabs value={tab} onChange={setTab} items={[{ value: 'plan', label: t(ag.planTab) }, { value: 'steps', label: t(ag.steps) }, { value: 'review', label: t(ag.review) }]} label={t(ag.title)}>
          <TabPanel value="plan" className="mt-5 flex min-w-0 flex-col gap-5">
            {run.brief ? <Card className="p-5 max-md:p-4"><CardHeader title={t(ag.brief)} /><p className="agent-source mb-0 text-[13px] text-ink-2">{run.brief.brief}</p></Card> : null}
            {run.report ? <AgentReportReview report={run.report} /> : null}
            {run.pool ? <AgentPoolReview pool={run.pool} /> : null}
            {run.state !== 'expired' ? <Plan run={run} canBuild={modelActions} onBuild={() => setConfirmation({ run, action: 'build' })} /> : null}
          </TabPanel>
          <TabPanel value="steps" className="mt-5"><Card><CardHeader title={t(ag.steps)} className="border-b border-line-2 px-5 py-4" />{!run.steps.length ? <p className="p-5 text-sm text-ink-2">{t(ag.noSteps)}</p> : <ol className="m-0 list-none p-0">{run.steps.map((step) => <li key={step.sequence} className="agent-item"><div className="flex flex-wrap justify-between gap-2"><strong>{count(step.sequence)}</strong><span className="k">{t(ag.stepState, { state: step.state })} · {shortDate(step.updatedAt)}</span></div>{step.operations.map((op, i) => <div key={i} className="mt-3 flex flex-col gap-1 text-[13px]"><span>{op.tool === 'read_report' ? t(ag.reportRead) : t(ag.operation, { tool: op.tool })}{op.handle ? ` · ${run.plan?.items.find((p) => p.handle === op.handle)?.name ?? op.handle}` : ''}</span><p className={`m-0 ${op.failed ? 'text-warn' : 'text-ink-2'}`}>{t(ag.outcome, { failed: String(op.failed), code: op.code })}</p></div>)}</li>)}</ol>}</Card></TabPanel>
          <TabPanel value="review" className="mt-5"><AgentReview run={run} mutations={mutations} permissions={workspace.permissions} blocked={blocked} owner={Boolean(own)} onPlanRequested={() => setTab('plan')} /></TabPanel>
        </PanelTabs>
        {run.state !== 'expired' ? <Conversation run={run} enabled={modelActions && ['needs_answers', 'ready', 'review', 'paused'].includes(run.state) && !run.uncertain} onAction={act} /> : null}
      </div><aside className="agent-aside"><AgentBudget run={run} /><AgentPrivacy /></aside></div>
    </>}
    {confirmation ? <ConfirmDialog open onOpenChange={(open) => { if (!open) setConfirmation(undefined) }} tone="warn" badge={t(ag.title)} title={t(confirmation.action === 'build' ? ag.build : ag.cancel)} description={t(confirmation.action === 'build' ? ag.buildBody : ag.cancelBody)} confirmLabel={t(confirmation.action === 'build' ? ag.build : ag.cancel)} onConfirm={async () => {
      await act({ action: confirmation.action, version: confirmation.run.version, planRevision: confirmation.run.planRevision, ...(confirmation.action === 'build' ? { planDiscards: confirmation.run.planDiscards } : {}) })
      if (confirmation.action === 'build') setTab('steps')
    }}>
      {confirmation.action === 'build' ? <div className="flex flex-col gap-3"><strong>{t(ag.planVersion, { revision: confirmation.run.planRevision })}</strong><ul className="m-0 pl-5">{confirmation.run.plan?.items.map((item) => <li key={item.handle}>{item.name}</li>)}</ul>{confirmation.run.planDiscards.length ? <><p className="note w m-0">{t(ag.replanImpact)}</p><ul className="m-0 pl-5">{confirmation.run.planDiscards.map((a) => <li key={a.id}>{confirmation.run.artifacts.find((item) => item.id === a.id)?.name} · {t(ag.revision, { revision: a.revision })}</li>)}</ul></> : null}</div> : null}
    </ConfirmDialog> : null}
    {limits ? <AgentLimitsDialog run={limits} onClose={() => setLimits(undefined)} onSave={mutations.limits} /> : null}
  </Page>
}

function Plan({ run, canBuild, onBuild }: { run: AgentRun; canBuild: boolean; onBuild: () => void }) {
  const t = useT()
  const plan = run.plan
  return <Card><CardHeader title={t(ag.plan)} caption={t(run.acceptedRevision === run.planRevision && run.acceptedRevision > 0 ? ag.accepted : ag.planVersion, { revision: run.planRevision })} className="border-b border-line-2 px-5 py-4 max-md:px-4" />
    {plan?.summary ? <p className="agent-source m-0 border-b border-line-2 px-5 py-4 text-[13px] text-ink-2 max-md:px-4">{plan.summary}</p> : null}
    {!plan?.items.length ? <p className="p-5 text-[13px] text-ink-2">{t(ag.noPlan)}</p> : <ol className="m-0 list-none p-0">{plan.items.map((item, i) => <li key={item.handle} className="agent-item flex gap-3"><span className="flex size-7 shrink-0 items-center justify-center rounded-full border border-line text-xs text-accent">{i + 1}</span><div className="min-w-0 flex-1"><div className="flex flex-wrap justify-between gap-2"><h3 className="m-0 text-sm">{item.name}</h3><Badge>{t(item.target ? ag.existingDrafts : ag.kind, { kind: item.kind })}</Badge></div><p className="my-2 text-[13px] text-ink-2">{item.purpose}</p><p className="k m-0">{t(ag[`${item.kind}Checks`])}</p>{item.dependsOn.length ? <p className="k mb-0">{t(ag.dependencies, { names: item.dependsOn.map((h) => plan.items.find((p) => p.handle === h)?.name ?? h).join(', ') })}</p> : null}</div></li>)}</ol>}
    {run.audience ? <div className="border-t border-line-2 p-5 max-md:p-4"><h3 className="m-0 text-sm">{t(ag.audience)}</h3><p className="text-[13px]">{run.audience.count !== undefined && run.audience.total !== undefined ? t(ag.count, { count: count(run.audience.count), total: count(run.audience.total), date: shortDate(run.audience.at) }) : t(ag.countUnavailable)}</p><p className="k mb-0">{t(ag.countHint)}</p></div> : null}
    {run.state === 'ready' ? <div className="flex flex-col gap-3 border-t border-line-2 p-5 max-md:p-4">
      {run.estimate ? <>
        <strong className="text-sm">{t(ag.buildEstimate, { low: dollars(run.estimate.lowMicros), high: dollars(run.estimate.highMicros) })}</strong>
        <p className="k m-0">{t(ag.estimateHint, { date: shortDate(run.estimate.pricingAt) })}</p>
        {run.estimate.highMicros > run.limits.costMicros - run.costMicros ? <p className="note w m-0">{t(ag.estimateCeiling)}</p> : null}
      </> : <p className="note w m-0">{t(ag.estimateUnknown)}</p>}
      <p className="k m-0">{t(ag.buildBody)}</p><div><Button variant="primary" disabled={!canBuild || !plan?.items.length || Boolean(plan.questions.length)} onClick={onBuild}>{t(ag.build)}</Button></div>
    </div> : null}
  </Card>
}

function Conversation({ run, enabled, onAction }: { run: AgentRun; enabled: boolean; onAction: (body: AgentAction) => Promise<void> }) {
  const t = useT()
  const [scrollRef, scrollProps] = useScrollRegion<HTMLDivElement>(t(ag.conversation), true)
  const [message, setMessage] = useState('')
  const [handle, setHandle] = useState('')
  const [reviewSharing, sharingDialog] = useSharingReview(run.model)
  const answering = run.state === 'needs_answers'
  const revisable = run.acceptedRevision > 0 && ['review', 'paused'].includes(run.state) ? run.artifacts.filter((a) => !a.decision && !a.targetId) : []
  const selected = revisable.some((a) => a.handle === handle) ? handle : ''
  return <Card className="p-5 max-md:p-4"><CardHeader title={t(answering ? ag.questions : ag.conversation)} caption={t(ag.conversationHint)} />
    {run.plan?.questions.length ? <ol className="pl-5 text-[13px]">{run.plan.questions.map((question, i) => <li key={i}>{question}</li>)}</ol> : null}
    {run.conversation.length ? <div ref={scrollRef} {...scrollProps} className="my-4 max-h-80 overflow-auto"><ol className="m-0 flex list-none flex-col gap-3 p-0">{run.conversation.map((m, i) => <li key={i} className="rounded-lg border border-line-2 bg-ground p-3"><div className="k">{t(ag.speaker, { role: m.role })}{m.handle ? ` · ${m.handle}` : ''}</div><p className="agent-source mb-0 mt-1 text-[13px]">{m.text}</p></li>)}</ol></div> : null}
    <form className="mt-4 flex flex-col gap-3" onSubmit={(e) => { e.preventDefault(); if (enabled && message.trim()) reviewSharing(() => onAction({ sharingReviewed: true, action: answering ? 'answer' : selected ? 'revise' : 'replan', version: run.version, planRevision: run.planRevision, message: message.trim(), ...(selected ? { handle: selected } : {}) }).then(() => setMessage(''))) }}>
      {revisable.length ? <Field label={t(ag.scope)}><Select value={selected} onChange={(e) => setHandle(e.target.value)} disabled={!enabled}><option value="">{t(ag.wholePlan)}</option>{revisable.map((a) => <option key={a.id} value={a.handle}>{a.name}</option>)}</Select></Field> : null}
      <Field label={t(answering ? ag.answer : ag.instruction)}><Textarea rows={3} value={message} maxLength={4000} onChange={(e) => setMessage(e.target.value)} disabled={!enabled} required /></Field>
      <div><Button type="submit" disabled={!enabled || !message.trim()}>{t(answering ? ag.answerSend : ag.send)}</Button></div>
    </form>
    {sharingDialog}
  </Card>
}
