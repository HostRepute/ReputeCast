import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Card, CardHeader } from '@/components/ui/Card'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { aiBase } from './api'
import { ag } from './agent.messages'
import { md } from './draft.messages'
import { dollars } from './format'

export type AgentRun = Schemas['AIAgentRunView']
export type AgentArtifact = Schemas['AIAgentArtifact']
export type AgentAction = Schemas['AIAgentAction']
export const agentBase = `${aiBase}/agent` as const
export const agentRuns = `${agentBase}/runs` as const
export const agentRunPath = `${agentRuns}/{id}` as const
export const agentRunning = (state: string) => ['planning', 'queued', 'building'].includes(state)

export function AgentBadge({ state }: { state: string }) {
  const t = useT()
  return <Badge tone={state === 'completed' || state === 'review' ? 'ok' : state === 'paused' || state === 'needs_answers' || state === 'ready' ? 'warn' : 'neutral'}>{t(ag.state, { state })}</Badge>
}

export function AgentPrivacy() {
  const t = useT()
  return <Card className="flex flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(ag.privacyTitle)} /><p className="m-0 text-[13px] text-ink-2">{t(ag.privacy)}</p><p className="k m-0">{t(md.sharingRisk)}</p><p className="k m-0">{t(md.providerPolicy)}</p><p className="k m-0 border-t border-line-2 pt-3">{t(ag.retention)}</p></Card>
}

export function AgentBudget({ run }: { run: Pick<AgentRun, 'limits' | 'toolCalls' | 'tokens' | 'costMicros' | 'uncertain'> }) {
  const t = useT()
  return <Card className="flex flex-col gap-4 p-5 max-md:p-4"><CardHeader title={t(ag.budget)} />
    <dl className="m-0 flex flex-col gap-3">{[
      [t(ag.calls), count(run.toolCalls), count(run.limits.toolCalls)],
      [t(ag.tokens), count(run.tokens), count(run.limits.tokens)],
      [t(ag.cost), dollars(run.costMicros), dollars(run.limits.costMicros)],
    ].map(([label, used, limit]) => <div key={label} className="flex flex-wrap justify-between gap-2 border-b border-line-2 pb-2"><dt className="text-ink-2">{label}</dt><dd className="m-0 font-semibold tabular-nums">{t(ag.used, { used: used!, limit: limit! })}</dd></div>)}</dl>
    {run.uncertain ? <p role="status" className="note w m-0">{t(ag.unknown)}</p> : null}
    <p className="k m-0">{t(ag.budgetHint)}</p>
  </Card>
}

/** No automatic mutation retries: reconnect reads the durable checkpoint first. */
export function useAgentMutation(id: string) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const locked = useRef(false)
  const [busy, setBusy] = useState(false)
  const path = { workspace: slug, id }
  const refresh = () => qc.invalidateQueries({ predicate: (q) => typeof q.queryKey[1] === 'string' && (q.queryKey[1].startsWith(aiBase) || /\/(campaigns|automations|segments|website)/.test(q.queryKey[1])) })
  async function submit(work: () => Promise<unknown>) {
    if (locked.current) throw new Error(t(ag.pendingAction))
    locked.current = true
    setBusy(true)
    try { await work() } finally {
      try { await refresh() } finally { locked.current = false; setBusy(false) }
    }
  }
  return {
    busy,
    action: (body: AgentAction) => submit(() => http.POST(`${agentRunPath}/actions`, { params: { path }, body })),
    decide: (body: Schemas['AIAgentDecision']) => submit(() => http.POST(`${agentRunPath}/decisions`, { params: { path }, body })),
    limits: (body: Schemas['AIAgentLimitChange']) => submit(() => http.PUT(`${agentRunPath}/limits`, { params: { path }, body })),
  }
}
