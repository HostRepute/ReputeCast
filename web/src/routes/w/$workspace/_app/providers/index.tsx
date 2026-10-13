import { createFileRoute } from '@tanstack/react-router'
import { ProvidersPage } from '@/features/providers/ProvidersPage'

export const Route = createFileRoute('/w/$workspace/_app/providers/')({
  component: ProvidersPage,
})
