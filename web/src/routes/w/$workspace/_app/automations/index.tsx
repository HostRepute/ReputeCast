import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AutomationsPage } from '@/features/automations/AutomationsPage'

const search = z.object({
  status: z.enum(['all', 'live', 'paused', 'drafts']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/automations/')({
  validateSearch: search,
  component: function AutomationsRoute() {
    return <AutomationsPage status={Route.useSearch().status ?? 'all'} />
  },
})
