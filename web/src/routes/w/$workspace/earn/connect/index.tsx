import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { ConnectWizard, type ConnectSearch } from '@/features/earn/ConnectWizard'

const search = z.object({
  /** The connection being set up: a reload resumes it. */
  id: z.uuid().optional().catch(undefined),
  step: z.enum(['provider', 'key', 'check', 'webhook', 'settings', 'done']).optional().catch(undefined),
  /** The key asked for: Connect live key opens on live. */
  mode: z.enum(['live', 'test']).optional().catch(undefined),
})

/** Connect a payment provider (EarnConnect): /earn/connect?id=…&step=webhook */
export const Route = createFileRoute('/w/$workspace/earn/connect/')({
  validateSearch: search,
  component: function EarnConnectRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: ConnectSearch, replace?: boolean) => void navigate({ search: { id: next.id, step: next.step, mode: next.mode }, replace }), [navigate])
    return <ConnectWizard search={s} onSearch={onSearch} />
  },
})
