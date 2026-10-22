import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { clock, clockSeconds, count, dayMonthDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ProvidersTabs, throwIfForbidden } from './common'
import { me } from './events.messages'
import { m } from './messages'

type EventType = Schemas['ProviderLogEvent']['type']

export interface EventsSearch {
  provider?: string
  type?: EventType
  q?: string
  page?: number
}

const PAGE_SIZE = 25

const typeBadge: Record<EventType, { tone: Tone; text: MessageDescriptor }> = {
  delivered: { tone: 'ok', text: me.typeDelivered },
  deferred: { tone: 'warn', text: me.typeDeferred },
  hard_bounce: { tone: 'bad', text: me.typeHardBounce },
  soft_bounce: { tone: 'warn', text: me.typeSoftBounce },
  complaint: { tone: 'bad', text: me.typeComplaint },
  rejected: { tone: 'bad', text: me.typeRejected },
}

const typeOptions: { value: EventType; label: MessageDescriptor }[] = [
  { value: 'delivered', label: me.optionDelivered },
  { value: 'deferred', label: me.optionDeferred },
  { value: 'hard_bounce', label: me.optionHardBounce },
  { value: 'soft_bounce', label: me.optionSoftBounce },
  { value: 'complaint', label: me.optionComplaint },
  { value: 'rejected', label: me.optionRejected },
]

/** 7.3 Events & sync: the raw event log, suppression sync and provider-side tracking. */
export function EventsPage({ search }: { search: EventsSearch }) {
  const t = useT()
  return (
    <Page title={t(m.title)} filters={<ProvidersTabs />}>
      <EventLog search={search} />
      <Grid min={420} gap={20}>
        <SyncCard />
        <TrackingCard />
      </Grid>
    </Page>
  )
}

function EventLog({ search }: { search: EventsSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const page = search.page ?? 1
  const query = { provider: search.provider, type: search.type, q: search.q, page, pageSize: PAGE_SIZE }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/provider-events', { params: { path: { workspace: slug }, query } }, { placeholderData: keepPreviousData })
  throwIfForbidden(q.error)
  const providers = $api.useQuery('get', '/api/v1/workspaces/{workspace}/providers', { params: { path: { workspace: slug } } })
  const go = (next: EventsSearch) =>
    void navigate({ to: '/w/$workspace/providers/events', params: { workspace: slug }, search: { ...search, page: undefined, ...next }, replace: true })
  const exportUrl = `/api/v1/workspaces/${encodeURIComponent(slug)}/provider-events/export?${new URLSearchParams(
    Object.entries({ provider: search.provider, type: search.type, q: search.q }).filter((e): e is [string, string] => Boolean(e[1])),
  ).toString()}`
  const data = q.data
  const pages = data ? Math.max(1, Math.ceil(data.total / data.pageSize)) : 1
  return (
    <section className="card flex flex-col" aria-labelledby="pv-event-log">
      <div className="flex flex-wrap items-center justify-between gap-2.5 border-b border-line-2 px-5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="pv-event-log">
            {t(me.logTitle)}
          </h2>
          <span className="k">{t(me.logLine)}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          <select className="in max-md:w-full" aria-label={t(me.providerFilter)} value={search.provider ?? ''} onChange={(e) => go({ provider: e.target.value || undefined })}>
            <option value="">{t(me.allProviders)}</option>
            {providers.data?.providers.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <select
            className="in max-md:w-full"
            aria-label={t(me.typeFilter)}
            value={search.type ?? ''}
            onChange={(e) => go({ type: (e.target.value || undefined) as EventType | undefined })}
          >
            <option value="">{t(me.allEvents)}</option>
            {typeOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {t(o.label)}
              </option>
            ))}
          </select>
          <SearchInput initial={search.q ?? ''} onSearch={(v) => go({ q: v || undefined })} />
          <ButtonAnchor href={exportUrl} newTab={false} download="provider-events.csv">
            {t(me.exportCsv)}
          </ButtonAnchor>
        </div>
      </div>
      {q.isError ? (
        <div className="p-5">
          <ErrorPanel title={t(me.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(me.loadFailedBody)}
          </ErrorPanel>
        </div>
      ) : (
        <Table className="pv-10 pv-top" phoneWrap fit aria-busy={q.isFetching || undefined}>
          <thead>
            <tr>
              <Th>{t(me.colReceived)}</Th>
              <Th>{t(me.colProvider)}</Th>
              <Th>{t(me.colRaw)}</Th>
              <Th>{t(me.colNormalized)}</Th>
              <Th>{t(me.colCode)}</Th>
              <Th>{t(me.colRecipient)}</Th>
              <Th>{t(me.colSource)}</Th>
            </tr>
          </thead>
          <tbody>
            {!data ? (
              Array.from({ length: 6 }, (_, i) => (
                <tr key={i}>
                  {[60, 90, 150, 80, 260, 120, 120].map((w, j) => (
                    <Td key={j}>
                      <Skeleton w={w} />
                    </Td>
                  ))}
                </tr>
              ))
            ) : data.items.length === 0 ? (
              <EmptyRow colSpan={7}>{search.q || search.type || search.provider ? t(me.noMatch) : t(me.empty)}</EmptyRow>
            ) : (
              data.items.map((e) => (
                <tr key={e.id}>
                  <Td className="mono whitespace-nowrap">{clockSeconds(e.receivedAt)}</Td>
                  <Td>{e.provider}</Td>
                  <Td>
                    <span className="pv-raw mono">{e.rawType}</span>
                  </Td>
                  <Td>
                    <Badge tone={typeBadge[e.type].tone}>{t(typeBadge[e.type].text)}</Badge>
                  </Td>
                  <Td className="mono text-[12px]">{[e.code, e.diagnostic].filter(Boolean).join(' ')}</Td>
                  <Td className="mono">{e.recipient}</Td>
                  <Td>{e.source || '–'}</Td>
                </tr>
              ))
            )}
          </tbody>
        </Table>
      )}
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-2.5">
        <span className="k" aria-live="polite">
          {data ? t(me.showing, { shown: count(data.items.length), total: data.total, days: data.retentionDays }) : t(me.loadingEvents)}
        </span>
        <div className="flex gap-1.5">
          <Button size="sm" disabled={page <= 1} onClick={() => go({ page: page - 1 > 1 ? page - 1 : undefined })}>
            {t(me.previous)}
          </Button>
          <Button size="sm" disabled={page >= pages} onClick={() => go({ page: page + 1 })}>
            {t(me.next)}
          </Button>
        </div>
      </div>
    </section>
  )
}

function SearchInput({ initial, onSearch }: { initial: string; onSearch: (v: string) => void }) {
  const t = useT()
  const [value, setValue] = useState(initial)
  useEffect(() => {
    if (value === initial) return
    const h = window.setTimeout(() => onSearch(value.trim()), 300)
    return () => window.clearTimeout(h)
  }, [value, initial, onSearch])
  return (
    <input
      className="in w-[220px] max-md:w-full"
      type="search"
      placeholder={t(me.searchPlaceholder)}
      aria-label={t(me.searchLabel)}
      value={value}
      onChange={(e) => setValue(e.target.value)}
    />
  )
}

const direction: Record<Schemas['SuppressionSyncRow']['direction'], MessageDescriptor> = {
  two_way: me.dirTwoWay,
  push: me.dirPush,
  pull: me.dirPull,
  off: me.dirOff,
  local: me.dirLocal,
  import: me.dirImport,
}

const syncStatus: Record<Schemas['SuppressionSyncRow']['status'], { tone: Tone; text: MessageDescriptor }> = {
  in_sync: { tone: 'ok', text: me.syncInSync },
  syncing: { tone: 'info', text: me.syncSyncing },
  error: { tone: 'bad', text: me.syncError },
  local: { tone: 'neutral', text: me.syncLocal },
  done: { tone: 'neutral', text: me.syncDone },
  off: { tone: 'neutral', text: me.syncOff },
}

const reasonBadge: Record<string, { tone: Tone; text: MessageDescriptor }> = {
  hard_bounce: { tone: 'bad', text: me.reasonHardBounce },
  complaint: { tone: 'bad', text: me.reasonComplaint },
  unsubscribe: { tone: 'neutral', text: me.reasonUnsubscribed },
  manual: { tone: 'neutral', text: me.reasonManual },
}

/** The shared suppression list and how each provider stays in sync with it. */
function SyncCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const reason = usePermissionReason('providers:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/suppression-sync', { params: { path: { workspace: slug } } })
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/suppression-sync/run', {
    onSuccess: (r) => {
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/suppression-sync'] })
      toast({ kind: 'success', title: r.queued ? tt(me.syncQueued, { count: r.queued }) : tt(me.nothingToSync), detail: tt(me.syncDetail) })
    },
    onError: (e) => toast({ kind: 'error', title: tt(me.syncFailed), detail: errorMessage(e) }),
  })
  const data = q.data
  const when = (at?: string) => {
    if (!at) return '–'
    const d = new Date(at)
    return now.getTime() - d.getTime() > 20 * 3600_000 ? dayMonthDate(d) : clock(d)
  }
  return (
    <section className="card flex flex-col" aria-labelledby="pv-sync">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="pv-sync">
            {t(me.syncTitle)}
          </h2>
          <span className="k">{data ? t(me.syncLine, { count: data.total }) : t(common.loading)}</span>
        </div>
        <Button size="sm" variant="primary" disabledReason={reason} loading={run.isPending} onClick={() => run.mutate({ params: { path: { workspace: slug } } })}>
          {t(me.syncAll)}
        </Button>
      </div>
      {q.isError ? (
        <div className="p-5">
          <ErrorPanel title={t(me.syncLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : (
        <Table className="pv-10 pv-top" phoneWrap fit>
          <thead>
            <tr>
              <Th>{t(me.colProvider)}</Th>
              <Th>{t(me.colDirection)}</Th>
              <Th>{t(me.colPulled)}</Th>
              <Th>{t(me.colPushed)}</Th>
              <Th>{t(me.colLastSync)}</Th>
              <Th>{t(me.colStatus)}</Th>
            </tr>
          </thead>
          <tbody>
            {!data
              ? Array.from({ length: 3 }, (_, i) => (
                  <tr key={i}>
                    {[120, 60, 40, 40, 40, 60].map((w, j) => (
                      <Td key={j}>
                        <Skeleton w={w} />
                      </Td>
                    ))}
                  </tr>
                ))
              : data.rows.map((r) => {
                  const st = syncStatus[r.status]
                  return (
                    <tr key={`${r.providerId ?? r.provider}-${r.direction}`}>
                      <Td className="font-semibold">
                        {/* The name stays whole; a long scope ("enforced before sending") wraps so the status fits the card. */}
                        <span className="whitespace-nowrap">{r.provider}</span>
                        <div className="k">{r.scope}</div>
                      </Td>
                      <Td className="whitespace-nowrap">{t(direction[r.direction])}</Td>
                      <Td num>{count(r.pulled)}</Td>
                      <Td num={r.pushed !== undefined}>{r.pushed !== undefined ? count(r.pushed) : '–'}</Td>
                      <Td className="whitespace-nowrap">{when(r.lastSync)}</Td>
                      <Td>
                        <Badge tone={st.tone} title={r.detail}>
                          {t(st.text)}
                        </Badge>
                      </Td>
                    </tr>
                  )
                })}
          </tbody>
        </Table>
      )}
      <div className="flex flex-wrap gap-4 border-t border-line-2 px-5 py-3 text-[12.5px]">
        {data?.reasons
          .filter((r) => reasonBadge[r.reason])
          .sort((a, b) => Object.keys(reasonBadge).indexOf(a.reason) - Object.keys(reasonBadge).indexOf(b.reason))
          .map((r) => (
            <span key={r.reason}>
              <Badge tone={reasonBadge[r.reason]!.tone} className="h-5">
                {t(reasonBadge[r.reason]!.text)}
              </Badge>{' '}
              {count(r.count)}
            </span>
          ))}
        <Link to="/w/$workspace/audience/suppressions" params={{ workspace: slug }} className="ml-auto font-medium">
          {t(me.openSuppressions)}
        </Link>
      </div>
    </section>
  )
}

/** Open pixels and link rewriting added by providers, and what to do about them. */
function TrackingCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('providers:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/tracking-checks', { params: { path: { workspace: slug } } })
  const act = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/tracking', {
    onSuccess: (r, v) => {
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/tracking-checks'] })
      const a = v.body.action
      toast({
        kind: 'success',
        title:
          a === 'disable'
            ? tt(me.turnedOff, { provider: r.provider })
            : a === 'keep_provider'
              ? tt(me.keeping, { provider: r.provider })
              : a === 'use_reputecast'
                ? tt(me.oursAgain)
                : tt(me.checkedAgain),
        detail: r.detail,
      })
    },
    onError: (e) => toast({ kind: 'error', title: tt(me.trackingFailed), detail: errorMessage(e) }),
  })
  const run = (providerId: string, action: Schemas['ProviderTrackingActionInput']['action']) =>
    act.mutate({ params: { path: { workspace: slug, providerId } }, body: { action } })
  const items = q.data?.items
  return (
    <section className="card flex flex-col" aria-labelledby="pv-tracking">
      <div className="flex flex-col border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-tracking">
          {t(me.trackingTitle)}
        </h2>
        <span className="k">{t(me.trackingLine)}</span>
      </div>
      <div className="flex flex-col">
        {q.isError ? (
          <div className="p-5">
            <ErrorPanel title={t(me.trackingLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
              {errorMessage(q.error)}
            </ErrorPanel>
          </div>
        ) : !items ? (
          Array.from({ length: 3 }, (_, i) => (
            <div key={i} className={cn('flex gap-3 px-5 py-3.5', i < 2 && 'border-b border-line-2')}>
              <Skeleton w={40} h={22} r={999} />
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton w={140} h={14} />
                <Skeleton w={260} h={12} />
              </div>
            </div>
          ))
        ) : (
          items.map((c, i) => {
            const last = i === items.length - 1
            const warn = c.status === 'warn' && c.mode === 'reputecast'
            return (
              <div key={c.providerId} className={cn('flex items-start gap-3 px-5 py-3.5', !last && 'border-b border-line-2', warn && 'bg-warn-tint')}>
                {c.mode === 'provider' ? (
                  <Badge>{t(me.kept)}</Badge>
                ) : c.status === 'ok' ? (
                  <Badge tone="ok">{t(me.ok)}</Badge>
                ) : c.status === 'warn' ? (
                  <Badge tone="warn">{t(me.warn)}</Badge>
                ) : (
                  <Badge>{t(me.unknown)}</Badge>
                )}
                <div className={cn('flex flex-1 flex-col', warn ? 'gap-1.5' : 'gap-0.5')}>
                  <span className="font-semibold">
                    {warn
                      ? t(me.addsTracking, { what: c.openPixel ? 'pixel' : 'links', provider: c.provider, scope: c.scope })
                      : c.mode === 'provider'
                        ? t(me.providerTracks, { provider: c.provider })
                        : c.provider}
                  </span>
                  <span className="k">{c.detail || (c.status === 'unknown' ? t(me.unknownDetail) : '')}</span>
                  {warn ? (
                    <div className="flex flex-wrap gap-2">
                      {c.canDisable ? (
                        <Button size="sm" variant="primary" disabledReason={reason} onClick={() => run(c.providerId, 'disable')}>
                          {t(me.turnOff, { provider: c.provider })}
                        </Button>
                      ) : null}
                      <Button size="sm" disabledReason={reason} onClick={() => run(c.providerId, 'keep_provider')}>
                        {t(me.keepProvider, { provider: c.provider })}
                      </Button>
                    </div>
                  ) : c.mode === 'provider' ? (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" disabledReason={reason} onClick={() => run(c.providerId, 'use_reputecast')}>
                        {t(me.useOurs)}
                      </Button>
                    </div>
                  ) : c.status === 'unknown' ? (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" disabledReason={reason} onClick={() => run(c.providerId, 'recheck')}>
                        {t(me.checkAgain)}
                      </Button>
                    </div>
                  ) : null}
                </div>
              </div>
            )
          })
        )}
      </div>
    </section>
  )
}
