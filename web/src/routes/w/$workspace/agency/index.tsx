import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { OverviewPage } from '@/features/agency/OverviewPage'

const search = z.object({
  // ?new=1 opens New client (from the workspace switcher).
  new: z.coerce.number().int().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/agency/')({
  validateSearch: search,
  component: OverviewPage,
})
