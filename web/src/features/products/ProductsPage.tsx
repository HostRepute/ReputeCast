import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Bar, StackBar } from '@/components/ui/Data'
import { Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, count, orList } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { roleName } from '@/lib/roles'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { fileName, saveBlob } from '../audience/download'
import { EarnPage } from '../earn/EarnPage'
import { ORDERS, PRODUCTS } from './api'
import { countsLine, day, filesText, money, orderTime, priceCard, priceMain, priceSub } from './format'
import { pm } from './messages'
import { OrderPanel } from './OrderPanel'
import { Cover, DownloadCell, IconNote, OrderStatus, mono } from './parts'
import './products.css'

type Overview = Schemas['ProductsOverview']
type Item = Schemas['ProductListItem']
type OrderList = Schemas['ProductOrderList']
type T = ReturnType<typeof useT>

export type OrderStatusFilter = 'paid' | 'refunded' | 'partially_refunded' | 'disputed' | 'test'

export type ProductsSearch = {
  /** The numbers' period (30 days when absent). */
  period?: '90d' | '12m'
  /** The orders' filters: search, product, status and date (30 days when absent). */
  q?: string
  product?: string
  status?: OrderStatusFilter
  date?: '7d' | '90d' | 'all'
  page?: number
  /** The order shown in the panel. */
  order?: string
}

const PAGE_SIZE = 10

/** Numbers count test payments while only a test key is connected (the server's numbersMode). */
export function testOnly(p: Schemas['ProductsPayments']): boolean {
  return p.connected && p.mode === 'test' && !p.hasLive
}

/**
 * Earn › Products (Products, ProductsStates): the period's numbers, the
 * products, the orders with their filters and the order panel, where sales
 * came from and buyers as contacts.
 */
export function ProductsPage({ search, onSearch }: { search: ProductsSearch; onSearch: (next: ProductsSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const period = search.period ?? '30d'
  const q = $api.useQuery('get', PRODUCTS, { params: { path: { workspace: slug }, query: { period } } }, { placeholderData: (p) => p })
  const d = q.data
  return (
    <EarnPage tab="products" title={t(pm.tableLabel)} banner={false}>
      {q.error && !d ? (
        <LoadError error={q.error} onRetry={() => void q.refetch()} />
      ) : !d ? (
        <LoadingCard />
      ) : (
        <Body data={d} search={search} onSearch={onSearch} />
      )}
      {search.order ? <OrderPanel id={search.order} payments={d?.payments} onClose={() => onSearch({ ...search, order: undefined })} /> : null}
    </EarnPage>
  )
}

function Body({ data, search, onSearch }: { data: Overview; search: ProductsSearch; onSearch: (next: ProductsSearch, replace?: boolean) => void }) {
  const t = useT()
  const canEdit = useCan('products:edit')
  const canPublish = useCan('products:publish')
  const canOrders = useCan('orders:manage')
  const viewer = !canEdit && !canPublish && !canOrders
  return (
    <>
      <Banners data={data} viewer={viewer} />
      {data.products.length === 0 ? (
        <EmptyProducts canEdit={canEdit} />
      ) : (
        <>
          <div className="pr-head">
            <span className="k pr-ink">
              {t(testOnly(data.payments) ? pm.summaryTest : pm.summaryLive, {
                period: data.period,
                range: t(pm.range, { from: day(data.from), until: day(data.until) }),
                currency: (data.payments.currency || 'usd').toUpperCase(),
              })}
            </span>
            <Segmented
              label={t(pm.periodLabel)}
              value={data.period}
              onChange={(v) => onSearch({ ...search, period: v === '30d' ? undefined : v }, true)}
              items={[
                { value: '30d', label: t(pm.period30) },
                { value: '90d', label: t(pm.period90) },
                { value: '12m', label: t(pm.period12) },
              ]}
            />
          </div>
          <Stats data={data} />
          <ProductsCard data={data} />
          <OrdersCard data={data} search={search} onSearch={onSearch} />
          <div className="pr-grid2">
            <SourcesCard data={data} />
            <BuyersCard data={data} />
          </div>
        </>
      )}
    </>
  )
}

/** Test mode, Stripe not connected, and read-only (ProductsStates). */
function Banners({ data, viewer }: { data: Overview; viewer: boolean }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const canManage = useCan('payments:manage')
  const p = data.payments
  const helpers = data.helpers.edit.length ? data.helpers.edit : data.helpers.publish
  return (
    <>
      {!p.connected ? (
        <IconNote
          tone="w"
          title={t(pm.noStripeTitle)}
          actions={
            <>
              {canManage ? (
                <ButtonLink to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }} size="sm" variant="primary">
                  {t(pm.connectStripe)}
                </ButtonLink>
              ) : null}
              <span className="k">{t(pm.paymentsPath)}</span>
            </>
          }
        >
          {t(pm.noStripeText)}
        </IconNote>
      ) : testOnly(p) ? (
        <IconNote
          tone="pur"
          role="status"
          title={t(pm.testTitle)}
          actions={
            canManage ? (
              <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ mode: 'live' }} size="sm">
                {t(pm.testAction)}
              </ButtonLink>
            ) : null
          }
        >
          {t(pm.testText, { account: p.account })}
        </IconNote>
      ) : null}
      {viewer ? (
        <IconNote tone="n" title={t(pm.readOnlyTitle, { name: user?.name ?? '', role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }) })}>
          <T msg={helpers.length ? pm.readOnlyTextAsk : pm.readOnlyText} values={{ mono, helpers: orList(helpers) }} />
        </IconNote>
      ) : null}
    </>
  )
}

function Stats({ data }: { data: Overview }) {
  const t = useT()
  const s = data.stats
  const cur = data.payments.currency || 'usd'
  const tile = (label: string, value: string, sub?: string) => (
    <div className="card pr-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      {sub ? <span className="s">{sub}</span> : null}
    </div>
  )
  return (
    <div className="pr-stats" role="group" aria-label={t(pm.statsLabel)}>
      {tile(t(pm.statOrders, { period: data.period }), count(s.orders), s.byProduct.length ? countsLine(t, s.byProduct) : undefined)}
      {tile(t(pm.statGross), money(s.grossCents, cur, 'never'), t(pm.statGrossSub))}
      {tile(
        t(pm.statRefunded),
        money(s.refundedCents, cur, 'never'),
        s.openDisputes ? t(pm.statRefundedSubDisputes, { orders: s.refundedOrders, disputes: s.openDisputes }) : t(pm.statRefundedSub, { orders: s.refundedOrders }),
      )}
      {tile(t(pm.statNet), money(s.netCents, cur, 'never'), t(pm.statNetSub))}
    </div>
  )
}

/** "New product": starts a draft and opens it in the editor. */
function NewProductButton({ size = 'sm' }: { size?: 'sm' | 'md' }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const create = $api.useMutation('post', PRODUCTS)
  return (
    <Button
      size={size}
      variant="primary"
      icon="plus"
      loading={create.isPending}
      busyLabel={t(pm.creating)}
      onClick={() =>
        create.mutate(
          { params: { path: { workspace: slug } }, body: {} },
          {
            onSuccess: (p) => void navigate({ to: '/w/$workspace/earn/products/$productId', params: { workspace: slug, productId: p.id } }),
            onError: (e) => toast({ kind: 'error', title: t(pm.createFailed), detail: errorMessage(e) }),
          },
        )
      }
    >
      {t(pm.newProduct)}
    </Button>
  )
}

function ProductsCard({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('products:edit')
  const test = testOnly(data.payments)
  const params = (p: Item) => ({ workspace: slug, productId: p.id })
  return (
    <section className="card" aria-labelledby="pr-products-h">
      <CardSection id="pr-products-h" title={t(pm.productsTitle, { count: data.products.length })} caption={t(pm.productsCaption)} actions={canEdit ? <NewProductButton /> : null} />
      <div className="pr-wide">
        <TableScroll label={t(pm.tableLabel)}>
          <table className="tbl pr-tbl" aria-label={t(pm.tableLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(pm.colProduct)}</th>
                <th scope="col">{t(pm.colPrice)}</th>
                <th scope="col">{t(pm.colStatus)}</th>
                <th scope="col" className="r">
                  {t(pm.colOrders)}
                </th>
                <th scope="col" className="r">
                  {t(pm.colGross)}
                </th>
                <th scope="col" className="r">
                  {t(pm.colRefunds)}
                </th>
                <th scope="col">{t(pm.colSalesPage)}</th>
                <th scope="col">
                  <span className="vh">{t(pm.colActions)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {data.products.map((p) => {
                const cur = p.currency || data.payments.currency || 'usd'
                const none = p.status === 'draft' && p.orders === 0
                const num = (v: string) =>
                  none ? (
                    '–'
                  ) : test ? (
                    <span className="pr-stk items-end">
                      <span>{v}</span>
                      <span className="k">{t(pm.testWord)}</span>
                    </span>
                  ) : (
                    v
                  )
                const sub = priceSub(t, p.price, cur)
                return (
                  <tr key={p.id}>
                    <td>
                      <div className="pr-pname">
                        <Cover url={p.coverUrl} alt={p.coverAlt} name={p.name} w={40} h={53} />
                        <span className="t">
                          <Link to="/w/$workspace/earn/products/$productId" params={params(p)}>
                            {p.name}
                          </Link>
                          <span className="k">{filesText(t, p.files)}</span>
                          {test && p.status === 'published' ? (
                            <Badge tone="purple" size="sm" className="self-start">
                              {t(pm.test)}
                            </Badge>
                          ) : null}
                        </span>
                      </div>
                    </td>
                    <td>
                      {sub ? (
                        <span className="pr-stk">
                          <span>{priceMain(t, p.price, cur)}</span>
                          <span className="k">{sub}</span>
                        </span>
                      ) : (
                        priceMain(t, p.price, cur)
                      )}
                    </td>
                    <td>
                      <ProductStatus p={p} connected={data.payments.connected} />
                    </td>
                    <td className="r pr-num">{num(count(p.orders))}</td>
                    <td className="r pr-num">{num(money(p.grossCents, cur, 'never'))}</td>
                    <td className="r pr-num">{num(p.refunds ? t(pm.refundsLine, { count: count(p.refunds), amount: money(p.refundedCents, cur, 'never') }) : '0')}</td>
                    <td>{p.salesPage.pageId ? <span className="mono text-[12.5px]">{p.salesPage.path}</span> : <span className="k">{t(pm.defaultPage)}</span>}</td>
                    <td className="r">
                      <ButtonLink
                        to="/w/$workspace/earn/products/$productId"
                        params={params(p)}
                        size="sm"
                        aria-label={canEdit ? t(pm.editProduct, { name: p.name }) : t(pm.viewProduct, { name: p.name })}
                      >
                        {canEdit ? t(pm.edit) : t(pm.view)}
                      </ButtonLink>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="pr-narrow">
        {data.products.map((p) => {
          const cur = p.currency || data.payments.currency || 'usd'
          return (
            <Link key={p.id} to="/w/$workspace/earn/products/$productId" params={params(p)} className="pr-pcard">
              <Cover url={p.coverUrl} alt={p.coverAlt} name={p.name} w={40} h={53} decorative />
              <span className="bd">
                <span className="hd">
                  <b className="font-semibold">{p.name}</b>
                  <StatusBadge status={p.status} />
                </span>
                <span className="k pr-ink">{priceCard(t, p.price, cur)}</span>
                <span className="k">
                  {p.status === 'draft' && p.orders === 0
                    ? t(pm.cardDraft, { name: p.statusBy })
                    : t(pm.cardNumbers, {
                        orders: p.orders,
                        gross: money(p.grossCents, cur, 'never'),
                        refunds: p.refunds ? t(pm.refundsLine, { count: count(p.refunds), amount: money(p.refundedCents, cur, 'never') }) : '0',
                      })}
                </span>
              </span>
            </Link>
          )
        })}
      </div>
    </section>
  )
}

function StatusBadge({ status }: { status: Item['status'] }) {
  const t = useT()
  if (status === 'published') return <Badge tone="ok">{t(pm.statusPublished)}</Badge>
  if (status === 'archived') return <Badge tone="neutral">{t(pm.statusArchived)}</Badge>
  return <Badge tone="neutral">{t(pm.statusDraft)}</Badge>
}

/** "Published / 1 Oct by Sofia Rossi", "Draft / Daniel Park". */
function ProductStatus({ p, connected }: { p: Item; connected: boolean }) {
  const t = useT()
  let sub: string | undefined
  if (p.status === 'draft') {
    // A draft with its own sales page waits for Stripe; the others name who's drafting them.
    sub = !connected && p.salesPage.pageId ? t(pm.draftNoStripe) : p.statusBy || undefined
  } else if (p.statusAt) {
    sub = p.statusBy ? t(pm.statusBy, { date: day(p.statusAt), name: p.statusBy }) : day(p.statusAt)
  }
  return (
    <span className="pr-stk">
      <StatusBadge status={p.status} />
      {sub ? <span className="k">{sub}</span> : null}
    </span>
  )
}

function OrdersCard({ data, search, onSearch }: { data: Overview; search: ProductsSearch; onSearch: (next: ProductsSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canExport = useCan('subscribers:export')
  const [q, setQ] = useState(search.q ?? '')
  // The box follows the address when it changes elsewhere (Clear filters, Back).
  const [shown, setShown] = useState(search.q)
  if (shown !== search.q) {
    setShown(search.q)
    setQ(search.q ?? '')
  }
  useEffect(() => {
    const h = window.setTimeout(() => {
      if ((search.q ?? '') !== q.trim()) onSearch({ ...search, q: q.trim() || undefined, page: undefined }, true)
    }, 300)
    return () => window.clearTimeout(h)
  }, [q, search, onSearch])
  const page = search.page ?? 1
  const query = { q: search.q, product: search.product, status: search.status, period: search.date ?? '30d', page, pageSize: PAGE_SIZE } as const
  const list = $api.useQuery('get', ORDERS, { params: { path: { workspace: slug }, query } }, { placeholderData: (p) => p })
  const o = list.data
  const test = testOnly(data.payments)
  const set = (patch: Partial<ProductsSearch>) => onSearch({ ...search, ...patch, page: undefined, order: search.order })
  const helpers = data.helpers.export
  return (
    <section className="card" aria-labelledby="pr-orders-h">
      <CardSection
        id="pr-orders-h"
        title={test ? t(pm.ordersTitleTest) : t(pm.ordersTitle)}
        caption={test ? t(pm.ordersCaptionTest) : t(pm.ordersCaption)}
        actions={
          <Button
            size="sm"
            icon="upload"
            disabledReason={canExport ? undefined : helpers.length ? t(pm.exportNeedsAsk, { helpers: orList(helpers) }) : t(pm.exportNeeds)}
            onClick={() => void exportOrders(slug, query, o?.total ?? 0, t)}
          >
            {t(pm.exportCsv)}
          </Button>
        }
      />
      <div className="pr-filters">
        <label className="pr-search">
          <Icon name="search" size={15} />
          <input type="search" aria-label={t(pm.searchLabel)} placeholder={t(pm.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <Select className="pr-f200" aria-label={t(pm.productLabel)} value={search.product ?? ''} onChange={(e) => set({ product: e.target.value || undefined })}>
          <option value="">{t(pm.productAll)}</option>
          {(o?.products ?? data.products.map((p) => ({ productId: p.id, name: p.name, count: 0 }))).map((p) => (
            <option key={p.productId} value={p.productId}>
              {p.name}
            </option>
          ))}
        </Select>
        <Select
          className="pr-f160"
          aria-label={t(pm.statusLabel)}
          value={search.status ?? ''}
          onChange={(e) => set({ status: (e.target.value || undefined) as OrderStatusFilter | undefined })}
        >
          <option value="">{t(pm.statusAll)}</option>
          <option value="paid">{t(pm.stPaid)}</option>
          <option value="refunded">{t(pm.stRefunded)}</option>
          <option value="partially_refunded">{t(pm.stPartial)}</option>
          <option value="disputed">{t(pm.stDisputed)}</option>
          <option value="test">{t(pm.stTest)}</option>
        </Select>
        <Select
          className="pr-f150"
          aria-label={t(pm.dateLabel)}
          value={search.date ?? '30d'}
          onChange={(e) => set({ date: e.target.value === '30d' ? undefined : (e.target.value as ProductsSearch['date']) })}
        >
          <option value="30d">{t(pm.date30)}</option>
          <option value="7d">{t(pm.date7)}</option>
          <option value="90d">{t(pm.date90)}</option>
          <option value="all">{t(pm.dateAll)}</option>
        </Select>
      </div>
      {list.error && !o ? (
        <p className="k px-[18px] py-4" role="alert">
          {errorMessage(list.error)}
        </p>
      ) : !o ? (
        <div className="pad" aria-busy="true">
          <Skeleton w="100%" h={180} />
        </div>
      ) : o.items.length === 0 ? (
        <div className="flex flex-wrap items-center gap-3 px-[18px] py-6">
          <span className="k">
            {search.q ? t(pm.noOrdersSearch, { q: search.q }) : search.product || search.status || search.date ? t(pm.noOrdersFilter) : t(pm.noOrders)}
          </span>
          {search.q || search.product || search.status || search.date ? (
            <Button size="sm" onClick={() => onSearch({ period: search.period }, true)}>
              {t(pm.clearFilters)}
            </Button>
          ) : null}
        </div>
      ) : (
        <OrdersTable list={o} selected={search.order} onOpen={(id) => onSearch({ ...search, order: id })} />
      )}
      {o && o.total > 0 ? (
        <div className="pr-foot">
          <span className="k">
            <T msg={pm.footer} values={{ shown: o.items.length, total: o.total, mono }} />
          </span>
          <div className="flex gap-2">
            <Button size="sm" disabled={page <= 1} onClick={() => onSearch({ ...search, page: page - 1 > 1 ? page - 1 : undefined })}>
              {t(pm.previous)}
            </Button>
            <Button size="sm" disabled={page * PAGE_SIZE >= o.total} onClick={() => onSearch({ ...search, page: page + 1 })}>
              {t(pm.next)}
            </Button>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function OrdersTable({ list, selected, onOpen }: { list: OrderList; selected?: string; onOpen: (id: string) => void }) {
  const t = useT()
  return (
    <>
      <div className="pr-wide">
        <TableScroll label={t(pm.ordersLabel)}>
          <table className="tbl pr-tbl" aria-label={t(pm.ordersLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(pm.colOrder)}</th>
                <th scope="col">{t(pm.colBuyer)}</th>
                <th scope="col">{t(pm.colProduct)}</th>
                <th scope="col" className="r">
                  {t(pm.colAmount)}
                </th>
                <th scope="col">{t(pm.colStatus)}</th>
                <th scope="col">{t(pm.colDownload)}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((o) => (
                <tr key={o.id} className={o.id === selected ? 'pr-click pr-sel' : 'pr-click'} aria-selected={o.id === selected || undefined} onClick={() => onOpen(o.id)}>
                  <td>
                    <span className="pr-stk">
                      <button
                        type="button"
                        className="pr-onum"
                        aria-label={t(pm.openOrder, { number: o.number })}
                        onClick={(e) => {
                          e.stopPropagation()
                          onOpen(o.id)
                        }}
                      >
                        {o.number}
                      </button>
                      <span className="k">{orderTime(o.paidAt)}</span>
                    </span>
                  </td>
                  <td>
                    <span className="mono text-[12.5px]">{o.buyerEmail}</span>
                  </td>
                  <td>{o.product}</td>
                  <td className="r">
                    <span className="pr-stk">
                      <span className="pr-num font-semibold">{money(o.amountCents, o.currency, 'always')}</span>
                      {o.payWhatYouWant ? <span className="k">{t(pm.pwywNote)}</span> : null}
                    </span>
                  </td>
                  <td>
                    <OrderStatus o={o} />
                  </td>
                  <td>
                    <DownloadCell d={o.download} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="pr-narrow">
        {list.items.map((o) => (
          <button key={o.id} type="button" className="pr-ocard" aria-current={o.id === selected || undefined} onClick={() => onOpen(o.id)}>
            <span className="hd">
              <span>
                <b className="mono font-semibold">{o.number}</b> <span className="k">{orderTime(o.paidAt)}</span>
              </span>
              <OrderStatus o={o} sub={false} dot={false} />
            </span>
            <span className="em">{o.buyerEmail}</span>
            <span className="k pr-ink">
              {o.product} · {money(o.amountCents, o.currency, 'always')}
            </span>
            <span className="self-start">
              <DownloadCell d={o.download} />
            </span>
          </button>
        ))}
      </div>
    </>
  )
}

async function exportOrders(slug: string, query: { q?: string; product?: string; status?: OrderStatusFilter; period: string }, n: number, t: T) {
  const id = `products-export-${Date.now()}`
  trackJob(slug, { id, kind: 'export', title: t(pm.exporting), status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response } = await http.GET(`${ORDERS}/export`, {
      params: { path: { workspace: slug }, query: { q: query.q, product: query.product, status: query.status, period: query.period as '30d' } },
      parseAs: 'blob',
    })
    if (data) saveBlob(data as unknown as Blob, fileName(response, 'orders.csv'))
    updateJob(slug, id, { status: 'done', detail: t(pm.exportedDetail), finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: t(pm.exported, { count: n }), detail: t(pm.exportedDetail) })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: t(pm.exportFailed), detail: errorMessage(e) })
  }
}

function SourcesCard({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const cur = data.payments.currency || 'usd'
  const max = Math.max(1, ...data.sources.map((s) => s.orders))
  const label = (s: Schemas['ProductSourceRow']) =>
    s.source === 'page' ? t(pm.srcPage, { product: s.product }) : s.source === 'shop' ? t(pm.srcShop, { product: s.product }) : t(pm.srcDefault, { product: s.product })
  return (
    <section className="card" aria-labelledby="pr-sources-h">
      <CardSection
        id="pr-sources-h"
        title={t(pm.sourcesTitle)}
        caption={t(pm.sourcesCaption)}
        actions={
          <Link to="/w/$workspace/audience/growth" params={{ workspace: slug }} className="pr-lnk">
            {t(pm.growthSources)}
          </Link>
        }
      />
      <div className="pad">
        {data.sources.length ? (
          <div className="pr-bars">
            {data.sources.map((s) => (
              <div key={`${s.productId}-${s.source}`} className="pr-srcrow">
                <div className="lt">
                  <span>{label(s)}</span>
                  <span className="pr-num font-semibold">
                    {count(s.orders)} <span className="k">{money(s.grossCents, cur, 'never')}</span>
                  </span>
                </div>
                <Bar value={(s.orders / max) * 100} label={t(pm.sourceLabel, { source: label(s), orders: s.orders, gross: money(s.grossCents, cur, 'never') })} />
              </div>
            ))}
          </div>
        ) : (
          <p className="k m-0">{t(pm.sourcesEmpty)}</p>
        )}
      </div>
    </section>
  )
}

function BuyersCard({ data }: { data: Overview }) {
  const t = useT()
  const b = data.buyers
  const total = b.contacts + b.joined + b.none
  const colors = { contacts: '#2a78d6', joined: '#1baf7a', none: '#8b94a3' }
  return (
    <section className="card" aria-labelledby="pr-buyers-h">
      <CardSection id="pr-buyers-h" title={t(pm.buyersTitle)} caption={t(pm.buyersCaption)} />
      <div className="pad flex flex-col gap-3">
        {total > 0 ? (
          <>
            <div className="flex flex-col gap-2.5">
              <StackBar
                height={10}
                label={t(pm.buyersBar, { contacts: b.contacts, joined: b.joined, none: b.none })}
                segments={[
                  { value: b.contacts, color: colors.contacts },
                  { value: b.joined, color: colors.joined },
                  { value: b.none, color: colors.none },
                ]}
              />
              <div className="pr-legend">
                <span>
                  <i style={{ background: colors.contacts }} aria-hidden="true" />
                  {t(pm.buyersContacts, { count: count(b.contacts) })}
                </span>
                <span>
                  <i style={{ background: colors.joined }} aria-hidden="true" />
                  {t(pm.buyersJoined, { count: count(b.joined) })}
                </span>
                <span>
                  <i style={{ background: colors.none }} aria-hidden="true" />
                  {t(pm.buyersNone, { count: count(b.none) })}
                </span>
              </div>
            </div>
            <span className="k pr-ink">
              {b.ticked > 0 && b.offerText
                ? t(pm.buyersNote, {
                    product: b.product,
                    ticked: b.ticked,
                    offer: b.offerText,
                    byProduct: andList(b.tickedBy.map((x) => t(pm.statOrdersBy, { count: count(x.count), name: x.name }))),
                  })
                : t(pm.buyersNoteNoTicks, { product: b.product })}
            </span>
          </>
        ) : (
          <span className="k">{t(pm.buyersEmpty)}</span>
        )}
      </div>
    </section>
  )
}

function EmptyProducts({ canEdit }: { canEdit: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canPages = useCan('website:edit')
  return (
    <section className="card pr-empty" aria-labelledby="pr-empty-h">
      <span className="covers" role="img" aria-label={t(pm.emptyCovers)}>
        <EmptyCover kind="guide" />
        <EmptyCover kind="kit" />
        <EmptyCover kind="planner" />
      </span>
      <h2 className="tt" id="pr-empty-h">
        {t(pm.emptyTitle)}
      </h2>
      <p className="ds">{t(pm.emptyText)}</p>
      {canEdit ? (
        <div className="acts">
          <NewProductButton size="md" />
          {canPages ? <ButtonLink to="/w/$workspace/forms/pages" params={{ workspace: slug }}>{t(pm.startWithPage)}</ButtonLink> : null}
        </div>
      ) : null}
    </section>
  )
}

/** The empty state's three covers (ProductsStates): a guide, a template kit and a planner. */
function EmptyCover({ kind }: { kind: 'guide' | 'kit' | 'planner' }) {
  const box = { position: 'relative' as const, width: 44, height: 59, borderRadius: 4, overflow: 'hidden', boxShadow: '0 1px 2px rgba(11,18,32,.18)', display: 'inline-block' }
  const abs = (s: Record<string, string | number>) => ({ position: 'absolute' as const, ...s })
  if (kind === 'guide') {
    return (
      <span style={{ ...box, background: '#0B1220' }} aria-hidden="true">
        <span style={abs({ left: 0, right: 0, top: '58%', height: '14%', background: '#4338CA' })} />
        <span style={abs({ left: 0, right: 0, top: '72%', height: '6%', background: '#3730A3' })} />
      </span>
    )
  }
  if (kind === 'kit') {
    const sq = (left: string, top: string, background: string) => <span style={abs({ left, top, width: '34%', height: '22%', borderRadius: 2, background })} />
    return (
      <span style={{ ...box, background: '#EDE9FE' }} aria-hidden="true">
        {sq('12%', '24%', '#4338CA')}
        {sq('54%', '24%', '#fff')}
        {sq('12%', '52%', '#fff')}
        {sq('54%', '52%', '#3730A3')}
      </span>
    )
  }
  return (
    <span style={{ ...box, background: '#fff', border: '1px solid #DDD6FE', boxSizing: 'border-box' }} aria-hidden="true">
      <span style={abs({ left: '14%', right: '14%', top: '12%', height: '12%', borderRadius: 2, background: '#4338CA' })} />
      {['40%', '55%', '70%', '85%'].map((top) => (
        <span key={top} style={abs({ left: '14%', right: '14%', top, height: 1, background: '#DDD6FE' })} />
      ))}
    </span>
  )
}

/** Loading (ProductsStates): two numbers and three product rows. */
function LoadingCard() {
  const t = useT()
  return (
    <section className="card pr-loading" aria-busy="true" aria-label={t(pm.loading)}>
      <span className="vh" role="status">
        {t(pm.loading)}
      </span>
      <div className="pr-stats">
        {[0, 1].map((i) => (
          <div key={i} className="card pr-stat gap-2">
            <Skeleton w={90} h={10} />
            <Skeleton w={70} h={20} />
            <Skeleton w={120} h={10} />
          </div>
        ))}
      </div>
      <div>
        {[0, 1, 2].map((i) => (
          <div key={i} className="row3">
            <Skeleton w={40} h={52} />
            <Skeleton w="34%" h={12} />
            <Skeleton w="40%" h={12} />
            <Skeleton w={50} h={28} className="ml-auto" />
          </div>
        ))}
      </div>
    </section>
  )
}

/** "Couldn't load products and orders" (ProductsStates). */
function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const ref = problemReference(error)
  const text =
    error instanceof ApiError && error.status >= 500
      ? t(pm.errorServer, { status: error.status })
      : error instanceof TypeError
        ? t(pm.errorNetwork)
        : t(pm.errorOther, { message: errorMessage(error) })
  return (
    <section className="card pr-error" role="alert">
      <Icon name="alertCircle" size={26} />
      <h2 className="tt">{t(pm.errorTitle)}</h2>
      <span className="k max-w-[520px]">{text}</span>
      <div className="acts">
        <Button variant="primary" onClick={onRetry}>
          {t(pm.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(pm.statusOnOverview)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(pm.reference, { id: ref })}</span> : null}
    </section>
  )
}
