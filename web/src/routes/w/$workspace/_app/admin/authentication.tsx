import { createFileRoute } from '@tanstack/react-router'
import { AuthenticationPage } from '@/features/admin/auth/AuthenticationPage'

export const Route = createFileRoute('/w/$workspace/_app/admin/authentication')({
  component: AuthenticationPage,
})
