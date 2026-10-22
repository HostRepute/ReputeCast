import { Navigate } from '@tanstack/react-router'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { TableScroll, Th } from '@/components/ui/Table'
import { NotifuseTemplatesDone } from '@/features/transactional-import/NotifuseOption'
import { WelcomePromptCard } from '@/features/welcome-prompt/WelcomePromptCard'
import { andList, clock, count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ImportFrame, stepPath, currentStep, useImportDetail, type Detail } from './ImportFrame'
import { LiveLog } from './ProgressStep'
import { downloadImportFile } from './SourceStep'
import { ReadingSource } from './states'
import { $api, errorMessage } from '@/api/client'
import { serviceReasonKeys, serviceReasonRow } from '@/features/verification/ImportCheck'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './messages'

/** "3 minutes 10 seconds" */
function took(ms: number) {
  const s = Math.max(1, Math.round(ms / 1000))
  const min = Math.floor(s / 60)
  const rest = s % 60
  if (min === 0) return t(m.tookSeconds, { s: rest })
  return rest ? t(m.tookBoth, { min, s: rest }) : t(m.tookMinutes, { min })
}

/** Why rows were skipped or changed, in the report's words. */
function reasonRow(key: string, d: Detail): [string, string] | null {
  switch (key) {
    case 'invalid':
      return [t(m.reasonInvalid), t(m.reasonInvalidWhat)]
    case 'disposable':
      return [t(m.reasonDisposable), t(m.reasonAsChosen)]
    case 'trap':
      return [t(m.reasonTrap), t(m.reasonTrapWhat)]
    case 'typo_fixed':
      return [t(m.reasonTypoFixed), t(m.reasonTypoFixedWhat)]
    case 'typo':
      return [t(m.reasonTypo), t(m.reasonAsChosen)]
    case 'role':
      return d.options.risky.role === 'skip'
        ? [t(m.reasonRole), t(m.reasonAsChosen)]
        : d.options.risky.role === 'import'
          ? [t(m.reasonRole), t(m.reasonRoleImported)]
          : [t(m.reasonRole), t(m.reasonRoleExcluded)]
    case 'stale':
      return d.options.risky.stale === 'skip' ? [t(m.reasonStale), t(m.reasonAsChosen)] : [t(m.reasonStale), t(m.reasonStaleWhat)]
    case 'already_suppressed':
      return [t(m.reasonAlready), t(m.reasonAlreadyWhat, { source: d.sourceName })]
    case 'status_skipped':
      return [t(m.reasonStatusSkipped), t(m.reasonStatusSkippedWhat)]
  }
  return null
}

const reasonOrder = ['invalid', 'disposable', 'trap', 'typo_fixed', 'typo', 'role', 'stale', 'already_suppressed', 'status_skipped', ...serviceReasonKeys]

/** The import summary (MigrationDone). */
export function ReportStep({ importId }: { importId: string }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  if (d && currentStep(d) !== 'report') {
    return <Navigate to={stepPath[currentStep(d)]} params={{ workspace: slug, importId }} replace />
  }
  const badge =
    d?.status === 'done' ? (
      <Badge tone="ok">{tr(m.badgeDone)}</Badge>
    ) : d?.status === 'cancelled' ? (
      <Badge tone="warn">{tr(m.badgeStopped)}</Badge>
    ) : d?.status === 'failed' ? (
      <Badge tone="bad">{tr(m.badgeFailed)}</Badge>
    ) : undefined
  return (
    <ImportFrame title={d ? tr(m.titleFrom, { source: d.sourceName }) : tr(m.title)} documentTitle={tr(m.summaryTitle)} badge={badge} narrow>
      {q.isError ? (
        <ErrorPanel title={tr(m.loadFailedThe)} actions={<Button onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title={tr(m.loadingSummary)} />
      ) : (
        <Summary d={d} now={q.dataUpdatedAt} />
      )}
    </ImportFrame>
  )
}

function Summary({ d, now }: { d: Detail; now: number }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const flows = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations', { params: { path: { workspace: slug } } }, { retry: false })
  const s = d.stats
  const imported = (s?.imported ?? d.progress.imported) + (s?.merged ?? d.progress.merged)
  const suppressed = s?.suppressed ?? d.progress.suppressed
  const pending = s?.pending ?? d.progress.pending
  const skipped = s?.skipped ?? d.progress.skipped
  const reasons = s?.reasons ?? {}
  const by = s?.suppressedBy ?? {}
  const lists = s?.lists ?? []
  const finished = d.finishedAt
  // "09:54" today, "18 Sep, 16:40" before.
  const finishedAt = finished ? (dayMonthDate(finished) === dayMonthDate(now) ? clock(finished) : `${dayMonthDate(finished)}, ${clock(finished)}`) : ''
  const listsLine =
    lists.length === 1
      ? tr(m.listOne, { name: lists[0]!.name, count: count(lists[0]!.contacts) })
      : lists.length > 1
        ? tr(m.listsMany, {
            lists: andList(lists.map((l, i) => tr(i === 0 ? m.listFirst : m.listNext, { name: l.name, count: count(l.contacts) }))),
          })
        : ''
  const skippedParts = [
    reasons.invalid || reasons.invalid_service ? tr(m.skippedInvalid, { count: count((reasons.invalid ?? 0) + (reasons.invalid_service ?? 0)) }) : '',
    reasons.disposable ? tr(m.skippedDisposable, { count: count(reasons.disposable) }) : '',
    reasons.trap ? tr(m.skippedTrap, { count: count(reasons.trap) }) : '',
  ].filter(Boolean)
  const known = (reasons.invalid ?? 0) + (reasons.invalid_service ?? 0) + (reasons.disposable ?? 0) + (reasons.trap ?? 0)
  if (skipped - known > 0 && skippedParts.length > 0) skippedParts.push(tr(m.skippedOther, { count: count(skipped - known) }))
  const rows = reasonOrder
    .filter((k) => (reasons[k] ?? 0) > 0)
    .map((k) => [k, reasons[k]!, reasonRow(k, d) ?? serviceReasonRow(k)] as const)
    .filter((r): r is readonly [string, number, [string, string]] => r[2] !== null)
  const flow = flows.data?.items?.find((f) => /re-?permission/i.test(f.name))
  const firstList = lists[0]?.name
  const tiles: [string, number, string, string?][] = [
    [tr(m.tileImported), s?.imported ?? d.progress.imported, tr(m.tileImportedNote), '#86EFAC'],
    [tr(m.tileMerged), s?.merged ?? d.progress.merged, tr(m.tileExistingUpdated)],
    [
      tr(m.tileSuppressed),
      suppressed,
      [
        by.unsubscribe ? tr(m.suppressedUnsubscribed, { count: count(by.unsubscribe) }) : '',
        by.hard_bounce ? tr(m.suppressedBounced, { count: count(by.hard_bounce) }) : '',
        by.complaint ? tr(m.suppressedComplaints, { count: count(by.complaint) }) : '',
      ]
        .filter(Boolean)
        .join(' · ') || tr(m.tileNeverResubscribed),
      '#FCA5A5',
    ],
    [tr(m.tilePending), pending, s?.confirmsSent ? tr(m.pendingSent) : tr(m.pendingWaiting), '#FCD34D'],
    [tr(m.tileSkipped), skipped, skippedParts.join(' · ') || tr(m.tileSkippedNote)],
  ]
  const headline =
    d.status === 'done'
      ? tr(m.headlineDone, { n: imported, count: count(imported), source: d.sourceName })
      : d.status === 'cancelled'
        ? tr(m.headlineStopped, { n: imported, count: count(imported), source: d.sourceName })
        : tr(m.headlineFailed, { source: d.sourceName })
  const duration = s?.durationMs ? took(s.durationMs) : undefined
  const doneLead = [
    finished
      ? duration
        ? tr(m.finishedAtTook, { time: finishedAt, took: duration })
        : tr(m.finishedAt, { time: finishedAt })
      : duration
        ? tr(m.finishedTook, { took: duration })
        : tr(m.finished),
    listsLine,
    suppressed > 0 ? tr(m.consentCarriedSuppressed, { n: suppressed, count: count(suppressed) }) : tr(m.consentCarried),
  ]
    .filter(Boolean)
    .join(' ')
  return (
    <>
      <div className="flex flex-wrap items-center gap-4.5">
        {d.status === 'done' ? (
          <svg width="56" height="56" viewBox="0 0 40 40" aria-hidden="true">
            <circle cx="20" cy="20" r="19" fill="#BBF7D0" />
            <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
          </svg>
        ) : null}
        <div className="flex flex-col gap-1">
          <h2 className="m-0 text-[26px] font-bold tracking-[-0.02em]">{headline}</h2>
          <span className="lead">
            {d.status === 'done'
              ? doneLead
              : d.status === 'cancelled'
                ? finished
                  ? tr(m.stoppedAt, { time: clock(finished) })
                  : tr(m.stopped)
                : `${d.error?.message ?? tr(m.stoppedError)} ${tr(m.rowsStay)}`}
          </span>
        </div>
      </div>
      <div className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
        {tiles.map(([k, n, note, border]) => (
          <div key={k} className="card tile" style={border ? { borderColor: border } : undefined}>
            <span className="k">{k}</span>
            <span className="n text-[22px]">{count(n)}</span>
            <span className="k">{note}</span>
          </div>
        ))}
      </div>
      {s?.templates ? <NotifuseTemplatesDone stats={s.templates} /> : null}
      {d.status === 'done' ? <WelcomePromptCard where={{ context: 'import', id: d.id }} /> : null}
      <section className="card flex flex-col">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
          <h3 className="h2">{tr(m.skippedChanged)}</h3>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" disabled={skipped === 0} onClick={() => void downloadImportFile(slug, d.id, 'skipped')}>
              {tr(m.downloadSkipped, { count: count(skipped) })}
            </Button>
            <Button size="sm" onClick={() => void downloadImportFile(slug, d.id, 'report')}>
              {tr(m.downloadReport)}
            </Button>
          </div>
        </div>
        <TableScroll label={tr(m.importReport)}>
          <table className="tbl">
            <thead>
              <tr>
                <Th>{tr(m.colReason)}</Th>
                <Th align="right">{tr(m.colRows)}</Th>
                <Th>{tr(m.colWhat)}</Th>
              </tr>
            </thead>
            <tbody>
              {rows.map(([k, n, [label, what]]) => (
                <tr key={k}>
                  <td>{label}</td>
                  <td className="r n">{count(n)}</td>
                  <td className="k">{what}</td>
                </tr>
              ))}
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={3} className="py-6 text-center text-ink-3">
                    {tr(m.everyRow)}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </TableScroll>
      </section>
      {d.status === 'done' ? (
        <section className="card flex flex-col gap-3 px-5 py-4.5">
          <h3 className="h2">{tr(m.nextSteps)}</h3>
          <div className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(240px,1fr))]">
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{tr(m.warmTitle)}</span>
              <span className="k">{tr(m.warmBody, { named: firstList ? 'yes' : 'no', list: firstList ?? '' })}</span>
            </div>
            {pending > 0 ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">{tr(m.watchPending, { count: count(pending) })}</span>
                <span className="k">
                  {[s?.confirmsSent && finished ? tr(m.repermissionSent, { time: clock(finished) }) : '', tr(m.nonResponders)].filter(Boolean).join(' ')}
                </span>
                {flow ? (
                  <ButtonLink size="sm" className="self-start" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: flow.id }}>
                    {tr(m.openFlow)}
                  </ButtonLink>
                ) : (
                  <ButtonLink size="sm" className="self-start" to="/w/$workspace/automations" params={{ workspace: slug }}>
                    {tr(m.openFlow)}
                  </ButtonLink>
                )}
              </div>
            ) : null}
            {suppressed > 0 ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">{tr(m.synced)}</span>
                <span className="k">
                  {tr(m.syncedBody, {
                    pushed: s?.pushedTo?.length ? 'yes' : 'no',
                    n: suppressed,
                    count: count(suppressed),
                    providers: andList((s?.pushedTo ?? []).map((p) => p.replace(/^Amazon /, ''))),
                    at: finished ? 'yes' : 'no',
                    time: finished ? clock(finished) : '',
                  })}
                </span>
                <ButtonLink size="sm" className="self-start" to="/w/$workspace/audience/suppressions" params={{ workspace: slug }}>
                  {tr(m.viewSuppressions)}
                </ButtonLink>
              </div>
            ) : null}
          </div>
        </section>
      ) : (
        <LiveLog d={d} />
      )}
      <div className="flex flex-wrap justify-end gap-2.5">
        <ButtonLink to="/w/$workspace/import" params={{ workspace: slug }}>
          {tr(m.importAnother)}
        </ButtonLink>
        <ButtonLink variant="primary" className="wnext" to="/w/$workspace/audience" params={{ workspace: slug }}>
          {tr(m.viewContacts)}
        </ButtonLink>
      </div>
    </>
  )
}
