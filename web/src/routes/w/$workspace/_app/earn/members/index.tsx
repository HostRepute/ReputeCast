import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { MembersPage, type MembersSearch } from '@/features/earn/MembersPage'

const search = z.object({
  status: z.enum(['with_access', 'paying', 'trialing', 'past_due', 'cancels_soon', 'complimentary', 'gift', 'ended']).optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  tier: z.string().max(64).optional().catch(undefined),
  period: z.enum(['month', 'year']).optional().catch(undefined),
  source: z.enum(['campaign', 'automation', 'page', 'offer', 'website', 'import', 'gift', 'grant', 'direct']).optional().catch(undefined),
  campaign: z.uuid().optional().catch(undefined),
  offer: z.string().max(64).optional().catch(undefined),
  since: z.enum(['30d', '90d']).optional().catch(undefined),
  before: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/)
    .optional()
    .catch(undefined),
  test: z.boolean().optional().catch(undefined),
  page: z.number().int().min(1).max(100000).optional().catch(undefined),
  /** The paid member shown in the panel. */
  member: z.uuid().optional().catch(undefined),
  /** The import dialog: "new" or an import's id. */
  import: z.union([z.literal('new'), z.uuid()]).optional().catch(undefined),
  /** The Give complimentary access dialog. */
  comp: z.boolean().optional().catch(undefined),
})

/** Earn › Paid members (EarnMembers). */
export const Route = createFileRoute('/w/$workspace/_app/earn/members/')({
  validateSearch: search,
  component: function MembersRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: MembersSearch, replace?: boolean) => void navigate({ search: next as z.infer<typeof search>, replace }), [navigate])
    return <MembersPage search={s} onSearch={onSearch} />
  },
})
