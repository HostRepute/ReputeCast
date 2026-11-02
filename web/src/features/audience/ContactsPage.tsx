import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { createColumnHelper, rowSelectionFeature, tableFeatures, useTable, type RowSelectionState } from '@tanstack/react-table'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { Check, SelectAll } from '@/components/ui/SelectAll'
import { TableScroll, Th } from '@/components/ui/Table'
import { andList, count, dayMonthDate, dayMonthYear, shortDate } from '@/lib/format'
import { trackJob, updateJob, useBackgroundJobs } from '@/lib/jobs'
import { Note } from '@/components/ui/Note'
import { VerifyDialog } from '@/features/verification/VerifyDialog'
import { ColumnChooser, columnLabels, defaultColumns, fieldKeys, parseColumns, SortHeader, type BuiltinColumn, type ColumnId, type ContactSort } from './ContactColumns'
import { ViewChips } from './ViewChips'
import { m as lfm } from './listsfields.messages'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { useFirstLoadSettled } from '@/lib/hooks'
import { AddressCheckFilter } from '@/features/verification/AddressCheck'
import { AudienceTabs, useAudienceSummary } from './AudienceTabs'
import { AddContactDialog, BulkDialog, BulkResultPanel, EraseDialog, SaveViewDialog, type BulkKind, type BulkSelection } from './ContactDialogs'
import { BandSelect, GrowthSourceField, ScoreCell, ScoreGrowthChips, ScoreGrowthFields, scoreGrowthMoreKeys, useTotalLine, type ScoreGrowthDraft } from '@/features/scoring/ContactsScore'
import { fileName, saveBlob } from './download'
import { consentFilterLabel, consentLabel, consentTone, mailboxColor, mailboxes, mailboxLabel, tierClass, tierLabel, tiers, type Consent, type Mailbox, type Tier } from './labels'
import { LanguageFilter } from '@/features/page-languages/contacts'
import { contactListQuery, contactViewsQuery, filterKeys, filterOf } from './queries'
import { PaidStatusCell, PaidStatusSelect, PaidTierChip } from '@/features/earn/ContactsPaid'
import { BoughtFilter } from '@/features/products/filter'
import { StoreChips, StoreContactFields, storeMoreKeys, type StoreDraft } from '@/features/commerce/contacts'
import './audience.css'
import { ChannelChips } from '@/features/push/ContactFilters'
import { MembersChips } from '@/features/website/elsewhere'
import { SmsContactFilters, usePhoneColumn } from '@/features/sms/ContactFilters'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m as cfm } from '@/features/sms/contactfilters.messages'
import { m as am } from './messages'
import { m } from './contacts.messages'

export interface ContactsSearch {
  q?: string
  list?: string
  tier?: Tier
  mailbox?: Mailbox
  consent?: Consent
  source?: string
  tag?: string
  joinedAfter?: string
  /** Joined on or before this day, and the contacts of one import (review 2026-11 AU-10). */
  joinedBefore?: string
  import?: string
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
  sort?: ContactSort
  /** The latest address check (X14). */
  check?: 'valid' | 'invalid' | 'risky' | 'unknown' | 'never'
  /** Paid status (Earn, X16): a status, or with_access at paidTier and above. */
  paid?: 'with_access' | 'trialing' | 'past_due' | 'complimentary' | 'gift' | 'ended' | 'free'
  paidTier?: string
  /** Members of the website: contacts who signed in to it (X18). */
  signedIn?: boolean
  /** Buyers of digital products (X19): any product, or a product's id. */
  bought?: string
  /** Store customers and what they spent (X20). */
  storeCustomer?: string
  spentFrom?: string
  spentTo?: string
  /** The phone and SMS consent (X21). */
  phone?: 'yes' | 'no'
  phoneCountry?: string
  smsConsent?: string
  /** A linked push subscription, and WhatsApp consent on a list (X22). */
  pushSubscribed?: boolean
  whatsAppConsent?: boolean
  page?: number
  size?: number
  /** The page's keyset cursor (review 2026-11 AU-1), and the columns shown (AU-9). */
  cursor?: string
  cols?: string
  /** A saved view's id: opens with its filters and columns (review 2026-11 AU-19). */
  view?: string
}

type Item = Schemas['ContactListItem']
type Filter = Schemas['ContactFilter']

function sameFilter(a: Filter, b: Filter) {
  return filterKeys.every((k) => (a[k as keyof Filter] ?? '') === (b[k as keyof Filter] ?? ''))
}

/** "Tue 7 Oct" within the last half year, "14 Aug" before, "3 Jun 2025" for older years. */
function lastOpen(iso: string | undefined, machine: boolean) {
  if (!iso) return t(m.never)
  const d = new Date(iso)
  const days = (Date.now() - d.getTime()) / 86_400_000
  const s = days < 14 ? shortDate(d) : days < 330 ? dayMonthDate(d) : dayMonthYear(d)
  return machine ? t(m.machineOpen, { date: s }) : s
}

function joined(iso: string) {
  return dayMonthYear(new Date(iso))
}

const features = tableFeatures({ rowSelectionFeature })
const col = createColumnHelper<typeof features, Item>()

/** 7.9 Contacts: saved views, filters, bulk actions and the table. */
export function ContactsPage({ search }: { search: ContactsSearch }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const settled = useFirstLoadSettled()
  const filter = filterOf(search)
  const page = search.page ?? 1
  const size = search.size ?? 10
  const sort: ContactSort = search.sort ?? 'activity'
  const summary = useAudienceSummary()
  const phones = usePhoneColumn()
  const paidOn = (summary.data?.paidTiers.length ?? 0) > 0
  const columns = parseColumns(search.cols) ?? defaultColumns(paidOn, phones)
  const fields = fieldKeys(columns)
  const views = useQuery(contactViewsQuery(slug))
  // "Count all" asks for the exact count of this filter only (review D5).
  const filterKey = JSON.stringify(filter)
  const [exactFor, setExactFor] = useState<string>()
  const exact = exactFor === filterKey
  const list = useQuery(contactListQuery(slug, search, fields, exact))
  // A saved view's own address (review 2026-11 AU-19): open it with its
  // filters and columns, or say it's gone.
  const viewId = search.view
  const savedViews = views.data
  useEffect(() => {
    if (!viewId || !savedViews) return
    const v = savedViews.items.find((x) => x.id === viewId)
    if (!v) toast({ kind: 'error', title: tr(lfm.viewMissing) })
    void navigate({
      to: '/w/$workspace/audience',
      params: { workspace: slug },
      search: v ? { ...(v.filters as ContactsSearch), cols: v.columns.length ? v.columns.join(',') : undefined, size: search.size } : { ...search, view: undefined },
      replace: true,
    })
  }, [viewId, savedViews]) // eslint-disable-line react-hooks/exhaustive-deps
  // The selection belongs to one page of one filter: another page or filter
  // starts with nothing selected; "all matching" belongs to the filter.
  const key = JSON.stringify({ filter, page, size, cursor: search.cursor })
  const [picked, setPicked] = useState<{ key: string; rows: RowSelectionState }>({ key, rows: {} })
  const selection = picked.key === key ? picked.rows : {}
  const setSelection = (rows: RowSelectionState) => {
    setPicked({ key, rows })
    setAllFor(undefined)
  }
  const [allFor, setAllFor] = useState<string>()
  const all = allFor === filterKey
  const [adding, setAdding] = useState(false)
  const [saving, setSaving] = useState(false)
  const [bulk, setBulk] = useState<BulkKind | null>(null)
  const [erasing, setErasing] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [result, setResult] = useState<Schemas['ContactBulkResult'] | null>(null)
  const addReason = usePermissionReason('subscribers:create')
  const importReason = usePermissionReason('subscribers:import')
  const exportReason = usePermissionReason('subscribers:export')

  const go = (next: ContactsSearch) => void navigate({ to: '/w/$workspace/audience', params: { workspace: slug }, search: next })
  const setFilter = (patch: Partial<ContactsSearch>) => go({ ...search, ...patch, page: undefined, cursor: undefined })

  const empty = summary.data && summary.data.contacts === 0
  const ids = Object.keys(selection)
  const data = list.data
  const total = data?.total ?? 0
  const sel: BulkSelection = all ? { allMatching: true, filter } : { ids }
  const n = all ? total : ids.length
  const pageFull = data !== undefined && ids.length > 0 && ids.length === data.items.length
  const more = data !== undefined && (data.totalCapped || total > data.items.length)
  const stale = list.error instanceof ApiError && list.error.status === 422 && Boolean(search.cursor)
  return (
    <Page
      title={tr(am.title)}
      filters={<AudienceTabs />}
      actions={
        <>
          {importReason ? (
            <Button disabledReason={importReason}>{tr(m.import)}</Button>
          ) : (
            <ButtonLink to="/w/$workspace/import" params={{ workspace: slug }}>
              {tr(m.import)}
            </ButtonLink>
          )}
          <Button variant="primary" disabledReason={addReason} onClick={() => setAdding(true)}>
            {tr(m.addContact)}
          </Button>
        </>
      }
      contentClassName="py-5 gap-3.5"
    >
      {empty ? (
        <AudienceEmpty lists={summary.data?.lists ?? []} />
      ) : (
        <>
          {/* The filters grow a row as each feature's data arrives: until the
              first data is in, they stay mounted but hidden behind the table's
              skeleton, and then show at once, so nothing below jumps. */}
          {settled ? null : <ContactsSkeleton total={summary.data?.contacts} />}
          <div className={settled ? 'contents' : 'hidden'}>
            <ViewChips
              views={views.data}
              active={views.data?.items.find((v) => Object.keys(filter).length > 0 && sameFilter(filter, v.filters) && (v.columns.join(',') || undefined) === search.cols)?.id}
              filter={filter}
              columns={search.cols}
              onPick={(f, cols) => go({ ...(f as ContactsSearch), cols, size: search.size })}
              onSave={() => setSaving(true)}
            />
            {/* The website's members (features/website), while it's on. */}
            <MembersChips
              signedIn={Boolean(search.signedIn)}
              onSignedIn={(on) => setFilter({ signedIn: on || undefined })}
              paid={search.paid === 'with_access' && !search.paidTier}
              onPaid={(on) => setFilter({ paid: on ? 'with_access' : undefined, paidTier: undefined })}
            />
            {/* Push subscribed and WhatsApp consent (X22). */}
            <ChannelChips
              push={Boolean(search.pushSubscribed)}
              onPush={(on) => setFilter({ pushSubscribed: on || undefined })}
              whatsapp={Boolean(search.whatsAppConsent)}
              onWhatsApp={(on) => setFilter({ whatsAppConsent: on || undefined })}
            />
            <Filters
              search={search}
              total={data?.total}
              capped={data?.totalCapped ?? false}
              lists={summary.data?.lists ?? []}
              paidTiers={summary.data?.paidTiers ?? []}
              onChange={setFilter}
              tools={
                <>
                  <ColumnChooser columns={columns} onChange={(c) => go({ ...search, cols: c.join(',') })} onReset={() => go({ ...search, cols: undefined })} />
                  {Object.keys(filter).length > 0 ? (
                    <Button disabledReason={exportReason} onClick={() => void startExport(slug, { allMatching: true, filter }, total)}>
                      {tr(m.exportView)}
                    </Button>
                  ) : null}
                </>
              }
            />
            <ExportsReady />
            {result ? <BulkResultPanel result={result} onDismiss={() => setResult(null)} /> : null}
            {ids.length > 0 || all ? (
              <BulkBar
                n={n}
                all={all}
                counting={all && (!exact || list.isFetching)}
                onAction={(k) => (k === 'erase' ? setErasing(true) : k === 'verify' ? setVerifying(true) : setBulk(k))}
                onExport={() => void (all ? startExport(slug, sel, n) : exportCSV(slug, { ids }, ids.length))}
                onClear={() => setSelection({})}
              />
            ) : null}
            {pageFull && more ? (
              <div className="banner" role="status">
                {all ? tr(m.allMatchingSelected, { count: countLine(total, data.totalCapped) }) : tr(m.pageSelected, { n: ids.length, count: count(ids.length) })}{' '}
                <button
                  type="button"
                  className="font-semibold text-accent underline underline-offset-2"
                  onClick={() => {
                    setAllFor(all ? undefined : filterKey)
                    // Acting on everything needs the exact number (the erase types it).
                    if (!all) setExactFor(filterKey)
                  }}
                >
                  {all ? tr(m.selectPageOnly) : tr(m.selectAllMatching, { count: countLine(total, data.totalCapped) })}
                </button>
              </div>
            ) : null}
            {stale ? (
              <Note tone="warn">
                {tr(m.staleCursor)}{' '}
                <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={() => go({ ...search, cursor: undefined, page: undefined })}>
                  {tr(m.firstPage)}
                </button>
              </Note>
            ) : list.isError ? (
              <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{tr(common.retry)}</Button>}>
                {errorMessage(list.error)} {tr(m.nothingChanged)}
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
                columns={columns}
                filtered={Object.keys(filter).length > 0}
                sort={sort}
                counting={exact && list.isFetching}
                onCountAll={() => setExactFor(filterKey)}
                onSort={(s) => go({ ...search, sort: s === 'activity' ? undefined : s, page: undefined, cursor: undefined })}
                onPage={(next) =>
                  next === 'next'
                    ? go({ ...search, cursor: list.data.nextCursor, page: page + 1 })
                    : page <= 2
                      ? go({ ...search, cursor: undefined, page: undefined })
                      : go({ ...search, cursor: list.data.prevCursor, page: page - 1 })
                }
                onSize={(s) => go({ ...search, size: s === 10 ? undefined : s, page: undefined, cursor: undefined })}
                onClear={() => go({ size: search.size })}
              />
            )}
          </div>
        </>
      )}
      {adding ? <AddContactDialog onClose={() => setAdding(false)} lists={summary.data?.lists ?? []} /> : null}
      {saving ? <SaveViewDialog onClose={() => setSaving(false)} filter={filter} columns={parseColumns(search.cols)} /> : null}
      {bulk ? (
        <BulkDialog
          kind={bulk}
          onClose={() => setBulk(null)}
          selection={sel}
          n={n}
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
          selection={sel}
          n={n}
          stripe={summary.data?.stripe ?? false}
          onDone={() => {
            setSelection({})
            setErasing(false)
          }}
        />
      ) : null}
      {verifying ? (
        <VerifyDialog
          open
          onOpenChange={(o) => !o && setVerifying(false)}
          target={all ? { kind: 'selection', filter } : { kind: 'selection', contactIds: ids }}
          origin="contacts"
          from={tr(m.verifyFrom, { n, count: count(n) })}
          onStarted={() => {
            setVerifying(false)
            setSelection({})
          }}
        />
      ) : null}
    </Page>
  )
}

/** "48,213", or "10,000+" past the count's cap. */
function countLine(total: number, capped: boolean | undefined) {
  return capped ? t(m.totalCapped, { count: count(total) }) : count(total)
}

/** Starts a background export (AU-4); the jobs tray follows it and offers the file. */
export async function startExport(slug: string, selection: BulkSelection, n: number) {
  try {
    const { data } = await http.POST('/api/v1/workspaces/{workspace}/contacts/jobs/export', { params: { path: { workspace: slug } }, body: selection })
    if (data) trackJob(slug, { id: data.id, kind: 'export', title: t(m.exporting, { n, count: count(n) }), status: 'queued', startedAt: new Date().toISOString() })
    toast({ kind: 'success', title: t(m.exportStarted), detail: t(m.exportStartedBody) })
  } catch (e) {
    toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
  }
}

/** Finished exports this member started, with their download (from the jobs list). */
function ExportsReady() {
  const tr = useT()
  const { slug } = useWorkspace()
  const { jobs } = useBackgroundJobs(slug)
  const ready = jobs.filter((j) => j.kind === 'export' && j.status === 'done' && j.downloadUrl)
  if (!ready.length) return null
  return (
    <>
      {ready.map((j) => (
        <Note key={j.id} tone="info">
          <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <b>{j.detail ?? j.title}</b>
            <a className="font-semibold text-accent underline underline-offset-2" href={j.downloadUrl} download>
              {tr(m.download)}
            </a>
          </span>
        </Note>
      ))}
    </>
  )
}

/** Exports the selection as CSV; the jobs tray shows it while it runs. */
export async function exportCSV(slug: string, body: Schemas['ContactExportInput'], n: number) {
  const id = `export-${Date.now()}`
  const what = { n, count: count(n) }
  trackJob(slug, { id, kind: 'export', title: t(m.exporting, what), status: 'running', startedAt: new Date().toISOString() })
  try {
    const { data, response } = await http.POST('/api/v1/workspaces/{workspace}/contacts/export', { params: { path: { workspace: slug } }, body, parseAs: 'blob' })
    if (data) saveBlob(data as unknown as Blob, fileName(response, 'contacts.csv'))
    updateJob(slug, id, { status: 'done', detail: t(m.exportedDetail, what), finishedAt: new Date().toISOString() })
    toast({ kind: 'success', title: t(m.exported, what), detail: t(m.exportInAudit) })
  } catch (e) {
    updateJob(slug, id, { status: 'failed', detail: errorMessage(e), finishedAt: new Date().toISOString() })
    toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
  }
}

function Filters({
  search,
  total,
  capped,
  lists,
  paidTiers,
  onChange,
  tools,
}: {
  search: ContactsSearch
  total?: number
  /** More than 10,000 match: the line reads "10,000+" (review D5). */
  capped: boolean
  lists: Schemas['AudienceListSummary'][]
  paidTiers: Schemas['AudiencePaidTier'][]
  onChange: (patch: Partial<ContactsSearch>) => void
  /** The columns and export buttons, at the row's end. */
  tools?: ReactNode
}) {
  const tr = useT()
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
    [search.source, search.tag, search.joinedAfter || search.joinedBefore, search.segment, search.import].filter(Boolean).length +
    scoreGrowthMoreKeys.filter((k) => search[k] !== undefined).length +
    storeMoreKeys.filter((k) => search[k] !== undefined).length
  const totalLine = useTotalLine()
  const phone = 'max-md:w-0 max-md:min-w-full'
  const sel = `in ${phone}`
  return (
    <div className="card flex flex-wrap items-center gap-2 px-3 py-2.5">
      <label className="vh" htmlFor="contacts-q">
        {tr(m.search)}
      </label>
      <input
        id="contacts-q"
        type="search"
        className="in w-[260px] max-md:min-w-full"
        placeholder={tr(m.searchPlaceholder)}
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <select className={sel} aria-label={tr(m.list)} value={search.list ?? ''} onChange={(e) => onChange({ list: e.target.value || undefined })}>
        <option value="">{tr(m.anyList)}</option>
        {lists.map((l) => (
          <option key={l.id} value={l.id}>
            {l.name}
          </option>
        ))}
      </select>
      <select className={sel} aria-label={tr(m.tier)} value={search.tier ?? ''} onChange={(e) => onChange({ tier: (e.target.value || undefined) as Tier | undefined })}>
        <option value="">{tr(m.anyTier)}</option>
        {tiers.map((t) => (
          <option key={t} value={t}>
            {tierLabel[t]}
          </option>
        ))}
      </select>
      <BandSelect className={sel} value={search.band} onChange={(band) => onChange({ band })} />
      <select className={sel} aria-label={tr(m.mailbox)} value={search.mailbox ?? ''} onChange={(e) => onChange({ mailbox: (e.target.value || undefined) as Mailbox | undefined })}>
        <option value="">{tr(m.anyMailbox)}</option>
        {mailboxes.map((m) => (
          <option key={m} value={m}>
            {mailboxLabel[m]}
          </option>
        ))}
      </select>
      <select
        className={sel}
        aria-label={tr(m.consent)}
        value={search.consent ?? ''}
        onChange={(e) => onChange({ consent: (e.target.value || undefined) as Consent | undefined })}
      >
        <option value="">{tr(m.anyConsent)}</option>
        {(['double', 'single', 'pending'] as Consent[]).map((c) => (
          <option key={c} value={c}>
            {consentFilterLabel[c]}
          </option>
        ))}
      </select>
      <LanguageFilter className={sel} value={search.language} onChange={(language) => onChange({ language })} />
      <AddressCheckFilter className={sel} value={search.check} onChange={(check) => onChange({ check })} />
      <PaidStatusSelect className={sel} tiers={paidTiers} paid={search.paid} tier={search.paidTier} onChange={(f) => onChange(f as Partial<ContactsSearch>)} />
      {/* Bought a product (X19), with products:view. */}
      <BoughtFilter className={sel} value={search.bought} onChange={(bought) => onChange({ bought })} />
      <SmsContactFilters className={sel} search={search} onChange={onChange} />
      <MoreFilters search={search} n={more} onChange={onChange} />
      <span className="k ml-auto" aria-live="polite">
        {total === undefined ? '' : totalLine(total, search.sort ?? 'activity', capped)}
      </span>
      {tools}
      <ScoreGrowthChips search={search} onChange={onChange} />
      <StoreChips search={search} onChange={onChange} />
    </div>
  )
}

function MoreFilters({ search, n, onChange }: { search: ContactsSearch; n: number; onChange: (patch: Partial<ContactsSearch>) => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } }, { enabled: open })
  const [tag, setTag] = useState(search.tag ?? '')
  const [after, setAfter] = useState(search.joinedAfter ?? '')
  const [before, setBefore] = useState(search.joinedBefore ?? '')
  const [imp, setImp] = useState(search.import ?? '')
  const [segment, setSegment] = useState(search.segment ?? '')
  const imports = $api.useQuery('get', '/api/v1/workspaces/{workspace}/imports', { params: { path: { workspace: slug } } }, { enabled: open })
  const { me } = useWorkspace()
  const scoreGrowth = (): ScoreGrowthDraft => Object.fromEntries(scoreGrowthMoreKeys.map((k) => [k, search[k]]))
  const [extra, setExtra] = useState<ScoreGrowthDraft>(scoreGrowth)
  const storeDraft = (): StoreDraft => ({ storeCustomer: search.storeCustomer, spentFrom: search.spentFrom, spentTo: search.spentTo })
  const [stores, setStores] = useState<StoreDraft>(storeDraft)
  const toggle = (o: boolean) => {
    if (o) {
      // Opening shows the filters that apply now.
      setTag(search.tag ?? '')
      setAfter(search.joinedAfter ?? '')
      setBefore(search.joinedBefore ?? '')
      setImp(search.import ?? '')
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
      label={tr(m.moreFilters)}
      // With the store fields (X20) it outgrows a laptop screen: it scrolls within what's left of it.
      className="max-h-(--radix-popover-content-available-height) w-[360px] gap-3 overflow-y-auto p-3.5"
      trigger={<button className="btn">{n ? tr(m.moreFiltersCount, { n }) : tr(m.moreFilters)}</button>}
    >
      {/* Source: how each contact first joined, from this workspace's own sources (review 2026-11 EN-7). */}
      <GrowthSourceField value={extra.growthSource} onChange={(growthSource) => setExtra({ ...extra, growthSource })} open={open} />
      <Field label={tr(m.tag)}>
        <Input value={tag} onChange={(e) => setTag(e.target.value)} placeholder={'customer'} />
      </Field>
      <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
        <legend className="mb-1.5 text-xs font-semibold text-ink-2">{tr(m.joinedBetween)}</legend>
        <div className="flex items-center gap-2">
          <Input type="date" aria-label={tr(m.joinedFrom)} value={after} max={before || undefined} onChange={(e) => setAfter(e.target.value)} />
          <span className="k">{tr(m.and)}</span>
          <Input type="date" aria-label={tr(m.joinedUntil)} value={before} min={after || undefined} onChange={(e) => setBefore(e.target.value)} />
        </div>
        <span className="k">{tr(m.joinedHint, { tz: me.timezone })}</span>
      </fieldset>
      <Field label={tr(m.fromImport)}>
        <Select value={imp} onChange={(e) => setImp(e.target.value)}>
          <option value="">{tr(m.anyImport)}</option>
          {/* An import in the address bar shows before the list has loaded. */}
          {imp && !imports.data?.items.some((i) => i.id === imp) ? <option value={imp}>{imp.slice(0, 8)}</option> : null}
          {imports.data?.items
            .filter((i) => i.status === 'done')
            .map((i) => (
              <option key={i.id} value={i.id}>
                {tr(m.importOption, { source: i.fileName || i.sourceName, day: i.finishedAt ? dayMonthDate(new Date(i.finishedAt)) : '' })}
              </option>
            ))}
        </Select>
      </Field>
      <Field label={tr(m.segment)}>
        <Select value={segment} onChange={(e) => setSegment(e.target.value)}>
          <option value="">{tr(m.anySegment)}</option>
          {segments.data?.items.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
      </Field>
      <ScoreGrowthFields draft={extra} onChange={setExtra} />
      <StoreContactFields draft={stores} onChange={setStores} open={open} />
      <div className="flex justify-end gap-2">
        <Button
          size="sm"
          onClick={() => {
            onChange({
              source: undefined,
              tag: undefined,
              joinedAfter: undefined,
              joinedBefore: undefined,
              import: undefined,
              segment: undefined,
              ...Object.fromEntries([...scoreGrowthMoreKeys, ...storeMoreKeys].map((k) => [k, undefined])),
            })
            setOpen(false)
          }}
        >
          {tr(m.clear)}
        </Button>
        <Button
          size="sm"
          variant="primary"
          onClick={() => {
            onChange({
              tag: tag.trim() || undefined,
              joinedAfter: after || undefined,
              joinedBefore: before || undefined,
              import: imp || undefined,
              segment: segment || undefined,
              ...extra,
              ...stores,
            })
            setOpen(false)
          }}
        >
          {tr(m.apply)}
        </Button>
      </div>
    </Popover>
  )
}

function BulkBar({
  n,
  all,
  counting,
  onAction,
  onExport,
  onClear,
}: {
  n: number
  /** Everything matching the filter is selected (AU-5). */
  all: boolean
  /** The exact number of everything matching is on its way: actions wait for it. */
  counting: boolean
  onAction: (k: BulkKind | 'erase' | 'verify') => void
  onExport: () => void
  onClear: () => void
}) {
  const tr = useT()
  const editReason = usePermissionReason('subscribers:edit')
  const exportReason = usePermissionReason('subscribers:export')
  const deleteReason = usePermissionReason('subscribers:delete')
  const verifyReason = usePermissionReason('subscribers:verify')
  const canExport = useCan('subscribers:export')
  const wait = counting ? tr(m.counting) : undefined
  return (
    <div className="card flex flex-wrap items-center gap-2.5 border-accent-line bg-accent-tint px-3 py-2" role="region" aria-label={tr(m.bulkActions)}>
      <span className="inline-flex items-center gap-1.5 text-[13px] font-semibold" role="status">
        {counting ? <span className="spin sm" aria-hidden="true" /> : null}
        {counting ? tr(m.counting) : all ? tr(m.allMatching, { count: count(n) }) : tr(m.selected, { count: count(n) })}
      </span>
      <Button size="sm" disabledReason={editReason ?? wait} onClick={() => onAction('add_to_list')}>
        {tr(m.addToList)}
      </Button>
      <Button size="sm" disabledReason={editReason ?? wait} onClick={() => onAction('remove_from_list')}>
        {tr(m.removeFromList)}
      </Button>
      <Button size="sm" disabledReason={editReason ?? wait} onClick={() => onAction('add_tag')}>
        {tr(m.addTag)}
      </Button>
      <Button size="sm" disabledReason={editReason ?? wait} onClick={() => onAction('remove_tag')}>
        {tr(m.removeTag)}
      </Button>
      <Button size="sm" disabledReason={editReason ?? wait} onClick={() => onAction('resend_optin')}>
        {tr(m.resendOptin)}
      </Button>
      <Button size="sm" disabledReason={verifyReason ?? wait} onClick={() => onAction('verify')}>
        {tr(m.verifyAddresses)}
      </Button>
      <Button size="sm" disabledReason={exportReason ?? wait} title={tr(m.exportTitle)} onClick={onExport}>
        {tr(m.exportCsv)}
      </Button>
      <span className="k">{canExport ? tr(m.exportLogged) : tr(m.exportNeeds)}</span>
      <Button size="sm" className="dline" disabledReason={editReason ?? wait} onClick={() => onAction('suppress')}>
        {tr(m.suppress)}
      </Button>
      <Button size="sm" className="dline" disabledReason={deleteReason ?? wait} onClick={() => onAction('erase')}>
        {tr(m.delete)}
      </Button>
      <Button size="sm" className="ml-auto" onClick={onClear}>
        {tr(m.clearSelection)}
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
  columns,
  filtered,
  onPage,
  onSize,
  onClear,
  sort,
  onSort,
  counting,
  onCountAll,
}: {
  data: Schemas['ContactList']
  selection: RowSelectionState
  onSelection: (s: RowSelectionState) => void
  page: number
  size: number
  /** The columns after Contact, in order (AU-9). */
  columns: ColumnId[]
  filtered: boolean
  onPage: (dir: 'next' | 'prev') => void
  onSize: (s: number) => void
  onClear: () => void
  sort: ContactSort
  onSort: (s: ContactSort) => void
  /** The exact count is being counted (review D5). */
  counting: boolean
  onCountAll: () => void
}) {
  const { slug } = useWorkspace()
  const fields = $api.useQuery('get', '/api/v1/workspaces/{workspace}/fields', { params: { path: { workspace: slug } } }, { enabled: columns.some((c) => c.startsWith('field:')) })
  const fieldLabel = (key: string) => fields.data?.items.find((f) => f.key === key)?.label || key
  const tableColumns = useMemo(
    () =>
      col.columns([
        col.display({
          id: 'select',
          header: ({ table }) => (
            <SelectAll
              label={t(m.selectAll)}
              all={table.getIsAllPageRowsSelected()}
              some={table.getIsSomePageRowsSelected()}
              onChange={(on) => table.toggleAllPageRowsSelected(on)}
            />
          ),
          cell: ({ row }) => (
            <Check
              aria-label={t(m.selectRow, { name: row.original.name || row.original.email })}
              checked={row.getIsSelected()}
              onChange={row.getToggleSelectedHandler()}
            />
          ),
        }),
      ]),
    [],
  )
  const table = useTable(
    {
      features,
      columns: tableColumns,
      data: data.items,
      getRowId: (r) => r.id,
      state: { rowSelection: selection },
      onRowSelectionChange: (u) => onSelection(typeof u === 'function' ? u(selection) : u),
    },
    (s) => ({ rowSelection: s.rowSelection }),
  )
  const header = table.getHeaderGroups()[0]!.headers[0]!
  const tr = useT()
  const from = data.items.length === 0 ? 0 : (page - 1) * size + 1
  const to = from === 0 ? 0 : from + data.items.length - 1
  const head = (c: ColumnId) => {
    switch (c) {
      case 'score':
        return <SortHeader key={c} label={tr(m.colScore)} sort="score" current={sort} onSort={onSort} />
      case 'last_open':
        return <SortHeader key={c} label={tr(m.colLastOpen)} sort="last_open" current={sort} onSort={onSort} />
      case 'joined':
        return <SortHeader key={c} label={tr(m.colJoined)} sort="joined" current={sort} onSort={onSort} />
      case 'phone':
        return <Th key={c}>{tr(cfm.column)}</Th>
    }
    if (c.startsWith('field:')) return <Th key={c}>{fieldLabel(c.slice('field:'.length))}</Th>
    return <Th key={c}>{tr(columnLabels[c as BuiltinColumn])}</Th>
  }
  const cell = (c: ColumnId, row: Item) => {
    const tier = row.tier as Tier
    const consent = row.consent as Consent
    switch (c) {
      case 'paid':
        return (
          <td key={c} className="nw">
            <PaidStatusCell paid={row.paid} />
          </td>
        )
      case 'phone':
        return (
          <td key={c} className="mono nw">
            {row.phone || <span className="k">–</span>}
          </td>
        )
      case 'lists':
        return (
          <td key={c} className="lists">
            {row.lists.join(', ') || '–'}
          </td>
        )
      case 'tier':
        return (
          <td key={c}>
            <span className={`badge ${tierClass[tier]}`}>{tierLabel[tier]}</span>
          </td>
        )
      case 'score':
        return (
          <td key={c} className="nw">
            <ScoreCell score={row.score} band={row.scoreBand} />
          </td>
        )
      case 'consent':
        return (
          <td key={c}>
            {row.suppressed ? <Badge tone="bad">{tr(m.suppressed)}</Badge> : <Badge tone={consentTone[consent]}>{consentLabel[consent]}</Badge>}
          </td>
        )
      case 'mailbox':
        // Mailbox provider, last open and source wrap so the table fits its card (Contacts board).
        return (
          <td key={c} className="wrap">
            <span className="flex items-center gap-1.5">
              <span className="dot" style={{ color: mailboxColor[row.mailboxProvider] ?? mailboxColor.other }} aria-hidden="true" />
              {row.mailboxLabel}
            </span>
          </td>
        )
      case 'last_open':
        return (
          <td key={c} className="wrap">
            {lastOpen(row.lastOpenAt, row.lastOpenMachine)}
          </td>
        )
      case 'joined':
        return <td key={c}>{joined(row.joinedAt)}</td>
      case 'source':
        return (
          <td key={c} className="wrap">
            {row.sourceLabel}
          </td>
        )
    }
    const v = row.fields?.[c.slice('field:'.length)]
    return (
      <td key={c} className="wrap">
        {v === undefined || v === null || v === '' ? <span className="k">–</span> : String(v)}
      </td>
    )
  }
  const showPaidChip = columns.includes('paid')
  return (
    <section className="card" aria-label={tr(m.contacts)}>
      <TableScroll label={tr(m.contacts)}>
        <table className="tbl ctbl phone-wrap">
          <thead>
            <tr>
              <th scope="col">
                <table.FlexRender header={header} />
              </th>
              <SortHeader label={tr(m.colContact)} sort="email" current={sort} onSort={onSort} />
              {columns.map(head)}
            </tr>
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => {
              const c = row.original
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
                    {showPaidChip ? <PaidTierChip paid={c.paid} /> : null}
                    <div className="k mono">{c.email}</div>
                  </td>
                  {columns.map((k) => cell(k, c))}
                </tr>
              )
            })}
            {data.items.length === 0 ? (
              <tr>
                <td colSpan={2 + columns.length} className="py-8 text-center text-ink-3">
                  {filtered ? (
                    <>
                      {tr(m.noMatch)}{' '}
                      <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onClear}>
                        {tr(m.clearFilters)}
                      </button>
                    </>
                  ) : (
                    tr(m.noneOnPage)
                  )}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4 py-2.5">
        <span className="k inline-flex items-center gap-1.5" aria-live="polite">
          {counting ? <span className="spin" aria-hidden="true" /> : null}
          {tr(m.range, { from: count(from), to: count(to), total: data.totalCapped ? tr(m.totalCapped, { count: count(data.total) }) : count(data.total) })}
          {counting ? (
            <> · {tr(m.counting)}</>
          ) : data.totalCapped ? (
            <>
              {' · '}
              <button type="button" className="font-semibold text-accent underline underline-offset-2" onClick={onCountAll}>
                {tr(m.countAll)}
              </button>
            </>
          ) : null}
        </span>
        <div className="flex items-center gap-1.5 max-md:flex-wrap">
          <select className="in h-[30px] max-md:w-0 max-md:min-w-full" aria-label={tr(m.rowsPerPage)} value={size} onChange={(e) => onSize(Number(e.target.value))}>
            {[10, 50, 100].map((n) => (
              <option key={n} value={n}>
                {tr(m.perPage, { n })}
              </option>
            ))}
          </select>
          <Button size="sm" disabled={page <= 1 && !data.prevCursor} onClick={() => onPage('prev')}>
            {tr(m.previous)}
          </Button>
          <Button size="sm" disabled={!data.nextCursor} onClick={() => onPage('next')}>
            {tr(m.next)}
          </Button>
        </div>
      </div>
    </section>
  )
}

/** Loading: rows shaped like the table (AudienceStates › Loading). */
export function ContactsSkeleton({ total }: { total?: number }) {
  const tr = useT()
  const widths = [
    [140, 200],
    [120, 180],
    [160, 220],
    [130, 190],
  ]
  return (
    <Loading label={total ? tr(m.loadingCount, { count: count(total) }) : tr(m.loadingContacts)} className="card flex flex-col">
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
  const tr = useT()
  const { slug } = useWorkspace()
  const sync = $api.useQuery('get', '/api/v1/workspaces/{workspace}/suppressions', { params: { path: { workspace: slug }, query: { pageSize: 1 } } })
  const pulled = sync.data?.sync.filter((p) => p.pulled > 0) ?? []
  const names = lists.map((l) => l.name)
  const listText = names.length ? `${tr(m.emptyListsReady, { n: names.length, lists: andList(names) })} ` : ''
  return (
    <section className="card flex flex-col items-center gap-3 border-dashed px-7 py-10 text-center">
      <svg width="56" height="56" viewBox="0 0 56 56" aria-hidden="true">
        <circle cx="22" cy="20" r="8" fill="none" stroke="var(--color-muted-line)" strokeWidth="2" />
        <path d="M8 44a14 14 0 0 1 28 0M38 14a7 7 0 0 1 0 12M50 44a12 12 0 0 0-9-11.5" fill="none" stroke="var(--color-muted-line)" strokeWidth="2" strokeLinecap="round" />
      </svg>
      <h2 className="m-0 text-[18px] font-bold">{tr(m.emptyTitle)}</h2>
      <span className="max-w-[520px] text-ink-2">
        {listText}
        {tr(m.emptyBody)}
      </span>
      <div className="mt-1.5 flex flex-wrap justify-center gap-2.5">
        <ButtonLink variant="primary" className="min-h-10" to="/w/$workspace/import" params={{ workspace: slug }}>
          {tr(m.importContacts)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/forms" params={{ workspace: slug }}>
          {tr(m.createForm)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          {tr(m.apiDocs)}
        </ButtonLink>
      </div>
      {pulled.length ? (
        <span className="k">
          {tr(m.suppressionsInPlace, { pulled: pulled.map((p) => tr(m.pulledFrom, { count: count(p.pulled), provider: shortProvider(p.name) })).join(', ') })}
        </span>
      ) : null}
    </section>
  )
}

/** "SES" for "Amazon SES". */
export function shortProvider(name: string) {
  return name.replace(/^Amazon /, '')
}
