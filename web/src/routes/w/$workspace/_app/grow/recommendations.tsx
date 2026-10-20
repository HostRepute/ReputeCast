import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { RecommendationsPage } from '@/features/grow/RecommendationsPage'

const search = z.object({
  /** Opens "Add a recommendation". */
  add: z.boolean().optional().catch(undefined),
})

/** Grow › Recommendations: you recommend, recommending you, where they show and the partner profile. */
export const Route = createFileRoute('/w/$workspace/_app/grow/recommendations')({
  validateSearch: search,
  component: function RecommendationsRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <RecommendationsPage search={s} onSearch={(next) => void navigate({ search: next, replace: true })} />
  },
})
