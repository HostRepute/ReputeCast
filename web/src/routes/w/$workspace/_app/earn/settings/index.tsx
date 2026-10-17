import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { SettingsPage, type SettingsTab } from '@/features/earn/SettingsPage'

const search = z.object({
  tab: z.enum(['payments', 'emails', 'access', 'attribution', 'pages']).optional().catch(undefined),
})

/** Earn › Settings: Payments · Member emails · Access · Attribution · Pages. */
export const Route = createFileRoute('/w/$workspace/_app/earn/settings/')({
  validateSearch: search,
  component: function EarnSettingsRoute() {
    const { tab } = Route.useSearch()
    const navigate = Route.useNavigate()
    const onTab = useCallback((next: SettingsTab) => void navigate({ search: { tab: next === 'payments' ? undefined : next }, replace: true }), [navigate])
    return <SettingsPage tab={tab ?? 'payments'} onTab={onTab} />
  },
})
