import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ConnectPage } from '@/features/hostrepute/ConnectPage'

/** ?result= and ?reason= come back from the OAuth callback; ?key focuses the API key. */
const search = z.object({
  result: z.enum(['connected', 'error']).optional().catch(undefined),
  reason: z.string().max(40).optional().catch(undefined),
  key: z.coerce.boolean().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/deliverability/connect')({
  validateSearch: search,
  component: function ConnectRoute() {
    return <ConnectPage search={Route.useSearch()} />
  },
})
