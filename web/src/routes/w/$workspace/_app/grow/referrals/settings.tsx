import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SettingsPage } from '@/features/grow/SettingsPage'

const search = z.object({
  program: z.uuid().optional().catch(undefined),
})

/** Grow › Referrals › Settings. */
export const Route = createFileRoute('/w/$workspace/_app/grow/referrals/settings')({
  validateSearch: search,
  component: function ReferralSettingsRoute() {
    return <SettingsPage search={Route.useSearch()} />
  },
})
