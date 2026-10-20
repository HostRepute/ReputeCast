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
import { Truncate } from '@/components/ui/Truncate'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, count, money, pct, rate, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { RouteBadges } from './RouteBadges'
import { CampaignStatusBadge } from './StatusBadge'
import { NewCampaignDialog } from '@/features/digests/TypePicker'
import { DigestBadgeLink, RecurringTable, useDigestCount } from '@/features/digests/RecurringTable'
import { mc } from '@/features/earn/composer/composer.messages'
import { PaidOnlyBadge } from '@/features/earn/composer/EarnBlocks'
import { IssueStatusBadge } from '@/features/digests/shared'
import { m as dm } from '@/features/digests/messages'
import { useCampaignWebStates, WebCell } from '@/features/website/elsewhere'
import { me } from '@/features/website/elsewhere.messages'

export type CampaignFilter = 'all' | 'drafts' | 'scheduled' | 'sending' | 'sent' | 'recurring' | 'paid'

export interface CampaignsSearch {
  status?: CampaignFilter
  q?: string
  page?: number
}

const PAGE_SIZE = 25

/** 7.7 Campaigns list: KPIs for the last 30 days, filters and the table. */
export function CampaignsPage({ search }: { search: CampaignsSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = search.status ?? 'all'
  const page = search.page ?? 1
  const newReason = usePermissionReason('campaigns:edit')
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns',
    {
      params: {
        path: { workspace: slug },
        // Paid only (Earn): every status, campaigns for paid members only.
        query: { status: status === 'recurring' || status === 'paid' ? 'all' : status, paidOnly: status === 'paid' || undefined, q: search.q ?? '', page, pageSize: PAGE_SIZE },
      },
    },
    { placeholderData: keepPreviousData },
  )
  const counts = list.data?.counts
  // "New campaign" asks what kind first: Regular or Recurring digest (DigestSetup).
  const [picking, setPicking] = useState(false)
  return (
    <Page
      title={t(m.title)}
      filters={<CampaignFilters status={status} counts={counts} q={search.q} />}
      actions={
        <>
          <SearchBox initial={search.q ?? ''} status={status} />
          {newReason ? (
            <Button variant="primary" disabledReason={newReason}>
              {t(m.newCampaign)}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => setPicking(true)}>
              {t(m.newCampaign)}
            </Button>
          )}
          {picking ? <NewCampaignDialog open onOpenChange={setPicking} /> : null}
        </>
      }
    >
      <SummaryCards />
      {status === 'recurring' ? (
        <RecurringTable onNew={() => setPicking(true)} newReason={newReason} />
      ) : list.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(list.error)} {t(m.nothingChanged)}
        </ErrorPanel>
      ) : list.isPending ? (
        <TableSkeleton />
      ) : list.data.total === 0 ? (
        search.q || status !== 'all' ? (
          <EmptyState
            title={search.q ? t(m.noMatch, { query: search.q }) : t(m.noStatus, { status })}
            action={
              <Link className="btn" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                {t(m.showAll)}
              </Link>
            }
          >
            {t(m.noMatchBody, { count: list.data.counts.all })}
          </EmptyState>
        ) : (
          <EmptyState
            title={t(m.emptyTitle)}
            action={
              newReason ? null : (
                <Button variant="primary" onClick={() => setPicking(true)}>
                  {t(m.newCampaign)}
                </Button>
              )
            }
          >
            {t(m.emptyBody)}
          </EmptyState>
        )
      ) : (
        <CampaignTable data={list.data} page={page} status={status} q={search.q} />
      )}
    </Page>
  )
}

function CampaignFilters({ status, counts, q }: { status: CampaignFilter; counts?: Schemas['CampaignStatusCounts']; q?: string }) {
  const t = useT()
  const digests = useDigestCount()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const label = (filter: CampaignFilter, v?: number) => (v !== undefined ? t(m.filterCount, { filter, count: count(v) }) : t(m.filter, { filter }))
  return (
    <FilterTabs
      label={t(m.filterLabel)}
      className="ml-3 max-md:ml-0"
      value={status}
      onChange={(next) =>
        void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: { status: next === 'all' ? undefined : next, q: q || undefined } })
      }
      items={[
        { value: 'all', label: label('all', counts?.all) },
        { value: 'drafts', label: label('drafts', counts?.drafts) },
        { value: 'scheduled', label: label('scheduled', counts?.scheduled) },
        { value: 'sending', label: label('sending', counts?.sending) },
        { value: 'sent', label: label('sent', counts?.sent) },
        { value: 'recurring', label: digests !== undefined ? t(dm.recurringTabCount, { count: digests }) : t(dm.recurringTab) },
        ...(counts?.paidOnly || status === 'paid' ? [{ value: 'paid' as const, label: t(mc.paidOnlyFilter, { count: count(counts?.paidOnly ?? 0) }) }] : []),
      ]}
    />
  )
}

function SearchBox({ initial, status }: { initial: string; status: CampaignFilter }) {
  const t = useT()
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
        {t(m.search)}
      </label>
      <input id="campaign-search" type="search" className="in w-[220px] max-md:w-full" placeholder={t(m.search)} value={value} onChange={(e) => setValue(e.target.value)} />
    </>
  )
}

function SummaryCards() {
  const t = useT()
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
    { key: 'sent', label: t(m.sentLast, { days: s.days }), value: count(s.sent), sub: t(m.campaignCount, { count: s.campaigns }) },
    { key: 'delivered', label: t(m.delivered), value: pct(s.delivered), sub: t(m.deliveredSub, { bounces: pct(s.bounces), complaints: rate(s.complaints) }) },
    { key: 'opens', label: t(m.humanOpens), value: pct(s.humanOpens), sub: t(m.humanOpensSub) },
    { key: 'clicks', label: t(m.clicks), value: pct(s.clicks), sub: t(m.clicksSub, { rate: pct(s.clickToOpen) }) },
    {
      key: 'cost',
      label: t(m.costLast, { days: s.days }),
      value: money(s.cost),
      sub: s.costBy.map((c) => `${c.short} ${money(c.cost, c.cost % 1 === 0 ? 0 : 2)}`).join(' · ') || t(m.noSends),
    },
  ]
  return (
    <Grid min={200} gap={16}>
      {cards.map((c) => (
        <div key={c.key} className="card flex flex-col gap-0.5 px-4.5 py-3.5">
          <span className="k line-clamp-2" title={c.label}>
            {c.label}
          </span>
          <span className="n text-[22px]">{c.value}</span>
          <span className="k">{c.sub}</span>
        </div>
      ))}
    </Grid>
  )
}

/** "brevo" → "Brevo": the platform a campaign was imported from. */
function originName(origin: string) {
  return origin[0]!.toUpperCase() + origin.slice(1)
}

function subtitle(c: Schemas['CampaignListItem']) {
  const parts: string[] = []
  if (c.status === 'scheduled' && c.scheduledAt) {
    parts.push(`${shortDate(c.scheduledAt)} · ${clock(c.scheduledAt)}${c.timezone ? ` ${c.timezone}` : ''}`)
  } else if (c.startedAt) {
    parts.push(`${shortDate(c.startedAt)} · ${clock(c.startedAt)}`)
  } else if (c.updatedAt !== c.createdAt && c.updatedBy) {
    parts.push(tt(m.edited, { time: clock(c.updatedAt), name: c.updatedBy }))
  } else {
    parts.push(tt(m.created, { time: clock(c.createdAt) }))
  }
  if (c.origin) parts.push(tt(m.importedFrom, { origin: originName(c.origin) }))
  // A digest issue: when it was prepared, or how many posts it went out with.
  if (c.digest && (c.status === 'sent' || c.status === 'sending')) parts.push(tt(dm.issuePosts, { count: c.digest.posts }))
  else if (c.digest?.preparedAt) parts.push(tt(dm.issuePrepared, { time: clock(c.digest.preparedAt) }))
  parts.push(...c.notes)
  return parts.join(' · ')
}

function CampaignTable({ data, page, status, q }: { data: Schemas['CampaignList']; page: number; status: CampaignFilter; q?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const from = (page - 1) * data.pageSize
  const shown = from + data.items.length
  const params = (id: string) => ({ workspace: slug, campaignId: id })
  // The website's Web column (features/website), while it's on.
  const web = useCampaignWebStates(data.items.map((c) => c.id))
  return (
    <section className="card" aria-label={t(m.title)}>
      {/* The board's key and provider cells wrap; a long audience ("Weekly Digest + Product
          Updates") wraps too, so the actions stay in view at 1440 as on the board. */}
      <Table
        phoneWrap
        className={
          web
            ? // With the Web column the cells sit a little closer and two headers wrap, so the actions stay in view at 1440.
              '[&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[92px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(4)]:min-w-[86px] [&_td:nth-child(4)]:whitespace-normal [&_td:nth-child(9)]:min-w-[104px] [&_td:nth-child(9)]:whitespace-normal [&_td:nth-child(9)_.badge]:my-px [&_td]:whitespace-nowrap [&_td]:!px-[9px] [&_th]:!px-[9px] [&_th:nth-child(7)]:!whitespace-normal [&_th:nth-child(9)]:!whitespace-normal'
            : '[&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[86px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(8)]:min-w-[120px] [&_td:nth-child(8)]:whitespace-normal [&_td:nth-child(8)_.badge]:my-px [&_td]:whitespace-nowrap'
        }
      >
        <thead>
          <tr>
            <Th>{t(m.campaign)}</Th>
            <Th>{t(m.status)}</Th>
            {web ? <Th>{t(me.colWeb)}</Th> : null}
            <Th>{t(m.audience)}</Th>
            <Th align="right">{t(m.recipients)}</Th>
            <Th align="right">{t(m.delivered)}</Th>
            <Th align="right">{t(m.humanOpens)}</Th>
            <Th align="right">{t(m.clicks)}</Th>
            <Th>{t(m.sendingProviders)}</Th>
            <Th>
              <span className="vh">{t(m.actions)}</span>
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
                  {/* A name too long for the column ends in an ellipsis, whole on hover and focus. */}
                  <Truncate text={c.name}>
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
                  </Truncate>
                  {c.digest ? (
                    <div className="my-1">
                      <DigestBadgeLink digest={c.digest} />
                    </div>
                  ) : null}
                  {c.paidOnly ? <PaidOnlyBadge className="my-1" /> : null}
                  <div className="k">{subtitle(c)}</div>
                </Td>
                <Td>
                  {c.digest && (c.digest.issueStatus === 'waiting' || c.digest.issueStatus === 'held') ? (
                    <IssueStatusBadge status={c.digest.issueStatus} />
                  ) : (
                    <CampaignStatusBadge status={c.status} blocked={c.blocked} />
                  )}
                </Td>
                {web ? (
                  <Td>
                    <WebCell state={web.get(c.id)} status={c.status} />
                  </Td>
                ) : null}
                <Td className={c.audience ? undefined : 'k'}>{c.audience || t(m.notSet)}</Td>
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
                <Td>{c.origin ? <span className="k">{t(m.imported, { origin: originName(c.origin) })}</span> : <RouteBadges routes={c.routes} sent={sent} />}</Td>
                <Td>
                  <div className="row-actions flex flex-wrap justify-end gap-1.5">
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
          {t(m.showing, { range: data.items.length === data.total ? count(data.total) : `${count(from + 1)}–${count(shown)}`, total: count(data.total) })}
        </span>
        <div className="flex gap-1.5">
          <PageLink page={page - 1} disabled={page <= 1} status={status} q={q}>
            {t(m.previous)}
          </PageLink>
          <PageLink page={page + 1} disabled={shown >= data.total} status={status} q={q}>
            {t(m.next)}
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
  const t = useT()
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
              {r.verdict === 'blocked' ? t(m.blockingIssues, { count: r.issues }) : t(m.readiness, { count: r.issues })}
            </ButtonLink>
          ) : null}
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
            {t(m.edit)}
          </ButtonLink>
        </>
      )
    }
    case 'sending':
    case 'paused':
      return (
        <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/live" params={params}>
          {t(m.liveMonitor)}
        </ButtonLink>
      )
    default:
      return (
        <>
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
            {t(m.report)}
          </ButtonLink>
          <Button size="sm">{t(m.duplicate)}</Button>
        </>
      )
  }
}

function TableSkeleton() {
  const t = useT()
  return (
    <section className="card" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
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
