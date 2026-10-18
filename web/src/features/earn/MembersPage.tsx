import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Banner, Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, orList } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { useCan, useWorkspace } from '@/lib/session'
import { fileName, saveBlob } from '../audience/download'
import { ComplimentaryDialog } from './ComplimentaryDialog'
import { EARN, EarnPage, copyText } from './EarnPage'
import { money, nextLine, platformName, priceLine, sourceLine, statusLabel, statusTone, tierLine, type MemberRow } from './format'
import { ImportDialog } from './ImportDialog'
import { MemberPanel } from './MemberPanel'
import { mm } from './members.messages'
import { me } from './messages'

export type MembersSearch = {
  status?: Status
  q?: string
  tier?: string
  period?: 'month' | 'year'
  source?: string
  campaign?: string
  offer?: string
  since?: '30d' | '90d'
  before?: string
  test?: boolean
  page?: number
  member?: string
  import?: string
  comp?: boolean
}

type Status = 'with_access' | 'paying' | 'trialing' | 'past_due' | 'cancels_soon' | 'complimentary' | 'gift' | 'ended'
type List = Schemas['EarnMemberList']

const statuses: Status[] = ['with_access', 'paying', 'trialing', 'past_due', 'cancels_soon', 'complimentary', 'gift', 'ended']

/** Earn › Paid members (EarnMembers): the summary line, filters, status chips, the list and the member panel. */
export function MembersPage({ search, onSearch }: { search: MembersSearch; onSearch: (next: MembersSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = search.status ?? 'with_access'
  const [q, setQ] = useState(search.q ?? '')
  // The box follows the address when it changes elsewhere (Clear search, Back).
  const [shown, setShown] = useState(search.q)
  if (shown !== search.q) {
    setShown(search.q)
    setQ(search.q ?? '')
  }
  useEffect(() => {
    const h = window.setTimeout(() => {
      if ((search.q ?? '') !== q) onSearch({ ...search, q: q || undefined, page: undefined }, true)
    }, 300)
    return () => window.clearTimeout(h)
  }, [q, search, onSearch])
  const query = {
    status,
    q: search.q,
    tier: search.tier,
    period: search.period,
    source: search.source as never,
    campaign: search.campaign,
    offer: search.offer,
    since: search.since,
    before: search.before,
    test: search.test,
    page: search.page ?? 1,
    pageSize: 8,
  }
  const list = $api.useQuery('get', `${EARN}/members`, { params: { path: { workspace: slug }, query } }, { placeholderData: (p) => p })
  const d = list.data
  const set = (patch: Partial<MembersSearch>) => onSearch({ ...search, ...patch, page: patch.page })
  return (
    <EarnPage tab="members" title={t(mm.tableLabel)}>
      {list.error && !d ? (
        <ErrorPanel title={t(mm.errorTitle)} actions={<Button onClick={() => void list.refetch()}>{t(me.retry)}</Button>}>
          {errorMessage(list.error)} {t(mm.errorText)}
        </ErrorPanel>
      ) : !d ? (
        <div className="card pad" aria-busy="true">
          <Skeleton w="100%" h={420} />
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <Header list={d} search={search} onImport={() => onSearch({ ...search, import: 'new' })} onComp={() => onSearch({ ...search, comp: true })} />
          {!d.canManage ? (
            <Note tone="neutral">
              <T
                msg={d.helpers.length ? mm.readOnly : mm.readOnlyNoHelpers}
                values={{ manage: <span className="mono">earn:manage</span>, export: <span className="mono">subscribers:export</span>, helpers: orList(d.helpers) }}
              />
            </Note>
          ) : null}
          {status === 'past_due' && d.counts.pastDue > 0 ? <PastDueBanner list={d} search={search} /> : null}
          {search.test && d.testMembers > 0 ? <TestBanner list={d} /> : null}
          <section className="card" aria-label={t(mm.tableLabel)}>
            <div className="ea-filters">
              <label className="ea-search">
                <Icon name="search" size={15} />
                <input type="search" aria-label={t(mm.searchLabel)} placeholder={t(mm.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
              </label>
              <Select className="ea-f130" aria-label={t(mm.tier)} value={search.tier ?? ''} onChange={(e) => set({ tier: e.target.value || undefined })}>
                <option value="">{t(mm.tierAll)}</option>
                {d.tiers.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
              </Select>
              <Select className="ea-f170" aria-label={t(mm.periodAll)} value={search.period ?? ''} onChange={(e) => set({ period: (e.target.value || undefined) as MembersSearch['period'] })}>
                <option value="">{t(mm.periodAll)}</option>
                <option value="month">{t(me.periodMonthly)}</option>
                <option value="year">{t(me.periodYearly)}</option>
              </Select>
              <Select
                className="ea-f140"
                aria-label={t(mm.source)}
                value={search.campaign ? `campaign:${search.campaign}` : search.source ? `kind:${search.source}` : ''}
                onChange={(e) => {
                  const v = e.target.value
                  if (v.startsWith('campaign:')) set({ campaign: v.slice(9), source: undefined })
                  else if (v.startsWith('kind:')) set({ source: v.slice(5), campaign: undefined })
                  else set({ source: undefined, campaign: undefined })
                }}
              >
                <option value="">{t(mm.sourceAll)}</option>
                {d.campaigns.map((c) => (
                  <option key={c.id} value={`campaign:${c.id}`}>
                    {c.name}
                  </option>
                ))}
                <option value="kind:page">{t(mm.sourceUpgrade)}</option>
                {d.platforms.length ? (
                  d.platforms.map((p) => (
                    <option key={p} value="kind:import">
                      {t(mm.sourceImported, { platform: platformName(p) })}
                    </option>
                  ))
                ) : (
                  <option value="kind:import">{t(mm.sourceImported, { platform: platformName('csv') })}</option>
                )}
                <option value="kind:gift">{t(mm.sourceGift)}</option>
                <option value="kind:grant">{t(mm.sourceGrant)}</option>
              </Select>
              <Select className="ea-f120" aria-label={t(mm.offerAll)} value={search.offer ?? ''} onChange={(e) => set({ offer: e.target.value || undefined })}>
                <option value="">{t(mm.offerAll)}</option>
                {d.offers.map((o) => (
                  <option key={o} value={o}>
                    {o}
                  </option>
                ))}
              </Select>
              <Select
                className="ea-f170"
                aria-label={t(mm.sinceAll)}
                value={search.before ? 'before' : (search.since ?? '')}
                onChange={(e) => {
                  const v = e.target.value
                  if (v === 'before' && d.movedAt) set({ before: d.movedAt.slice(0, 10), since: undefined })
                  else set({ since: (v || undefined) as MembersSearch['since'], before: undefined })
                }}
              >
                <option value="">{t(mm.sinceAll)}</option>
                <option value="30d">{t(mm.since30)}</option>
                <option value="90d">{t(mm.since90)}</option>
                {d.movedAt ? <option value="before">{t(mm.sinceBefore, { date: dayMonthDate(d.movedAt) })}</option> : null}
              </Select>
            </div>
            <div className="ea-chiprow">
              <div className="ea-chips" role="group" aria-label={t(mm.statusLabel)}>
                {statuses.map((s) => (
                  <button key={s} type="button" className="ea-schip" aria-pressed={status === s} onClick={() => set({ status: s === 'with_access' ? undefined : s })}>
                    {chipLabel(t, s)}
                    {s === 'ended' ? null : <span className="c">{count(chipCount(d.counts, s))}</span>}
                  </button>
                ))}
              </div>
              <label className="ea-testtoggle">
                {t(mm.showTest)}
                <input type="checkbox" role="switch" className="tg" checked={Boolean(search.test)} onChange={(e) => set({ test: e.target.checked || undefined })} />
              </label>
            </div>
            <MembersTable list={d} search={search} onOpen={(id) => onSearch({ ...search, member: id })} onClear={() => onSearch({ ...search, q: undefined, page: undefined })} onSearchEnded={() => onSearch({ ...search, status: 'ended' })} />
            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3">
              <span className="k">
                {search.test && d.testShown > 0
                  ? t(mm.showingTest, { test: d.testShown, shown: d.items.length - d.testShown, total: d.counts.withAccess })
                  : t(mm.showing, { shown: d.items.length, total: d.total, status })}
              </span>
              <div className="flex gap-2">
                <Button size="sm" disabled={(search.page ?? 1) <= 1} onClick={() => set({ page: (search.page ?? 1) - 1 })}>
                  {t(mm.previous)}
                </Button>
                <Button size="sm" disabled={(search.page ?? 1) * 8 >= d.total} onClick={() => set({ page: (search.page ?? 1) + 1 })}>
                  {t(mm.next)}
                </Button>
              </div>
            </div>
          </section>
        </div>
      )}
      {search.member ? <MemberPanel id={search.member} onClose={() => onSearch({ ...search, member: undefined })} /> : null}
      {search.comp && d ? <ComplimentaryDialog tiers={d.tiers} onClose={() => onSearch({ ...search, comp: undefined })} /> : null}
      {search.import ? <ImportDialog id={search.import === 'new' ? undefined : search.import} onClose={() => onSearch({ ...search, import: undefined })} onChange={(id) => onSearch({ ...search, import: id }, true)} /> : null}
    </EarnPage>
  )
}

function chipLabel(t: ReturnType<typeof useT>, s: Status): string {
  return {
    with_access: t(mm.withAccess),
    paying: t(mm.paying),
    trialing: t(mm.trialing),
    past_due: t(mm.pastDue),
    cancels_soon: t(mm.cancelsSoon),
    complimentary: t(mm.complimentary),
    gift: t(mm.gift),
    ended: t(mm.ended),
  }[s]
}

function chipCount(c: Schemas['EarnMemberCounts'], s: Status): number {
  return {
    with_access: c.withAccess,
    paying: c.paying,
    trialing: c.trialing,
    past_due: c.pastDue,
    cancels_soon: c.cancelsSoon,
    complimentary: c.complimentary,
    gift: c.gift,
    ended: c.ended,
  }[s]
}

function Header({ list, search, onImport, onComp }: { list: List; search: MembersSearch; onImport: () => void; onComp: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canExport = useCan('subscribers:export')
  const b = (c: React.ReactNode[]) => <b>{c}</b>
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="m-0 text-[15px]">
        <T
          msg={list.counts.pastDue ? mm.summary : mm.summaryNoPastDue}
          values={{ b, paying: count(list.counts.paying), mrr: money(list.mrrCents, list.currency || 'usd', 'never'), pastDue: count(list.counts.pastDue) }}
        />
      </p>
      <div className="flex flex-wrap gap-2">
        {list.canImport ? (
          <Button icon="upload" onClick={onImport}>
            {t(mm.importMembers)}
          </Button>
        ) : null}
        <Button disabledReason={canExport ? undefined : t(mm.readOnlyNoHelpers, { manage: 'earn:manage', export: 'subscribers:export' })} onClick={() => void exportMembers(slug, search, list.total, t)}>
          {t(mm.exportCsv)}
        </Button>
        {list.canManage ? (
          <Button variant="primary" onClick={onComp}>
            {t(mm.giveComp)}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

async function exportMembers(slug: string, search: MembersSearch, n: number, t: ReturnType<typeof useT>) {
  const id = `earn-export-${Date.now()}`
  trackJob(slug, { id, kind: 'export', title: t(mm.exporting), status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response } = await http.GET(`${EARN}/members/export`, {
      params: {
        path: { workspace: slug },
        query: { status: search.status ?? 'with_access', q: search.q, tier: search.tier, period: search.period, source: search.source as never, campaign: search.campaign, offer: search.offer, since: search.since, before: search.before, test: search.test },
      },
      parseAs: 'blob',
    })
    if (data) saveBlob(data as unknown as Blob, fileName(response, 'paid-members.csv'))
    updateJob(slug, id, { status: 'done', detail: t(mm.exportedDetail), finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: t(mm.exported, { count: n }), detail: t(mm.exportedDetail) })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: t(mm.exportFailed), detail: errorMessage(e) })
  }
}

function PastDueBanner({ list, search }: { list: List; search: MembersSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Banner
      tone="warn"
      title={t(mm.pastDueTitle, { count: list.counts.pastDue })}
      actions={
        <>
          <ButtonLink to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'access' }} size="sm">
            {t(mm.accessRules)}
          </ButtonLink>
          <Button size="sm" onClick={() => void exportMembers(slug, search, list.counts.pastDue, t)}>
            {t(mm.exportThese, { count: list.counts.pastDue })}
          </Button>
        </>
      }
    >
      {list.accessRule === 'end' ? t(mm.pastDueEnd) : t(mm.pastDueKeep)}
    </Banner>
  )
}

function TestBanner({ list }: { list: List }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canClear = useCan('payments:manage')
  const [open, setOpen] = useState(false)
  const clear = $api.useMutation('post', `${EARN}/test-data/clear`)
  return (
    <>
      <Banner
        tone="info"
        title={t(mm.testShowing, { count: list.testMembers })}
        actions={
          canClear ? (
            <Button size="sm" variant="danger" onClick={() => setOpen(true)}>
              {t(mm.clearTest)}
            </Button>
          ) : null
        }
      >
        {t(mm.testText)}
      </Banner>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        tone="bad"
        badge={t(mm.clearTest)}
        title={t(mm.clearTestTitle)}
        description={t(mm.clearTestText, { count: list.testMembers })}
        confirmLabel={t(mm.clearTest)}
        onConfirm={async () => {
          await clear.mutateAsync({ params: { path: { workspace: slug } } })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
          toast({ kind: 'deleted', title: t(mm.clearTestDone) })
        }}
      />
    </>
  )
}

function MembersTable({ list, search, onOpen, onClear, onSearchEnded }: { list: List; search: MembersSearch; onOpen: (id: string) => void; onClear: () => void; onSearchEnded: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (list.items.length === 0) {
    if (search.q) {
      return (
        <div className="px-[18px] pb-6">
          <EmptyState
            title={t(mm.noResults, { q: search.q })}
            action={
              <>
                <Button onClick={onClear}>{t(mm.clearSearch)}</Button>
                {search.status === 'ended' ? null : <Button onClick={onSearchEnded}>{t(mm.searchEnded)}</Button>}
              </>
            }
          >
            {t(mm.noResultsText)}
          </EmptyState>
        </div>
      )
    }
    if (list.counts.withAccess === 0 && list.counts.ended === 0 && !search.test) {
      return (
        <div className="px-[18px] pb-6">
          <EmptyState
            title={t(mm.noMembersTitle)}
            action={
              <>
                <Button icon="copy" onClick={() => void copyText(list.upgradeUrl, t, toast)}>
                  {t(mm.copyUpgrade)}
                </Button>
                {list.canImport ? (
                  <ButtonLink to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ import: 'new' }}>
                    {t(mm.importMembers)}
                  </ButtonLink>
                ) : null}
              </>
            }
          >
            {t(mm.noMembersText)} <span className="mono">{list.upgradeHost}</span>
          </EmptyState>
        </div>
      )
    }
    return <p className="k px-[18px] pb-6">{t(mm.noneInFilter)}</p>
  }
  return (
    <>
      <TableScroll label={t(mm.tableLabel)} className="ea-table">
        <table className="tbl" aria-label={t(mm.tableLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(mm.colMember)}</th>
              <th scope="col">{t(mm.colTier)}</th>
              <th scope="col">{t(mm.colStatus)}</th>
              <th scope="col" className="max-lg:hidden">
                {t(mm.colSince)}
              </th>
              <th scope="col">{t(mm.colNext)}</th>
              <th scope="col" className="r">
                {t(mm.colLtv)}
              </th>
              <th scope="col" className="max-lg:hidden">
                {t(mm.colSource)}
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((r) => (
              <tr key={r.id} className={r.test ? 'ea-rowbtn ea-test' : 'ea-rowbtn'} onClick={() => onOpen(r.id)}>
                <td className="ea-member">
                  <button type="button" className="m-0 flex flex-col items-start border-0 bg-transparent p-0 text-left" onClick={() => onOpen(r.id)}>
                    <span className="nm">{r.hidden ? t(mm.hiddenName) : r.name || r.email}</span>
                    {r.hidden ? null : r.name ? <span className="em">{r.email}</span> : null}
                    {r.test ? <span className="em">{t(mm.testMode)}</span> : null}
                  </button>
                  {r.test ? <Badge tone="purple" className="mt-1">{t(mm.test)}</Badge> : null}
                </td>
                <td>
                  <div className="font-semibold">{tierLine(t, r)}</div>
                  <div className="k">{priceLine(t, r)}</div>
                </td>
                <td>
                  <StatusBadges r={r} />
                </td>
                <td className="whitespace-nowrap max-lg:hidden">{dayMonthYear(r.since)}</td>
                <td>{nextLine(t, r)}</td>
                <td className="r ea-num">{money(r.ltvCents, r.currency || 'usd', 'never')}</td>
                <td className="max-lg:hidden">
                  <SourceCell r={r} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <div className="ea-cards">
        {list.items.map((r) => (
          <button key={r.id} type="button" className="ea-mcard" onClick={() => onOpen(r.id)}>
            <span className="flex flex-wrap items-center gap-2">
              <span className="font-semibold">{r.hidden ? t(mm.hiddenName) : r.name || r.email}</span>
              <StatusBadges r={r} />
            </span>
            <span className="text-[13px]">
              {tierLine(t, r)} · {priceLine(t, r)}
            </span>
            <span className="k">
              {nextLine(t, r)} · {t(mm.memberSince).toLowerCase()} {dayMonthYear(r.since)} · {t(mm.ltv).toLowerCase()} {money(r.ltvCents, r.currency || 'usd', 'never')}
            </span>
          </button>
        ))}
      </div>
    </>
  )
}

export function StatusBadges({ r }: { r: MemberRow }) {
  const t = useT()
  return (
    <span className="inline-flex flex-col items-start gap-1">
      {r.status === 'gift' ? (
        <span className="badge pur">
          <GiftIcon />
          {statusLabel(t, r.status)}
        </span>
      ) : (
        <Badge tone={statusTone(r.status)}>{statusLabel(t, r.status)}</Badge>
      )}
      {r.cancelAt && r.status !== 'canceled' ? <span className="badge sm mut">{t(me.cancelsOn, { date: dayMonthDate(r.cancelAt) })}</span> : null}
      {r.accessPaused ? <span className="badge sm warn">{t(me.accessPaused)}</span> : null}
    </span>
  )
}

/** The gift badge's icon (EarnMembers). */
function GiftIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2.5" y="6.5" width="11" height="7.5" rx="1" />
      <path d="M1.8 4.2h12.4v2.3H1.8zM8 4.2V14M8 4.2c-1.2-2.3-4-2.6-4-.9 0 .9 1.6.9 4 .9zM8 4.2c1.2-2.3 4-2.6 4-.9 0 .9-1.6.9-4 .9z" />
    </svg>
  )
}

function SourceCell({ r }: { r: MemberRow }) {
  const t = useT()
  const { slug } = useWorkspace()
  const label = sourceLine(t, r)
  const stop = (e: { stopPropagation: () => void }) => e.stopPropagation()
  if (r.source.kind === 'campaign' && r.source.campaignId) {
    return (
      <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: r.source.campaignId }} className="ea-src" onClick={stop}>
        {label}
      </Link>
    )
  }
  if (r.source.kind === 'import' && r.source.importId) {
    return (
      <Link to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ import: r.source.importId }} className="ea-src" onClick={stop}>
        {label}
      </Link>
    )
  }
  if (r.kind === 'gift') {
    return (
      <Link to="/w/$workspace/earn/gifts" params={{ workspace: slug }} className="ea-src" onClick={stop}>
        {label}
      </Link>
    )
  }
  return <span>{label}</span>
}
