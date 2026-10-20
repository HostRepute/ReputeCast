import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { MilestonesPage } from '@/features/grow/MilestonesPage'

const search = z.object({
  program: z.uuid().optional().catch(undefined),
  /** The milestone whose editor is open ("new" for a new one). */
  edit: z.string().max(40).optional().catch(undefined),
})

/** Grow › Referrals › Milestones: /grow/referrals/milestones?edit=… */
export const Route = createFileRoute('/w/$workspace/_app/grow/referrals/milestones')({
  validateSearch: search,
  component: function MilestonesRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <MilestonesPage search={s} onSearch={(next) => void navigate({ search: next, replace: true })} />
  },
})
