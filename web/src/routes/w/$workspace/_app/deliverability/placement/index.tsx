import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TestsPage } from '@/features/placement/TestsPage'

const id = z.string().max(64).optional().catch(undefined)

const search = z.object({
  /** Opens Run a placement test with what the entry point knows. */
  run: z.enum(['placement', 'review', 'composer', 'warmup', 'template']).optional().catch(undefined),
  kind: z.enum(['campaign', 'template', 'transactional']).optional().catch(undefined),
  content: id,
  through: z.enum(['route', 'provider', 'compare']).optional().catch(undefined),
  provider: id,
  from: z.string().max(200).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/deliverability/placement/')({
  validateSearch: search,
  component: function TestsRoute() {
    const s = Route.useSearch()
    const preset = s.run
      ? { origin: s.run, content: s.kind && s.content ? { kind: s.kind, id: s.content } : undefined, through: s.through, providers: s.provider ? [s.provider] : undefined, from: s.from }
      : undefined
    return <TestsPage preset={preset} />
  },
})
