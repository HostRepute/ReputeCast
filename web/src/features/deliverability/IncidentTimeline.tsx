import { Link } from '@tanstack/react-router'
import { Fragment } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { cn } from '@/lib/cn'
import { shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { severityTone } from './words'

function IncidentLink({ l }: { l: Schemas['DeliverabilityIncidentLink'] }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  switch (l.kind) {
    case 'warmup':
      return (
        <Link to="/w/$workspace/providers/warmup" params={params}>
          {l.label}
        </Link>
      )
    case 'campaign':
    case 'campaign_report':
      return (
        <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: l.id ?? '' }}>
          {l.label}
        </Link>
      )
    case 'provider':
      return (
        <Link to="/w/$workspace/providers" params={params}>
          {l.label}
        </Link>
      )
    case 'provider_events':
      return (
        <Link to="/w/$workspace/providers/events" params={params}>
          {l.label}
        </Link>
      )
    case 'import':
      return (
        <Link to="/w/$workspace/import" params={params}>
          {l.label}
        </Link>
      )
  }
  return (
    <Link to="/w/$workspace/deliverability" params={params}>
      {l.label}
    </Link>
  )
}

/** Incident timeline, linked to campaigns and providers. */
export function IncidentTimeline({ t, connected, className }: { t: Schemas['DeliverabilityIncidentTimeline']; connected: boolean; className?: string }) {
  return (
    <section className={cn('card min-w-0', className)} aria-labelledby="h-incidents">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="h-incidents">
          Incident timeline
        </h2>
        <span className="k">{connected || t.fromHostRepute ? 'linked to campaigns and providers' : 'from provider events and auto-pause history'}</span>
      </div>
      {t.items.length ? (
        <ol className="m-0 list-none p-0">
          {t.items.map((it, i) => (
            <li key={it.id} className={cn('flex items-start gap-3 px-4.5 py-3 text-[13px]', i > 0 && 'border-t border-line-2')}>
              <Badge tone={severityTone(it.severity)} className="mt-px">
                <time dateTime={it.occurredAt}>{shortDate(it.occurredAt)}</time>
              </Badge>
              <div className="flex flex-col gap-0.5">
                <span className="font-semibold">{it.title}</span>
                <span className="k">
                  {it.detail}
                  {it.links.length ? ' ' : null}
                  {it.links.map((l, j) => (
                    <Fragment key={j}>
                      {j > 0 ? ' · ' : null}
                      <IncidentLink l={l} />
                    </Fragment>
                  ))}
                </span>
              </div>
            </li>
          ))}
        </ol>
      ) : (
        <p className="px-4.5 py-4 text-[13px] text-ink-2">No incidents in the last 45 days. Listings, auto-pauses, outages and warm-up slowdowns appear here.</p>
      )}
      {t.total > t.items.length ? (
        <div className="border-t border-line-2 px-4.5 py-2.5">
          <span className="k">
            Showing the latest {t.items.length} of {t.total} in the last 45 days
          </span>
        </div>
      ) : null}
    </section>
  )
}
