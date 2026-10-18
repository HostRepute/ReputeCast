import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { EarnPage } from '@/features/earn/EarnPage'
import { me } from '@/features/earn/messages'
import { useT } from '@/i18n'

/** Earn › Products (product conversions). */
export const Route = createFileRoute('/w/$workspace/_app/earn/products/')({
  component: function ProductsRoute() {
    const t = useT()
    return (
      <EarnPage tab="products" title={t(me.tabProducts)}>
        <NotBuilt screen={t(me.tabProducts)} />
      </EarnPage>
    )
  },
})
