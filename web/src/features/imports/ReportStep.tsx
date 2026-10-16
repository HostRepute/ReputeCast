import { Navigate } from '@tanstack/react-router'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { TableScroll } from '@/components/ui/Table'
import { NotifuseTemplatesDone } from '@/features/transactional-import/NotifuseOption'
import { WelcomePromptCard } from '@/features/welcome-prompt/WelcomePromptCard'
import { andList, clock, count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ImportFrame, stepPath, currentStep, useImportDetail, type Detail } from './ImportFrame'
import { LiveLog } from './ProgressStep'
import { downloadImportFile } from './SourceStep'
import { ReadingSource } from './states'
import { $api, errorMessage } from '@/api/client'

/** "3 minutes 10 seconds" */
function took(ms: number) {
  const s = Math.max(1, Math.round(ms / 1000))
  const m = Math.floor(s / 60)
  const rest = s % 60
  const sec = `${rest} second${rest === 1 ? '' : 's'}`
  if (m === 0) return sec
  const min = `${m} minute${m === 1 ? '' : 's'}`
  return rest ? `${min} ${sec}` : min
}

/** Why rows were skipped or changed, in the report's words. */
function reasonRow(key: string, d: Detail): [string, string] | null {
  const src = d.sourceName
  switch (key) {
    case 'invalid':
      return ['Invalid address', 'Not imported. Listed in the CSV with the row number and the raw value.']
    case 'disposable':
      return ['Disposable domain', 'Not imported, as chosen in the preview.']
    case 'trap':
      return ['Spam-trap pattern', 'Not imported, as chosen. Addresses are redacted in the report.']
    case 'typo_fixed':
      return ['Likely typo, fixed', 'Imported with the corrected domain and tagged "review-typo". Original value kept in the report.']
    case 'typo':
      return ['Likely typo', 'Not imported, as chosen in the preview.']
    case 'role':
      return d.options.risky.role === 'skip'
        ? ['Role address', 'Not imported, as chosen in the preview.']
        : d.options.risky.role === 'import'
          ? ['Role address', 'Imported normally, as chosen in the preview.']
          : ['Role address', 'Imported and excluded from Broadcasts automatically.']
    case 'stale':
      return d.options.risky.stale === 'skip'
        ? ['Never engaged, 2+ years old', 'Not imported, as chosen in the preview.']
        : ['Never engaged, 2+ years old', 'Imported as pending; a re-permission email went out through the System stream.']
    case 'already_suppressed':
      return ['Already suppressed in ReputeCast', `${src} said subscribed, ReputeCast had a bounce or unsubscribe. Suppression wins; not re-subscribed.`]
    case 'status_skipped':
      return ['Status set to Skip', 'Not imported, as chosen in the consent step.']
  }
  return null
}

const reasonOrder = ['invalid', 'disposable', 'trap', 'typo_fixed', 'typo', 'role', 'stale', 'already_suppressed', 'status_skipped']

/** The import summary (MigrationDone). */
export function ReportStep({ importId }: { importId: string }) {
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  if (d && currentStep(d) !== 'report') {
    return <Navigate to={stepPath[currentStep(d)]} params={{ workspace: slug, importId }} replace />
  }
  const badge =
    d?.status === 'done' ? (
      <Badge tone="ok">Done</Badge>
    ) : d?.status === 'cancelled' ? (
      <Badge tone="warn">Stopped</Badge>
    ) : d?.status === 'failed' ? (
      <Badge tone="bad">Failed</Badge>
    ) : undefined
  return (
    <ImportFrame title={d ? `Import contacts · ${d.sourceName}` : 'Import contacts'} documentTitle="Import summary" badge={badge} narrow>
      {q.isError ? (
        <ErrorPanel title="Couldn't load the import" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title="Loading the summary" />
      ) : (
        <Summary d={d} now={q.dataUpdatedAt} />
      )}
    </ImportFrame>
  )
}

function Summary({ d, now }: { d: Detail; now: number }) {
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
      ? `${lists[0]!.name} now has ${count(lists[0]!.contacts)} contacts.`
      : lists.length > 1
        ? `${andList(lists.map((l, i) => (i === 0 ? `${l.name} now has ${count(l.contacts)}` : `${l.name} ${count(l.contacts)}`)))} contacts.`
        : ''
  const skippedParts = [
    reasons.invalid ? `${count(reasons.invalid)} invalid` : '',
    reasons.disposable ? `${count(reasons.disposable)} disposable` : '',
    reasons.trap ? `${count(reasons.trap)} trap-like` : '',
  ].filter(Boolean)
  const known = (reasons.invalid ?? 0) + (reasons.disposable ?? 0) + (reasons.trap ?? 0)
  if (skipped - known > 0 && skippedParts.length > 0) skippedParts.push(`${count(skipped - known)} other`)
  const rows = reasonOrder
    .filter((k) => (reasons[k] ?? 0) > 0)
    .map((k) => [k, reasons[k]!, reasonRow(k, d)] as const)
    .filter((r): r is readonly [string, number, [string, string]] => r[2] !== null)
  const flow = flows.data?.items?.find((f) => /re-?permission/i.test(f.name))
  const firstList = lists[0]?.name ?? 'this list'
  const tiles: [string, number, string, string?][] = [
    ['Imported', s?.imported ?? d.progress.imported, 'new, subscribed', '#86EFAC'],
    ['Merged', s?.merged ?? d.progress.merged, 'existing updated'],
    [
      'Suppressed',
      suppressed,
      [by.unsubscribe ? `${count(by.unsubscribe)} unsubscribed` : '', by.hard_bounce ? `${count(by.hard_bounce)} bounced` : '', by.complaint ? `${count(by.complaint)} complaints` : '']
        .filter(Boolean)
        .join(' · ') || 'never re-subscribed',
      '#FCA5A5',
    ],
    ['Pending re-permission', pending, s?.confirmsSent ? 'confirmation sent via System stream' : 'waiting for a new opt-in', '#FCD34D'],
    ['Skipped', skipped, skippedParts.join(' · ') || 'invalid, disposable, traps'],
  ]
  const headline =
    d.status === 'done'
      ? `${count(imported)} ${imported === 1 ? 'contact' : 'contacts'} imported from ${d.sourceName}`
      : d.status === 'cancelled'
        ? `Import stopped · ${count(imported)} ${imported === 1 ? 'contact' : 'contacts'} imported from ${d.sourceName}`
        : `The import from ${d.sourceName} failed`
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
            {d.status === 'done' ? (
              <>
                {finished ? `Finished ${finishedAt}` : 'Finished'}
                {s?.durationMs ? `, ${took(s.durationMs)}` : ''}. {listsLine} Consent carried over for every row
                {suppressed > 0 ? `; ${count(suppressed)} ${suppressed === 1 ? 'address was' : 'addresses were'} suppressed and will never be re-subscribed by an import` : ''}.
              </>
            ) : d.status === 'cancelled' ? (
              `Stopped${finished ? ` at ${clock(finished)}` : ''}. Rows imported before that stay; nothing else was changed.`
            ) : (
              `${d.error?.message ?? 'The import stopped with an error.'} Rows imported before the error stay.`
            )}
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
          <h3 className="h2">Skipped and changed rows</h3>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" disabled={skipped === 0} onClick={() => void downloadImportFile(slug, d.id, 'skipped')}>
              Download skipped rows · CSV, {count(skipped)}
            </Button>
            <Button size="sm" onClick={() => void downloadImportFile(slug, d.id, 'report')}>
              Download full report · CSV
            </Button>
          </div>
        </div>
        <TableScroll label="Import report">
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">Reason</th>
                <th scope="col" className="r">
                  Rows
                </th>
                <th scope="col">What happened</th>
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
                    Every row was imported as it was.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </TableScroll>
      </section>
      {d.status === 'done' ? (
        <section className="card flex flex-col gap-3 px-5 py-4.5">
          <h3 className="h2">Recommended next steps</h3>
          <div className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(240px,1fr))]">
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Warm them in gradually</span>
              <span className="k">
                Send Readiness will warn if a campaign to {firstList} includes more than 20% never-mailed-from-this-domain contacts. Consider a first send to the Active tier only.
              </span>
            </div>
            {pending > 0 ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">Watch the {count(pending)} pending</span>
                <span className="k">
                  {s?.confirmsSent && finished ? `Re-permission emails went out at ${clock(finished)}. ` : ''}Non-responders are suppressed automatically after 14 days.
                </span>
                {flow ? (
                  <ButtonLink size="sm" className="self-start" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: flow.id }}>
                    Open re-permission flow
                  </ButtonLink>
                ) : (
                  <ButtonLink size="sm" className="self-start" to="/w/$workspace/automations" params={{ workspace: slug }}>
                    Open re-permission flow
                  </ButtonLink>
                )}
              </div>
            ) : null}
            {suppressed > 0 ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">Suppressions synced</span>
                <span className="k">
                  {count(suppressed)} new {suppressed === 1 ? 'address' : 'addresses'} {s?.pushedTo?.length ? `pushed to ${andList(s.pushedTo.map((p) => p.replace(/^Amazon /, '')))}` : 'added to the shared list'}
                  {finished ? ` at ${clock(finished)}` : ''}. The next campaign can't reach them on any provider.
                </span>
                <ButtonLink size="sm" className="self-start" to="/w/$workspace/audience/suppressions" params={{ workspace: slug }}>
                  View suppression list
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
          Import another source
        </ButtonLink>
        <ButtonLink variant="primary" className="wnext" to="/w/$workspace/audience" params={{ workspace: slug }}>
          View contacts
        </ButtonLink>
      </div>
    </>
  )
}
