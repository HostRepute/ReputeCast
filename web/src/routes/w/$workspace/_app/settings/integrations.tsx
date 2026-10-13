import { createFileRoute } from '@tanstack/react-router'
import { IntegrationsPage } from '@/features/hostrepute/IntegrationsPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/integrations')({
  component: IntegrationsPage,
})
