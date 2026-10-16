import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { FlowEditorPage, NewAutomationPage, type EditorSearch } from '@/features/automations/FlowEditor'

const search = z.object({
  step: z.string().max(64).optional().catch(undefined),
  panel: z.enum(['step', 'stats']).optional().catch(undefined),
  template: z.enum(['welcome-series', 're-engagement', 'onboarding']).optional().catch(undefined),
  /** The list a list-triggered template starts on (the welcome-series prompt). */
  list: z.uuid().optional().catch(undefined),
})

/** The full-screen flow editor; /automations/new?template=… creates a draft first. */
export const Route = createFileRoute('/w/$workspace/automations/$automationId')({
  validateSearch: search,
  component: AutomationEditorRoute,
})

function AutomationEditorRoute() {
  const { automationId } = Route.useParams()
  const s = Route.useSearch()
  const navigate = Route.useNavigate()
  const onSearch = useCallback((next: EditorSearch) => void navigate({ search: { step: next.step, panel: next.panel }, replace: true, resetScroll: false }), [navigate])
  if (automationId === 'new') return <NewAutomationPage template={s.template} list={s.list} />
  return <FlowEditorPage key={automationId} automationId={automationId} search={{ step: s.step, panel: s.panel }} onSearch={onSearch} />
}
