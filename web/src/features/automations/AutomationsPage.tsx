import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { FilterTabs } from '@/components/ui/Tabs'
import { count, dayMonthDate, pct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ct, RevenueCell } from '@/features/commerce/automation'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './messages'
import { AutomationStatusBadge } from './StatusBadge'
import { EmptyAutomations, StarterTemplates, TemplateGrid } from './TemplateCards'
import './automations.css'
import { SmsFlowLine } from '@/features/sms/AutomationList'

export type AutomationFilter = 'all' | 'live' | 'paused' | 'drafts'

type Item = Schemas['AutomationListItem']

/** 7.10 Automations list: performance, filters, starter templates and states. */
export function AutomationsPage({ status }: { status: AutomationFilter }) {
  const t = useT()
  const { slug } = useWorkspace()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations', { params: { path: { workspace: slug }, query: { status } } })
  const templates = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/templates', { params: { path: { workspace: slug } } }, { staleTime: Infinity })
  const [creating, setCreating] = useState(false)
  const newReason = usePermissionReason('automations:edit')
  const data = list.data
  const empty = data !== undefined && data.counts.all === 0
  // No access or no such workspace: the shell's 403/404 state, not a panel.
  if (list.error instanceof ApiError && (list.error.status === 403 || list.error.status === 404)) throw list.error
  return (
    <Page
      title={t(m.title)}
      filters={empty ? undefined : <Filters status={status} counts={data?.counts} />}
      actions={
        empty ? undefined : (
          <>
            {data ? (
              <Badge tone="purple">
                <span className="dot" style={{ color: 'var(--chart-ses)' }} aria-hidden="true" />
                {t(m.route, { route: data.route.label })}
              </Badge>
            ) : null}
            <Button variant="primary" disabledReason={newReason} onClick={() => setCreating(true)}>
              {t(m.newAutomation)}
            </Button>
          </>
        )
      }
    >
      {list.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(list.error)} {t(m.stillRunning)}
        </ErrorPanel>
      ) : list.isPending ? (
        <ListSkeleton />
      ) : empty ? (
        <EmptyAutomations templates={templates.data?.templates ?? []} routeLabel={list.data.route.label} />
      ) : (
        <Loaded data={list.data} status={status} templates={templates.data?.templates} />
      )}
      <Dialog
        open={creating}
        onOpenChange={setCreating}
        title={t(m.createTitle)}
        size="lg"
        description={data ? t(m.routeLead, { route: data.route.label }) : undefined}
      >
        <TemplateGrid templates={templates.data?.templates ?? []} />
      </Dialog>
    </Page>
  )
}

function Loaded({ data, status, templates }: { data: Schemas['AutomationList']; status: AutomationFilter; templates?: Schemas['AutomationStarterTemplate'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      {data.alerts.length > 0 ? <Alerts alerts={data.alerts} /> : null}
      <Summary />
      {data.items.length === 0 ? (
        <EmptyState
          title={t(m.noneWithStatus, { status })}
          action={
            <Link className="btn" to="/w/$workspace/automations" params={{ workspace: slug }}>
              {t(m.showAll)}
            </Link>
          }
        >
          {t(m.noneWithStatusBody, { count: data.counts.all })}
        </EmptyState>
      ) : (
        <AutomationTable items={data.items} />
      )}
      {templates ? <StarterTemplates templates={templates} /> : null}
    </>
  )
}

function Filters({ status, counts }: { status: AutomationFilter; counts?: Schemas['AutomationStatusCounts'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const label = (filter: AutomationFilter, v?: number) => (v !== undefined ? t(m.filterCount, { filter, count: count(v) }) : t(m.filter, { filter }))
  return (
    <FilterTabs
      label={t(m.filterLabel)}
      className="ml-3 max-md:ml-0"
      value={status}
      onChange={(next) => void navigate({ to: '/w/$workspace/automations', params: { workspace: slug }, search: { status: next === 'all' ? undefined : next } })}
      items={[
        { value: 'all', label: label('all', counts?.all) },
        { value: 'live', label: label('live', counts?.live) },
        { value: 'paused', label: label('paused', counts?.paused) },
        { value: 'drafts', label: label('drafts', counts?.drafts) },
      ]}
    />
  )
}

function Summary() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/summary', { params: { path: { workspace: slug }, query: { days: 7 } } })
  if (q.isPending || q.isError) {
    return (
      <Grid min={200} gap={16} aria-hidden={q.isPending}>
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="card flex flex-col gap-2 px-4.5 py-3.5">
            <Skeleton w={120} h={10} />
            <Skeleton w={80} h={20} />
            <Skeleton w={150} h={10} />
          </div>
        ))}
      </Grid>
    )
  }
  const s = q.data
  const sentBy = s.sentBy
    .map((p) =>
      p.fallback && p.lastDay ? t(m.sentByFailover, { provider: p.short, count: count(p.sent), date: dayMonthDate(`${p.lastDay}T12:00:00Z`) }) : `${p.short} ${count(p.sent)}`,
    )
    .join(' · ')
  const cards = [
    { label: t(m.inFlowsNow), value: count(s.inFlow), sub: t(m.acrossLive, { count: s.liveCount }) },
    { label: t(m.emailsSent, { days: s.days }), value: count(s.sent), sub: sentBy || t(m.noEmailsYet) },
    { label: t(m.humanOpens), value: pct(s.humanOpens), sub: t(m.clicksRate, { rate: pct(s.clicks) }) },
    { label: t(m.goalsReached, { days: s.days }), value: count(s.goals), sub: s.goalsBy.map((g) => `${g.name} ${count(g.count)}`).join(' · ') || t(m.noneYet) },
  ]
  return (
    <Grid min={200} gap={16}>
      {cards.map((c) => (
        <div key={c.label} className="card flex flex-col gap-0.5 px-4.5 py-3.5">
          <span className="k">{c.label}</span>
          <span className="n text-[22px]">{c.value}</span>
          <span className="k">{c.sub}</span>
        </div>
      ))}
    </Grid>
  )
}

/** Whole percents, or one decimal under 20% where the decimal matters. */
function goalPct(v: number): string {
  return v >= 20 ? pct(Math.round(v), 0) : pct(v, 1)
}

function subtitle(a: Item): string {
  if (a.status === 'paused' && a.pausedAt) {
    return tt(m.pausedLine, {
      version: a.liveVersion ?? '',
      date: dayMonthDate(a.pausedAt),
      hasBy: a.pausedBy ? 'yes' : 'no',
      name: a.pausedBy ?? '',
      hasReason: a.pauseReason ? 'yes' : 'no',
      reason: a.pauseReason ?? '',
    })
  }
  if (a.status === 'draft' || a.liveVersion === undefined) {
    return [tt(m.draftWord), a.description].filter(Boolean).join(' · ')
  }
  return [`v${a.liveVersion}`, a.publishedAt ? tt(m.publishedOn, { date: dayMonthDate(a.publishedAt) }) : '', a.description].filter(Boolean).join(' · ')
}

function AutomationTable({ items }: { items: Item[] }) {
  const { slug } = useWorkspace()
  const t = useT()
  const dash = (
    <Td align="right" className="k">
      –
    </Td>
  )
  return (
    <section className="card" aria-label={t(m.tableLabel)}>
      <Table phoneWrap fit className="au-list [&_td:first-child]:min-w-[150px] [&_td:nth-child(3)]:min-w-[150px] [&_td:nth-child(9)]:min-w-[150px]">
        <thead>
          <tr>
            <Th>{t(m.colAutomation)}</Th>
            <Th>{t(m.colStatus)}</Th>
            <Th>{t(m.colTrigger)}</Th>
            <Th align="right">{t(m.colSteps)}</Th>
            <Th align="right">{t(m.colEntered)}</Th>
            <Th align="right">{t(m.colInFlow)}</Th>
            <Th align="right">{t(m.colCompleted)}</Th>
            <Th align="right">{t(m.colHumanOpens)}</Th>
            <Th align="right">{t(m.colGoal)}</Th>
            <Th align="right">{t(ct.revenueColumn)}</Th>
            <Th>
              <span className="vh">{t(m.colActions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {items.map((a) => {
            const params = { workspace: slug, automationId: a.id }
            return (
              <tr key={a.id}>
                <Td>
                  <Link to="/w/$workspace/automations/$automationId" params={params} className="font-semibold text-ink no-underline hover:text-ink">
                    {a.name}
                  </Link>
                  <div className="k">{subtitle(a)}</div>
                  {/* Phase 8: the channels a flow sends on, and what its texts cost. */}
                  <SmsFlowLine channels={a.channels} cost={a.channelCost} />
                </Td>
                <Td>
                  <AutomationStatusBadge status={a.status} />
                </Td>
                <Td>{a.trigger}</Td>
                <Td align="right" num>
                  {a.steps}
                </Td>
                {a.entered !== undefined ? (
                  <Td align="right" num>
                    {count(a.entered)}
                  </Td>
                ) : (
                  dash
                )}
                {a.inFlow !== undefined ? (
                  <Td align="right" num>
                    {count(a.inFlow)}
                  </Td>
                ) : (
                  dash
                )}
                {a.completed !== undefined ? (
                  <Td align="right" num>
                    {pct(Math.round(a.completed), 0)}
                  </Td>
                ) : (
                  dash
                )}
                {a.humanOpens !== undefined ? (
                  <Td align="right" num>
                    {pct(a.humanOpens)}
                  </Td>
                ) : (
                  dash
                )}
                {a.goal ? (
                  a.goal.percent !== undefined ? (
                    <Td align="right" num>
                      {a.goal.label} · {goalPct(a.goal.percent)}
                    </Td>
                  ) : (
                    <Td align="right" className="k">
                      {a.goal.label}
                    </Td>
                  )
                ) : (
                  dash
                )}
                {a.revenue && a.revenue.orders > 0 ? (
                  <Td align="right" num>
                    <RevenueCell revenue={a.revenue} />
                  </Td>
                ) : (
                  dash
                )}
                <Td>
                  <ButtonLink size="sm" to="/w/$workspace/automations/$automationId" params={params}>
                    {a.status === 'draft' ? t(m.edit) : t(m.open)}
                  </ButtonLink>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
    </section>
  )
}

/** Flow problems: sending held for want of a healthy provider, or a failing step. */
function Alerts({ alerts }: { alerts: Schemas['AutomationAlert'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Grid min={380} gap={16}>
      {alerts.map((a) => (
        <div key={a.id} role="alert" className="card flex flex-col gap-2 px-4.5 py-4" style={{ borderColor: a.kind === 'sending_held' ? 'var(--bad-line)' : '#FCD34D' }}>
          <div className="flex items-center gap-2.5">
            {a.kind === 'sending_held' ? (
              <Badge tone="bad" dot>
                {t(m.pausedAutomatically)}
              </Badge>
            ) : (
              <Badge tone="warn">{t(m.stepFailing)}</Badge>
            )}
            <span className="font-semibold">{a.title}</span>
          </div>
          <span className="k">{a.detail}</span>
          <div className="flex flex-wrap gap-2">
            {a.actions.map((act, i) => {
              const variant = i === 0 ? 'primary' : 'default'
              if (act.kind === 'providers') {
                return (
                  <ButtonLink key={act.label} size="sm" variant={variant} to="/w/$workspace/providers" params={{ workspace: slug }}>
                    {act.label}
                  </ButtonLink>
                )
              }
              if (act.kind === 'routing') {
                return (
                  <ButtonLink key={act.label} size="sm" variant={variant} to="/w/$workspace/providers/routing" params={{ workspace: slug }}>
                    {act.label}
                  </ButtonLink>
                )
              }
              return (
                <ButtonLink
                  key={act.label}
                  size="sm"
                  variant={variant}
                  to="/w/$workspace/automations/$automationId"
                  params={{ workspace: slug, automationId: a.automationId }}
                  search={{ step: act.nodeId || undefined, panel: act.kind === 'step_log' ? 'stats' : undefined }}
                >
                  {act.label}
                </ButtonLink>
              )
            })}
          </div>
        </div>
      ))}
    </Grid>
  )
}

/** Loading: rows shaped like the table (AutomationsStates › Loading). */
function ListSkeleton() {
  const t = useT()
  const rows = [
    [180, 160],
    [220, 190],
    [150, 170],
  ]
  return (
    <section className="card flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingList)}
      </span>
      {rows.map(([name, trigger], i) => (
        <div key={i} className="ver" style={i === 0 ? { borderTop: 0 } : undefined}>
          <Skeleton w={name} h={14} />
          <Skeleton w={60} h={22} r={999} />
          <Skeleton w={trigger} h={14} />
          <Skeleton w={60} h={14} className="ml-auto" />
          <Skeleton w={60} h={14} />
        </div>
      ))}
    </section>
  )
}
