import { createFileRoute } from '@tanstack/react-router'
import { SSOProvidersPage } from '@/features/admin/sso/SSOProvidersPage'

export const Route = createFileRoute('/w/$workspace/_app/admin/sso/')({
  component: SSOProvidersPage,
})
