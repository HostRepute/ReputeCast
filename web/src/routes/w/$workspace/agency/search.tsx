import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SearchPage } from '@/features/agency/SearchPage'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/agency/search')({
  validateSearch: search,
  component: SearchPage,
})
