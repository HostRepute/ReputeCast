import { keepPreviousData, useInfiniteQuery } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { auditTime, count, monthDayTimeSeconds } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useCan, useMe, useWorkspace } from '@/lib/session'
import type { AuditSearch } from '@/routes/w/$workspace/_app/settings/audit'
import { AccessTable, copyText } from '../access/parts'
import { AuditStreamDialog } from './AuditStreamDialog'
import { zonedDayStart } from './zoned'

type Item = Schemas['AuditEventItem']
type List = Schemas['AuditEventList']

const PAGE = 10

/** The API filters for the current search (custom ranges become instants in the workspace timezone). */
function apiFilters(s: AuditSearch, tz: string) {
  const custom = s.range === 'custom'
  return {
    q: s.q || undefined,
    actorType: s.actorType,
    actor: s.actor,
    action: s.action,
    category: s.category,
    result: s.result,
    range: (custom ? 'all' : s.range) as Exclude<AuditSearch['range'], 'custom'>,
    since: custom && s.since ? zonedDayStart(s.since, tz).toISOString() : undefined,
    until: custom && s.until ? zonedDayStart(s.until, tz, 1).toISOString() : undefined,
  }
}

/** A5 §5.11 Audit log (SettingsAudit): who did what, to what, from where, and whether it was allowed. */
export function AuditPage({ search }: { search: AuditSearch }) {
  const { slug, me } = useWorkspace()
  const can = useCan('audit:view')
  const filters = apiFilters(search, me.timezone)
  const q = useInfiniteQuery({
    queryKey: ['audit', slug, filters],
    queryFn: async ({ pageParam }) =>
      (await http.GET('/api/v1/workspaces/{workspace}/audit', { params: { path: { workspace: slug }, query: { ...filters, cursor: pageParam || undefined, limit: PAGE } } }))
        .data!,
    initialPageParam: '',
    getNextPageParam: (last: List) => last.nextCursor || undefined,
    placeholderData: keepPreviousData,
    enabled: can,
  })
  const first = q.data?.pages[0]
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  return (
    <Page title="Settings" tabs={settingsTabs(slug)}>
      {!can ? (
        <Note tone="neutral" label="Read-only">
          {deniedReason('audit:view')}
        </Note>
      ) : (
        <>
          <TopBar list={first} filters={filters} />
          <div className="grid grid-cols-[minmax(0,1fr)_380px] items-start gap-5 max-[1100px]:grid-cols-1">
            <section className="card" aria-labelledby="ev-h">
              <div className="sec">
                <div className="flex min-w-0 flex-col gap-0.5">
                  <h2 className="h2" id="ev-h">
                    Events
                  </h2>
                  <span className="k">Who did what, to what, from where, and whether it was allowed</span>
                </div>
                <Filters search={search} list={first} />
              </div>
              {q.isError ? (
                <div className="pad">
                  <ErrorPanel title="Couldn't load the audit log" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
                    {errorMessage(q.error)}
                  </ErrorPanel>
                </div>
              ) : !q.data ? (
                <ListSkeleton />
              ) : (
                <>
                  <AccessTable caption="Audit entries" aria-busy={q.isFetching && !q.isFetchingNextPage ? true : undefined}>
                    <thead>
                      <tr>
                        <Th>Time</Th>
                        <Th>Actor</Th>
                        <Th>Event · target · details · source</Th>
                        <Th>Result</Th>
                        <Th>
                          <span className="vh">Details</span>
                        </Th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.length === 0 ? (
                        <EmptyRow colSpan={5}>
                          <span className="col items-center gap-2">
                            No entries match these filters.
                            <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{}}>
                              Clear filters
                            </ButtonLink>
                          </span>
                        </EmptyRow>
                      ) : (
                        items.map((it) => <EventRow key={it.id} it={it} selected={it.id === search.event} search={search} />)
                      )}
                    </tbody>
                  </AccessTable>
                  {q.hasNextPage ? (
                    <div className="row justify-center border-t border-line-2 px-4.5 py-3">
                      <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
                        Show older entries
                      </Button>
                    </div>
                  ) : null}
                </>
              )}
            </section>
            <Detail id={search.event} />
          </div>
        </>
      )}
    </Page>
  )
}

function TopBar({ list, filters }: { list?: List; filters: ReturnType<typeof apiFilters> }) {
  const { slug } = useWorkspace()
  const me = useMe().data
  const [stream, setStream] = useState(false)
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) if (v) qs.set(k, String(v))
  const exportURL = (format: 'csv' | 'json') => `/api/v1/workspaces/${slug}/audit/export?${new URLSearchParams([...qs, ['format', format]]).toString()}`
  const st = list?.stream
  return (
    <div className="row justify-between">
      <div className="row gap-2">
        {!st ? (
          <Skeleton w={190} h={22} r={999} />
        ) : st.enabled && st.failingSince ? (
          <Badge tone="bad" dot>
            Stream to {st.host} failing
          </Badge>
        ) : st.enabled ? (
          <Badge tone="ok" dot>
            Streaming to {st.host}
          </Badge>
        ) : (
          <Badge tone="neutral" dot>
            Not streaming
          </Badge>
        )}
        {me?.user.isInstanceAdmin ? (
          <ButtonLink size="sm" to="/w/$workspace/admin/audit" params={{ workspace: slug }}>
            Instance-level view
          </ButtonLink>
        ) : null}
      </div>
      <div className="row gap-2">
        {list ? (
          <span className="k">
            Retained {list.retentionDays} days · immutable · {count(list.total)} entries
          </span>
        ) : null}
        <ButtonAnchor size="sm" href={exportURL('csv')} newTab={false} download="">
          Export CSV
        </ButtonAnchor>
        <ButtonAnchor size="sm" href={exportURL('json')} newTab={false} download="">
          Export JSON
        </ButtonAnchor>
        <Button size="sm" disabledReason={st ? undefined : 'Loading…'} onClick={() => setStream(true)}>
          Webhook stream · {st?.enabled ? 'on' : 'off'}
        </Button>
      </div>
      {stream ? <AuditStreamDialog onClose={() => setStream(false)} /> : null}
    </div>
  )
}

const RANGES: { value: NonNullable<AuditSearch['range']>; label: string }[] = [
  { value: '7d', label: 'Last 7 days' },
  { value: '24h', label: 'Last 24 hours' },
  { value: '30d', label: 'Last 30 days' },
  { value: '90d', label: 'Last 90 days' },
  { value: 'all', label: 'All retained' },
  { value: 'custom', label: 'Custom' },
]

function Filters({ search, list }: { search: AuditSearch; list?: List }) {
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const [q, setQ] = useState(search.q ?? '')
  const set = (patch: Partial<AuditSearch>) =>
    void navigate({ to: '/w/$workspace/settings/audit', params: { workspace: slug }, search: { ...search, ...patch, event: undefined }, replace: true })
  // Search as you type, a beat after the last key.
  useEffect(() => {
    if ((search.q ?? '') === q) return
    const h = window.setTimeout(() => set({ q: q || undefined }), 300)
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- `set` changes every render; the search value is what matters
  }, [q])
  const actor = search.actor ? list?.actors.find((a) => a.id === search.actor) : undefined
  return (
    <div className="row gap-2">
      <label className="vh" htmlFor="aq">
        Search
      </label>
      <input id="aq" className="in w-[220px] max-sm:w-full" placeholder="Search target or request ID" value={q} onChange={(e) => setQ(e.target.value)} type="search" />
      <select
        className="in w-[150px]"
        aria-label="Filter by actor type"
        value={search.actorType ?? ''}
        onChange={(e) => set({ actorType: (e.target.value || undefined) as AuditSearch['actorType'], actor: undefined })}
      >
        <option value="">Any actor type</option>
        <option value="user">Members</option>
        <option value="api_key">API keys</option>
        <option value="mcp_agent">MCP agents</option>
        <option value="system">System</option>
      </select>
      <select
        className="in w-[150px]"
        aria-label="Filter by action"
        value={search.category ?? (search.action ? `a:${search.action}` : '')}
        onChange={(e) => {
          const v = e.target.value
          set(v.startsWith('a:') ? { action: v.slice(2), category: undefined } : { category: (v || undefined) as AuditSearch['category'], action: undefined })
        }}
      >
        <option value="">Any action</option>
        {(list?.categories ?? []).map((c) => (
          <option key={c.key} value={c.key}>
            {c.label}
          </option>
        ))}
        {search.action ? <option value={`a:${search.action}`}>{search.action}</option> : null}
      </select>
      <select
        className="in w-[120px]"
        aria-label="Filter by result"
        value={search.result ?? ''}
        onChange={(e) => set({ result: (e.target.value || undefined) as AuditSearch['result'] })}
      >
        <option value="">Any result</option>
        <option value="allowed">Allowed</option>
        <option value="denied">Denied</option>
        <option value="error">Error</option>
      </select>
      <select
        className="in w-[130px]"
        aria-label="Date range"
        value={search.range ?? '7d'}
        onChange={(e) => set({ range: e.target.value === '7d' ? undefined : (e.target.value as AuditSearch['range']) })}
      >
        {RANGES.map((r) => (
          <option key={r.value} value={r.value}>
            {r.label}
          </option>
        ))}
      </select>
      {search.range === 'custom' ? (
        <>
          <label className="row gap-1.5 text-[12.5px]">
            From
            <input className="in w-[150px]" type="date" value={search.since ?? ''} onChange={(e) => set({ since: e.target.value || undefined })} />
          </label>
          <label className="row gap-1.5 text-[12.5px]">
            To
            <input className="in w-[150px]" type="date" value={search.until ?? ''} onChange={(e) => set({ until: e.target.value || undefined })} />
          </label>
        </>
      ) : null}
      {actor ? (
        <span className="chip">
          {actor.name}
          <button type="button" aria-label={`Stop filtering by ${actor.name}`} onClick={() => set({ actor: undefined })}>
            ×
          </button>
        </span>
      ) : null}
      <Badge tone="neutral">Workspace · {me.workspace.name}</Badge>
    </div>
  )
}

/** "web · 203.0.113.9", "mcp · rc_agent_7a…c2", "guard · threshold" */
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
    <div className="flex min-w-0 items-center gap-2.5">
      {avatar}
      <div className="flex min-w-0 flex-col leading-[1.3]">
        <span className="font-semibold">{a.name}</span>
        <span className={a.type === 'user' ? 'k mono' : 'k'}>{a.label}</span>
      </div>
    </div>
  )
}

function EventRow({ it, selected, search }: { it: Item; selected: boolean; search: AuditSearch }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const details = it.summary || (it.result !== 'allowed' && it.reason ? it.reason : '')
  return (
    <tr className={selected ? 'bg-accent-soft' : it.result === 'allowed' ? undefined : 'bg-bad-tint'}>
      <Td nowrap>{auditTime(it.occurredAt, now)}</Td>
      <Td>
        <ActorCell a={it.actor} />
      </Td>
      <Td>
        <div className="col gap-0.5">
          <span className="mono font-semibold">{it.action}</span>
          <span>{it.target}</span>
          {details ? <span className="k">{details}</span> : null}
          <span className="k mono text-[11px]!">{sourceText(it.source)}</span>
        </div>
      </Td>
      <Td>
        {it.result === 'allowed' ? <Badge tone="ok">Allowed</Badge> : <Badge tone="bad">{it.result === 'denied' ? 'Denied' : 'Error'}</Badge>}
      </Td>
      <Td>
        <ButtonLink
          size="sm"
          to="/w/$workspace/settings/audit"
          params={{ workspace: slug }}
          search={{ ...search, event: it.id }}
          aria-current={selected ? 'true' : undefined}
          aria-label={`Details of ${it.action} at ${auditTime(it.occurredAt, now)}`}
          resetScroll={false}
        >
          Details
        </ButtonLink>
      </Td>
    </tr>
  )
}

/** "routing_rule" → "Routing rule" */
function typeLabel(t: string) {
  const s = t.replace(/_/g, ' ')
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function Detail({ id }: { id?: string }) {
  const { slug, me } = useWorkspace()
  const router = useRouter()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audit/events/{eventId}', { params: { path: { workspace: slug, eventId: id ?? '' } } }, { enabled: Boolean(id) })
  if (!id)
    return (
      <section className="card pad col gap-1.5 min-[1101px]:sticky min-[1101px]:top-4" aria-label="Entry details">
        <span className="font-semibold">Entry details</span>
        <span className="k">Pick an entry to see who did it, from where, the permission it needed and what changed.</span>
      </section>
    )
  if (q.isError)
    return (
      <ErrorPanel title="Couldn't load this entry" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  if (!q.data)
    return (
      <section className="card pad col gap-3" aria-busy="true">
        <span className="vh" role="status">
          Loading the entry
        </span>
        <Skeleton w={140} h={16} />
        <Skeleton w="80%" />
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} w={`${85 - i * 7}%`} />
        ))}
      </section>
    )
  const d = q.data
  const e = d.event
  const result = e.result === 'allowed' ? 'Allowed' : e.result === 'denied' ? 'Denied' : 'Error'
  return (
    <section className="card min-[1101px]:sticky min-[1101px]:top-4" aria-labelledby="h-ev">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-ev">
            {e.action}
          </h2>
          <span className="k">
            {monthDayTimeSeconds(e.occurredAt)} {me.timezone}
            {e.requestId ? (
              <>
                {' '}
                · request <span className="mono">{e.requestId}</span>
              </>
            ) : null}
          </span>
        </div>
        <Badge tone={e.result === 'allowed' ? 'ok' : 'bad'}>{result}</Badge>
      </div>
      <div className="pad col gap-3">
        <KeyValues
          className="text-[12.5px]"
          items={[
            ['Who', [e.actor.name, e.actor.label, e.actor.role].filter(Boolean).join(' · ')],
            ['What', d.what],
            [
              'Target',
              <>
                {typeLabel(e.targetType)} {e.targetId ? <span className="mono">{e.targetId}</span> : e.targetName}
              </>,
            ],
            ['Workspace', d.workspace],
            [
              'From',
              d.ip ? (
                <>
                  <span className="mono">{d.ip}</span>
                  {d.client ? ` · ${d.client}` : ''}
                </>
              ) : (
                sourceText(e.source)
              ),
            ],
            ['Result', [result, d.permission, e.result !== 'allowed' ? e.reason : undefined].filter(Boolean).join(' · ')],
          ]}
        />
        {d.diff.length > 0 ? (
          <>
            <span className="text-[12px] font-semibold text-ink-2">Before → after</span>
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
          Entries can&apos;t be edited or deleted from the UI.{d.related.length ? ` Related: ${d.related.join('; ')}.` : ''}
        </Note>
        <div className="row gap-2">
          {d.link ? (
            <Button size="sm" onClick={() => router.history.push(d.link!)}>
              {d.linkLabel || 'Open'}
            </Button>
          ) : null}
          <Button
            size="sm"
            onClick={async () => {
              const ok = await copyText(JSON.stringify(d.json, null, 2))
              toast({ kind: ok ? 'success' : 'error', title: ok ? 'Entry copied as JSON' : 'Copy failed' })
            }}
          >
            Copy JSON
          </Button>
        </div>
      </div>
    </section>
  )
}

function ListSkeleton() {
  return (
    <div className="flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        Loading the audit log
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
