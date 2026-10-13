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
import { cn } from '@/lib/cn'
import { clock, clockSeconds, count, dayMonthDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ProvidersTabs } from './common'

type EventType = Schemas['ProviderLogEvent']['type']

export interface EventsSearch {
  provider?: string
  type?: EventType
  q?: string
  page?: number
}

const PAGE_SIZE = 25

const typeBadge: Record<EventType, { tone: Tone; text: string }> = {
  delivered: { tone: 'ok', text: 'delivered' },
  deferred: { tone: 'warn', text: 'deferred' },
  hard_bounce: { tone: 'bad', text: 'hard bounce' },
  soft_bounce: { tone: 'warn', text: 'soft bounce' },
  complaint: { tone: 'bad', text: 'complaint' },
  rejected: { tone: 'bad', text: 'rejected' },
}

const typeOptions: { value: EventType; label: string }[] = [
  { value: 'delivered', label: 'Delivered' },
  { value: 'deferred', label: 'Deferred' },
  { value: 'hard_bounce', label: 'Hard bounce' },
  { value: 'soft_bounce', label: 'Soft bounce' },
  { value: 'complaint', label: 'Complaint' },
  { value: 'rejected', label: 'Rejected' },
]

/** 7.3 Events & sync: the raw event log, suppression sync and provider-side tracking. */
export function EventsPage({ search }: { search: EventsSearch }) {
  return (
    <Page title="Sending providers & routing" filters={<ProvidersTabs />}>
      <EventLog search={search} />
      <Grid min={420} gap={20}>
        <SyncCard />
        <TrackingCard />
      </Grid>
    </Page>
  )
}

function EventLog({ search }: { search: EventsSearch }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const page = search.page ?? 1
  const query = { provider: search.provider, type: search.type, q: search.q, page, pageSize: PAGE_SIZE }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/provider-events', { params: { path: { workspace: slug }, query } }, { placeholderData: keepPreviousData })
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
            Event log
          </h2>
          <span className="k">Every raw provider event beside the normalized ReputeCast event. The original code is always kept.</span>
        </div>
        <div className="flex flex-wrap gap-2">
          <select className="in pr-2.5 max-md:w-full" aria-label="Provider" value={search.provider ?? ''} onChange={(e) => go({ provider: e.target.value || undefined })}>
            <option value="">All sending providers</option>
            {providers.data?.providers.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <select className="in pr-2.5 max-md:w-full" aria-label="Event type" value={search.type ?? ''} onChange={(e) => go({ type: (e.target.value || undefined) as EventType | undefined })}>
            <option value="">All events</option>
            {typeOptions.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
          <SearchInput initial={search.q ?? ''} onSearch={(v) => go({ q: v || undefined })} />
          <ButtonAnchor href={exportUrl} newTab={false} download="provider-events.csv">
            Export CSV
          </ButtonAnchor>
        </div>
      </div>
      {q.isError ? (
        <div className="p-5">
          <ErrorPanel title="Couldn't load the event log" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Events keep being recorded.
          </ErrorPanel>
        </div>
      ) : (
        <Table className="pv-10 pv-top" phoneWrap aria-busy={q.isFetching || undefined}>
          <thead>
            <tr>
              <Th>Received</Th>
              <Th>Sending provider</Th>
              <Th>Raw event</Th>
              <Th>Normalized</Th>
              <Th>Original code · diagnostic</Th>
              <Th>Recipient</Th>
              <Th>Source</Th>
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
              <EmptyRow colSpan={7}>
                {search.q || search.type || search.provider ? 'No events match these filters.' : 'No provider events yet: they appear here as soon as a provider reports a delivery, bounce or complaint.'}
              </EmptyRow>
            ) : (
              data.items.map((e) => (
                <tr key={e.id}>
                  <Td className="mono whitespace-nowrap">{clockSeconds(e.receivedAt)}</Td>
                  <Td>{e.provider}</Td>
                  <Td>
                    <span className="pv-raw mono">{e.rawType}</span>
                  </Td>
                  <Td>
                    <Badge tone={typeBadge[e.type].tone}>{typeBadge[e.type].text}</Badge>
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
          {data ? `Showing ${count(data.items.length)} of ${count(data.total)} events · retained ${data.retentionDays} days` : 'Loading events…'}
        </span>
        <div className="flex gap-1.5">
          <Button size="sm" disabled={page <= 1} onClick={() => go({ page: page - 1 > 1 ? page - 1 : undefined })}>
            Previous
          </Button>
          <Button size="sm" disabled={page >= pages} onClick={() => go({ page: page + 1 })}>
            Next
          </Button>
        </div>
      </div>
    </section>
  )
}

function SearchInput({ initial, onSearch }: { initial: string; onSearch: (v: string) => void }) {
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
      placeholder="Recipient or message id"
      aria-label="Search events"
      value={value}
      onChange={(e) => setValue(e.target.value)}
    />
  )
}

const direction: Record<Schemas['SuppressionSyncRow']['direction'], string> = {
  two_way: 'Two-way',
  push: 'Push only',
  pull: 'Pull only',
  off: 'Off',
  local: 'Local only',
  import: 'One-time pull',
}

const syncStatus: Record<Schemas['SuppressionSyncRow']['status'], { tone: Tone; text: string }> = {
  in_sync: { tone: 'ok', text: 'In sync' },
  syncing: { tone: 'info', text: 'Syncing' },
  error: { tone: 'bad', text: 'Error' },
  local: { tone: 'neutral', text: 'Enforced locally' },
  done: { tone: 'neutral', text: 'Done' },
  off: { tone: 'neutral', text: 'Off' },
}

const reasonBadge: Record<string, { tone: Tone; text: string }> = {
  hard_bounce: { tone: 'bad', text: 'hard bounce' },
  complaint: { tone: 'bad', text: 'complaint' },
  unsubscribe: { tone: 'neutral', text: 'unsubscribed' },
  manual: { tone: 'neutral', text: 'manual' },
}

/** The shared suppression list and how each provider stays in sync with it. */
function SyncCard() {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const reason = usePermissionReason('providers:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/suppression-sync', { params: { path: { workspace: slug } } })
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/suppression-sync/run', {
    onSuccess: (r) => {
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/suppression-sync'] })
      toast({ kind: 'success', title: r.queued ? `Syncing ${r.queued} provider${r.queued === 1 ? '' : 's'}` : 'Nothing to sync', detail: 'Pulled addresses join the shared list; the list is pushed back out.' })
    },
    onError: (e) => toast({ kind: 'error', title: "The sync didn't start", detail: errorMessage(e) }),
  })
  const data = q.data
  const when = (t?: string) => {
    if (!t) return '–'
    const d = new Date(t)
    return now.getTime() - d.getTime() > 20 * 3600_000 ? dayMonthDate(d) : clock(d)
  }
  return (
    <section className="card flex flex-col" aria-labelledby="pv-sync">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="pv-sync">
            Shared suppression list sync
          </h2>
          <span className="k">{data ? `${count(data.total)} addresses · a bounce at one provider stops mail everywhere` : 'Loading…'}</span>
        </div>
        <Button size="sm" variant="primary" disabledReason={reason} loading={run.isPending} onClick={() => run.mutate({ params: { path: { workspace: slug } } })}>
          Sync all now
        </Button>
      </div>
      {q.isError ? (
        <div className="p-5">
          <ErrorPanel title="Couldn't load the sync status" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : (
        <Table className="pv-10 pv-top" phoneWrap>
          <thead>
            <tr>
              <Th>Sending provider</Th>
              <Th>Direction</Th>
              <Th>Pulled in</Th>
              <Th>Pushed out</Th>
              <Th>Last sync</Th>
              <Th>Status</Th>
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
                      <Td className="font-semibold whitespace-nowrap">
                        {r.provider}
                        <div className="k">{r.scope}</div>
                      </Td>
                      <Td className="whitespace-nowrap">{direction[r.direction]}</Td>
                      <Td num>{count(r.pulled)}</Td>
                      <Td num={r.pushed !== undefined}>{r.pushed !== undefined ? count(r.pushed) : '–'}</Td>
                      <Td className="whitespace-nowrap">{when(r.lastSync)}</Td>
                      <Td>
                        <Badge tone={st.tone} title={r.detail}>
                          {st.text}
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
                {reasonBadge[r.reason]!.text}
              </Badge>{' '}
              {count(r.count)}
            </span>
          ))}
        <Link to="/w/$workspace/audience/suppressions" params={{ workspace: slug }} className="ml-auto font-medium">
          Open suppression list
        </Link>
      </div>
    </section>
  )
}

/** Open pixels and link rewriting added by providers, and what to do about them. */
function TrackingCard() {
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
        title: a === 'disable' ? `Turned off in ${r.provider}` : a === 'keep_provider' ? `Keeping ${r.provider}'s tracking` : a === 'use_reputecast' ? 'ReputeCast tracks opens and clicks again' : 'Checked again',
        detail: r.detail,
      })
    },
    onError: (e) => toast({ kind: 'error', title: "Tracking didn't change", detail: errorMessage(e) }),
  })
  const run = (providerId: string, action: Schemas['ProviderTrackingActionInput']['action']) =>
    act.mutate({ params: { path: { workspace: slug, providerId } }, body: { action } })
  const items = q.data?.items
  return (
    <section className="card flex flex-col" aria-labelledby="pv-tracking">
      <div className="flex flex-col border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-tracking">
          Provider-side tracking check
        </h2>
        <span className="k">ReputeCast tracks opens and clicks itself. A provider doing it too double-counts and rewrites links twice.</span>
      </div>
      <div className="flex flex-col">
        {q.isError ? (
          <div className="p-5">
            <ErrorPanel title="Couldn't load the tracking checks" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
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
          items.map((t, i) => {
            const last = i === items.length - 1
            const warn = t.status === 'warn' && t.mode === 'reputecast'
            const what = t.openPixel ? 'its own open pixel' : 'its own link tracking'
            return (
              <div key={t.providerId} className={cn('flex items-start gap-3 px-5 py-3.5', !last && 'border-b border-line-2', warn && 'bg-warn-tint')}>
                {t.mode === 'provider' ? (
                  <Badge>Kept</Badge>
                ) : t.status === 'ok' ? (
                  <Badge tone="ok">OK</Badge>
                ) : t.status === 'warn' ? (
                  <Badge tone="warn">Warn</Badge>
                ) : (
                  <Badge>Unknown</Badge>
                )}
                <div className={cn('flex flex-1 flex-col', warn ? 'gap-1.5' : 'gap-0.5')}>
                  <span className="font-semibold">
                    {warn ? `${t.provider} adds ${what} on the ${t.scope}` : t.mode === 'provider' ? `${t.provider} tracks opens and clicks; ReputeCast's are ignored` : t.provider}
                  </span>
                  <span className="k">{t.detail || (t.status === 'unknown' ? "ReputeCast couldn't read this setting yet." : '')}</span>
                  {warn ? (
                    <div className="flex flex-wrap gap-2">
                      {t.canDisable ? (
                        <Button size="sm" variant="primary" disabledReason={reason} onClick={() => run(t.providerId, 'disable')}>
                          Turn off in {t.provider} via API
                        </Button>
                      ) : null}
                      <Button size="sm" disabledReason={reason} onClick={() => run(t.providerId, 'keep_provider')}>
                        Keep {t.provider}&apos;s tracking, ignore ours
                      </Button>
                    </div>
                  ) : t.mode === 'provider' ? (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" disabledReason={reason} onClick={() => run(t.providerId, 'use_reputecast')}>
                        Use ReputeCast&apos;s tracking again
                      </Button>
                    </div>
                  ) : t.status === 'unknown' ? (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" disabledReason={reason} onClick={() => run(t.providerId, 'recheck')}>
                        Check again
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
