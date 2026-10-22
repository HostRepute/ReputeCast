import { Link } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { CardSection } from '@/components/ui/Card'
import { TableScroll, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { useCan, useWorkspace } from '@/lib/session'
import { CAMPAIGN_SALES } from './elsewhere'
import { mr } from './elsewhere.messages'
import { money } from './format'
import './products.css'

/**
 * The campaign report's product sales (brief §6: "the Revenue section gains
 * rows for product sales"; CommerceEmails: the October Product Update's
 * Revenue section): the orders credited to this email, by product. A block
 * of its own until the Revenue section takes the rows. It shows only when
 * something sold, and only with products:view.
 */
export function ReportProductSales({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', CAMPAIGN_SALES, { params: { path: { workspace: slug, campaign: campaignId } } }, { enabled: can })
  const s = q.data
  if (!can || !s || s.orders === 0) return null
  const cur = s.currency || 'usd'
  return (
    <section className="card min-w-0" aria-labelledby="rep-ps-h">
      <CardSection
        id="rep-ps-h"
        title={t(mr.title)}
        caption={t(mr.caption)}
        actions={
          <Link to="/w/$workspace/earn/products" params={{ workspace: slug }} className="pr-lnk">
            {t(mr.products)}
          </Link>
        }
      />
      <TableScroll label={t(mr.title)}>
        <table className="tbl phone-wrap">
          <thead>
            <tr>
              <Th>{t(mr.colProduct)}</Th>
              <Th align="right">{t(mr.colOrders)}</Th>
              <Th align="right">{t(mr.colRevenue)}</Th>
            </tr>
          </thead>
          <tbody>
            {s.products.map((p) => (
              <tr key={p.productId}>
                <td>
                  <b className="font-semibold">{p.name}</b>
                </td>
                <td className="r n nw">{t(mr.orders, { count: p.orders })}</td>
                <td className="r n nw">{money(p.revenueCents, cur)}</td>
              </tr>
            ))}
            {s.products.length > 1 ? (
              <tr>
                <td>
                  <b>{t(mr.total)}</b>
                </td>
                <td className="r n nw">{t(mr.orders, { count: s.orders })}</td>
                <td className="r nw">
                  <b className="n">{money(s.revenueCents, cur)}</b>
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      <div className="k border-t border-line-2 px-4.5 py-2.5">{t(mr.note, { days: s.window })}</div>
    </section>
  )
}
