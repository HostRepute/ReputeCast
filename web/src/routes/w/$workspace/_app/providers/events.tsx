import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { EventsPage } from '@/features/providers/EventsPage'

const search = z.object({
  provider: z.string().uuid().optional().catch(undefined),
  type: z.enum(['delivered', 'deferred', 'hard_bounce', 'soft_bounce', 'complaint', 'rejected']).optional().catch(undefined),
  q: z.string().max(254).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/providers/events')({
  validateSearch: search,
  component: function EventsRoute() {
    return <EventsPage search={Route.useSearch()} />
  },
})
