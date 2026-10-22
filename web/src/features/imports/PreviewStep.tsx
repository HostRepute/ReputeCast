import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { TableScroll, Th } from '@/components/ui/Table'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { NotRead, StepFrame } from './FieldsStep'
import { downloadImportFile } from './SourceStep'
import { ReadingSource } from './states'
import { FoundBy, ImportVerifyCard, serviceInvalidLabel, useCheckedNotes } from '@/features/verification/ImportCheck'
import { m as vm } from '@/features/verification/messages'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './messages'

type Preview = Schemas['ImportPreview']
type Options = Schemas['ImportOptions']

/** Step 4: the dry run (MigrationPreview). */
export function PreviewStep({ importId }: { importId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  const preview = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/imports/{id}/preview',
    { params: { path: { workspace: slug, id: importId } } },
    { enabled: Boolean(d?.stagedAt), placeholderData: keepPreviousData },
  )
  const p = preview.data
  return (
    <StepFrame d={d} step={4} hints={{ 3: p ? t(m.hintSuppress, { count: count(p.suppressCount) }) : undefined, 4: t(m.hintNothingWritten) }} mainClassName="s34">
      <WizardHeading title={p ? t(m.previewTitle, { n: p.rows, count: count(p.rows) }) : t(m.checkingRows)} lead={t(m.previewLead)} />
      {q.isError || preview.isError ? (
        <ErrorPanel title={t(m.previewFailed)} actions={<Button onClick={() => void (q.isError ? q.refetch() : preview.refetch())}>{t(common.retry)}</Button>}>
          {errorMessage(q.error ?? preview.error)} {t(m.nothingImported)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title={t(m.loading)} />
      ) : !d.stagedAt ? (
        <NotRead d={d} />
      ) : !p ? (
        <ReadingSource title={t(m.checkingEvery)} detail={t(m.rows, { n: d.stagedRows, count: count(d.stagedRows) })} />
      ) : (
        <Result d={d} p={p} refreshing={preview.isFetching} />
      )}
    </StepFrame>
  )
}

function Result({ d, p, refreshing }: { d: Detail; p: Preview; refreshing: boolean }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/imports/{id}')
  const start = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/start')
  const reason = usePermissionReason('subscribers:import')
  const [err, setErr] = useState<string | null>(null)
  // The optional check with a verification service (X14).
  const t = useT()
  const checked = useCheckedNotes(p)
  const [noVerify, setNoVerify] = useState(false)
  const refetch = () => qc.invalidateQueries({ predicate: (x) => String((x.queryKey as unknown[])[1] ?? '').includes('/imports/') })
  const change = async (patch: Partial<Options>) => {
    setErr(null)
    try {
      await update.mutateAsync({ params: { path: { workspace: slug, id: d.id } }, body: { options: { ...d.options, ...patch } } })
      await refetch()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const run = async () => {
    setErr(null)
    try {
      await start.mutateAsync({ params: { path: { workspace: slug, id: d.id } } })
      await refetch()
      await navigate({ to: stepPath.progress, params: { workspace: slug, importId: d.id } })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const tiles: [string, number, string, string?][] = [
    [t(m.tileWillImport), p.willImport, checked?.willImport ?? t(m.tileWillImportNote), '#86EFAC'],
    [t(m.tileMergedExisting), p.merged, t(m.tileMergedNote)],
    [t(m.tileSuppressed), p.suppressed, t(m.tileSuppressedNote), '#FCA5A5'],
    [t(m.tileInvalid), p.invalid, checked?.invalid ?? t(m.tileInvalidNote)],
    [t(m.tileRisky), p.risky, checked?.risky ?? t(m.tileRiskyNote), '#FCD34D'],
  ]
  if (p.notImported > 0) tiles.push([t(m.tileNotImported), p.notImported, t(m.tileNotImportedNote)])
  return (
    <>
      <div className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(170px,1fr))]" aria-busy={refreshing}>
        {tiles.map(([k, n, note, border]) => (
          <div key={k} className="card tile" style={border ? { borderColor: border } : undefined}>
            <span className="k">{k}</span>
            <span className="n text-[22px]">{count(n)}</span>
            <span className="k">{note}</span>
          </div>
        ))}
      </div>
      {err ? <Note tone="bad">{err}</Note> : null}
      {p.verification || noVerify ? null : <ImportVerifyCard importId={d.id} rows={p.rows} onSkip={() => setNoVerify(true)} />}
      {p.riskyGroups.length > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">{p.verification ? t(vm.importRiskyAfter, { count: p.risky }) : t(m.riskyTitle, { count: count(p.risky) })}</h3>
            <span className="k">{checked?.header ?? t(m.riskyNote)}</span>
          </div>
          <TableScroll label={t(m.riskyTable)}>
            <table className="tbl mpre">
              <thead>
                <tr>
                  <Th>{t(m.colGroup)}</Th>
                  <Th align="right">{t(m.colRows)}</Th>
                  {p.verification ? (
                    <Th>{t(vm.colFoundBy)}</Th>
                  ) : (
                    <>
                      <Th>{t(m.colWhy)}</Th>
                      <Th>{t(m.colExamples)}</Th>
                    </>
                  )}
                  <Th className="w-[260px]">{t(m.colAction)}</Th>
                </tr>
              </thead>
              <tbody>
                {p.riskyGroups.map((g) => (
                  <tr key={g.key}>
                    <td className="font-semibold">{g.label}</td>
                    <td className="r n">{count(g.rows)}</td>
                    {p.verification ? (
                      <td>
                        <FoundBy g={g} service={p.verification.serviceName} />
                      </td>
                    ) : (
                      <>
                        <td className="k">{g.why}</td>
                        <td className="mono k">{g.examples}</td>
                      </>
                    )}
                    <td>
                      <select
                        aria-label={t(m.actionFor, { label: g.label })}
                        className="in w-full"
                        value={g.action}
                        disabled={update.isPending}
                        onChange={(e) => void change({ risky: { ...d.options.risky, [g.key]: e.target.value } })}
                      >
                        {g.actions.map((a) => (
                          <option key={a.value} value={a.value}>
                            {a.label}
                          </option>
                        ))}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        </section>
      ) : null}
      {p.duplicates.length > 0 || p.merged > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">{t(m.duplicatesTitle, { count: count(p.merged) })}</h3>
            <label className="k flex items-center gap-2 max-sm:w-full">
              <span className="whitespace-nowrap">{t(m.onConflict)}</span>
              <select
                className="in max-sm:min-w-0 max-sm:flex-1"
                value={p.onConflict}
                disabled={update.isPending}
                onChange={(e) => void change({ onConflict: e.target.value as Options['onConflict'] })}
              >
                <option value="fill">{t(m.conflictFill)}</option>
                <option value="overwrite">{t(m.conflictOverwrite, { source: d.sourceName })}</option>
                <option value="keep">{t(m.conflictKeep)}</option>
              </select>
            </label>
          </div>
          <TableScroll label={t(m.duplicates)}>
            <table className="tbl mpre">
              <thead>
                <tr>
                  <Th>{t(m.colEmail)}</Th>
                  <Th>{t(m.colInReputeCast)}</Th>
                  <Th>{t(m.colInSource, { source: d.sourceName })}</Th>
                  <Th>{t(m.colResult)}</Th>
                </tr>
              </thead>
              <tbody>
                {p.duplicates.map((x) => (
                  <tr key={x.email}>
                    <td className="mono">{x.email}</td>
                    <td className={x.suppressed ? 'k text-bad-fg' : 'k'}>{x.inReputeCast}</td>
                    <td className="k">{x.inSource}</td>
                    <td>
                      <Badge tone={x.resultTone === 'bad' ? 'bad' : 'ok'}>{x.result}</Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        </section>
      ) : null}
      {p.invalid > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">{t(m.invalidTitle, { count: count(p.invalid) })}</h3>
            <Button size="sm" onClick={() => void downloadImportFile(slug, d.id, 'invalid')}>
              {t(m.downloadThese)}
            </Button>
          </div>
          <div className="k flex flex-wrap gap-4 px-5 py-3">
            {p.invalidGroups.map((g) => (
              <span key={g.problem} className="mono">
                {g.service
                  ? serviceInvalidLabel(t, g.service, g.rows)
                  : `${g.example ? `${g.example} (${g.problem})` : g.problem === 'no MX record' ? t(m.noMx) : g.problem} · ${count(g.rows)}`}
              </span>
            ))}
          </div>
        </section>
      ) : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.consent} params={{ workspace: slug, importId: d.id }}>
          {t(m.back)}
        </ButtonLink>
        <Button
          variant="primary"
          className="wnext"
          disabledReason={reason ?? (p.importCount + p.suppressCount === 0 ? t(m.nothingToImport) : undefined)}
          loading={start.isPending}
          onClick={() => void run()}
        >
          {p.suppressCount > 0
            ? t(m.importButtonSuppressions, { n: p.importCount, count: count(p.importCount), s: p.suppressCount, suppressions: count(p.suppressCount) })
            : t(m.importButton, { n: p.importCount, count: count(p.importCount) })}
        </Button>
      </div>
    </>
  )
}
