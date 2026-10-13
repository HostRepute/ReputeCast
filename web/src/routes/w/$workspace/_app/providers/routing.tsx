import { createFileRoute } from '@tanstack/react-router'
import { RoutingPage } from '@/features/providers/RoutingPage'

export const Route = createFileRoute('/w/$workspace/_app/providers/routing')({
  component: RoutingPage,
})
