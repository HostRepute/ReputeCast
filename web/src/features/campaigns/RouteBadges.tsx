import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { count, trimPct } from '@/lib/format'
import { m } from './messages'

type Share = Schemas['RouteShare']

/**
 * Sending providers of a campaign: planned split for unsent campaigns
 * ("SES 90%"), recipients per provider once sent ("SES 43,182").
 */
export function RouteBadges({ routes, sent }: { routes: Share[]; sent: boolean }) {
  const t = useT()
  if (routes.length === 0) return <span className="k">{t(m.broadcastsRoute)}</span>
  return (
    <span className="inline-flex flex-wrap gap-x-1.5 gap-y-1 align-middle">
      {routes.map((r) => (
        <Badge key={r.providerId} tone={r.adapter === 'ses' ? 'purple' : 'neutral'}>
          {r.short} {sent ? count(r.sent || r.planned) : trimPct(Math.round(r.percent))}
        </Badge>
      ))}
    </span>
  )
}
