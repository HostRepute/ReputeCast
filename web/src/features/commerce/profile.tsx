import { Link } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { defineMessages, useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { money } from './api'
import './commerce.css'

type Order = Schemas['CommerceContactOrder']
type Store = Schemas['CommerceContactStore']
type ActivityItem = Schemas['ContactActivityItem']

/**
 * A contact's store orders and what each store knows of their consent
 * (CommerceEmails: Sam Rivera's profile). Orders show whatever the
 * consent; "Not added to Acme Merch" says why the store's list doesn't have
 * them.
 */
const cp = defineMessages('commerce.profile', {
  title: 'Store orders · {store} · {count}',
  titleMany: 'Store orders · {count}',
  order: '{name} · {item}',
  orderMore: '{name} · {item} and {more, plural, one {# more} other {# more}}',
  orderLine: '{date} · {amount} · credited to {credited}',
  orderLineNone: '{date} · {amount}',
  refunded: '{amount} refunded',
  notAdded: "Not added to {list}: didn't accept marketing in {platform}",
  onList: 'On {list}: accepted marketing in {platform} on {date}',
  onListNoDate: 'On {list}: accepted marketing in {platform}',
  unsubscribed: 'Unsubscribed from marketing in {platform}',
  accepted: 'Accepted marketing in {platform}',
  spent: '{orders, plural, one {# order} other {# orders}} · {amount} spent in {store}',
  noOrders: "No orders yet; {store}'s customer since the sync.",
  storePage: '{store} attribution',
  // The timeline's store events.
  placed: 'Placed order {order} in {store} · {amount}',
  placedNoName: 'Placed an order in {store} · {amount}',
  abandoned: 'Left a checkout in {store} · {amount}',
  creditedTo: 'credited to {credited}',
  acceptedAt: 'accepted marketing at checkout',
  notAccepted: "didn't accept marketing at checkout",
  enteredFlow: 'entered {flow}',
})

/**
 * The timeline's store events (types store_order and checkout_abandoned):
 * a title and the lines under it, or null for any other event.
 */
export function useStoreActivity(): (it: ActivityItem) => { title: string; meta: string[] } | null {
  const t = useT()
  return (it) => {
    const s = it.store
    if (!s || (it.type !== 'store_order' && it.type !== 'checkout_abandoned')) return null
    const amount = money(s.total, s.currency, 'always')
    const items = s.items.length > 3 ? [...s.items.slice(0, 3), '…'].join(', ') : s.items.join(', ')
    if (it.type === 'checkout_abandoned') {
      return { title: t(cp.abandoned, { store: s.name, amount }), meta: [items, s.flow ? t(cp.enteredFlow, { flow: s.flow }) : ''].filter(Boolean) }
    }
    return {
      title: s.order ? t(cp.placed, { order: s.order, store: s.name, amount }) : t(cp.placedNoName, { store: s.name, amount }),
      meta: [items, s.accepted ? t(cp.acceptedAt) : t(cp.notAccepted), s.credited ? t(cp.creditedTo, { credited: s.credited }) : ''].filter(Boolean),
    }
  }
}

const platformName = (p: string) => (p === 'woocommerce' ? 'WooCommerce' : 'Shopify')

function ConsentLine({ s }: { s: Store }) {
  const t = useT()
  const platform = platformName(s.platform)
  const list = s.listName || s.name
  const text =
    s.marketing === 'subscribed' && s.onList
      ? s.consentAt
        ? t(cp.onList, { list, platform, date: dayMonthDate(s.consentAt) })
        : t(cp.onListNoDate, { list, platform })
      : s.marketing === 'subscribed'
        ? t(cp.accepted, { platform })
        : s.marketing === 'unsubscribed'
          ? t(cp.unsubscribed, { platform })
          : t(cp.notAdded, { list, platform })
  const ok = s.marketing === 'subscribed'
  return (
    <span className="flex items-start gap-1.5 text-[12px] font-medium text-ink-nav">
      <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke={ok ? '#15803D' : '#6B7280'} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="mt-0.5 flex-none">
        {ok ? <path d="m3 8 3.5 3.5L13 5" /> : <path d="M8 2.5a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11ZM4.5 11.5l7-7" />}
      </svg>
      <span>{text}</span>
    </span>
  )
}

function OrderRow({ o }: { o: Order }) {
  const t = useT()
  const item = o.items[0] ?? ''
  const title = o.items.length > 1 ? t(cp.orderMore, { name: o.name, item, more: o.items.length - 1 }) : item ? t(cp.order, { name: o.name, item }) : o.name
  const amount = money(o.total, o.currency, 'always')
  return (
    <div className="flex items-start gap-2.5">
      <span className="cm-merch" style={{ width: 36, height: 36 }} role="img" aria-label={item || o.name}>
        <svg width="20" height="14" viewBox="0 0 40 28" fill="none" stroke="#9AA3B2" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M13 4l-8 5 3 5 4-2v14h16V12l4 2 3-5-8-5c-1 2-3 3-7 3s-6-1-7-3z" />
        </svg>
      </span>
      <span className="flex min-w-0 flex-col gap-1">
        <b className="text-[13px] font-semibold">{title}</b>
        <span className="k">
          {o.credited ? t(cp.orderLine, { date: dayMonthDate(o.placedAt), amount, credited: o.credited }) : t(cp.orderLineNone, { date: dayMonthDate(o.placedAt), amount })}
          {o.refunded > 0 ? ` · ${t(cp.refunded, { amount: money(o.refunded, o.currency, 'always') })}` : ''}
        </span>
      </span>
    </div>
  )
}

/** The profile's Store orders card; nothing when no store knows the contact. */
export function StoreOrdersCard({ contactId }: { contactId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/stores', { params: { path: { workspace: slug, contactId } } }, { staleTime: 30_000 })
  const d = q.data
  if (!d || (d.orders.length === 0 && d.stores.length === 0)) return null
  const one = d.stores.length === 1 ? d.stores[0] : undefined
  const n = d.orders.length
  return (
    <section className="card flex flex-col" aria-labelledby="store-orders-h">
      <div className="flex items-center justify-between gap-3 px-4.5 py-3.5">
        <h2 className="h2" id="store-orders-h">
          {one ? t(cp.title, { store: one.name, count: n }) : t(cp.titleMany, { count: n })}
        </h2>
        {one ? (
          <Link to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: one.storeId }} className="ih-lnk text-[13px]">
            {t(cp.storePage, { store: one.name })}
          </Link>
        ) : null}
      </div>
      <div className="flex flex-col gap-3 px-4.5 pb-3.5">
        {d.orders.slice(0, 5).map((o) => (
          <OrderRow key={o.id} o={o} />
        ))}
        {d.stores.map((s) => (
          <div key={s.storeId} className="flex flex-col gap-1">
            {s.orders === 0 ? <span className="k">{t(cp.noOrders, { store: s.name })}</span> : d.stores.length > 1 ? <span className="k">{t(cp.spent, { orders: s.orders, amount: money(s.spent, s.currency, 'always'), store: s.name })}</span> : null}
            <ConsentLine s={s} />
          </div>
        ))}
      </div>
    </section>
  )
}
