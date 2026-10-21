import { keepPreviousData, useInfiniteQuery } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, copyText } from '@/features/access/parts'
import { m as ma } from '@/features/audit/messages'
import { zonedDayStart } from '@/features/audit/zoned'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, instanceAuditTime, monthDayTimeSeconds } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { AgencyAuditFilter } from '@/features/agency/AdminParts'
import { RetentionDialog, StreamDialog } from './AuditSettingsDialogs'
import { m as mi } from './messages'

type Item = Schemas['InstanceAuditItem']
type List = Schemas['InstanceAuditList']

export interface InstanceAuditSearch {
  q?: string
  /** A workspace slug, or "instance" for instance-level entries only. */
  workspace?: string
  actorType?: 'user' | 'api_key' | 'mcp_agent' | 'system'
  result?: 'allowed' | 'denied'
  /** The agency's actions only (agency accounts). */
  category?: 'agency'
  /** Last 30 days when unset. */
  range?: '7d' | '90d' | 'custom'
  since?: string
  until?: string
  /** The entry shown in the details dialog. */
  event?: string
}

const PAGE = 25

/** The API filters for the current search (custom ranges are days in the admin's timezone). */
function apiFilters(s: InstanceAuditSearch, tz: string) {
  const custom = s.range === 'custom'
  return {
    q: s.q || undefined,
    workspace: s.workspace,
    actorType: s.actorType,
    result: s.result,
    category: s.category,
    range: s.range === 'custom' ? ('all' as const) : (s.range ?? ('30d' as const)),
    since: custom && s.since ? zonedDayStart(s.since, tz).toISOString() : undefined,
    until: custom && s.until ? zonedDayStart(s.until, tz, 1).toISOString() : undefined,
  }
}

/** Instance admin › Audit log (AuditInstance): sign-ins, SSO, policies, users and every workspace's events. */
export function InstanceAuditPage({ search }: { search: InstanceAuditSearch }) {
  const t = useT()
  return (
    <AdminPage documentTitle={t(mi.documentTitle)} back="audit">
      <InstanceAudit search={search} />
    </AdminPage>
  )
}

function InstanceAudit({ search }: { search: InstanceAuditSearch }) {
  const t = useT()
  const { me } = useWorkspace()
  const filters = apiFilters(search, me.timezone)
  const q = useInfiniteQuery({
    queryKey: ['admin-audit', filters],
    queryFn: async ({ pageParam }) => (await http.GET('/api/v1/admin/audit', { params: { query: { ...filters, cursor: pageParam || undefined, limit: PAGE } } })).data!,
    initialPageParam: '',
    getNextPageParam: (last: List) => last.nextCursor || undefined,
    placeholderData: keepPreviousData,
  })
  const settings = $api.useQuery('get', '/api/v1/admin/audit/settings')
  const [dialog, setDialog] = useState<'retention' | 'stream'>()
  const first = q.data?.pages[0]
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  return (
    <>
      <TopBar list={first} loading={q.isPending} filters={filters} onRetention={() => setDialog('retention')} />
      <section className="card" aria-labelledby="iev-h">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2" id="iev-h">
              {t(ma.eventsTitle)}
            </h2>
            <span className="k">{t(mi.eventsCaption)}</span>
          </div>
          <Filters search={search} list={first} />
        </div>
        {q.isError ? (
          <div className="pad">
            <ErrorPanel title={t(ma.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
              {errorMessage(q.error)}
            </ErrorPanel>
          </div>
        ) : !q.data ? (
          <ListSkeleton />
        ) : (
          <>
            <AccessTable caption={t(mi.tableCaption)} aria-busy={q.isFetching && !q.isFetchingNextPage ? true : undefined}>
              <thead>
                <tr>
                  <Th>{t(ma.colTime)}</Th>
                  <Th>{t(ma.colActor)}</Th>
                  <Th>{t(ma.colEvent)}</Th>
                  <Th>{t(ma.colResult)}</Th>
                  <Th>
                    <span className="vh">{t(ma.colDetails)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {items.length === 0 ? (
                  <EmptyRow colSpan={5}>
                    <span className="col items-center gap-2">
                      {t(mi.noMatch)}
                      <ClearFilters />
                    </span>
                  </EmptyRow>
                ) : (
                  items.map((it) => <EventRow key={it.id} it={it} search={search} />)
                )}
              </tbody>
            </AccessTable>
            {q.hasNextPage ? (
              <div className="row justify-center border-t border-line-2 px-4.5 py-3">
                <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
                  {t(ma.showOlder)}
                </Button>
              </div>
            ) : null}
          </>
        )}
      </section>
      <div className="grid gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fill,minmax(min(380px,100%),1fr))]">
        <RetentionCard settings={settings.data} error={settings.isError ? errorMessage(settings.error) : undefined} onRetention={() => setDialog('retention')} onStream={() => setDialog('stream')} />
      </div>
      {search.event ? <DetailDialog id={search.event} search={search} /> : null}
      {dialog === 'retention' ? <RetentionDialog onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'stream' ? <StreamDialog onClose={() => setDialog(undefined)} /> : null}
    </>
  )
}

function TopBar({ list, loading, filters, onRetention }: { list?: List; loading: boolean; filters: ReturnType<typeof apiFilters>; onRetention: () => void }) {
  const t = useT()
  const [busy, setBusy] = useState<'csv' | 'json'>()
  const [failed, setFailed] = useState<string>()
  const download = async (format: 'csv' | 'json') => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(filters)) if (v) qs.set(k, String(v))
    qs.set('format', format)
    const label = format.toUpperCase()
    setBusy(format)
    setFailed(undefined)
    const ctrl = new AbortController()
    const timer = window.setTimeout(() => ctrl.abort(), 60_000)
    try {
      const res = await fetch(`/api/v1/admin/audit/export?${qs.toString()}`, { credentials: 'same-origin', signal: ctrl.signal })
      if (!res.ok) {
        const p = (await res.json().catch(() => undefined)) as { detail?: string } | undefined
        throw new Error(p?.detail ?? `HTTP ${res.status}`)
      }
      const blob = await res.blob()
      const name = /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') ?? '')?.[1] ?? `audit-instance.${format}`
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = name
      a.click()
      window.setTimeout(() => URL.revokeObjectURL(a.href), 1000)
      toast({ kind: 'success', title: tt(mi.exportDownloaded, { format: label }), detail: tt(ma.loggedAs, { action: 'audit.exported' }) })
    } catch (e) {
      setFailed(
        e instanceof DOMException && e.name === 'AbortError'
          ? tt(mi.exportTimedOut, { format: label, seconds: count(60) })
          : tt(mi.exportError, { format: label, error: errorMessage(e) }),
      )
    } finally {
      window.clearTimeout(timer)
      setBusy(undefined)
    }
  }
  return (
    <>
      <div className="row justify-between">
        <div className="row gap-2">
          <Badge tone="purple">{t(mi.instanceLevel)}</Badge>
          <span className="k">{t(mi.instanceLevelCaption)}</span>
        </div>
        <div className="row gap-2">
          {list ? <span className="k">{t(mi.retained, { days: list.retentionDays })}</span> : loading ? <Skeleton w={160} h={12} /> : null}
          <Button size="sm" loading={busy === 'csv'} onClick={() => void download('csv')}>
            {t(ma.exportCsv)}
          </Button>
          <Button size="sm" loading={busy === 'json'} onClick={() => void download('json')}>
            {t(ma.exportJson)}
          </Button>
          <Button size="sm" onClick={onRetention}>
            {t(mi.retention)}
          </Button>
        </div>
      </div>
      {failed ? (
        <Note tone="bad" label={t(mi.exportFailed)} role="alert">
          {failed}
        </Note>
      ) : null}
    </>
  )
}

/** The date ranges; unset is the last 30 days. */
const RANGES: { value: '' | NonNullable<InstanceAuditSearch['range']>; range: string }[] = [
  { value: '', range: '30d' },
  { value: '7d', range: '7d' },
  { value: '90d', range: '90d' },
  { value: 'custom', range: 'custom' },
]

function useSetSearch(search: InstanceAuditSearch) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  return (patch: Partial<InstanceAuditSearch>) =>
    void navigate({ to: '/w/$workspace/admin/audit', params: { workspace: slug }, search: clean({ ...search, ...patch, event: undefined }), replace: true })
}

function clean(s: InstanceAuditSearch): InstanceAuditSearch {
  return Object.fromEntries(Object.entries(s).filter(([, v]) => v !== undefined && v !== '')) as InstanceAuditSearch
}

function ClearFilters() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <ButtonLink size="sm" to="/w/$workspace/admin/audit" params={{ workspace: slug }} search={{}}>
      {t(ma.clearFilters)}
    </ButtonLink>
  )
}

function Filters({ search, list }: { search: InstanceAuditSearch; list?: List }) {
  const t = useT()
  const set = useSetSearch(search)
  const [q, setQ] = useState(search.q ?? '')
  // Search as you type, a beat after the last key.
  useEffect(() => {
    if ((search.q ?? '') === q) return
    const h = window.setTimeout(() => set({ q: q.trim() || undefined }), 300)
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- `set` changes every render; the search value is what matters
  }, [q])
  return (
    <div className="row gap-2 max-md:basis-full">
      <label className="vh" htmlFor="iq">
        {t(common.search)}
      </label>
      <input
        id="iq"
        className="in w-[240px] max-md:w-full"
        placeholder={t(mi.searchPlaceholder)}
        value={q}
        onChange={(e) => setQ(e.target.value)}
        type="search"
      />
      <select className="in w-[160px] max-md:w-full" aria-label={t(mi.filterWorkspace)} value={search.workspace ?? ''} onChange={(e) => set({ workspace: e.target.value || undefined })}>
        <option value="">{t(mi.allWorkspaces)}</option>
        {(list?.workspaces ?? []).map((w) => (
          <option key={w.slug} value={w.slug}>
            {w.name}
          </option>
        ))}
        <option value="instance">{t(mi.instanceOnly)}</option>
      </select>
      <select
        className="in w-[150px] max-md:w-full"
        aria-label={t(ma.filterActorType)}
        value={search.actorType ?? ''}
        onChange={(e) => set({ actorType: (e.target.value || undefined) as InstanceAuditSearch['actorType'] })}
      >
        <option value="">{t(ma.anyActorType)}</option>
        <option value="user">{t(ma.actorType, { type: 'user' })}</option>
        <option value="api_key">{t(ma.actorType, { type: 'api_key' })}</option>
        <option value="mcp_agent">{t(ma.actorType, { type: 'mcp_agent' })}</option>
        <option value="system">{t(ma.actorType, { type: 'system' })}</option>
      </select>
      <select
        className="in w-[120px] max-md:w-full"
        aria-label={t(ma.filterResult)}
        value={search.result ?? ''}
        onChange={(e) => set({ result: (e.target.value || undefined) as InstanceAuditSearch['result'] })}
      >
        <option value="">{t(ma.anyResult)}</option>
        <option value="allowed">{t(ma.result, { result: 'allowed' })}</option>
        <option value="denied">{t(ma.result, { result: 'denied' })}</option>
      </select>
      <AgencyAuditFilter value={search.category} onChange={(category) => set({ category })} />
      <select
        className="in w-[130px] max-md:w-full"
        aria-label={t(ma.dateRange)}
        value={search.range ?? ''}
        onChange={(e) => set({ range: (e.target.value || undefined) as InstanceAuditSearch['range'] })}
      >
        {RANGES.map((r) => (
          <option key={r.value} value={r.value}>
            {t(ma.range, { range: r.range })}
          </option>
        ))}
      </select>
      {search.range === 'custom' ? (
        <>
          <label className="row gap-1.5 text-[12.5px]">
            {t(ma.rangeFrom)}
            <input className="in w-[150px] max-md:w-full" type="date" value={search.since ?? ''} onChange={(e) => set({ since: e.target.value || undefined })} />
          </label>
          <label className="row gap-1.5 text-[12.5px]">
            {t(ma.rangeTo)}
            <input className="in w-[150px] max-md:w-full" type="date" value={search.until ?? ''} onChange={(e) => set({ until: e.target.value || undefined })} />
          </label>
        </>
      ) : null}
    </div>
  )
}

/** "auth · 203.0.113.77", "web · 203.0.113.9" */
function sourceText(s: Item['source']) {
  return s.detail ? `${s.channel} · ${s.detail}` : s.channel
}

function ActorCell({ a }: { a: Item['actor'] }) {
  const avatar =
    a.type === 'mcp_agent' ? (
      <Avatar name={a.name} color="purple" text="AI" />
    ) : a.type === 'system' ? (
      <Avatar name="System" color="muted" text="⚙" />
    ) : a.type === 'api_key' ? (
      <Avatar name={a.name} color="neutral" text="K" />
    ) : (
      <Avatar name={a.name} color={a.avatarColor || 'info'} />
    )
  return (
    // Members wrap under the avatar on phones; the system stays on one line (as on the board).
    <div className={a.type === 'user' ? 'flex min-w-0 items-center gap-2.5 max-md:flex-wrap' : 'flex min-w-0 items-center gap-2.5'}>
      {avatar}
      <div className="flex min-w-0 flex-col leading-[1.3]">
        <span className="font-semibold">{a.name}</span>
        <span className={a.type === 'user' ? 'k mono' : 'k'}>{a.label}</span>
      </div>
    </div>
  )
}

function EventRow({ it, search }: { it: Item; search: InstanceAuditSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const when = instanceAuditTime(it.occurredAt, now)
  const details = it.summary || (it.result !== 'allowed' && it.reason ? it.reason : '')
  return (
    <tr className={it.result === 'allowed' ? undefined : 'bg-bad-tint'}>
      <Td nowrap>{when}</Td>
      <Td>
        <ActorCell a={it.actor} />
      </Td>
      <Td>
        <div className="col gap-0.5">
          <span className="mono font-semibold">{it.action}</span>
          <span>
            {it.target}
            {it.workspace ? <span className="k"> · {it.workspace}</span> : null}
          </span>
          {details ? <span className="k">{details}</span> : null}
          <span className="k mono text-[11px]!">{sourceText(it.source)}</span>
        </div>
      </Td>
      <Td>
        {it.result === 'allowed' ? <Badge tone="ok">{t(ma.result, { result: it.result })}</Badge> : <Badge tone="bad">{t(ma.result, { result: it.result })}</Badge>}
      </Td>
      <Td>
        <ButtonLink
          size="sm"
          to="/w/$workspace/admin/audit"
          params={{ workspace: slug }}
          search={{ ...search, event: it.id }}
          aria-label={t(ma.detailsOf, { action: it.action, time: when })}
          resetScroll={false}
        >
          {t(ma.details)}
        </ButtonLink>
      </Td>
    </tr>
  )
}

function RetentionCard({ settings, error, onRetention, onStream }: { settings?: Schemas['InstanceAuditSettings']; error?: string; onRetention: () => void; onStream: () => void }) {
  const t = useT()
  const ws = settings?.workspaceStreams ?? []
  const hosts = [...new Set(ws.map((w) => w.host))]
  const st = settings?.stream
  return (
    <section className="card" aria-labelledby="ret-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ret-h">
            {t(mi.cardTitle)}
          </h2>
        </div>
      </div>
      <div className="pad col gap-2.5">
        {error ? (
          <Note tone="bad" label={t(ma.notLoaded)}>
            {error}
          </Note>
        ) : !settings ? (
          <div className="col gap-2" aria-busy="true">
            <Skeleton w="90%" />
            <Skeleton w="80%" />
            <Skeleton w="40%" />
          </div>
        ) : (
          <KeyValues
            className="text-[12.5px]"
            items={[
              [t(mi.kvRetention), t(mi.retentionValue, { days: settings.retentionDays, isDefault: settings.retentionDays === 365 ? 'yes' : 'no' })],
              [
                t(mi.kvStreaming),
                t(mi.streamingValue, {
                  workspace: hosts.length ? 'yes' : 'no',
                  hosts: hosts.join(', '),
                  instance: st?.enabled ? (st.failingSince ? 'failing' : 'on') : 'off',
                  host: st?.host ?? '',
                }),
              ],
              [t(mi.kvExport), t(mi.exportValue)],
            ]}
          />
        )}
        <div className="row gap-2">
          <Button size="sm" onClick={onRetention}>
            {t(mi.changeRetention)}
          </Button>
          <Button size="sm" onClick={onStream}>
            {st?.enabled ? t(mi.streamButtonOn) : t(mi.streamButtonOff)}
          </Button>
        </div>
      </div>
    </section>
  )
}

const ACRONYMS: Record<string, string> = { api: 'API', mcp: 'MCP', sso: 'SSO', dns: 'DNS', smtp: 'SMTP', ip: 'IP', url: 'URL' }

/** "sso_provider" → "SSO provider", "api_key" → "API key" */
function typeLabel(t: string) {
  return t
    .split(/[_-]/)
    .filter(Boolean)
    .map((w, i) => ACRONYMS[w.toLowerCase()] ?? (i === 0 ? w.charAt(0).toUpperCase() + w.slice(1) : w))
    .join(' ')
}

function DetailDialog({ id, search }: { id: string; search: InstanceAuditSearch }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const router = useRouter()
  const navigate = useNavigate()
  const close = () => void navigate({ to: '/w/$workspace/admin/audit', params: { workspace: slug }, search: { ...search, event: undefined }, replace: true, resetScroll: false })
  const q = $api.useQuery('get', '/api/v1/admin/audit/events/{eventId}', { params: { path: { eventId: id } } })
  const d = q.data
  const e = d?.event
  const result = e ? t(ma.result, { result: e.result }) : ''
  const open = d?.adminPath ? `/w/${slug}/admin/${d.adminPath}` : d?.link
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && close()}
      size="lg"
      eyebrow={e ? <Badge tone={e.result === 'allowed' ? 'ok' : 'bad'}>{result}</Badge> : undefined}
      title={e ? <span className="mono">{e.action}</span> : t(ma.entryDetails)}
      description={
        e ? (
          <span className="text-[13px]">
            <T
              msg={ma.entryTime}
              values={{
                time: monthDayTimeSeconds(e.occurredAt),
                timezone: me.timezone,
                hasRequest: e.requestId ? 'yes' : 'no',
                requestId: e.requestId ?? '',
                mono: (c) => <span className="mono">{c}</span>,
              }}
            />
          </span>
        ) : undefined
      }
      footer={
        <>
          {d ? (
            <Button
              onClick={async () => {
                const ok = await copyText(JSON.stringify(d.json, null, 2))
                toast({ kind: ok ? 'success' : 'error', title: ok ? tt(ma.jsonCopied) : tt(ma.copyFailed) })
              }}
            >
              {t(ma.copyJson)}
            </Button>
          ) : null}
          {open ? (
            <Button onClick={() => router.history.push(open)}>{d?.linkLabel || t(ma.open)}</Button>
          ) : null}
          <Button variant="primary" onClick={close}>
            {t(common.done)}
          </Button>
        </>
      }
    >
      {q.isError ? (
        <Note tone="bad" label={t(ma.notLoaded)}>
          {errorMessage(q.error)}
        </Note>
      ) : !d || !e ? (
        <div className="col gap-2" aria-busy="true">
          <span className="vh" role="status">
            {t(ma.loadingEntry)}
          </span>
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} w={`${85 - i * 7}%`} />
          ))}
        </div>
      ) : (
        <>
          <KeyValues
            className="text-[12.5px]"
            items={[
              [t(ma.kvWho), [e.actor.name, e.actor.label].filter(Boolean).join(' · ')],
              [t(ma.kvWhat), d.what],
              [
                t(ma.kvTarget),
                <>
                  {typeLabel(e.targetType)} {e.targetId && e.targetType !== 'user' ? <span className="mono">{e.targetId}</span> : e.targetName}
                </>,
              ],
              [t(ma.kvWorkspace), d.workspace],
              [
                t(ma.kvFrom),
                d.ip ? (
                  <>
                    <span className="mono">{d.ip}</span>
                    {d.client ? ` · ${d.client}` : ''}
                  </>
                ) : (
                  sourceText(e.source)
                ),
              ],
              [t(ma.kvResult), [result, d.permission, e.result !== 'allowed' ? e.reason : undefined].filter(Boolean).join(' · ')],
            ]}
          />
          {d.diff.length > 0 ? (
            <>
              <span className="text-[12px] font-semibold text-ink-2">{t(ma.beforeAfter)}</span>
              <div className="diff">
                {d.diff.map((l, i) => (
                  <div key={i} className={l.kind === 'same' ? undefined : l.kind}>
                    {l.kind === 'del' ? '− ' : l.kind === 'add' ? '+ ' : ''}
                    {l.text}
                  </div>
                ))}
              </div>
            </>
          ) : null}
          <Note tone="neutral" className="text-[12.5px]">
            {t(ma.immutable, { hasRelated: d.related.length ? 'yes' : 'no', related: d.related.join('; ') })}
          </Note>
        </>
      )}
    </Dialog>
  )
}

function ListSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        {t(ma.loading)}
      </span>
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="row flex-nowrap gap-4 border-t border-line-2 px-3 py-3.5">
          <Skeleton w={70} h={12} />
          <Skeleton w={28} h={28} r={14} />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton w={140} h={12} />
            <Skeleton w="60%" h={10} />
            <Skeleton w="40%" h={10} />
          </div>
          <Skeleton w={64} h={20} r={999} />
          <Skeleton w={64} h={28} />
        </div>
      ))}
    </div>
  )
}
