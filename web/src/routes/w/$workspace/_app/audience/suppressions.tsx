import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SuppressionsPage } from '@/features/audience/SuppressionsPage'
// Exports show in the jobs tray beside imports.
import '@/features/imports/jobsSource'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  reason: z.enum(['hard_bounce', 'complaint', 'unsubscribe', 'manual']).optional().catch(undefined),
  source: z.string().max(60).optional().catch(undefined),
  since: z.enum(['7d', '30d']).optional().catch(undefined),
  import: z.string().uuid().optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/suppressions')({
  validateSearch: search,
  component: function SuppressionsRoute() {
    return <SuppressionsPage search={Route.useSearch()} />
  },
})
