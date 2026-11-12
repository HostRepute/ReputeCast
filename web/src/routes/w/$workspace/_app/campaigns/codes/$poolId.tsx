import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { PoolPage } from '@/features/coupons/PoolPage'

const search = z.object({
  // Send Readiness's fixes: ?add=1 opens Add codes, ?create=N Create more with N.
  add: z.coerce.number().int().optional().catch(undefined),
  create: z.coerce.number().int().min(1).max(50000).optional().catch(undefined),
})

// M59: a discount code pool (DiscountCodePool, DiscountCodesStates).
export const Route = createFileRoute('/w/$workspace/_app/campaigns/codes/$poolId')({
  validateSearch: search,
  component: function DiscountCodePoolRoute() {
    const { poolId } = Route.useParams()
    const s = Route.useSearch()
    return <PoolPage id={poolId} adding={s.add === 1} create={s.create} />
  },
})
