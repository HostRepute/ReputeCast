import { Navigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { clock, clockSeconds, count, pct as percent } from '@/lib/format'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, stepPath, useImportDetail, useRefreshImport, type Detail } from './ImportFrame'
import { PausedCard, ReadingSource } from './states'
import { m } from './messages'

const logTone: Record<string, Tone> = { start: 'ok', done: 'ok', sync: 'info', retry: 'warn', error: 'bad', info: 'neutral' }

/** "about 1 minute left" from the pace so far. */
function eta(d: Detail, now: number) {
  const p = d.progress
  if (!d.startedAt || p.processed <= 0 || p.total <= p.processed) return undefined
  const elapsed = now - new Date(d.startedAt).getTime()
  if (elapsed <= 0) return undefined
  const left = ((p.total - p.processed) * elapsed) / p.processed
  const mins = Math.ceil(left / 60_000)
  return tt(m.etaMinutes, { n: Math.max(1, mins) })
}

/** Step 5: the import running (MigrationProgress). */
export function ProgressStep({ importId }: { importId: string }) {
  const t = useT()
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
      title={d ? t(m.titleFrom, { source: d.sourceName }) : t(m.title)}
      badge={<Badge tone="neutral">{t(m.stepOf, { step: 5, total: 5 })}</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          {t(m.leaveRunning)}
        </ButtonLink>
      }
      step={5}
      hints={{ 5: t(m.hintRunning) }}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailedThe)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.keepsRunning)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title={t(m.loading)} />
      ) : (
        <Running d={d} now={q.dataUpdatedAt} />
      )}
    </ImportFrame>
  )
}

function Running({ d, now }: { d: Detail; now: number }) {
  const t = useT()
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
    [t(m.tileImported), p.imported, t(m.tileImportedNote)],
    [t(m.tileMerged), p.merged, t(m.tileMergedUpdated)],
    [t(m.tileSuppressed), p.suppressed, t(m.tileNeverResubscribed)],
    [t(m.tilePending), p.pending, t(m.tilePendingNote)],
    [t(m.tileSkipped), p.skipped, t(m.tileSkippedNote)],
  ]
  const line = [t(m.processed, { processed: count(p.processed), total: count(p.total) })]
  if (left) line.push(left)
  if (d.startedAt) line.push(d.createdBy ? t(m.startedBy, { time: clock(d.startedAt), name: d.createdBy }) : t(m.startedAt, { time: clock(d.startedAt) }))
  return (
    <>
      {d.status === 'paused' ? <PausedCard d={d} now={now} /> : null}
      <section className="card flex flex-col gap-3.5 px-6 py-5.5" aria-busy="true">
        <div className="flex flex-wrap items-center gap-3.5">
          <span className="spin h-8 w-8 border-[3px]" aria-hidden="true" />
          <div className="flex flex-[1_1_300px] flex-col gap-0.5">
            <span className="text-[20px] font-bold tracking-[-0.01em]" role="status">
              {t(m.importingFrom, { source: d.sourceName, percent: percent(pct, 0) })}
            </span>
            <span className="k">{line.join(' · ')}</span>
          </div>
          <Button className="stop" disabledReason={reason} onClick={() => setStopping(true)}>
            {t(m.stopImport)}
          </Button>
        </div>
        <div className="bar h-3 rounded-[6px]" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} aria-label={t(m.progress)}>
          <i className="rounded-[6px]" style={{ width: `${pct}%`, background: 'var(--accent)' }} />
        </div>
        <span className="k">{t(m.canLeave)}</span>
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
        badge={t(m.stopImport)}
        title={t(m.stopTitle, { source: d.sourceName })}
        description={t(m.stopBody, { processed: count(p.processed), total: count(p.total) })}
        confirmLabel={t(m.stopImport)}
        busyLabel={t(m.stopping)}
        onConfirm={async () => {
          await cancel.mutateAsync({ params: { path: { workspace: slug, id: d.id } } })
          await refresh()
        }}
      />
    </>
  )
}

export function LiveLog({ d }: { d: Detail }) {
  const t = useT()
  const p = d.progress
  const entries: Schemas['ImportLogEntry'][] = d.log
  return (
    <section className="card flex flex-col px-5 py-4" aria-label={t(m.liveLog)}>
      <div className="mb-1.5 flex items-center justify-between">
        <h2 className="h2">{t(m.liveLog)}</h2>
        {p.batchSize > 0 ? <span className="k">{t(m.batchesOf, { count: count(p.batchSize) })}</span> : null}
      </div>
      <div aria-live="polite">
        {entries.length === 0 ? <p className="k m-0 py-2">{t(m.firstBatch)}</p> : null}
        {entries.map((e, i) => (
          <div key={`${e.at}-${i}`} className={`log ${i === 0 ? 'first' : ''}`}>
            <span className="mono k">{clockSeconds(e.at)}</span>
            {e.kind === 'running' ? (
              <span className="spin mt-0.5" aria-label={t(m.running)} />
            ) : (
              <Badge tone={logTone[e.kind] ?? 'neutral'} className="h-5">
                {t(m.logKind, { kind: e.kind })}
              </Badge>
            )}
            <span>{e.message}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
