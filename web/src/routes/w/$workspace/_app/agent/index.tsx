import { createFileRoute } from '@tanstack/react-router'
import { AgentRunsPage } from '@/features/ai/AgentRunsPage'

export const Route = createFileRoute('/w/$workspace/_app/agent/')({ component: AgentRunsPage })
