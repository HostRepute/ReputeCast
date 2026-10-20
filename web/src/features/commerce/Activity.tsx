import { useInfiniteQuery } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { money, storesBase, type ActivityItem, type StorePage } from './api'
import { ca, cm } from './messages'
import './commerce.css'

type T = ReturnType<typeof useT>

/** What an entry's icon says: its colour and its glyph. */
type Glyph = 'dots' | 'check' | 'arrow' | 'clock' | 'cross'

const iconStyle: Record<Glyph, { bg: string; stroke: string }> = {
  dots: { bg: '#EEF0F3', stroke: '#4B5563' },
  check: { bg: '#DCFCE7', stroke: '#166534' },
  arrow: { bg: '#DBEAFE', stroke: '#1E40AF' },
  clock: { bg: '#FEF3C7', stroke: '#92400E' },
  cross: { bg: '#FEE2E2', stroke: '#991B1B' },
}

export function Glyph({ kind }: { kind: Glyph }) {
  const { stroke } = iconStyle[kind]
  const common = { width: 14, height: 14, viewBox: '0 0 16 16', fill: 'none', stroke, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const, 'aria-hidden': true }
  switch (kind) {
    case 'check':
      return (
        <svg {...common} strokeWidth="2">
          <path d="m3 8 3.5 3.5L13 5" />
        </svg>
      )
    case 'arrow':
      return (
        <svg {...common} strokeWidth="1.9">
          <path d="M3 8h10M9 4l4 4-4 4" />
        </svg>
      )
    case 'clock':
      return (
        <svg {...common} strokeWidth="1.8">
          <circle cx="8" cy="8" r="6.2" />
          <path d="M8 4.8V8l2.2 1.6" />
        </svg>
      )
    case 'cross':
      return (
        <svg {...common} strokeWidth="2">
          <path d="M4 4l8 8M12 4l-8 8" />
        </svg>
      )
  }
  return (
    <svg {...common} strokeWidth="2">
      <circle cx="3.5" cy="8" r=".9" />
      <circle cx="8" cy="8" r=".9" />
      <circle cx="12.5" cy="8" r=".9" />
    </svg>
  )
}

/** The facts an entry is worded with, besides its own. */
export interface ActivityContext {
  store: string
  platform: string
  list?: string
  window: number
}

export interface Described {
  title: string
  sub: string
  glyph: Glyph
}

/** An entry in words: its title and its line. */
export function describe(t: T, it: ActivityItem, c: ActivityContext): Described {
  const amount = money(it.amount, it.currency || 'USD', 'always')
  const credited = it.credited ?? ''
  switch (it.kind) {
    case 'order': {
      const title = it.customer ? t(ca.orderNamed, { order: it.order ?? '', amount, customer: it.customer }) : t(ca.order, { order: it.order ?? '', amount })
      let sub: string
      if (it.recovered && credited) {
        sub = it.creditedStep ? t(ca.orderRecoveredStep, { credited, step: it.creditedStep }) : t(ca.orderRecovered, { credited })
      } else if (!it.accepted && !it.contact) {
        sub = t(ca.orderNotAccepted)
      } else if (!it.accepted) {
        sub = c.list
          ? credited
            ? t(ca.orderNotAcceptedContactCredited, { list: c.list, credited })
            : t(ca.orderNotAcceptedContact, { list: c.list })
          : t(ca.orderNotAcceptedNoList)
      } else if (c.list) {
        sub = credited ? t(ca.orderAcceptedCredited, { list: c.list, credited }) : t(ca.orderAccepted, { list: c.list })
      } else {
        sub = credited ? (it.creditedStep ? t(ca.orderCreditedStep, { credited, step: it.creditedStep }) : t(ca.orderCredited, { credited })) : t(ca.orderPlain)
      }
      return { title, sub, glyph: it.recovered ? 'check' : 'dots' }
    }
    case 'refund':
      return { title: t(ca.refund, { order: it.order ?? '', amount }), sub: t(ca.refundSub), glyph: 'dots' }
    case 'checkout_abandoned':
      return {
        title: it.customer ? t(ca.checkoutAbandoned, { customer: it.customer, amount }) : t(ca.checkoutAbandonedAnon, { amount }),
        sub: t(ca.checkoutEntered, { store: c.store, flow: it.flow ?? '', hours: it.hours ?? 1 }),
        glyph: 'arrow',
      }
    case 'checkout_not_emailed':
      return {
        title: it.customer ? t(ca.checkoutAbandoned, { customer: it.customer, amount }) : t(ca.checkoutAbandonedAnon, { amount }),
        sub: t(ca.checkoutNotEmailed),
        glyph: 'dots',
      }
    case 'reconciliation':
      return {
        title: it.by ? t(ca.reconciliationBy, { by: it.by }) : t(ca.reconciliation),
        sub: t(ca.reconciliationSub, { missed: it.missed, products: it.products }),
        glyph: 'check',
      }
    case 'webhook_missing':
      return { title: t(ca.webhookMissing, { topic: it.topic ?? '' }), sub: t(ca.webhookMissingSub, { platform: c.platform, time: clock(it.at) }), glyph: 'clock' }
    case 'webhook_registered':
      return {
        title: it.topic ? t(ca.webhookRegistered, { topic: it.topic }) : t(ca.webhooksRegistered),
        sub: it.polled ? t(ca.webhookRegisteredSub, { by: it.by ?? '' }) : t(ca.webhookRegisteredSubPlain, { by: it.by ?? '' }),
        glyph: 'check',
      }
    case 'customer_unsubscribed':
      return { title: t(ca.customerUnsubscribed, { customer: it.customer ?? '', platform: c.platform }), sub: t(ca.customerUnsubscribedSub, { list: c.list ?? '' }), glyph: 'dots' }
    case 'unsubscribe_passed':
      return { title: t(ca.unsubscribePassed, { customer: it.customer ?? '' }), sub: t(ca.unsubscribePassedSub, { platform: c.platform }), glyph: 'check' }
    case 'import_started':
      return { title: t(ca.importStarted), sub: t(ca.importStartedSub, { by: it.by ?? '' }), glyph: 'arrow' }
    case 'import_finished':
      return { title: t(ca.importFinished), sub: t(ca.importFinishedSub, { products: it.products }), glyph: 'check' }
    case 'connected':
      return { title: t(ca.connected), sub: t(ca.connectedSub, { by: it.by ?? '' }), glyph: 'check' }
    case 'reconnected':
      if (it.reason === 'approved') return { title: t(ca.approvedAgain), sub: t(ca.connectedSub, { by: it.by ?? '' }), glyph: 'check' }
      return { title: t(ca.reconnected), sub: t(ca.reconnectedSub, { by: it.by ?? '' }), glyph: 'check' }
    case 'disconnected':
      return { title: t(ca.disconnected), sub: t(ca.disconnectedSub, { by: it.by ?? '' }), glyph: 'cross' }
    case 'revoked':
      return { title: t(ca.revoked, { platform: c.platform }), sub: it.reason === 'uninstalled' ? t(ca.revokedUninstalled) : t(ca.revokedRefused), glyph: 'cross' }
    case 'rate_limited':
      return { title: t(ca.rateLimited, { platform: c.platform }), sub: t(ca.rateLimitedSub, { time: it.until ? clock(it.until) : '' }), glyph: 'clock' }
  }
  return { title: t(ca.syncError), sub: it.message ?? '', glyph: 'cross' }
}

/** Today's entries show their time; earlier ones their day. */
export function entryTime(at: string, now = new Date()): string {
  return dayMonthDate(at) === dayMonthDate(now) ? clock(at) : dayMonthDate(at)
}

/** One entry of the log (`.ev`). */
export function ActivityEntry({ item, c, open, onToggle }: { item: ActivityItem; c: ActivityContext; open?: boolean; onToggle?: () => void }) {
  const t = useT()
  const d = describe(t, item, c)
  const body = (
    <>
      <span className="ih-ic" style={{ background: iconStyle[d.glyph].bg, border: 0 }}>
        <Glyph kind={d.glyph} />
      </span>
      <div className="ih-tx">
        <span style={{ fontWeight: 600 }}>{d.title}</span>
        <span className="k">{open ? expandedMeta(t, item, c) : d.sub}</span>
      </div>
      <span className="ih-tm">{entryTime(item.at)}</span>
    </>
  )
  if (!onToggle) return <div className="ih-ev">{body}</div>
  return (
    <>
      {/* Its name is what it shows (WCAG 2.5.3, label in name). */}
      <button type="button" className="ih-ev w-full cursor-pointer border-x-0 border-b-0 bg-transparent p-0 py-3 text-left font-[inherit] text-[inherit]" aria-expanded={open} onClick={onToggle}>
        {body}
      </button>
      {open ? <ActivityDetails item={item} c={c} /> : null}
    </>
  )
}

function expandedMeta(t: T, it: ActivityItem, c: ActivityContext): string {
  return it.topic
    ? t(ca.expandedMeta, { date: dayMonthDate(it.at), time: clock(it.at), platform: c.platform, topic: it.topic })
    : t(ca.expandedMetaPlain, { date: dayMonthDate(it.at), time: clock(it.at) })
}

/** An order's facts, opened in the log: marketing, contact, lists, automations, attribution. */
export function ActivityDetails({ item: it, c }: { item: ActivityItem; c: ActivityContext }) {
  const t = useT()
  const rows: [ReactNode, ReactNode][] = []
  if (it.kind === 'order' || it.kind === 'checkout_abandoned' || it.kind === 'checkout_not_emailed') {
    rows.push([t(ca.dMarketing), it.accepted ? t(ca.dMarketingYes, { platform: c.platform }) : t(ca.dMarketingNo, { platform: c.platform })])
    rows.push([t(ca.dContact), it.contact ? t(ca.dContactYes, { name: it.customer ?? '' }) : t(ca.dContactNone)])
    rows.push([t(ca.dLists), it.lists.length ? it.lists.join(', ') : t(ca.dListsNone)])
    rows.push([t(ca.dAutomations), it.flows.length ? it.flows.join(', ') : t(ca.dAutomationsNone)])
  }
  if (it.kind === 'order') {
    rows.push([t(ca.dCounted), it.contact ? t(ca.dCountedContact, { store: c.store }) : t(ca.dCountedValue, { store: c.store })])
    const credited = it.credited ?? ''
    rows.push([
      t(ca.dAttribution),
      it.matched === 'click_token' && credited
        ? t(ca.dAttrToken, { credited })
        : it.matched === 'email' && credited
          ? t(ca.dAttrEmail, { credited })
          : t(ca.dAttrNone, { days: it.window || c.window }),
    ])
  }
  if (!rows.length) return null
  return (
    <div className="cm-detail">
      <KeyValues items={rows} className="cm-kv" />
    </div>
  )
}

type LogFilter = 'all' | 'orders' | 'checkouts' | 'problems'

const filterKinds: Record<LogFilter, string[]> = {
  all: [],
  orders: ['order', 'refund'],
  checkouts: ['checkout_abandoned', 'checkout_not_emailed'],
  problems: ['webhook_missing', 'revoked', 'rate_limited', 'sync_error', 'disconnected'],
}

/** The full log: every entry of 30 days, a page at a time; an entry opens its facts. */
export function ActivityLogDialog({ page, c, open, onOpenChange }: { page: StorePage; c: ActivityContext; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [filter, setFilter] = useState<LogFilter>('all')
  const [opened, setOpened] = useState<string | null>(null)
  const q = useInfiniteQuery({
    queryKey: ['get', `${storesBase}/{id}/activity`, slug, page.store.id, filter],
    initialPageParam: '',
    queryFn: async ({ pageParam }) => {
      const { data } = await http.GET(`${storesBase}/{id}/activity`, {
        params: { path: { workspace: slug, id: page.store.id }, query: { cursor: pageParam || undefined, kinds: filterKinds[filter].length ? filterKinds[filter] : undefined, limit: 25 } },
      })
      return data!
    },
    getNextPageParam: (last) => last.next || undefined,
    enabled: open,
  })
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  return (
    <Dialog open={open} onOpenChange={onOpenChange} size="lg" title={t(cm.activityLogTitle, { name: page.store.name })} description={t(cm.activityLogCaption)}>
      <div className="flex flex-col gap-3">
        <div className="seg self-start" role="group" aria-label={t(cm.activityFilter)}>
          {(['all', 'orders', 'checkouts', 'problems'] as LogFilter[]).map((f) => (
            <button key={f} type="button" aria-pressed={filter === f} className={filter === f ? 'on' : undefined} onClick={() => setFilter(f)}>
              {t({ all: cm.activityAll, orders: cm.activityOrdersOnly, checkouts: cm.activityCheckoutsOnly, problems: cm.activityProblems }[f])}
            </button>
          ))}
        </div>
        {q.isLoading ? (
          <div className="flex flex-col gap-3">
            <Skeleton h={16} w="70%" />
            <Skeleton h={16} w="60%" />
            <Skeleton h={16} w="65%" />
          </div>
        ) : items.length === 0 ? (
          <span className="k">{t(cm.activityEmpty)}</span>
        ) : (
          <div className="flex flex-col">
            {items.map((it) => (
              <ActivityEntry key={it.id} item={it} c={c} open={opened === it.id} onToggle={() => setOpened(opened === it.id ? null : it.id)} />
            ))}
          </div>
        )}
        {q.hasNextPage ? (
          <Button size="sm" className="self-start" onClick={() => void q.fetchNextPage()} disabled={q.isFetchingNextPage}>
            {t(cm.activityMore)}
          </Button>
        ) : null}
      </div>
    </Dialog>
  )
}
