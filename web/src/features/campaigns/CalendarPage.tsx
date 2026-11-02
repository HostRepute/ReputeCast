import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useState, type DragEvent, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { FilterTabs } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { calendarDate, clock, count, getDisplayTimeZone, monthYear, weekdayName, weekStartsOn, zonedParts } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { mk } from './calendar.messages'
import { useRefreshCampaigns } from './manage'

type Item = Schemas['CampaignCalendarItem']
/** A calendar day in the workspace's time zone: "2026-10-14". */
type Day = string

export interface CalendarSearch {
  view?: 'month' | 'week'
  /** A day of the month or week shown; today when unset. */
  date?: string
}

const pad = (n: number) => String(n).padStart(2, '0')
const utc = (d: Day) => {
  const [y, m, dd] = d.split('-').map(Number)
  return new Date(Date.UTC(y!, m! - 1, dd!))
}
const dayOf = (d: Date): Day => d.toISOString().slice(0, 10)
export const addDays = (d: Day, n: number): Day => {
  const x = utc(d)
  x.setUTCDate(x.getUTCDate() + n)
  return dayOf(x)
}
/** The day a moment falls on in the workspace's time zone. */
export const dayKey = (at: string | Date): Day => {
  const p = zonedParts(at)
  return `${p.year}-${pad(p.month)}-${pad(p.day)}`
}
export const startOfWeek = (d: Day): Day => addDays(d, -((utc(d).getUTCDay() - weekStartsOn() + 7) % 7))
/** Noon of a day, safe to format in any zone. */
const noon = (d: Day) => new Date(utc(d).getTime() + 12 * 3600_000)

/** The days a view shows: whole weeks around the month, or one week. */
export function viewDays(view: 'month' | 'week', anchor: Day): Day[] {
  if (view === 'week') {
    const s = startOfWeek(anchor)
    return Array.from({ length: 7 }, (_, i) => addDays(s, i))
  }
  const first = `${anchor.slice(0, 7)}-01`
  const next = dayOf(new Date(Date.UTC(utc(first).getUTCFullYear(), utc(first).getUTCMonth() + 1, 1)))
  const s = startOfWeek(first)
  const weeks = Math.ceil((utc(next).getTime() - utc(s).getTime()) / (7 * 86400_000))
  return Array.from({ length: weeks * 7 }, (_, i) => addDays(s, i))
}

const reports = { email: '/w/$workspace/campaigns/$campaignId/report', sms: '/w/$workspace/campaigns/$campaignId/sms-report', push: '/w/$workspace/campaigns/$campaignId/push-report', whatsapp: '/w/$workspace/campaigns/$campaignId/whatsapp-report' } as const
const reviews = { email: '/w/$workspace/campaigns/$campaignId/review', sms: '/w/$workspace/campaigns/$campaignId/sms-review', push: '/w/$workspace/campaigns/$campaignId/push-review', whatsapp: '/w/$workspace/campaigns/$campaignId/whatsapp-review' } as const
type ChannelKey = keyof typeof reports
const channelOf = (c: string): ChannelKey => (c in reports ? (c as ChannelKey) : 'email')

/** A scheduled campaign moves on any channel; texts, push and WhatsApp re-plan their own sends (review 2026-11 CA-9). */
const movable = (it: Item) => it.kind === 'scheduled'

/** Takes a scheduled campaign back to a draft, on its own channel's operation. */
function unscheduleOn(channel: ChannelKey, workspace: string, campaignId: string) {
  const path = { params: { path: { workspace, campaignId } } }
  switch (channel) {
    case 'sms':
      return http.POST('/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/unschedule', path)
    case 'push':
      return http.POST('/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/unschedule', path)
    case 'whatsapp':
      return http.POST('/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/unschedule', path)
    default:
      return http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', path)
  }
}

/** Campaigns › Calendar: scheduled and sent campaigns and digest issues by month or week (CA-3). */
export function CalendarPage({ search }: { search: CalendarSearch }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const canMove = useCan('campaigns:send')
  const view = search.view ?? 'month'
  const today = dayKey(new Date())
  const anchor = search.date ?? today
  const days = viewDays(view, anchor)
  const from = days[0]!
  const to = addDays(days[days.length - 1]!, 1)
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/calendar',
    { params: { path: { workspace: slug }, query: { from, to } } },
    { placeholderData: keepPreviousData },
  )
  const [moving, setMoving] = useState<{ item: Item; day: Day }>()
  const [dragging, setDragging] = useState<Item>()
  const [over, setOver] = useState<Day>()
  const go = (patch: CalendarSearch) =>
    void navigate({ to: '/w/$workspace/campaigns/calendar', params: { workspace: slug }, search: { view: search.view, date: search.date, ...patch } })
  const step = (n: number) => {
    if (view === 'week') return go({ date: addDays(anchor, 7 * n) })
    const a = utc(`${anchor.slice(0, 7)}-01`)
    go({ date: dayOf(new Date(Date.UTC(a.getUTCFullYear(), a.getUTCMonth() + n, 1))) })
  }
  const byDay = new Map<Day, Item[]>()
  for (const it of q.data?.items ?? []) {
    const k = dayKey(it.at)
    byDay.set(k, [...(byDay.get(k) ?? []), it])
  }
  const heading = view === 'week' ? `${calendarDate(days[0]!, 'monthDay')} – ${calendarDate(days[6]!, 'monthDay')}` : monthYear(noon(`${anchor.slice(0, 7)}-15`), 'long')
  // Drag and drop: a scheduled email onto a day still ahead.
  const drop = {
    over: (day: Day) => (e: DragEvent) => {
      if (!dragging || day < today) return
      e.preventDefault()
      e.dataTransfer.dropEffect = 'move'
      if (over !== day) setOver(day)
    },
    leave: () => setOver(undefined),
    drop: (day: Day) => (e: DragEvent) => {
      e.preventDefault()
      setOver(undefined)
      if (dragging && day >= today && day !== dayKey(dragging.at)) setMoving({ item: dragging, day })
      setDragging(undefined)
    },
  }
  const ev = (it: Item, withButton = false) => (
    <CalendarEvent
      key={`${it.kind}-${it.id}`}
      it={it}
      draggable={canMove && movable(it)}
      onDragStart={() => setDragging(it)}
      onDragEnd={() => {
        setDragging(undefined)
        setOver(undefined)
      }}
      dragging={dragging === it}
      action={
        withButton && canMove && movable(it) ? (
          <Button size="sm" aria-label={tr(mk.rescheduleFor, { name: it.name })} onClick={() => setMoving({ item: it, day: dayKey(it.at) })}>
            {tr(mk.reschedule)}
          </Button>
        ) : null
      }
    />
  )
  return (
    <Page
      title={tr(mk.title)}
      documentTitle={tr(mk.documentTitle)}
      breadcrumb={[{ label: tr(mk.crumbCampaigns), to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: tr(mk.title) }]}
    >
      <section className="card" aria-labelledby="cal-h">
        <div className="sec flex-wrap">
          <div className="flex flex-wrap items-center gap-2.5">
            <Button size="sm" aria-label={tr(view === 'week' ? mk.previousWeek : mk.previousMonth)} onClick={() => step(-1)}>
              <span aria-hidden="true">‹</span>
            </Button>
            <h2 className="h2 min-w-[150px] text-center max-md:min-w-0 max-md:flex-1" id="cal-h" aria-live="polite">
              {heading}
            </h2>
            <Button size="sm" aria-label={tr(view === 'week' ? mk.nextWeek : mk.nextMonth)} onClick={() => step(1)}>
              <span aria-hidden="true">›</span>
            </Button>
            <Button size="sm" onClick={() => go({ date: undefined })}>
              {tr(mk.today)}
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <ul className="k m-0 flex list-none flex-wrap gap-2.5 p-0" aria-label={tr(mk.legend)}>
              <li className="inline-flex items-center gap-1.5">
                <span className="dot text-info-fg" aria-hidden="true" /> {tr(mk.scheduled)}
              </li>
              <li className="inline-flex items-center gap-1.5">
                <span className="dot text-accent" aria-hidden="true" /> {tr(mk.sent)}
              </li>
              <li className="inline-flex items-center gap-1.5">
                <span className="dot text-warn-bar" aria-hidden="true" /> {tr(mk.digestIssue)}
              </li>
            </ul>
            <FilterTabs
              label={tr(mk.view)}
              value={view}
              onChange={(v) => go({ view: v === 'month' ? undefined : v })}
              items={[
                { value: 'month', label: tr(mk.month) },
                { value: 'week', label: tr(mk.week) },
              ]}
            />
          </div>
        </div>
        {canMove ? <p className="k m-0 px-4 pt-2.5 max-md:hidden">{tr(mk.dragHint)}</p> : null}
        {q.isError ? (
          <div className="p-4">
            <ErrorPanel title={tr(mk.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{tr(common.retry)}</Button>}>
              {errorMessage(q.error)}
            </ErrorPanel>
          </div>
        ) : q.isPending ? (
          <div className="flex flex-col gap-2 p-4" aria-busy="true">
            <span className="vh" role="status">
              {tr(mk.loading)}
            </span>
            {Array.from({ length: 5 }, (_, i) => (
              <Skeleton key={i} w="100%" h={56} />
            ))}
          </div>
        ) : (
          <>
            {view === 'month' ? (
              <MonthGrid days={days} month={anchor.slice(0, 7)} today={today} byDay={byDay} over={over} drop={drop} ev={ev} onMore={(day) => go({ view: 'week', date: day })} />
            ) : null}
            {/* The week view, and the month as a list on phones. */}
            <ul className={view === 'month' ? 'm-0 hidden list-none p-0 max-md:block' : 'm-0 list-none p-0'}>
              {days
                .filter((d) => view === 'week' || (d.slice(0, 7) === anchor.slice(0, 7) && byDay.has(d)))
                .map((d) => (
                  <li
                    key={d}
                    className={`flex items-start gap-3 border-t border-line-2 px-4 py-2.5 first:border-t-0 ${over === d ? 'bg-accent-tint' : ''} ${d < today ? 'opacity-80' : ''}`}
                    onDragOver={drop.over(d)}
                    onDragLeave={drop.leave}
                    onDrop={drop.drop(d)}
                  >
                    <span className={`k w-[96px] shrink-0 pt-1 ${d === today ? 'font-bold text-ink' : ''}`}>{calendarDate(d, 'shortDate')}</span>
                    <div className="flex min-w-0 flex-1 flex-col gap-1.5">{(byDay.get(d) ?? []).map((it) => ev(it, true))}</div>
                    {!byDay.has(d) ? <span className="k pt-1">–</span> : null}
                  </li>
                ))}
            </ul>
            {(q.data.items.length === 0 && view === 'week') || (view === 'month' && q.data.items.every((it) => dayKey(it.at).slice(0, 7) !== anchor.slice(0, 7))) ? (
              <p className={`k m-0 border-t border-line-2 px-4 py-3 ${view === 'month' ? 'md:hidden' : ''}`}>{tr(mk.nothing, { view })}</p>
            ) : null}
          </>
        )}
      </section>
      {moving ? (
        moving.item.approved ? (
          <ApprovedMoveDialog item={moving.item} onClose={() => setMoving(undefined)} />
        ) : (
          <RescheduleDialog item={moving.item} day={moving.day} onClose={() => setMoving(undefined)} />
        )
      ) : null}
    </Page>
  )
}

type Drop = {
  over: (day: Day) => (e: DragEvent) => void
  leave: () => void
  drop: (day: Day) => (e: DragEvent) => void
}

/** The month as a table: weeks in rows, a day's items in its cell; three show, the rest open the week. */
function MonthGrid({
  days,
  month,
  today,
  byDay,
  over,
  drop,
  ev,
  onMore,
}: {
  days: Day[]
  month: string
  today: Day
  byDay: Map<Day, Item[]>
  over?: Day
  drop: Drop
  ev: (it: Item) => ReactNode
  onMore: (day: Day) => void
}) {
  const tr = useT()
  const weeks = Array.from({ length: days.length / 7 }, (_, i) => days.slice(i * 7, i * 7 + 7))
  return (
    <div className="p-4 max-md:hidden">
      <table className="w-full table-fixed border-collapse text-[12px]">
        <caption className="vh">{monthYear(noon(`${month}-15`), 'long')}</caption>
        <thead>
          <tr>
            {weeks[0]!.map((d) => (
              <th key={d} scope="col" className="border border-line bg-ground-2 px-2 py-1.5 text-left text-[11px] font-bold uppercase tracking-[.05em] text-ink-3">
                {weekdayName(noon(d))}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {weeks.map((w) => (
            <tr key={w[0]}>
              {w.map((d) => {
                const items = byDay.get(d) ?? []
                const out = d.slice(0, 7) !== month
                return (
                  <td
                    key={d}
                    className={`h-[104px] border border-line p-1.5 align-top ${out ? 'bg-ground-2 text-ink-3' : ''} ${over === d ? 'outline-2 -outline-offset-2 outline-dashed outline-info-fg' : ''}`}
                    aria-label={tr(mk.cell, { day: calendarDate(d, 'shortDate'), count: items.length })}
                    onDragOver={drop.over(d)}
                    onDragLeave={drop.leave}
                    onDrop={drop.drop(d)}
                  >
                    <div className="flex flex-col gap-1">
                      <span
                        className={
                          d === today
                            ? 'inline-flex h-[22px] w-[22px] items-center justify-center rounded-full bg-accent font-semibold text-white'
                            : 'font-semibold text-ink-2'
                        }
                        aria-current={d === today ? 'date' : undefined}
                      >
                        {Number(d.slice(8))}
                      </span>
                      {items.slice(0, 3).map((it) => ev(it))}
                      {items.length > 3 ? (
                        <button type="button" className="cursor-pointer self-start border-0 bg-transparent p-0 text-[11.5px] font-semibold text-accent underline underline-offset-2" onClick={() => onMore(d)}>
                          {tr(mk.more, { count: items.length - 3 })}
                        </button>
                      ) : null}
                    </div>
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const evTone = {
  scheduled: 'border-info-fg bg-info-tint text-info-fg',
  sent: 'border-accent bg-accent-tint text-ink',
  digest_issue: 'border-warn-bar bg-warn-tint text-warn-fg',
} as const

/** One item: a link to its review, report or digest; a scheduled email can be dragged. */
function CalendarEvent({
  it,
  draggable,
  onDragStart,
  onDragEnd,
  dragging,
  action,
}: {
  it: Item
  draggable: boolean
  onDragStart: () => void
  onDragEnd: () => void
  dragging: boolean
  action: ReactNode
}) {
  const tr = useT()
  const { slug } = useWorkspace()
  const time = it.local && it.localTime ? tr(mk.local, { time: it.localTime }) : clock(it.at)
  const line =
    it.kind === 'digest_issue'
      ? tr(mk.evDigest, { time })
      : it.kind === 'scheduled'
        ? tr(mk.evScheduled, { time, count: it.recipients != null ? count(it.recipients) : 'none' })
        : // Texts and push notifications count what went out on their own reports: the audience stands in.
          tr(mk.evSent, { time, status: it.status, count: count(it.sent || it.recipients || 0) })
  const cls = `cal-ev flex min-w-0 flex-col gap-px rounded-md border-l-[3px] px-1.5 py-1 text-[11.5px] leading-[1.3] no-underline ${evTone[it.kind]} ${draggable ? 'cursor-grab' : ''} ${dragging ? 'opacity-55' : ''}`
  const body = (
    <>
      <b className="truncate">{it.name}</b>
      <span className="truncate">{line}</span>
    </>
  )
  const ch = channelOf(it.channel)
  const link =
    it.kind === 'digest_issue' ? (
      <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: it.digestId ?? '' }} className={cls}>
        {body}
      </Link>
    ) : (
      <Link
        to={it.kind === 'scheduled' ? reviews[ch] : reports[ch]}
        params={{ workspace: slug, campaignId: it.id }}
        className={cls}
        draggable={draggable || undefined}
        onDragStart={
          draggable
            ? (e) => {
                e.dataTransfer.effectAllowed = 'move'
                e.dataTransfer.setData('text/plain', it.name)
                onDragStart()
              }
            : undefined
        }
        onDragEnd={draggable ? onDragEnd : undefined}
      >
        {body}
      </Link>
    )
  if (!action) return link
  return (
    <div className="flex items-center gap-2">
      <div className="min-w-0 flex-1">{link}</div>
      {action}
    </div>
  )
}

/**
 * An approved campaign's time is part of its approval (review 2026-11
 * CA-10): moving it means unscheduling it and asking again, said before
 * anyone picks a new time.
 */
export function ApprovedMoveDialog({ item, onClose }: { item: Item; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshCampaigns()
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const unschedule = async () => {
    setBusy(true)
    setErr(undefined)
    try {
      const ch = channelOf(item.channel)
      await unscheduleOn(ch, slug, item.id)
      await refresh()
      toast({ kind: 'success', title: t(mk.unscheduledAsk, { name: item.name }) })
      onClose()
      void navigate({ to: reviews[ch], params: { workspace: slug, campaignId: item.id } })
    } catch (e) {
      setErr(errorMessage(e))
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(mk.approvedTitle, { name: item.name })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void unschedule()}>
            {tr(mk.unscheduleAsk)}
          </Button>
        </>
      }
    >
      <p className="m-0 text-[13.5px]">{tr(mk.approvedBody, { when: `${calendarDate(dayKey(item.at), 'shortDate')}, ${clock(item.at).slice(0, 5)}` })}</p>
      {err ? (
        <Note tone="bad" className="mt-3">
          {err}
        </Note>
      ) : null}
    </Dialog>
  )
}

/**
 * Moves a scheduled campaign to another day and time: Send Readiness
 * rechecks an email before it goes out; texts, push and WhatsApp keep quiet
 * hours in each recipient's zone (CA-9).
 */
export function RescheduleDialog({ item, day, onClose }: { item: Item; day: Day; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const refresh = useRefreshCampaigns()
  const zone = getDisplayTimeZone() ?? 'UTC'
  const [date, setDate] = useState(day)
  const [time, setTime] = useState(item.local && item.localTime ? item.localTime : clock(item.at).slice(0, 5))
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const today = dayKey(new Date())
  const when = `${calendarDate(date, 'shortDate')}, ${time}`
  const email = channelOf(item.channel) === 'email'
  const save = async () => {
    if (!date || !/^\d{2}:\d{2}$/.test(time) || date < today) {
      setErr(tr(mk.pickFuture))
      return
    }
    setBusy(true)
    setErr(undefined)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/reschedule', {
        params: { path: { workspace: slug, campaignId: item.id } },
        body: { mode: item.local ? 'local' : 'at', date, time, timezone: zone },
      })
      await refresh()
      toast({ kind: 'success', title: t(mk.moved, { name: item.name, when }) })
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(mk.moveTitle, { name: item.name })}
      footer={
        <>
          <Link className="btn mr-auto" to={reviews[channelOf(item.channel)]} params={{ workspace: slug, campaignId: item.id }}>
            {tr(mk.openReview)}
          </Link>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {busy ? tr(mk.moving) : tr(mk.moveTo, { when })}
          </Button>
        </>
      }
    >
      <form
        noValidate
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="lbl mb-1.5 p-0">{tr(mk.dayAndTime)}</legend>
          <div className="flex flex-wrap gap-2">
            <label className="vh" htmlFor={`${id}-day`}>
              {tr(mk.day)}
            </label>
            <input id={`${id}-day`} type="date" className="in" value={date} min={today} onChange={(e) => setDate(e.target.value)} aria-describedby={`${id}-hint`} />
            <label className="vh" htmlFor={`${id}-time`}>
              {tr(mk.time)}
            </label>
            <input id={`${id}-time`} type="time" className="in" value={time} onChange={(e) => setTime(e.target.value)} aria-describedby={`${id}-hint`} />
          </div>
          <span className="k" id={`${id}-hint`}>
            {email ? (item.local ? tr(mk.localHint, { zone }) : tr(mk.zoneHint, { zone })) : item.local ? tr(mk.channelLocalHint, { zone }) : tr(mk.channelZoneHint, { zone })}
          </span>
        </fieldset>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}
