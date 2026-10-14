import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { FilterTabs } from '@/components/ui/Tabs'
import { clock, count, money, pct, rate, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { RouteBadges } from './RouteBadges'
import { CampaignStatusBadge } from './StatusBadge'

export type CampaignFilter = 'all' | 'drafts' | 'scheduled' | 'sending' | 'sent'

export interface CampaignsSearch {
  status?: CampaignFilter
  q?: string
  page?: number
}

const PAGE_SIZE = 25

/** 7.7 Campaigns list: KPIs for the last 30 days, filters and the table. */
export function CampaignsPage({ search }: { search: CampaignsSearch }) {
  const { slug } = useWorkspace()
  const status = search.status ?? 'all'
  const page = search.page ?? 1
  const newReason = usePermissionReason('campaigns:edit')
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns',
    { params: { path: { workspace: slug }, query: { status, q: search.q ?? '', page, pageSize: PAGE_SIZE } } },
    { placeholderData: keepPreviousData },
  )
  const counts = list.data?.counts
  return (
    <Page
      title="Campaigns"
      filters={<CampaignFilters status={status} counts={counts} q={search.q} />}
      actions={
        <>
          <SearchBox initial={search.q ?? ''} status={status} />
          {newReason ? (
            <Button variant="primary" disabledReason={newReason}>
              New campaign
            </Button>
          ) : (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
              New campaign
            </ButtonLink>
          )}
        </>
      }
    >
      <SummaryCards />
      {list.isError ? (
        <ErrorPanel title="Couldn't load campaigns" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
          {errorMessage(list.error)} Nothing was changed.
        </ErrorPanel>
      ) : list.isPending ? (
        <TableSkeleton />
      ) : list.data.total === 0 ? (
        search.q || status !== 'all' ? (
          <EmptyState
            title={search.q ? `No campaigns match “${search.q}”` : `No ${status} campaigns`}
            action={
              <Link className="btn" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                Show all campaigns
              </Link>
            }
          >
            {list.data.counts.all} campaigns in this workspace. Clear the search or pick another filter.
          </EmptyState>
        ) : (
          <EmptyState
            title="No campaigns yet"
            action={
              newReason ? null : (
                <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
                  New campaign
                </ButtonLink>
              )
            }
          >
            Your campaigns appear here with their audience, sending route and results. Start from a template or a blank MJML canvas.
          </EmptyState>
        )
      ) : (
        <CampaignTable data={list.data} page={page} status={status} q={search.q} />
      )}
    </Page>
  )
}

function CampaignFilters({ status, counts, q }: { status: CampaignFilter; counts?: Schemas['CampaignStatusCounts']; q?: string }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const n = (v?: number) => (v !== undefined ? ` · ${count(v)}` : '')
  return (
    <FilterTabs
      label="Filter"
      className="ml-3 max-md:ml-0"
      value={status}
      onChange={(next) =>
        void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: { status: next === 'all' ? undefined : next, q: q || undefined } })
      }
      items={[
        { value: 'all', label: `All${n(counts?.all)}` },
        { value: 'drafts', label: `Drafts${n(counts?.drafts)}` },
        { value: 'scheduled', label: `Scheduled${n(counts?.scheduled)}` },
        { value: 'sending', label: `Sending${n(counts?.sending)}` },
        { value: 'sent', label: `Sent${n(counts?.sent)}` },
      ]}
    />
  )
}

function SearchBox({ initial, status }: { initial: string; status: CampaignFilter }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [value, setValue] = useState(initial)
  useEffect(() => {
    if (value === initial) return
    const h = window.setTimeout(() => {
      void navigate({
        to: '/w/$workspace/campaigns',
        params: { workspace: slug },
        search: { status: status === 'all' ? undefined : status, q: value.trim() || undefined },
        replace: true,
      })
    }, 250)
    return () => window.clearTimeout(h)
  }, [value, initial, navigate, slug, status])
  return (
    <>
      <label className="vh" htmlFor="campaign-search">
        Search campaigns
      </label>
      <input id="campaign-search" type="search" className="in w-[220px] max-md:w-full" placeholder="Search campaigns" value={value} onChange={(e) => setValue(e.target.value)} />
    </>
  )
}

function SummaryCards() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/summary', { params: { path: { workspace: slug }, query: { days: 30 } } })
  if (q.isPending || q.isError) {
    return (
      <Grid min={200} gap={16} aria-hidden={q.isPending}>
        {Array.from({ length: 5 }, (_, i) => (
          <div key={i} className="card flex flex-col gap-2 px-4.5 py-3.5">
            <Skeleton w={110} h={10} />
            <Skeleton w={90} h={20} />
            <Skeleton w={140} h={10} />
          </div>
        ))}
      </Grid>
    )
  }
  const s = q.data
  const cards = [
    { label: `Sent · last ${s.days} days`, value: count(s.sent), sub: `${s.campaigns} ${s.campaigns === 1 ? 'campaign' : 'campaigns'}` },
    { label: 'Delivered', value: pct(s.delivered), sub: `bounces ${pct(s.bounces)} · complaints ${rate(s.complaints)}` },
    { label: 'Human opens', value: pct(s.humanOpens), sub: 'machine opens shown separately' },
    { label: 'Clicks', value: pct(s.clicks), sub: `click-to-open ${pct(s.clickToOpen)}` },
    { label: `Cost · last ${s.days} days`, value: money(s.cost), sub: s.costBy.map((c) => `${c.short} ${money(c.cost, c.cost % 1 === 0 ? 0 : 2)}`).join(' · ') || 'no sends' },
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

function subtitle(c: Schemas['CampaignListItem']) {
  const parts: string[] = []
  if (c.status === 'scheduled' && c.scheduledAt) {
    parts.push(`${shortDate(c.scheduledAt)} · ${clock(c.scheduledAt)}${c.timezone ? ` ${c.timezone}` : ''}`)
  } else if (c.startedAt) {
    parts.push(`${shortDate(c.startedAt)} · ${clock(c.startedAt)}`)
  } else if (c.updatedAt !== c.createdAt && c.updatedBy) {
    parts.push(`edited ${clock(c.updatedAt)} by ${c.updatedBy}`)
  } else {
    parts.push(`created ${clock(c.createdAt)}`)
  }
  if (c.origin) parts.push(`imported from ${c.origin[0]!.toUpperCase()}${c.origin.slice(1)}`)
  parts.push(...c.notes)
  return parts.join(' · ')
}

function CampaignTable({ data, page, status, q }: { data: Schemas['CampaignList']; page: number; status: CampaignFilter; q?: string }) {
  const { slug } = useWorkspace()
  const from = (page - 1) * data.pageSize
  const shown = from + data.items.length
  const params = (id: string) => ({ workspace: slug, campaignId: id })
  return (
    <section className="card" aria-label="Campaigns">
      {/* The board's key and provider cells wrap; a long audience ("Weekly Digest + Product
          Updates") wraps too, so the actions stay in view at 1440 as on the board. */}
      <Table
        phoneWrap
        className="[&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[86px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(8)]:min-w-[120px] [&_td:nth-child(8)]:whitespace-normal [&_td:nth-child(8)_.badge]:my-px [&_td]:whitespace-nowrap"
      >
        <thead>
          <tr>
            <Th>Campaign</Th>
            <Th>Status</Th>
            <Th>Audience</Th>
            <Th align="right">Recipients</Th>
            <Th align="right">Delivered</Th>
            <Th align="right">Human opens</Th>
            <Th align="right">Clicks</Th>
            <Th>Sending providers</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((c) => {
            const sent = ['sent', 'sending', 'paused', 'cancelled', 'failed'].includes(c.status)
            const editable = ['draft', 'approval', 'approved', 'scheduled'].includes(c.status)
            return (
              <tr key={c.id}>
                <Td>
                  {editable ? (
                    <Link to="/w/$workspace/campaigns/$campaignId/edit" params={params(c.id)} className="font-semibold text-ink no-underline">
                      {c.name}
                    </Link>
                  ) : c.status === 'sending' || c.status === 'paused' ? (
                    <Link to="/w/$workspace/campaigns/$campaignId/live" params={params(c.id)} className="font-semibold text-ink no-underline">
                      {c.name}
                    </Link>
                  ) : (
                    <Link to="/w/$workspace/campaigns/$campaignId/report" params={params(c.id)} className="font-semibold text-ink no-underline">
                      {c.name}
                    </Link>
                  )}
                  <div className="k">{subtitle(c)}</div>
                </Td>
                <Td>
                  <CampaignStatusBadge status={c.status} blocked={c.blocked} />
                </Td>
                <Td className={c.audience ? undefined : 'k'}>{c.audience || 'not set'}</Td>
                <Td align="right" num={c.recipients != null} className={c.recipients == null ? 'k' : undefined}>
                  {c.recipients != null ? count(c.recipients) : '–'}
                </Td>
                <Td align="right" num={c.delivered != null} className={c.delivered == null ? 'k' : undefined}>
                  {c.delivered != null ? pct(c.delivered) : '–'}
                </Td>
                <Td align="right" num={c.humanOpens != null} className={c.humanOpens == null ? 'k' : undefined}>
                  {c.humanOpens != null ? pct(c.humanOpens) : '–'}
                </Td>
                <Td align="right" num={c.clicks != null} className={c.clicks == null ? 'k' : undefined}>
                  {c.clicks != null ? pct(c.clicks) : '–'}
                </Td>
                <Td>{c.origin ? <span className="k">{c.origin[0]!.toUpperCase() + c.origin.slice(1)} · imported</span> : <RouteBadges routes={c.routes} sent={sent} />}</Td>
                <Td>
                  <div className="flex flex-wrap justify-end gap-1.5">
                    <RowActions c={c} />
                  </div>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-2.5">
        <span className="k" role="status">
          Showing {data.items.length === data.total ? count(data.total) : `${count(from + 1)}–${count(shown)}`} of {count(data.total)}
        </span>
        <div className="flex gap-1.5">
          <PageLink page={page - 1} disabled={page <= 1} status={status} q={q}>
            Previous
          </PageLink>
          <PageLink page={page + 1} disabled={shown >= data.total} status={status} q={q}>
            Next
          </PageLink>
        </div>
      </div>
    </section>
  )
}

function PageLink({ page, disabled, status, q, children }: { page: number; disabled: boolean; status: CampaignFilter; q?: string; children: string }) {
  const { slug } = useWorkspace()
  if (disabled) {
    return (
      <button type="button" className="btn sm" disabled>
        {children}
      </button>
    )
  }
  return (
    <Link
      className="btn sm"
      to="/w/$workspace/campaigns"
      params={{ workspace: slug }}
      search={{ status: status === 'all' ? undefined : status, q: q || undefined, page: page > 1 ? page : undefined }}
    >
      {children}
    </Link>
  )
}

function RowActions({ c }: { c: Schemas['CampaignListItem'] }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug, campaignId: c.id }
  switch (c.status) {
    case 'scheduled':
    case 'draft':
    case 'approval':
    case 'approved': {
      const r = c.readiness
      return (
        <>
          {r && r.verdict !== 'ready' ? (
            <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/review" params={params}>
              {r.verdict === 'blocked' ? `${r.issues} blocking ${r.issues === 1 ? 'issue' : 'issues'}` : `Readiness · ${r.issues}`}
            </ButtonLink>
          ) : null}
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
            Edit
          </ButtonLink>
        </>
      )
    }
    case 'sending':
    case 'paused':
      return (
        <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/live" params={params}>
          Live monitor
        </ButtonLink>
      )
    default:
      return (
        <>
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
            Report
          </ButtonLink>
          <Button size="sm">Duplicate</Button>
        </>
      )
  }
}

function TableSkeleton() {
  return (
    <section className="card" aria-busy="true">
      <span className="vh" role="status">
        Loading campaigns
      </span>
      <div className="flex flex-col">
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} className="flex items-center gap-6 border-b border-line-2 px-3 py-4 last:border-b-0">
            <div className="flex min-w-[200px] flex-col gap-2">
              <Skeleton w={170} h={12} />
              <Skeleton w={120} h={10} />
            </div>
            <Skeleton w={70} h={20} r={999} />
            <Skeleton w={90} h={12} />
            <Skeleton w={60} h={12} className="ml-auto" />
            <Skeleton w={50} h={12} />
            <Skeleton w={50} h={12} />
            <Skeleton w={120} h={20} r={999} />
          </div>
        ))}
      </div>
    </section>
  )
}
