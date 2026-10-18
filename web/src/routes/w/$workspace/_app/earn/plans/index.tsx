import { createFileRoute } from '@tanstack/react-router'
import { PlansPage } from '@/features/earn/PlansPage'

/** Earn › Plans & offers: the tiers, the upgrade page and, with offers, the offers. */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/')({
  component: PlansPage,
})
