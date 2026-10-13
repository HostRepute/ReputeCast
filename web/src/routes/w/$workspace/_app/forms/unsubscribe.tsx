import { createFileRoute } from '@tanstack/react-router'
import { UnsubscribeTab } from '@/features/forms/PageTabs'

export const Route = createFileRoute('/w/$workspace/_app/forms/unsubscribe')({
  component: UnsubscribeTab,
})
