import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ResultsPage } from '@/features/grow/ResultsPage'

const search = z.object({
  program: z.uuid().optional().catch(undefined),
  period: z.enum(['7d', '30d', 'all']).optional().catch(undefined),
  grouping: z.enum(['day', 'week']).optional().catch(undefined),
})

/** Grow › Referrals › Results: /grow/referrals?period=30d&grouping=day */
export const Route = createFileRoute('/w/$workspace/_app/grow/referrals/')({
  validateSearch: search,
  component: function ReferralsRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <ResultsPage search={s} onSearch={(next) => void navigate({ search: next, replace: true })} />
  },
})
