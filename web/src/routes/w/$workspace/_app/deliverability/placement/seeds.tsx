import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SeedsPage } from '@/features/placement/SeedsPage'

const id = z.string().max(64).optional().catch(undefined)

const search = z.object({
  /** Opens Add a seed mailbox. */
  add: z.boolean().optional().catch(undefined),
  /** Opens Reconnect for this seed. */
  reconnect: id,
  /** Back from Google's or Microsoft's consent. */
  oauth: z.enum(['cancelled', 'failed', 'forbidden']).optional().catch(undefined),
  reason: z.string().max(300).optional().catch(undefined),
  provider: z.string().max(32).optional().catch(undefined),
  added: id,
})

export const Route = createFileRoute('/w/$workspace/_app/deliverability/placement/seeds')({
  validateSearch: search,
  component: function SeedsRoute() {
    return <SeedsPage search={Route.useSearch()} />
  },
})
