import { Link } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { defineMessages, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { deliveredPct, secs } from './shared'

const mc = defineMessages('transactional.overviewCard', {
  title: 'Transactional · last 7 days',
  label: 'Transactional, last 7 days',
  line: "{product}'s product mail · {route}",
  route: '{primary}, then {fallback}',
  noRoute: 'no route yet',
  sent: 'Sent',
  delivered: 'Delivered',
  median: 'Median delivery',
  rejected: 'Rejected',
  noAlerts: 'No active alerts',
  alerts: '{count, plural, one {# active alert} other {# active alerts}}',
  failover: 'Failing over to {provider}',
  open: 'Open Transactional',
})

const API = '/api/v1/workspaces/{workspace}/transactional' as const

/**
 * The Overview's Transactional card (Main board): the last 7 days' sent,
 * delivery rate, median delivery and rejections, any active alert and the
 * route. It shows once the workspace sends transactional mail, to people
 * who can see it, while the section is in the sidebar.
 */
export function TransactionalOverviewCard() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const can = me.permissions.includes('transactional:view') && !(me.hiddenSections ?? []).includes('transactional')
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', `${API}/overview`, { params: { path: { workspace: slug }, query: { period: '7d' } } }, { enabled: can })
  const settings = $api.useQuery('get', `${API}/settings`, path, { enabled: can })
  const o = q.data
  if (!can || !o || (!o.stats.sent && !o.firstRun.published)) return null
  const r = o.route
  const route = r.primary ? (r.fallback ? t(mc.route, { primary: r.primary.name, fallback: r.fallback.name }) : r.primary.name) : t(mc.noRoute)
  const product = settings.data?.productName || me.workspace.name
  const metric = (label: string, value: string) => (
    <div className="flex flex-col gap-0.5">
      <span className="k">{label}</span>
      <span className="text-[18px] font-bold tabular-nums">{value}</span>
    </div>
  )
  return (
    <section className="card flex flex-wrap items-center" style={{ padding: '18px 22px', gap: '14px 30px' }} aria-label={t(mc.label)}>
      <div className="flex min-w-0 flex-col gap-0.5" style={{ flex: '1 1 240px' }}>
        <h2 className="h2">{t(mc.title)}</h2>
        <span className="k">{t(mc.line, { product, route })}</span>
      </div>
      {metric(t(mc.sent), count(o.stats.sent))}
      {metric(t(mc.delivered), o.stats.sent ? deliveredPct(o.stats.deliveryRate) : '—')}
      {metric(t(mc.median), secs(o.stats.medianMs))}
      {metric(t(mc.rejected), count(o.stats.rejected))}
      {o.failover ? (
        <Badge tone="info" dot>
          {t(mc.failover, { provider: o.failover.to?.name ?? '' })}
        </Badge>
      ) : o.alerts.length ? (
        <Badge tone="bad" dot>
          {t(mc.alerts, { count: o.alerts.length })}
        </Badge>
      ) : (
        <Badge tone="ok" dot>
          {t(mc.noAlerts)}
        </Badge>
      )}
      <Link to="/w/$workspace/transactional" params={{ workspace: slug }} className="text-[13px] font-semibold">
        {t(mc.open)}
      </Link>
    </section>
  )
}
