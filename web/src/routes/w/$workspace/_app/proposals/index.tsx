import { createFileRoute } from '@tanstack/react-router'
import { ProposalsPage } from '@/features/mcpconnect/ProposalsPage'

export const Route = createFileRoute('/w/$workspace/_app/proposals/')({ component: ProposalsPage })
