import { createFileRoute } from '@tanstack/react-router'
import { InstancePage } from '@/features/admin/instance/InstancePage'

export const Route = createFileRoute('/w/$workspace/_app/admin/instance')({
  component: InstancePage,
})
