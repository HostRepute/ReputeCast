import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { createColumnHelper, rowSelectionFeature, tableFeatures, useTable, type RowSelectionState } from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { TableScroll } from '@/components/ui/Table'
import { andList, count, dayMonthDate, shortDate } from '@/lib/format'
import { trackJob, updateJob } from '@/lib/jobs'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { AddressCheckFilter } from '@/features/verification/AddressCheck'
import { AudienceTabs, useAudienceSummary } from './AudienceTabs'
import { AddContactDialog, BulkDialog, BulkResultPanel, EraseDialog, SaveViewDialog, type BulkKind } from './ContactDialogs'
import { BandSelect, ScoreCell, ScoreGrowthChips, ScoreGrowthFields, scoreGrowthMoreKeys, ScoreHeader, useTotalLine, type ScoreGrowthDraft } from '@/features/scoring/ContactsScore'
import { fileName, saveBlob } from './download'
import { consentFilterLabel, consentLabel, consentTone, mailboxColor, mailboxes, mailboxLabel, tierClass, tierColor, tierLabel, tiers, type Consent, type Mailbox, type Tier } from './labels'
import { LanguageFilter } from '@/features/page-languages/contacts'
import { PaidStatusCell, PaidStatusSelect, PaidTierChip } from '@/features/earn/ContactsPaid'
import { StoreChips, StoreContactFields, storeMoreKeys, type StoreDraft } from '@/features/commerce/contacts'
import './audience.css'
import { MembersChips } from '@/features/website/elsewhere'

export interface ContactsSearch {
  q?: string
  list?: string
  tier?: Tier
  mailbox?: Mailbox
  consent?: Consent
  source?: string
  tag?: string
  joinedAfter?: string
  segment?: string
  language?: string
  // Scoring and Growth (X13).
  band?: 'cold' | 'warm' | 'hot' | 'none'
  scoreFrom?: number
  scoreTo?: number
  growthSource?: string
  utmSource?: string
  utmMedium?: string
  utmCampaign?: string
  referringDomain?: string
  sort?: 'activity' | 'score'
  /** The latest address check (X14). */
  check?: 'valid' | 'invalid' | 'risky' | 'unknown' | 'never'
  /** Paid status (Earn, X16): a status, or with_access at paidTier and above. */
  paid?: 'with_access' | 'trialing' | 'past_due' | 'complimentary' | 'gift' | 'ended' | 'free'
  paidTier?: string
  /** Members of the website: contacts who signed in to it (X18). */
  signedIn?: boolean
  /** Store customers and what they spent (X20). */
  storeCustomer?: string
  spentFrom?: string
  spentTo?: string
  page?: number
  size?: number
}

type Item = Schemas['ContactListItem']
type Filter = Schemas['ContactFilter']

const filterKeys = [
  'q',
  'list',
  'tier',
  'mailbox',
  'consent',
  'source',
  'tag',
  'joinedAfter',
  'segment',
  'language',
  // Scoring and Growth (X13).
  'band',
  'scoreFrom',
  'scoreTo',
  'growthSource',
  'utmSource',
  'utmMedium',
  'utmCampaign',
  'referringDomain',
  'sort',
  // The latest address check (X14).
  'check',
  // Paid status (Earn, X16).
  'paid',
  'paidTier',
  // Members of the website (X18).
  'signedIn',
  // Store customers and what they spent (X20).
  'storeCustomer',
  'spentFrom',
  'spentTo',
] as const

/** The filter part of the search params, without empty values. */
export function filterOf(s: ContactsSearch): Filter {
  const f: Record<string, string | boolean> = {}
  for (const k of filterKeys) {
    const v = s[k]
    if (v) f[k] = typeof v === 'boolean' ? v : String(v)
  }
  return f as Filter
}

function sameFilter(a: Filter, b: Filter) {
  return filterKeys.every((k) => (a[k as keyof Filter] ?? '') === (b[k as keyof Filter] ?? ''))
}

/** "Tue 7 Oct" within the last half year, "14 Aug" before, "3 Jun 2025" for older years. */
function lastOpen(iso: string | undefined, machine: boolean) {
  if (!iso) return 'never'
  const d = new Date(iso)
  const days = (Date.now() - d.getTime()) / 86_400_000
  const s = days < 14 ? shortDate(d) : days < 330 ? dayMonthDate(d) : `${dayMonthDate(d)} ${d.getFullYear()}`
  return machine ? `${s} · machine` : s
}

function joined(iso: string) {
  const d = new Date(iso)
  return `${dayMonthDate(d)} ${d.getFullYear()}`
}

const features = tableFeatures({ rowSelectionFeature })
const col = createColumnHelper<typeof features, Item>()

/** 7.9 Contacts: saved views, filters, bulk actions and the table. */
export function ContactsPage({ search }: { search: ContactsSearch }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const filter = filterOf(search)
  const page = search.page ?? 1
  const size = search.size ?? 10
  const summary = useAudienceSummary()
  const views = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/views', { params: { path: { workspace: slug } } })
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/contacts',
    { params: { path: { workspace: slug }, query: { ...filter, page, pageSize: size } } },
    { placeholderData: keepPreviousData },
  )
  // The selection belongs to one page of one filter: another page or filter
  // starts with nothing selected.
  const key = JSON.stringify({ filter, page, size })
  const [picked, setPicked] = useState<{ key: string; rows: RowSelectionState }>({ key, rows: {} })
  const selection = picked.key === key ? picked.rows : {}
  const setSelection = (rows: RowSelectionState) => setPicked({ key, rows })
  const [adding, setAdding] = useState(false)
  const [saving, setSaving] = useState(false)
  const [bulk, setBulk] = useState<BulkKind | null>(null)
  const [erasing, setErasing] = useState(false)
  const [result, setResult] = useState<Schemas['ContactBulkResult'] | null>(null)
  const addReason = usePermissionReason('subscribers:create')
  const importReason = usePermissionReason('subscribers:import')

  const go = (next: ContactsSearch) => void navigate({ to: '/w/$workspace/audience', params: { workspace: slug }, search: next })
  const setFilter = (patch: Partial<ContactsSearch>) => go({ ...search, ...patch, page: undefined })

  const empty = summary.data && summary.data.contacts === 0
  const ids = Object.keys(selection)
  return (
    <Page
      title="Audience"
      filters={<AudienceTabs />}
      actions={
        <>
          {importReason ? (
            <Button disabledReason={importReason}>Import</Button>
          ) : (
            <ButtonLink to="/w/$workspace/import" params={{ workspace: slug }}>
              Import
            </ButtonLink>
          )}
          <Button variant="primary" disabledReason={addReason} onClick={() => setAdding(true)}>
            Add contact
          </Button>
        </>
      }
      contentClassName="py-5 gap-3.5"
    >
      {empty ? (
        <AudienceEmpty lists={summary.data?.lists ?? []} />
      ) : (
        <>
          <SavedViews views={views.data} filter={filter} onPick={(f) => go({ ...f, size: search.size })} onSave={() => setSaving(true)} />
          {/* The website's members (features/website), while it's on. */}
          <MembersChips
            signedIn={Boolean(search.signedIn)}
            onSignedIn={(on) => setFilter({ signedIn: on || undefined })}
            paid={search.paid === 'with_access' && !search.paidTier}
            onPaid={(on) => setFilter({ paid: on ? 'with_access' : undefined, paidTier: undefined })}
          />
          <Filters search={search} total={list.data?.total} lists={summary.data?.lists ?? []} paidTiers={summary.data?.paidTiers ?? []} onChange={setFilter} />
          {result ? <BulkResultPanel result={result} onDismiss={() => setResult(null)} /> : null}
          {ids.length > 0 ? (
            <BulkBar
              n={ids.length}
              onAction={(k) => (k === 'erase' ? setErasing(true) : setBulk(k))}
              onExport={() => void exportCSV(slug, { ids }, ids.length)}
              onClear={() => setSelection({})}
            />
          ) : null}
          {list.isError ? (
            <ErrorPanel title="Couldn't load contacts" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
              {errorMessage(list.error)} Nothing was changed.
            </ErrorPanel>
          ) : list.isPending ? (
            <ContactsSkeleton total={summary.data?.contacts} />
          ) : (
            <ContactsTable
              data={list.data}
              selection={selection}
              onSelection={setSelection}
              page={page}
              size={size}
              filtered={Object.keys(filter).length > 0}
              paid={(summary.data?.paidTiers.length ?? 0) > 0}
              byScore={search.sort === 'score'}
              onSort={(score) => go({ ...search, sort: score ? 'score' : undefined, page: undefined })}
              onPage={(p) => go({ ...search, page: p > 1 ? p : undefined })}
              onSize={(s) => go({ ...search, size: s === 10 ? undefined : s, page: undefined })}
              onClear={() => go({ size: search.size })}
            />
          )}
        </>
      )}
      {adding ? <AddContactDialog onClose={() => setAdding(false)} lists={summary.data?.lists ?? []} /> : null}
      {saving ? <SaveViewDialog onClose={() => setSaving(false)} filter={filter} /> : null}
      {bulk ? (
        <BulkDialog
          kind={bulk}
          onClose={() => setBulk(null)}
          ids={ids}
          lists={summary.data?.lists ?? []}
          onDone={(r) => {
            setSelection({})
            if (r.failed > 0) setResult(r)
          }}
        />
      ) : null}
      {erasing ? (
        <EraseDialog
          onClose={() => setErasing(false)}
          ids={ids}
          onDone={() => {
            setSelection({})
            setErasing(false)
          }}
        />
      ) : null}
    </Page>
  )
}

/** Exports the selection as CSV; the jobs tray shows it while it runs. */
export async function exportCSV(slug: string, body: Schemas['ContactExportInput'], n: number) {
  const id = `export-${Date.now()}`
  const what = `${count(n)} ${n === 1 ? 'contact' : 'contacts'}`
  trackJob(slug, { id, kind: 'export', title: `Exporting ${what}`, status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response } = await http.POST('/api/v1/workspaces/{workspace}/contacts/export', { params: { path: { workspace: slug } }, body, parseAs: 'blob' })
    if (data) saveBlob(data as unknown as Blob, fileName(response, 'contacts.csv'))
    updateJob(slug, id, { status: 'done', detail: `${what} · the export is in the audit log`, finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: `Exported ${what}`, detail: 'The export is in the audit log.' })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: "Couldn't export", detail: errorMessage(e) })
  }
}

function SavedViews({ views, filter, onPick, onSave }: { views?: Schemas['ContactSavedViews']; filter: Filter; onPick: (f: ContactsSearch) => void; onSave: () => void }) {
  const none = Object.keys(filter).length === 0
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Saved views">
      <span className="k mr-1">Saved views</span>
      <button type="button" className="vchip" aria-pressed={none} onClick={() => onPick({})}>
        All contacts{views ? ` · ${count(views.all)}` : ''}
      </button>
      {views?.items.map((v) => {
        const tier = v.filters.tier as Tier | undefined
        const onlyTier = tier && Object.keys(v.filters).length === 1
        return (
          <button key={v.id} type="button" className="vchip" aria-pressed={!none && sameFilter(filter, v.filters)} onClick={() => onPick(v.filters as ContactsSearch)}>
            {onlyTier ? <span className="dot" style={{ color: tierColor[tier] }} aria-hidden="true" /> : null}
            {v.name} · {count(v.count)}
          </button>
        )
      })}
      <button type="button" className="vchip dashed" onClick={onSave}>
        Save current view
      </button>
    </div>
  )
}

const sourceOptions: [string, string][] = [
  ['import:brevo', 'Brevo import'],
  ['import:mailchimp', 'Mailchimp import'],
  ['import:csv', 'CSV import'],
  ['form:docs', 'Signup form · docs'],
  ['form:blog', 'Signup form · blog'],
  ['form:footer', 'Signup form · footer'],
  ['form:pricing', 'Signup form · pricing'],
  ['form:popup', 'Popup form'],
  ['api', 'API'],
  ['app', 'Added in ReputeCast'],
  // Grow (design expansion, X15): any referral, any partner.
  ['referral', 'Referral'],
  ['partner', 'Partner'],
]

function Filters({
  search,
  total,
  lists,
  paidTiers,
  onChange,
}: {
  search: ContactsSearch
  total?: number
  lists: Schemas['AudienceListSummary'][]
  paidTiers: Schemas['AudiencePaidTier'][]
  onChange: (patch: Partial<ContactsSearch>) => void
}) {
  // The box follows the URL (a saved view or "Clear filters" changes it)
  // and writes back after a short pause in typing.
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
  const more =
    [search.source, search.tag, search.joinedAfter, search.segment].filter(Boolean).length +
    scoreGrowthMoreKeys.filter((k) => search[k] !== undefined).length +
    storeMoreKeys.filter((k) => search[k] !== undefined).length
  const totalLine = useTotalLine()
  const phone = 'max-md:w-0 max-md:min-w-full'
  const sel = `in ${phone}`
  return (
    <div className="card flex flex-wrap items-center gap-2 px-3 py-2.5">
      <label className="vh" htmlFor="contacts-q">
        Search contacts
      </label>
      <input
        id="contacts-q"
        type="search"
        className="in w-[260px] max-md:min-w-full"
        placeholder="Search email, name or tag"
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <select className={sel} aria-label="List" value={search.list ?? ''} onChange={(e) => onChange({ list: e.target.value || undefined })}>
        <option value="">Any list</option>
        {lists.map((l) => (
          <option key={l.id} value={l.id}>
            {l.name}
          </option>
        ))}
      </select>
      <select className={sel} aria-label="Tier" value={search.tier ?? ''} onChange={(e) => onChange({ tier: (e.target.value || undefined) as Tier | undefined })}>
        <option value="">Any tier</option>
        {tiers.map((t) => (
          <option key={t} value={t}>
            {tierLabel[t]}
          </option>
        ))}
      </select>
      <BandSelect className={sel} value={search.band} onChange={(band) => onChange({ band })} />
      <select className={sel} aria-label="Mailbox provider" value={search.mailbox ?? ''} onChange={(e) => onChange({ mailbox: (e.target.value || undefined) as Mailbox | undefined })}>
        <option value="">Any mailbox provider</option>
        {mailboxes.map((m) => (
          <option key={m} value={m}>
            {mailboxLabel[m]}
          </option>
        ))}
      </select>
      <select
        className={sel}
        aria-label="Consent"
        value={search.consent ?? ''}
        onChange={(e) => onChange({ consent: (e.target.value || undefined) as Consent | undefined })}
      >
        <option value="">Any consent</option>
        {(['double', 'single', 'pending'] as Consent[]).map((c) => (
          <option key={c} value={c}>
            {consentFilterLabel[c]}
          </option>
        ))}
      </select>
      <LanguageFilter className={sel} value={search.language} onChange={(language) => onChange({ language })} />
      <AddressCheckFilter className={sel} value={search.check} onChange={(check) => onChange({ check })} />
      <PaidStatusSelect className={sel} tiers={paidTiers} paid={search.paid} tier={search.paidTier} onChange={(f) => onChange(f as Partial<ContactsSearch>)} />
      <MoreFilters search={search} n={more} onChange={onChange} />
      <span className="k ml-auto">{total === undefined ? '' : totalLine(total, search.sort === 'score')}</span>
      <ScoreGrowthChips search={search} onChange={onChange} />
      <StoreChips search={search} onChange={onChange} />
    </div>
  )
}

function MoreFilters({ search, n, onChange }: { search: ContactsSearch; n: number; onChange: (patch: Partial<ContactsSearch>) => void }) {
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } }, { enabled: open })
  const [source, setSource] = useState(search.source ?? '')
  const [tag, setTag] = useState(search.tag ?? '')
  const [after, setAfter] = useState(search.joinedAfter ?? '')
  const [segment, setSegment] = useState(search.segment ?? '')
  const scoreGrowth = (): ScoreGrowthDraft => Object.fromEntries(scoreGrowthMoreKeys.map((k) => [k, search[k]]))
  const [extra, setExtra] = useState<ScoreGrowthDraft>(scoreGrowth)
  const storeDraft = (): StoreDraft => ({ storeCustomer: search.storeCustomer, spentFrom: search.spentFrom, spentTo: search.spentTo })
  const [stores, setStores] = useState<StoreDraft>(storeDraft)
  const toggle = (o: boolean) => {
    if (o) {
      // Opening shows the filters that apply now.
      setSource(search.source ?? '')
      setTag(search.tag ?? '')
      setAfter(search.joinedAfter ?? '')
      setSegment(search.segment ?? '')
      setExtra(scoreGrowth())
      setStores(storeDraft())
    }
    setOpen(o)
  }
  return (
    <Popover
      open={open}
      onOpenChange={toggle}
      align="start"
      label="More filters"
      // With the store fields (X20) it outgrows a laptop screen: it scrolls within what's left of it.
      className="max-h-(--radix-popover-content-available-height) w-[360px] gap-3 overflow-y-auto p-3.5"
      trigger={<button className="btn">{n ? `More filters · ${n}` : 'More filters'}</button>}
    >
      <Field label="Source">
        <Select value={source} onChange={(e) => setSource(e.target.value)}>
          <option value="">Any source</option>
          {sourceOptions.map(([v, l]) => (
            <option key={v} value={v}>
              {l}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Tag">
        <Input value={tag} onChange={(e) => setTag(e.target.value)} placeholder="customer" />
      </Field>
      <Field label="Joined after">
        <Input type="date" value={after} onChange={(e) => setAfter(e.target.value)} />
      </Field>
      <Field label="Segment">
        <Select value={segment} onChange={(e) => setSegment(e.target.value)}>
          <option value="">Any segment</option>
          {segments.data?.items.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
      </Field>
      <ScoreGrowthFields draft={extra} onChange={setExtra} open={open} />
      <StoreContactFields draft={stores} onChange={setStores} open={open} />
      <div className="flex justify-end gap-2">
        <Button
          size="sm"
          onClick={() => {
            onChange({
              source: undefined,
              tag: undefined,
              joinedAfter: undefined,
              segment: undefined,
              ...Object.fromEntries([...scoreGrowthMoreKeys, ...storeMoreKeys].map((k) => [k, undefined])),
            })
            setOpen(false)
          }}
        >
          Clear
        </Button>
        <Button
          size="sm"
          variant="primary"
          onClick={() => {
            onChange({ source: source || undefined, tag: tag.trim() || undefined, joinedAfter: after || undefined, segment: segment || undefined, ...extra, ...stores })
            setOpen(false)
          }}
        >
          Apply
        </Button>
      </div>
    </Popover>
  )
}

function BulkBar({ n, onAction, onExport, onClear }: { n: number; onAction: (k: BulkKind | 'erase') => void; onExport: () => void; onClear: () => void }) {
  const editReason = usePermissionReason('subscribers:edit')
  const exportReason = usePermissionReason('subscribers:export')
  const deleteReason = usePermissionReason('subscribers:delete')
  const canExport = useCan('subscribers:export')
  return (
    <div className="card flex flex-wrap items-center gap-2.5 border-accent-line bg-accent-tint px-3 py-2" role="region" aria-label="Bulk actions">
      <span className="text-[13px] font-semibold" role="status">
        {count(n)} selected
      </span>
      <Button size="sm" disabledReason={editReason} onClick={() => onAction('add_to_list')}>
        Add to list
      </Button>
      <Button size="sm" disabledReason={editReason} onClick={() => onAction('add_tag')}>
        Add tag
      </Button>
      <Button size="sm" disabledReason={editReason} onClick={() => onAction('resend_optin')}>
        Resend opt-in
      </Button>
      <Button size="sm" disabledReason={exportReason} title="Needs subscribers:export · every export is logged" onClick={onExport}>
        Export CSV
      </Button>
      <span className="k">{canExport ? 'export is logged · needs subscribers:export' : 'needs subscribers:export'}</span>
      <Button size="sm" className="dline" disabledReason={editReason} onClick={() => onAction('suppress')}>
        Suppress
      </Button>
      <Button size="sm" className="dline" disabledReason={deleteReason} onClick={() => onAction('erase')}>
        Delete
      </Button>
      <Button size="sm" className="ml-auto" onClick={onClear}>
        Clear selection
      </Button>
    </div>
  )
}

function ContactsTable({
  data,
  selection,
  onSelection,
  page,
  size,
  filtered,
  paid,
  onPage,
  onSize,
  onClear,
  byScore,
  onSort,
}: {
  data: Schemas['ContactList']
  selection: RowSelectionState
  onSelection: (s: RowSelectionState) => void
  page: number
  size: number
  filtered: boolean
  /** Paid status shows (Earn, X16): the tier chip and the Paid column. */
  paid: boolean
  onPage: (p: number) => void
  onSize: (s: number) => void
  onClear: () => void
  byScore: boolean
  onSort: (score: boolean) => void
}) {
  const { slug } = useWorkspace()
  const columns = useMemo(
    () =>
      col.columns([
        col.display({
          id: 'select',
          header: ({ table }) => (
            <input
              type="checkbox"
              aria-label="Select all"
              checked={table.getIsAllPageRowsSelected()}
              ref={(el) => {
                if (el) el.indeterminate = table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()
              }}
              onChange={table.getToggleAllPageRowsSelectedHandler()}
            />
          ),
          cell: ({ row }) => (
            <input type="checkbox" aria-label={`Select ${row.original.name || row.original.email}`} checked={row.getIsSelected()} onChange={row.getToggleSelectedHandler()} />
          ),
        }),
      ]),
    [],
  )
  const table = useTable(
    {
      features,
      columns,
      data: data.items,
      getRowId: (r) => r.id,
      state: { rowSelection: selection },
      onRowSelectionChange: (u) => onSelection(typeof u === 'function' ? u(selection) : u),
    },
    (s) => ({ rowSelection: s.rowSelection }),
  )
  const header = table.getHeaderGroups()[0]!.headers[0]!
  const from = data.total === 0 ? 0 : (page - 1) * size + 1
  const to = Math.min(data.total, page * size)
  return (
    <section className="card" aria-label="Contacts">
      <TableScroll label="Contacts">
        <table className="tbl ctbl phone-wrap">
          <thead>
            <tr>
              <th scope="col">
                <table.FlexRender header={header} />
              </th>
              <th scope="col">Contact</th>
              {paid ? <th scope="col">Paid</th> : null}
              <th scope="col">Lists</th>
              <th scope="col">Tier</th>
              <ScoreHeader sorted={byScore} onSort={onSort} />
              <th scope="col">Consent</th>
              <th scope="col">Mailbox provider</th>
              <th scope="col">Last open</th>
              <th scope="col">Joined</th>
              <th scope="col">Source</th>
            </tr>
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => {
              const c = row.original
              const tier = c.tier as Tier
              const consent = c.consent as Consent
              return (
                <tr key={row.id} className={row.getIsSelected() ? 'sel' : undefined}>
                  <td>
                    <table.FlexRender cell={row.getAllCells()[0]!} />
                  </td>
                  <td>
                    <Link
                      to="/w/$workspace/audience/contacts/$contactId"
                      params={{ workspace: slug, contactId: c.id }}
                      className="font-semibold text-ink no-underline"
                    >
                      {c.name || c.email}
                    </Link>
                    {paid ? <PaidTierChip paid={c.paid} /> : null}
                    <div className="k mono">{c.email}</div>
                  </td>
                  {paid ? (
                    <td className="nw">
                      <PaidStatusCell paid={c.paid} />
                    </td>
                  ) : null}
                  <td className="lists">{c.lists.join(', ') || '–'}</td>
                  <td>
                    <span className={`badge ${tierClass[tier]}`}>{tierLabel[tier]}</span>
                  </td>
                  <td className="nw">
                    <ScoreCell score={c.score} band={c.scoreBand} />
                  </td>
                  <td>
                    {c.suppressed ? (
                      <Badge tone="bad">Suppressed</Badge>
                    ) : (
                      <Badge tone={consentTone[consent]}>{consentLabel[consent]}</Badge>
                    )}
                  </td>
                  {/* Mailbox provider, last open and source wrap so the table fits its card (Contacts board). */}
                  <td className="wrap">
                    <span className="flex items-center gap-1.5">
                      <span className="dot" style={{ color: mailboxColor[c.mailboxProvider] ?? mailboxColor.other }} aria-hidden="true" />
                      {c.mailboxLabel}
                    </span>
                  </td>
                  <td className="wrap">{lastOpen(c.lastOpenAt, c.lastOpenMachine)}</td>
                  <td>{joined(c.joinedAt)}</td>
                  <td className="wrap">{c.sourceLabel}</td>
                </tr>
              )
            })}
            {data.items.length === 0 ? (
              <tr>
                <td colSpan={10} className="py-8 text-center text-ink-3">
                  {filtered ? (
                    <>
                      No contacts match these filters.{' '}
                      <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onClear}>
                        Clear filters
                      </button>
                    </>
                  ) : (
                    'No contacts on this page.'
                  )}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4 py-2.5">
        <span className="k">
          {count(from)} to {count(to)} of {count(data.total)}
        </span>
        <div className="flex items-center gap-1.5 max-md:flex-wrap">
          <select className="in h-[30px] max-md:w-0 max-md:min-w-full" aria-label="Rows per page" value={size} onChange={(e) => onSize(Number(e.target.value))}>
            {[10, 50, 100].map((n) => (
              <option key={n} value={n}>
                {n} per page
              </option>
            ))}
          </select>
          <Button size="sm" disabled={page <= 1} onClick={() => onPage(page - 1)}>
            Previous
          </Button>
          <Button size="sm" disabled={to >= data.total} onClick={() => onPage(page + 1)}>
            Next
          </Button>
        </div>
      </div>
    </section>
  )
}

/** Loading: rows shaped like the table (AudienceStates › Loading). */
export function ContactsSkeleton({ total }: { total?: number }) {
  const widths = [
    [140, 200],
    [120, 180],
    [160, 220],
    [130, 190],
  ]
  return (
    <Loading label={total ? `Loading ${count(total)} contacts, page 1` : 'Loading contacts'} className="card flex flex-col">
      {widths.map(([a, b], i) => (
        <div key={i} className={`flex items-center gap-3 px-4 py-2.5 ${i ? 'border-t border-line-2' : ''}`}>
          <Skeleton w={16} h={16} />
          <Skeleton w={a} h={14} />
          <Skeleton w={b} h={14} />
          <Skeleton w={70} h={22} r={999} />
          <Skeleton w={110} h={22} r={999} />
          <Skeleton w={90} h={14} />
          <Skeleton w={80} h={14} className="ml-auto" />
        </div>
      ))}
    </Loading>
  )
}

/** Empty: no contacts yet (AudienceStates › Empty). */
function AudienceEmpty({ lists }: { lists: Schemas['AudienceListSummary'][] }) {
  const { slug } = useWorkspace()
  const sync = $api.useQuery('get', '/api/v1/workspaces/{workspace}/suppressions', { params: { path: { workspace: slug }, query: { pageSize: 1 } } })
  const pulled = sync.data?.sync.filter((p) => p.pulled > 0) ?? []
  const names = lists.map((l) => l.name)
  const listText = names.length ? `${andList(names)} ${names.length > 1 ? 'are' : 'is'} ready. ` : ''
  return (
    <section className="card flex flex-col items-center gap-3 border-dashed px-7 py-10 text-center">
      <svg width="56" height="56" viewBox="0 0 56 56" aria-hidden="true">
        <circle cx="22" cy="20" r="8" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M8 44a14 14 0 0 1 28 0M38 14a7 7 0 0 1 0 12M50 44a12 12 0 0 0-9-11.5" fill="none" stroke="#9AA3B2" strokeWidth="2" strokeLinecap="round" />
      </svg>
      <h2 className="m-0 text-[18px] font-bold">Your lists are empty</h2>
      <span className="max-w-[520px] text-ink-2">
        {listText}Bring contacts from another tool with their consent intact, add a signup form, or push them in through the API.
      </span>
      <div className="mt-1.5 flex flex-wrap justify-center gap-2.5">
        <ButtonLink variant="primary" className="min-h-10" to="/w/$workspace/import" params={{ workspace: slug }}>
          Import contacts
        </ButtonLink>
        <ButtonLink to="/w/$workspace/forms" params={{ workspace: slug }}>
          Create a signup form
        </ButtonLink>
        <ButtonLink to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          API docs
        </ButtonLink>
      </div>
      {pulled.length ? (
        <span className="k">
          Suppressions from connected providers ({pulled.map((p) => `${count(p.pulled)} from ${shortProvider(p.name)}`).join(', ')}) are already in place, so imported
          addresses that bounced before are skipped.
        </span>
      ) : null}
    </section>
  )
}

/** "SES" for "Amazon SES". */
export function shortProvider(name: string) {
  return name.replace(/^Amazon /, '')
}
