import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ResultPage } from '@/features/placement/ResultPage'

const search = z.object({
  /** The copy shown in the detail panel. */
  copy: z.string().max(64).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/deliverability/placement/$testId')({
  validateSearch: search,
  component: function ResultRoute() {
    const { testId } = Route.useParams()
    return <ResultPage testId={testId} copy={Route.useSearch().copy} />
  },
})
