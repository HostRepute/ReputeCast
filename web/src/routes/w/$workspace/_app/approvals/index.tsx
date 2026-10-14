import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { InboxPage } from '@/features/approvals/InboxPage'

const search = z.object({
  status: z.enum(['pending', 'approved', 'changes_requested', 'rejected', 'invalidated', 'all']).optional().catch(undefined),
  author: z.string().uuid().optional().catch(undefined),
  approver: z.string().uuid().optional().catch(undefined),
  days: z.coerce.number().int().min(0).max(3650).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/approvals/')({
  validateSearch: search,
  component: function ApprovalsRoute() {
    return <InboxPage search={Route.useSearch()} />
  },
})
