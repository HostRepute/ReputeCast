import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { cn } from '@/lib/cn'
import { clock, monthName } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useT } from '@/i18n'
import { m } from './messages'
import { bouncePct, complaintPct, rateColor, statusBadge } from '../providers/common'
import { bounceLabel, complaintLabel, costCell, quotaRow, rateCell, webhookAgo } from './words'

/** A labelled value over a thin bar (board: 12.5px rows, 6px bars). */
function Meter({ label, value, percent, color }: { label: string; value: string; percent: number; color: string }) {
  return (
    <>
      <div className="flex justify-between max-md:flex-wrap">
        <span className="k">{label}</span>
        <span className="n">{value}</span>
      </div>
      <div className="bar" aria-hidden="true">
        <i style={{ width: `${Math.max(0, Math.min(100, percent))}%`, background: color }} />
      </div>
    </>
  )
}

/** One sending provider: status, rates against its own limits, quota, rate, last webhook, cost (Main board). */
export function ProviderOverviewCard({ p, monthStart }: { p: Schemas['OverviewProviderCard']; monthStart: string }) {
  const t = useT()
  const now = useNow(30_000)
  const c = p.card
  const stale = Boolean(p.refresh)
  const badge = stale ? { tone: 'bad' as const, text: p.refresh!.lastSuccessAt ? t(m.staleSince, { time: clock(p.refresh!.lastSuccessAt) }) : t(m.stale) } : statusBadge(c, now)
  const b = c.bounce
  const cp = c.complaint
  const quota = stale ? null : quotaRow(c, now)
  const rate = rateCell(c)
  return (
    <article className={cn('card flex flex-col gap-3 px-5 py-4.5', (stale || c.status === 'down') && 'border-bad-line')} aria-label={p.displayName}>
      <div className="flex items-center justify-between gap-2.5 max-md:flex-wrap">
        <div className="flex flex-col">
          <span className="text-[15px] font-semibold">{p.displayName}</span>
          <span className="k">{p.detail}</span>
        </div>
        <Badge tone={badge.tone} dot>
          {badge.text}
        </Badge>
      </div>
      <div className={cn('flex flex-col gap-2 text-[12.5px]', stale && 'opacity-70')}>
        <Meter label={bounceLabel(b, stale)} value={bouncePct(b.value)} percent={b.value !== undefined ? (b.value / b.limit) * 100 : 0} color={rateColor(b.value, b.limit, b.stale)} />
        <Meter label={complaintLabel(cp)} value={complaintPct(cp.value)} percent={cp.value !== undefined ? (cp.value / cp.limit) * 100 : 0} color={rateColor(cp.value, cp.limit, cp.stale)} />
        {quota ? <Meter label={quota.label} value={quota.value} percent={quota.percent} color="var(--accent)" /> : null}
      </div>
      {stale ? (
        <span className="k">
          {c.webhook.kind
            ? t(m.eventsVia, {
                webhook: c.webhook.kind === 'Registered' ? 'yes' : 'no',
                feed: c.webhook.kind === 'Bounce mailbox' ? t(m.feedBounceMailbox) : c.webhook.kind,
                ago: webhookAgo(c.webhook.lastEventAt, now),
                unaffected: c.webhook.lastEventAt && !c.webhook.silentSince ? 'yes' : 'no',
              })
            : t(m.noFeed, { unaffected: c.webhook.lastEventAt && !c.webhook.silentSince ? 'yes' : 'no' })}
        </span>
      ) : (
        <div className="grid gap-2 text-[12px] [grid-template-columns:repeat(3,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
          <div>
            <div className="k">{rate.label}</div>
            <div className="n">{rate.value}</div>
          </div>
          <div>
            <div className="k">{t(m.lastWebhook)}</div>
            <div className="n">{webhookAgo(c.webhook.lastEventAt, now)}</div>
          </div>
          <div>
            <div className="k">{t(m.costMonth, { month: monthName(monthStart) })}</div>
            <div className="n">{costCell(c)}</div>
          </div>
        </div>
      )}
    </article>
  )
}
