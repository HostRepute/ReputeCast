import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Fragment, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, sourceHint, stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { PausedCard, ReadingSource, readingDetail, SourceFailed } from './states'
import { ImportLanguageRow } from '@/features/page-languages/imports'
import { m } from './messages'

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
  language: 'language',
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
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <ImportFrame
      title={d ? t(m.titleFrom, { source: d.sourceName }) : t(m.title)}
      badge={<Badge tone="neutral">{t(m.stepOf, { step, total: 5 })}</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          {t(m.exit)}
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
  const t = useT()
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
  const unit = d?.source === 'mailchimp' ? 'merge' : d?.source === 'csv' || d?.source === 'xlsx' ? 'columns' : 'fields'
  return (
    <StepFrame
      d={d}
      step={2}
      hints={d && d.fieldCount > 0 ? { 2: t(m.hintFields, { unit, n: d.fieldCount, count: count(d.fieldCount) }) } : undefined}
      mainClassName="s34 sfull ftd8"
    >
      <WizardHeading
        title={d ? (d.source === 'csv' || d.source === 'xlsx' ? t(m.mapColumns) : t(m.mapSourceFields, { source: d.sourceName })) : t(m.mapFields)}
        lead={t(m.mapLead)}
      />
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title={t(m.loading)} />
      ) : d.status === 'paused' ? (
        <PausedCard d={d} now={q.dataUpdatedAt} />
      ) : d.error && !d.stagedAt ? (
        <SourceFailed d={d} />
      ) : !d.stagedAt && d.status !== 'staging' ? (
        <NotRead d={d} />
      ) : reading ? (
        <ReadingSource
          title={
            d.source === 'csv' || d.source === 'xlsx'
              ? t(m.readingFile, { file: d.fileName })
              : t(m.readingFrom, { mailchimp: d.source === 'mailchimp' ? 'yes' : 'no', source: d.sourceName })
          }
          detail={readingDetail(d)}
        />
      ) : fields.isError ? (
        <ErrorPanel title={t(m.fieldsFailed)} actions={<Button onClick={() => fields.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(fields.error)}
        </ErrorPanel>
      ) : !fields.data || !fields.data.ready ? (
        <ReadingSource title={t(m.readingFieldsFrom, { source: d.sourceName })} detail={readingDetail(d)} />
      ) : (
        <Mapping key={fields.dataUpdatedAt} d={d} rows={fields.data.fields} />
      )}
    </StepFrame>
  )
}

function Mapping({ d, rows }: { d: Detail; rows: Row[] }) {
  const t = useT()
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
  const sourceCol = d.source === 'csv' || d.source === 'xlsx' ? t(m.colColumn) : t(m.colSourceField, { source: d.sourceName })
  return (
    <>
      <section className="card flex flex-col">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3">
          <div className="flex flex-wrap gap-1.5" role="status">
            <Badge tone="ok">{t(m.mappedCount, { count: count(counts.mapped) })}</Badge>
            <Badge tone="info">{t(m.newFields, { n: counts.new, count: count(counts.new) })}</Badge>
            <Badge tone="neutral">{t(m.skippedCount, { count: count(counts.skipped) })}</Badge>
          </div>
          <Button size="sm" loading={resetting} onClick={() => void reset()}>
            {t(m.resetGuesses)}
          </Button>
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
        <TableScroll label={t(m.fieldMapping)}>
          <table className="tbl">
            <thead>
              <tr>
                <Th>{sourceCol}</Th>
                <Th>{t(m.colSamples)}</Th>
                <Th className="w-[280px]">{t(m.colTarget)}</Th>
                <Th>{t(m.colType)}</Th>
                <Th>{t(m.colStatus)}</Th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const c = choices[r.source] ?? { target: r.target }
                const st = statusOf(c.target)
                const type =
                  st === 'skipped' ? '–' : c.target === r.target && r.status !== 'new' ? r.type : (builtinType[c.target] ?? (c.target.startsWith('custom:') ? 'custom' : r.type))
                return (
                  <Fragment key={r.source}>
                    <tr className={st === 'new' ? 'newf' : st === 'skipped' ? 'skipf' : undefined}>
                      <td className="mono">{r.label}</td>
                      <td className="k mono">{r.samples.join(' · ') || '–'}</td>
                      <td>
                        <select
                          aria-label={t(m.mapTo, { label: r.label })}
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
                            aria-label={t(m.typeOf, { label: r.label })}
                            className="in w-[110px]"
                            value={c.type ?? 'text'}
                            onChange={(e) => setChoices((all) => ({ ...all, [r.source]: { target: c.target, type: e.target.value as Choice['type'] } }))}
                          >
                            {(['text', 'number', 'date', 'boolean'] as const).map((v) => (
                              <option key={v} value={v}>
                                {t(m.fieldType, { type: v })}
                              </option>
                            ))}
                          </select>
                        ) : type === '–' ? (
                          type
                        ) : (
                          t(m.fieldType, { type })
                        )}
                      </td>
                      <td>
                        {st === 'new' ? (
                          <Badge tone="info">{t(m.newField)}</Badge>
                        ) : st === 'skipped' ? (
                          <Badge tone="neutral">{t(m.skipped)}</Badge>
                        ) : (
                          <Badge tone="ok">{t(m.mapped)}</Badge>
                        )}
                      </td>
                    </tr>
                    {c.target === 'language' ? <ImportLanguageRow importId={d.id} field={r.source} columns={5} /> : null}
                  </Fragment>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
        <div className="flex flex-wrap gap-4 border-t border-line-2 px-5 py-3 text-[13px] text-ink-2">
          <span>
            <T msg={m.duplicatesRule} values={{ b: (c) => <strong>{c}</strong> }} />
          </span>
        </div>
      </section>
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.source} params={{ workspace: slug, importId: d.id }}>
          {t(m.back)}
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason} loading={save.isPending} onClick={() => void submit()}>
          {t(m.continueConsent)}
        </Button>
      </div>
    </>
  )
}

/** Nothing was read from the source yet: back to step 1. */
export function NotRead({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Note tone="info">
      <span>
        {t(m.notRead, { source: d.sourceName })}{' '}
        <ButtonLink size="sm" to={stepPath.source} params={{ workspace: slug, importId: d.id }}>
          {t(m.backToSource)}
        </ButtonLink>
      </span>
    </Note>
  )
}
