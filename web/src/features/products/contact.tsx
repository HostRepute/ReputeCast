import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, t as tr, useT } from '@/i18n'
import { useNow } from '@/lib/hooks'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { ORDER, invalidateProducts, type OrderDetail } from './api'
import { PURCHASES, checkoutConsentText, purchaseChip, purchaseLine } from './elsewhere'
import { mp } from './elsewhere.messages'
import { money } from './format'
import { NewLinkDialog } from './OrderPanel'
import { Chip, Cover, OrderStatus, bold } from './parts'
import './products.css'

/*
 * Digital products on the contact profile (publish-sell brief §6): the
 * Purchases card, the timeline's bought and refunded events, and the
 * consent history's product checkout. The contacts list's filter is in
 * filter.tsx.
 */

type Purchase = Schemas['ContactPurchase']
type Item = Schemas['ContactActivityItem']
type ConsentEvent = Schemas['ContactConsentEvent']

/**
 * The Purchases card (CommerceEmails: Sam Rivera's profile): each product
 * the contact bought, with its order's status, the download link's state and
 * "Send a new link". Hidden without products:view, and for contacts who
 * never bought anything.
 */
export function PurchasesCard({ contactId }: { contactId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', PURCHASES, { params: { path: { workspace: slug, contact: contactId } } }, { enabled: can })
  const d = q.data
  if (!can) return null
  if (q.isError && !d) {
    return (
      <section className="card px-4.5 py-4" aria-label={t(mp.error)}>
        <ErrorPanel
          title={t(mp.error)}
          actions={
            <Button size="sm" onClick={() => void q.refetch()}>
              {t(mp.retry)}
            </Button>
          }
        >
          {errorMessage(q.error)}
        </ErrorPanel>
      </section>
    )
  }
  if (!d?.items.length) return null
  return (
    <section className="card flex flex-col" aria-labelledby="purchases-h">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4.5 py-3.5">
        <h2 className="h2" id="purchases-h">
          {t(mp.title, { count: d.items.length })}
        </h2>
        <span className="k">{t(mp.spent, { amount: money(d.spentCents, d.currency, 'always') })}</span>
      </div>
      <div className="flex flex-col px-4.5 pb-1.5">
        {d.items.map((o) => (
          <PurchaseRow key={o.id} o={o} canManage={d.canManage} />
        ))}
      </div>
    </section>
  )
}

/** One purchase: the product (linked to its order panel), when and what was paid, the download chip and the status. */
function PurchaseRow({ o, canManage }: { o: Purchase; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const [order, setOrder] = useState<OrderDetail>()
  const [opening, setOpening] = useState(false)
  const chip = purchaseChip(t, o.download, now)
  const revoked = o.download.state === 'revoked' || o.download.state === 'revoked_auto'
  const path = { workspace: slug, order: o.id }
  // "Send a new link" is the order panel's dialog: it needs the whole order.
  const open = async () => {
    setOpening(true)
    try {
      setOrder(await qc.fetchQuery($api.queryOptions('get', ORDER, { params: { path } })))
    } catch (e) {
      toast({ kind: 'error', title: t(mp.orderError, { number: String(o.number) }), detail: errorMessage(e) })
    } finally {
      setOpening(false)
    }
  }
  const done = (next: OrderDetail) => {
    qc.setQueryData($api.queryOptions('get', ORDER, { params: { path } }).queryKey, next)
    invalidateProducts(qc)
    void qc.invalidateQueries({ queryKey: ['get', PURCHASES] })
  }
  return (
    <div className="pkv items-start py-2.5">
      <div className="flex min-w-0 gap-2.5">
        <Cover url={o.coverUrl} alt={o.coverAlt} name={o.productName} w={36} h={48} decorative />
        <div className="flex min-w-0 flex-col items-start gap-1">
          <Link
            to="/w/$workspace/earn/products"
            params={{ workspace: slug }}
            search={{ order: o.id }}
            className="font-semibold text-ink no-underline"
            aria-label={t(mp.orderLink, { product: o.productName, number: String(o.number) })}
          >
            {o.productName}
          </Link>
          <span className="k">{purchaseLine(t, o, now)}</span>
          <Chip tone={chip.tone}>{chip.text}</Chip>
          {revoked ? null : (
            <Button size="sm" loading={opening} disabledReason={canManage ? undefined : deniedReason('orders:manage', [], t(mp.sendLinkAction))} onClick={() => void open()}>
              {t(mp.sendLink)}
            </Button>
          )}
        </div>
      </div>
      <span className="flex-none">
        <OrderStatus o={o} sub={false} />
      </span>
      {order ? <NewLinkDialog o={order} onClose={() => setOrder(undefined)} onDone={done} /> : null}
    </div>
  )
}

/** A purchase or a refund on the timeline: its title and the line under it; null for other events. */
export function purchaseActivity(it: Item): { title: ReactNode; meta: string[] } | null {
  if (it.category !== 'engagement' || (it.type !== 'bought' && it.type !== 'refunded')) return null
  return {
    title: <T msg={it.type === 'bought' ? mp.bought : mp.refunded} values={{ b: bold, product: it.subject }} />,
    // "$29.00 · order #1311", "$29.00 refunded · order #1311" (the server's words).
    meta: it.detail ? [it.detail] : [],
  }
}

/**
 * The timeline's icons for purchases, on its 16-unit grid: the Earn icon for
 * a purchase, and an arrow back for a refund.
 */
export const purchaseIcons = {
  earn: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M10 5.9H7.1a1.3 1.3 0 0 0 0 2.6h1.8a1.3 1.3 0 0 1 0 2.6H6M8 4.4v1.5M8 11.1v1.5" />
    </>
  ),
  back: <path d="M6 3.5 2.5 7 6 10.5M2.5 7H10a3.5 3.5 0 0 1 0 7H8" />,
}

/** A purchase's icon and tone on the timeline; null for other events. */
export function purchaseIcon(it: Item): ['earn', 'green'] | ['back', 'grey'] | null {
  if (it.category !== 'engagement') return null
  if (it.type === 'bought') return ['earn', 'green']
  if (it.type === 'refunded') return ['back', 'grey']
  return null
}

/**
 * The consent history's entry for a buyer who ticked a product's list offer
 * at checkout, in the app's language: "Product checkout: ticked “Also send
 * me the Weekly Digest” 15 Oct 2026 10:12". Undefined for other entries.
 */
export function checkoutConsent(e: ConsentEvent, when: string): string | undefined {
  return checkoutConsentText(tr, e, when)
}
