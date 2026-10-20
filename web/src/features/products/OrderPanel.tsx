import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, clockSeconds, dayMonthDate, dayMonthYear, decimal, orList } from '@/lib/format'
import { useMediaQuery, useNow, useReducedMotion } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { ORDER, PRODUCT, invalidateProducts, type OrderDetail } from './api'
import { bareUrl, day, eventTime, fileSize, filesText, money, orderStatus, orderTime, priceShort, sameDay } from './format'
import { pm } from './messages'
import { po } from './order.messages'
import { Cover, IconNote, PanelChip, mono } from './parts'

type T = ReturnType<typeof useT>
type Event = Schemas['ProductOrderEvent']
type Payments = Schemas['ProductsPayments']

/**
 * The order panel (Products, ProductsStates): a sheet on the right above
 * 1100 px; below, a card at the end of the page, scrolled into view, as the
 * tablet and phone boards show it.
 */
export function OrderPanel({ id, payments, onClose }: { id: string; payments?: Payments; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const narrow = useMediaQuery('(max-width: 1100px)')
  const q = $api.useQuery('get', ORDER, { params: { path: { workspace: slug, order: id } } })
  const o = q.data
  const body = q.error ? (
    <div className="pad">
      <ErrorPanel title={t(po.error)} actions={<Button onClick={() => void q.refetch()}>{t(pm.retry)}</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    </div>
  ) : !o ? (
    <div className="pad flex flex-col gap-3" aria-busy="true">
      <span className="vh" role="status">
        {t(po.loading)}
      </span>
      <Skeleton w="50%" h={14} />
      <Skeleton w="70%" h={24} />
      <Skeleton w="100%" h={180} />
      <Skeleton w="100%" h={220} />
    </div>
  ) : null
  if (narrow) {
    return (
      <Inline label={o ? t(po.panelLabel, { number: o.number }) : t(po.loading)} focusKey={o?.id}>
        {(heading) => body ?? <PanelContent o={o!} payments={payments} onClose={onClose} heading={heading} />}
      </Inline>
    )
  }
  return (
    <RDialog.Root open onOpenChange={(open) => !open && onClose()}>
      <RDialog.Portal>
        <RDialog.Overlay className="pr-ovl" />
        <RDialog.Content className="pr-sheet" aria-describedby={undefined}>
          {body ? (
            <>
              <RDialog.Title className="vh">{q.error ? t(po.error) : t(po.loading)}</RDialog.Title>
              {body}
            </>
          ) : (
            <PanelContent
              o={o!}
              payments={payments}
              onClose={onClose}
              heading={(children) => <RDialog.Title asChild>{children}</RDialog.Title>}
            />
          )}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

/** The panel as a card at the end of the page: it scrolls into view and its heading takes focus. */
function Inline({ label, focusKey, children }: { label: string; focusKey?: string; children: (heading: (h: ReactNode) => ReactNode) => ReactNode }) {
  const ref = useRef<HTMLElement>(null)
  const reduced = useReducedMotion()
  useEffect(() => {
    const el = ref.current
    if (!el || !focusKey) return
    el.scrollIntoView({ block: 'start', behavior: reduced ? 'auto' : 'smooth' })
    el.querySelector<HTMLElement>('h2')?.focus({ preventScroll: true })
  }, [focusKey, reduced])
  return (
    <section ref={ref} className="card pr-panel" aria-label={label}>
      {children((h) => h)}
    </section>
  )
}

function PanelContent({ o, payments, onClose, heading }: { o: OrderDetail; payments?: Payments; onClose: () => void; heading: (h: ReactNode) => ReactNode }) {
  const t = useT()
  const now = useNow()
  const status = orderStatus(t, o, now)
  return (
    <>
      <div className="pr-oph">
        <div className="hd">
          <span className="k">{o.productName}</span>
          {heading(<h2 tabIndex={-1}>{t(po.title, { number: o.number, amount: money(o.amountCents, o.currency, 'always') })}</h2>)}
          <div className="st">
            {o.test ? (
              <Badge tone="purple">{t(pm.test)}</Badge>
            ) : (
              <Badge tone={status.tone} dot={status.dot}>
                {status.label}
              </Badge>
            )}
            <span className="k">{statusLine(t, o)}</span>
          </div>
        </div>
        <Button size="sm" iconOnly icon="close" aria-label={t(po.close)} onClick={onClose} />
      </div>
      <div className="pr-opb">
        <ProductSection o={o} />
        <BuyerSection o={o} />
        <PaymentSection o={o} payments={payments} />
        <DeliverySection o={o} now={now} />
        {o.activity.length ? (
          <>
            <h3>{t(po.secActivity)}</h3>
            <ol className="pr-ols">
              {o.activity.map((ev, i) => (
                <EventRow key={`${ev.kind}-${ev.at}-${i}`} ev={ev} o={o} now={now} />
              ))}
            </ol>
          </>
        ) : null}
      </div>
      <Actions o={o} />
    </>
  )
}

/** "15 Oct 2026, 10:12". */
function dayTime(t: T, d: string): string {
  return t(po.when, { date: dayMonthYear(d), time: clock(d) })
}

/** The line next to the status badge: when it was paid, refunded or disputed. */
function statusLine(t: T, o: OrderDetail): string {
  const paid = dayTime(t, o.paidAt)
  const at = (d: string) => (sameDay(d, o.paidAt) ? clock(d) : orderTime(d))
  if (o.test) return paid
  switch (o.status) {
    case 'refunded':
      return o.refundedAt ? t(po.paidRefunded, { paid, refunded: at(o.refundedAt) }) : paid
    case 'partially_refunded':
      return o.refundedAt ? t(po.paidPartial, { paid, amount: money(o.refundedCents, o.currency, 'always'), refunded: at(o.refundedAt) }) : paid
    case 'disputed':
      if (o.disputeStatus === 'won') return t(po.paidDisputeWon, { paid })
      if (o.disputeStatus === 'lost') return t(po.paidDisputeLost, { paid })
      return o.disputedAt ? t(po.paidDisputeOpen, { paid, date: day(o.disputedAt) }) : paid
  }
  return paid
}

function Line({ label, children, className }: { label: ReactNode; children: ReactNode; className?: string }) {
  return (
    <div className="pr-oline">
      <span className="lab">{label}</span>
      <div className={className ? `val ${className}` : 'val'}>{children}</div>
    </div>
  )
}

function ProductSection({ o }: { o: OrderDetail }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      <h3>{t(po.secProduct)}</h3>
      <Line label={t(po.labProduct)} className="prod">
        <Cover url={o.productCover} alt={o.productAlt} name={o.productName} w={30} h={40} decorative />
        <div className="flex min-w-0 flex-col gap-0.5">
          <Link to="/w/$workspace/earn/products/$productId" params={{ workspace: slug, productId: o.productId }} className="self-start font-semibold text-ink">
            {o.productName}
          </Link>
          <span className="k">{t(po.productLine, { files: filesText(t, o.files), price: priceShort(t, o.price, o.currency) })}</span>
        </div>
      </Line>
    </>
  )
}

function BuyerSection({ o }: { o: OrderDetail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canContacts = useCan('subscribers:view')
  const b = o.buyer
  const c = o.consent
  let who = t(po.notContact)
  if (b.contactId && b.contactSince) {
    const when = `${dayMonthDate(b.contactSince)}, ${clock(b.contactSince)}`
    who = t(po.contactSince, { when })
    if (b.list && b.listStatus === 'subscribed' && b.doubleOptIn && b.confirmedAt) {
      who = sameDay(b.confirmedAt, b.contactSince)
        ? t(po.joinedConfirmed, { when, list: b.list, time: clock(b.confirmedAt) })
        : t(po.joinedConfirmedDay, { when, list: b.list, date: day(b.confirmedAt) })
    } else if (b.list && b.listStatus === 'subscribed') who = t(po.joinedSingle, { when, list: b.list })
    else if (b.list && b.listStatus === 'pending') who = t(po.joinedPending, { when, list: b.list })
  }
  let consent = t(po.consentNone)
  if (c.offered && !c.ticked) consent = t(po.consentNotTicked)
  if (c.offered && c.ticked) {
    const when = c.at ? `${dayMonthDate(c.at)}, ${clockSeconds(c.at)}` : ''
    consent = c.page && c.ip ? t(po.consentTicked, { text: c.text, when, page: bareUrl(c.page), ip: c.ip }) : t(po.consentTickedShort, { text: c.text, when })
  }
  return (
    <>
      <h3>{t(po.secBuyer)}</h3>
      <Line label={t(po.labEmail)}>
        {b.contactId && canContacts ? (
          <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: b.contactId }} className="strong">
            {o.buyerEmail}
          </Link>
        ) : (
          <span className="mono">{o.buyerEmail}</span>
        )}
        <span className="k">{who}</span>
      </Line>
      <Line label={t(po.labConsent)}>
        <span>{consent}</span>
      </Line>
    </>
  )
}

function PaymentSection({ o, payments }: { o: OrderDetail; payments?: Payments }) {
  const t = useT()
  const p = o.payment
  const cur = o.currency
  const amount = money(o.amountCents, cur, 'always')
  const account = p.account || payments?.account || t(po.stripe)
  const refundedEvent = [...o.activity].reverse().find((e) => e.kind === 'refunded')?.stripe
  let line: string
  let button: ReactNode = null
  const open = p.url ? (
    <span>
      <ButtonAnchor href={p.url} size="sm" iconRight="external">
        {t(po.openInStripe)}
      </ButtonAnchor>
    </span>
  ) : null
  if (o.test) {
    line = t(po.payTest, { amount, account })
    button = open
  } else if (o.status === 'refunded') {
    const time = p.refundedAt ? clock(p.refundedAt) : ''
    line = refundedEvent ? t(po.payRefunded, { amount: money(p.refundedCents, cur, 'always'), event: refundedEvent, time }) : t(po.payRefundedPlain, { amount: money(p.refundedCents, cur, 'always'), time })
  } else if (o.status === 'partially_refunded') {
    const time = p.refundedAt ? clock(p.refundedAt) : ''
    const refunded = money(p.refundedCents, cur, 'always')
    line = refundedEvent ? t(po.payPartial, { refunded, amount, event: refundedEvent, time }) : t(po.payPartialPlain, { refunded, amount, time })
    button = open
  } else if (o.status === 'disputed') {
    const fee = p.disputeFeeCents ? money(p.disputeFeeCents, cur, 'always') : ''
    if (p.disputeStatus === 'won') line = t(po.payDisputeWon, { amount })
    else if (p.disputeStatus === 'lost') line = fee ? t(po.payDisputeLost, { amount, fee }) : t(po.payDisputeLostNoFee, { amount })
    else line = fee ? t(po.payDisputeOpen, { amount, fee }) : t(po.payDisputeOpenNoFee, { amount })
  } else {
    const mode = p.mode
    line = payments?.taxAtCheckout ? t(po.payPaid, { amount, tax: money(p.taxCents, cur, 'always'), account, mode }) : t(po.payPaidNoTax, { amount, account, mode })
    button = open
  }
  return (
    <>
      <h3>{t(po.secPayment)}</h3>
      <Line label={t(po.stripe)}>
        <span>{line}</span>
        {button}
      </Line>
    </>
  )
}

function DeliverySection({ o, now }: { o: OrderDetail; now: Date }) {
  const t = useT()
  const { slug } = useWorkspace()
  const d = o.delivery
  let email: ReactNode
  if (d) {
    const v = { template: d.template, version: d.version, provider: d.provider }
    let text: string
    if (d.deliveredAt) text = t(d.newLink ? po.deliveryDeliveredNew : po.deliveryDelivered, { ...v, time: clockSeconds(d.deliveredAt) })
    else if (['bounced', 'complained', 'rejected', 'failed', 'cancelled'].includes(d.status)) text = t(po.deliveryFailed, { ...v, status: d.status })
    else if (d.sentAt) text = t(d.newLink ? po.deliverySentNew : po.deliverySent, { ...v, time: clockSeconds(d.sentAt) })
    else text = t(po.deliveryQueued, v)
    email = d.messageId ? (
      <>
        <Link to="/w/$workspace/transactional/messages/$messageId" params={{ workspace: slug, messageId: d.messageId }} className="strong">
          {text}
        </Link>
        <span className="k">{t(po.opensInMessages)}</span>
      </>
    ) : (
      <span>{text}</span>
    )
  } else {
    email = <span>{o.download.state === 'failed' && o.download.error ? t(po.deliveryNoneFailed, { reason: o.download.error }) : t(po.deliveryNone)}</span>
  }
  return (
    <>
      <h3>{t(po.secDelivery)}</h3>
      <Line label={t(po.labEmail)}>{email}</Line>
      <Line label={t(po.labDownload)}>
        <span>
          <PanelChip d={o.download} />
        </span>
        <span className="k">{downloadNote(t, o, now)}</span>
      </Line>
    </>
  )
}

/** The line under the download chip. */
function downloadNote(t: T, o: OrderDetail, now: Date): string {
  const d = o.download
  const n = o.downloads
  if (d.state === 'revoked') {
    if (d.revokeReason === 'refund' && !d.revokedBy) return t(po.revokedRefund)
    if (d.revokedBy && d.revokedAt) return t(po.revokedBy, { name: d.revokedBy, date: day(d.revokedAt) })
    return d.revokedAt ? t(po.revokedSystem, { date: day(d.revokedAt) }) : ''
  }
  if (d.state === 'revoked_auto') {
    if (!n.count) return t(po.dlNever)
    return n.addresses > 1 ? t(po.dlBeforeDisputeAddresses, { count: n.count, addresses: n.addresses }) : t(po.dlBeforeDispute, { count: n.count })
  }
  if (d.state === 'failed') return t(po.failedNote)
  if (d.state === 'pending') return t(po.pendingNote)
  if (d.newLink && (d.state === 'delivered' || d.state === 'sent')) return d.newLinkBy ? t(po.newLinkBy, { name: d.newLinkBy }) : t(po.newLinkBuyer)
  if (!n.count || !n.last) return t(po.dlNever)
  const today = sameDay(n.last, now)
  if (n.count === 1) return today ? t(po.dlOnceToday, { time: clock(n.last) }) : t(po.dlOnceDay, { date: day(n.last), time: clock(n.last) })
  if (n.addresses > 1) return t(po.dlManyAddresses, { count: n.count, addresses: n.addresses, date: day(n.last), time: clock(n.last) })
  return today ? t(po.dlManyToday, { count: n.count, time: clock(n.last) }) : t(po.dlManyDay, { count: n.count, date: day(n.last), time: clock(n.last) })
}

type EvTone = 'ok' | 'info' | 'mut' | 'bad' | 'warn'
const evIcon: Record<EvTone, IconName> = { ok: 'check', info: 'arrowRight', mut: 'more', bad: 'close', warn: 'clock' }

/** One line of the activity: what happened, who did it, Stripe's event. */
function eventView(t: T, ev: Event, o: OrderDetail): { tone: EvTone; title: string; detail?: string } {
  const amount = (minor?: number) => money(minor ?? 0, ev.currency || o.currency, 'always')
  switch (ev.kind) {
    case 'checkout_started':
      return { tone: 'info', title: t(po.evCheckout), detail: ev.page ? t(ev.ticked ? po.evCheckoutFromTicked : po.evCheckoutFrom, { page: bareUrl(ev.page) }) : undefined }
    case 'paid':
      return { tone: 'ok', title: t(po.evPaid), detail: ev.stripe ? t(po.evStripe, { event: ev.stripe }) : undefined }
    case 'delivery_sent':
      return { tone: 'ok', title: t(po.evSent), detail: ev.template ? t(po.evSentDetail, { template: ev.template, version: ev.version ?? 0, provider: ev.provider ?? '' }) : undefined }
    case 'delivery_delivered':
      return {
        tone: 'ok',
        title: t(po.evDelivered),
        detail: ev.template
          ? ev.seconds
            ? t(po.evDeliveredDetail, { template: ev.template, version: ev.version ?? 0, provider: ev.provider ?? '', seconds: decimal(ev.seconds, 1) })
            : t(po.evDeliveredDetailNoTime, { template: ev.template, version: ev.version ?? 0, provider: ev.provider ?? '' })
          : undefined,
      }
    case 'delivery_failed':
      return {
        tone: 'bad',
        title: t(po.evFailed),
        detail: ev.template && ev.provider ? t(po.evFailedDetail, { template: ev.template, provider: ev.provider, note: ev.note ?? '' }) : ev.note ? t(po.evFailedNote, { note: ev.note }) : undefined,
      }
    case 'downloaded':
      if ((ev.count ?? 1) > 1) return { tone: 'ok', title: t(po.evDownloadedMany, { count: ev.count ?? 0 }), detail: t(po.evDownloadedManyDetail, { date: dayMonthDate(ev.at), time: clock(ev.at) }) }
      return {
        tone: 'ok',
        title: t(po.evDownloaded),
        // The link's count, unless the download was on an earlier link (the current one then shows 0).
        detail: ev.file
          ? (ev.used ?? 0) >= 1
            ? t(po.evDownloadedDetail, { used: ev.used ?? 0, allowed: ev.allowed ?? 0, file: ev.file, size: fileSize(t, ev.bytes ?? 0) })
            : t(po.evDownloadedFile, { file: ev.file, size: fileSize(t, ev.bytes ?? 0) })
          : undefined,
      }
    case 'refunded':
      return {
        tone: 'mut',
        title: t(po.evRefunded),
        detail: ev.stripe
          ? ev.by && ev.byStripe
            ? t(po.evRefundedDetailBy, { event: ev.stripe, amount: amount(ev.amount), name: ev.by })
            : t(po.evRefundedDetail, { event: ev.stripe, amount: amount(ev.amount) })
          : amount(ev.amount),
      }
    case 'dispute_opened':
      return {
        tone: 'bad',
        title: t(po.evDisputeOpened),
        detail: ev.stripe
          ? ev.respondBy
            ? t(po.evDisputeOpenedDetail, { event: ev.stripe, reason: ev.reason ?? '', date: day(ev.respondBy) })
            : t(po.evDisputeOpenedDetailNoDue, { event: ev.stripe, reason: ev.reason ?? '' })
          : undefined,
      }
    case 'dispute_closed':
      return { tone: ev.outcome === 'won' ? 'ok' : 'bad', title: ev.outcome === 'won' ? t(po.evDisputeWon) : t(po.evDisputeLost), detail: ev.stripe ? t(po.evStripe, { event: ev.stripe }) : undefined }
    case 'access_revoked':
      if (ev.reason === 'dispute') return { tone: 'mut', title: t(po.evRevokedDispute), detail: t(po.evBySystem) }
      if (ev.reason === 'refund') return { tone: 'mut', title: t(po.evRevoked), detail: t(po.evRevokedRefund) }
      return { tone: 'mut', title: t(po.evRevoked), detail: ev.by ? t(po.evBy, { name: ev.by }) : t(po.evBySystem) }
    case 'access_restored':
      return { tone: 'ok', title: t(po.evRestored), detail: ev.by ? t(po.evBy, { name: ev.by }) : undefined }
    case 'new_link':
      return { tone: 'ok', title: ev.by ? t(po.evNewLink, { name: ev.by }) : t(po.evNewLinkPlain), detail: ev.email ? t(po.evNewLinkDetail, { email: ev.email }) : undefined }
    case 'buyer_asked':
      return { tone: 'warn', title: t(po.evBuyerAsked), detail: ev.note || t(po.evBuyerAskedPage) }
  }
  return { tone: 'mut', title: ev.kind }
}

function EventRow({ ev, o, now }: { ev: Event; o: OrderDetail; now: Date }) {
  const t = useT()
  const v = eventView(t, ev, o)
  return (
    <li className="pr-ev">
      <span className={`ic ${v.tone}`} aria-hidden="true">
        <Icon name={evIcon[v.tone]} size={14} strokeWidth={v.tone === 'ok' ? 2 : 1.9} />
      </span>
      <div className="tx">
        <b>{v.title}</b>
        {v.detail ? <span className="k">{v.detail}</span> : null}
      </div>
      <time className="tm" dateTime={ev.at}>
        {eventTime(ev.at, now)}
      </time>
    </li>
  )
}

/** The panel's actions, or why there are none (orders:manage). */
function Actions({ o }: { o: OrderDetail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canOrders = useCan('orders:manage')
  const [dialog, setDialog] = useState<'link' | 'revoke'>()
  const restore = $api.useMutation('post', `${ORDER}/restore`)
  const p = o.payment
  const path = { workspace: slug, order: o.id }
  const queryKey = $api.queryOptions('get', ORDER, { params: { path } }).queryKey
  const done = (next: OrderDetail) => {
    qc.setQueryData(queryKey, next)
    invalidateProducts(qc)
  }
  if (!canOrders) {
    return (
      <div className="pr-opacts">
        <IconNote tone="n">
          <T msg={o.helpers.length ? po.ordersNoteAsk : po.ordersNote} values={{ mono, helpers: orList(o.helpers) }} />
        </IconNote>
      </div>
    )
  }
  const revoked = o.download.state === 'revoked' || o.download.state === 'revoked_auto'
  const disputeOpen = o.status === 'disputed' && p.disputeStatus === 'open'
  return (
    <div className="pr-opacts">
      {disputeOpen && p.disputeUrl ? (
        <ButtonAnchor href={p.disputeUrl} size="sm" iconRight="external">
          {t(po.respondInStripe)}
        </ButtonAnchor>
      ) : null}
      {revoked ? null : (
        <Button size="sm" disabledReason={o.canSendLink ? undefined : t(po.revokedNoLink)} onClick={() => setDialog('link')}>
          {t(po.sendLink)}
        </Button>
      )}
      {o.canRevoke ? (
        <Button size="sm" variant="danger" onClick={() => setDialog('revoke')}>
          {t(po.revoke)}
        </Button>
      ) : null}
      {revoked ? (
        <Button
          size="sm"
          loading={restore.isPending}
          disabledReason={o.canRestore ? undefined : t(po.noRestoreRefunded)}
          onClick={() =>
            restore.mutate(
              { params: { path } },
              {
                onSuccess: (next) => {
                  done(next)
                  toast({ kind: 'success', title: t(po.restoredToast, { number: o.number }), detail: t(po.restoredDetail) })
                },
                onError: (e) => toast({ kind: 'error', title: t(po.restoreFailed), detail: errorMessage(e) }),
              },
            )
          }
        >
          {t(po.restore)}
        </Button>
      ) : null}
      {!disputeOpen && p.url ? (
        <ButtonAnchor href={p.url} size="sm" iconRight="external">
          {o.status === 'refunded' || o.test ? t(po.openInStripe) : t(po.refundInStripe)}
        </ButtonAnchor>
      ) : null}
      {dialog === 'link' ? <NewLinkDialog o={o} onClose={() => setDialog(undefined)} onDone={done} /> : null}
      <RevokeDialog o={o} open={dialog === 'revoke'} onClose={() => setDialog(undefined)} onDone={done} />
    </div>
  )
}

/** "Send a new link to d.okoro@example.org?" (ProductsStates). The contact profile's Purchases card opens it too (contact.tsx). */
export function NewLinkDialog({ o, onClose, onDone }: { o: OrderDetail; onClose: () => void; onDone: (next: OrderDetail) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const send = $api.useMutation('post', `${ORDER}/new-link`)
  const product = $api.useQuery('get', PRODUCT, { params: { path: { workspace: slug, product: o.productId } } })
  const asked = o.activity.some((e) => e.kind === 'buyer_asked')
  const [by, setBy] = useState(asked ? t(po.requestedByBuyer) : '')
  const delivery = (product.data?.live ?? product.data?.draft)?.delivery
  const d = o.delivery
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && !send.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(po.sendEyebrow, { number: o.number })}</span>}
      title={t(po.sendTitle, { email: o.buyerEmail })}
      footer={
        <>
          <span className="k mr-auto">{t(po.sendAudit)}</span>
          <Button onClick={onClose} disabled={send.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={send.isPending}
            busyLabel={t(po.sending)}
            onClick={() =>
              send.mutate(
                { params: { path: { workspace: slug, order: o.id } }, body: { requestedBy: by.trim() || undefined } },
                {
                  onSuccess: (next) => {
                    onDone(next)
                    onClose()
                    const sent = next.delivery?.sentAt ?? next.download.newLinkAt ?? new Date().toISOString()
                    toast({
                      kind: 'success',
                      title: t(po.sent, { email: next.buyerEmail }),
                      detail: next.download.expiresAt ? t(po.sentDetail, { time: clock(sent), date: day(next.download.expiresAt) }) : undefined,
                    })
                  },
                  onError: (e) => toast({ kind: 'error', title: t(po.sendFailed), detail: errorMessage(e) }),
                },
              )
            }
          >
            {t(po.sendConfirm)}
          </Button>
        </>
      }
    >
      <span>{delivery ? t(po.sendText, { days: delivery.linkDays, downloads: delivery.downloads }) : t(po.sendTextShort)}</span>
      <dl className="kv">
        <dt>{t(po.sendOrder)}</dt>
        <dd>{t(po.sendOrderValue, { number: o.number, product: o.productName, amount: money(o.amountCents, o.currency, 'always') })}</dd>
        {d?.template ? (
          <>
            <dt>{t(po.sendTemplate)}</dt>
            <dd>{o.route ? t(po.sendTemplateValue, { template: d.template, version: d.version, route: o.route }) : t(po.sendTemplateNoRoute, { template: d.template, version: d.version })}</dd>
          </>
        ) : null}
        <dt>
          <label htmlFor="pr-requested-by">{t(po.sendRequestedBy)}</label>
        </dt>
        <dd>
          <Input id="pr-requested-by" className="w-full" value={by} maxLength={200} placeholder={t(po.sendRequestedPlaceholder)} onChange={(e) => setBy(e.target.value)} />
        </dd>
      </dl>
    </Dialog>
  )
}

function RevokeDialog({ o, open, onClose, onDone }: { o: OrderDetail; open: boolean; onClose: () => void; onDone: (next: OrderDetail) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const revoke = $api.useMutation('post', `${ORDER}/revoke`)
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(v) => !v && onClose()}
      tone="bad"
      badge={t(po.revokeBadge)}
      title={t(po.revokeTitle, { number: o.number })}
      description={t(po.revokeText, { email: o.buyerEmail })}
      confirmLabel={t(po.revokeConfirm)}
      busyLabel={t(po.revoking)}
      onConfirm={async () => {
        const next = await revoke.mutateAsync({ params: { path: { workspace: slug, order: o.id } } })
        onDone(next)
        toast({ kind: 'success', title: t(po.revokedToast, { number: o.number }) })
      }}
    />
  )
}
