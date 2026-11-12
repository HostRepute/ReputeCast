import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, SwitchInput, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, orList, pct, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { copyText } from '../EarnPage'
import { money } from '../format'
import { ActivitySection, BlockPreview, Head, PanelError, PanelLoading, Section, Sheet, complete } from './BookingPanel'
import { AmountInput, CancelDialog, amountInput, parseAmount, useRefresh } from './dialogs'
import { msp } from './messages'
import { Initials, NoticeNote, Two, WS, type SlotItem } from './parts'
import { mr } from './requests.messages'

type Item = Schemas['SponsorRequestItem']
type Detail = Schemas['SponsorRequestDetail']
type SelfServe = Schemas['SponsorSelfServe']

/** "news.acme.io/sponsor": the media kit's address without its scheme. */
export function hostOf(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** "Fabrikam Studio" → "FS", when the request isn't filed under a library sponsor yet. */
function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  return (
    words
      .slice(0, 2)
      .map((w) => [...w][0] ?? '')
      .join('')
      .toUpperCase() || '?'
  )
}

const when = (d: string) => `${dayMonthDate(d)}, ${clock(d)}`

/** The issue's line under a request: open, held for another sponsor, held until the pay-by date, booked… */
function issueState(t: ReturnType<typeof useT>, r: Item): string {
  switch (r.status) {
    case 'requested':
      if (r.issueState !== 'held') return t(mr.issueOpen)
      if (r.heldBy) return t(mr.issueHeldFor, { name: r.heldBy })
      return r.heldUntil ? t(mr.issueHeldUntil, { date: shortDate(r.heldUntil) }) : t(mr.issueBooked)
    case 'awaiting_payment':
      return r.paymentDueAt ? t(mr.issueHeldUntil, { date: shortDate(r.paymentDueAt) }) : ''
    case 'booked':
      return t(mr.issueBooked)
    case 'declined':
      return t(mr.issueDeclined)
    case 'expired':
      return t(mr.issueExpired)
  }
  return ''
}

/** Requested, Awaiting payment · pay by …, Booked · paid by Stripe …, Declined, Expired. */
export function RequestStatus({ r }: { r: Item }) {
  const t = useT()
  switch (r.status) {
    case 'requested':
      return <Badge tone="warn">{t(mr.stRequested)}</Badge>
    case 'awaiting_payment':
      return <Two badge={<Badge tone="info">{t(mr.stAwaiting)}</Badge>} detail={r.paymentDueAt ? t(mr.payBy, { date: shortDate(r.paymentDueAt) }) : undefined} />
    case 'booked': {
      const paid = r.paidOn ? (r.paidSource === 'stripe' ? t(mr.paidStripe, { date: dayMonthDate(r.paidOn) }) : t(mr.paidHand, { date: dayMonthDate(r.paidOn) })) : undefined
      return <Two badge={<Badge tone="ok">{t(mr.stBooked)}</Badge>} detail={paid} />
    }
    case 'declined':
      return <Two badge={<Badge tone="neutral">{t(mr.stDeclined)}</Badge>} detail={r.closedAt ? t(mr.declinedOn, { date: dayMonthDate(r.closedAt) }) : undefined} />
    case 'expired':
      return <Two badge={<Badge tone="neutral">{t(mr.stExpired)}</Badge>} detail={r.closedAt ? t(mr.expiredOn, { date: dayMonthDate(r.closedAt) }) : undefined} />
  }
  return <Badge tone="neutral">{t(mr.stCanceled)}</Badge>
}

const newestFirst = (a: Item, b: Item) => new Date(b.submittedAt).getTime() - new Date(a.submittedAt).getTime()

/**
 * The Requests tab (EarnSponsorRequests): waiting for a person, waiting for
 * the payment, and decided in the last 30 days. A row opens its panel.
 */
export function RequestsTab({ selected, onOpen }: { selected?: string; onOpen: (r: Item) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-requests`, { params: { path: { workspace: slug } } })
  if (q.error && !q.data) return <RequestsError error={q.error} onRetry={() => void q.refetch()} />
  if (!q.data) return <RequestsLoading />
  const { items, pageUrl, selfServe, canManage } = q.data
  if (!items.length) return <NoRequests pageUrl={pageUrl} selfServe={selfServe} />
  const groups: { key: string; label: string; rows: Item[] }[] = [
    { key: 'waiting', label: t(mr.grpWaiting, { count: items.filter((r) => r.status === 'requested').length }), rows: items.filter((r) => r.status === 'requested').sort(newestFirst) },
    { key: 'payment', label: t(mr.grpPayment, { count: items.filter((r) => r.status === 'awaiting_payment').length }), rows: items.filter((r) => r.status === 'awaiting_payment').sort(newestFirst) },
    {
      key: 'decided',
      label: t(mr.grpDecided, { count: items.filter((r) => r.status !== 'requested' && r.status !== 'awaiting_payment').length }),
      rows: items.filter((r) => r.status !== 'requested' && r.status !== 'awaiting_payment').sort(newestFirst),
    },
  ].filter((g) => g.rows.length)
  return (
    <>
      <div className="max-md:hidden">
        <div className="overflow-x-auto">
          <table className="tbl sp-tbl" aria-label={t(mr.listLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(mr.colRequest)}</th>
                <th scope="col" className="max-lg:hidden">
                  {t(mr.colSlot)}
                </th>
                <th scope="col">{t(mr.colIssue)}</th>
                <th scope="col" className="r">
                  {t(mr.colPrice)}
                </th>
                <th scope="col">{t(mr.colSubmitted)}</th>
                <th scope="col">{t(mr.colStatus)}</th>
                <th scope="col">
                  <span className="vh">{t(mr.open)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <RequestGroup key={g.key} label={g.label}>
                  {g.rows.map((r) => (
                    <tr key={r.id} className={selected === r.id ? 'sel' : undefined}>
                      <td className="nw">
                        <div className="flex items-center gap-2.5">
                          <Initials initials={r.sponsor?.initials || initialsOf(r.company)} logo={r.sponsor?.logoUrl} />
                          <div className="sp-col start">
                            <span className="font-semibold">{r.company}</span>
                            <span className="k">{r.name}</span>
                          </div>
                        </div>
                      </td>
                      <td className="nw max-lg:hidden">{r.slotName}</td>
                      <td>
                        <div className="sp-col start">
                          <span className="nw">{t(mr.issueLine, { issue: r.issue, date: shortDate(r.issueOn) })}</span>
                          <span className="k">{issueState(t, r)}</span>
                        </div>
                      </td>
                      <td className="r num">{money(r.priceCents, r.currency)}</td>
                      <td className="nw">
                        <span className="ea-num">{when(r.submittedAt)}</span>
                      </td>
                      <td>
                        <RequestStatus r={r} />
                      </td>
                      <td className="act">
                        <Button size="sm" onClick={() => onOpen(r)} aria-label={`${r.status === 'requested' && canManage ? t(mr.review) : t(mr.open)} · ${r.company}`}>
                          {r.status === 'requested' && canManage ? t(mr.review) : t(mr.open)}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </RequestGroup>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="hidden max-md:flex max-md:flex-col">
        {groups.flatMap((g) => g.rows).map((r) => (
          <button key={r.id} type="button" className="sp-lst" onClick={() => onOpen(r)}>
            <span className="sp-lst-top">
              <b className="font-semibold">{r.company}</b>
              <span className="ea-num font-semibold">{money(r.priceCents, r.currency)}</span>
            </span>
            <span className="k" style={{ color: 'var(--ink-2)' }}>
              {t(mr.phoneLine, { issue: t(mr.issueLine, { issue: r.issue, date: shortDate(r.issueOn) }), slot: r.slotName })}
            </span>
            <span className="sp-lst-badges">
              <RequestStatus r={r} />
              {/* An approved request's date is its pay-by date: the row keeps to one line of badges. */}
              {r.status === 'awaiting_payment' ? null : <span className="k">{t(mr.sentOn, { date: dayMonthDate(r.submittedAt) })}</span>}
            </span>
          </button>
        ))}
      </div>
      <div className="sp-foot">
        <span className="k" style={{ flex: '1 1 360px' }}>
          {t(mr.foot)}
        </span>
        {selfServe && pageUrl ? <KitLink url={pageUrl} /> : null}
      </div>
    </>
  )
}

function RequestGroup({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <tr className="grp">
        <td colSpan={7}>{label}</td>
      </tr>
      {children}
    </>
  )
}

/** "Open the media kit": a link that opens the public page in a new tab. */
function KitLink({ url }: { url: string }) {
  const t = useT()
  return (
    <a className="sp-lnk" href={url} target="_blank" rel="noopener noreferrer">
      <Icon name="external" size={14} />
      {t(mr.openKit)}
    </a>
  )
}

/** "Open the media kit" beside the tabs while the Requests tab shows. */
export function OpenKitButton() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-requests`, { params: { path: { workspace: slug } } })
  if (!q.data?.selfServe || !q.data.pageUrl) return null
  return (
    <ButtonAnchor size="sm" href={q.data.pageUrl} icon="external">
      {t(mr.openKit)}
    </ButtonAnchor>
  )
}

function NoRequests({ pageUrl, selfServe }: { pageUrl: string; selfServe: boolean }) {
  const t = useT()
  return (
    <div className="sp-empty">
      <span className="ico" aria-hidden="true">
        <Icon name="mail" size={26} />
      </span>
      <span className="tt">{t(mr.emptyTitle)}</span>
      <span className="ds">{t(mr.emptyText, { host: hostOf(pageUrl) })}</span>
      {selfServe && pageUrl ? (
        <Button variant="primary" icon="copy" onClick={() => void copyText(pageUrl, t, toast)}>
          {t(mr.copyKit)}
        </Button>
      ) : null}
    </div>
  )
}

function RequestsLoading() {
  return (
    <div className="flex flex-col" aria-busy="true">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="sp-skrow">
          <Skeleton w={150} h={14} />
          <Skeleton w={110} h={14} />
          <Skeleton w="100%" h={10} className="flex-1" />
          <Skeleton w={90} h={20} r={999} />
          <Skeleton w={60} h={28} />
        </div>
      ))}
    </div>
  )
}

function RequestsError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const ref = problemReference(error)
  return (
    <div className="flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <span style={{ color: 'var(--bad-strong)' }}>
        <Icon name="alertCircle" size={28} />
      </span>
      <span className="text-[16px] font-bold">{t(mr.loadError)}</span>
      <span className="k max-w-[420px]">
        {errorMessage(error)} {t(mr.loadErrorText)}
      </span>
      <Button variant="primary" onClick={onRetry}>
        {t(mr.retry)}
      </Button>
      {ref ? <span className="k mono">{ref}</span> : null}
    </div>
  )
}

/**
 * Self-serve booking (EarnSponsorRequests): the switch that publishes the
 * media kit, what the kit shows, and the self-serve slots.
 */
export function SelfServeCard({ slots, onEditSlot }: { slots: SlotItem[]; onEditSlot: (s: SlotItem) => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-page`, { params: { path: { workspace: slug } } })
  if (!q.data) {
    return q.error ? null : (
      <section className="card" aria-busy="true">
        <div className="pad flex flex-col gap-3">
          <Skeleton w={180} h={18} />
          <Skeleton w="70%" h={12} />
          <Skeleton w="100%" h={120} />
        </div>
      </section>
    )
  }
  return <SelfServeForm key={q.data.updatedAt ?? 'new'} d={q.data} slots={slots} onEditSlot={onEditSlot} />
}

function SelfServeForm({ d, slots, onEditSlot }: { d: SelfServe; slots: SlotItem[]; onEditSlot: (s: SlotItem) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefresh()
  const [kit, setKit] = useState(d.mediaKit)
  const [contact, setContact] = useState(d.contactEmail)
  const [noStripe, setNoStripe] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', `${WS}/sponsor-page`)
  const ro = d.canManage ? undefined : t(mr.cantDecide)
  const on = d.selfServe
  const host = hostOf(d.pageUrl)
  const stats = d.stats[0]
  const dirty = JSON.stringify(kit) !== JSON.stringify(d.mediaKit) || contact !== d.contactEmail
  const done = (x: SelfServe) => {
    qc.setQueryData(['get', `${WS}/sponsor-page`, { params: { path: { workspace: slug } } }], x)
    refresh()
  }
  const fail = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const f: Record<string, string> = {}
      for (const x of e.problem.errors) f[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
      setErrors(f)
      return
    }
    toast({ kind: 'error', title: errorMessage(e) })
  }
  // The switch saves at once, with the media kit as last saved.
  const turn = (next: boolean) =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { selfServe: next, mediaKit: d.mediaKit, contactEmail: d.contactEmail } },
      {
        onSuccess: (x) => {
          setNoStripe(false)
          done(x)
          toast({ kind: 'success', title: next ? t(mr.turnedOn) : t(mr.turnedOff) })
        },
        onError: (e) => {
          if (e instanceof ApiError && e.status === 409) setNoStripe(true)
          else fail(e)
        },
      },
    )
  const submit = () =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { selfServe: on, mediaKit: { ...kit, about: kit.about.trim(), audience: kit.audience.trim() }, contactEmail: contact.trim() } },
      {
        onSuccess: (x) => {
          setErrors({})
          done(x)
          toast({ kind: 'success', title: t(mr.saved) })
        },
        onError: fail,
      },
    )
  const selfSlots = slots.filter((s) => s.selfServe)
  const numbers: { key: 'subscribers' | 'openRate' | 'clickRate'; label: string; hint: string }[] = [
    { key: 'subscribers', label: t(mr.numSubscribers), hint: stats ? t(mr.numSubscribersHint, { value: count(stats.subscribers) }) : t(mr.numNoSends) },
    { key: 'openRate', label: t(mr.numOpen), hint: stats?.openRate != null ? t(mr.numOpenHint, { value: pct(stats.openRate, 1) }) : t(mr.numNoSends) },
    { key: 'clickRate', label: t(mr.numClick), hint: stats?.clickRate != null ? t(mr.numClickHint, { value: pct(stats.clickRate, 1) }) : t(mr.numNoSends) },
  ]
  return (
    <section className="card" aria-labelledby="sp-selfserve">
      <div className="sec">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-0.5">
          <h2 className="h2" id="sp-selfserve">
            {t(mr.ssTitle)}
          </h2>
          <span className="k">{t(mr.ssHint)}</span>
        </div>
        <div className="row gap-2">
          <Badge tone={on ? 'ok' : 'neutral'}>{on ? t(mr.ssOn) : t(mr.ssOff)}</Badge>
          <SwitchInput checked={on} aria-label={t(mr.ssToggle)} disabled={Boolean(ro) || save.isPending} title={ro} onChange={(e) => turn(e.target.checked)} />
        </div>
      </div>
      {noStripe ? (
        <div className="pad pb-0">
          <NoticeNote
            tone="warn"
            title={t(mr.ssNoStripeTitle)}
            actions={
              <Link className="btn sm" to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }}>
                {t(mr.connectStripe)}
              </Link>
            }
          >
            {t(mr.ssNoStripeText)}
          </NoticeNote>
        </div>
      ) : null}
      {!on ? (
        <div className="pad flex flex-col gap-2">
          <span>
            <T msg={mr.ssOffText} values={{ host, mono: (c) => <span className="mono">{c}</span> }} />
          </span>
          <span className="k">{t(mr.ssOffNote)}</span>
        </div>
      ) : (
        <div className="pad flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="lbl" id="sp-kit-url">
              {t(mr.kitPage)}
            </span>
            <div className="flex flex-wrap items-center gap-2">
              <input className="in mono min-w-0 flex-[1_1_260px]" readOnly value={d.pageUrl} aria-labelledby="sp-kit-url" />
              <Button size="sm" icon="copy" onClick={() => void copyText(d.pageUrl, t, toast)}>
                {t(mr.copy)}
              </Button>
              <ButtonAnchor size="sm" href={d.pageUrl} iconRight="external">
                {t(mr.openShort)}
              </ButtonAnchor>
            </div>
            <span className="k">{t(mr.kitPageHint)}</span>
          </div>
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0" disabled={Boolean(ro)}>
            <legend className="mb-2 p-0 font-semibold">{t(mr.numbers)}</legend>
            <div className="flex flex-wrap gap-2">
              {numbers.map((n) => (
                <label key={n.key} className={`opt px-3 py-2.5 ${kit[n.key] ? 'on' : ''}`} style={{ flex: '1 1 200px' }} aria-label={n.label}>
                  <input type="checkbox" checked={kit[n.key]} onChange={(e) => setKit({ ...kit, [n.key]: e.target.checked })} />
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className="font-semibold">{n.label}</span>
                    <span className="k">{n.hint}</span>
                  </span>
                </label>
              ))}
            </div>
            {stats ? <span className="k">{t(mr.numbersNote, { list: stats.listName })}</span> : null}
          </fieldset>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(280px,100%),1fr))] gap-3">
            <Field label={t(mr.about)} hint={t(mr.counter, { count: [...kit.about].length, max: 600 })} error={errors['mediaKit.about']}>
              <Textarea rows={4} maxLength={600} value={kit.about} readOnly={Boolean(ro)} onChange={(e) => setKit({ ...kit, about: e.target.value })} />
            </Field>
            <Field label={t(mr.audience)} hint={t(mr.counter, { count: [...kit.audience].length, max: 300 })} error={errors['mediaKit.audience']}>
              <Textarea rows={4} maxLength={300} value={kit.audience} readOnly={Boolean(ro)} onChange={(e) => setKit({ ...kit, audience: e.target.value })} />
            </Field>
          </div>
          <Field label={t(mr.contact)} hint={t(mr.contactHint)} error={errors.contactEmail}>
            <Input type="email" className="max-w-[420px]" value={contact} readOnly={Boolean(ro)} onChange={(e) => setContact(e.target.value)} />
          </Field>
          <div className="flex flex-col gap-1.5">
            <span className="lbl">{t(mr.slotsTitle)}</span>
            {selfSlots.map((s) => (
              <div key={s.id} className="sp-ssrow">
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span>
                    <T msg={mr.slotLine} values={{ slot: s.name, list: s.listName, price: money(s.priceCents, s.currency), b: (c) => <b className="font-semibold">{c}</b> }} />
                  </span>
                  <span className="k">{t(mr.slotTerms, { lead: s.leadDays, pay: s.payDays })}</span>
                </span>
                {d.canManage ? (
                  <Button size="sm" onClick={() => onEditSlot(s)} aria-label={`${t(mr.editSlot)} · ${s.name}`}>
                    {t(mr.editSlot)}
                  </Button>
                ) : null}
              </div>
            ))}
            <span className="k">{t(mr.slotsHint)}</span>
          </div>
        </div>
      )}
      <div className="ea-foot">
        <span className="k">{t(mr.ssFoot)}</span>
        {on ? (
          <Button size="sm" variant="primary" disabledReason={ro} disabled={!dirty} loading={save.isPending} onClick={submit}>
            {t(mr.save)}
          </Button>
        ) : null}
      </div>
    </section>
  )
}

/** A request's panel with its own link (?tab=requests&request=…). */
export function RequestPanel({ id, onClose }: { id: string; onClose: () => void }) {
  const t = useT()
  return (
    <Sheet label={t(mr.loading)} onClose={onClose}>
      <RequestBody id={id} onClose={onClose} />
    </Sheet>
  )
}

/** The panel's content: the request, the creative, what approving does; once approved, its payment and activity. */
export function RequestBody({ id, onClose }: { id: string; onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-requests/{id}`, { params: { path: { workspace: slug, id } } })
  if (q.error) return <PanelError error={q.error} onRetry={() => void q.refetch()} onClose={onClose} />
  if (!q.data) return <PanelLoading onClose={onClose} />
  return <RequestView d={q.data} onClose={onClose} reload={async () => (await q.refetch()).data} />
}

function RequestView({ d, onClose, reload }: { d: Detail; onClose: () => void; reload: () => Promise<Detail | undefined> }) {
  const t = useT()
  const [dialog, setDialog] = useState<'approve' | 'decline' | 'cancel'>()
  const ro = !d.canManage
  const requested = d.status === 'requested'
  const awaiting = d.status === 'awaiting_payment'
  const eyebrow = awaiting ? t(mr.eyebrowBooking, { list: d.listName }) : ro ? t(mr.eyebrowReadOnly, { list: d.listName }) : t(mr.eyebrowRequest, { list: d.listName })
  const title = t(mr.panelTitle, { issue: d.issue, date: shortDate(d.issueOn), slot: d.slotName })
  const amount = money(d.priceCents, d.currency)
  const badges = requested ? (
    <Two badge={<Badge tone="warn">{t(mr.stRequested)}</Badge>} detail={t(mr.sentLine, { when: when(d.submittedAt) })} />
  ) : (
    <RequestStatus r={d} />
  )
  return (
    <>
      <Head eyebrow={eyebrow} title={title} badges={badges} onClose={onClose} />
      <div className="ea-sheet-body">
        {ro && !requested ? <ReadOnlyNote helpers={d.helpers} /> : null}
        <Section title={t(mr.secRequest)}>
          <dl className="sp-kv">
            <dt>{t(mr.company)}</dt>
            <dd>
              <span className="inline-flex flex-wrap items-center gap-2">
                <Initials initials={d.initials || initialsOf(d.company)} size="sm" />
                <span className="font-semibold">{d.company}</span>
                {d.website ? <span className="k">{d.website}</span> : null}
              </span>
            </dd>
            <dt>{t(mr.contactRow)}</dt>
            <dd>
              {d.name}
              <span className="k">
                {' · '}
                <span className="mono">{d.email}</span>
              </span>
            </dd>
            {awaiting ? null : (
              <>
                <dt>{t(mr.slot)}</dt>
                <dd>{t(mr.slotList, { slot: d.slotName, list: d.listName })}</dd>
                <dt>{t(mr.issue)}</dt>
                <dd>
                  {t(mr.issueLine, { issue: d.issue, date: shortDate(d.issueOn) })}
                  {requested ? <span className="k"> · {issueState(t, d)}</span> : null}
                </dd>
              </>
            )}
            <dt>{t(mr.price)}</dt>
            <dd>
              <span className="ea-num">{amount}</span>
              {!awaiting && d.slots[0] && d.slots[0].id === d.slotId && d.slots[0].priceCents === d.priceCents ? <span className="k"> · {t(mr.slotsPrice)}</span> : null}
            </dd>
            {awaiting ? null : (
              <>
                <dt>{t(mr.sent)}</dt>
                <dd>
                  {when(d.submittedAt)}
                  <span className="k"> · {t(mr.fromKit)}</span>
                </dd>
              </>
            )}
          </dl>
        </Section>
        {d.notes && !awaiting ? (
          <Section title={t(mr.secNote)}>
            <blockquote className="sp-quote">{d.notes}</blockquote>
          </Section>
        ) : null}
        {requested || d.status === 'declined' || d.status === 'expired' ? <CreativeSection d={d} /> : null}
        {d.status === 'declined' && d.declinedReason ? (
          <Section title={t(mr.secReason)}>
            <blockquote className="sp-quote">{d.declinedReason}</blockquote>
          </Section>
        ) : null}
        {requested ? (
          <Section title={t(mr.secIfApprove)}>
            <span className="k" style={{ color: 'var(--ink-nav)' }}>
              {t(mr.ifApprove, { company: d.company, amount, days: d.payDays })}
            </span>
          </Section>
        ) : null}
        {awaiting ? (
          <Section title={t(mr.secPayment)} aside={<Badge tone="info">{t(mr.stAwaiting)}</Badge>}>
            <span className="flex flex-col gap-0.5">
              <span>
                <T
                  msg={mr.awaitingLine}
                  values={{ amount: <b className="ea-num font-semibold">{amount}</b>, date: d.paymentDueAt ? shortDate(d.paymentDueAt) : '' }}
                />
              </span>
              {d.approvedAt ? <span className="k">{t(mr.payVia, { name: d.name, date: dayMonthDate(d.approvedAt) })}</span> : null}
            </span>
            {d.paymentDueAt ? (
              <span className="k" style={{ color: 'var(--ink-nav)' }}>
                {t(mr.payExpire, { date: shortDate(d.paymentDueAt), issue: d.issue })}
              </span>
            ) : null}
          </Section>
        ) : null}
        {requested ? null : <ActivitySection b={d} />}
      </div>
      {requested ? (
        <div className="ea-sheet-foot">
          <Button size="sm" variant="danger" disabledReason={ro ? t(mr.cantDecide) : undefined} onClick={() => setDialog('decline')}>
            {t(mr.decline)}
          </Button>
          <span className="k ml-auto">{ro ? t(mr.cantDecide) : t(mr.recorded)}</span>
          <Button size="sm" variant="primary" disabledReason={ro ? t(mr.cantDecide) : undefined} onClick={() => setDialog('approve')}>
            {t(mr.approve)}
          </Button>
        </div>
      ) : awaiting && !ro ? (
        <div className="ea-sheet-foot">
          <Button size="sm" variant="danger" onClick={() => setDialog('cancel')}>
            {t(mr.cancelBooking)}
          </Button>
          <span className="k ml-auto">{t(mr.recorded)}</span>
        </div>
      ) : null}
      {dialog === 'approve' ? <ApproveDialog d={d} reload={reload} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'decline' ? <DeclineDialog d={d} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'cancel' ? (
        <CancelDialog
          target={{ bookingId: d.id, issue: d.issue, sponsor: d.company, status: 'awaiting_payment', amountCents: d.priceCents, currency: d.currency }}
          onClose={() => setDialog(undefined)}
          onDone={onClose}
        />
      ) : null}
    </>
  )
}

/** Creative: as the Sponsor block shows it; the files stay hidden until approved. */
function CreativeSection({ d }: { d: Detail }) {
  const t = useT()
  const ok = complete(d.creative)
  const files = d.files.map((f) => t(mr.fileLine, { name: f.name, size: fileSize(f.sizeBytes) }))
  return (
    <Section title={t(mr.secCreative)} aside={ok ? <Badge tone="ok">{t(mr.complete)}</Badge> : <Badge tone="warn">{t(mr.incomplete)}</Badge>}>
      <span className="k">{t(mr.creativeHint)}</span>
      <BlockPreview creative={d.creative} sponsor={d.company} initials={d.initials || initialsOf(d.company)} logo={d.logo} label={d.label} />
      <dl className="sp-kv">
        {d.creative.link ? (
          <>
            <dt>{t(mr.link)}</dt>
            <dd className="min-w-0 break-all">
              <span className="mono">{d.creative.link}</span>
              <span className="k"> · {t(mr.utmNote)}</span>
            </dd>
          </>
        ) : null}
        {files.length ? (
          <>
            <dt>{t(mr.files)}</dt>
            <dd>{files.join(' · ')}</dd>
          </>
        ) : null}
      </dl>
    </Section>
  )
}

/** "18 KB", "640 KB", "1.2 MB". */
function fileSize(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

/** "Read-only · Aisha Bello, Viewer": who can approve and decline. */
function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  return (
    <NoticeNote tone="neutral" title={t(msp.readOnlyTitle, { name: user?.name || user?.email || '', role })}>
      {helpers.length ? t(mr.readOnlyText, { helpers: orList(helpers) }) : t(mr.readOnlyNoHelpers)}
    </NoticeNote>
  )
}

/** Days from now until a date, at least 0. */
function daysFromNow(iso: string): number {
  return Math.max(0, Math.round((new Date(iso).getTime() - Date.now()) / 86_400_000))
}

/** Approve: perhaps for another slot, issue or price; holds the issue until the pay-by date and emails the pay link. */
function ApproveDialog({ d, reload, onClose }: { d: Detail; reload: () => Promise<Detail | undefined>; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefresh()
  const [slotId, setSlotId] = useState(d.slots.some((s) => s.id === d.slotId) ? d.slotId : (d.slots[0]?.id ?? ''))
  const slot = d.slots.find((s) => s.id === slotId)
  const firstFree = (s: typeof slot) => s?.issues.find((i) => !i.taken)?.number ?? s?.issues[0]?.number ?? 0
  const [issue, setIssue] = useState(() => (slot?.issues.some((i) => i.number === d.issue) ? d.issue : firstFree(slot)))
  const currency = slot?.currency ?? d.currency
  const [price, setPrice] = useState(amountInput(d.priceCents, currency).replace(/\.00$/, ''))
  const [taken, setTaken] = useState<number>()
  const [error, setError] = useState<string>()
  const approve = $api.useMutation('post', `${WS}/sponsor-requests/{id}/approve`)
  const chosen = slot?.issues.find((i) => i.number === issue)
  const submit = () => {
    const cents = parseAmount(price, currency)
    if (cents === undefined || cents <= 0) return setError(t(mr.approvePriceHint))
    setError(undefined)
    approve.mutate(
      { params: { path: { workspace: slug, id: d.id } }, body: { slotId, issue, priceCents: cents } },
      {
        onSuccess: (x) => {
          qc.setQueryData(['get', `${WS}/sponsor-requests/{id}`, { params: { path: { workspace: slug, id: d.id } } }], x)
          refresh()
          toast({ kind: 'success', title: t(mr.approvedToast, { company: d.company, date: x.paymentDueAt ? shortDate(x.paymentDueAt) : '' }) })
          onClose()
        },
        onError: async (e) => {
          // Another booking took the issue meanwhile: say who holds it, and keep the dialog open to choose another.
          if (e instanceof ApiError && e.status === 409) {
            const fresh = await reload()
            if (fresh?.slots.find((s) => s.id === slotId)?.issues.find((i) => i.number === issue)?.taken) {
              setTaken(issue)
              refresh()
              return
            }
          }
          toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  const takenIssue = taken !== undefined ? slot?.issues.find((i) => i.number === taken) : undefined
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mr.approveEyebrow, { company: d.company })}</span>}
      title={t(mr.approveTitle, { company: d.company })}
      className="max-w-[560px]"
      footer={
        <>
          <span className="k mr-auto">{t(mr.approveRecorded)}</span>
          <Button onClick={onClose}>{t(mr.cancel)}</Button>
          <Button variant="primary" loading={approve.isPending} disabled={!slot || !issue || taken === issue} onClick={submit}>
            {t(mr.approveSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {taken !== undefined ? (
          <NoticeNote tone="warn" title={takenIssue?.heldBy && takenIssue.heldUntil ? t(mr.takenBanner, { holder: takenIssue.heldBy, issue: taken, date: shortDate(takenIssue.heldUntil) }) : t(mr.takenBannerNoDate, { issue: taken })} />
        ) : null}
        <span style={{ color: 'var(--ink-nav)' }}>{t(mr.approveText, { company: d.company })}</span>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-3">
          <Field label={t(mr.approveSlot)}>
            <Select
              value={slotId}
              onChange={(e) => {
                const next = d.slots.find((s) => s.id === e.target.value)
                setSlotId(e.target.value)
                setIssue(firstFree(next))
                setTaken(undefined)
              }}
            >
              {d.slots.map((s) => (
                <option key={s.id} value={s.id}>
                  {t(mr.slotList, { slot: s.name, list: s.listName })}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t(mr.approveIssue)} error={taken !== undefined && taken === issue ? t(mr.issueTaken, { issue }) : undefined}>
            <Select value={String(issue)} onChange={(e) => setIssue(Number(e.target.value))}>
              {(slot?.issues ?? []).map((i) => (
                <option key={i.number} value={i.number}>
                  {t(mr.issueLine, { issue: i.number, date: shortDate(i.issueOn) })}
                </option>
              ))}
            </Select>
          </Field>
        </div>
        <Field label={t(mr.approvePrice)} hint={t(mr.approvePriceHint)} error={error}>
          <AmountInput currency={currency} value={price} onChange={setPrice} />
        </Field>
        <dl className="sp-kv">
          <dt>{t(mr.approvePayBy)}</dt>
          <dd>{chosen ? t(mr.payByValue, { date: shortDate(chosen.paymentDueAt), days: daysFromNow(chosen.paymentDueAt) }) : '–'}</dd>
          <dt>{t(mr.approveCreative)}</dt>
          <dd>{complete(d.creative) ? t(mr.creativeCame) : t(mr.incomplete)}</dd>
        </dl>
      </div>
    </Dialog>
  )
}

/** Decline with a reason the sponsor is emailed. */
function DeclineDialog({ d, onClose }: { d: Detail; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefresh()
  const [reason, setReason] = useState('')
  const decline = $api.useMutation('post', `${WS}/sponsor-requests/{id}/decline`)
  const submit = () =>
    decline.mutate(
      { params: { path: { workspace: slug, id: d.id } }, body: { reason: reason.trim() } },
      {
        onSuccess: (x) => {
          qc.setQueryData(['get', `${WS}/sponsor-requests/{id}`, { params: { path: { workspace: slug, id: d.id } } }], x)
          refresh()
          toast({ kind: 'success', title: t(mr.declinedToast, { company: d.company }) })
          onClose()
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mr.approveEyebrow, { company: d.company })}</span>}
      title={t(mr.declineTitle, { company: d.company })}
      className="max-w-[540px]"
      footer={
        <>
          <span className="k mr-auto">{t(mr.approveRecorded)}</span>
          <Button onClick={onClose}>{t(mr.cancel)}</Button>
          <Button variant="danger-primary" loading={decline.isPending} disabled={!reason.trim()} onClick={submit}>
            {t(mr.declineSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>{t(mr.declineText, { name: d.name })}</span>
        <Field label={t(mr.reason)} hint={t(mr.reasonHint)}>
          <Textarea rows={4} maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  )
}
