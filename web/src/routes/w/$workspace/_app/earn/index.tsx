import { createFileRoute } from '@tanstack/react-router'
import { m } from '@/components/shell/messages'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { Page } from '@/components/shell/Page'
import { useT } from '@/i18n'

// The sidebar's Earn section (phase 6, payments and paid subscriptions).
export const Route = createFileRoute('/w/$workspace/_app/earn/')({
  component: function EarnRoute() {
    const t = useT()
    return (
      <Page title={t(m.navEarn)}>
        <NotBuilt screen={t(m.navEarn)} />
      </Page>
    )
  },
})
