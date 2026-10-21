import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, count, dayMonthYear, orList, pct, shortDate, weekdayName } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { EarnPage } from '../EarnPage'
import { day, money } from '../format'
import { me } from '../messages'
import { BookingPanel, IssuePanel } from './BookingPanel'
import { CancelDialog, ExtendHoldDialog, LinkCampaignDialog, PaymentDialog, SlotDialog, SponsorDialog, type CancelTarget, type PaymentTarget } from './dialogs'
import { msp } from './messages'
import {
  BookingCell,
  CreativeCell,
  Initials,
  MoneyIcon,
  NoticeNote,
  PaymentCell,
  WS,
  campaignLine,
  ends,
  isToday,
  price,
  sponsorKeys,
  type IssueRow,
  type Overview,
  type SlotItem,
  type SponsorItem,
} from './parts'
import './sponsors.css'

export type SponsorsTab = 'bookings' | 'sponsors' | 'slots'
export type SponsorsRange = 'around' | 'upcoming' | 'recent' | 'all'

export interface SponsorsSearch {
  tab?: SponsorsTab
  slot?: string
  range?: SponsorsRange
  /** The booking shown in the panel. */
  booking?: string
  /** An open issue of the slot shown in the panel. */
  issue?: number
}

type Notice = Schemas['SponsorNotice']

type PageDialog =
  | { kind: 'sponsor'; sponsor?: SponsorItem }
  | { kind: 'slot'; slot?: SlotItem }
  | { kind: 'payment'; target: PaymentTarget }
  | { kind: 'cancel'; target: CancelTarget }
  | { kind: 'extend'; bookingId: string; issue: number; sponsor: string; holdUntil?: string; issueOn: string }
  | { kind: 'link'; bookingId: string; issue: number }

/**
 * Earn › Sponsors (EarnSponsors): the numbers, what needs a person, and the
 * Bookings · Sponsors · Slots tabs. A booking or an open issue opens in a
 * panel with its own link.
 */
export function SponsorsPage({ search, onSearch }: { search: SponsorsSearch; onSearch: (next: SponsorsSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug, me: wme } = useWorkspace()
  const canView = wme.permissions.includes('earn:view')
  const q = $api.useQuery(
    'get',
    `${WS}/sponsors/overview`,
    { params: { path: { workspace: slug }, query: { slot: search.slot, range: search.range } } },
    { enabled: canView, placeholderData: keepPreviousData },
  )
  const [dialog, setDialog] = useState<PageDialog>()
  const data = q.data
  const slotId = search.slot ?? data?.slotId
  const close = () => onSearch({ ...search, booking: undefined, issue: undefined })
  const openBooking = (id: string) => onSearch({ ...search, booking: id, issue: undefined })
  const openIssue = (n: number) => onSearch({ ...search, booking: undefined, issue: n, slot: slotId })
  const rows = data ? [...data.past, ...data.upcoming] : []
  const canManage = data?.canManage ?? false
  // "Book an issue": the first open issue to come, or the one after the shown range.
  const bookNext = () => {
    if (!data) return
    const open = data.upcoming.find((r) => !r.booking && !r.past)
    if (open) return openIssue(open.number)
    const last = rows[rows.length - 1]
    openIssue(last ? last.number + 1 : (data.slots.find((s) => s.id === slotId)?.firstNumber ?? 1))
  }
  const actions = data && canManage && data.slots.length ? (
    <Button variant="primary" onClick={bookNext}>
      {t(msp.bookIssue)}
    </Button>
  ) : undefined
  return (
    <EarnPage tab="sponsors" title={t(me.tabSponsors)} actions={actions}>
      {q.error && !data ? (
        <LoadError error={q.error} onRetry={() => void q.refetch()} />
      ) : !data ? (
        <Loading />
      ) : (
        <Body
          data={data}
          search={search}
          onSearch={onSearch}
          selected={search.booking ?? (search.issue ? `${slotId}:${search.issue}` : undefined)}
          openBooking={openBooking}
          openIssue={openIssue}
          setDialog={setDialog}
        />
      )}
      {search.booking ? (
        <BookingPanel id={search.booking} onClose={close} />
      ) : search.issue && slotId ? (
        <IssuePanel slotId={slotId} number={search.issue} library={data?.library ?? []} onClose={close} onBooked={(id) => onSearch({ ...search, booking: id, issue: undefined }, true)} />
      ) : null}
      {dialog?.kind === 'sponsor' ? <SponsorDialog sponsor={dialog.sponsor} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'slot' ? <SlotDialog slot={dialog.slot} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'payment' ? <PaymentDialog target={dialog.target} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'cancel' ? <CancelDialog target={dialog.target} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'extend' ? (
        <ExtendHoldDialog bookingId={dialog.bookingId} issue={dialog.issue} sponsor={dialog.sponsor} holdUntil={dialog.holdUntil} issueOn={dialog.issueOn} onClose={() => setDialog(undefined)} />
      ) : null}
      {dialog?.kind === 'link' ? <LinkCampaignDialog bookingId={dialog.bookingId} issue={dialog.issue} onClose={() => setDialog(undefined)} /> : null}
    </EarnPage>
  )
}

function Body({
  data,
  search,
  onSearch,
  selected,
  openBooking,
  openIssue,
  setDialog,
}: {
  data: Overview
  search: SponsorsSearch
  onSearch: (next: SponsorsSearch, replace?: boolean) => void
  selected?: string
  openBooking: (id: string) => void
  openIssue: (n: number) => void
  setDialog: (d: PageDialog) => void
}) {
  const t = useT()
  const tab = search.tab ?? 'bookings'
  const rows = [...data.past, ...data.upcoming]
  const canManage = data.canManage
  if (!data.slots.length) {
    return (
      <div className="flex flex-col gap-5">
        {canManage ? null : <ReadOnlyNote helpers={data.helpers} />}
        <NoSlots canManage={canManage} hasSponsors={data.library.length > 0} onSlot={() => setDialog({ kind: 'slot' })} onSponsor={() => setDialog({ kind: 'sponsor' })} />
      </div>
    )
  }
  const setTab = (next: SponsorsTab) => onSearch({ ...search, tab: next === 'bookings' ? undefined : next }, true)
  const tabs: { value: SponsorsTab; label: string; count: number }[] = [
    { value: 'bookings', label: t(msp.tabBookings), count: rows.length },
    { value: 'sponsors', label: t(msp.tabSponsors), count: data.library.length },
    { value: 'slots', label: t(msp.tabSlots), count: data.slots.length },
  ]
  const label = tab === 'bookings' ? t(msp.bookingsLabel) : tab === 'sponsors' ? t(msp.libraryLabel) : t(msp.tabSlots)
  return (
    <div className="flex flex-col gap-5">
      <Tiles data={data} />
      {canManage ? null : <ReadOnlyNote helpers={data.helpers} />}
      {canManage && data.notices.length ? (
        <div className="flex flex-col gap-3">
          {data.notices.map((n) => (
            <NoticeItem key={`${n.kind}:${n.bookingId}`} n={n} currency={data.currency} openBooking={openBooking} setDialog={setDialog} />
          ))}
        </div>
      ) : null}
      <section className="card" aria-label={label}>
        <div className="sp-sec">
          <div className="sp-ptabs" role="tablist" aria-label={t(msp.sectionsLabel)}>
            {tabs.map((x) => (
              <button
                key={x.value}
                type="button"
                role="tab"
                id={`sp-tab-${x.value}`}
                aria-controls="sp-tabpanel"
                aria-selected={tab === x.value}
                tabIndex={tab === x.value ? 0 : -1}
                className="sp-fchip"
                onClick={() => setTab(x.value)}
                onKeyDown={(e) => {
                  if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
                  const i = tabs.findIndex((y) => y.value === tab)
                  const next = tabs[(i + (e.key === 'ArrowRight' ? 1 : tabs.length - 1)) % tabs.length]!
                  setTab(next.value)
                  document.getElementById(`sp-tab-${next.value}`)?.focus()
                }}
              >
                {x.label} <b>{count(x.count)}</b>
              </button>
            ))}
          </div>
          {tab === 'bookings' ? (
            <div className="sp-selects">
              <select className="in" aria-label={t(msp.slotSelect)} value={data.slotId} onChange={(e) => onSearch({ ...search, slot: e.target.value, booking: undefined, issue: undefined })}>
                {data.slots.map((s) => (
                  <option key={s.id} value={s.id}>
                    {t(msp.slotName, { list: s.listName, slot: s.name })}
                  </option>
                ))}
              </select>
              <select
                className="in"
                aria-label={t(msp.rangeSelect)}
                value={data.range}
                onChange={(e) => onSearch({ ...search, range: e.target.value === 'around' ? undefined : (e.target.value as SponsorsRange) }, true)}
              >
                <option value="around">{t(msp.rangeAround)}</option>
                <option value="upcoming">{t(msp.rangeUpcoming)}</option>
                <option value="recent">{t(msp.rangeRecent)}</option>
                <option value="all">{t(msp.rangeAll)}</option>
              </select>
            </div>
          ) : canManage && tab === 'sponsors' ? (
            <Button size="sm" icon="plus" onClick={() => setDialog({ kind: 'sponsor' })}>
              {t(msp.newSponsor)}
            </Button>
          ) : canManage && tab === 'slots' ? (
            <Button size="sm" icon="plus" onClick={() => setDialog({ kind: 'slot' })}>
              {t(msp.newSlot)}
            </Button>
          ) : null}
        </div>
        <div id="sp-tabpanel" role="tabpanel" aria-labelledby={`sp-tab-${tab}`}>
          {tab === 'bookings' ? (
            <Bookings data={data} selected={selected} openBooking={openBooking} openIssue={openIssue} />
          ) : tab === 'sponsors' ? (
            <Library data={data} onEdit={(sponsor) => setDialog({ kind: 'sponsor', sponsor })} />
          ) : (
            <Slots data={data} onEdit={(slot) => setDialog({ kind: 'slot', slot })} />
          )}
        </div>
      </section>
    </div>
  )
}

/** The four numbers above the tabs. */
function Tiles({ data }: { data: Overview }) {
  const t = useT()
  const x = data.tiles
  const cur = data.currency
  const nums = (ns: number[]) => ns.map((n) => `#${n}`)
  return (
    <div className="sp-stats">
      <div className="card sp-stat">
        <span className="k">{t(msp.incomeLabel)}</span>
        <span className="v">{money(x.income30dCents, cur, 'never')}</span>
        <span className="s">
          {x.incomeIssues
            ? t(msp.incomeSub, { count: x.incomeIssues, paid: money(x.paid30dCents, cur, 'never'), unpaid: money(x.unpaid30dCents, cur, 'never') })
            : t(msp.incomeNone)}
        </span>
      </div>
      <div className="card sp-stat">
        <span className="k">{t(msp.nextLabel, { count: x.nextCount })}</span>
        <span className="v">{t(msp.nextValue, { count: x.nextBooked })}</span>
        <span className="s">{t(msp.nextSub, { hold: x.nextHold, open: x.nextOpen, from: x.nextFrom, to: x.nextTo })}</span>
      </div>
      <div className="card sp-stat">
        <span className="k">{t(msp.clickLabel)}</span>
        <span className="v">{x.avgClickRate === undefined || x.avgClickRate === null ? '—' : pct(x.avgClickRate, 1)}</span>
        <span className="s">{x.avgClickIssues.length ? t(msp.clickSub, { issues: nums(x.avgClickIssues).join(', ') }) : t(msp.clickNone)}</span>
      </div>
      <div className="card sp-stat">
        <span className="k">{t(msp.aheadLabel)}</span>
        <span className="v">{money(x.bookedAheadCents, cur, 'never')}</span>
        <span className="s">
          {x.bookedAheadIssues.length
            ? x.holdAheadCents
              ? t(msp.aheadSubHold, { issues: andList(nums(x.bookedAheadIssues)), hold: money(x.holdAheadCents, cur, 'never') })
              : t(msp.aheadSub, { issues: andList(nums(x.bookedAheadIssues)) })
            : x.holdAheadCents
              ? t(msp.aheadOnlyHold, { hold: money(x.holdAheadCents, cur, 'never') })
              : t(msp.aheadNone)}
        </span>
      </div>
    </div>
  )
}

/** What needs a person: a late creative, an overdue payment, a hold about to lapse, an issue without a campaign. */
function NoticeItem({ n, currency, openBooking, setDialog }: { n: Notice; currency: string; openBooking: (id: string) => void; setDialog: (d: PageDialog) => void }) {
  const t = useT()
  const qc = useQueryClient()
  const { slug } = useWorkspace()
  const update = $api.useMutation('put', `${WS}/sponsor-bookings/{id}`)
  const email = n.contactEmail ? (
    <ButtonAnchor size="sm" href={`mailto:${n.contactEmail}`} newTab={false}>
      {t(msp.emailContact, { name: n.contactName || n.contactEmail })}
    </ButtonAnchor>
  ) : null
  switch (n.kind) {
    case 'creative_late':
      return (
        <NoticeNote
          tone="warn"
          title={t(msp.lateTitle, { issue: n.issue, days: Math.max(0, n.daysToIssue), sponsor: n.sponsor, ends: ends(n.sponsor) })}
          actions={
            <>
              {email}
              <Button size="sm" onClick={() => openBooking(n.bookingId)}>
                {t(msp.openBooking)}
              </Button>
            </>
          }
        >
          {n.due && isToday(n.due) ? t(msp.lateTextToday, { issue: n.issue }) : t(msp.lateText, { issue: n.issue, date: n.due ? shortDate(n.due) : '' })}
        </NoticeNote>
      )
    case 'payment_overdue':
      return (
        <NoticeNote
          tone="warn"
          title={t(msp.overdueTitle, { sponsor: n.sponsor, amount: money(n.amountCents, currency), issue: n.issue, date: day(n.due) })}
          actions={
            <>
              <Button
                size="sm"
                onClick={() => setDialog({ kind: 'payment', target: { bookingId: n.bookingId, issue: n.issue, sponsor: n.sponsor, amountCents: n.amountCents, currency } })}
              >
                {t(msp.recordPayment)}
              </Button>
              {email}
            </>
          }
        >
          {t(msp.overdueText)}
        </NoticeNote>
      )
    case 'hold_lapsing': {
      const book = () =>
        update.mutate(
          { params: { path: { workspace: slug, id: n.bookingId } }, body: { status: 'booked' } },
          {
            onSuccess: () => {
              for (const k of sponsorKeys) void qc.invalidateQueries({ queryKey: k })
              toast({ kind: 'success', title: t(msp.bookFor, { issue: n.issue, sponsor: n.sponsor }) })
            },
            onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
          },
        )
      return (
        <NoticeNote
          tone="info"
          title={t(msp.lapseTitle, { sponsor: n.sponsor, ends: ends(n.sponsor), issue: n.issue, date: n.due ? shortDate(n.due) : '' })}
          actions={
            <>
              <Button size="sm" loading={update.isPending} onClick={book}>
                {t(msp.bookFor, { issue: n.issue, sponsor: n.sponsor })}
              </Button>
              <Button size="sm" onClick={() => setDialog({ kind: 'extend', bookingId: n.bookingId, issue: n.issue, sponsor: n.sponsor, holdUntil: n.due, issueOn: n.issueOn })}>
                {t(msp.extendHold)}
              </Button>
              <Button
                size="sm"
                onClick={() =>
                  setDialog({
                    kind: 'cancel',
                    target: { bookingId: n.bookingId, issue: n.issue, sponsor: n.sponsor, status: 'hold', holdUntil: n.due, amountCents: n.amountCents, currency },
                  })
                }
              >
                {t(msp.release, { issue: n.issue })}
              </Button>
            </>
          }
        >
          {t(msp.lapseText, { issue: n.issue })}
        </NoticeNote>
      )
    }
    case 'no_campaign':
      return (
        <NoticeNote
          tone="neutral"
          title={t(msp.noCampaignTitle, { issue: n.issue })}
          actions={
            <Button size="sm" onClick={() => setDialog({ kind: 'link', bookingId: n.bookingId, issue: n.issue })}>
              {t(msp.linkByHand)}
            </Button>
          }
        >
          {t(msp.noCampaignText, { list: n.listName, date: shortDate(n.issueOn), issue: n.issue })}
        </NoticeNote>
      )
  }
  return null
}

/** Bookings: a row per issue of the slot, the sent ones first. */
function Bookings({ data, selected, openBooking, openIssue }: { data: Overview; selected?: string; openBooking: (id: string) => void; openIssue: (n: number) => void }) {
  const t = useT()
  const slot = data.slots.find((s) => s.id === data.slotId) ?? data.slots[0]!
  const rows = [...data.past, ...data.upcoming]
  const canManage = data.canManage
  const open = (r: IssueRow) => (r.booking ? openBooking(r.booking.id) : openIssue(r.number))
  const isSel = (r: IssueRow) => (r.booking ? selected === r.booking.id : selected === `${r.slotId}:${r.number}`)
  const schedule = slot.everyDays === 7 ? t(msp.everyWeekday, { weekday: weekdayName(slot.firstOn) }) : t(msp.everyDays, { days: slot.everyDays })
  const terms = new Set(data.library.map((s) => s.termsDays))
  const termsDays = terms.size === 1 ? [...terms][0]! : 14
  return (
    <>
      <div className="sp-slotline">
        <span className="sp-pill">
          <span className="dot" style={{ color: slot.selling ? 'var(--ok-bar)' : 'var(--ink-4)' }} aria-hidden="true" />
          {t(msp.slotName, { list: slot.listName, slot: slot.name })}
        </span>
        <span className="k" style={{ color: 'var(--ink-nav)' }}>
          {slot.placement
            ? t(msp.slotLine, { placement: slot.placement, price: money(slot.priceCents, slot.currency), schedule })
            : t(msp.slotLineNoPlacement, { price: money(slot.priceCents, slot.currency), schedule })}
        </span>
      </div>
      <div className="max-md:hidden">
        <div className="overflow-x-auto">
          <table className="tbl sp-tbl" aria-label={t(msp.bookingsLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(msp.colIssue)}</th>
                <th scope="col" className="max-lg:hidden">
                  {t(msp.colSlot)}
                </th>
                <th scope="col">{t(msp.colBooking)}</th>
                <th scope="col">{t(msp.colSponsor)}</th>
                <th scope="col" className="r">
                  {t(msp.colPrice)}
                </th>
                <th scope="col">{t(msp.colCreative)}</th>
                <th scope="col">{t(msp.colPayment)}</th>
                <th scope="col" className="max-lg:hidden">
                  <span className="vh">{t(msp.colActions)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {data.past.length ? (
                <tr className="grp">
                  <td colSpan={8}>{t(msp.groupSent, { count: data.past.length })}</td>
                </tr>
              ) : null}
              {data.past.map((r) => (
                <BookingRow key={r.number} r={r} currency={slot.currency} canManage={canManage} selected={isSel(r)} onOpen={() => open(r)} />
              ))}
              {data.upcoming.length ? (
                <tr className="grp">
                  <td colSpan={8}>{t(msp.groupComing, { count: data.upcoming.length })}</td>
                </tr>
              ) : null}
              {data.upcoming.map((r) => (
                <BookingRow key={r.number} r={r} currency={slot.currency} canManage={canManage} selected={isSel(r)} onOpen={() => open(r)} />
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="hidden max-md:flex max-md:flex-col">
        {rows.map((r) => (
          <button key={r.number} type="button" className="sp-lst" onClick={() => open(r)}>
            <span className="sp-lst-top">
              <b className="font-semibold">{t(msp.issueOn, { issue: r.number, date: shortDate(r.issueOn) })}</b>
              <span className="ea-num font-semibold">{r.booking ? money(r.booking.priceCents, slot.currency) : ''}</span>
            </span>
            <span className="k" style={{ color: 'var(--ink-2)' }}>
              {r.booking?.sponsor ? t(msp.slotSponsor, { slot: r.slotName, sponsor: r.booking.sponsor.name }) : t(msp.openForBooking, { slot: r.slotName })}
            </span>
            <span className="sp-lst-badges">
              <BookingCell row={r} />
              {r.booking?.status === 'booked' ? <CreativeCell b={r.booking} /> : null}
              {r.booking && r.booking.status !== 'hold' ? <PaymentCell b={r.booking} /> : null}
            </span>
          </button>
        ))}
      </div>
      <div className="sp-foot">
        <span className="k" style={{ flex: '1 1 360px' }}>
          {terms.size > 1 ? t(msp.footRulesMixed, { creative: slot.creativeDays }) : t(msp.footRules, { creative: slot.creativeDays, terms: termsDays })}
        </span>
        <Popover
          align="end"
          label={t(msp.howCount)}
          trigger={
            <button type="button" className="sp-lnk">
              <Icon name="info" size={14} />
              {t(msp.howCount)}
            </button>
          }
        >
          <div className="flex max-w-[360px] flex-col gap-1.5 p-2.5 text-[13px]">
            <b>{t(msp.howCount)}</b>
            <span className="text-ink-2">{t(msp.howCountText)}</span>
          </div>
        </Popover>
      </div>
    </>
  )
}

function BookingRow({ r, currency, canManage, selected, onOpen }: { r: IssueRow; currency: string; canManage: boolean; selected: boolean; onOpen: () => void }) {
  const t = useT()
  const b = r.booking
  let action: string | undefined
  if (b?.status === 'sent') action = t(msp.actReport)
  else if (b) action = canManage ? t(msp.actOpen) : t(msp.actView)
  else if (!r.past && canManage) action = t(msp.actBook)
  return (
    <tr className={selected ? 'sel' : undefined}>
      <td>
        <div className="sp-col">
          <button type="button" className="sp-issue" onClick={onOpen}>
            {t(msp.issueOn, { issue: r.number, date: shortDate(r.issueOn) })}
          </button>
          <span className="k">{campaignLine(t, r)}</span>
        </div>
      </td>
      <td className="nw max-lg:hidden">{r.slotName}</td>
      <td>
        <BookingCell row={r} />
      </td>
      <td>
        {b?.sponsor ? (
          <span className="sp-who">
            <span className="inline-flex max-lg:hidden">
              <Initials initials={b.sponsor.initials} logo={b.sponsor.logoUrl} />
            </span>
            <span className="font-semibold">{b.sponsor.name}</span>
          </span>
        ) : (
          <span className="k">–</span>
        )}
      </td>
      <td className="r num">{price(b, currency)}</td>
      <td>
        <CreativeCell b={b} />
      </td>
      <td>
        <PaymentCell b={b} />
      </td>
      <td className="act max-lg:hidden">
        {action ? (
          <Button size="sm" onClick={onOpen}>
            {action}
          </Button>
        ) : null}
      </td>
    </tr>
  )
}

/** Sponsors: the library with its bookings, amounts and click rates. */
function Library({ data, onEdit }: { data: Overview; onEdit: (s: SponsorItem) => void }) {
  const t = useT()
  if (!data.library.length) {
    return <p className="m-0 px-[18px] py-6 text-ink-2">{t(msp.libraryEmpty)}</p>
  }
  return (
    <>
      <div className="overflow-x-auto">
        <table className="tbl sp-tbl" aria-label={t(msp.libraryLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(msp.colSponsor)}</th>
              <th scope="col">{t(msp.colContact)}</th>
              <th scope="col">{t(msp.colBookings)}</th>
              <th scope="col" className="r">
                {t(msp.colBooked)}
              </th>
              <th scope="col" className="r">
                {t(msp.colPaid)}
              </th>
              <th scope="col" className="r">
                {t(msp.colClickRate)}
              </th>
              <th scope="col">
                <span className="vh">{t(msp.colActions)}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {data.library.map((s) => {
              const issues = [...new Set([...s.issues, ...s.hold])].sort((a, b) => a - b).map((n) => `#${n}`)
              return (
                <tr key={s.id}>
                  <td className="nw">
                    <div className="flex items-center gap-2.5">
                      <Initials initials={s.initials} logo={s.logoUrl} size="lg" />
                      <div className="sp-col start">
                        <span className="font-semibold">{s.name}</span>
                        {s.website ? <span className="k">{s.website}</span> : null}
                      </div>
                    </div>
                  </td>
                  <td className="nw">
                    <div className="sp-col start">
                      <span>{s.contactName || '–'}</span>
                      {s.contactEmail ? (
                        <span className="k">
                          <span className="mono">{s.contactEmail}</span>
                        </span>
                      ) : null}
                    </div>
                  </td>
                  <td className="nw">
                    <div className="sp-col start">
                      <span>{issues.length ? issues.join(', ') : <span className="k">{t(msp.noBookings)}</span>}</span>
                      {s.hold.length ? <span className="k">{s.hold.length === 1 ? t(msp.onHold, { issue: s.hold[0]! }) : t(msp.holdsOn, { issues: andList(s.hold.map((n) => `#${n}`)) })}</span> : null}
                    </div>
                  </td>
                  <td className="r num nw">
                    <div className="sp-col end">
                      <span>{money(s.bookedCents, s.currency || data.currency, 'never')}</span>
                      {s.holdCents ? <span className="k">{t(msp.plusHold, { amount: money(s.holdCents, s.currency || data.currency, 'never') })}</span> : null}
                    </div>
                  </td>
                  <td className="r num nw">
                    <div className="sp-col end">
                      <span>{money(s.paidCents, s.currency || data.currency, 'never')}</span>
                      <span className="k">{t(msp.paysAfter, { days: s.termsDays })}</span>
                    </div>
                  </td>
                  <td className="r num nw">
                    <div className="sp-col end">
                      <span>{s.clickRate === undefined || s.clickRate === null ? '—' : pct(s.clickRate, 1)}</span>
                      {s.clickRateIssue ? <span className="k">#{s.clickRateIssue}</span> : null}
                    </div>
                  </td>
                  <td className="act">
                    {data.canManage ? (
                      <Button size="sm" onClick={() => onEdit(s)} aria-label={`${t(msp.edit)} · ${s.name}`}>
                        {t(msp.edit)}
                      </Button>
                    ) : null}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <div className="sp-foot">
        <span className="k">{t(msp.libraryFoot)}</span>
      </div>
    </>
  )
}

/** Slots: where each sits, its price and schedule. */
function Slots({ data, onEdit }: { data: Overview; onEdit: (s: SlotItem) => void }) {
  const t = useT()
  return (
    <>
      {data.slots.map((s) => (
        <div key={s.id} className="sp-slot">
          <div className="sp-slot-main">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="flex items-center gap-2.5">
                <span className="h2">{s.name}</span>
                {s.selling ? <Badge tone="ok">{t(msp.selling)}</Badge> : <Badge tone="neutral">{t(msp.paused)}</Badge>}
              </span>
              {data.canManage ? (
                <Button size="sm" onClick={() => onEdit(s)} aria-label={`${t(msp.edit)} · ${s.name}`}>
                  {t(msp.edit)}
                </Button>
              ) : null}
            </div>
            <dl className="sp-kv">
              <dt>{t(msp.slotList)}</dt>
              <dd>{s.listName}</dd>
              {s.placement ? (
                <>
                  <dt>{t(msp.slotWhere)}</dt>
                  <dd>{s.placement}</dd>
                </>
              ) : null}
              <dt>{t(msp.slotPrice)}</dt>
              <dd>{t(msp.slotPriceValue, { price: money(s.priceCents, s.currency) })}</dd>
              <dt>{t(msp.slotBookings)}</dt>
              <dd>{t(msp.slotOnePer)}</dd>
              <dt>{t(msp.slotIssues)}</dt>
              <dd>
                {s.everyDays === 7
                  ? t(msp.slotIssuesValue, { weekday: weekdayName(s.firstOn), number: s.firstNumber, date: dayMonthYear(s.firstOn) })
                  : t(msp.slotIssuesEvery, { days: s.everyDays, number: s.firstNumber, date: dayMonthYear(s.firstOn) })}
              </dd>
              <dt>{t(msp.slotCreative)}</dt>
              <dd>{t(msp.slotCreativeValue, { days: s.creativeDays })}</dd>
              <dt>{t(msp.slotCampaign)}</dt>
              <dd>{t(msp.slotCampaignValue)}</dd>
            </dl>
          </div>
          <div className="sp-outline" role="img" aria-label={t(msp.slotPreview, { slot: s.name })}>
            <i className="d" style={{ width: '40%', height: 10 }} />
            <div className="flex flex-col gap-[5px]">
              <i className="d" style={{ width: '85%', height: 9 }} />
              <i style={{ width: '100%', height: 8 }} />
              <i style={{ width: '70%', height: 8 }} />
            </div>
            <div className="slot">
              <span>{s.name}</span>
              <i style={{ width: '90%', height: 6 }} />
            </div>
            <div className="flex flex-col gap-[5px]">
              <i className="d" style={{ width: '80%', height: 9 }} />
              <i style={{ width: '95%', height: 8 }} />
            </div>
            <i style={{ width: '100%', height: 18, background: 'var(--ground)' }} />
          </div>
        </div>
      ))}
    </>
  )
}

/** No slots yet: sell a placement in your issues. */
function NoSlots({ canManage, hasSponsors, onSlot, onSponsor }: { canManage: boolean; hasSponsors: boolean; onSlot: () => void; onSponsor: () => void }) {
  const t = useT()
  return (
    <div className="card sp-empty">
      <span className="ico" aria-hidden="true">
        <MoneyIcon size={26} />
      </span>
      <span className="tt">{t(msp.noSlotsTitle)}</span>
      <span className="ds">{t(msp.noSlotsText)}</span>
      {canManage ? (
        <div className="mt-1 flex flex-wrap justify-center gap-2.5">
          <Button variant="primary" icon="plus" onClick={onSlot}>
            {t(msp.newSlot)}
          </Button>
          {hasSponsors ? null : <Button onClick={onSponsor}>{t(msp.noSlotsSponsorFirst)}</Button>}
        </div>
      ) : null}
      <span className="k">{t(msp.noSlotsNote)}</span>
    </div>
  )
}

/** Read-only · Aisha Bello, Viewer: who can help. */
function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const name = user?.name || user?.email || ''
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  return (
    <NoticeNote tone="neutral" title={t(msp.readOnlyTitle, { name, role })}>
      <T msg={helpers.length ? msp.readOnlyText : msp.readOnlyNoHelpers} values={{ permission: <span className="mono">{'sponsors:manage'}</span>, helpers: orList(helpers) }} />
    </NoticeNote>
  )
}

function Loading() {
  const t = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true" aria-label={t(msp.loadingLabel)}>
      <div className="sp-stats">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="card sp-stat">
            <Skeleton w={120} h={12} />
            <Skeleton w={90} h={24} className="mt-1" />
            <Skeleton w={150} h={10} className="mt-1" />
          </div>
        ))}
      </div>
      <div className="card flex flex-col gap-3.5 p-[18px]">
        <div className="flex gap-1.5">
          <Skeleton w={110} h={30} r={999} />
          <Skeleton w={104} h={30} r={999} />
          <Skeleton w={74} h={30} r={999} />
        </div>
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="sp-skrow">
            <Skeleton w={110} h={14} />
            <Skeleton w={70} h={20} r={999} />
            <Skeleton w="100%" h={10} className="flex-1" />
            <Skeleton w={60} h={28} />
          </div>
        ))}
      </div>
    </div>
  )
}

/** Couldn't load bookings (the board's centred card). */
function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const ref = problemReference(error)
  return (
    <div className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <span style={{ color: 'var(--bad-strong)' }}>
        <Icon name="alertCircle" size={28} />
      </span>
      <span className="text-[16px] font-bold">{t(msp.errorTitle)}</span>
      <span className="k max-w-[420px]">
        {errorMessage(error)} {t(msp.errorText)}
      </span>
      <div className="flex gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(me.retry)}
        </Button>
      </div>
      {ref ? <span className="k mono">{ref}</span> : null}
    </div>
  )
}
