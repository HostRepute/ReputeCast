import { createFileRoute } from '@tanstack/react-router'
import { ArchiveTab } from '@/features/forms/PageTabs'

export const Route = createFileRoute('/w/$workspace/_app/forms/archive')({
  component: ArchiveTab,
})
