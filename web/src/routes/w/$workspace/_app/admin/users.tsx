import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { UsersPage } from '@/features/admin/users/UsersPage'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  workspace: z.string().max(64).optional().catch(undefined),
  status: z.enum(['active', 'deactivated']).optional().catch(undefined),
  twofa: z.enum(['on', 'off', 'idp']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/admin/users')({
  validateSearch: search,
  component: function AdminUsersRoute() {
    return <UsersPage search={Route.useSearch()} />
  },
})
