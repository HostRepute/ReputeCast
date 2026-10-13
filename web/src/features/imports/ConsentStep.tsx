import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { andList, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { stepPath, useImportDetail, WizardHeading, type Detail } from './ImportFrame'
import { NotRead, StepFrame } from './FieldsStep'
import { ReadingSource } from './states'

type Mapping = Schemas['ImportConsentMapping']
type ConsentMap = Schemas['ImportConsentMap']

const tone: Record<string, Tone> = { ok: 'ok', warn: 'warn', bad: 'bad', neutral: 'neutral' }

/** Step 3: carry consent over (MigrationConsent). */
export function ConsentStep({ importId }: { importId: string }) {
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
  const mapped = fm.data?.ready ? `${count(fm.data.mapped + fm.data.new)} mapped, ${count(fm.data.skipped)} skipped` : undefined
  return (
    <StepFrame d={d} step={3} hints={{ 2: mapped, 3: 'Who may be emailed' }} mainClassName="s34 sfull">
      <WizardHeading
        title="Carry consent over, exactly"
        lead={`Each ${d?.sourceName ?? 'source'} status maps to a ReputeCast status. Anyone who left or bounced goes straight to the shared suppression list and can never be re-subscribed by an import.`}
      />
      {q.isError || consent.isError ? (
        <ErrorPanel title="Couldn't load the consent mapping" actions={<Button onClick={() => void (q.isError ? q.refetch() : consent.refetch())}>Retry</Button>}>
          {errorMessage(q.error ?? consent.error)}
        </ErrorPanel>
      ) : !d ? (
        <ReadingSource title="Loading the import" />
      ) : !d.stagedAt ? (
        <NotRead d={d} />
      ) : !consent.data ? (
        <ReadingSource title={`Counting contacts per ${d.sourceName} status`} />
      ) : (
        <ConsentForm key={consent.dataUpdatedAt} d={d} m={consent.data} />
      )}
    </StepFrame>
  )
}

function ConsentForm({ d, m }: { d: Detail; m: Mapping }) {
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
  const trustNote = `${m.evidenceSource ? `${m.evidenceSource} are stored as consent evidence on each contact.` : 'The source’s opt-in is kept as consent evidence.'} ${
    m.doubleOptIn ? 'Recommended when the source audience used double opt-in, as this one did.' : 'Only choose this when every contact opted in.'
  }`
  const value = (status: string, fallback: string) => (status in map ? (map as unknown as Record<string, string>)[status] : fallback)
  return (
    <>
      <section className="card flex flex-col">
        <div className="tscroll">
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">{d.sourceName} status</th>
                <th scope="col" className="r">
                  Contacts
                </th>
                <th scope="col" className="w-[360px]">
                  Becomes in ReputeCast
                </th>
                <th scope="col">Effect</th>
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
                      aria-label={`Handling for ${s.label}`}
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
        </div>
        <div className="flex flex-wrap items-center gap-2 border-t border-line-2 px-5 py-3">
          <Badge tone="ok">
            {count(m.suppressCount)} {m.suppressCount === 1 ? 'address' : 'addresses'} will be suppressed
          </Badge>
          <span className="k">They count toward the shared list and sync to every provider before the first send to this list.</span>
        </div>
      </section>
      <section className="card flex flex-col gap-3 px-5 py-4.5">
        <h3 className="h2">Consent evidence and re-permission</h3>
        <fieldset className="mgrid m-0 grid gap-3 border-0 p-0 [grid-template-columns:repeat(auto-fit,minmax(300px,1fr))]">
          <legend className="vh">How consent is carried over</legend>
          <Choice
            checked={evidence === 'trust'}
            onPick={() => setEvidence('trust')}
            title={`Trust ${d.sourceName}'s ${m.doubleOptIn ? 'double opt-in' : 'opt-in'}`}
            description={trustNote}
          />
          <Choice
            checked={evidence === 'repermission'}
            onPick={() => setEvidence('repermission')}
            title="Re-permission everyone first"
            description={`Imports as "pending" and sends one confirmation email via the System stream. Contacts who don't confirm within 14 days are suppressed. Safer for old or single-opt-in lists.`}
          />
        </fieldset>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={pull} onChange={(e) => setPull(e.target.checked)} />
          <span>
            Pull existing suppressions from connected sending providers before importing{' '}
            {m.providers.length > 0 ? (
              <span className="k">
                · {andList(m.providers)} {m.inSync > 0 ? `are already in sync (${count(m.inSync)} addresses), so this just re-checks` : 'are checked for addresses they already block'}
              </span>
            ) : null}
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={history} onChange={(e) => setHistory(e.target.checked)} />
          <span>
            Record the import in each contact's consent history {m.historyExample ? <span className="k">· "{m.historyExample}"</span> : null}
          </span>
        </label>
      </section>
      {err ? <Note tone="bad">{err}</Note> : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to={stepPath.fields} params={{ workspace: slug, importId: d.id }}>
          Back
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason} loading={save.isPending} onClick={() => void submit()}>
          Continue to preview
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
