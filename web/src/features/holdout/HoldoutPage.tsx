import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Loading, Skeleton, Spinner } from '@/components/ui/Feedback'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { AudienceTabs } from '@/features/audience/AudienceTabs'
import { money } from '@/features/earn/format'
import { formatList, T, t, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { ago, count, dayMonthDate, decimal, pct, trimPct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './holdout.css'

type Results = Schemas['HoldoutResults']
type Metric = Schemas['HoldoutMetric']
type Window = Results['window']

export interface HoldoutSearch {
  window?: Window
  period?: string
}

const resultsPath = '/api/v1/workspaces/{workspace}/holdout/results' as const

/** "+18%", "−8%": a lift rounded to a whole percent, with its sign. */
export function liftText(lift: number): string {
  const r = Math.round(lift)
  if (r === 0) return pct(0, 0)
  return r > 0 ? `+${pct(r, 0)}` : `−${pct(-r, 0)}`
}

/** "96%": confidence rounded down (never overstated), at most 99%. */
export function confidenceText(confidence: number): string {
  return pct(Math.min(99, Math.floor(confidence * 100 + 1e-9)), 0)
}

/** A metric's value in its unit: a rate in percent, money per contact, orders per contact. */
function metricValue(x: Metric, currency: string, side: 'holdout' | 'marketing'): string {
  const v = x[side]
  if (x.key === 'conversion_rate' || x.key === 'site_visitors') return pct(v, 1)
  if (x.key === 'revenue_per_contact') return money(v, currency, 'always')
  return decimal(v, 3)
}

/**
 * Audience › Holdout (M63 G-33): what marketing adds, measured against the
 * holdout group. The verdict first, then a tile per measure with its lift
 * and confidence, the two groups side by side (chart or table), and how
 * the numbers are counted.
 */
export function HoldoutPage({ search }: { search: HoldoutSearch }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const window = search.window ?? 'period'
  const q = $api.useQuery(
    'get',
    resultsPath,
    { params: { path: { workspace: slug }, query: { window, period: search.period } } },
    {
      placeholderData: (prev) => prev,
      // While a recompute runs, look again until the new results are in.
      refetchInterval: (query) => (query.state.data?.computing ? 5000 : false),
    },
  )
  const d = q.data
  if (!d && q.error instanceof ApiError && q.error.status === 403) throw q.error
  const go = (patch: Partial<HoldoutSearch>) =>
    void navigate({ to: '/w/$workspace/audience/holdout', params: { workspace: slug }, search: { ...search, ...patch }, replace: true })
  return (
    <Page title={tr(m.audience)} documentTitle={tr(m.documentTitle)} filters={<AudienceTabs />} contentClassName="ho-page">
      {!d ? (
        q.isError ? (
          <LoadError onRetry={() => void q.refetch()} />
        ) : (
          <HoldoutSkeleton />
        )
      ) : d.verdict === 'none' || !d.period ? (
        <NoHoldout />
      ) : (
        <HoldoutBody d={d} onWindow={(w) => go({ window: w === 'period' ? undefined : w })} onMeasured={() => void q.refetch()} />
      )}
    </Page>
  )
}

function HoldoutBody({ d, onWindow, onMeasured }: { d: Results; onWindow: (w: Window) => void; onMeasured: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const period = d.period!
  const latest = d.periods[0]
  const ended = !period.open && !(latest?.open ?? false)
  const pending = d.verdict === 'pending'
  return (
    <>
      {ended && period.endedAt ? (
        <div className="note n" role="status">
          <Icon name="info" size={16} />
          <div className="flex min-w-0 flex-[1_1_320px] flex-wrap items-center gap-x-3 gap-y-1">
            <span>{tr(m.off, { from: dayMonthDate(period.startedAt), to: dayMonthDate(period.endedAt) })}</span>
            <span>
              <Link to="/w/$workspace/settings/sending-rules" params={{ workspace: slug }} className="font-semibold">
                {tr(m.turnOn)}
              </Link>
            </span>
          </div>
        </div>
      ) : null}
      {ended ? null : <Toolbar d={d} onWindow={onWindow} onMeasured={onMeasured} />}
      <Verdict d={d} />
      {pending || d.verdict === 'too_few' ? null : (
        <>
          <Tiles d={d} />
          <Comparison d={d} />
          {d.campaignsIncluded > 0 ? (
            <div className="note i" role="note">
              <Icon name="info" size={16} />
              <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
                <span className="font-bold">{tr(m.includedTitle, { count: count(d.campaignsIncluded) })}</span>
                <div>{tr(m.includedBody)}</div>
              </div>
            </div>
          ) : null}
          <div className="flex flex-col gap-1">
            <span className="k">{tr(m.converterNote, { currency: d.currency.toUpperCase() })}</span>
            {/* Site visitors come from Website tracking (M62), when a site is tracked. */}
            {d.tracking ? <span className="k">{tr(m.visitorNote)}</span> : null}
            <span className="k">{tr(m.alwaysNote)}</span>
            <Link to="/w/$workspace/settings/sending-rules" params={{ workspace: slug }} className="ho-lnk self-start">
              {tr(m.settingsLink)}
            </Link>
          </div>
        </>
      )}
    </>
  )
}

/** Window (Since … · Last 30 days · Last 90 days), when the results were computed, Recompute. */
function Toolbar({ d, onWindow, onMeasured }: { d: Results; onWindow: (w: Window) => void; onMeasured: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState(false)
  const computing = d.computing || busy
  const refused = () =>
    toast({ kind: 'info', title: t(m.recomputeSoon, { ago: d.computedAt ? ago(d.computedAt) : ago(null) }), detail: t(m.recomputeEvery) })
  const recompute = async () => {
    if (d.nextMeasureAt && new Date(d.nextMeasureAt).getTime() > Date.now()) {
      refused()
      return
    }
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/holdout/measure', { params: { path: { workspace: slug } } })
      onMeasured()
    } catch (e) {
      if (e instanceof ApiError && e.status === 429) refused()
      else toast({ kind: 'error', title: t(m.recomputeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5">
      <Segmented
        label={tr(m.window)}
        className="ho-seg"
        value={d.window}
        onChange={onWindow}
        items={[
          { value: 'period', label: tr(m.since, { date: dayMonthDate(d.period!.startedAt) }) },
          { value: '30d', label: tr(m.last30) },
          { value: '90d', label: tr(m.last90) },
        ]}
      />
      <div className="flex flex-wrap items-center gap-x-3.5 gap-y-2.5">
        {computing ? (
          <span className="k inline-flex items-center gap-2" role="status" id="ho-computing">
            <Spinner size={14} />
            {tr(m.computing)}
          </span>
        ) : d.computedAt ? (
          <span className="k">{tr(m.computedAgo, { ago: ago(d.computedAt) })}</span>
        ) : null}
        <Button disabledReason={computing ? tr(m.computing) : undefined} aria-describedby={computing ? 'ho-computing' : undefined} onClick={() => void recompute()}>
          {tr(m.recompute)}
        </Button>
      </div>
    </div>
  )
}

/** The verdict on revenue per contact, in one sentence, and who was compared. */
function Verdict({ d }: { d: Results }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const period = d.period!
  const rev = d.metrics.find((x) => x.key === 'revenue_per_contact')
  let sentence: string
  switch (d.verdict) {
    case 'added':
      sentence = tr(m.added, { lift: pct(Math.round(rev?.lift ?? 0), 0), confidence: confidenceText(rev?.confidence ?? 0) })
      break
    case 'lost': {
      // The holdout group's lead over the marketing group.
      const lead = rev && rev.marketing > 0 ? ((rev.holdout - rev.marketing) / rev.marketing) * 100 : 0
      sentence = tr(m.lost, { lift: pct(Math.round(lead), 0), confidence: confidenceText(rev?.confidence ?? 0) })
      break
    }
    case 'too_few':
      sentence = tr(m.tooFew, { count: count(d.holdout.contacts), needed: count(d.minHoldout) })
      break
    case 'too_early':
      sentence = tr(m.tooEarly, { date: dayMonthDate(period.startedAt) })
      break
    case 'pending':
      sentence = tr(m.pending)
      break
    default:
      sentence = tr(m.noDifference)
  }
  const scope = period.campaigns && period.automations ? 'both' : period.campaigns ? 'campaigns' : 'automations'
  return (
    <section className="card pad flex flex-col gap-1.5" aria-labelledby="ho-v">
      <h2 id="ho-v" className="m-0 text-[20px] leading-[1.3] font-semibold tracking-[-0.01em]">
        {sentence}
      </h2>
      {d.verdict === 'too_few' ? (
        <span className="k">
          <T
            msg={m.tooFewHelp}
            values={{
              link: (c) => (
                <Link to="/w/$workspace/settings/sending-rules" params={{ workspace: slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        </span>
      ) : d.verdict !== 'pending' && d.from && d.to ? (
        <span className="k">
          {tr(m.groups, {
            holdout: d.holdout.contacts,
            marketing: d.marketing.contacts,
            percent: trimPct(period.percent),
            scope,
            from: dayMonthDate(d.from),
            to: dayMonthDate(d.to),
          })}
        </span>
      ) : null}
    </section>
  )
}

const tileLabel: Record<Metric['key'], typeof m.conversionRate> = {
  conversion_rate: m.conversionRate,
  revenue_per_contact: m.revenuePerContact,
  orders_per_contact: m.ordersPerContact,
  site_visitors: m.siteVisitors,
}

/** A tile per measure: the marketing group's value, the holdout's, the lift and its confidence. */
function Tiles({ d }: { d: Results }) {
  const tr = useT()
  const busy = d.computing
  return (
    <div className={cn('ho-tiles', d.metrics.length > 3 && 'four')} role="group" aria-busy={busy || undefined} aria-label={busy ? tr(m.computing) : undefined}>
      {d.metrics.map((x) => (
        <div key={x.key} className={cn('card ho-stat', busy && 'dim')}>
          <span className="k">{tr(tileLabel[x.key])}</span>
          <span className="v">
            <T msg={m.marketingValue} values={{ value: metricValue(x, d.currency, 'marketing'), vh: (c) => <span className="vh">{c}</span> }} />
          </span>
          <span className="s">{tr(m.holdoutValue, { value: metricValue(x, d.currency, 'holdout') })}</span>
          {x.tested ? (
            <Badge tone={busy ? 'neutral' : x.significant ? (x.lift > 0 ? 'ok' : 'warn') : 'neutral'} dot className="mt-1.5 self-start">
              {tr(m.liftConfidence, { lift: liftText(x.lift), confidence: confidenceText(x.confidence) })}
            </Badge>
          ) : null}
        </div>
      ))}
    </div>
  )
}

/** Revenue per contact, the two groups as bars with their values, or as a table. */
function Comparison({ d }: { d: Results }) {
  const tr = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const rev = d.metrics.find((x) => x.key === 'revenue_per_contact')
  const conv = d.metrics.find((x) => x.key === 'conversion_rate')
  const mk = rev?.marketing ?? 0
  const ho = rev?.holdout ?? 0
  const max = Math.max(mk, ho)
  const mkText = money(mk, d.currency, 'always')
  const hoText = money(ho, d.currency, 'always')
  const others = d.otherCurrencies.filter((o) => o.holdoutCents + o.marketingCents !== 0)
  const bar = (label: string, value: string, v: number, color: string): ReactNode => (
    <div className="flex flex-col gap-1.5">
      <span className="flex justify-between gap-2.5 text-[13px]">
        <span className="font-semibold">{label}</span>
        <span className="num font-bold">{value}</span>
      </span>
      <div className="bar ho-bar">
        <i style={{ width: `${max > 0 ? Math.max(0, (v / max) * 100) : 0}%`, background: color }} />
      </div>
    </div>
  )
  return (
    <section className="card" aria-labelledby="ho-cmp">
      <div className="sec">
        <h2 className="h2" id="ho-cmp">
          {tr(m.revenuePerContact)}
        </h2>
        <Segmented
          label={tr(m.view)}
          value={view}
          onChange={setView}
          items={[
            { value: 'chart', label: tr(m.chart) },
            { value: 'table', label: tr(m.table) },
          ]}
        />
      </div>
      {view === 'chart' ? (
        <div className="pad">
          <div role="img" aria-label={tr(m.chartLabel, { marketing: mkText, holdout: hoText })} className="flex flex-col gap-3.5">
            {bar(tr(m.marketingGroup), mkText, mk, 'var(--accent)')}
            {bar(tr(m.holdoutGroup), hoText, ho, 'var(--chart-human)')}
          </div>
          {others.length ? (
            <span className="k mt-3 block">
              {tr(m.otherCurrencies, {
                amounts: formatList(
                  others.map((o) => t(m.otherCurrency, { amount: money(o.holdoutCents + o.marketingCents, o.currency, 'never'), currency: o.currency.toUpperCase() })),
                ),
              })}
            </span>
          ) : null}
        </div>
      ) : (
        <TableScroll>
          <table className="tbl">
            <caption className="vh">{tr(m.tableCaption)}</caption>
            <thead>
              <tr>
                <th scope="col">{tr(m.colGroup)}</th>
                <th scope="col" className="r">
                  {tr(m.colContacts)}
                </th>
                <th scope="col" className="r">
                  {tr(m.colConverters)}
                </th>
                <th scope="col" className="r">
                  {tr(m.colConversionRate)}
                </th>
                <th scope="col" className="r">
                  {tr(m.colOrders)}
                </th>
                <th scope="col" className="r">
                  {tr(m.colRevenue)}
                </th>
                <th scope="col" className="r">
                  {tr(m.colRevenuePerContact)}
                </th>
                {d.tracking ? (
                  <th scope="col" className="r">
                    {tr(m.colSiteVisitors)}
                  </th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {(
                [
                  [m.marketingGroup, d.marketing, conv?.marketing, mk],
                  [m.holdoutGroup, d.holdout, conv?.holdout, ho],
                ] as const
              ).map(([label, g, rate, perContact]) => (
                <tr key={label.id}>
                  <td className="nw font-semibold">{tr(label)}</td>
                  <td className="r num">{count(g.contacts)}</td>
                  <td className="r num">{count(g.converters)}</td>
                  <td className="r num">{pct(rate ?? 0, 1)}</td>
                  <td className="r num">{count(g.orders)}</td>
                  <td className="r num">{money(g.revenueCents, d.currency, 'never')}</td>
                  <td className="r num">{money(perContact, d.currency, 'always')}</td>
                  {d.tracking ? <td className="r num">{count(g.visitors)}</td> : null}
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
    </section>
  )
}

function HoldoutSkeleton() {
  const tr = useT()
  return (
    <Loading label={tr(m.loading)} className="flex flex-col gap-5">
      <div className="card pad flex flex-col gap-2.5">
        <Skeleton w="62%" h={22} />
        <Skeleton w="84%" h={12} />
      </div>
      <div className="ho-tiles">
        {[0, 1, 2].map((i) => (
          <div key={i} className="card ho-stat gap-2">
            <Skeleton w={110} h={10} />
            <Skeleton w={72} h={22} />
            <Skeleton w={90} h={10} />
            <Skeleton w={150} h={20} r={999} />
          </div>
        ))}
      </div>
    </Loading>
  )
}

/** Opened before the workspace ever had a holdout group. */
function NoHoldout() {
  const tr = useT()
  const { slug } = useWorkspace()
  return (
    <div className="card ho-empty">
      <svg width="72" height="56" viewBox="0 0 72 56" aria-hidden="true">
        <path d="M6 46h60" stroke="var(--ink-4)" strokeWidth="2" />
        <rect x="14" y="16" width="14" height="30" rx="2" fill="var(--line-strong)" />
        <rect x="34" y="22" width="14" height="24" rx="2" fill="var(--line-strong)" />
        <rect x="54" y="24" width="10" height="22" rx="2" fill="none" stroke="var(--accent)" strokeWidth="2" strokeDasharray="3 3" />
      </svg>
      <h2 className="tt">{tr(m.noneTitle)}</h2>
      <ButtonLink variant="primary" to="/w/$workspace/settings/sending-rules" params={{ workspace: slug }}>
        {tr(m.setUp)}
      </ButtonLink>
    </div>
  )
}

/** The results didn't load: what failed, what ReputeCast kept, what to do. */
function LoadError({ onRetry }: { onRetry: () => void }) {
  const tr = useT()
  return (
    <div className="card ho-error" role="alert">
      <Icon name="alertCircle" size={28} className="text-bad-strong" />
      <span className="text-[16px] font-bold">{tr(m.loadFailed)}</span>
      <span className="k max-w-[420px] text-ink-2">{tr(m.loadFailedKept)}</span>
      <span className="k max-w-[420px] text-ink-2">{tr(m.loadFailedDo)}</span>
      <div className="flex gap-2">
        <Button variant="primary" onClick={onRetry}>
          {tr(m.retry)}
        </Button>
      </div>
    </div>
  )
}
