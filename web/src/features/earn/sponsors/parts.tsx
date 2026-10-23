import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { dayMonthDate, shortDate, zonedParts } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { money } from '../format'
import { msp } from './messages'

export type IssueRow = Schemas['SponsorIssueRow']
export type BookingBrief = Schemas['SponsorBookingBrief']
export type SponsorPayment = Schemas['SponsorPayment']
export type SponsorItem = Schemas['SponsorItem']
export type SlotItem = Schemas['SponsorSlotItem']
export type Overview = Schemas['SponsorOverview']

/** The sponsorship operations live under the workspace. */
export const WS = '/api/v1/workspaces/{workspace}' as const

/** The queries a change to sponsors, slots or bookings makes stale. */
export const sponsorKeys = [
  ['get', `${WS}/sponsors/overview`],
  ['get', `${WS}/sponsor-bookings/{id}`],
  ['get', `${WS}/sponsor-slots/{id}/issues/{number}`],
] as const

function dayKey(d: string | Date): number {
  const p = zonedParts(d)
  return p.year * 10000 + p.month * 100 + p.day
}

/** The day falls on today, in the workspace's time zone. */
export function isToday(d: string | undefined | null): boolean {
  return Boolean(d) && dayKey(d!) === dayKey(new Date())
}

/** The day is before today, in the workspace's time zone. */
export function isPastDay(d: string | undefined | null): boolean {
  return Boolean(d) && dayKey(d!) < dayKey(new Date())
}

/** Calendar days from today to the day (0 today, negative before). */
export function daysUntil(d: string): number {
  const a = zonedParts(new Date())
  const b = zonedParts(d)
  return Math.round((Date.UTC(b.year, b.month - 1, b.day) - Date.UTC(a.year, a.month - 1, a.day)) / 86_400_000)
}

/** The day as a date input wants it: "2026-10-20". */
export function inputDay(d: string | Date): string {
  const p = zonedParts(d)
  return `${p.year}-${String(p.month).padStart(2, '0')}-${String(p.day).padStart(2, '0')}`
}

/** For the possessive in English messages: "s" when the name ends in an s (Northwind Analytics’ terms). */
export function ends(name: string): string {
  return /s$/i.test(name.trim()) ? 's' : 'other'
}

/** A sponsor's logo, or its initials in a rounded square. */
export function Initials({ initials, logo, size, className }: { initials: string; logo?: string; size?: 'sm' | 'lg' | 'xl'; className?: string }) {
  return (
    <span className={cn('sp-ini', size, className)} aria-hidden="true">
      {logo ? <img src={logo} alt="" /> : initials}
    </span>
  )
}

/** "Open": an outlined badge. */
export function OpenBadge() {
  const t = useT()
  return (
    <Badge tone="neutral" className="sp-open">
      {t(msp.badgeOpen)}
    </Badge>
  )
}

/** A badge with its detail beside it ("Hold until Mon 20 Oct"). */
export function Two({ badge, detail }: { badge: ReactNode; detail?: ReactNode }) {
  if (!detail) return <>{badge}</>
  return (
    <span className="sp-two">
      {badge}
      <span className="k">{detail}</span>
    </span>
  )
}

/** The Booking column: Open, Hold until …, Booked, Sent · report shared … */
export function BookingCell({ row }: { row: IssueRow }) {
  const t = useT()
  const b = row.booking
  if (!b) {
    return row.proposal ? (
      <span className="sp-two">
        <OpenBadge />
        <Badge tone="purple">{t(msp.badgeProposed)}</Badge>
      </span>
    ) : (
      <OpenBadge />
    )
  }
  switch (b.status) {
    case 'hold':
      return <Two badge={<Badge tone="info">{t(msp.badgeHold)}</Badge>} detail={b.holdUntil ? t(msp.holdUntil, { date: shortDate(b.holdUntil) }) : undefined} />
    case 'booked':
      return <Badge tone="ok">{t(msp.badgeBooked)}</Badge>
  }
  let detail: string | undefined
  if (b.sharedAt) detail = t(msp.reportShared, { date: dayMonthDate(b.sharedAt) })
  else if (!b.final && b.resultsFinalAt) detail = t(msp.resultsFinalOn, { date: dayMonthDate(b.resultsFinalAt) })
  else if (b.final && b.payment.status !== 'paid') detail = t(msp.resultsFinal)
  return <Two badge={<Badge tone="neutral">{t(msp.badgeSent)}</Badge>} detail={detail} />
}

/** The Creative column: Waiting · due Fri 17 Oct, or Ready; a dash once sent or while on hold. */
export function CreativeCell({ b }: { b?: BookingBrief }) {
  const t = useT()
  if (!b || b.status !== 'booked') return <span className="k">–</span>
  if (b.creativeStatus === 'ready') return <Badge tone="ok">{t(msp.creativeReady)}</Badge>
  const due = b.creativeDue
  const detail = !due ? undefined : isToday(due) ? t(msp.dueToday, { date: shortDate(due) }) : isPastDay(due) ? t(msp.wasDue, { date: shortDate(due) }) : t(msp.dueOn, { date: shortDate(due) })
  return <Two badge={<Badge tone="warn">{t(msp.creativeWaiting)}</Badge>} detail={detail} />
}

/** The Payment column: Paid 3 Oct, Unpaid · due 28 Oct, Overdue · was due 28 Oct; a dash while on hold. */
export function PaymentCell({ b }: { b?: BookingBrief }) {
  if (!b || b.status === 'hold') return <span className="k">–</span>
  return <PaymentBadge p={b.payment} />
}

export function PaymentBadge({ p }: { p: SponsorPayment }) {
  const t = useT()
  if (p.status === 'paid') return <Two badge={<Badge tone="ok">{t(msp.paymentPaid)}</Badge>} detail={p.paidOn ? dayMonthDate(p.paidOn) : undefined} />
  if (p.status === 'overdue') return <Two badge={<Badge tone="warn">{t(msp.paymentOverdue)}</Badge>} detail={p.due ? t(msp.paymentWasDue, { date: dayMonthDate(p.due) }) : undefined} />
  return <Two badge={<Badge tone="neutral">{t(msp.paymentUnpaid)}</Badge>} detail={p.due ? t(msp.dueOn, { date: dayMonthDate(p.due) }) : undefined} />
}

/** The line under an issue: "Weekly Digest #143 · draft", or "No campaign yet". */
export function campaignLine(t: ReturnType<typeof useT>, row: IssueRow): string {
  if (!row.campaign) return t(msp.noCampaign)
  return t(msp.campaignStatus, { name: row.campaign.name, status: row.campaign.status })
}

/** "$750", or a dash for an open issue. */
export function price(b: BookingBrief | undefined, currency: string): string {
  return b ? money(b.priceCents, currency, 'auto') : '–'
}

/** The note icon of the boards: an alert for warnings, an i otherwise. */
export function NoteIcon({ tone }: { tone: 'warn' | 'info' | 'neutral' }) {
  const color = tone === 'warn' ? 'var(--warn-bar)' : tone === 'info' ? '#1d4ed8' : 'var(--ink-2)'
  return (
    <span style={{ color, display: 'inline-flex', flex: 'none', marginTop: 1 }}>
      <Icon name={tone === 'warn' ? 'alertCircle' : 'info'} size={16} />
    </span>
  )
}

/** A board note: icon, bold title, text and its actions. */
export function NoticeNote({ tone, title, children, actions }: { tone: 'warn' | 'info' | 'neutral'; title: ReactNode; children?: ReactNode; actions?: ReactNode }) {
  return (
    <div className={cn('note', tone === 'warn' ? 'w' : tone === 'info' ? 'i' : 'n')} role="note">
      <NoteIcon tone={tone} />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{title}</span>
        {children || actions ? (
          <div>
            {children}
            {actions ? <div className="mt-2 flex flex-wrap gap-2">{actions}</div> : null}
          </div>
        ) : null}
      </div>
    </div>
  )
}

/** A campaign's name, linked to its report once sent and to the composer before. */
export function CampaignLink({ id, name, status }: { id: string; name: string; status: string }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug, campaignId: id }
  return status === 'sent' || status === 'sending' ? (
    <Link to="/w/$workspace/campaigns/$campaignId/report" params={params}>
      {name}
    </Link>
  ) : (
    <Link to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
      {name}
    </Link>
  )
}

/** The "banknote" glyph of Record payment. */
export function MoneyIcon({ size = 14 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="1.5" y="4" width="13" height="8" rx="1.5" />
      <circle cx="8" cy="8" r="1.8" />
      <path d="M4 6.5v3M12 6.5v3" />
    </svg>
  )
}
