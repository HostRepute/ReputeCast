import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ReviewPage } from '@/features/grow/ReviewPage'

const search = z.object({
  program: z.uuid().optional().catch(undefined),
  signal: z.enum(['burst', 'shared_ip', 'numbered']).optional().catch(undefined),
  referrer: z.uuid().optional().catch(undefined),
})

/** Grow › Referrals › Review: /grow/referrals/review?signal=burst */
export const Route = createFileRoute('/w/$workspace/_app/grow/referrals/review')({
  validateSearch: search,
  component: function ReviewRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <ReviewPage search={s} onSearch={(next) => void navigate({ search: next, replace: true })} />
  },
})
