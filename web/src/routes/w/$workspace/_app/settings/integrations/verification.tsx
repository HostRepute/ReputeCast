import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ServicesPage } from '@/features/verification/ServicesPage'

const search = z.object({
  /** The service shown below the cards. */
  service: z.enum(['zerobounce', 'kickbox', 'neverbounce', 'bouncer', 'emailable']).optional().catch(undefined),
  /** Opens Connect for that service. */
  connect: z.boolean().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/verification')({
  validateSearch: search,
  component: function VerificationRoute() {
    return <ServicesPage search={Route.useSearch()} />
  },
})
