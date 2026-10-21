import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { SmsConnectWizard, type SmsWizardSearch } from '@/features/sms/Connect'

const search = z.object({
  setup: z.uuid().optional().catch(undefined),
  provider: z.uuid().optional().catch(undefined),
  step: z.enum(['connection', 'senders', 'test', 'countries']).optional().catch(undefined),
})

/** The SMS path, steps 2 to 5: /providers/connect/sms/twilio?setup=…&provider=…&step=senders */
export const Route = createFileRoute('/w/$workspace/providers/connect/sms/$adapter')({
  validateSearch: search,
  component: SmsConnectRoute,
})

function SmsConnectRoute() {
  const { adapter } = Route.useParams()
  const s = Route.useSearch()
  const navigate = Route.useNavigate()
  const onSearch = useCallback((next: SmsWizardSearch, replace?: boolean) => void navigate({ search: next, replace }), [navigate])
  return <SmsConnectWizard adapter={adapter} search={s} onSearch={onSearch} />
}
