import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { PoolsPage } from '@/features/coupons/PoolsPage'

const search = z.object({
  status: z.enum(['active', 'archived']).optional().catch(undefined),
  // ?new=1 opens New pool (the composer's "Manage discount codes" when no pool exists).
  new: z.coerce.number().int().optional().catch(undefined),
})

// M59: Campaigns › Discount codes (DiscountCodes, DiscountCodesStates).
export const Route = createFileRoute('/w/$workspace/_app/campaigns/codes/')({
  validateSearch: search,
  component: function DiscountCodesRoute() {
    const s = Route.useSearch()
    return <PoolsPage status={s.status ?? 'active'} creating={s.new === 1} />
  },
})
