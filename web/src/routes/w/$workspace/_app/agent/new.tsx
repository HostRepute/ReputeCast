import { createFileRoute, useRouterState } from '@tanstack/react-router'
import { z } from 'zod'
import { AgentBriefPage, handoffBrief } from '@/features/ai/AgentBriefPage'

export const Route = createFileRoute('/w/$workspace/_app/agent/new')({
  validateSearch: z.object({ kind: z.enum(['campaign', 'automation', 'post']).optional().catch(undefined) }),
  component: function NewAgentRoute() {
    const { workspace } = Route.useParams()
    // The assistant's agent-task card hands its brief over in the history state.
    const state = useRouterState({ select: (s) => s.location.state })
    return <AgentBriefPage initialKind={Route.useSearch().kind} initialBrief={handoffBrief(state, workspace)} />
  },
})
