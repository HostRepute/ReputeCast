import { createFileRoute } from '@tanstack/react-router'
import { EarnOverviewPage } from '@/features/earn/Overview'
import { EarnHome } from '@/features/earn/revenue/EarnHome'
import { RevenueOverview } from '@/features/earn/revenue/RevenueOverview'

/**
 * Earn › Overview: the launch checklist until Earn is live (the six steps
 * done, the live key among them), then the revenue overview. A workspace
 * that records conversions or sells sponsorships without a payment
 * provider sees those instead of the checklist.
 */
export const Route = createFileRoute('/w/$workspace/_app/earn/')({
  component: function EarnRoute() {
    return <EarnHome checklist={<EarnOverviewPage launched={<RevenueOverview />} />} />
  },
})
