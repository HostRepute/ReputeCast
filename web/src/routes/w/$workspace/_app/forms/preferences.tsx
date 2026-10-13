import { createFileRoute } from '@tanstack/react-router'
import { PreferencesTab } from '@/features/forms/PageTabs'

export const Route = createFileRoute('/w/$workspace/_app/forms/preferences')({
  component: PreferencesTab,
})
