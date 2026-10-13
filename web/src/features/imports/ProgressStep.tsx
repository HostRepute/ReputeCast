import { Navigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { clock, clockSeconds, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, stepPath, useImportDetail, useRefreshImport, type Detail } from './ImportFrame'
import { PausedCard, ReadingSource } from './states'

const logTone: Record<string, Tone> = { start: 'ok', done: 'ok', sync: 'info', retry: 'warn', error: 'bad', info: 'neutral' }

/** "about 1 minute left" from the pace so far. */
function eta(d: Detail, now: number) {
  const p = d.progress
  if (!d.startedAt || p.processed <= 0 || p.total <= p.processed) return undefined
  const elapsed = now - new Date(d.startedAt).getTime()
  if (elapsed <= 0) return undefined
  const left = ((p.total - p.processed) * elapsed) / p.processed
  const mins = Math.ceil(left / 60_000)
  return mins <= 1 ? 'about 1 minute left' : `about ${mins} minutes left`
}

/** Step 5: the import running (MigrationProgress). */
export function ProgressStep({ importId }: { importId: string }) {
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  if (d && (d.status === 'done' || d.status === 'failed' || d.status === 'cancelled')) {
    return <Navigate to={stepPath.report} params={{ workspace: slug, importId }} replace />
  }
  if (d && d.status !== 'running' && d.status !== 'paused') {
    return <Navigate to={stepPath.preview} params={{ workspace: slug, importId }} replace />
  }
  return (
    <ImportFrame
      title={d ? `Import contacts · ${d.sourceName}` : 'Import contacts'}
      badge={<Badge tone="neutral">Step 5 of 5</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          Leave, keep running
        </ButtonLink>
      }
      step={5}
      hints={{ 5: 'Running in the background' }}
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the import" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} The import keeps running.
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title="Loading the import" />
      ) : (
        <Running d={d} now={q.dataUpdatedAt} />
      )}
    </ImportFrame>
  )
}

function Running({ d, now }: { d: Detail; now: number }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshImport()
  const cancel = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/cancel')
  const reason = usePermissionReason('subscribers:import')
  const [stopping, setStopping] = useState(false)
  const p = d.progress
  // Rounded like the board (67.8% → 68%), never 100% before the last row.
  const pct = p.total > 0 ? Math.min(p.processed < p.total ? 99 : 100, Math.round((p.processed / p.total) * 100)) : 0
  const left = eta(d, now)
  const tiles: [string, number, string][] = [
    ['Imported', p.imported, 'new, subscribed'],
    ['Merged', p.merged, 'existing contacts updated'],
    ['Suppressed', p.suppressed, 'never re-subscribed'],
    ['Pending re-permission', p.pending, 'old, never engaged'],
    ['Skipped', p.skipped, 'invalid, disposable, traps'],
  ]
  return (
    <>
      {d.status === 'paused' ? <PausedCard d={d} now={now} /> : null}
      <section className="card flex flex-col gap-3.5 px-6 py-5.5" aria-busy="true">
        <div className="flex flex-wrap items-center gap-3.5">
          <span className="spin h-8 w-8 border-[3px]" aria-hidden="true" />
          <div className="flex flex-[1_1_300px] flex-col gap-0.5">
            <span className="text-[20px] font-bold tracking-[-0.01em]" role="status">
              Importing from {d.sourceName} · {pct}%
            </span>
            <span className="k">
              {count(p.processed)} of {count(p.total)} rows processed{left ? ` · ${left}` : ''}
              {d.startedAt ? ` · started ${clock(d.startedAt)}${d.createdBy ? ` by ${d.createdBy}` : ''}` : ''}
            </span>
          </div>
          <Button className="stop" disabledReason={reason} onClick={() => setStopping(true)}>
            Stop import
          </Button>
        </div>
        <div className="bar h-3 rounded-[6px]" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} aria-label="Import progress">
          <i className="rounded-[6px]" style={{ width: `${pct}%`, background: 'var(--accent)' }} />
        </div>
        <span className="k">
          You can leave this page. The import keeps running and you'll get a notification with the report. Contacts already imported are usable right away.
        </span>
      </section>
      <div className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(170px,1fr))]">
        {tiles.map(([k, n, note]) => (
          <div key={k} className="card tile">
            <span className="k">{k}</span>
            <span className="n text-[22px]">{count(n)}</span>
            <span className="k">{note}</span>
          </div>
        ))}
      </div>
      <LiveLog d={d} />
      <ConfirmDialog
        open={stopping}
        onOpenChange={setStopping}
        badge="Stop import"
        title={`Stop importing from ${d.sourceName}?`}
        description={`${count(p.processed)} of ${count(p.total)} rows are already in; they stay. The rest isn't imported, and you can start a new import later.`}
        confirmLabel="Stop import"
        busyLabel="Stopping…"
        onConfirm={async () => {
          await cancel.mutateAsync({ params: { path: { workspace: slug, id: d.id } } })
          await refresh()
        }}
      />
    </>
  )
}

export function LiveLog({ d }: { d: Detail }) {
  const p = d.progress
  const entries: Schemas['ImportLogEntry'][] = d.log
  return (
    <section className="card flex flex-col px-5 py-4" aria-label="Live log">
      <div className="mb-1.5 flex items-center justify-between">
        <h2 className="h2">Live log</h2>
        {p.batchSize > 0 ? <span className="k">batches of {count(p.batchSize)}</span> : null}
      </div>
      <div aria-live="polite">
        {entries.length === 0 ? <p className="k m-0 py-2">Waiting for the first batch…</p> : null}
        {entries.map((e, i) => (
          <div key={`${e.at}-${i}`} className={`log ${i === 0 ? 'first' : ''}`}>
            <span className="mono k">{clockSeconds(e.at)}</span>
            {e.kind === 'running' ? (
              <span className="spin mt-0.5" aria-label="running" />
            ) : (
              <Badge tone={logTone[e.kind] ?? 'neutral'} className="h-5">
                {e.kind}
              </Badge>
            )}
            <span>{e.message}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
