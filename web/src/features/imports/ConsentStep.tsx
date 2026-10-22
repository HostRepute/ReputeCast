import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { TableScroll, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { NotRead, StepFrame } from './FieldsStep'
import { ReadingSource } from './states'
import { SmsImportConsent } from '@/features/sms/ImportConsent'
import { m as im } from './messages'

type Mapping = Schemas['ImportConsentMapping']
type ConsentMap = Schemas['ImportConsentMap']

const tone: Record<string, Tone> = { ok: 'ok', warn: 'warn', bad: 'bad', neutral: 'neutral' }

/** Step 3: carry consent over (MigrationConsent). */
export function ConsentStep({ importId }: { importId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useImportDetail(importId)
  const d = q.data
  const consent = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/imports/{id}/consent',
    { params: { path: { workspace: slug, id: importId } } },
    { enabled: Boolean(d?.stagedAt) },
  )
  const fm = $api.useQuery('get', '/api/v1/workspaces/{workspace}/imports/{id}/fields', { params: { path: { workspace: slug, id: importId } } }, { enabled: Boolean(d?.stagedAt) })
  const mapped = fm.data?.ready ? t(im.hintMapped, { mapped: count(fm.data.mapped + fm.data.new), skipped: count(fm.data.skipped) }) : undefined
  return (
    <StepFrame d={d} step={3} hints={{ 2: mapped, 3: t(im.hintWho) }} mainClassName="s34 sfull">
      <WizardHeading title={t(im.consentTitle)} lead={t(im.consentLead, { named: d ? 'yes' : 'no', source: d?.sourceName ?? '' })} />
      {q.isError || consent.isError ? (
        <ErrorPanel title={t(im.consentFailed)} actions={<Button onClick={() => void (q.isError ? q.refetch() : consent.refetch())}>{t(common.retry)}</Button>}>
          {errorMessage(q.error ?? consent.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title={t(im.loading)} />
      ) : !d.stagedAt ? (
        <NotRead d={d} />
      ) : !consent.data ? (
        <ReadingSource title={t(im.counting, { source: d.sourceName })} />
      ) : (
        <ConsentForm key={consent.dataUpdatedAt} d={d} m={consent.data} />
      )}
    </StepFrame>
  )
}

function ConsentForm({ d, m }: { d: Detail; m: Mapping }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/imports/{id}/consent')
  const reason = usePermissionReason('subscribers:import')
  const handling = (status: string) => m.statuses.find((s) => s.status === status)?.handling
  const [map, setMap] = useState<ConsentMap>(() => ({
    subscribed: (handling('subscribed') as ConsentMap['subscribed']) ?? 'confirmed',
    pending: (handling('pending') as ConsentMap['pending']) ?? 'pending',
    transactional: (handling('transactional') as ConsentMap['transactional']) ?? 'skip',
  }))
  const [evidence, setEvidence] = useState(m.evidence)
  const [pull, setPull] = useState(m.pullSuppressions)
  const [history, setHistory] = useState(m.recordHistory)
  const [err, setErr] = useState<string | null>(null)
  const submit = async () => {
    setErr(null)
    try {
      await save.mutateAsync({ params: { path: { workspace: slug, id: d.id } }, body: { map, evidence, pullSuppressions: pull, recordHistory: history } })
      await qc.invalidateQueries({ predicate: (x) => String((x.queryKey as unknown[])[1] ?? '').includes('/imports') })
      await navigate({ to: stepPath.preview, params: { workspace: slug, importId: d.id } })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const trustNote = `${m.evidenceSource ? t(im.evidenceStored, { evidence: m.evidenceSource }) : t(im.evidenceKept)} ${m.doubleOptIn ? t(im.trustDouble) : t(im.trustSingle)}`
  const value = (status: string, fallback: string) => (status in map ? (map as unknown as Record<string, string>)[status] : fallback)
  return (
    <>
      <section className="card flex flex-col">
        <TableScroll label={t(im.consentTable)}>
          {/* The board lets the status badges wrap on phones ("archived with "abuse" complaint"). */}
          <table className="tbl phone-wrap">
            <thead>
              <tr>
                <Th>{t(im.colSourceStatus, { source: d.sourceName })}</Th>
                <Th align="right">{t(im.colContacts)}</Th>
                <Th className="w-[360px]">{t(im.colBecomes)}</Th>
                <Th>{t(im.colEffect)}</Th>
              </tr>
            </thead>
            <tbody>
              {m.statuses.map((s) => (
                <tr key={s.status} className={s.locked ? 'supf' : undefined}>
                  <td>
                    <Badge tone={tone[s.tone] ?? 'neutral'}>{s.label}</Badge>
                    {s.note ? <div className="k">{s.note}</div> : null}
                  </td>
                  <td className="r n">{count(s.contacts)}</td>
                  <td>
                    <select
                      aria-label={t(im.handlingFor, { label: s.label })}
                      className="in"
                      value={s.locked ? s.handling : value(s.status, s.handling)}
                      aria-readonly={s.locked || undefined}
                      onChange={(e) => setMap((x) => ({ ...x, [s.status]: e.target.value }))}
                    >
                      {s.options.map((o) => (
                        <option key={o.value} value={o.value}>
                          {o.label}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="k">{s.effect}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
        <div className="flex flex-wrap items-center gap-2 border-t border-line-2 px-5 py-3">
          <Badge tone="ok">{t(im.willSuppress, { n: m.suppressCount, count: count(m.suppressCount) })}</Badge>
          <span className="k">{t(im.sharedList)}</span>
        </div>
      </section>
      <section className="card flex flex-col gap-3 px-5 py-4.5">
        <h3 className="h2">{t(im.evidenceTitle)}</h3>
        <fieldset className="mgrid m-0 grid gap-3 border-0 p-0 [grid-template-columns:repeat(auto-fit,minmax(300px,1fr))]">
          <legend className="vh">{t(im.evidenceLegend)}</legend>
          <Choice
            checked={evidence === 'trust'}
            onPick={() => setEvidence('trust')}
            title={t(im.trust, { double: m.doubleOptIn ? 'yes' : 'no', source: d.sourceName })}
            description={trustNote}
          />
          <Choice checked={evidence === 'repermission'} onPick={() => setEvidence('repermission')} title={t(im.repermission)} description={t(im.repermissionBody)} />
        </fieldset>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={pull} onChange={(e) => setPull(e.target.checked)} />
          <span>
            {t(im.pullBefore)}{' '}
            {m.providers.length > 0 ? (
              <span className="k">
                {m.inSync > 0 ? t(im.inSync, { providers: andList(m.providers), count: count(m.inSync) }) : t(im.checked, { providers: andList(m.providers) })}
              </span>
            ) : null}
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={history} onChange={(e) => setHistory(e.target.checked)} />
          <span>
            {t(im.recordHistory)} {m.historyExample ? <span className="k">· "{m.historyExample}"</span> : null}
          </span>
        </label>
      </section>
      {/* Phase 8: SMS consent comes in only with its evidence columns. */}
      {m.sms ? <SmsImportConsent sms={m.sms} /> : null}
      {err ? <Note tone="bad">{err}</Note> : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.fields} params={{ workspace: slug, importId: d.id }}>
          {t(im.back)}
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason} loading={save.isPending} onClick={() => void submit()}>
          {t(im.continuePreview)}
        </Button>
      </div>
    </>
  )
}

/** A radio option laid out as on the board: title, line break, explanation. */
function Choice({ checked, onPick, title, description }: { checked: boolean; onPick: () => void; title: string; description: string }) {
  const id = useId()
  return (
    <label className={`opt ${checked ? 'opt-on' : ''}`} aria-label={title}>
      <input type="radio" name="evidence" checked={checked} onChange={onPick} aria-describedby={id} />
      <span>
        <span className="font-semibold">{title}</span>
        <br />
        <span className="k" id={id}>
          {description}
        </span>
      </span>
    </label>
  )
}
