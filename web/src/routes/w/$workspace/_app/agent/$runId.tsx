import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AgentRunPage } from '@/features/ai/AgentRunPage'

export const Route = createFileRoute('/w/$workspace/_app/agent/$runId')({
  validateSearch: z.object({ tab: z.enum(['plan', 'steps', 'review']).optional().catch(undefined) }),
  component: function AgentRunRoute() {
    const navigate = Route.useNavigate()
    return <AgentRunPage key={Route.useParams().runId} id={Route.useParams().runId} selectedTab={Route.useSearch().tab} onTabChange={(tab) => { void navigate({ search: { tab }, replace: true, resetScroll: false }) }} />
  },
})
