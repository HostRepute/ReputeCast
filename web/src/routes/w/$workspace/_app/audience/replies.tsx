import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { RepliesPage } from '@/features/replies/RepliesPage'

const search = z.object({
  view: z.enum(['open', 'done', 'auto', 'spam', 'other', 'unsubscribes']).optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  campaign: z.string().uuid().optional().catch(undefined),
  automation: z.string().uuid().optional().catch(undefined),
  reply: z.string().uuid().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/replies')({
  validateSearch: search,
  component: function RepliesRoute() {
    return <RepliesPage search={Route.useSearch()} />
  },
})
