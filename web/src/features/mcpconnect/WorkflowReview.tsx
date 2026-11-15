import type { Schemas } from '@/api/client'
import { Note } from '@/components/ui/Note'
import { SegmentPreview } from '@/features/ai/AgentReview'
import { GraphReview } from '@/features/ai/AutomationFromBrief'
import { DraftEmailReview } from '@/features/ai/DraftFromBrief'
import { emptyRefs, type FlowGraph } from '@/features/automations/graph'
import { useT } from '@/i18n'
import { mp } from './proposals.messages'

export function WorkflowReview({ proposal: p }: { proposal: Schemas['MCPProposalView'] }) {
  const t = useT()
  const workflow = p.workflow
  if (!workflow) return null
  const afterLabel = p.decision ? t(mp.status, { decision: p.decision }) : undefined
  const refs = (graph: FlowGraph, emails: Schemas['ProposedEmail'][], before: boolean) => {
    const trigger = graph.nodes.find((n) => n.type === 'trigger')?.data.trigger
    const name = !before && workflow.segment ? workflow.segmentName : p.triggerName
    return {
      ...emptyRefs,
      lists: new Map(trigger?.listId && name ? [[trigger.listId, name]] : []),
      segments: new Map(trigger?.segmentId && name ? [[trigger.segmentId, name]] : []),
      emails: new Map(emails.map((e) => [before ? e.key.split(':')[0]! : e.key, { name: e.name, latestVersion: 1, subject: e.subject }])),
    }
  }
  return <div className="flex min-w-0 flex-col gap-4">
    {workflow.flow ? <Note>{t(workflow.segment ? mp.bundleBody : mp.flowBody)}</Note> : <Note>{t(mp.segmentBody)}</Note>}
    {workflow.segment ? <section aria-label={t(mp.kind, { kind: 'segment' })} className="flex min-w-0 flex-col gap-3">
      <h3 className="m-0 text-sm">{workflow.segmentName || p.name}</h3><SegmentPreview definition={workflow.segment} />
    </section> : null}
    {workflow.flow ? <>
      <div className={`grid min-w-0 gap-4 ${p.beforeFlow ? 'grid-cols-2 max-md:grid-cols-1' : ''}`}>
        {p.beforeFlow ? <GraphReview before graph={p.beforeFlow.graph} refs={refs(p.beforeFlow.graph, p.beforeFlow.emails, true)} /> : null}
        <GraphReview graph={workflow.flow.graph} refs={refs(workflow.flow.graph, workflow.flow.emails, false)} afterLabel={afterLabel} />
      </div>
      {workflow.flow.emails.map((email, i) => {
        // Source projection assigns stable email1… keys in saved-source order,
        // independent of the order in which the client returns revised emails.
        const before = p.beforeFlow?.emails.find((_, index) => email.key === `email${index + 1}`)
        return <details key={email.key} className="min-w-0 rounded-lg border border-line-2 p-3"><summary className="min-h-11 cursor-pointer py-3 text-sm font-semibold [overflow-wrap:anywhere]">{email.name}</summary>
          <DraftEmailReview content={email} before={before} html={workflow.previews[i] ?? ''} afterLabel={afterLabel} />
        </details>
      })}
    </> : null}
    <p className="k m-0">{t(mp.workflowChecks)}</p>
  </div>
}
