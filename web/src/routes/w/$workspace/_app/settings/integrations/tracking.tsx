import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TrackingPage } from '@/features/tracking/TrackingPage'

/** ?add opens "Add a site" (the catalog's Set up). */
const search = z.object({
  add: z.coerce.boolean().optional().catch(undefined),
})

/** Settings › Integrations › Website tracking: the sites the tracker runs on (M62). */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/tracking')({
  validateSearch: search,
  component: function TrackingRoute() {
    return <TrackingPage add={Route.useSearch().add} />
  },
})
