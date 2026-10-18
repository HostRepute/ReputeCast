import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { EarnPage } from '@/features/earn/EarnPage'
import { me } from '@/features/earn/messages'
import { useT } from '@/i18n'

/** Earn › Gifts (gift subscriptions). */
export const Route = createFileRoute('/w/$workspace/_app/earn/gifts/')({
  component: function GiftsRoute() {
    const t = useT()
    return (
      <EarnPage tab="gifts" title={t(me.tabGifts)}>
        <NotBuilt screen={t(me.tabGifts)} />
      </EarnPage>
    )
  },
})
