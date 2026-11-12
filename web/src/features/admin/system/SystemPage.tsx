import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { KeyValues, StatTile } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable } from '@/features/access/parts'
import { T, t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { byteSize, clock, count, logTime } from '@/lib/format'
import { AdminPage } from '../AdminPage'
import { m } from './messages'
import { Pager } from '@/components/ui/Pager'

type View = Schemas['SystemView']
type Job = Schemas['SystemJob']
type Show = '' | 'retrying' | 'given_up'

const systemKey = ['get', '/api/v1/admin/system']
const jobsKey = ['get', '/api/v1/admin/system/jobs']

/** What runs on each of the queues ReputeCast creates. */
const queueWhat: Record<string, MessageDescriptor> = {
  sending: m.queueSending,
  events: m.queueEvents,
  default: m.queueDefault,
  system: m.queueSystem,
  maintenance: m.queueMaintenance,
}

/** "4 s", "2 min 05 s", "1 h 12 min". */
export function waited(seconds: number): string {
  const s = Math.round(seconds)
  if (s < 60) return tt(m.seconds, { n: s })
  if (s < 3600) return tt(m.minutes, { m: Math.floor(s / 60), s: String(s % 60).padStart(2, '0') })
  return tt(m.hours, { h: Math.floor(s / 3600), m: Math.floor((s % 3600) / 60) })
}

/** The commands the footnote names; never translated. */
const commands = { doctor: 'reputecast doctor', reseal: 'reputecast secrets reseal' }

/** "webhooks.deliver #48213" */
const jobName = (j: Job) => `${j.kind} #${j.id}`

/**
 * Instance admin › System (InstanceSystem; review 2026-11 OP-10): failed
 * background jobs with retry, cancel and delete; the queues and their lag;
 * the database's largest tables; the file storage (folder or bucket).
 */
export function SystemPage() {
  const t = useT()
  // Read again every 30 seconds while the tab is open.
  const q = $api.useQuery('get', '/api/v1/admin/system', {}, { refetchInterval: 30_000 })
  return (
    <AdminPage documentTitle={t(m.documentTitle)}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <div className="col gap-3" aria-busy="true">
          <span className="vh" role="status">
            {t(m.loading)}
          </span>
          <Skeleton w="100%" h={96} />
          <Skeleton w="100%" h={240} />
        </div>
      ) : (
        <>
          <Tiles v={q.data} />
          <FailedJobs />
          <Queues v={q.data} />
          <div className="grid items-start gap-5 [grid-template-columns:repeat(auto-fit,minmax(min(420px,100%),1fr))]">
            <Tables v={q.data} />
            <Files v={q.data} />
          </div>
          <Note tone="neutral" role="note">
            <span className="col gap-1">
              <span className="font-semibold">{t(m.cliTitle)}</span>
              <span>
                <T msg={m.cliBody} values={{ doctor: <code className="mono">{commands.doctor}</code>, reseal: <code className="mono">{commands.reseal}</code> }} />
              </span>
            </span>
          </Note>
        </>
      )}
    </AdminPage>
  )
}

function Tiles({ v }: { v: View }) {
  const t = useT()
  const failed = v.jobs.retrying + v.jobs.givenUp
  return (
    <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(min(160px,100%),1fr))]">
      <StatTile
        className="card"
        label={t(m.tileWaiting)}
        value={count(v.jobs.waiting)}
        delta={v.jobs.waiting ? t(m.tileWaitingLag, { seconds: waited(v.jobs.lagSeconds) }) : t(m.tileWaitingNone)}
      />
      <StatTile className="card" label={t(m.tileRunning)} value={count(v.jobs.running)} delta={t(m.tileRunningSub)} />
      <StatTile
        className="card"
        label={t(m.tileFailed)}
        value={<span className={failed ? 'text-bad-fg' : undefined}>{count(failed)}</span>}
        delta={t(m.tileFailedSub, { retrying: v.jobs.retrying, givenUp: v.jobs.givenUp })}
      />
      <StatTile
        className="card"
        label={t(m.tileDatabase)}
        value={byteSize(v.database.bytes)}
        delta={t(m.tileDatabaseSub, { version: v.database.version.split(' ')[0] ?? '', connections: v.database.connections, max: v.database.maxConnections })}
      />
      <StatTile
        className="card"
        label={t(m.tileFiles)}
        value={v.files.missing ? '—' : byteSize(v.files.bytes)}
        delta={v.files.missing || !v.files.files ? t(m.tileFilesNone) : t(m.tileFilesSub, { count: v.files.files })}
      />
    </div>
  )
}

// ── Failed jobs ──────────────────────────────────────────────────────────

function FailedJobs() {
  const t = useT()
  const qc = useQueryClient()
  const [show, setShow] = useState<Show>('')
  const [busy, setBusy] = useState<number>()
  const [stopping, setStopping] = useState<Job>()
  const q = $api.useInfiniteQuery(
    'get',
    '/api/v1/admin/system/jobs',
    { params: { query: show ? { state: show } : {} } },
    { pageParamName: 'cursor', initialPageParam: '', getNextPageParam: (last) => last.nextCursor || undefined, refetchInterval: 30_000 },
  )
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: jobsKey }), qc.invalidateQueries({ queryKey: systemKey })])
  const retry = async (j: Job) => {
    setBusy(j.id)
    try {
      await http.POST('/api/v1/admin/system/jobs/{jobId}/retry', { params: { path: { jobId: j.id } } })
      toast({ kind: 'success', title: tt(m.retried, { job: jobName(j) }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <section className="card" aria-labelledby="sys-failed-h">
      <CardSection
        title={<span id="sys-failed-h">{q.data && !q.hasNextPage ? t(m.failedTitleCount, { count: count(items.length) }) : t(m.failedTitle)}</span>}
        caption={t(m.failedCaption)}
        actions={
          <label className="flex items-center gap-2">
            <span className="vh">{t(m.failedShow)}</span>
            <select className="in w-auto max-md:w-full" value={show} onChange={(e) => setShow(e.target.value as Show)}>
              <option value="">{t(m.showAll)}</option>
              <option value="retrying">{t(m.showRetrying)}</option>
              <option value="given_up">{t(m.showGivenUp)}</option>
            </select>
          </label>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <div className="pad" aria-busy="true">
          <Skeleton w="100%" h={20} />
        </div>
      ) : !items.length ? (
        <p className="pad k m-0">{show ? t(m.noFailedState) : t(m.noFailed)}</p>
      ) : (
        <>
          <AccessTable caption={t(m.failedTable)}>
            <thead>
              <tr>
                <Th>{t(m.colJob)}</Th>
                <Th>{t(m.colWorkspace)}</Th>
                <Th>{t(m.colState)}</Th>
                <Th>{t(m.colAttempts)}</Th>
                <Th>{t(m.colError)}</Th>
                <Th>{t(m.colFailed)}</Th>
                <Th>
                  <span className="vh">{t(m.colActions)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {items.map((j) => (
                <tr key={j.id}>
                  <Td>
                    <span className="col gap-0.5">
                      <span className="mono font-semibold">{j.kind}</span>
                      <span className="k">{t(m.jobWhere, { queue: j.queue, id: String(j.id) })}</span>
                    </span>
                  </Td>
                  <Td>{j.workspace?.name || <span className="k">{t(m.noWorkspace)}</span>}</Td>
                  <Td>
                    {j.state === 'retrying' ? (
                      <Badge tone="warn" dot>
                        {t(m.stateRetrying)}
                      </Badge>
                    ) : (
                      <Badge tone="bad" dot>
                        {t(m.stateGivenUp)}
                      </Badge>
                    )}
                  </Td>
                  <Td>{t(m.attempts, { attempt: j.attempt, max: j.maxAttempts })}</Td>
                  <Td className="min-w-[240px] max-w-[380px] whitespace-normal">
                    {j.error ? <span className="mono block break-words text-[12px] text-ink-2">{j.error}</span> : <span className="k">{t(m.noError)}</span>}
                    {j.state === 'retrying' && j.nextAt ? <span className="k block">{t(m.nextTry, { time: clock(j.nextAt) })}</span> : null}
                  </Td>
                  <Td className="whitespace-nowrap">{j.failedAt ? logTime(j.failedAt) : '—'}</Td>
                  <Td>
                    <span className="flex gap-1.5">
                      <Button size="sm" loading={busy === j.id} aria-label={t(m.retryLabel, { job: jobName(j) })} onClick={() => void retry(j)}>
                        {j.state === 'retrying' ? t(m.retryNow) : t(m.retry)}
                      </Button>
                      <Button
                        size="sm"
                        variant={j.state === 'given_up' ? 'danger' : undefined}
                        aria-label={t(j.state === 'retrying' ? m.cancelLabel : m.deleteLabel, { job: jobName(j) })}
                        onClick={() => setStopping(j)}
                      >
                        {j.state === 'retrying' ? t(m.cancel) : t(m.delete)}
                      </Button>
                    </span>
                  </Td>
                </tr>
              ))}
            </tbody>
          </AccessTable>
          {q.hasNextPage ? (
            <div className="pad flex justify-center border-t border-line-2">
              <Pager query={q} label={t(m.loadMore)} size="sm" />
            </div>
          ) : null}
        </>
      )}
      <p className="pad k m-0 border-t border-line-2">{t(m.failedFootnote)}</p>
      {stopping ? <StopDialog job={stopping} onClose={() => setStopping(undefined)} onDone={refresh} /> : null}
    </section>
  )
}

/** Cancel a retrying job, or delete one that gave up. */
function StopDialog({ job, onClose, onDone }: { job: Job; onClose: () => void; onDone: () => Promise<unknown> }) {
  const t = useT()
  const cancel = job.state === 'retrying'
  const name = jobName(job)
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={t(cancel ? m.cancelBadge : m.deleteBadge)}
      title={t(cancel ? m.cancelTitle : m.deleteTitle, { job: name })}
      description={t(cancel ? m.cancelBody : m.deleteBody)}
      confirmLabel={t(cancel ? m.cancelBadge : m.deleteBadge)}
      cancelLabel={t(m.keepIt)}
      busyLabel={t(cancel ? m.cancelling : m.deleting)}
      onConfirm={async () => {
        await http.POST('/api/v1/admin/system/jobs/{jobId}/discard', { params: { path: { jobId: job.id } } })
        toast({ kind: cancel ? 'success' : 'deleted', title: tt(cancel ? m.cancelled : m.deleted, { job: name }) })
        await onDone()
        onClose()
      }}
    />
  )
}

// ── Queues, tables, files ────────────────────────────────────────────────

function Queues({ v }: { v: View }) {
  const t = useT()
  return (
    <section className="card" aria-labelledby="sys-queues-h">
      <CardSection title={<span id="sys-queues-h">{t(m.queuesTitle)}</span>} caption={t(m.queuesCaption)} />
      <AccessTable caption={t(m.queuesTable)} className="max-[1100px]:[&_td:nth-child(n+3)]:whitespace-nowrap">
        <thead>
          <tr>
            <Th>{t(m.colQueue)}</Th>
            <Th>{t(m.colWhat)}</Th>
            <Th align="right">{t(m.colWaiting)}</Th>
            <Th align="right">{t(m.colScheduled)}</Th>
            <Th align="right">{t(m.colRunning)}</Th>
            <Th align="right">{t(m.colLag)}</Th>
          </tr>
        </thead>
        <tbody>
          {v.queues.map((q) => (
            <tr key={q.name}>
              <Td className="mono">{q.name}</Td>
              <Td>{queueWhat[q.name] ? t(queueWhat[q.name]!) : '—'}</Td>
              <Td align="right" className="whitespace-nowrap">{count(q.waiting)}</Td>
              <Td align="right" className="whitespace-nowrap">{count(q.scheduled)}</Td>
              <Td align="right" className="whitespace-nowrap">{count(q.running)}</Td>
              <Td align="right" className="whitespace-nowrap">{q.lagSeconds === undefined ? '—' : waited(q.lagSeconds)}</Td>
            </tr>
          ))}
        </tbody>
      </AccessTable>
    </section>
  )
}

function Tables({ v }: { v: View }) {
  const t = useT()
  const d = v.database
  return (
    <section className="card" aria-labelledby="sys-tables-h">
      <CardSection
        title={<span id="sys-tables-h">{t(m.tablesTitle)}</span>}
        caption={t(m.tablesCaption, { size: byteSize(d.bytes), version: String(d.schemaVersion), current: d.schemaCurrent ? 'yes' : 'no' })}
      />
      {/* The numbers keep one line ("16 / MB" made every row two lines on phones):
          AccessTable's td rule outranks the cells' own whitespace-nowrap. */}
      <AccessTable caption={t(m.tablesTable)} className="max-[1100px]:[&_td:not(:first-child)]:whitespace-nowrap">
        <thead>
          <tr>
            <Th>{t(m.colTable)}</Th>
            <Th align="right">{t(m.colRows)}</Th>
            <Th align="right">{t(m.colSize)}</Th>
            <Th align="right">{t(m.colIndexes)}</Th>
          </tr>
        </thead>
        <tbody>
          {d.tables.map((r) => (
            <tr key={r.name}>
              <Td className="mono">{r.name}</Td>
              <Td align="right" className="whitespace-nowrap">{count(r.rows)}</Td>
              <Td align="right" className="whitespace-nowrap">{byteSize(r.bytes)}</Td>
              <Td align="right" className="whitespace-nowrap">{byteSize(r.indexBytes)}</Td>
            </tr>
          ))}
        </tbody>
      </AccessTable>
    </section>
  )
}

function Files({ v }: { v: View }) {
  const t = useT()
  const f = v.files
  const s3 = f.backend === 's3'
  let holds: ReactNode
  if (f.missing) holds = t(m.filesMissing)
  else if (f.error) holds = <span className="text-bad-fg">{t(m.filesError, { error: f.error })}</span>
  else holds = t(f.partial ? m.filesHoldsPartial : m.filesHoldsValue, { count: f.files, size: byteSize(f.bytes) })
  // Where the files are: the folder, or the bucket (never its keys).
  const where: [ReactNode, ReactNode][] = s3
    ? [
        [t(m.filesBucket), <span key="bucket" className="mono break-all">{f.bucket}</span>],
        ...(f.prefix ? [[t(m.filesPrefix), <span key="prefix" className="mono break-all">{f.prefix}</span>] as [ReactNode, ReactNode]] : []),
        [
          t(m.filesEndpoint),
          f.endpoint ? (
            <span key="endpoint" className="mono break-all">
              {f.endpoint}
            </span>
          ) : f.region ? (
            t(m.filesEndpointAwsRegion, { region: f.region })
          ) : (
            t(m.filesEndpointAws)
          ),
        ],
      ]
    : [[t(m.filesFolder), <span key="dir" className="mono break-all">{f.dir}</span>]]
  return (
    <section className="card" aria-labelledby="sys-files-h">
      <CardSection title={<span id="sys-files-h">{t(m.filesTitle)}</span>} caption={t(s3 ? m.filesCaptionS3 : m.filesCaption)} />
      <div className="pad">
        <KeyValues
          className="m-0"
          items={[
            [t(m.filesStorage), t(s3 ? m.filesStorageS3 : m.filesStorageLocal)],
            ...where,
            [t(m.filesHolds), holds],
            [t(m.filesCounted), t(m.filesCountedAt, { time: clock(f.countedAt) })],
            [t(m.filesShared), t(s3 ? m.filesSharedValueS3 : m.filesSharedValue)],
          ]}
        />
      </div>
    </section>
  )
}
