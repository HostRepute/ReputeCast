import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, t, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, clock, count, suppressedWhen } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { AudienceTabs } from './AudienceTabs'
import { shortProvider } from './ContactsPage'
import { fileName, saveBlob } from './download'
import { providerColor } from './labels'
import { AddSuppressionsDialog, ImportSuppressionsDialog, RemoveSuppressionDialog } from './SuppressionDialogs'
import './audience.css'
import { SmsSuppressions, SuppressionChannels, type SmsSuppressionSearch } from '@/features/sms/Suppressions'
import { m as am } from './messages'
import { m } from './suppressions.messages'

export interface SuppressionsSearch {
  q?: string
  reason?: SuppressionReason
  source?: string
  since?: '7d' | '30d'
  /** All mail (every stream, transactional too) or Marketing only (Broadcasts and Automations). */
  appliesTo?: 'all' | 'marketing'
  import?: string
  page?: number
  /** Phase 8: the SMS suppression list instead of email's. */
  channel?: 'sms'
  smsReason?: SmsSuppressionSearch['smsReason']
  smsProvider?: string
  smsPage?: number
}

export type SuppressionReason = 'hard_bounce' | 'complaint' | 'unsubscribe' | 'manual' | 'invalid' | 'spam_trap'
type Item = Schemas['SuppressionItem']
type List = Schemas['SuppressionList']

const reasonMessage: Record<string, MessageDescriptor> = {
  hard_bounce: m.reasonHardBounce,
  complaint: m.reasonComplaint,
  unsubscribe: m.reasonUnsubscribe,
  manual: m.reasonManual,
  spam_trap: m.reasonSpamTrap,
  invalid: m.reasonInvalid,
}
/** A suppression reason's name, or the code itself for one the app doesn't know. */
export function reasonLabel(r: string): string {
  const d = reasonMessage[r]
  return d ? t(d) : r
}
/** Bounces, complaints, traps and invalid addresses block every stream. */
const blocking = new Set(['hard_bounce', 'complaint', 'spam_trap', 'invalid'])

/** 7.12 Suppression list: one list for every provider, with sync state. */
export function SuppressionsPage({ search }: { search: SuppressionsSearch }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const page = search.page ?? 1
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/suppressions',
    {
      params: {
        path: { workspace: slug },
        query: { q: search.q, reason: search.reason, source: search.source, since: search.since, appliesTo: search.appliesTo, import: search.import, page, pageSize: 10 },
      },
    },
    { placeholderData: keepPreviousData },
  )
  const [dialog, setDialog] = useState<'add' | 'import' | null>(null)
  const [removing, setRemoving] = useState<Item | null>(null)
  const editReason = usePermissionReason('subscribers:edit')
  const exportReason = usePermissionReason('subscribers:export')
  const go = (next: SuppressionsSearch) => void navigate({ to: '/w/$workspace/audience/suppressions', params: { workspace: slug }, search: next })
  const setFilter = (patch: Partial<SuppressionsSearch>) => go({ ...search, ...patch, page: undefined })
  const d = list.data
  if (!d && list.error instanceof ApiError && list.error.status === 403) throw list.error
  const filtered = Boolean(search.q || search.reason || search.source || search.since || search.appliesTo || search.import)
  return (
    <Page
      title={tr(am.title)}
      documentTitle={tr(m.documentTitle)}
      filters={<AudienceTabs />}
      actions={
        <>
          <Button disabledReason={editReason} onClick={() => setDialog('import')}>
            {tr(m.importFrom)}
          </Button>
          <Button disabledReason={exportReason} onClick={() => void exportSuppressions(slug, d?.counts.total)}>
            {tr(m.export)}
          </Button>
          <Button variant="primary" disabledReason={editReason} onClick={() => setDialog('add')}>
            {tr(m.addAddress)}
          </Button>
        </>
      }
    >
      {list.isError && !d ? (
        <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(list.error)} {tr(m.nothingChanged)}
        </ErrorPanel>
      ) : !d ? (
        <SuppressionsSkeleton />
      ) : (
        <>
          <Counts d={d} />
          <SyncStrip d={d} />
          {search.channel === 'sms' ? (
            <SmsSuppressions
              search={search}
              onSearch={(s) => go({ ...s, channel: 'sms' })}
              switcher={<SuppressionChannels current="sms" emailTotal={d.counts.total} onChange={(c) => go({ channel: c === 'sms' ? 'sms' : undefined })} />}
            />
          ) : (
          <>
          <Filters
            search={search}
            d={d}
            onChange={setFilter}
            top={<SuppressionChannels current="email" emailTotal={d.counts.total} onChange={(c) => go({ channel: c === 'sms' ? 'sms' : undefined })} />}
          />
          {search.import ? <GroupNote d={d} onBack={() => setFilter({ import: undefined })} /> : null}
          <Note tone="info">
            <span>
              <T msg={m.scopeNote} values={{ b: (c) => <b>{c}</b> }} />
            </span>
          </Note>
          <Table
            d={d}
            page={page}
            filtered={filtered}
            editReason={editReason}
            onRemove={setRemoving}
            onGroup={(id) => setFilter({ import: id, q: undefined })}
            onPage={(p) => go({ ...search, page: p > 1 ? p : undefined })}
            onClear={() => go({})}
          />
          </>
          )}
        </>
      )}
      {dialog === 'add' ? <AddSuppressionsDialog onClose={() => setDialog(null)} /> : null}
      {dialog === 'import' ? <ImportSuppressionsDialog onClose={() => setDialog(null)} /> : null}
      {removing ? <RemoveSuppressionDialog item={removing} onClose={() => setRemoving(null)} /> : null}
    </Page>
  )
}

/** Downloads the whole list as CSV; the jobs tray shows it while it runs. */
async function exportSuppressions(slug: string, total: number | undefined) {
  const id = `export-suppressions-${Date.now()}`
  const what = total === undefined ? t(m.exportWhole) : t(m.exportCount, { n: total, count: count(total) })
  trackJob(slug, { id, kind: 'export', title: t(m.exporting, { what }), status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response, error } = await http.GET('/api/v1/workspaces/{workspace}/suppressions/export', { params: { path: { workspace: slug } }, parseAs: 'blob' })
    if (error || !data) throw new Error(errorMessage(error))
    saveBlob(data as unknown as Blob, fileName(response, 'suppressions.csv'))
    updateJob(slug, id, { status: 'done', detail: t(m.exportedDetail, { what }), finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: t(m.exported, { what }), detail: t(m.exportInAudit) })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
  }
}

function Counts({ d }: { d: List }) {
  const tr = useT()
  const names = d.sync.map((s) => shortProvider(s.name))
  const cards: [string, number, string][] = [
    [tr(m.countTotal), d.counts.total, names.length > 0 ? tr(m.countTotalNote, { providers: andList(names) }) : tr(m.countTotalNoteNone)],
    [tr(m.countHard), d.counts.hardBounce, tr(m.allMailBlocked)],
    [tr(m.countComplaints), d.counts.complaint, tr(m.allMailBlocked)],
    [tr(m.countUnsubscribed), d.counts.unsubscribe, tr(m.marketingOnlyNote)],
    [tr(m.countManual), d.counts.manual, tr(m.manualNote)],
  ]
  if (d.counts.other > 0) cards.push([tr(m.countOther), d.counts.other, tr(m.allMailBlocked)])
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
      {cards.map(([k, n, note]) => (
        <div key={k} className="card flex flex-col gap-0.5 px-4.5 py-3.5">
          <span className="k">{k}</span>
          <span className="n text-[22px]">{count(n)}</span>
          <span className="k">{note}</span>
        </div>
      ))}
    </div>
  )
}

const syncBadge: Record<string, { tone: 'ok' | 'info' | 'bad' | 'neutral'; label: MessageDescriptor }> = {
  in_sync: { tone: 'ok', label: m.inSync },
  syncing: { tone: 'info', label: m.syncing },
  error: { tone: 'bad', label: m.syncFailed },
  off: { tone: 'neutral', label: m.syncOff },
}

function SyncStrip({ d }: { d: List }) {
  const tr = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-wrap" aria-label={tr(m.syncLabel)}>
      {d.sync.length === 0 ? (
        <div className="syncc">
          <span className="font-semibold">{tr(m.noProvider)}</span>
          <span className="k">{tr(m.noProviderBody)}</span>
        </div>
      ) : (
        d.sync.map((s) => {
          const b = syncBadge[s.status] ?? syncBadge.off!
          return (
            <div key={s.providerId} className="syncc">
              <div className="flex items-center justify-between gap-2">
                <span className="font-semibold">{s.name}</span>
                <Badge tone={b.tone}>{s.at && s.status === 'in_sync' ? tr(m.inSyncAt, { time: clock(s.at) }) : tr(b.label)}</Badge>
              </div>
              <span className="k">
                {s.local
                  ? tr(m.syncLocal, { pulled: count(s.pulled), note: s.pulledNote })
                  : tr(m.syncPushed, { pulled: count(s.pulled), note: s.pulledNote, pushed: count(s.pushed) })}
              </span>
              {s.status === 'error' && s.detail ? <span className="k text-bad-fg">{s.detail}</span> : null}
            </div>
          )
        })
      )}
      <div className="flex flex-none items-center px-4.5 py-3.5">
        {d.sync.length === 0 ? (
          <ButtonLink size="sm" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            {tr(m.connectProvider)}
          </ButtonLink>
        ) : (
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }}>
            {tr(m.syncSettings)}
          </ButtonLink>
        )}
      </div>
    </section>
  )
}

function Filters({ search, d, onChange, top }: { search: SuppressionsSearch; d: List; onChange: (patch: Partial<SuppressionsSearch>) => void; top?: ReactNode }) {
  const tr = useT()
  // The box follows the URL and writes back after a short pause in typing.
  const [q, setQ] = useState(search.q ?? '')
  const [urlQ, setUrlQ] = useState(search.q ?? '')
  if ((search.q ?? '') !== urlQ) {
    setUrlQ(search.q ?? '')
    setQ(search.q ?? '')
  }
  useEffect(() => {
    if (q === urlQ) return
    const h = window.setTimeout(() => onChange({ q: q.trim() || undefined }), 250)
    return () => window.clearTimeout(h)
  }, [q, urlQ, onChange])
  const phone = 'max-md:w-0 max-md:min-w-full'
  const sel = `in ${phone}`
  return (
    <div className="card flex flex-wrap items-center gap-2 px-3 py-2.5">
      {top}
      <label className="vh" htmlFor="supp-q">
        {tr(m.search)}
      </label>
      <input
        id="supp-q"
        type="search"
        className="in w-[260px] max-md:min-w-full"
        placeholder={tr(m.searchPlaceholder)}
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <select
        className={sel}
        aria-label={tr(m.reason)}
        value={search.reason ?? ''}
        onChange={(e) => onChange({ reason: (e.target.value || undefined) as SuppressionReason | undefined })}
      >
        <option value="">{tr(m.anyReason)}</option>
        {(['hard_bounce', 'complaint', 'unsubscribe', 'manual', 'invalid', 'spam_trap'] as const).map((r) => (
          <option key={r} value={r}>
            {reasonLabel(r)}
          </option>
        ))}
      </select>
      <select className={sel} aria-label={tr(m.sourceProvider)} value={search.source ?? ''} onChange={(e) => onChange({ source: e.target.value || undefined })}>
        <option value="">{tr(m.anySource)}</option>
        {d.sources.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </select>
      <select
        className={sel}
        aria-label={tr(m.appliesTo)}
        value={search.appliesTo ?? ''}
        onChange={(e) => onChange({ appliesTo: (e.target.value || undefined) as 'all' | 'marketing' | undefined })}
      >
        <option value="">{tr(m.appliesBoth)}</option>
        <option value="all">{tr(m.allMail)}</option>
        <option value="marketing">{tr(m.marketingOnly)}</option>
      </select>
      <select className={sel} aria-label={tr(m.date)} value={search.since ?? ''} onChange={(e) => onChange({ since: (e.target.value || undefined) as '7d' | '30d' | undefined })}>
        <option value="">{tr(m.anyTime)}</option>
        <option value="7d">{tr(m.last7)}</option>
        <option value="30d">{tr(m.last30)}</option>
      </select>
      <span className="k ml-auto" role="status">
        {tr(m.addresses, { n: d.matching, count: count(d.matching) })}
      </span>
    </div>
  )
}

function GroupNote({ d, onBack }: { d: List; onBack: () => void }) {
  const tr = useT()
  const source = d.sources.find((s) => s.value.startsWith('import:'))?.label
  const shown = { n: d.matching, count: count(d.matching) }
  return (
    <Note tone="info">
      <span>
        {source ? tr(m.showingImport, { ...shown, source }) : tr(m.showingOneImport, shown)}{' '}
        <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onBack}>
          {tr(m.showAll)}
        </button>
      </span>
    </Note>
  )
}

function ReasonBadge({ reasons }: { reasons: string[] }) {
  const bad = reasons.some((r) => blocking.has(r))
  return <Badge tone={bad ? 'bad' : 'neutral'}>{reasons.map((r) => reasonLabel(r)).join(' · ')}</Badge>
}

function SourceBadge({ s }: { s: Item['source'] }) {
  if (s.adapter === 'ses') return <Badge tone="purple">{s.label}</Badge>
  if (s.adapter) {
    return (
      <span className="badge mut">
        <span className="dot" style={{ color: providerColor[s.adapter] ?? 'var(--chart-other)' }} aria-hidden="true" />
        {s.label}
      </span>
    )
  }
  return <Badge tone="neutral">{s.label}</Badge>
}

function Table({
  d,
  page,
  filtered,
  editReason,
  onRemove,
  onGroup,
  onPage,
  onClear,
}: {
  d: List
  page: number
  filtered: boolean
  editReason?: string
  onRemove: (it: Item) => void
  onGroup: (importId: string) => void
  onPage: (p: number) => void
  onClear: () => void
}) {
  const tr = useT()
  const last = page * d.pageSize >= d.total
  return (
    <section className="card" aria-label={tr(m.tableLabel)}>
      <TableScroll label={tr(m.tableLabel)}>
        <table className="tbl stbl phone-wrap">
          <thead>
            <tr>
              <Th>{tr(m.colAddress)}</Th>
              <Th>{tr(m.colReason)}</Th>
              <Th>{tr(m.colSource)}</Th>
              <Th>{tr(m.colWhen)}</Th>
              <Th>{tr(m.colApplies)}</Th>
              <th scope="col">
                <span className="vh">{tr(m.colActions)}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {d.items.map((it) => (
              <tr key={it.id}>
                <td className="mono" style={{ whiteSpace: 'normal', maxWidth: 230, overflowWrap: 'anywhere' }}>
                  {it.group ? tr(m.groupRows, { address: it.address, n: it.rows, count: count(it.rows) }) : it.address}
                </td>
                <td>
                  <div className="flex flex-col items-start gap-1">
                    <ReasonBadge reasons={it.reasons} />
                    {it.originalCode ? (
                      <span className="mono k" style={{ fontSize: 12, whiteSpace: 'normal' }}>
                        {it.originalCode}
                      </span>
                    ) : null}
                  </div>
                </td>
                <td>
                  <SourceBadge s={it.source} />
                </td>
                <td>
                  <div className="flex flex-col gap-0.5">
                    <span>{suppressedWhen(it.at)}</span>
                    {it.context ? <span className="k">{it.context}</span> : null}
                  </div>
                </td>
                <td>{it.scope === 'marketing' ? <Badge tone="info">{tr(m.marketingOnly)}</Badge> : <Badge tone="neutral">{tr(m.allMail)}</Badge>}</td>
                <td>
                  {it.group && it.importId ? (
                    <Button size="sm" onClick={() => onGroup(it.importId!)} aria-label={tr(m.viewRowsLabel, { n: it.rows, count: count(it.rows), address: it.address })}>
                      {tr(m.viewRows)}
                    </Button>
                  ) : (
                    <Button size="sm" disabledReason={editReason} onClick={() => onRemove(it)} aria-label={tr(m.removeLabel, { address: it.address })}>
                      {tr(m.remove)}
                    </Button>
                  )}
                </td>
              </tr>
            ))}
            {d.items.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-8 text-center text-ink-3">
                  {filtered ? (
                    <>
                      {tr(m.noMatch)}{' '}
                      <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onClear}>
                        {tr(m.clearFilters)}
                      </button>
                    </>
                  ) : (
                    tr(m.emptyList)
                  )}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4 py-2.5">
        <span className="k">{tr(m.removeNote)}</span>
        <div className="flex gap-1.5">
          <Button size="sm" disabled={page <= 1} onClick={() => onPage(page - 1)}>
            {tr(m.previous)}
          </Button>
          <Button size="sm" disabled={last} onClick={() => onPage(page + 1)}>
            {tr(m.next)}
          </Button>
        </div>
      </div>
    </section>
  )
}

function SuppressionsSkeleton() {
  const tr = useT()
  return (
    <Loading label={tr(m.loading)} className="flex flex-col gap-5">
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="card flex flex-col gap-2 px-4.5 py-3.5">
            <Skeleton w={120} h={11} />
            <Skeleton w={70} h={22} />
            <Skeleton w={150} h={11} />
          </div>
        ))}
      </div>
      <div className="card flex flex-col gap-3 p-4">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-4">
            <Skeleton w={160} h={13} />
            <Skeleton w={84} h={22} r={999} />
            <Skeleton w={96} h={22} r={999} />
            <Skeleton w={150} h={12} />
            <Skeleton w={70} h={12} />
          </div>
        ))}
      </div>
    </Loading>
  )
}
