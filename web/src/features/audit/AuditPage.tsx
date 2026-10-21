import { keepPreviousData, useInfiniteQuery } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { useRestoreList } from '@/features/audience/DeleteListDialog'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { auditTime, count, monthDayTimeSeconds, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { useCan, useMe, usePermissionReason, useWorkspace } from '@/lib/session'
import type { AuditSearch } from '@/routes/w/$workspace/_app/settings/audit'
import { AccessTable, copyText } from '../access/parts'
import { AuditStreamDialog } from './AuditStreamDialog'
import { m } from './messages'
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
  const t = useT()
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
  // Without a pick, the panel shows the latest entry that changed something (as on the board).
  const selected = search.event ?? items.find((i) => i.changes)?.id
  return (
    <Page title={t(m.title)} tabs={settingsTabs(slug)}>
      {!can ? (
        <ForbiddenState permission="audit:view" title={t(m.forbidden)} />
      ) : (
        <>
          <TopBar list={first} filters={filters} />
          <div className="grid grid-cols-[minmax(0,1fr)_380px] items-start gap-5 max-md:grid-cols-1">
            <section className="card" aria-labelledby="ev-h">
              <div className="sec">
                <div className="flex min-w-0 flex-col gap-0.5">
                  <h2 className="h2" id="ev-h">
                    {t(m.eventsTitle)}
                  </h2>
                  <span className="k">{t(m.eventsCaption)}</span>
                </div>
                <Filters search={search} list={first} />
              </div>
              {q.isError ? (
                <div className="pad">
                  <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
                    {errorMessage(q.error)}
                  </ErrorPanel>
                </div>
              ) : !q.data ? (
                <ListSkeleton />
              ) : (
                <>
                  <AccessTable caption={t(m.tableCaption)} aria-busy={q.isFetching && !q.isFetchingNextPage ? true : undefined}>
                    <thead>
                      <tr>
                        <Th>{t(m.colTime)}</Th>
                        <Th>{t(m.colActor)}</Th>
                        <Th>{t(m.colEvent)}</Th>
                        <Th>{t(m.colResult)}</Th>
                        <Th>
                          <span className="vh">{t(m.colDetails)}</span>
                        </Th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.length === 0 ? (
                        <EmptyRow colSpan={5}>
                          <span className="col items-center gap-2">
                            {t(m.noMatch)}
                            <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{}}>
                              {t(m.clearFilters)}
                            </ButtonLink>
                          </span>
                        </EmptyRow>
                      ) : (
                        items.map((it) => <EventRow key={it.id} it={it} selected={it.id === selected} search={search} />)
                      )}
                    </tbody>
                  </AccessTable>
                  {q.hasNextPage ? (
                    <div className="row justify-center border-t border-line-2 px-4.5 py-3">
                      <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
                        {t(m.showOlder)}
                      </Button>
                    </div>
                  ) : null}
                </>
              )}
            </section>
            <Detail id={selected} pending={!q.data} />
          </div>
        </>
      )}
    </Page>
  )
}

function TopBar({ list, filters }: { list?: List; filters: ReturnType<typeof apiFilters> }) {
  const t = useT()
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
            {t(m.streamFailingBadge, { host: st.host ?? '' })}
          </Badge>
        ) : st.enabled ? (
          <Badge tone="ok" dot>
            {t(m.streamingTo, { host: st.host ?? '' })}
          </Badge>
        ) : (
          <Badge tone="neutral" dot>
            {t(m.notStreaming)}
          </Badge>
        )}
        {me?.user.isInstanceAdmin ? (
          <ButtonLink size="sm" to="/w/$workspace/admin/audit" params={{ workspace: slug }}>
            {t(m.instanceView)}
          </ButtonLink>
        ) : null}
      </div>
      <div className="row gap-2">
        {list ? <span className="k">{t(m.retained, { days: list.retentionDays, count: list.total, total: count(list.total) })}</span> : null}
        <ButtonAnchor size="sm" href={exportURL('csv')} newTab={false} download="">
          {t(m.exportCsv)}
        </ButtonAnchor>
        <ButtonAnchor size="sm" href={exportURL('json')} newTab={false} download="">
          {t(m.exportJson)}
        </ButtonAnchor>
        <Button size="sm" disabledReason={st ? undefined : t(common.loading)} onClick={() => setStream(true)}>
          {t(m.webhookStream, { on: st?.enabled ? 'yes' : 'no' })}
        </Button>
      </div>
      {stream ? <AuditStreamDialog onClose={() => setStream(false)} /> : null}
    </div>
  )
}

const RANGES: NonNullable<AuditSearch['range']>[] = ['7d', '24h', '30d', '90d', 'all', 'custom']

function Filters({ search, list }: { search: AuditSearch; list?: List }) {
  const t = useT()
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
        {t(common.search)}
      </label>
      <input id="aq" className="in w-[220px] max-sm:w-full" placeholder={t(m.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} type="search" />
      <select
        className="in w-[150px] max-md:w-full"
        aria-label={t(m.filterActorType)}
        value={search.actorType ?? ''}
        onChange={(e) => set({ actorType: (e.target.value || undefined) as AuditSearch['actorType'], actor: undefined })}
      >
        <option value="">{t(m.anyActorType)}</option>
        <option value="user">{t(m.actorType, { type: 'user' })}</option>
        <option value="api_key">{t(m.actorType, { type: 'api_key' })}</option>
        <option value="mcp_agent">{t(m.actorType, { type: 'mcp_agent' })}</option>
        <option value="system">{t(m.actorType, { type: 'system' })}</option>
      </select>
      <select
        className="in w-[150px] max-md:w-full"
        aria-label={t(m.filterAction)}
        value={search.category ?? (search.action ? `a:${search.action}` : '')}
        onChange={(e) => {
          const v = e.target.value
          set(v.startsWith('a:') ? { action: v.slice(2), category: undefined } : { category: (v || undefined) as AuditSearch['category'], action: undefined })
        }}
      >
        <option value="">{t(m.anyAction)}</option>
        {(list?.categories ?? []).map((c) => (
          <option key={c.key} value={c.key}>
            {c.label}
          </option>
        ))}
        {search.action ? <option value={`a:${search.action}`}>{search.action}</option> : null}
      </select>
      <select
        className="in w-[120px] max-md:w-full"
        aria-label={t(m.filterResult)}
        value={search.result ?? ''}
        onChange={(e) => set({ result: (e.target.value || undefined) as AuditSearch['result'] })}
      >
        <option value="">{t(m.anyResult)}</option>
        <option value="allowed">{t(m.result, { result: 'allowed' })}</option>
        <option value="denied">{t(m.result, { result: 'denied' })}</option>
        <option value="error">{t(m.result, { result: 'error' })}</option>
      </select>
      <select
        className="in w-[130px] max-md:w-full"
        aria-label={t(m.dateRange)}
        value={search.range ?? '7d'}
        onChange={(e) => set({ range: e.target.value === '7d' ? undefined : (e.target.value as AuditSearch['range']) })}
      >
        {RANGES.map((r) => (
          <option key={r} value={r}>
            {t(m.range, { range: r })}
          </option>
        ))}
      </select>
      {search.range === 'custom' ? (
        <>
          <label className="row gap-1.5 text-[12.5px]">
            {t(m.rangeFrom)}
            <input className="in w-[150px] max-md:w-full" type="date" value={search.since ?? ''} onChange={(e) => set({ since: e.target.value || undefined })} />
          </label>
          <label className="row gap-1.5 text-[12.5px]">
            {t(m.rangeTo)}
            <input className="in w-[150px] max-md:w-full" type="date" value={search.until ?? ''} onChange={(e) => set({ until: e.target.value || undefined })} />
          </label>
        </>
      ) : null}
      {actor ? (
        <span className="chip">
          {actor.name}
          <button type="button" aria-label={t(m.stopFiltering, { name: actor.name })} onClick={() => set({ actor: undefined })}>
            ×
          </button>
        </span>
      ) : null}
      <Badge tone="neutral">{t(m.workspaceBadge, { name: me.workspace.name })}</Badge>
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
    // Members wrap under the avatar on phones; agents and the system stay on one line (as on the board).
    <div className={a.type === 'user' ? 'flex min-w-0 items-center gap-2.5 max-md:flex-wrap' : 'flex min-w-0 items-center gap-2.5'}>
      {avatar}
      <div className="flex min-w-0 flex-col leading-[1.3]">
        <span className="font-semibold">{a.name}</span>
        <span className={a.type === 'user' ? 'k mono' : 'k'}>{a.label}</span>
      </div>
    </div>
  )
}

function EventRow({ it, selected, search }: { it: Item; selected: boolean; search: AuditSearch }) {
  const t = useT()
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
        {it.result === 'allowed' ? <Badge tone="ok">{t(m.result, { result: it.result })}</Badge> : <Badge tone="bad">{t(m.result, { result: it.result })}</Badge>}
      </Td>
      <Td>
        <ButtonLink
          size="sm"
          to="/w/$workspace/settings/audit"
          params={{ workspace: slug }}
          search={{ ...search, event: it.id }}
          aria-current={selected ? 'true' : undefined}
          aria-label={t(m.detailsOf, { action: it.action, time: auditTime(it.occurredAt, now) })}
          resetScroll={false}
        >
          {t(m.details)}
        </ButtonLink>
      </Td>
    </tr>
  )
}

const ACRONYMS: Record<string, string> = { api: 'API', mcp: 'MCP', sso: 'SSO', dns: 'DNS', smtp: 'SMTP', ip: 'IP', url: 'URL', ai: 'AI' }

/** "routing_rule" → "Routing rule", "api_key" → "API key" */
function typeLabel(t: string) {
  return t
    .split(/[_-]/)
    .filter(Boolean)
    .map((w, i) => ACRONYMS[w.toLowerCase()] ?? (i === 0 ? w.charAt(0).toUpperCase() + w.slice(1) : w))
    .join(' ')
}

function Detail({ id, pending }: { id?: string; pending?: boolean }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const router = useRouter()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audit/events/{eventId}', { params: { path: { workspace: slug, eventId: id ?? '' } } }, { enabled: Boolean(id) })
  // A deleted list comes back from its entry for 30 days.
  const restoreList = useRestoreList()
  const cantRestore = usePermissionReason('lists:manage')
  const [restoring, setRestoring] = useState(false)
  if (!id && !pending)
    return (
      <section className="card pad col gap-1.5 md:sticky md:top-4" aria-label={t(m.entryDetails)}>
        <span className="font-semibold">{t(m.entryDetails)}</span>
        <span className="k">{t(m.pickEntry)}</span>
      </section>
    )
  if (q.isError)
    return (
      <ErrorPanel title={t(m.entryLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  if (!q.data)
    return (
      <section className="card pad col gap-3" aria-busy="true">
        <span className="vh" role="status">
          {t(m.loadingEntry)}
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
  const result = t(m.result, { result: e.result })
  return (
    <section className="card md:sticky md:top-4" aria-labelledby="h-ev">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-ev">
            {e.action}
          </h2>
          <span className="k">
            <T
              msg={m.entryTime}
              values={{
                time: monthDayTimeSeconds(e.occurredAt),
                timezone: me.timezone,
                hasRequest: e.requestId ? 'yes' : 'no',
                requestId: e.requestId ?? '',
                mono: (c) => <span className="mono">{c}</span>,
              }}
            />
          </span>
        </div>
        <Badge tone={e.result === 'allowed' ? 'ok' : 'bad'}>{result}</Badge>
      </div>
      <div className="pad col gap-3">
        <KeyValues
          className="text-[12.5px]"
          items={[
            [t(m.kvWho), [e.actor.name, e.actor.label, e.actor.role ? roleName({ name: e.actor.role }) : undefined].filter(Boolean).join(' · ')],
            [t(m.kvWhat), d.what],
            [
              t(m.kvTarget),
              <>
                {typeLabel(e.targetType)} {e.targetId ? <span className="mono">{e.targetId}</span> : e.targetName}
              </>,
            ],
            [t(m.kvWorkspace), d.workspace],
            [
              t(m.kvFrom),
              d.ip ? (
                <>
                  <span className="mono">{d.ip}</span>
                  {d.client ? ` · ${d.client}` : ''}
                </>
              ) : (
                sourceText(e.source)
              ),
            ],
            [t(m.kvResult), [result, d.permission, e.result !== 'allowed' ? e.reason : undefined].filter(Boolean).join(' · ')],
          ]}
        />
        {d.diff.length > 0 ? (
          <>
            <span className="text-[12px] font-semibold text-ink-2">{t(m.beforeAfter)}</span>
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
          {t(m.immutable, { hasRelated: d.related.length ? 'yes' : 'no', related: d.related.join('; ') })}
        </Note>
        <div className="row gap-2">
          {d.restore ? (
            <Button
              size="sm"
              loading={restoring}
              busyLabel={t(m.restoring)}
              disabledReason={cantRestore}
              onClick={async () => {
                setRestoring(true)
                await restoreList(d.restore!.id, d.restore!.name)
                setRestoring(false)
              }}
            >
              {t(m.restoreList)}
            </Button>
          ) : null}
          {d.link ? (
            <Button size="sm" onClick={() => router.history.push(d.link!)}>
              {d.linkLabel || t(m.open)}
            </Button>
          ) : null}
          <Button
            size="sm"
            onClick={async () => {
              const ok = await copyText(JSON.stringify(d.json, null, 2))
              toast({ kind: ok ? 'success' : 'error', title: ok ? tt(m.jsonCopied) : tt(m.copyFailed) })
            }}
          >
            {t(m.copyJson)}
          </Button>
          {d.restore ? <span className="k">{t(m.restorableUntil, { date: shortDate(d.restore.until) })}</span> : null}
        </div>
      </div>
    </section>
  )
}

function ListSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
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
