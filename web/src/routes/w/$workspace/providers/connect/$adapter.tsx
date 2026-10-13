import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { SetupWizard, type WizardSearch } from '@/features/connect/SetupWizard'

const search = z.object({
  setup: z.uuid().optional().catch(undefined),
  step: z.enum(['connection', 'domain', 'test', 'streams', 'contacts', 'done']).optional().catch(undefined),
})

/** 7.2 steps 2–6 and the summary: /providers/connect/ses?setup=…&step=domain */
export const Route = createFileRoute('/w/$workspace/providers/connect/$adapter')({
  validateSearch: search,
  component: ConnectAdapterRoute,
})

function ConnectAdapterRoute() {
  const { adapter } = Route.useParams()
  const s = Route.useSearch()
  const navigate = Route.useNavigate()
  const onSearch = useCallback((next: WizardSearch, replace?: boolean) => void navigate({ search: { setup: next.setup, step: next.step }, replace }), [navigate])
  return <SetupWizard adapter={adapter} search={s} onSearch={onSearch} />
}
