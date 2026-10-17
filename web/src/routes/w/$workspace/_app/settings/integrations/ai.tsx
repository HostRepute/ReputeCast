import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AISettingsPage } from '@/features/ai/SettingsPage'

/** ?connect opens the connect dialog on a provider (the catalog's Connect). */
const search = z.object({
  connect: z.enum(['anthropic', 'openai', 'google', 'azure', 'compatible']).optional().catch(undefined),
})

/** Settings › Integrations › AI assistant: the workspace's own model. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/ai')({
  validateSearch: search,
  component: function AIRoute() {
    return <AISettingsPage connect={Route.useSearch().connect} />
  },
})
