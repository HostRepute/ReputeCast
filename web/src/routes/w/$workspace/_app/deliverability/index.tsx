import { createFileRoute } from '@tanstack/react-router'
import { DeliverabilityPage } from '@/features/deliverability/DeliverabilityPage'

export const Route = createFileRoute('/w/$workspace/_app/deliverability/')({
  component: DeliverabilityPage,
})
