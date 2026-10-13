import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { NotRead, StepFrame } from './FieldsStep'
import { downloadImportFile } from './SourceStep'
import { ReadingSource } from './states'

type Preview = Schemas['ImportPreview']
type Options = Schemas['ImportOptions']

/** Step 4: the dry run (MigrationPreview). */
export function PreviewStep({ importId }: { importId: string }) {
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
    <StepFrame d={d} step={4} hints={{ 3: p ? `${count(p.suppressCount)} to suppress` : undefined, 4: 'Nothing is written yet' }} mainClassName="s34">
      <WizardHeading
        title={p ? `${count(p.rows)} rows checked. Here's what would happen.` : 'Checking the rows'}
        lead="Dry run against your current contacts and the shared suppression list. Adjust the risky groups, then import."
      />
      {q.isError || preview.isError ? (
        <ErrorPanel title="Couldn't run the preview" actions={<Button onClick={() => void (q.isError ? q.refetch() : preview.refetch())}>Retry</Button>}>
          {errorMessage(q.error ?? preview.error)} Nothing was imported.
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title="Loading the import" />
      ) : !d.stagedAt ? (
        <NotRead d={d} />
      ) : !p ? (
        <ReadingSource title="Checking every row against your contacts and the suppression list" detail={`${count(d.stagedRows)} rows`} />
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
    ['Will be imported', p.willImport, 'new contacts, subscribed', '#86EFAC'],
    ['Merged with existing', p.merged, 'already in ReputeCast'],
    ['Suppressed', p.suppressed, 'unsubscribed, bounced, complained', '#FCA5A5'],
    ['Invalid, skipped', p.invalid, 'not an email address'],
    ['Risky, your call', p.risky, 'role, disposable, typo, trap-like', '#FCD34D'],
  ]
  if (p.notImported > 0) tiles.push(['Not imported', p.notImported, 'statuses set to Skip'])
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
      {p.riskyGroups.length > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">Risky addresses · {count(p.risky)}</h3>
            <span className="k">imported addresses that bounce or hit traps hurt every provider's reputation</span>
          </div>
          <div className="tscroll">
            <table className="tbl mpre">
              <thead>
                <tr>
                  <th scope="col">Group</th>
                  <th scope="col" className="r">
                    Rows
                  </th>
                  <th scope="col">Why it's risky</th>
                  <th scope="col">Examples</th>
                  <th scope="col" className="w-[260px]">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody>
                {p.riskyGroups.map((g) => (
                  <tr key={g.key}>
                    <td className="font-semibold">{g.label}</td>
                    <td className="r n">{count(g.rows)}</td>
                    <td className="k">{g.why}</td>
                    <td className="mono k">{g.examples}</td>
                    <td>
                      <select
                        aria-label={`Action for ${g.label}`}
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
          </div>
        </section>
      ) : null}
      {p.duplicates.length > 0 || p.merged > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">Duplicates · {count(p.merged)} already in ReputeCast</h3>
            <label className="k flex items-center gap-2">
              On conflict
              <select
                className="in"
                value={p.onConflict}
                disabled={update.isPending}
                onChange={(e) => void change({ onConflict: e.target.value as Options['onConflict'] })}
              >
                <option value="fill">Fill empty fields only (recommended)</option>
                <option value="overwrite">Overwrite with {d.sourceName} values</option>
                <option value="keep">Keep ReputeCast values</option>
              </select>
            </label>
          </div>
          <div className="tscroll">
            <table className="tbl mpre">
              <thead>
                <tr>
                  <th scope="col">Email</th>
                  <th scope="col">In ReputeCast</th>
                  <th scope="col">In {d.sourceName}</th>
                  <th scope="col">Result</th>
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
          </div>
        </section>
      ) : null}
      {p.invalid > 0 ? (
        <section className="card flex flex-col">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
            <h3 className="h2">Invalid · {count(p.invalid)} skipped</h3>
            <Button size="sm" onClick={() => void downloadImportFile(slug, d.id, 'invalid')}>
              Download these rows
            </Button>
          </div>
          <div className="k flex flex-wrap gap-4 px-5 py-3">
            {p.invalidGroups.map((g) => (
              <span key={g.problem} className="mono">
                {g.example ? `${g.example} (${g.problem})` : g.problem === 'no MX record' ? 'domain has no MX record' : g.problem} · {count(g.rows)}
              </span>
            ))}
          </div>
        </section>
      ) : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.consent} params={{ workspace: slug, importId: d.id }}>
          Back
        </ButtonLink>
        <Button
          variant="primary"
          className="wnext"
          disabledReason={reason ?? (p.importCount + p.suppressCount === 0 ? 'Nothing to import: every row is invalid or skipped.' : undefined)}
          loading={start.isPending}
          onClick={() => void run()}
        >
          Import {count(p.importCount)} {p.importCount === 1 ? 'contact' : 'contacts'}
          {p.suppressCount > 0 ? ` and ${count(p.suppressCount)} ${p.suppressCount === 1 ? 'suppression' : 'suppressions'}` : ''}
        </Button>
      </div>
    </>
  )
}
