import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ReferrersPage } from '@/features/grow/ReferrersPage'

const search = z.object({
  program: z.uuid().optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  /** 0 any, -1 none yet, n reached n. */
  milestone: z.number().int().min(-1).max(100000).optional().catch(undefined),
  held: z.boolean().optional().catch(undefined),
  paused: z.boolean().optional().catch(undefined),
  page: z.number().int().min(1).optional().catch(undefined),
  /** The referrer whose panel is open (their link). */
  referrer: z.uuid().optional().catch(undefined),
})

/** Grow › Referrals › Referrers: /grow/referrals/referrers?referrer=… */
export const Route = createFileRoute('/w/$workspace/_app/grow/referrals/referrers')({
  validateSearch: search,
  component: function ReferrersRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <ReferrersPage search={s} onSearch={(next, replace = true) => void navigate({ search: next, replace })} />
  },
})
