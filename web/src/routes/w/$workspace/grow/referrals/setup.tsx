import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SetupGuide } from '@/features/grow/SetupGuide'

const search = z.object({
  /** The program being set up: a reload resumes its saved step. */
  program: z.uuid().optional().catch(undefined),
  step: z.number().int().min(1).max(6).optional().catch(undefined),
})

/** Set up a referral program (ReferralSetup): a guide in five steps, outside the app shell. */
export const Route = createFileRoute('/w/$workspace/grow/referrals/setup')({
  validateSearch: search,
  component: function ReferralSetupRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <SetupGuide search={s} onSearch={(next) => void navigate({ search: next, replace: true })} />
  },
})
