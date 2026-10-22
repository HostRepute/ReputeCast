import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { ago, clock, count, money, monthName, pct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { failoverText, providerColor, ProvidersTabs, streamName, throwIfForbidden, type Stream } from './common'
import { m } from './messages'
import { mo } from './overview.messages'
import { AuthBanner, AuthCard, AwaitingTable, IncidentTimeline, OutageAlert, SilentAlert } from './ProviderIncidents'
import { CompactRow, DownRow, OutageRow, ProviderRow, SilentRow } from './ProviderRows'
import { ProviderSettingsDialog } from './ProviderSettingsDialog'

type Overview = Schemas['ProvidersOverview']

/** 7.3 Sending providers: health, limits, cost and routing of every connected provider. */
export function ProvidersPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/providers', { params: { path: { workspace: slug } } }, { refetchInterval: 60_000 })
  throwIfForbidden(q.error)
  const [settings, setSettings] = useState<{ id: string; focus?: 'credentials' } | null>(null)
  const data = q.data
  const empty = data && data.providers.length === 0
  const settingsCard = settings && data?.providers.find((p) => p.id === settings.id)
  return (
    <Page title={t(m.title)} filters={<ProvidersTabs />} actions={data && !empty ? <HeaderActions data={data} /> : null}>
      {q.isError ? (
        <ErrorPanel title={t(mo.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(mo.loadFailedBody)}
        </ErrorPanel>
      ) : q.isPending ? (
        <ProvidersSkeleton />
      ) : empty ? (
        <NoProviders />
      ) : (
        <Body data={data!} onSettings={(id, focus) => setSettings({ id, focus })} />
      )}
      {settingsCard ? <ProviderSettingsDialog card={settingsCard} open={Boolean(settings)} focus={settings?.focus} onOpenChange={(o) => !o && setSettings(null)} /> : null}
    </Page>
  )
}

function HeaderActions({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('providers:manage')
  const down = data.providers.filter((p) => p.status === 'down').length
  const degraded = data.providers.filter((p) => p.status === 'degraded').length
  const n = data.summary.connected
  return (
    <>
      {down ? (
        <Badge tone="bad" dot>
          {t(mo.providersDown, { count: down })}
        </Badge>
      ) : degraded ? (
        <Badge tone="warn" dot>
          {t(mo.providersDegraded, { count: degraded })}
        </Badge>
      ) : data.summary.checkedAt ? (
        <span className="k">{n === 1 ? t(mo.checkedOne, { ago: ago(data.summary.checkedAt) }) : t(mo.checkedAll, { count: n, ago: ago(data.summary.checkedAt) })}</span>
      ) : null}
      {reason ? (
        <Button variant="primary" disabledReason={reason}>
          {t(mo.addProvider)}
        </Button>
      ) : (
        <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
          {t(mo.addProvider)}
        </ButtonLink>
      )}
    </>
  )
}

function Body({ data, onSettings }: { data: Overview; onSettings: (id: string, focus?: 'credentials') => void }) {
  const monthStart = data.summary.monthStart
  if (data.outage) {
    const o = data.outage
    const down = data.providers.find((p) => p.id === o.providerId)
    const others = data.providers.filter((p) => p.id !== o.providerId)
    return (
      <>
        <OutageAlert outage={o} />
        {data.silent ? <SilentAlert silent={data.silent} /> : null}
        <section className="card flex flex-col">
          {down ? <DownRow card={down} outage={o} /> : null}
          {others.map((c, i) => (
            <OutageRow key={c.id} card={c} outage={o} last={i === others.length - 1} />
          ))}
        </section>
        {o.incidents.length ? <IncidentTimeline incidents={o.incidents} /> : null}
      </>
    )
  }
  if (data.silent) {
    const s = data.silent
    const quiet = data.providers.find((p) => p.id === s.providerId)
    const others = data.providers.filter((p) => p.id !== s.providerId)
    return (
      <>
        <SilentAlert silent={s} />
        <section className="card flex flex-col">
          {quiet ? <SilentRow card={quiet} silent={s} /> : null}
          {others.map((c, i) => (
            <CompactRow key={c.id} card={c} silent={s} last={i === others.length - 1} />
          ))}
        </section>
        {s.awaiting.length ? <AwaitingTable silent={s} /> : null}
      </>
    )
  }
  const af = data.authFailure
  const cards = af ? data.providers.filter((p) => p.id !== af.providerId) : data.providers
  return (
    <>
      {af ? (
        <>
          <AuthBanner af={af} onUpdate={() => onSettings(af.providerId, 'credentials')} />
          <AuthCard af={af} card={data.providers.find((p) => p.id === af.providerId)} />
        </>
      ) : null}
      <Summary data={data} />
      {cards.length ? (
        <section className="card flex flex-col">
          {cards.map((c, i) => (
            <ProviderRow key={c.id} card={c} monthStart={monthStart} last={i === cards.length - 1} onSettings={() => onSettings(c.id)} />
          ))}
        </section>
      ) : null}
      {data.routing.length ? <RoutingSummary routes={data.routing} /> : null}
    </>
  )
}

function Tile({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card flex flex-col gap-0.5 px-[18px] py-3.5">
      <span className="k">{label}</span>
      <span className="n text-[22px]">{value}</span>
      <span className="k">{sub}</span>
    </div>
  )
}

/** The four KPI tiles: providers, volume, cost and the shared suppression list. */
function Summary({ data }: { data: Overview }) {
  const t = useT()
  const s = data.summary
  return (
    <Grid min={220} gap={16}>
      <Tile
        label={t(mo.tileProviders)}
        value={t(mo.tileConnected, { count: s.connected })}
        sub={s.byStatus.map((b) => t(mo.statusCount, { count: b.count, status: b.status })).join(' · ')}
      />
      <Tile label={t(mo.tileSent)} value={count(s.sentThisMonth)} sub={s.sentBy.map((b) => `${b.short} ${count(b.value)}`).join(' · ')} />
      <Tile
        label={t(mo.tileCost, { month: monthName(s.monthStart, 'long') })}
        value={money(s.costToDate)}
        sub={s.costPer1000 !== undefined ? t(mo.costBlended, { amount: money(s.costPer1000), thousand: count(1000) }) : t(mo.costNoMail)}
      />
      <Tile
        label={t(mo.tileSuppression)}
        value={count(s.suppressions)}
        sub={s.syncedAt ? (s.inSync ? t(mo.syncedAll, { time: clock(s.syncedAt) }) : t(mo.syncing, { time: clock(s.syncedAt) })) : t(mo.enforced)}
      />
    </Grid>
  )
}

/** Routing per stream: the split, its fallback and failover policy. */
function RoutingSummary({ routes }: { routes: Schemas['ProviderStreamRoute'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2">{t(mo.routingTitle)}</h2>
        <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="text-[13px] font-medium">
          {t(mo.routingEdit)}
        </Link>
      </div>
      <Grid min={300} gap={0}>
        {routes.map((r, i) => (
          <StreamRoute key={r.stream} r={r} last={i === routes.length - 1} />
        ))}
      </Grid>
    </section>
  )
}

function StreamRoute({ r, last }: { r: Schemas['ProviderStreamRoute']; last: boolean }) {
  const t = useT()
  const single = r.splits.length === 1
  const legend = [
    ...r.splits.map((s) => ({ color: providerColor[s.color], text: single ? t(mo.legendPrimary, { provider: s.short }) : t(mo.legendShare, { provider: s.short, share: pct(s.percent, 0) }) })),
    ...(r.fallback ? [{ color: providerColor[r.fallback.color], text: t(mo.legendFallback, { provider: r.fallback.short }) }] : []),
  ]
  const stream = streamName[r.stream as Stream]
  const label = t(mo.routeLabel, { stream, legend: legend.map((l) => l.text).join(', ') })
  return (
    <div className={cn('flex flex-col gap-2 px-5 py-4', !last && 'border-r border-line-2')}>
      <span className="font-semibold">{stream}</span>
      <div className="flex h-2.5 gap-0.5 overflow-hidden rounded-[5px]" role="img" aria-label={label}>
        {r.splits.length ? (
          r.splits.map((s) => <div key={s.providerId} style={{ width: `${s.percent}%`, background: providerColor[s.color] }} />)
        ) : (
          <div className="w-full bg-line-2" />
        )}
      </div>
      <div className="flex gap-3.5 text-[12.5px] max-md:flex-wrap">
        {legend.length ? (
          legend.map((l) => (
            <span key={l.text} className="flex items-center gap-1.5">
              <span className="dot" style={{ color: l.color }} aria-hidden="true" />
              {l.text}
            </span>
          ))
        ) : (
          <span>{t(mo.noProviderYet)}</span>
        )}
      </div>
      <span className="k">
        {r.conditions ? t(mo.routeConditionsFailover, { count: r.conditions, policy: failoverText[r.failover] }) : t(mo.routeFailover, { policy: failoverText[r.failover] })}
      </span>
    </div>
  )
}

/** Empty state: nothing connected yet. */
function NoProviders() {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('providers:manage')
  return (
    <section className="card flex flex-col items-center gap-3 border-dashed px-7 py-10 text-center">
      <svg width="56" height="56" viewBox="0 0 56 56" aria-hidden="true">
        <rect x="8" y="10" width="40" height="15" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <rect x="8" y="31" width="40" height="15" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <circle cx="16" cy="17.5" r="2" fill="#9AA3B2" />
        <circle cx="16" cy="38.5" r="2" fill="#9AA3B2" />
      </svg>
      <h2 className="m-0 text-[18px] font-bold">{t(mo.emptyTitle)}</h2>
      <span className="max-w-[520px] text-ink-2">{t(mo.emptyBody)}</span>
      <div className="mt-1.5 flex flex-wrap justify-center gap-2.5">
        {reason ? (
          <Button variant="primary" className="min-h-10" disabledReason={reason}>
            {t(mo.connectProvider)}
          </Button>
        ) : (
          <ButtonLink variant="primary" className="min-h-10" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            {t(mo.connectProvider)}
          </ButtonLink>
        )}
        <ButtonLink to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
          {t(mo.browseAdapters)}
        </ButtonLink>
      </div>
      <span className="k">{t(mo.emptyFoot)}</span>
    </section>
  )
}

/** Loading: skeletons shaped like two provider rows. */
function ProvidersSkeleton() {
  const t = useT()
  const row = (lines: number[], bars: number, chips: number[], last = false) => (
    <div className={cn('flex flex-wrap gap-5 px-[22px] py-5', !last && 'border-b border-line-2')}>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex gap-2.5">
          <Skeleton w={lines[0]} h={20} />
          <Skeleton w={80} h={22} r={999} />
        </div>
        {lines.slice(1).map((w, i) => (
          <Skeleton key={i} w={w} h={12} />
        ))}
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {Array.from({ length: bars }, (_, i) => (
          <Skeleton key={i} h={6} />
        ))}
      </div>
      <div className="flex flex-[1_1_300px] flex-wrap gap-1.5">
        {chips.map((w, i) => (
          <Skeleton key={i} w={w} h={24} />
        ))}
      </div>
    </div>
  )
  return (
    <Loading label={t(mo.loading)}>
      <div className="card flex flex-col">
        {row([130, 200, 240, 180], 4, [120, 150, 110, 130, 90])}
        {row([100, 220, 180], 3, [140, 100, 120], true)}
      </div>
    </Loading>
  )
}
