import { createFileRoute } from '@tanstack/react-router'
import { WebsiteTab } from '@/features/website/WebsiteTab'

export const Route = createFileRoute('/w/$workspace/_app/forms/website/')({
  component: WebsiteTab,
})
