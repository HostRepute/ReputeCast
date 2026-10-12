import { createFileRoute } from '@tanstack/react-router'
import { AppShell } from '@/components/shell/AppShell'

/** Screens with the sidebar (everything except wizards and full-screen editors). */
export const Route = createFileRoute('/w/$workspace/_app')({
  component: AppShell,
})
