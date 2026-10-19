import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { EarnPage } from '@/features/earn/EarnPage'
import { me } from '@/features/earn/messages'
import { useT } from '@/i18n'

/** Earn › Sponsors (placements and bookings). */
export const Route = createFileRoute('/w/$workspace/_app/earn/sponsors/')({
  component: function SponsorsRoute() {
    const t = useT()
    return (
      <EarnPage tab="sponsors" title={t(me.tabSponsors)}>
        <NotBuilt screen={t(me.tabSponsors)} />
      </EarnPage>
    )
  },
})
