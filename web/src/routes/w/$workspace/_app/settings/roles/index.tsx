import { createFileRoute } from '@tanstack/react-router'
import { RolesPage } from '@/features/roles/RolesPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/roles/')({
  component: RolesPage,
})
