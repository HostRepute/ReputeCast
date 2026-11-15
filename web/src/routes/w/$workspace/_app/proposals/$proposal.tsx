import { createFileRoute } from '@tanstack/react-router'
import { ProposalPage } from '@/features/mcpconnect/ProposalPage'

export const Route = createFileRoute('/w/$workspace/_app/proposals/$proposal')({
  component: function ProposalRoute() {
    const { proposal } = Route.useParams()
    return <ProposalPage key={proposal} id={proposal} />
  },
})
