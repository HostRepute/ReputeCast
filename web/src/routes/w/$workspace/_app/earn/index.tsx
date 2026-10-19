import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { me } from '@/features/earn/messages'
import { EarnOverviewPage } from '@/features/earn/Overview'
import { StoreRevenueCard } from '@/features/commerce/revenue'
import { useT } from '@/i18n'

/**
 * Earn › Overview: the launch checklist until Earn is live (the six steps
 * done, the live key among them), then the revenue overview.
 */
export const Route = createFileRoute('/w/$workspace/_app/earn/')({
  component: function EarnRoute() {
    const t = useT()
    // "Store revenue from email" (X20) until the revenue overview (X17) takes it.
    return (
      <EarnOverviewPage
        launched={
          <>
            <StoreRevenueCard />
            <NotBuilt screen={t(me.tabOverview)} />
          </>
        }
      />
    )
  },
})
