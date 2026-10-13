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
import { AutomationStatusBadge } from './StatusBadge'
import { EmptyAutomations, StarterTemplates, TemplateGrid } from './TemplateCards'
import './automations.css'

export type AutomationFilter = 'all' | 'live' | 'paused' | 'drafts'

type Item = Schemas['AutomationListItem']

/** 7.10 Automations list: performance, filters, starter templates and states. */
export function AutomationsPage({ status }: { status: AutomationFilter }) {
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
      title="Automations"
      filters={empty ? undefined : <Filters status={status} counts={data?.counts} />}
      actions={
        empty ? undefined : (
          <>
            {data ? (
              <Badge tone="purple">
                <span className="dot" style={{ color: 'var(--chart-ses)' }} aria-hidden="true" />
                Automations route · {data.route.label}
              </Badge>
            ) : null}
            <Button variant="primary" disabledReason={newReason} onClick={() => setCreating(true)}>
              New automation
            </Button>
          </>
        )
      }
    >
      {list.isError ? (
        <ErrorPanel title="Couldn't load automations" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
          {errorMessage(list.error)} Flows keep running; only this page failed to load.
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
        title="Start with a template or a blank canvas"
        size="lg"
        description={data ? `Flows send on the Automations route (${data.route.label}) and fail over automatically, so a provider outage never stalls a welcome email.` : undefined}
      >
        <TemplateGrid templates={templates.data?.templates ?? []} />
      </Dialog>
    </Page>
  )
}

function Loaded({ data, status, templates }: { data: Schemas['AutomationList']; status: AutomationFilter; templates?: Schemas['AutomationStarterTemplate'][] }) {
  const { slug } = useWorkspace()
  return (
    <>
      {data.alerts.length > 0 ? <Alerts alerts={data.alerts} /> : null}
      <Summary />
      {data.items.length === 0 ? (
        <EmptyState
          title={`No ${status === 'drafts' ? 'drafts' : status} automations`}
          action={
            <Link className="btn" to="/w/$workspace/automations" params={{ workspace: slug }}>
              Show all automations
            </Link>
          }
        >
          {count(data.counts.all)} automations in this workspace. Pick another filter to see them.
        </EmptyState>
      ) : (
        <AutomationTable items={data.items} />
      )}
      {templates ? <StarterTemplates templates={templates} /> : null}
    </>
  )
}

function Filters({ status, counts }: { status: AutomationFilter; counts?: Schemas['AutomationStatusCounts'] }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const n = (v?: number) => (v !== undefined ? ` · ${count(v)}` : '')
  return (
    <FilterTabs
      label="Filter"
      className="ml-3 max-md:ml-0"
      value={status}
      onChange={(next) => void navigate({ to: '/w/$workspace/automations', params: { workspace: slug }, search: { status: next === 'all' ? undefined : next } })}
      items={[
        { value: 'all', label: `All${n(counts?.all)}` },
        { value: 'live', label: `Live${n(counts?.live)}` },
        { value: 'paused', label: `Paused${n(counts?.paused)}` },
        { value: 'drafts', label: `Drafts${n(counts?.drafts)}` },
      ]}
    />
  )
}

function Summary() {
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
    .map((p) => `${p.short} ${count(p.sent)}${p.fallback && p.lastDay ? ` (failover on ${dayMonthDate(`${p.lastDay}T12:00:00Z`)})` : ''}`)
    .join(' · ')
  const cards = [
    { label: 'Contacts in flows now', value: count(s.inFlow), sub: `across ${s.liveCount} live ${s.liveCount === 1 ? 'automation' : 'automations'}` },
    { label: `Emails sent · ${s.days} d`, value: count(s.sent), sub: sentBy || 'no emails yet' },
    { label: 'Human opens', value: pct(s.humanOpens), sub: `clicks ${pct(s.clicks)}` },
    { label: `Goals reached · ${s.days} d`, value: count(s.goals), sub: s.goalsBy.map((g) => `${g.name} ${count(g.count)}`).join(' · ') || 'none yet' },
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
  return v >= 20 ? `${Math.round(v)}%` : `${v.toFixed(1)}%`
}

function subtitle(a: Item): string {
  if (a.status === 'paused' && a.pausedAt) {
    return `v${a.liveVersion} · paused ${dayMonthDate(a.pausedAt)}${a.pausedBy ? ` by ${a.pausedBy}` : ''}${a.pauseReason ? `: ${a.pauseReason}` : ''}`
  }
  if (a.status === 'draft' || a.liveVersion === undefined) {
    return ['draft', a.description].filter(Boolean).join(' · ')
  }
  return [`v${a.liveVersion}`, a.publishedAt ? `published ${dayMonthDate(a.publishedAt)}` : '', a.description].filter(Boolean).join(' · ')
}

function AutomationTable({ items }: { items: Item[] }) {
  const { slug } = useWorkspace()
  const dash = (
    <Td align="right" className="k">
      –
    </Td>
  )
  return (
    <section className="card" aria-label="Automations">
      <Table
        phoneWrap
        className="[&_td]:whitespace-nowrap [&_td:first-child]:min-w-[150px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[150px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(9)]:min-w-[150px] [&_td:nth-child(9)]:whitespace-normal"
      >
        <thead>
          <tr>
            <Th>Automation</Th>
            <Th>Status</Th>
            <Th>Trigger</Th>
            <Th align="right">Steps</Th>
            <Th align="right">Entered · 30 d</Th>
            <Th align="right">In flow</Th>
            <Th align="right">Completed</Th>
            <Th align="right">Human opens</Th>
            <Th align="right">Goal</Th>
            <Th>
              <span className="vh">Actions</span>
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
                    {Math.round(a.completed)}%
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
                <Td>
                  <ButtonLink size="sm" to="/w/$workspace/automations/$automationId" params={params}>
                    {a.status === 'draft' ? 'Edit' : 'Open'}
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
  const { slug } = useWorkspace()
  return (
    <Grid min={380} gap={16}>
      {alerts.map((a) => (
        <div key={a.id} role="alert" className="card flex flex-col gap-2 px-4.5 py-4" style={{ borderColor: a.kind === 'sending_held' ? 'var(--bad-line)' : '#FCD34D' }}>
          <div className="flex items-center gap-2.5">
            {a.kind === 'sending_held' ? (
              <Badge tone="bad" dot>
                Paused automatically
              </Badge>
            ) : (
              <Badge tone="warn">Step failing</Badge>
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
  const rows = [
    [180, 160],
    [220, 190],
    [150, 170],
  ]
  return (
    <section className="card flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        Loading automations and their 30-day stats
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
