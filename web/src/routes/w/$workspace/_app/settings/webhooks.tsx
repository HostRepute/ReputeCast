import { createFileRoute } from '@tanstack/react-router'
import { WebhooksPage } from '@/features/webhooks/WebhooksPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/webhooks')({
  component: WebhooksPage,
})
