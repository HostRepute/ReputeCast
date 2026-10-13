import { createFileRoute } from '@tanstack/react-router'
import { ProvidersSettingsPage } from '@/features/senders/ProvidersSettingsPage'

/** 7.13 Settings › Providers: provider summaries and workspace-wide sending defaults. */
export const Route = createFileRoute('/w/$workspace/_app/settings/providers')({
  component: ProvidersSettingsPage,
})
