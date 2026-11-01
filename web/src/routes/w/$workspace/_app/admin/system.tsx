import { createFileRoute } from '@tanstack/react-router'
import { SystemPage } from '@/features/admin/system/SystemPage'

export const Route = createFileRoute('/w/$workspace/_app/admin/system')({
  component: SystemPage,
})
