import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { andList, clock, count, suppressedWhen } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { AudienceTabs } from './AudienceTabs'
import { shortProvider } from './ContactsPage'
import { fileName, saveBlob } from './download'
import { providerColor } from './labels'
import { AddSuppressionsDialog, ImportSuppressionsDialog, RemoveSuppressionDialog } from './SuppressionDialogs'
import './audience.css'

export interface SuppressionsSearch {
  q?: string
  reason?: SuppressionReason
  source?: string
  since?: '7d' | '30d'
  import?: string
  page?: number
}

export type SuppressionReason = 'hard_bounce' | 'complaint' | 'unsubscribe' | 'manual'
type Item = Schemas['SuppressionItem']
type List = Schemas['SuppressionList']

export const reasonLabel: Record<string, string> = {
  hard_bounce: 'Hard bounce',
  complaint: 'Complaint',
  unsubscribe: 'Unsubscribed',
  manual: 'Manual',
  spam_trap: 'Spam trap',
  invalid: 'Invalid address',
}
/** Bounces, complaints, traps and invalid addresses block every stream. */
const blocking = new Set(['hard_bounce', 'complaint', 'spam_trap', 'invalid'])

/** 7.12 Suppression list: one list for every provider, with sync state. */
export function SuppressionsPage({ search }: { search: SuppressionsSearch }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const page = search.page ?? 1
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/suppressions',
    {
      params: {
        path: { workspace: slug },
        query: { q: search.q, reason: search.reason, source: search.source, since: search.since, import: search.import, page, pageSize: 10 },
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
  const filtered = Boolean(search.q || search.reason || search.source || search.since || search.import)
  return (
    <Page
      title="Audience"
      documentTitle="Suppression list"
      filters={<AudienceTabs />}
      actions={
        <>
          <Button disabledReason={editReason} onClick={() => setDialog('import')}>
            Import from provider or CSV
          </Button>
          <Button disabledReason={exportReason} onClick={() => void exportSuppressions(slug, d?.counts.total)}>
            Export
          </Button>
          <Button variant="primary" disabledReason={editReason} onClick={() => setDialog('add')}>
            Add address
          </Button>
        </>
      }
    >
      {list.isError && !d ? (
        <ErrorPanel title="Couldn't load the suppression list" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
          {errorMessage(list.error)} Nothing was changed; sending still checks the list.
        </ErrorPanel>
      ) : !d ? (
        <SuppressionsSkeleton />
      ) : (
        <>
          <Counts d={d} />
          <SyncStrip d={d} />
          <Filters search={search} d={d} onChange={setFilter} />
          {search.import ? <GroupNote d={d} onBack={() => setFilter({ import: undefined })} /> : null}
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
      {dialog === 'add' ? <AddSuppressionsDialog onClose={() => setDialog(null)} /> : null}
      {dialog === 'import' ? <ImportSuppressionsDialog onClose={() => setDialog(null)} /> : null}
      {removing ? <RemoveSuppressionDialog item={removing} onClose={() => setRemoving(null)} /> : null}
    </Page>
  )
}

/** Downloads the whole list as CSV; the jobs tray shows it while it runs. */
async function exportSuppressions(slug: string, total: number | undefined) {
  const id = `export-suppressions-${Date.now()}`
  const what = total === undefined ? 'the suppression list' : `${count(total)} suppressed ${total === 1 ? 'address' : 'addresses'}`
  trackJob(slug, { id, kind: 'export', title: `Exporting ${what}`, status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response, error } = await http.GET('/api/v1/workspaces/{workspace}/suppressions/export', { params: { path: { workspace: slug } }, parseAs: 'blob' })
    if (error || !data) throw new Error(errorMessage(error))
    saveBlob(data as unknown as Blob, fileName(response, 'suppressions.csv'))
    updateJob(slug, id, { status: 'done', detail: `${what} · the export is in the audit log`, finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: `Exported ${what}`, detail: 'The export is in the audit log.' })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: "Couldn't export the suppression list", detail: errorMessage(e) })
  }
}

function Counts({ d }: { d: List }) {
  const names = d.sync.map((s) => shortProvider(s.name))
  const cards: [string, number, string][] = [
    ['Suppressed, all providers', d.counts.total, names.length > 0 ? `one list, enforced on ${andList(names)}` : 'one list, enforced on every provider you connect'],
    ['Hard bounces', d.counts.hardBounce, 'all streams blocked'],
    ['Complaints', d.counts.complaint, 'all streams blocked'],
    ['Unsubscribed', d.counts.unsubscribe, 'Broadcasts and Automations blocked, System allowed'],
    ['Manual', d.counts.manual, 'added by the team or API'],
  ]
  if (d.counts.other > 0) cards.push(['Spam traps and invalid', d.counts.other, 'all streams blocked'])
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

const syncBadge: Record<string, { tone: 'ok' | 'info' | 'bad' | 'neutral'; label: string }> = {
  in_sync: { tone: 'ok', label: 'In sync' },
  syncing: { tone: 'info', label: 'Syncing' },
  error: { tone: 'bad', label: 'Sync failed' },
  off: { tone: 'neutral', label: 'Sync off' },
}

function SyncStrip({ d }: { d: List }) {
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-wrap" aria-label="Suppression sync with sending providers">
      {d.sync.length === 0 ? (
        <div className="syncc">
          <span className="font-semibold">No sending provider connected</span>
          <span className="k">The list applies to every provider you connect; their existing suppressions are pulled in when you do.</span>
        </div>
      ) : (
        d.sync.map((s) => {
          const b = syncBadge[s.status] ?? syncBadge.off!
          return (
            <div key={s.providerId} className="syncc">
              <div className="flex items-center justify-between gap-2">
                <span className="font-semibold">{s.name}</span>
                <Badge tone={b.tone}>
                  {b.label}
                  {s.at && s.status === 'in_sync' ? ` · ${clock(s.at)}` : ''}
                </Badge>
              </div>
              <span className="k">
                {s.local
                  ? `${count(s.pulled)} ${s.pulledNote} · checked before sending`
                  : `${count(s.pulled)} ${s.pulledNote} · ${count(s.pushed)} pushed`}
              </span>
              {s.status === 'error' && s.detail ? <span className="k text-bad-fg">{s.detail}</span> : null}
            </div>
          )
        })
      )}
      <div className="flex flex-none items-center px-4.5 py-3.5">
        {d.sync.length === 0 ? (
          <ButtonLink size="sm" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            Connect a provider
          </ButtonLink>
        ) : (
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }}>
            Sync settings
          </ButtonLink>
        )}
      </div>
    </section>
  )
}

function Filters({ search, d, onChange }: { search: SuppressionsSearch; d: List; onChange: (patch: Partial<SuppressionsSearch>) => void }) {
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
      <label className="vh" htmlFor="supp-q">
        Search suppressed addresses
      </label>
      <input
        id="supp-q"
        type="search"
        className="in w-[260px] max-md:min-w-full"
        placeholder="Search address or domain"
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <select
        className={sel}
        aria-label="Reason"
        value={search.reason ?? ''}
        onChange={(e) => onChange({ reason: (e.target.value || undefined) as SuppressionReason | undefined })}
      >
        <option value="">Any reason</option>
        {(['hard_bounce', 'complaint', 'unsubscribe', 'manual'] as const).map((r) => (
          <option key={r} value={r}>
            {reasonLabel[r]}
          </option>
        ))}
      </select>
      <select className={sel} aria-label="Source provider" value={search.source ?? ''} onChange={(e) => onChange({ source: e.target.value || undefined })}>
        <option value="">Any source provider</option>
        {d.sources.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </select>
      <select className={sel} aria-label="Date" value={search.since ?? ''} onChange={(e) => onChange({ since: (e.target.value || undefined) as '7d' | '30d' | undefined })}>
        <option value="">Any time</option>
        <option value="7d">Last 7 days</option>
        <option value="30d">Last 30 days</option>
      </select>
      <span className="k ml-auto" role="status">
        {count(d.matching)} {d.matching === 1 ? 'address' : 'addresses'}
      </span>
    </div>
  )
}

function GroupNote({ d, onBack }: { d: List; onBack: () => void }) {
  const source = d.sources.find((s) => s.value.startsWith('import:'))?.label
  return (
    <Note tone="info">
      <span>
        Showing the {count(d.matching)} {d.matching === 1 ? 'address' : 'addresses'} {source ? `from the ${source}` : 'from one import'}.{' '}
        <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onBack}>
          Show the whole list
        </button>
      </span>
    </Note>
  )
}

function ReasonBadge({ reasons }: { reasons: string[] }) {
  const bad = reasons.some((r) => blocking.has(r))
  return <Badge tone={bad ? 'bad' : 'neutral'}>{reasons.map((r) => reasonLabel[r] ?? r).join(' · ')}</Badge>
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
  const last = page * d.pageSize >= d.total
  return (
    <section className="card" aria-label="Suppressed addresses">
      <div className="tscroll">
        <table className="tbl stbl phone-wrap">
          <thead>
            <tr>
              <th scope="col">Address</th>
              <th scope="col">Reason</th>
              <th scope="col">Source provider</th>
              <th scope="col">Original code</th>
              <th scope="col">When</th>
              <th scope="col">Campaign or flow</th>
              <th scope="col">Scope</th>
              <th scope="col">
                <span className="vh">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {d.items.map((it) => (
              <tr key={it.id}>
                <td className="mono">{it.group ? `${it.address} · ${count(it.rows)} rows` : it.address}</td>
                <td>
                  <ReasonBadge reasons={it.reasons} />
                </td>
                <td>
                  <SourceBadge s={it.source} />
                </td>
                <td className="mono text-[12px]">{it.originalCode || '–'}</td>
                <td>{suppressedWhen(it.at)}</td>
                <td>{it.context || '–'}</td>
                <td className="k">{it.scope === 'marketing' ? 'Broadcasts, Automations' : 'All streams'}</td>
                <td>
                  {it.group && it.importId ? (
                    <Button size="sm" onClick={() => onGroup(it.importId!)} aria-label={`View the ${count(it.rows)} rows from ${it.address}`}>
                      View rows
                    </Button>
                  ) : (
                    <Button size="sm" disabledReason={editReason} onClick={() => onRemove(it)} aria-label={`Remove ${it.address}`}>
                      Remove
                    </Button>
                  )}
                </td>
              </tr>
            ))}
            {d.items.length === 0 ? (
              <tr>
                <td colSpan={8} className="py-8 text-center text-ink-3">
                  {filtered ? (
                    <>
                      No suppressed addresses match these filters.{' '}
                      <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onClear}>
                        Clear filters
                      </button>
                    </>
                  ) : (
                    'Nothing is suppressed yet. Hard bounces, complaints and unsubscribes land here on their own, from every provider.'
                  )}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4 py-2.5">
        <span className="k">
          Removing an address asks for a reason and is logged in the audit log; bounced addresses can't be re-added to a list without a new opt-in.
        </span>
        <div className="flex gap-1.5">
          <Button size="sm" disabled={page <= 1} onClick={() => onPage(page - 1)}>
            Previous
          </Button>
          <Button size="sm" disabled={last} onClick={() => onPage(page + 1)}>
            Next
          </Button>
        </div>
      </div>
    </section>
  )
}

function SuppressionsSkeleton() {
  return (
    <Loading label="Loading the suppression list" className="flex flex-col gap-5">
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
