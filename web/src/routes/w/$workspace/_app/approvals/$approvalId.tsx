import { createFileRoute } from '@tanstack/react-router'
import { ApprovalPage } from '@/features/approvals/ApprovalPage'

export const Route = createFileRoute('/w/$workspace/_app/approvals/$approvalId')({
  component: function ApprovalRoute() {
    const { approvalId } = Route.useParams()
    return <ApprovalPage key={approvalId} approvalId={approvalId} />
  },
})
