import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { RoleEditorPage } from '@/features/roles/RoleEditorPage'

const search = z.object({
  /** Duplicate this role into a new custom role. */
  from: z.string().max(64).optional().catch(undefined),
  /** Set after a save, to show the confirmation. */
  saved: z.enum(['created', 'updated']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/settings/roles/$roleId')({
  validateSearch: search,
  component: function RoleEditorRoute() {
    const { roleId } = Route.useParams()
    return <RoleEditorPage roleId={roleId} search={Route.useSearch()} />
  },
})
