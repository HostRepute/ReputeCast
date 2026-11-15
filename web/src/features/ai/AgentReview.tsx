import { useRef, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { m as segmentMessages } from '@/features/audience/segments.messages'
import { emptyRefs, type FlowGraph, type Refs } from '@/features/automations/graph'
import { mc as postMessages } from '@/features/website/composer/messages'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { ag } from './agent.messages'
import { agentRunning, type AgentArtifact, type AgentRun, type useAgentMutation } from './agent'
import { GraphReview } from './AutomationFromBrief'
import { DraftEmailReview } from './DraftFromBrief'
import { md } from './draft.messages'

type Decision = Schemas['AIAgentDecision']['decision']
const decisions = { kept: ag.keep, discarded: ag.discard, accepted: ag.accept, rejected: ag.reject }
const descriptions = { kept: ag.keepBody, discarded: ag.discardBody, accepted: ag.acceptBody, rejected: ag.rejectBody }
export const artifactPermission = { campaign: 'campaigns:edit', automation: 'automations:edit', segment: 'segments:edit', post: 'website:edit' }

/** A decision captures displayed revisions, not whatever arrives during polling. */
export function AgentReview({ run, mutations, permissions, blocked, owner, onPlanRequested }: {
  run: AgentRun
  mutations: ReturnType<typeof useAgentMutation>
  permissions: readonly string[]
  blocked: boolean
  owner: boolean
  onPlanRequested?: () => void
}) {
  const t = useT()
  const [confirm, setConfirm] = useState<{ decision: Decision; version: number; artifacts: AgentArtifact[] }>()
  const [source, setSource] = useState<{ action: 'propose' | 'refresh'; artifact: AgentArtifact; version: number; planRevision: number }>()
  const [conflict, setConflict] = useState('')
  const heading = useRef<HTMLHeadingElement>(null)
  // A decision removes the buttons that opened its dialog: keyboard focus
  // continues from the review heading instead of the start of the page.
  const settle = () => requestAnimationFrame(() => { if (document.activeElement === document.body) heading.current?.focus() })
  const grouped = new Map<string, AgentArtifact[]>()
  const order = new Map(run.plan?.items.map((item, i) => [item.handle, i]))
  for (const artifact of [...run.artifacts].sort((a, b) => (order.get(a.handle) ?? Infinity) - (order.get(b.handle) ?? Infinity))) {
    const key = `${artifact.targetId ? 'existing' : 'new'}:${artifact.bundle}`
    grouped.set(key, [...(grouped.get(key) ?? []), artifact])
  }
  const groups = [...grouped.values()]
  const pending = groups.filter((g) => g.some((a) => !a.decision)).length
  const inactive = blocked || agentRunning(run.state) || run.state === 'expired' || !run.acceptedRevision || mutations.busy
  const decide = async () => {
    if (!confirm) return
    try {
      await mutations.decide({ version: confirm.version, decision: confirm.decision, artifacts: confirm.artifacts.map(({ id, revision }) => ({ id, revision })) })
      setConflict('')
    } catch (error) {
      if (confirm.decision === 'accepted' && error instanceof ApiError && error.code === 'ai_draft_changed') setConflict(confirm.artifacts[0]!.id)
      throw error
    }
  }
  return <div className="flex flex-col gap-5">
    <div role="status" className="flex flex-wrap items-center justify-between gap-3"><h2 ref={heading} tabIndex={-1} className="h2">{t(ag.review)}</h2><Badge tone={pending ? 'warn' : 'ok'}>{t(ag.remaining, { count: pending })}</Badge></div>
    <p className="note i m-0">{t(ag.noSend)}</p>
    {!groups.length ? <p className="k">{t(ag.noDrafts)}</p> : null}
    {(['new', 'existing'] as const).map((section) => {
      const visible = groups.filter((g) => Boolean(g[0]!.targetId) === (section === 'existing'))
      if (!visible.length) return null
      return <section key={section} className="flex min-w-0 flex-col gap-4" aria-label={t(section === 'new' ? ag.newDrafts : ag.existingDrafts)}>
        <h2 className="h2">{t(section === 'new' ? ag.newDrafts : ag.existingDrafts)}</h2>
        {visible.map((group) => {
          const first = group[0]!
          const undecided = group.filter((a) => !a.decision)
          const complete = (run.plan?.items.filter((p) => p.bundle === first.bundle) ?? []).every((p) => group.some((a) => a.handle === p.handle && !a.decision))
          const missingPermission = group.map((a) => artifactPermission[a.kind]).find((p) => !permissions.includes(p))
          const allowed = !missingPermission
          const disabled = inactive || !allowed
          return <Card key={first.id}>
            <CardHeader title={first.name} caption={first.bundle} className="border-b border-line-2 px-5 py-4 max-md:px-4" actions={<Badge tone={undecided.length ? 'warn' : 'neutral'}>{t(ag.decision, { decision: undecided.length ? '' : first.decision })}</Badge>} />
            <div className="flex min-w-0 flex-col gap-4 p-5 max-md:p-4">{group.map((artifact) => <article key={`${artifact.id}:${artifact.revision}`} className="min-w-0">
              <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="m-0 text-sm">{artifact.name}</h3><span className="k">{t(ag.kind, { kind: artifact.kind })} · {t(ag.revision, { revision: artifact.revision })}</span>{artifact.decision ? <Badge>{t(ag.decision, { decision: artifact.decision })}</Badge> : null}</div>
              {artifact.email || artifact.automation || artifact.segment || artifact.post ? <ArtifactDisclosure artifact={artifact} /> : null}
              {artifact.resultId ? <div className="mt-3"><ArtifactHandoff artifact={artifact} /></div> : null}
              {owner && artifact.kind !== 'segment' && (artifact.decision === 'kept' || artifact.decision === 'accepted') ? <Button className="mt-3" disabledReason={missingPermission ? deniedReason(missingPermission) : undefined} disabled={blocked || mutations.busy || !['review', 'completed', 'paused'].includes(run.state) || !permissions.includes('ai:use')} onClick={() => setSource({ action: 'propose', artifact, version: run.version, planRevision: run.planRevision })}>{t(ag.propose)}</Button> : null}
              {artifact.id === conflict ? <div role="alert" className="note w mt-3"><p className="m-0">{t(ag.conflict)}</p>{owner ? <Button className="mt-3" disabled={disabled || !permissions.includes('ai:use')} onClick={() => setSource({ action: 'refresh', artifact, version: run.version, planRevision: run.planRevision })}>{t(ag.refresh)}</Button> : null}</div> : null}
            </article>)}</div>
            {undecided.length ? <div className="flex flex-col gap-3 border-t border-line-2 px-5 py-4 max-md:px-4">
              {section === 'new' && !complete ? <p className="k m-0">{t(ag.incomplete)}</p> : null}
              <div className="flex flex-wrap gap-2">{(section === 'new' ? ['kept', 'discarded'] : ['accepted', 'rejected']).map((key) => {
                const decision = key as Decision
                return <Button key={decision} variant={decision === 'kept' || decision === 'accepted' ? 'primary' : decision === 'discarded' ? 'danger' : 'default'} disabledReason={missingPermission ? deniedReason(missingPermission) : undefined} disabled={inactive || (decision === 'kept' && !complete) || (decision === 'accepted' && group.some((a) => a.id === conflict))} onClick={() => setConfirm({ decision, version: run.version, artifacts: undecided })}>{t(decisions[decision])}</Button>
              })}</div>
            </div> : null}
          </Card>
        })}
      </section>
    })}
    {confirm ? <ConfirmDialog open onOpenChange={(open) => { if (!open) { setConfirm(undefined); settle() } }} tone={confirm.decision === 'discarded' ? 'bad' : 'warn'} badge={t(ag.review)} title={t(decisions[confirm.decision])} description={t(descriptions[confirm.decision])} confirmLabel={t(decisions[confirm.decision])} onConfirm={decide}>
      <ul className="m-0 pl-5">{confirm.artifacts.map((a) => <li key={a.id}>{a.name} · {t(ag.revision, { revision: a.revision })}</li>)}</ul>
    </ConfirmDialog> : null}
    {source ? <ConfirmDialog open onOpenChange={(open) => { if (!open) { setSource(undefined); settle() } }} tone="warn" badge={t(ag.plan)} title={t(source.action === 'refresh' ? ag.refresh : ag.propose)} description={t(source.action === 'refresh' ? ag.refreshBody : ag.proposeBody)} confirmLabel={t(source.action === 'refresh' ? ag.refresh : ag.propose)} reason={{ label: t(ag.instruction), hint: t(ag.conversationHint), minLength: 1 }} onConfirm={async ({ reason }) => {
      await mutations.action({ sharingReviewed: true, action: source.action, version: source.version, planRevision: source.planRevision, handle: source.artifact.handle, message: reason })
      onPlanRequested?.()
    }}><div className="flex flex-col gap-3"><p className="m-0 text-sm">{t(md.sharingDisclosure, { recipient: run.model })}</p><p className="m-0 text-sm">{t(md.sharingRisk)}</p><p className="k m-0">{t(md.providerPolicy)}</p></div></ConfirmDialog> : null}
  </div>
}

function ArtifactDisclosure({ artifact }: { artifact: AgentArtifact }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  return <details className="mt-2" onToggle={(event) => setOpen(event.currentTarget.open)}><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ag.preview)}</summary>{open ? <ArtifactPreview artifact={artifact} /> : null}</details>
}

function ArtifactPreview({ artifact: a }: { artifact: AgentArtifact }) {
  const t = useT()
  const afterLabel = a.decision ? `${t(ag.revision, { revision: a.revision })} · ${t(ag.decision, { decision: a.decision })}` : undefined
  if (a.email) return <DraftEmailReview content={a.email} before={a.before} html={a.html ?? ''} afterLabel={afterLabel} />
  if (a.automation) return <div className="flex flex-col gap-4">
    <div className={`grid min-w-0 gap-4 ${a.beforeAutomation ? 'grid-cols-2 max-md:grid-cols-1' : ''}`}>
      {a.beforeAutomation ? <GraphReview before graph={a.beforeAutomation} refs={{ ...triggerRefs(a.beforeAutomation, a.triggerName), emails: new Map((a.beforeEmails ?? []).map((e) => [e.id, { name: e.name, latestVersion: e.version, subject: e.content.subject }])) }} /> : null}
      <GraphReview graph={a.automation.graph} refs={{ ...triggerRefs(a.automation.graph, a.triggerName), emails: new Map(a.automation.emails.map((e) => [e.key, { name: e.name, latestVersion: 1, subject: e.subject }])) }} afterLabel={afterLabel} />
    </div>
    {a.automation.emails.map((email, i) => <details key={email.key}><summary className="min-h-11 cursor-pointer leading-[44px]">{email.name}</summary><DraftEmailReview content={email} before={a.beforeEmails?.find((e) => e.key === email.key)?.content} html={a.previews?.[i] ?? ''} afterLabel={afterLabel} /></details>)}
  </div>
  if (a.post) return <DraftEmailReview content={{ subject: a.post.title, preheader: a.post.excerpt, mjml: a.post.mjml }} before={a.before} html={a.html ?? ''} labels={{ subject: t(postMessages.title), preheader: t(postMessages.excerpt) }} afterLabel={afterLabel} />
  if (a.segment) return <SegmentPreview definition={a.segment} />
  return null
}

function triggerRefs(graph: FlowGraph, name?: string): Refs {
  const trigger = graph.nodes.find((node) => node.type === 'trigger')?.data.trigger
  return {
    ...emptyRefs,
    lists: new Map(name && trigger?.listId ? [[trigger.listId, name]] : []),
    segments: new Map(name && trigger?.segmentId ? [[trigger.segmentId, name]] : []),
  }
}

export function SegmentPreview({ definition }: { definition: Schemas['SegmentDefinition'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments/catalog', { params: { path: { workspace: slug } } })
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} />
  if (!q.data) return <Skeleton h={100} />
  return <div className="flex flex-col gap-3"><p className="k m-0">{t(segmentMessages.combine)}</p>{definition.groups.map((group, index) => <section key={index} className="rounded-lg border border-line-2 p-3" aria-label={t(segmentMessages.group, { n: index + 1 })}>
    <h4 className="m-0 text-sm">{t(group.match === 'all' ? segmentMessages.matchAll : segmentMessages.matchAny)}</h4>
    <ul className="mb-0 flex list-none flex-col gap-3 p-0">{group.conditions.map((condition, i) => {
      const field = q.data.fields.find((item) => item.key === condition.field)
      const operator = field?.ops.find((item) => item.key === condition.op)?.label ?? condition.op
      const value = field?.options.find((item) => item.value === condition.value)?.label ?? condition.value
      const noValue = ['never', 'ever', 'is_empty', 'is_not_empty', 'is_true', 'is_false'].includes(condition.op)
      return <li key={i} className="agent-source flex flex-col gap-1 text-[13px]"><strong>{field?.label ?? condition.field}</strong><span>{operator}</span>{!noValue ? <span>{field?.kind === 'period' ? <>{t(segmentMessages.period, { amount: count(condition.amount ?? 0), unit: condition.unit })}{value ? ` · ${value}` : ''}</> : condition.op === 'between' ? <>{value} – {condition.to}</> : value}</span> : null}</li>
    })}</ul>
  </section>)}</div>
}

function ArtifactHandoff({ artifact: a }: { artifact: AgentArtifact }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (!a.resultId) return null
  switch (a.kind) {
    case 'campaign': return <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: a.resultId }}>{t(ag.composer)}</ButtonLink>
    case 'automation': return <ButtonLink to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: a.resultId }}>{t(ag.flowEditor)}</ButtonLink>
    case 'segment': return <ButtonLink to="/w/$workspace/audience/segments/$segmentId" params={{ workspace: slug, segmentId: a.resultId }}>{t(ag.segmentEditor)}</ButtonLink>
    case 'post': return <ButtonLink to="/w/$workspace/forms/website/posts/$post" params={{ workspace: slug, post: a.resultId }}>{t(ag.postEditor)}</ButtonLink>
  }
}
