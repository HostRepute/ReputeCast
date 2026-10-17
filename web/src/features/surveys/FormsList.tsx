import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { count, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { CreateFormDialog } from '@/features/forms/FormBuilderPage'
import { FormsTabs, hostPath, throwIfForbidden, useFormsOverview } from '@/features/forms/common'
import { FormsLoading } from '@/features/forms/FormsStates'
import { useSurveys } from './api'
import { m } from './messages'
import './surveys.css'

type FormSummary = Schemas['SignupFormSummary']
type Survey = Schemas['Survey']

/** A survey's status badge: draft, open (until when) or closed. */
export function SurveyStatus({ s }: { s: Survey }) {
  const t = useT()
  if (s.status === 'draft') return <span className="badge mut">{t(m.statusDraft)}</span>
  if (s.status === 'closed') return <span className="badge mut">{s.closedAt ? t(m.statusClosed, { date: shortDate(s.closedAt) }) : t(m.statusClosedNow)}</span>
  return (
    <span className="badge ok">
      <span className="dot" aria-hidden="true" />
      {s.closesAt ? t(m.statusOpenCloses, { date: shortDate(s.closesAt) }) : t(m.statusOpen)}
    </span>
  )
}

function initials(name: string) {
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((p) => p[0]!.toUpperCase())
      .join('') || '·'
  )
}

type Row = { key: string; name: string; type: 'signup' | 'survey'; form?: FormSummary; survey?: Survey }

/**
 * Forms & pages › Forms (brief §5.9): signup forms and surveys with a Type
 * column. "New form…" asks which kind.
 */
export function FormsList() {
  const t = useT()
  const { slug } = useWorkspace()
  const overview = useFormsOverview()
  const surveys = useSurveys()
  throwIfForbidden(overview.error)
  const [q, setQ] = useState('')
  const [picking, setPicking] = useState(false)
  const [creatingForm, setCreatingForm] = useState(false)
  const reason = usePermissionReason('forms:edit')
  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [
      ...(overview.data?.forms ?? []).map((f) => ({ key: `f:${f.id}`, name: f.name, type: 'signup' as const, form: f })),
      ...(surveys.data?.items ?? []).map((s) => ({ key: `s:${s.id}`, name: s.name, type: 'survey' as const, survey: s })),
    ]
    const needle = q.trim().toLowerCase()
    return needle ? out.filter((r) => r.name.toLowerCase().includes(needle)) : out
  }, [overview.data, surveys.data, q])
  const total = (overview.data?.forms.length ?? 0) + (surveys.data?.items.length ?? 0)
  const page = (body: React.ReactNode) => (
    <Page title="Forms & public pages" filters={<FormsTabs />}>
      {body}
      <NewFormDialog
        open={picking}
        onOpenChange={setPicking}
        onSignup={() => {
          setPicking(false)
          setCreatingForm(true)
        }}
      />
      <CreateFormDialog open={creatingForm} onOpenChange={setCreatingForm} />
    </Page>
  )
  if (overview.isError) {
    return page(
      <ErrorPanel title={t(m.errorTitle)} actions={<Button onClick={() => void overview.refetch()}>{t(m.retry)}</Button>}>
        {errorMessage(overview.error)}
      </ErrorPanel>,
    )
  }
  if (overview.isPending || surveys.isPending) return page(<FormsLoading />)
  const o = overview.data
  if (!total) {
    return page(
      <section className="card flex flex-col items-center gap-2.5 border-dashed px-7 py-8 text-center">
        <h2 className="m-0 text-[17px] font-bold">{t(m.emptyTitle)}</h2>
        <p className="max-w-[560px] text-ink-2">{t(m.emptyBody, { url: hostPath(o.hostedUrl) })}</p>
        <div className="flex flex-wrap justify-center gap-2.5">
          <Button variant="primary" onClick={() => setPicking(true)} disabledReason={reason}>
            {t(m.newForm)}
          </Button>
          <a className="btn" href={o.hostedUrl} target="_blank" rel="noreferrer">
            {t(m.openHosted)}
          </a>
        </div>
      </section>,
    )
  }
  // Signup forms first, then surveys, each as the API lists them.
  const sorted = rows
  return page(
    <section className="card min-w-0" aria-labelledby="sv-forms-h">
      <div className="sv-list-h">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="sv-forms-h">
            {t(m.formsTitle, { count: total })}
          </h2>
          <span className="k">{t(m.formsSub)}</span>
        </div>
        <div className="sv-acts">
          <input className="in sv-search" type="search" aria-label={t(m.searchForms)} placeholder={t(m.searchForms)} value={q} onChange={(e) => setQ(e.target.value)} />
          <Button variant="primary" icon="plus" onClick={() => setPicking(true)} disabledReason={reason}>
            {t(m.newForm)}
          </Button>
        </div>
      </div>
      {sorted.length ? (
        <div className="overflow-x-auto">
          <table className="tbl sv-tbl" aria-label={t(m.formsTable)}>
            <thead>
              <tr>
                <th scope="col">{t(m.colName)}</th>
                <th scope="col">{t(m.colType)}</th>
                <th scope="col">{t(m.colWhere)}</th>
                <th scope="col">{t(m.colStatus)}</th>
                <th scope="col">{t(m.colActivity)}</th>
                <th scope="col">{t(m.colOwner)}</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((r) =>
                r.form ? (
                  <FormRow key={r.key} f={r.form} slug={slug} />
                ) : r.survey ? (
                  <SurveyRow key={r.key} s={r.survey} slug={slug} />
                ) : null,
              )}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="k" style={{ padding: '4px 20px 18px' }}>
          {t(m.noMatchForms, { query: q.trim() })}
        </p>
      )}
    </section>,
  )
}

function FormRow({ f, slug }: { f: FormSummary; slug: string }) {
  const t = useT()
  // The list has each form's summary; its sites and numbers come with its details.
  const detail = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}', { params: { path: { workspace: slug, form: f.id } } }, { staleTime: 60_000 })
  const d = detail.data
  const origins = d?.allowedOrigins ?? []
  const sites = origins.length ? origins.map((o) => o.replace(/^https?:\/\//, '')).join(', ') : t(m.yourSite)
  const where = f.kind === 'hosted' ? t(m.whereHosted, { url: hostPath(f.hostedUrl) }) : f.kind === 'popup' ? t(m.wherePopup, { sites }) : t(m.whereEmbedded, { sites })
  return (
    <tr>
      <td className="font-semibold">
        <Link to="/w/$workspace/forms" params={{ workspace: slug }} search={{ form: f.id }} className="sv-name">
          {f.name}
        </Link>
      </td>
      <td>
        <span className="badge mut">{t(m.typeSignup)}</span>
      </td>
      <td className="sv-where">{where}</td>
      <td>
        <span className={`badge ${f.status === 'live' ? 'ok' : 'mut'}`}>{f.status === 'live' ? t(m.formLive) : f.status === 'paused' ? t(m.formPaused) : t(m.formDraft)}</span>
      </td>
      <td className="nowrap">{d ? t(m.submissions, { count: d.stats.submissions, days: d.stats.days }) : <span className="k">…</span>}</td>
      <td className="k">—</td>
    </tr>
  )
}

function SurveyRow({ s, slug }: { s: Survey; slug: string }) {
  const t = useT()
  return (
    <tr>
      <td className="font-semibold">
        <Link to="/w/$workspace/forms/surveys/$surveyId" params={{ workspace: slug, surveyId: s.id }} className="sv-name">
          {s.name}
        </Link>
      </td>
      <td>
        <span className="badge info">{t(m.typeSurvey)}</span>
      </td>
      <td className="sv-where">{t(m.whereHosted, { url: hostPath(s.url) })}</td>
      <td>
        <SurveyStatus s={s} />
      </td>
      <td className="nowrap">{t(m.startedCompleted, { started: count(s.started), completed: count(s.completed) })}</td>
      <td className="nowrap">
        {s.createdBy ? (
          <span className="sv-owner">
            <span className="sv-av" aria-hidden="true">
              {initials(s.createdBy)}
            </span>
            {s.createdBy}
          </span>
        ) : (
          <span className="k">—</span>
        )}
      </td>
    </tr>
  )
}

/** "New form…": a signup form or a survey. */
export function NewFormDialog({ open, onOpenChange, onSignup }: { open: boolean; onOpenChange: (o: boolean) => void; onSignup: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [kind, setKind] = useState<'signup' | 'survey'>('survey')
  const [busy, setBusy] = useState(false)
  const go = async () => {
    if (kind === 'signup') {
      onSignup()
      return
    }
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/surveys', { params: { path: { workspace: slug } }, body: { name: t(m.untitled) } })
      onOpenChange(false)
      if (data) void navigate({ to: '/w/$workspace/forms/surveys/$surveyId', params: { workspace: slug, surveyId: data.id } })
    } catch (e) {
      toast({ kind: 'error', title: t(m.createFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={t(m.newFormCrumb)}
      title={t(m.newFormTitle)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void go()}>
            {kind === 'survey' ? t(m.createSurvey) : t(m.continue)}
          </Button>
        </>
      }
    >
      <fieldset className="sv-kinds">
        <legend className="k">{t(m.whatKind)}</legend>
        {(['signup', 'survey'] as const).map((k) => (
          <label key={k} className={`sv-kind${kind === k ? ' on' : ''}`}>
            <input type="radio" name="sv-kind" checked={kind === k} onChange={() => setKind(k)} />
            <b>{k === 'signup' ? t(m.kindSignup) : t(m.kindSurvey)}</b>
            <span className="k">{k === 'signup' ? t(m.kindSignupHint) : t(m.kindSurveyHint)}</span>
          </label>
        ))}
      </fieldset>
    </Dialog>
  )
}
