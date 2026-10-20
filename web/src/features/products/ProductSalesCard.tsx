import { Link } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { CardSection } from '@/components/ui/Card'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { SUMMARY } from './api'
import { money } from './format'
import { ps } from './sales.messages'
import './products.css'

/**
 * Earn › Overview's "Product sales" card (design expansion phase 7, §6):
 * the last 30 days' orders, gross and refunds, in the overview's card style
 * (EarnOverview: Product conversions). It shows once something sold.
 */
export function ProductSalesCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', SUMMARY, { params: { path: { workspace: slug } } }, { enabled: can })
  const s = q.data
  if (!can || !s || s.orders === 0) return null
  const cur = s.currency || 'usd'
  return (
    <div className="pr-sales">
      <section className="card" aria-labelledby="pr-sales-h">
        <CardSection
          id="pr-sales-h"
          title={t(ps.title)}
          actions={
            <Link to="/w/$workspace/earn/products" params={{ workspace: slug }} className="pr-lnk">
              {t(ps.products)}
            </Link>
          }
        />
        <div className="pad flex flex-col gap-3">
          <div className="pr-big">
            <div>
              <span className="v">{count(s.orders)}</span>
              <span className="k">{t(ps.orders, { count: s.orders })}</span>
            </div>
            <div>
              <span className="v">{money(s.grossCents, cur, 'never')}</span>
              <span className="k">{t(ps.gross)}</span>
            </div>
            <div>
              <span className="v">{money(s.refundedCents, cur, 'never')}</span>
              <span className="k">{t(ps.refunded, { count: s.refundedOrders })}</span>
            </div>
          </div>
          <span className="k">{t(ps.note, { net: money(s.grossCents - s.refundedCents, cur, 'never'), products: s.products })}</span>
        </div>
      </section>
    </div>
  )
}
