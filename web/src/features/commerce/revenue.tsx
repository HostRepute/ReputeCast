import { Link } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useOptionalWorkspace, useWorkspace } from '@/lib/session'
import { commerceBase, money, share } from './api'
import { cr } from './elsewhere.messages'
import './commerce.css'

type Row = Schemas['CommerceCampaignStoreRow']

/*
 * Store orders in revenue (design expansion phase 7): the campaign
 * report's store rows ("Acme Merch · 31 orders · $1,612"), in a section of
 * their own after Earn's Revenue and Sponsor sections (X17, which shows
 * upgrades and conversions), and Earn › Overview's "Store revenue from
 * email" card, beside the revenue overview's Conversions.
 */

const platformName = (p: string) => (p === 'woocommerce' ? 'WooCommerce' : 'Shopify')

/** A campaign's store rows: one per store with orders credited to it. */
export function useCampaignStoreRevenue(campaignId: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{id}/store-revenue', { params: { path: { workspace: slug, id: campaignId } } }, { staleTime: 60_000 })
}

/** The Revenue table's store rows (Source, Count, Revenue, How it's credited). */
export function StoreRevenueRows({ rows }: { rows: Row[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      {rows.map((r) => (
        <tr key={r.storeId}>
          <td>
            <Link to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: r.storeId }} className="font-semibold text-ink no-underline hover:underline">
              {r.name}
            </Link>
          </td>
          <td className="r nw">
            <span className="num">{t(cr.orders, { count: r.orders })}</span>
          </td>
          <td className="r nw">
            <span className="num">{money(r.revenue, r.currency)}</span>
          </td>
          <td>
            <span className="k text-ink-nav">{t(cr.credited, { platform: platformName(r.platform), days: r.days, token: count(r.byToken), email: count(r.byEmail) })}</span>
          </td>
        </tr>
      ))}
    </>
  )
}

/**
 * The report's Revenue section, store rows only (nothing without a credited
 * order); a shared report, outside the workspace, leaves it out.
 */
export function CampaignStoreRevenue({ campaignId, campaign }: { campaignId: string; campaign: string }) {
  return useOptionalWorkspace() ? <StoreRevenueSection campaignId={campaignId} campaign={campaign} /> : null
}

function StoreRevenueSection({ campaignId, campaign }: { campaignId: string; campaign: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useCampaignStoreRevenue(campaignId)
  const rows = (q.data?.rows ?? []).filter((r) => r.orders > 0)
  if (!rows.length) return null
  const cur = rows[0]!.currency
  const total = rows.reduce((s, r) => s + r.revenue, 0)
  return (
    <section className="card min-w-0" aria-labelledby="rep-rev-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="rep-rev-h">
            {t(cr.title, { campaign })}
          </h2>
          <span className="k">{t(cr.caption)}</span>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="tbl compact" aria-label={t(cr.title, { campaign })}>
          <thead>
            <tr>
              <th scope="col">{t(cr.source)}</th>
              <th scope="col" className="r">
                {t(cr.count)}
              </th>
              <th scope="col" className="r">
                {t(cr.revenue)}
              </th>
              <th scope="col">{t(cr.how)}</th>
            </tr>
          </thead>
          <tbody>
            <StoreRevenueRows rows={rows} />
            <tr>
              <td>
                <b>{t(cr.total)}</b>
              </td>
              <td className="r" />
              <td className="r">
                <b className="num">{money(total, cur)}</b>
              </td>
              <td>
                <span className="k">{t(cr.totalNote)}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      {rows.length === 1 ? (
        <div className="flex justify-end border-t border-line-2 px-4.5 py-2.5">
          <Link to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: rows[0]!.storeId }} className="ih-lnk text-[13px]">
            {t(cr.attribution, { store: rows[0]!.name })}
          </Link>
        </div>
      ) : null}
    </section>
  )
}

/** Earn › Overview: the stores' revenue credited to emails in the last 30 days. */
export function StoreRevenueCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${commerceBase}/revenue`, { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const r = q.data
  if (!r || r.stores === 0) return null
  return (
    <section className="card flex flex-col gap-1.5 px-4.5 py-4" aria-labelledby="earn-store-rev">
      <h3 className="k font-semibold text-ink-2" id="earn-store-rev">
        {t(cr.cardTitle)}
      </h3>
      {r.credited > 0 ? (
        <>
          <span className="text-[24px] font-bold tracking-[-0.01em]">{t(cr.cardValue, { amount: money(r.creditedRevenue, r.currency), total: money(r.revenue, r.currency) })}</span>
          <span className="k">{t(cr.cardSub, { share: share(r.share), stores: r.stores, orders: r.credited, days: r.days })}</span>
        </>
      ) : (
        <span className="k">{t(cr.cardNone, { days: r.days })}</span>
      )}
    </section>
  )
}
