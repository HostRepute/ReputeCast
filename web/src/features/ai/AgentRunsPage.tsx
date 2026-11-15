import { useInfiniteQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Pager } from '@/components/ui/Pager'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { AgentBadge, agentRuns } from './agent'
import { ag } from './agent.messages'
import { aiBase } from './api'
import { AutomationFromBriefDialog } from './AutomationFromBrief'
import { DraftFromBriefDialog } from './DraftFromBrief'
import { md } from './draft.messages'
import { dollars } from './format'
import { PendingDrafts } from './PendingDrafts'

export function AgentRunsPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const me = useMe()
  const can = useCan('ai:use')
  const canCampaigns = useCan('campaigns:view')
  const canAutomations = useCan('automations:view')
  const navigate = useNavigate()
  const [campaign, setCampaign] = useState<Schemas['AICampaignProposal']>()
  const [automation, setAutomation] = useState<Schemas['AIAutomationProposal']>()
  const q = useInfiniteQuery({
    queryKey: ['get', agentRuns, { params: { path: { workspace: slug } } }],
    initialPageParam: '',
    queryFn: async ({ pageParam, signal }) => (await http.GET(agentRuns, { params: { path: { workspace: slug }, query: { cursor: pageParam || undefined } }, signal })).data!,
    getNextPageParam: (page) => page.next || undefined,
    refetchInterval: 15_000,
  })
  if (q.error instanceof ApiError && [403, 404].includes(q.error.status)) throw q.error
  const items = q.data?.pages.flatMap((p) => p.items)
  const newBrief = can ? <ButtonLink variant="primary" to="/w/$workspace/agent/new" params={{ workspace: slug }}>{t(ag.newBrief)}</ButtonLink> : undefined
  return <Page title={t(ag.runs)} subtitle={t(ag.lead)} actions={newBrief}>
    {canCampaigns ? <PendingDrafts kind="campaigns" all onOpen={async (id) => {
      const { data } = await http.GET(`${aiBase}/drafts/campaigns/{proposal}`, { params: { path: { workspace: slug, proposal: id } } })
      if (data?.decision) throw new Error(t(md.alreadyDecided))
      setCampaign(data)
    }} /> : null}
    {canAutomations ? <PendingDrafts kind="automations" all onOpen={async (id) => {
      const { data } = await http.GET(`${aiBase}/drafts/automations/{proposal}`, { params: { path: { workspace: slug, proposal: id } } })
      if (data?.decision) throw new Error(t(md.alreadyDecided))
      setAutomation(data)
    }} /> : null}
    {campaign ? <DraftFromBriefDialog initialProposal={campaign} onClose={() => setCampaign(undefined)} onAccepted={(id) => navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: id } })} /> : null}
    {automation ? <AutomationFromBriefDialog initialProposal={automation} onClose={() => setAutomation(undefined)} onAccepted={(id) => navigate({ to: '/w/$workspace/automations/$automationId', params: { workspace: slug, automationId: id } })} /> : null}
    {q.error ? <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} /> : null}
    {!items ? <Skeleton h={200} /> : !items.length ? <EmptyState title={t(ag.empty)} action={newBrief}>{t(ag.emptyBody)}</EmptyState> : <Card><ul className="m-0 list-none p-0">{items.map((run) => <li key={run.id} className="flex flex-wrap items-center justify-between gap-4 border-b border-line-2 p-5 last:border-0 max-md:p-4">
      <div className="flex min-w-0 flex-1 flex-col gap-1"><Link className="font-semibold text-ink" to="/w/$workspace/agent/$runId" params={{ workspace: slug, runId: run.id }}>{run.title || t(ag.title)}</Link><span className="k">{run.model} · {count(run.tokens)} · {dollars(run.costMicros)}</span><span className="k">{t(ag.owner, { name: run.ownerId === me.data?.user.id ? t(ag.you) : run.ownerName })}</span><span className="k">{t(ag.updated, { date: shortDate(run.updatedAt) })} · {t(ag.expires, { date: shortDate(run.expiresAt) })}</span>{run.uncertain ? <span className="text-xs text-warn">{t(ag.unknown)}</span> : null}</div><AgentBadge state={run.state} />
    </li>)}</ul></Card>}
    {q.hasNextPage ? <div><Pager query={q} label={t(ag.loadMore)} size="md" /></div> : null}
  </Page>
}
