import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { GrowthPage } from '@/features/growth/GrowthPage'

const day = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .optional()
  .catch(undefined)

const search = z.object({
  period: z.enum(['30d', '90d', '12m', 'custom']).optional().catch(undefined),
  from: day,
  to: day,
  list: z.string().uuid().optional().catch(undefined),
  imports: z.boolean().optional().catch(undefined),
  cohort: z.string().max(200).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/growth')({
  validateSearch: search,
  component: function GrowthRoute() {
    return <GrowthPage search={Route.useSearch()} />
  },
})
