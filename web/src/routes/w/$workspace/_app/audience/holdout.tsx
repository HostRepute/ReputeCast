import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { HoldoutPage } from '@/features/holdout/HoldoutPage'

const search = z.object({
  window: z.enum(['period', '30d', '90d']).optional().catch(undefined),
  period: z.string().uuid().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/holdout')({
  validateSearch: search,
  component: function HoldoutRoute() {
    return <HoldoutPage search={Route.useSearch()} />
  },
})
