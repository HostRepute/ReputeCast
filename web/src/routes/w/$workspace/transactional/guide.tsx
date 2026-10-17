import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TransactionalGuidePage, type GuideStep } from '@/features/transactional/GuidePage'

const search = z.object({
  /** The step open (1–5, or done); without it the guide opens at the first step not done. */
  step: z.union([z.coerce.number().int().min(1).max(5), z.literal('done')]).optional().catch(undefined),
})

/** The first-send guide, full screen like the provider setup wizard. */
export const Route = createFileRoute('/w/$workspace/transactional/guide')({
  validateSearch: search,
  component: function TransactionalGuideRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <TransactionalGuidePage step={s.step as GuideStep | undefined} onStep={(step) => void navigate({ search: { step } })} />
  },
})
