import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select, Toggle } from '@/components/ui/Field'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, orList } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { EARN, EarnPage } from '../EarnPage'
import { day, money } from '../format'
import { DeliveryDialog, OptionDialog } from './GiftDialogs'
import { GiftIcon, giftWhen } from './parts'
import { mg } from './messages'
import './gifts.css'

type View = Schemas['GiftsView']
type Row = Schemas['GiftRow']
type Option = Schemas['GiftOption']
type Filter = 'all' | 'scheduled' | 'not_redeemed' | 'redeemed' | 'expired' | 'refunded'
type TT = ReturnType<typeof useT>

type PageDialog = { kind: 'option'; option?: Option } | { kind: 'delivery'; row: Row }

/** Copies text; false when the browser refused. */
async function copy(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms)
    return () => clearTimeout(id)
  }, [value, ms])
  return v
}

/**
 * Earn › Gifts (EarnGifts): the numbers, the gift options and the gift
 * page, the gifts with their delivery and redemption, the gift emails and
 * the rules readers see.
 */
export function GiftsPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', `${EARN}/gifts`, path)
  const [dialog, setDialog] = useState<PageDialog>()
  const v = q.data
  const actions = v?.page.on && v.connected ? (
    <ButtonAnchor href={v.page.url} iconRight="external">
      {t(mg.openPage)}
    </ButtonAnchor>
  ) : null
  let body: ReactNode
  if (q.error) {
    body = (
      <ErrorPanel title={t(mg.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(mg.retry)}</Button>}>
        {t(mg.errorText, { error: errorMessage(q.error) })}
      </ErrorPanel>
    )
  } else if (!v) {
    body = (
      <div className="flex flex-col gap-4" aria-busy="true">
        <div className="gf-stats">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="card gf-stat">
              <Skeleton w="60%" h={12} />
              <Skeleton w="40%" h={22} className="mt-2" />
            </div>
          ))}
        </div>
        <div className="card pad">
          <Skeleton w="100%" h={180} />
        </div>
      </div>
    )
  } else {
    body = <GiftsBody v={v} setDialog={setDialog} />
  }
  return (
    <EarnPage tab="gifts" title={t(mg.title)} actions={actions}>
      <div className="flex flex-col gap-5">{body}</div>
      {dialog?.kind === 'option' && v ? <OptionDialog v={v} option={dialog.option} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'delivery' ? <DeliveryDialog row={dialog.row} onClose={() => setDialog(undefined)} /> : null}
    </EarnPage>
  )
}

function GiftsBody({ v, setDialog }: { v: View; setDialog: (d: PageDialog) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const short = v.options[0]?.short || v.tiers[0]?.short || ''
  const ending = v.emails.find((e) => e.email === 'gift_ending')
  return (
    <>
      {!v.canEdit || !v.canManage ? <ReadOnlyNote v={v} /> : null}
      {v.ending ? (
        <div className="note w" role="note">
          <Icon name="alertCircle" size={16} />
          <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
            <span className="font-bold">{t(mg.endingTitle, { template: ending?.template ?? 'gift-ending' })}</span>
            <div>
              {v.ending.inPerson
                ? t(mg.endingPerson, { date: day(v.ending.due), buyer: v.ending.buyer })
                : t(mg.endingTo, { date: day(v.ending.due), buyer: v.ending.buyer, recipient: v.ending.recipient })}
              <div className="mt-2 flex flex-wrap gap-2">
                {v.ending.templateId ? (
                  <ButtonLink size="sm" to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: v.ending.templateId }}>
                    {t(mg.openTemplate, { template: ending?.template ?? 'gift-ending' })}
                  </ButtonLink>
                ) : null}
                <ButtonLink size="sm" to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'emails' }}>
                  {t(mg.memberEmails)}
                </ButtonLink>
              </div>
            </div>
          </div>
        </div>
      ) : null}
      {!v.connected ? (
        <div className="note i" role="note">
          <Icon name="info" size={16} />
          <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
            <span className="font-bold">{t(mg.noProviderTitle)}</span>
            <div>
              {t(mg.noProviderText, { provider: v.provider })}
              {v.canEdit ? (
                <div className="mt-2 flex flex-wrap gap-2">
                  <ButtonLink size="sm" variant="primary" to="/w/$workspace/earn/settings" params={{ workspace: slug }}>
                    {t(mg.connect, { provider: v.provider })}
                  </ButtonLink>
                  {v.tiers.length ? (
                    <Button size="sm" onClick={() => setDialog({ kind: 'option' })}>
                      {t(mg.newGiftOption)}
                    </Button>
                  ) : null}
                </div>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
      {v.connected && !v.page.on && v.options.length ? (
        <div className="note n" role="note">
          <Icon name="info" size={16} />
          <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
            <span className="font-bold">{t(mg.pageOffNoteTitle)}</span>
            <div>{t(mg.pageOffNote, { host: v.page.host })}</div>
          </div>
        </div>
      ) : null}
      <Numbers v={v} />
      {v.needsLook.length ? <NeedsLook v={v} setDialog={setDialog} /> : null}
      {v.options.length ? (
        <OptionsCard v={v} setDialog={setDialog} short={short} />
      ) : (
        <EmptyOptions v={v} short={short} onNew={() => setDialog({ kind: 'option' })} />
      )}
      <GiftsTable v={v} setDialog={setDialog} />
      <div className="gf-two-cards">
        <EmailsCard v={v} short={short} />
        <RulesCard v={v} short={short} />
      </div>
    </>
  )
}

function ReadOnlyNote({ v }: { v: View }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  const helpers = v.helpers.length ? orList(v.helpers) : ''
  const msg = !v.canEdit && !v.canManage ? mg.readOnlyBoth : !v.canEdit ? mg.readOnlyEdit : mg.readOnlyManage
  return (
    <div className="note n" role="note">
      <span style={{ color: 'var(--ink-2)', display: 'inline-flex', flex: 'none', marginTop: 1 }}>
        <Icon name="info" size={16} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(mg.readOnlyTitle, { name: user?.name || user?.email || '', role })}</span>
        <div>
          <T msg={msg} values={{ helpers: helpers || t(mg.helpersFallback) }} />
        </div>
      </div>
    </div>
  )
}

function Numbers({ v }: { v: View }) {
  const t = useT()
  const n = v.numbers
  const split = n.byMonths.map((x) => t(mg.soldSplit, { count: x.count, months: x.months })).join(', ')
  const next = n.nextAt ? giftWhen(n.nextAt, n.nextTimezone) : undefined
  return (
    <div className="gf-stats">
      <div className="card gf-stat">
        <span className="k">{t(mg.soldLabel)}</span>
        <span className="v">{count(n.sold)}</span>
        <span className="s">{n.sold ? t(mg.soldSub, { amount: money(n.soldCents, v.currency, 'never'), split }) : t(mg.soldNone)}</span>
      </div>
      <div className="card gf-stat">
        <span className="k">{t(mg.redeemedLabel)}</span>
        <span className="v">{t(mg.redeemedValue, { redeemed: count(n.redeemed), delivered: count(n.delivered) })}</span>
        <span className="s">{n.delivered ? t(mg.redeemedSub, { waiting: n.waiting }) : t(mg.redeemedNone)}</span>
      </div>
      <div className="card gf-stat">
        <span className="k">{t(mg.scheduledLabel)}</span>
        <span className="v">{count(n.scheduled)}</span>
        <span className="s">{next ? t(mg.scheduledSub, next) : t(mg.scheduledNone)}</span>
      </div>
      <div className="card gf-stat">
        <span className="k">{t(mg.accessLabel)}</span>
        <span className="v">{count(n.withAccess)}</span>
        <span className="s">{t(mg.accessSub)}</span>
      </div>
    </div>
  )
}

function OptionsCard({ v, setDialog, short }: { v: View; setDialog: (d: PageDialog) => void; short: string }) {
  const t = useT()
  return (
    <section className="card" aria-labelledby="gf-opts-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="gf-opts-h">
            {t(mg.optionsTitle, { count: v.options.length })}
          </h2>
          <span className="k">{t(mg.optionsCaption, { tier: short })}</span>
        </div>
        {v.canEdit ? (
          <Button size="sm" icon="plus" onClick={() => setDialog({ kind: 'option' })}>
            {t(mg.newOption)}
          </Button>
        ) : null}
      </div>
      <div className="gf-opts">
        <div className="gf-cards">
          {v.options.map((o) => (
            <div key={o.id} className="gf-ocard">
              <div className="flex items-center justify-between gap-2">
                <span className="k">{o.tier}</span>
                <GiftIcon className="ic" />
              </div>
              <div className="flex flex-wrap items-baseline gap-2.5">
                <span className="gf-olen">{t(mg.length, { months: o.months })}</span>
                <span className="gf-oprice">{money(o.amountCents, o.currency, 'auto')}</span>
              </div>
              <span className="k text-ink-nav">{t(mg.perMonth, { price: money(Math.round(o.amountCents / o.months), o.currency, 'auto') })}</span>
              <div className="gf-ofoot">
                <span className="k">{o.sold ? t(mg.optionSold, { count: o.sold, amount: money(o.soldCents, o.currency, 'auto') }) : t(mg.optionSoldNone)}</span>
                {v.canEdit ? (
                  <Button size="sm" onClick={() => setDialog({ kind: 'option', option: o })} aria-label={`${t(mg.edit)} · ${t(mg.oName, { tier: o.tier, months: o.months })}`}>
                    {t(mg.edit)}
                  </Button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
        <PageSettings v={v} />
      </div>
    </section>
  )
}

function EmptyOptions({ v, short, onNew }: { v: View; short: string; onNew: () => void }) {
  const t = useT()
  return (
    <section className="card flex flex-col items-center gap-3 px-6 py-10 text-center" aria-labelledby="gf-empty-h">
      <span className="inline-flex h-14 w-14 items-center justify-center rounded-full" style={{ background: 'var(--purple-bg)', color: 'var(--purple-fg)' }}>
        <GiftIcon size={24} />
      </span>
      <h2 className="m-0 text-[17px] font-semibold" id="gf-empty-h">
        {t(mg.emptyTitle, { tier: short || t(mg.title) })}
      </h2>
      <p className="m-0 max-w-[560px] text-ink-2">{v.tiers.length ? t(mg.emptyText) : t(mg.noTiers)}</p>
      {v.canEdit && v.tiers.length ? (
        <Button variant="primary" icon="plus" onClick={onNew}>
          {t(mg.newGiftOption)}
        </Button>
      ) : null}
      {v.tiers.length ? <span className="k">{t(mg.emptyHint)}</span> : null}
    </section>
  )
}

function PageSettings({ v }: { v: View }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const save = $api.useMutation('put', `${EARN}/gifts/page`)
  const p = v.page
  const set = (on: boolean, linkMonths: number) =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { on, linkMonths: linkMonths as 3 | 6 | 12 } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts`] })
          toast({ kind: 'success', title: t(mg.pageSaved), detail: t(mg.auditedNote) })
        },
        onError: (e) => toast({ kind: 'error', title: t(mg.pageTitle), detail: errorMessage(e) }),
      },
    )
  const changed = p.changedAt && p.changedBy ? giftWhen(p.changedAt, undefined) : undefined
  return (
    <div className="gf-set">
      <div className="flex items-start justify-between gap-3">
        <span className="flex flex-col gap-0.5">
          <span className="font-semibold">{t(mg.pageTitle)}</span>
          <span className="k">{p.on ? t(mg.pageOn) : t(mg.pageOff)}</span>
        </span>
        <Toggle label={t(mg.pageTitle)} hideLabel checked={p.on} disabled={!v.canEdit || save.isPending} onChange={(e) => set(e.target.checked, p.linkMonths)} />
      </div>
      <div className="gf-linkrow">
        <input className="in mono" value={p.url} readOnly aria-label={t(mg.pageLink)} />
        <Button size="sm" icon="copy" onClick={() => void copy(p.url).then((ok) => ok && toast({ kind: 'success', title: t(mg.copied), detail: p.url }))}>
          {t(mg.copy)}
        </Button>
        <ButtonAnchor size="sm" href={p.url} iconRight="external">
          {t(mg.open)}
        </ButtonAnchor>
      </div>
      <label className="lbl">
        {t(mg.linkMonths)}
        <Select className="w-[160px]" value={String(p.linkMonths)} disabled={!v.canEdit || save.isPending} onChange={(e) => set(p.on, Number(e.target.value))}>
          {[3, 6, 12].map((m) => (
            <option key={m} value={m}>
              {t(mg.linkMonthsOption, { months: m })}
            </option>
          ))}
        </Select>
        <span className="k font-medium">{t(mg.linkHint)}</span>
      </label>
      <span className="k">{t(mg.messagesKept)}</span>
      {changed && !p.on ? <span className="k">{t(mg.turnedOff, { name: p.changedBy ?? '', ...changed })}</span> : null}
    </div>
  )
}

const filters: { value: Filter; label: keyof typeof mg; count: keyof Schemas['GiftCounts'] }[] = [
  { value: 'all', label: 'fAll', count: 'all' },
  { value: 'scheduled', label: 'fScheduled', count: 'scheduled' },
  { value: 'not_redeemed', label: 'fNotRedeemed', count: 'notRedeemed' },
  { value: 'redeemed', label: 'fRedeemed', count: 'redeemed' },
  { value: 'expired', label: 'fExpired', count: 'expired' },
  { value: 'refunded', label: 'fRefunded', count: 'refunded' },
]

function GiftsTable({ v, setDialog }: { v: View; setDialog: (d: PageDialog) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [filter, setFilter] = useState<Filter>('all')
  const [search, setSearch] = useState('')
  const q = useDebounced(search.trim(), 250)
  // The page goes back to the first when the filter or the search changes.
  const key = `${filter}|${q}`
  const [paged, setPaged] = useState({ key, page: 1 })
  const page = paged.key === key ? paged.page : 1
  const setPage = (next: (p: number) => number) => setPaged({ key, page: next(page) })
  const list = $api.useQuery(
    'get',
    `${EARN}/gifts/list`,
    { params: { path: { workspace: slug }, query: { status: filter, q: q || undefined, page } } },
    { placeholderData: keepPreviousData },
  )
  const l = list.data
  const from = l ? (l.page - 1) * l.pageSize + 1 : 0
  const to = l ? from + l.items.length - 1 : 0
  return (
    <section className="card" aria-labelledby="gf-list-h">
      <div className="gf-sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="gf-list-h">
            {t(mg.giftsTitle)}
          </h2>
          <span className="k">{v.canManage ? t(mg.giftsCaption, { host: v.page.host }) : t(mg.giftsCaptionRo)}</span>
        </div>
        <label className="gf-search">
          <Icon name="search" size={14} />
          <input aria-label={t(mg.searchLabel)} placeholder={t(mg.search)} value={search} onChange={(e) => setSearch(e.target.value)} />
        </label>
      </div>
      <div className="gf-fchips" role="group" aria-label={t(mg.filterLabel)}>
        {filters.map((f) => (
          <button key={f.value} type="button" className="gf-fchip" aria-pressed={filter === f.value} onClick={() => setFilter(f.value)}>
            {t(mg[f.label])} <b>{l ? count(l.counts[f.count]) : '…'}</b>
          </button>
        ))}
      </div>
      {list.error ? (
        <div className="pad">
          <ErrorPanel title={t(mg.errorTitle)} actions={<Button onClick={() => void list.refetch()}>{t(mg.retry)}</Button>}>
            {errorMessage(list.error)}
          </ErrorPanel>
        </div>
      ) : !l ? (
        <div className="pad" aria-busy="true">
          <Skeleton w="100%" h={160} />
        </div>
      ) : l.items.length === 0 ? (
        <p className="pad k m-0">{l.counts.all === 0 && !q ? t(mg.none, { host: v.page.host }) : t(mg.noneFilter)}</p>
      ) : (
        <>
          <div className="relative overflow-x-auto max-md:hidden">
            <table className="tbl gf-tbl" aria-labelledby="gf-list-h">
              <thead>
                <tr>
                  <th scope="col">{t(mg.colBought)}</th>
                  <th scope="col">{t(mg.colFrom)}</th>
                  <th scope="col">{t(mg.colTo)}</th>
                  <th scope="col">{t(mg.colGift)}</th>
                  <th scope="col" className="r">
                    {t(mg.colPaid)}
                  </th>
                  <th scope="col">{t(mg.colDelivery)}</th>
                  <th scope="col">{t(mg.colStatus)}</th>
                  {v.canManage ? (
                    <th scope="col">
                      <span className="vh">{t(mg.colActions)}</span>
                    </th>
                  ) : null}
                </tr>
              </thead>
              <tbody>
                {l.items.map((r) => (
                  <tr key={r.id}>
                    <td className="nw">{day(r.boughtAt)}</td>
                    <td className="nw">
                      <span className="font-semibold">{r.buyer}</span>
                    </td>
                    <td>
                      <To r={r} />
                    </td>
                    <td className="nw">{t(mg.giftLine, { tier: r.short, months: r.months })}</td>
                    <td className="r num">{money(r.amountCents, r.currency, 'always')}</td>
                    <td className="nw">
                      <Delivery r={r} t={t} />
                    </td>
                    <td>
                      <Status r={r} provider={v.provider} t={t} />
                    </td>
                    {v.canManage ? (
                      <td className="act">
                        <RowMenu r={r} provider={v.provider} setDialog={setDialog} />
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {/* Phones: a row per gift. */}
          <ul className="m-0 hidden list-none p-0 max-md:block" aria-labelledby="gf-list-h">
            {l.items.map((r) => (
              <li key={r.id} className="gf-row">
                <span className="lt">
                  <b className="font-semibold">
                    {r.buyer} → {r.inPerson ? t(mg.inPerson) : r.recipientName || r.recipientEmail}
                  </b>
                  {v.canManage ? <RowMenu r={r} provider={v.provider} setDialog={setDialog} /> : null}
                </span>
                <span className="k text-ink-2">
                  {t(mg.giftLine, { tier: r.short, months: r.months })} · {money(r.amountCents, r.currency, 'always')} · {day(r.boughtAt)}
                </span>
                <Status r={r} provider={v.provider} t={t} />
              </li>
            ))}
          </ul>
          <div className="gf-foot">
            <span className="k">{t(mg.showing, { from, to, total: l.total })}</span>
            <div className="flex gap-2">
              <Button size="sm" disabled={l.page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                {t(mg.previous)}
              </Button>
              <Button size="sm" disabled={to >= l.total} onClick={() => setPage((p) => p + 1)}>
                {t(mg.next)}
              </Button>
            </div>
          </div>
        </>
      )}
    </section>
  )
}

function To({ r }: { r: Row }) {
  const t = useT()
  if (r.inPerson) return <span>{t(mg.inPerson)}</span>
  if (r.recipientName) {
    return (
      <div className="gf-two">
        <span>{r.recipientName}</span>
        <span className="mono k">{r.recipientEmail}</span>
      </div>
    )
  }
  return <span className="mono text-[12.5px]">{r.recipientEmail}</span>
}

function Delivery({ r, t }: { r: Row; t: TT }) {
  if (r.status === 'scheduled' && r.deliverAt) {
    const w = giftWhen(r.deliverAt, r.timezone)
    return (
      <>
        {w.date}, {w.time}
      </>
    )
  }
  const at = r.deliveredAt ?? r.deliverAt
  if (r.inPerson && at) {
    return (
      <div className="gf-two">
        <span>{giftWhen(at, r.timezone).date}</span>
        <span className="k">{t(mg.linkTo, { name: r.buyer.split(' ')[0] ?? r.buyer })}</span>
      </div>
    )
  }
  if (at) return <>{giftWhen(at, r.timezone).date}</>
  return <span className="k">{t(mg.atOnce)}</span>
}

/** A gift's status and its line; `look` is the needs-a-look card's (a warning). */
function Status({ r, provider, t, look }: { r: Row; provider: string; t: TT; look?: boolean }) {
  let badge: ReactNode
  let line: string | undefined
  switch (r.status) {
    case 'scheduled':
      badge = <Badge tone="info">{t(mg.stScheduled)}</Badge>
      break
    case 'not_redeemed':
      badge = look ? <Badge tone="warn">{t(mg.stNotRedeemedShort)}</Badge> : <Badge tone="neutral">{t(mg.stNotRedeemed)}</Badge>
      if (r.remindedAt) line = t(mg.stReminded, { date: day(r.remindedAt) })
      else if (r.remindAt) line = t(mg.stRemindOn, { date: day(r.remindAt) })
      break
    case 'redeemed': {
      badge = <Badge tone="ok">{t(mg.stRedeemed)}</Badge>
      const date = r.redeemedAt ? day(r.redeemedAt) : ''
      const ends = r.accessEnds ? day(r.accessEnds) : ''
      line = r.redeemedBy ? t(mg.stRedeemedBy, { date, email: r.redeemedBy, ends }) : t(mg.stRedeemedLine, { date, ends })
      break
    }
    case 'expired':
      badge = <Badge tone="neutral">{t(mg.stExpired)}</Badge>
      if (r.linkExpiresAt) line = t(mg.stExpiredLine, { date: day(r.linkExpiresAt) })
      break
    default:
      badge = <Badge tone="neutral">{t(mg.stRefunded)}</Badge>
      if (r.refundedAt) line = t(mg.stRefundedLine, { provider, date: day(r.refundedAt) })
  }
  const notSent = r.status === 'not_redeemed' && r.emailStatus && r.emailStatus !== 'sent'
  return (
    <span className="gf-st">
      {badge}
      {line ? <span className="k">{line}</span> : null}
      {notSent ? <span className="k text-warn-fg">{t(mg.emailNotSent, { reason: r.emailStatus })}</span> : null}
    </span>
  )
}

function RowMenu({ r, provider, setDialog }: { r: Row; provider: string; setDialog: (d: PageDialog) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const resend = $api.useMutation('post', `${EARN}/gifts/{id}/resend`)
  const link = $api.useMutation('post', `${EARN}/gifts/{id}/link`)
  const path = { params: { path: { workspace: slug, id: r.id } } }
  const to = r.inPerson ? t(mg.inPerson) : r.recipientName || r.recipientEmail
  const scheduled = r.status === 'scheduled'
  const linkOk = scheduled || r.status === 'not_redeemed'
  const when = scheduled && r.deliverAt ? giftWhen(r.deliverAt, r.timezone) : undefined
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts/list`] })
  }
  return (
    <Menu
      label={t(mg.menuLabel, { name: to })}
      className="gf-menu"
      trigger={
        <Button size="sm" iconOnly aria-label={t(mg.actionsFor, { name: r.buyer })}>
          <Icon name="more" size={14} />
        </Button>
      }
    >
      <MenuItem disabled={!scheduled} onSelect={() => setDialog({ kind: 'delivery', row: r })}>
        <span className="mi">
          <Icon name="calendar" size={14} />
          <span>{t(mg.mChange)}</span>
          {when ? <span className="k">{`${when.date}, ${when.time}`}</span> : null}
        </span>
      </MenuItem>
      <MenuItem
        disabled={r.status !== 'not_redeemed' || resend.isPending}
        onSelect={() =>
          resend.mutate(path, {
            onSuccess: () => {
              refresh()
              toast({ kind: 'success', title: t(mg.resent, { email: r.inPerson ? r.buyerEmail : r.recipientEmail }), detail: t(mg.auditedNote) })
            },
            onError: (e) => toast({ kind: 'error', title: t(mg.mResend), detail: errorMessage(e) }),
          })
        }
      >
        <span className="mi">
          <Icon name="mail" size={14} />
          <span>{t(mg.mResend)}</span>
          {scheduled ? <span className="k">{t(mg.mNotSent)}</span> : null}
        </span>
      </MenuItem>
      <MenuItem
        disabled={!linkOk || link.isPending}
        onSelect={() =>
          link.mutate(path, {
            onSuccess: (res) =>
              void copy(res.url).then((ok) => toast(ok ? { kind: 'success', title: t(mg.linkCopied), detail: t(mg.auditedNote) } : { kind: 'error', title: t(mg.mCopy), detail: res.url })),
            onError: (e) => toast({ kind: 'error', title: t(mg.mCopy), detail: errorMessage(e) }),
          })
        }
      >
        <span className="mi">
          <Icon name="link" size={14} />
          <span>{t(mg.mCopy)}</span>
          <span className="k">{t(mg.mAudited)}</span>
        </span>
      </MenuItem>
      {r.paymentUrl ? (
        <>
          <MenuSeparator />
          <MenuItem asChild>
            <a href={r.paymentUrl} className="mi">
              <Icon name="external" size={14} />
              <span>{t(mg.mPayment, { provider })}</span>
            </a>
          </MenuItem>
          <span className="k gf-mnote block">{t(mg.mRefundNote, { provider })}</span>
        </>
      ) : null}
    </Menu>
  )
}

function NeedsLook({ v, setDialog }: { v: View; setDialog: (d: PageDialog) => void }) {
  const t = useT()
  return (
    <div className="gf-looks">
      {v.needsLook.map((r) => (
        <LookCard key={r.id} r={r} v={v} setDialog={setDialog} t={t} />
      ))}
    </div>
  )
}

function LookCard({ r, v, t }: { r: Row; v: View; setDialog: (d: PageDialog) => void; t: TT }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const resend = $api.useMutation('post', `${EARN}/gifts/{id}/resend`)
  const link = $api.useMutation('post', `${EARN}/gifts/{id}/link`)
  const path = { params: { path: { workspace: slug, id: r.id } } }
  const gift = t(mg.giftLine, { tier: r.short, months: r.months })
  const amount = money(r.amountCents, r.currency, 'always')
  const to = r.inPerson ? t(mg.inPerson) : r.recipientName || r.recipientEmail
  const refunded = r.status === 'refunded'
  const when = r.deliverAt ? giftWhen(r.deliverAt, r.timezone) : undefined
  let sub = t(mg.lookBought, { gift, amount, date: day(r.boughtAt) })
  if (r.deliveredAt && !refunded) sub = t(mg.lookDelivered, { gift, amount, date: day(r.deliveredAt) })
  else if (refunded && when && !r.deliveredAt) sub = t(mg.lookScheduled, { gift, amount, date: day(r.boughtAt), when: `${when.date}, ${when.time}` })
  const until = r.linkExpiresAt ? day(r.linkExpiresAt) : ''
  return (
    <section className={`card gf-look${refunded ? '' : ' gf-look-warn'}`}>
      <div className="gf-look-h">
        <span className="font-bold">{t(mg.lookTitle, { buyer: r.buyer, to })}</span>
        <span className="k">{sub}</span>
        <div>
          <Status r={r} provider={v.provider} t={t} look />
        </div>
      </div>
      <div className="gf-look-b">
        <span className="text-ink-nav">
          {refunded
            ? when && !r.deliveredAt
              ? t(mg.lookRefunded, { provider: v.provider, date: r.refundedAt ? day(r.refundedAt) : '', when: when.date })
              : t(mg.lookRefundedSent, { provider: v.provider, date: r.refundedAt ? day(r.refundedAt) : '' })
            : r.remindedAt
              ? t(mg.lookReminded, { date: day(r.remindedAt), until })
              : t(mg.lookRemindSoon, { until })}
        </span>
        <div className="flex flex-wrap gap-2">
          {refunded ? (
            r.paymentUrl ? (
              <ButtonAnchor size="sm" href={r.paymentUrl} icon="external">
                {t(mg.mPayment, { provider: v.provider })}
              </ButtonAnchor>
            ) : null
          ) : v.canManage ? (
            <>
              <Button
                size="sm"
                loading={resend.isPending}
                onClick={() =>
                  resend.mutate(path, {
                    onSuccess: () => {
                      void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts`] })
                      void qc.invalidateQueries({ queryKey: ['get', `${EARN}/gifts/list`] })
                      toast({ kind: 'success', title: t(mg.resent, { email: r.inPerson ? r.buyerEmail : r.recipientEmail }), detail: t(mg.auditedNote) })
                    },
                    onError: (e) => toast({ kind: 'error', title: t(mg.mResend), detail: errorMessage(e) }),
                  })
                }
              >
                {t(mg.mResend)}
              </Button>
              <Button
                size="sm"
                icon="link"
                loading={link.isPending}
                onClick={() =>
                  link.mutate(path, {
                    onSuccess: (res) => void copy(res.url).then((ok) => toast(ok ? { kind: 'success', title: t(mg.linkCopied), detail: t(mg.auditedNote) } : { kind: 'error', title: t(mg.mCopy), detail: res.url })),
                    onError: (e) => toast({ kind: 'error', title: t(mg.mCopy), detail: errorMessage(e) }),
                  })
                }
              >
                {t(mg.mCopy)}
              </Button>
            </>
          ) : null}
        </div>
      </div>
    </section>
  )
}

function EmailsCard({ v, short }: { v: View; short: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const desc = (e: Schemas['GiftEmail']) => {
    switch (e.email) {
      case 'gift_purchased':
        return t(mg.eReceipt)
      case 'gift_received':
        return t(mg.eReceived)
      default:
        return e.status === 'live' ? t(mg.eEndingLive, { tier: short }) : t(mg.eEnding, { tier: short })
    }
  }
  const from = v.fromEmail ? (v.fromName ? `${v.fromName} <${v.fromEmail}>` : v.fromEmail) : ''
  const kind = v.categoryKind === 'essential' || !v.categoryKind ? t(mg.kindEssential) : v.categoryKind
  return (
    <section className="card" aria-labelledby="gf-em-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="gf-em-h">
            {t(mg.emailsTitle)}
          </h2>
          <span className="k">{t(mg.emailsCaption)}</span>
        </div>
      </div>
      <div className="-mt-px">
        {v.emails.map((e) => (
          <div key={e.email} className="gf-erow">
            <span className="gf-eic">
              <Icon name="mail" size={14} />
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-[3px]">
              <div className="flex flex-wrap items-center gap-2">
                {e.templateId ? (
                  <Link
                    to="/w/$workspace/transactional/templates/$templateId"
                    params={{ workspace: slug, templateId: e.templateId }}
                    className="font-semibold text-ink no-underline"
                  >
                    {t(mg.eName, { email: e.email })}
                  </Link>
                ) : (
                  <span className="font-semibold">{t(mg.eName, { email: e.email })}</span>
                )}
                <span className="mono k">{e.template}</span>
                {e.status === 'live' ? (
                  <Badge tone="ok">{t(mg.vLive, { n: e.liveVersion })}</Badge>
                ) : e.status === 'draft' ? (
                  <Badge tone="warn">{t(mg.vDraft, { n: e.draftVersion || 1 })}</Badge>
                ) : (
                  <Badge tone="bad">{t(mg.vMissing)}</Badge>
                )}
              </div>
              <span className="k text-ink-2">{desc(e)}</span>
            </div>
          </div>
        ))}
      </div>
      <div className="gf-foot">
        <span className="k flex-[1_1_300px]">
          {from
            ? t(mg.emailsFoot, { category: v.categoryName || 'Memberships', kind, from })
            : t(mg.emailsFootNoFrom, { category: v.categoryName || 'Memberships', kind })}
        </span>
        <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'emails' }} className="lnk">
          {t(mg.allEmails)}
        </Link>
      </div>
    </section>
  )
}

function RulesCard({ v, short }: { v: View; short: string }) {
  const t = useT()
  const lists = v.listNames.length ? v.listNames.join(', ') : ''
  return (
    <section className="card" aria-labelledby="gf-rules-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="gf-rules-h">
            {t(mg.rulesTitle)}
          </h2>
          <span className="k">{t(mg.rulesCaption)}</span>
        </div>
      </div>
      <div className="pad">
        <dl className="kv">
          <dt>{t(mg.rLink)}</dt>
          <dd>{t(mg.rLinkText, { months: v.page.linkMonths })}</dd>
          <dt>{t(mg.rCard)}</dt>
          <dd>{lists ? t(mg.rCardText, { lists }) : t(mg.rCardTextNoList)}</dd>
          <dt>{t(mg.rAccess)}</dt>
          <dd>{t(mg.rAccessText, { tier: short })}</dd>
          <dt>{t(mg.rPaid)}</dt>
          <dd>{t(mg.rPaidText, { tier: short })}</dd>
          <dt>{t(mg.rRefunds)}</dt>
          <dd>{t(mg.rRefundsText, { provider: v.provider })}</dd>
        </dl>
      </div>
    </section>
  )
}
