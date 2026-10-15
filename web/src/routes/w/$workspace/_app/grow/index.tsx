import { createFileRoute } from '@tanstack/react-router'
import { m } from '@/components/shell/messages'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { Page } from '@/components/shell/Page'
import { useT } from '@/i18n'

// The sidebar's Grow section (phase 5, referrals and recommendations).
export const Route = createFileRoute('/w/$workspace/_app/grow/')({
  component: function GrowRoute() {
    const t = useT()
    return (
      <Page title={t(m.navGrow)}>
        <NotBuilt screen={t(m.navGrow)} />
      </Page>
    )
  },
})
