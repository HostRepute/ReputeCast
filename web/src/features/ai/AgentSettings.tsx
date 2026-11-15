import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Toggle } from '@/components/ui/Field'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { useAISettings, useRefreshAI } from './api'
import { AgentBudget, agentBase, type AgentRun } from './agent'
import { ag } from './agent.messages'
import { dollars } from './format'

type Limits = Schemas['AIAgentLimits']

export function AgentSettings() {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('integrations:manage')
  const q = $api.useQuery('get', `${agentBase}/settings`, { params: { path: { workspace: slug } } })
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} />
  if (!q.data) return <Skeleton h={200} />
  return <SettingsForm key={q.data.revision} settings={q.data} can={can} />
}

function SettingsForm({ settings, can }: { settings: Schemas['AIAgentSettings']; can: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [enabled, setEnabled] = useState(settings.enabled)
  const [limits, setLimits] = useState(settings.limits)
  const refresh = useRefreshAI()
  const save = $api.useMutation('put', `${agentBase}/settings`, { onSuccess: () => { void refresh(); toast({ kind: 'success', title: t(ag.saved) }) } })
  const probe = $api.useMutation('post', `${agentBase}/test`, { onSettled: () => { void refresh() } })
  const capability = settings.capability?.state ?? 'untested'
  return <Card className="flex flex-col gap-4 p-5 max-md:p-4"><CardHeader title={t(ag.settings)} caption={t(ag.settingsHint)} />
    <div className="flex flex-col items-start gap-2">
      <div role="status"><Badge tone={capability === 'supported' ? 'ok' : capability === 'unsupported' ? 'warn' : 'neutral'} className="whitespace-normal">{t(ag.capability, { state: capability })}</Badge></div>
      <p className="k m-0">{t(ag.toolsHint)}</p>
      {can ? <Button loading={probe.isPending} disabled={save.isPending} onClick={() => probe.mutate({ params: { path: { workspace: slug } } })}>{t(ag.testTools)}</Button> : null}
      {probe.error ? <p className="note b m-0" role="alert">{errorMessage(probe.error)}</p> : null}
    </div>
    <form className="flex flex-col gap-4" onSubmit={(e) => { e.preventDefault(); if (can && !save.isPending) save.mutate({ params: { path: { workspace: slug } }, body: { enabled, limits, revision: settings.revision } }) }}>
      <Toggle label={t(ag.enabled)} checked={enabled} onChange={(e) => setEnabled(e.target.checked)} disabled={!can || save.isPending} />
      <fieldset disabled={!can || save.isPending} className="min-w-0"><legend className="mb-3 text-sm font-semibold">{t(ag.defaults)}</legend><LimitFields limits={limits} onChange={setLimits} /></fieldset>
      <p className="k m-0">{t(ag.defaultsHint)}</p>
      {save.error ? <p className="note b m-0" role="alert">{errorMessage(save.error)}</p> : null}
      {can ? <div><Button type="submit" variant="primary" loading={save.isPending}>{t(common.save)}</Button></div> : null}
    </form>
  </Card>
}

function LimitFields({ limits, minimum, onChange }: { limits: Limits; minimum?: Limits; onChange: (value: Limits) => void }) {
  const t = useT()
  return <div className="grid grid-cols-3 gap-4 max-lg:grid-cols-1">
    <Field label={t(ag.calls)} hint={minimum ? t(ag.currentLimit, { limit: count(minimum.toolCalls) }) : undefined}><Input type="number" required min={minimum?.toolCalls ?? 1} max={200} step={1} value={limits.toolCalls || ''} onChange={(e) => onChange({ ...limits, toolCalls: Number(e.target.value) })} /></Field>
    <Field label={t(ag.tokens)} hint={minimum ? t(ag.currentLimit, { limit: count(minimum.tokens) }) : undefined}><Input type="number" required min={minimum?.tokens ?? 1000} max={2000000} step={1} value={limits.tokens || ''} onChange={(e) => onChange({ ...limits, tokens: Number(e.target.value) })} /></Field>
    <Field label={t(ag.costLimit)} hint={minimum ? t(ag.currentLimit, { limit: dollars(minimum.costMicros) }) : undefined}><Input type="number" required min={(minimum?.costMicros ?? 1) / 1e6} max={100} step="any" value={limits.costMicros / 1e6 || ''} onChange={(e) => onChange({ ...limits, costMicros: Math.round(Number(e.target.value) * 1e6) })} /></Field>
  </div>
}

export function AgentLimitsDialog({ run, onClose, onSave }: { run: AgentRun; onClose: () => void; onSave: (body: Schemas['AIAgentLimitChange']) => Promise<void> }) {
  const t = useT()
  const settings = useAISettings()
  const [limits, setLimits] = useState(run.limits)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const ready = Boolean(settings.data?.connection && !settings.isError && !settings.isPaused)
  return <Dialog open onOpenChange={(open) => { if (!open && !busy) onClose() }} dismissible={!busy} title={t(ag.raise)} description={t(ag.raiseBody)} size="lg" footer={<><Button disabled={busy} onClick={onClose}>{t(common.cancel)}</Button><Button type="submit" form="agent-limits" variant="primary" loading={busy} disabled={!ready}>{t(common.save)}</Button></>}>
    <form id="agent-limits" className="flex flex-col gap-4" onSubmit={(e) => {
      e.preventDefault()
      if (busy || !ready) return
      setBusy(true)
      setError('')
      void onSave({ version: run.version, limits }).then(onClose).catch((err) => setError(errorMessage(err))).finally(() => setBusy(false))
    }}><AgentBudget run={run} />{settings.data?.connection ? <p className="note i m-0">{settings.data.connection.cap.kind === 'none' ? t(ag.monthlyUnlimited) : t(ag.monthlyRemaining, {
      remaining: settings.data.connection.cap.kind === 'tokens' ? count(Math.max(0, settings.data.connection.cap.tokens - settings.data.usage.tokens - settings.data.usage.reservedTokens)) : dollars(Math.max(0, Math.floor(settings.data.connection.cap.usd * 1e6) - settings.data.usage.costMicros - settings.data.usage.reservedCostMicros)),
      date: shortDate(settings.data.usage.resetsAt),
    })}</p> : null}{settings.error ? <ErrorPanel title={errorMessage(settings.error)} actions={<Button onClick={() => void settings.refetch()}>{t(common.retry)}</Button>} /> : settings.isPending ? <Skeleton h={40} /> : !ready ? <p role="status" className="note w m-0">{t(ag.unavailable)}</p> : null}<fieldset disabled={busy || !ready}><LimitFields limits={limits} minimum={run.limits} onChange={setLimits} /></fieldset>{error ? <p role="alert" className="note b m-0">{error}</p> : null}</form>
  </Dialog>
}
