import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AgentBriefPage } from '@/features/ai/AgentBriefPage'

export const Route = createFileRoute('/w/$workspace/_app/agent/new')({
  validateSearch: z.object({ kind: z.enum(['campaign', 'automation', 'post']).optional().catch(undefined) }),
  component: function NewAgentRoute() { return <AgentBriefPage initialKind={Route.useSearch().kind} /> },
})
