import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type ReactNode } from 'react'
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
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { BulkBar, CalendarLink, deletedLine, RestoreButton, RowActionDialogs, RowMenu, TagPills, UnscheduleButton, type RowAction } from './manage'
import { mm } from './manage.messages'
import { RouteBadges } from './RouteBadges'
import { CampaignStatusBadge } from './StatusBadge'
import { NewCampaignDialog } from '@/features/digests/TypePicker'
import { DigestBadgeLink, RecurringTable, useDigestCount } from '@/features/digests/RecurringTable'
import { mc } from '@/features/earn/composer/composer.messages'
import { PaidOnlyBadge } from '@/features/earn/composer/EarnBlocks'
import { IssueStatusBadge } from '@/features/digests/shared'
import { SponsorChip, useSponsorChips } from '@/features/earn/sponsors/chips'
import { m as dm } from '@/features/digests/messages'
import { useCampaignWebStates, WebCell } from '@/features/website/elsewhere'
import { me } from '@/features/website/elsewhere.messages'
import { ChannelFilter } from '@/features/channels/ChannelFilter'
import type { Channel } from '@/features/channels/channel'
import { m as chm } from '@/features/channels/messages'
import { SmsCampaignHead, SmsCampaignRow } from '@/features/sms/CampaignRow'
import { ChannelCampaignRow } from '@/features/push/CampaignRow'
import { Check, SelectAll } from '@/components/ui/SelectAll'
import { useSelection } from '@/lib/selection'

export type CampaignFilter = 'all' | 'drafts' | 'scheduled' | 'sending' | 'sent' | 'recurring' | 'paid' | 'archived' | 'deleted'

export type CampaignWhen = '7d' | '30d' | 'quarter' | 'custom'

export interface CampaignsSearch {
  status?: CampaignFilter
  q?: string
  page?: number
  /** Phase 8: only one channel's campaigns. */
  channel?: Channel
  /** Review 2026-11 CA-2: when it went or goes out (from and to with custom), its sender, a list, a tag. */
  when?: CampaignWhen
  from?: string
  to?: string
  sender?: string
  list?: string
  tag?: string
}

const PAGE_SIZE = 25

/** YYYY-MM-DD of a local day. */
function isoDay(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** The days a "when" filter covers: from (inclusive) and to (the day after the last). */
export function whenRange(s: CampaignsSearch, now = new Date()): { from?: string; to?: string } {
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
  switch (s.when) {
    case '7d':
      return { from: isoDay(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 6)), to: isoDay(tomorrow) }
    case '30d':
      return { from: isoDay(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 29)), to: isoDay(tomorrow) }
    case 'quarter':
      return { from: isoDay(new Date(now.getFullYear(), Math.floor(now.getMonth() / 3) * 3, 1)), to: isoDay(new Date(now.getFullYear(), Math.floor(now.getMonth() / 3) * 3 + 3, 1)) }
    case 'custom': {
      // "To" is the last day shown; the API takes the day after it.
      const to = s.to ? new Date(`${s.to}T00:00:00`) : undefined
      return { from: s.from, to: to ? isoDay(new Date(to.getFullYear(), to.getMonth(), to.getDate() + 1)) : undefined }
    }
  }
  return {}
}

/** The list's search with a change, back on the first page; "all" is the default status. */
function nextSearch(s: CampaignsSearch, patch: Partial<CampaignsSearch>): CampaignsSearch {
  const n: CampaignsSearch = { ...s, page: undefined, ...patch }
  if (n.status === 'all') n.status = undefined
  if (!n.q) n.q = undefined
  if (n.when !== 'custom') {
    n.from = undefined
    n.to = undefined
  }
  return n
}

function filtered(s: CampaignsSearch) {
  return Boolean(s.when || s.sender || s.list || s.tag)
}

/** 7.7 Campaigns list: KPIs for the last 30 days, filters and the table. */
export function CampaignsPage({ search }: { search: CampaignsSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = search.status ?? 'all'
  const page = search.page ?? 1
  const newReason = usePermissionReason('campaigns:edit')
  const navigate = useNavigate()
  const range = whenRange(search)
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns',
    {
      params: {
        path: { workspace: slug },
        // Paid only (Earn): every status, campaigns for paid members only.
        query: {
          status: status === 'recurring' || status === 'paid' ? 'all' : status,
          paidOnly: status === 'paid' || undefined,
          q: search.q ?? '',
          channel: search.channel,
          from: range.from,
          to: range.to,
          sender: search.sender,
          list: search.list,
          tag: search.tag,
          page,
          pageSize: PAGE_SIZE,
        },
      },
    },
    { placeholderData: keepPreviousData },
  )
  const counts = list.data?.counts
  // "New campaign" asks what kind first: Regular or Recurring digest (DigestSetup).
  const [picking, setPicking] = useState(false)
  const go = (patch: Partial<CampaignsSearch>) => void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: nextSearch(search, patch) })
  // Phase 8: the channel chips head the table once a second channel is set up;
  // the filters of CA-2 follow them.
  const top = (
    <>
      <ChannelFilter value={search.channel} onChange={(channel) => go({ channel })} />
      {status === 'deleted' ? null : <FiltersBar search={search} onChange={go} />}
    </>
  )
  return (
    <Page
      title={t(m.title)}
      filters={<CampaignFilters status={status} counts={counts} search={search} />}
      actions={
        <>
          <CalendarLink />
          <SearchBox initial={search.q ?? ''} search={search} />
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
        search.q || status !== 'all' || search.channel || filtered(search) ? (
          <>
            {search.channel || filtered(search) ? <section className="card">{top}</section> : null}
            <EmptyState
              title={search.q ? t(m.noMatch, { query: search.q }) : search.channel && status === 'all' ? t(chm.noChannelCampaigns, { channel: search.channel }) : t(m.noStatus, { status })}
              action={
                <Link className="btn" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                  {t(m.showAll)}
                </Link>
              }
            >
              {t(m.noMatchBody, { count: list.data.counts.all })}
            </EmptyState>
          </>
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
        <CampaignTable data={list.data} page={page} status={status} search={search} top={top} />
      )}
    </Page>
  )
}

function CampaignFilters({ status, counts, search }: { status: CampaignFilter; counts?: Schemas['CampaignStatusCounts']; search: CampaignsSearch }) {
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
      onChange={(next) => void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: nextSearch(search, { status: next }) })}
      items={[
        { value: 'all', label: label('all', counts?.all) },
        { value: 'drafts', label: label('drafts', counts?.drafts) },
        { value: 'scheduled', label: label('scheduled', counts?.scheduled) },
        { value: 'sending', label: label('sending', counts?.sending) },
        { value: 'sent', label: label('sent', counts?.sent) },
        { value: 'recurring', label: digests !== undefined ? t(dm.recurringTabCount, { count: digests }) : t(dm.recurringTab) },
        ...(counts?.paidOnly || status === 'paid' ? [{ value: 'paid' as const, label: t(mc.paidOnlyFilter, { count: count(counts?.paidOnly ?? 0) }) }] : []),
        // Review 2026-11 CA-1: archived sent campaigns, and deleted ones while they can come back.
        ...(counts?.archived || status === 'archived' ? [{ value: 'archived' as const, label: label('archived', counts?.archived) }] : []),
        ...(counts?.deleted || status === 'deleted' ? [{ value: 'deleted' as const, label: label('deleted', counts?.deleted) }] : []),
      ]}
    />
  )
}

function SearchBox({ initial, search }: { initial: string; search: CampaignsSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [value, setValue] = useState(initial)
  useEffect(() => {
    if (value === initial) return
    const h = window.setTimeout(() => {
      void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: nextSearch(search, { q: value.trim() || undefined }), replace: true })
    }, 250)
    return () => window.clearTimeout(h)
  }, [value, initial, navigate, slug, search])
  return (
    <>
      <label className="vh" htmlFor="campaign-search">
        {t(m.search)}
      </label>
      <input
        id="campaign-search"
        type="search"
        className="in w-[220px] max-md:w-full"
        placeholder={t(mm.searchPlaceholder)}
        value={value}
        onChange={(e) => setValue(e.target.value)}
      />
    </>
  )
}

/** When, sender, audience list and tag (CampaignsComplete board, strip 2). */
function FiltersBar({ search, onChange }: { search: CampaignsSearch; onChange: (patch: Partial<CampaignsSearch>) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const senders = $api.useQuery('get', '/api/v1/workspaces/{workspace}/senders', { params: { path: { workspace: slug } } })
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } })
  const tags = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/tags', { params: { path: { workspace: slug } } })
  const tagList = tags.data?.tags ?? []
  return (
    <div className="flex flex-wrap items-end gap-2 border-b border-line-2 px-3 py-2.5" role="group" aria-label={t(mm.filtersLabel)}>
      <label className="vh" htmlFor={`${id}-when`}>
        {t(mm.whenLabel)}
      </label>
      <select id={`${id}-when`} className="in max-md:flex-1" value={search.when ?? ''} onChange={(e) => onChange({ when: (e.target.value || undefined) as CampaignWhen | undefined })}>
        <option value="">{t(mm.whenAny)}</option>
        <option value="7d">{t(mm.when7)}</option>
        <option value="30d">{t(mm.when30)}</option>
        <option value="quarter">{t(mm.whenQuarter)}</option>
        <option value="custom">{t(mm.whenCustom)}</option>
      </select>
      {search.when === 'custom' ? (
        <>
          <span className="field">
            <label className="lbl" htmlFor={`${id}-from`}>
              {t(mm.from)}
            </label>
            <input id={`${id}-from`} type="date" className="in" value={search.from ?? ''} max={search.to} onChange={(e) => onChange({ when: 'custom', from: e.target.value || undefined, to: search.to })} />
          </span>
          <span className="field">
            <label className="lbl" htmlFor={`${id}-to`}>
              {t(mm.to)}
            </label>
            <input id={`${id}-to`} type="date" className="in" value={search.to ?? ''} min={search.from} onChange={(e) => onChange({ when: 'custom', from: search.from, to: e.target.value || undefined })} />
          </span>
        </>
      ) : null}
      <label className="vh" htmlFor={`${id}-sender`}>
        {t(mm.senderLabel)}
      </label>
      <select id={`${id}-sender`} className="in max-w-[220px] max-md:max-w-none max-md:flex-1" value={search.sender ?? ''} onChange={(e) => onChange({ sender: e.target.value || undefined })}>
        <option value="">{t(mm.senderAny)}</option>
        {(senders.data?.senders ?? []).map((x) => (
          <option key={x.id} value={x.id}>
            {x.name ? `${x.name} · ${x.email}` : x.email}
          </option>
        ))}
      </select>
      <label className="vh" htmlFor={`${id}-list`}>
        {t(mm.listLabel)}
      </label>
      <select id={`${id}-list`} className="in max-w-[200px] max-md:max-w-none max-md:flex-1" value={search.list ?? ''} onChange={(e) => onChange({ list: e.target.value || undefined })}>
        <option value="">{t(mm.listAny)}</option>
        {(lists.data?.items ?? []).map((l) => (
          <option key={l.id} value={l.id}>
            {l.name}
          </option>
        ))}
      </select>
      {tagList.length || search.tag ? (
        <>
          <label className="vh" htmlFor={`${id}-tag`}>
            {t(mm.tagLabel)}
          </label>
          <select id={`${id}-tag`} className="in max-md:flex-1" value={search.tag ?? ''} onChange={(e) => onChange({ tag: e.target.value || undefined })}>
            <option value="">{t(mm.tagAny)}</option>
            {[...new Set([...tagList, ...(search.tag ? [search.tag] : [])])].map((tag) => (
              <option key={tag} value={tag}>
                {tag}
              </option>
            ))}
          </select>
        </>
      ) : null}
      {filtered(search) ? (
        <Button size="sm" variant="ghost" className="mb-[3px] self-end" onClick={() => onChange({ when: undefined, sender: undefined, list: undefined, tag: undefined })}>
          {t(mm.clearFilters)}
        </Button>
      ) : null}
    </div>
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

function CampaignTable({ data, page, status, search, top }: { data: Schemas['CampaignList']; page: number; status: CampaignFilter; search: CampaignsSearch; top?: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const channel = search.channel
  const from = (page - 1) * data.pageSize
  const shown = from + data.items.length
  const params = (id: string) => ({ workspace: slug, campaignId: id })
  const sponsors = useSponsorChips()
  // The website's Web column (features/website), while it's on.
  const web = useCampaignWebStates(data.items.map((c) => c.id))
  // Review 2026-11 CA-2: rows are selected for bulk actions, page by page;
  // deleted campaigns are only restored, one at a time.
  const canEdit = useCan('campaigns:edit')
  const selectable = canEdit && status !== 'deleted'
  const [action, setAction] = useState<RowAction>()
  // Another page or filter: what was picked is out of view and drops out.
  const sel = useSelection(data.items.map((c) => c.id))
  const selected = data.items.filter((c) => sel.has(c.id))
  const toggle = sel.toggle
  const pick = (c: Schemas['CampaignListItem']) =>
    selectable ? (
      <Check checked={sel.has(c.id)} aria-label={t(mm.selectOne, { name: c.name })} onChange={(e) => toggle(c.id, e.target.checked)} />
    ) : undefined
  // Everything in the ⋯ menu but Report needs campaigns:edit: viewers get none.
  const menu = (c: Schemas['CampaignListItem']) => (status === 'deleted' ? <RestoreButton c={c} /> : canEdit ? <RowMenu c={c} onAction={setAction} /> : null)
  return (
    <section className="card" aria-label={t(m.title)}>
      {top}
      {selected.length ? (
        <div className="border-b border-line-2 p-2.5">
          <BulkBar selected={selected} archivedTab={status === 'archived'} onClear={sel.clear} />
        </div>
      ) : null}
      {/* The board's key and provider cells wrap; a long audience ("Weekly Digest + Product
          Updates") wraps too, so the actions stay in view at 1440 as on the board. */}
      <Table
        phoneWrap
        className={
          web
            ? // With the Web column the cells sit a little closer and two headers wrap, so the actions stay in view at 1440.
              '[&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[92px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(4)]:min-w-[86px] [&_td:nth-child(4)]:whitespace-normal [&_td:nth-child(9)]:min-w-[104px] [&_td:nth-child(9)]:whitespace-normal [&_td:nth-child(9)_.badge]:my-px [&_td]:whitespace-nowrap [&_td]:!px-[7px] [&_th]:!px-[7px] [&_th:nth-child(7)]:!whitespace-normal [&_th:nth-child(9)]:!whitespace-normal'
            : '[&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal [&_td:nth-child(3)]:min-w-[86px] [&_td:nth-child(3)]:whitespace-normal [&_td:nth-child(8)]:min-w-[120px] [&_td:nth-child(8)]:whitespace-normal [&_td:nth-child(8)_.badge]:my-px [&_td]:whitespace-nowrap'
        }
      >
        <thead>
          {/* Phase 8: with SMS chosen, texts' own columns (opt-outs and cost instead of opens). */}
          {channel === 'sms' ? (
            <SmsCampaignHead lead={selectable ? <SelectAll all={sel.all} some={sel.some} label={t(mm.selectAll)} onChange={(on) => sel.setMany(data.items.map((c) => c.id), on)} /> : undefined} />
          ) : (
            <tr>
              <Th>
                <span className="inline-flex items-center gap-2">
                  {selectable ? <SelectAll all={sel.all} some={sel.some} label={t(mm.selectAll)} onChange={(on) => sel.setMany(data.items.map((c) => c.id), on)} /> : null}
                  {t(m.campaign)}
                </span>
              </Th>
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
          )}
        </thead>
        <tbody>
          {data.items.map((c) => {
            if (c.channel === 'sms')
              return (
                <SmsCampaignRow
                  key={c.id}
                  c={c}
                  textColumns={channel === 'sms'}
                  web={web && channel !== 'sms' ? <span className="k">–</span> : undefined}
                  lead={pick(c)}
                  more={menu(c)}
                  selected={sel.has(c.id)}
                />
              )
            // Push and WhatsApp (X22): their own composer, review and report.
            if ((c.channel === 'push' || c.channel === 'whatsapp') && channel !== 'sms')
              return <ChannelCampaignRow key={c.id} c={c} web={web ? <span className="k">–</span> : undefined} lead={pick(c)} more={menu(c)} selected={sel.has(c.id)} />
            const sent = ['sent', 'sending', 'paused', 'cancelled', 'failed'].includes(c.status)
            const editable = ['draft', 'approval', 'approved', 'scheduled'].includes(c.status) && status !== 'deleted'
            const line = status === 'deleted' ? deletedLine(c) : subtitle(c)
            return (
              <tr key={c.id} className={sel.has(c.id) ? 'sel' : undefined}>
                <Td>
                  <div className="flex items-start gap-2">
                    {pick(c)}
                    <div className="min-w-0 flex-1">
                      {/* A name too long for the column ends in an ellipsis, whole on hover and focus. */}
                      <Truncate text={c.name}>
                        {status === 'deleted' ? (
                          <span className="font-semibold text-ink">{c.name}</span>
                        ) : editable ? (
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
                      <TagPills tags={c.tags} />
                      {c.digest || sponsors.has(c.id) ? (
                        <div className="my-1 flex flex-wrap gap-1.5">
                          {c.digest ? <DigestBadgeLink digest={c.digest} /> : null}
                          {sponsors.has(c.id) ? <SponsorChip sponsor={sponsors.get(c.id) ?? ''} /> : null}
                        </div>
                      ) : null}
                      {c.paidOnly ? <PaidOnlyBadge className="my-1" /> : null}
                      <div className="k">{line}</div>
                    </div>
                  </div>
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
                    {status === 'deleted' ? null : <RowActions c={c} />}
                    {menu(c)}
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
          <PageLink page={page - 1} disabled={page <= 1} search={search}>
            {t(m.previous)}
          </PageLink>
          <PageLink page={page + 1} disabled={shown >= data.total} search={search}>
            {t(m.next)}
          </PageLink>
        </div>
      </div>
      <RowActionDialogs action={action} onClose={() => setAction(undefined)} />
    </section>
  )
}

function PageLink({ page, disabled, search, children }: { page: number; disabled: boolean; search: CampaignsSearch; children: string }) {
  const { slug } = useWorkspace()
  if (disabled) {
    return (
      <button type="button" className="btn sm" disabled>
        {children}
      </button>
    )
  }
  return (
    <Link className="btn sm" to="/w/$workspace/campaigns" params={{ workspace: slug }} search={{ ...nextSearch(search, {}), page: page > 1 ? page : undefined }}>
      {children}
    </Link>
  )
}

function RowActions({ c }: { c: Schemas['CampaignListItem'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const editReason = usePermissionReason('campaigns:edit')
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
          {/* Review 2026-11 CA-2: a scheduled campaign is unscheduled where it is. */}
          {c.status === 'scheduled' ? <UnscheduleButton c={c} /> : null}
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
          {/* Review 2026-11 CA-1: people who can edit duplicate from the row's ⋯ menu;
              others see why they can't. */}
          {editReason ? (
            <Button size="sm" disabledReason={editReason}>
              {t(m.duplicate)}
            </Button>
          ) : null}
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
