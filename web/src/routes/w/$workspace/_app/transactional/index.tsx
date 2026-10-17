import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TransactionalOverviewPage } from '@/features/transactional/OverviewPage'

const search = z.object({
  period: z.enum(['24h', '7d', '30d']).optional().catch(undefined),
})

/** Transactional › Templates: the period's numbers, the route, the templates and the keys that can send. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/')({
  validateSearch: search,
  component: function TransactionalRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <TransactionalOverviewPage
        period={s.period ?? '7d'}
        onPeriod={(p) => void navigate({ search: (prev) => ({ ...prev, period: p === '7d' ? undefined : p }), replace: true })}
      />
    )
  },
})
