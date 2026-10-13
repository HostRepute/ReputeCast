import { createFileRoute } from '@tanstack/react-router'
import { SubscribePageTab } from '@/features/forms/PageTabs'

export const Route = createFileRoute('/w/$workspace/_app/forms/subscribe')({
  component: SubscribePageTab,
})
