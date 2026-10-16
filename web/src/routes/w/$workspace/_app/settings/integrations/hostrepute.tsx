import { createFileRoute } from '@tanstack/react-router'
import { IntegrationsPage } from '@/features/hostrepute/IntegrationsPage'

/** Settings › Integrations › HostRepute: the connected view, moved under the catalog. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/hostrepute')({
  component: IntegrationsPage,
})
