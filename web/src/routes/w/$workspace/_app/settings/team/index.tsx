import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TeamPage } from '@/features/team/TeamPage'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  role: z.string().max(64).optional().catch(undefined),
  status: z.enum(['active', 'invited', 'deactivated']).optional().catch(undefined),
  twofa: z.enum(['on', 'off', 'idp', 'required']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/settings/team/')({
  validateSearch: search,
  component: function TeamRoute() {
    return <TeamPage search={Route.useSearch()} />
  },
})
