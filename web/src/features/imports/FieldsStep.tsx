import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, sourceHint, stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { PausedCard, ReadingSource, readingDetail, SourceFailed } from './states'

type Row = Schemas['ImportFieldRow']
interface Choice {
  target: string
  type?: 'text' | 'number' | 'date' | 'boolean'
}

/** What ReputeCast stores for a built-in target. */
const builtinType: Record<string, string> = {
  email: 'email',
  first_name: 'text',
  last_name: 'text',
  name: 'text',
  company: 'text',
  country: 'text',
  timezone: 'timezone',
  language: 'text',
  tags: 'tags',
  consent_evidence: 'consent',
  status: 'status',
}

const statusOf = (t: string) => (t === 'skip' ? 'skipped' : t.startsWith('new:') ? 'new' : 'mapped')

/** The frame shared by steps 2 to 4: title, step badge, exit. */
export function StepFrame({
  d,
  step,
  hints,
  mainClassName,
  children,
}: {
  d?: Detail
  step: 2 | 3 | 4
  hints?: Partial<Record<1 | 2 | 3 | 4 | 5, string>>
  mainClassName?: string
  children: ReactNode
}) {
  const { slug } = useWorkspace()
  return (
    <ImportFrame
      title={d ? `Import contacts · ${d.sourceName}` : 'Import contacts'}
      badge={<Badge tone="neutral">Step {step} of 5</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          Exit
        </ButtonLink>
      }
      step={step}
      hints={{ ...(d ? { 1: sourceHint(d) } : {}), ...hints }}
      mainClassName={mainClassName}
    >
      {children}
    </ImportFrame>
  )
}

/** Step 2: map the source's fields (MigrationFields). */
export function FieldsStep({ importId }: { importId: string }) {
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  const reading = d && (d.status === 'staging' || d.status === 'paused' || !d.stagedAt)
  const fields = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/imports/{id}/fields',
    { params: { path: { workspace: slug, id: importId } } },
    { enabled: Boolean(d && !reading) },
  )
  const unit = d?.source === 'mailchimp' ? 'merge fields' : d?.source === 'csv' || d?.source === 'xlsx' ? 'columns' : 'fields'
  return (
    <StepFrame d={d} step={2} hints={d && d.fieldCount > 0 ? { 2: `${count(d.fieldCount)} ${unit} found` } : undefined} mainClassName="s34 sfull ftd8">
      <WizardHeading
        title={d ? (d.source === 'csv' || d.source === 'xlsx' ? 'Map your columns to ReputeCast' : `Map ${d.sourceName} fields to ReputeCast`) : 'Map fields'}
        lead="Matches were guessed from names and sample values. New custom fields are created on import; unmapped fields are skipped, never lost from the source."
      />
      {q.isError ? (
        <ErrorPanel title="Couldn't load this import" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title="Loading the import" />
      ) : d.status === 'paused' ? (
        <PausedCard d={d} now={q.dataUpdatedAt} />
      ) : d.error && !d.stagedAt ? (
        <SourceFailed d={d} />
      ) : !d.stagedAt && d.status !== 'staging' ? (
        <NotRead d={d} />
      ) : reading ? (
        <ReadingSource
          title={d.source === 'csv' || d.source === 'xlsx' ? `Reading ${d.fileName}` : `Reading ${d.source === 'mailchimp' ? 'audiences and merge fields' : 'lists and fields'} from ${d.sourceName}`}
          detail={readingDetail(d)}
        />
      ) : fields.isError ? (
        <ErrorPanel title="Couldn't load the fields" actions={<Button onClick={() => fields.refetch()}>Retry</Button>}>
          {errorMessage(fields.error)}
        </ErrorPanel>
      ) : !fields.data || !fields.data.ready ? (
        <ReadingSource title={`Reading fields from ${d.sourceName}`} detail={readingDetail(d)} />
      ) : (
        <Mapping key={fields.dataUpdatedAt} d={d} rows={fields.data.fields} />
      )}
    </StepFrame>
  )
}

function Mapping({ d, rows }: { d: Detail; rows: Row[] }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/imports/{id}/fields')
  const reason = usePermissionReason('subscribers:import')
  const initial = (rs: Row[]) => Object.fromEntries(rs.map((r) => [r.source, { target: r.target, type: r.status === 'new' ? (r.type as Choice['type']) : undefined }])) as Record<string, Choice>
  const [choices, setChoices] = useState<Record<string, Choice>>(() => initial(rows))
  const [resetting, setResetting] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const counts = { mapped: 0, new: 0, skipped: 0 }
  for (const r of rows) counts[statusOf(choices[r.source]?.target ?? r.target) as keyof typeof counts]++
  const reset = async () => {
    setErr(null)
    setResetting(true)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/imports/{id}/fields', { params: { path: { workspace: slug, id: d.id }, query: { guesses: true } } })
      if (data) setChoices(initial(data.fields))
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setResetting(false)
    }
  }
  const submit = async () => {
    setErr(null)
    try {
      await save.mutateAsync({
        params: { path: { workspace: slug, id: d.id } },
        body: { fields: rows.map((r) => ({ source: r.source, target: choices[r.source]?.target ?? r.target, type: choices[r.source]?.type })) },
      })
      await qc.invalidateQueries({ predicate: (x) => String((x.queryKey as unknown[])[1] ?? '').includes('/imports') })
      await navigate({ to: stepPath.consent, params: { workspace: slug, importId: d.id } })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const sourceCol = d.source === 'csv' || d.source === 'xlsx' ? 'Column' : `${d.sourceName} field`
  return (
    <>
      <section className="card flex flex-col">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3">
          <div className="flex flex-wrap gap-1.5" role="status">
            <Badge tone="ok">{count(counts.mapped)} mapped</Badge>
            <Badge tone="info">
              {count(counts.new)} new {counts.new === 1 ? 'field' : 'fields'}
            </Badge>
            <Badge tone="neutral">{count(counts.skipped)} skipped</Badge>
          </div>
          <Button size="sm" loading={resetting} onClick={() => void reset()}>
            Reset to guesses
          </Button>
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
        <TableScroll label="Field mapping">
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">{sourceCol}</th>
                <th scope="col">Sample values</th>
                <th scope="col" className="w-[280px]">
                  ReputeCast field
                </th>
                <th scope="col">Type</th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const c = choices[r.source] ?? { target: r.target }
                const st = statusOf(c.target)
                const type =
                  st === 'skipped' ? '–' : c.target === r.target && r.status !== 'new' ? r.type : (builtinType[c.target] ?? (c.target.startsWith('custom:') ? 'custom' : r.type))
                return (
                  <tr key={r.source} className={st === 'new' ? 'newf' : st === 'skipped' ? 'skipf' : undefined}>
                    <td className="mono">{r.label}</td>
                    <td className="k mono">{r.samples.join(' · ') || '–'}</td>
                    <td>
                      <select
                        aria-label={`Map ${r.label} to`}
                        className="in"
                        value={c.target}
                        onChange={(e) => {
                          const target = e.target.value
                          setChoices((all) => ({ ...all, [r.source]: { target, type: target.startsWith('new:') ? (all[r.source]?.type ?? 'text') : undefined } }))
                        }}
                      >
                        {r.options.map((o) => (
                          <option key={o.value} value={o.value}>
                            {o.label}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className={st === 'new' ? undefined : 'k'}>
                      {st === 'new' ? (
                        <select
                          aria-label={`Type of ${r.label}`}
                          className="in w-[110px]"
                          value={c.type ?? 'text'}
                          onChange={(e) => setChoices((all) => ({ ...all, [r.source]: { target: c.target, type: e.target.value as Choice['type'] } }))}
                        >
                          <option value="text">text</option>
                          <option value="number">number</option>
                          <option value="date">date</option>
                          <option value="boolean">boolean</option>
                        </select>
                      ) : (
                        type
                      )}
                    </td>
                    <td>
                      {st === 'new' ? <Badge tone="info">New field</Badge> : st === 'skipped' ? <Badge tone="neutral">Skipped</Badge> : <Badge tone="ok">Mapped</Badge>}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
        <div className="flex flex-wrap gap-4 border-t border-line-2 px-5 py-3 text-[13px] text-ink-2">
          <span>
            <strong>Duplicates:</strong> matched on email, case-insensitive. Existing contacts are merged; imported values fill empty fields only, unless you choose "overwrite" in the
            preview step.
          </span>
        </div>
      </section>
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.source} params={{ workspace: slug, importId: d.id }}>
          Back
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason} loading={save.isPending} onClick={() => void submit()}>
          Continue to consent
        </Button>
      </div>
    </>
  )
}

/** Nothing was read from the source yet: back to step 1. */
export function NotRead({ d }: { d: Detail }) {
  const { slug } = useWorkspace()
  return (
    <Note tone="info">
      <span>
        Nothing has been read from {d.sourceName} yet. Choose what to import first.{' '}
        <ButtonLink size="sm" to={stepPath.source} params={{ workspace: slug, importId: d.id }}>
          Back to the source
        </ButtonLink>
      </span>
    </Note>
  )
}
