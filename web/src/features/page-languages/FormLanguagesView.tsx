import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { FormStatusBadge, FormsTabs, hostPath, inputOf, throwIfForbidden, useInvalidateForms, type FormInput, type Overview, type SignupForm } from '@/features/forms/common'
import { FormPreview, type PreviewLanguage } from '@/features/forms/FormPreview'
import { T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { dayMonthDate, dayMonthYear } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { BuiltinLabelsSection, CopyStatusBadge, CopySwitcher, ShippedLabelsRow, useLanguageName, type CopyStatus } from './copy'
import { mf } from './messages'

type FormCopy = Schemas['SignupFormCopy']
type FormCopyLanguage = Schemas['SignupFormCopyLanguage']
type CopyText = Schemas['SignupFormCopyText']
type Mode = FormCopy['languageMode']

/** The form's copy of every language (FormBuilderLanguages). */
export function useFormCopy(formId: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}/copy', { params: { path: { workspace: slug, form: formId } } })
}

/**
 * The form builder in a language (FormBuilderLanguages, FormsStates): the
 * form language, the copy in the language picked under "Copy in" with the
 * preview in it, and each language's status, built-in labels and consent
 * evidence.
 */
export function FormLanguagesView({ form, overview, language }: { form: SignupForm; overview: Overview; language: string }) {
  const t = useT()
  const copy = useFormCopy(form.id)
  throwIfForbidden(copy.error)
  if (copy.isError) {
    return (
      <Page title={t(mf.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(mf.loadFailed)} actions={<Button onClick={() => copy.refetch()}>{t(mf.retry)}</Button>}>
          {errorMessage(copy.error)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!copy.data) {
    return (
      <Page title={t(mf.pageTitle)} filters={<FormsTabs />}>
        <div className="card pl-builder" aria-busy="true">
          <span className="vh" role="status">
            {t(mf.loading)}
          </span>
          <div className="pl-pane pl-left">
            <div className="pl-sec">
              <Skeleton h={18} w="70%" />
            </div>
            <div className="pl-sec">
              <Skeleton h={58} />
              <Skeleton h={58} />
              <Skeleton h={58} />
            </div>
          </div>
          <div className="pl-center">
            <div className="pl-stage">
              <Skeleton h={520} r={16} />
            </div>
          </div>
          <div className="pl-pane pl-right">
            <div className="pl-sec">
              <Skeleton h={160} />
            </div>
          </div>
        </div>
      </Page>
    )
  }
  const data = copy.data
  const code = data.languages.some((l) => l.code === language) ? language : data.copyLanguage
  const lang = data.languages.find((l) => l.code === code)!
  return <LanguageEditor key={`${code}:${JSON.stringify(lang.copy)}:${data.languageMode}:${data.fixedLanguage}`} form={form} overview={overview} data={data} code={code} />
}

const emptyText = (c: CopyText): CopyText => ({ ...c, fields: { ...(c.fields ?? {}) } })

function sameText(a: CopyText, b: CopyText): boolean {
  return JSON.stringify({ ...a, fields: sorted(a.fields) }) === JSON.stringify({ ...b, fields: sorted(b.fields) })
}

function sorted(m: Record<string, string> | undefined): Record<string, string> {
  return Object.fromEntries(Object.entries(m ?? {}).filter(([, v]) => v !== '').sort(([a], [b]) => a.localeCompare(b)))
}

function LanguageEditor({ form, overview, data, code }: { form: SignupForm; overview: Overview; data: FormCopy; code: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const name = useLanguageName()
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason)
  const lang = data.languages.find((l) => l.code === code)!
  const source = data.languages.find((l) => l.isSource) ?? data.languages[0]!
  const initial = useMemo(() => emptyText(lang.copy), [lang])
  const [text, setText] = useState<CopyText>(initial)
  const [mode, setMode] = useState<{ mode: Mode; fixed: string }>({ mode: data.languageMode, fixed: data.fixedLanguage })
  const [started, setStarted] = useState(lang.isSource || lang.status !== 'not_set_up')
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const consentRef = useRef<HTMLTextAreaElement>(null)
  const params = { path: { workspace: slug, form: form.id } }
  const saveCopy = $api.useMutation('put', '/api/v1/workspaces/{workspace}/forms/{form}/copy/{language}')
  const saveMode = $api.useMutation('put', '/api/v1/workspaces/{workspace}/forms/{form}/language')
  const confirm = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/copy/{language}/consent-reviewed')
  const dirtyText = !sameText(text, initial)
  const dirtyMode = mode.mode !== data.languageMode || (mode.mode === 'fixed' && mode.fixed !== data.fixedLanguage)
  const dirty = dirtyText || dirtyMode
  const fallback = name(data.fallback)
  const set = <K extends keyof CopyText>(k: K, v: CopyText[K]) => setText((x) => ({ ...x, [k]: v }))

  const pick = (next: string) => {
    void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { form: form.id, language: next } })
  }

  const publish = async () => {
    if (!dirty) {
      toast({ kind: 'info', title: t(mf.nothingNew), detail: t(mf.nothingNewDetail, { language: name(code) }) })
      return true
    }
    setErrors({})
    try {
      let out: FormCopy | undefined
      if (dirtyMode) {
        out = await saveMode.mutateAsync({ params, body: { mode: mode.mode, fixed: mode.mode === 'fixed' ? mode.fixed : undefined } })
        if (!dirtyText) toast({ kind: 'success', title: t(mf.languageSaved) })
      }
      if (dirtyText) {
        out = await saveCopy.mutateAsync({ params: { path: { ...params.path, language: code } }, body: { ...text, fields: sorted(text.fields) } })
        const now = out.languages.find((l) => l.code === code)
        const detail = lang.isSource
          ? t(mf.savedSource)
          : now?.status === 'live'
            ? t(mf.savedLive, { language: name(code) })
            : t(mf.savedIncomplete, { count: now?.missing ?? 0, language: name(code) })
        toast({ kind: 'success', title: t(mf.saved, { language: name(code) }), detail })
      }
      await invalidate()
      return true
    } catch (e) {
      const fieldErrors: Record<string, string> = {}
      if (e instanceof ApiError) {
        for (const x of e.problem.errors ?? []) {
          if (x.location && x.message) fieldErrors[x.location.replace(/^body\./, '')] = x.message
        }
      }
      setErrors(fieldErrors)
      toast({ kind: 'error', title: dirtyText ? t(mf.saveFailed, { language: name(code) }) : t(mf.languageFailed), detail: errorMessage(e) })
      throw e
    }
  }

  const keepConsent = async () => {
    try {
      await confirm.mutateAsync({ params: { path: { ...params.path, language: code } } })
      toast({ kind: 'success', title: t(mf.reviewConfirmed, { language: name(code) }), detail: t(mf.reviewConfirmedDetail, { source: name(source.code), version: data.consentVersion }) })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mf.reviewFailed, { language: name(code) }), detail: errorMessage(e) })
    }
  }

  const copySource = () => {
    const s = source.copy
    setText({
      heading: s.heading,
      description: s.description,
      button: s.button,
      consentText: s.consentText,
      afterSubmit: s.afterSubmit || source.defaultAfterSubmit,
      fields: { ...(s.fields ?? {}) },
    })
    setStarted(true)
  }

  const site = form.allowedOrigins[0] ?? t(mf.yourSite)
  const where =
    form.kind === 'hosted' ? t(mf.previewHosted, { host: hostPath(form.hostedUrl) }) : t(form.kind === 'popup' ? mf.previewPopup : mf.previewEmbedded, { site })
  // A language that isn't set up shows its visitors the fallback's form:
  // the preview shows that until someone starts writing.
  const shown = started ? lang : (data.languages.find((l) => l.code === data.fallback) ?? source)
  const preview = usePreviewInput(form, data, shown, started ? text : emptyText(shown.copy))
  const previewLang = usePreviewLanguage(shown.code)
  const review = data.languages.filter((l) => l.consentReview)
  const showReview = lang.consentReview || (lang.isSource && review.length > 0)
  const labelsInLeft = !started && lang.labels.shipped

  return (
    <Page
      title={t(mf.pageTitle)}
      filters={<FormsTabs />}
      actions={
        <>
          <FormStatusBadge form={form} />
          <Button onClick={() => (setText(initial), setMode({ mode: data.languageMode, fixed: data.fixedLanguage }))} disabledReason={reason}>
            {t(mf.discard)}
          </Button>
          <Button variant="primary" onClick={() => void publish().catch(() => undefined)} loading={saveCopy.isPending || saveMode.isPending} busyLabel={t(mf.saving)} disabledReason={reason}>
            {t(mf.save)}
          </Button>
        </>
      }
    >
      <div className="card pl-builder">
        <aside className="pl-pane pl-left" aria-label={t(mf.settingsAside)}>
          <div className="pl-sec">
            <div className="flex items-center justify-between gap-2.5">
              <span className="font-semibold">{t(mf.formTitle, { name: form.name })}</span>
              <Badge tone="neutral" className="pl-kind">
                {t(form.kind === 'hosted' ? mf.kindHosted : form.kind === 'popup' ? mf.kindPopup : mf.kindEmbedded)}
              </Badge>
            </div>
          </div>
          <div className="pl-sec">
            <fieldset className="pl-fieldset">
              <legend className="k pl-legend">{t(mf.formLanguage)}</legend>
              <ModeOption mode="page" current={mode.mode} readOnly={readOnly} onPick={(m) => setMode((x) => ({ ...x, mode: m }))} title={t(mf.modePage)}>
                <T msg={mf.modePageNote} values={{ mono: (c) => <span className="mono">{c}</span> }} />
              </ModeOption>
              <ModeOption mode="browser" current={mode.mode} readOnly={readOnly} onPick={(m) => setMode((x) => ({ ...x, mode: m }))} title={t(mf.modeBrowser)}>
                {t(mf.modeBrowserNote, { host: hostPath(overview.hostedUrl) })}
              </ModeOption>
              <ModeOption mode="fixed" current={mode.mode} readOnly={readOnly} onPick={(m) => setMode((x) => ({ ...x, mode: m, fixed: x.fixed || data.copyLanguage }))} title={t(mf.modeFixed)}>
                {t(mf.modeFixedNote)}
              </ModeOption>
            </fieldset>
            {mode.mode === 'fixed' ? (
              <select
                className="fb-in"
                aria-label={t(mf.fixedLanguage)}
                value={mode.fixed}
                disabled={readOnly}
                aria-invalid={errors.fixed ? true : undefined}
                onChange={(e) => setMode((x) => ({ ...x, fixed: e.target.value }))}
              >
                {data.languages.map((l) => (
                  <option key={l.code} value={l.code} lang={l.code}>
                    {l.native}
                  </option>
                ))}
              </select>
            ) : null}
            <span className="k">{t(mf.fallbackNote, { fallback })}</span>
          </div>
          <div className="pl-sec last">
            <div className="flex items-center justify-between gap-2">
              <span className="k font-semibold">{t(mf.copyIn)}</span>
              {!lang.isSource && lang.by && lang.at ? <span className="k">{t(mf.copyBy, { name: lang.by, date: dayMonthYear(lang.at) })}</span> : null}
            </div>
            <CopySwitcher languages={data.languages.map(switcherOf)} value={code} onPick={pick} />
            {started ? (
              <CopyFields
                code={code}
                lang={lang}
                source={source}
                data={data}
                text={text}
                set={set}
                setField={(k, v) => setText((x) => ({ ...x, fields: { ...(x.fields ?? {}), [k]: v } }))}
                readOnly={readOnly}
                errors={errors}
                consentRef={consentRef}
              />
            ) : (
              <>
                <div className="note w" role="note">
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="8" cy="8" r="6.2" />
                    <path d="M8 4.8v3.6M8 10.9v.1" />
                  </svg>
                  <div className="pl-note-body flex flex-col gap-1">
                    <span className="font-bold">{t(mf.notSetUpTitle, { language: name(code) })}</span>
                    <div>
                      {lang.labels.shipped ? t(mf.notSetUpText, { language: name(code), fallback }) : t(mf.notSetUpTextLabels, { language: name(code), fallback })}
                      <div className="mt-2 flex flex-wrap gap-2">
                        <Button size="sm" variant="primary" onClick={copySource} disabledReason={reason}>
                          {t(mf.copySource, { language: name(source.code) })}
                        </Button>
                        <Button size="sm" onClick={() => setStarted(true)} disabledReason={reason}>
                          {t(mf.fromScratch)}
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
                {labelsInLeft ? <ShippedLabelsRow code={code} /> : null}
              </>
            )}
          </div>
        </aside>

        <section className="pl-center" aria-label={t(mf.previewWhere, { where, language: shown.native })}>
          <div className="fb-bar">
            <span className="k">
              <T msg={mf.previewWhere} values={{ where, language: shown.native, lang: (c) => <span lang={shown.code}>{c}</span> }} />
            </span>
            <div className="fb-seg ml-auto" role="group" aria-label={t(mf.previewSize)}>
              <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                {t(mf.desktop)}
              </button>
              <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                {t(mf.mobile)}
              </button>
            </div>
          </div>
          <div className="pl-stage">
            <FormPreview form={preview} overview={overview} mobile={device === 'mobile'} width={460} readOnly onToggleDefault={() => undefined} language={previewLang} />
          </div>
        </section>

        <aside className="pl-pane pl-right" aria-label={t(mf.languagesAside)}>
          {showReview ? (
            <ConsentReview data={data} source={source} lang={lang} readOnly={readOnly} onUpdate={() => consentRef.current?.focus()} onKeep={() => void keepConsent()} busy={confirm.isPending} />
          ) : null}
          <CopyByLanguage data={data} />
          {labelsInLeft ? null : (
            <div className="pl-sec">
              <BuiltinLabelsSection code={code} labels={lang.labels} />
            </div>
          )}
          <Evidence lang={lang} />
        </aside>
      </div>
      <UnsavedChangesGuard
        dirty={dirty && !readOnly}
        title={t(mf.leaveTitle)}
        description={t(mf.leaveText, { language: name(code), name: form.name })}
        saveLabel={t(mf.saveAndLeave)}
        onSave={publish}
      />
    </Page>
  )
}

function switcherOf(l: FormCopyLanguage) {
  return { code: l.code, native: l.native, status: l.status as CopyStatus, missing: l.missing }
}

function ModeOption({ mode, current, readOnly, onPick, title, children }: { mode: Mode; current: Mode; readOnly: boolean; onPick: (m: Mode) => void; title: string; children: ReactNode }) {
  const on = mode === current
  return (
    <label className={cn('pl-rad', on && 'on')}>
      <input type="radio" name="form-language" value={mode} checked={on} disabled={readOnly} onChange={() => onPick(mode)} />
      <span className="flex flex-col gap-px font-semibold">
        {title}
        <span className="k font-normal">{children}</span>
      </span>
    </label>
  )
}

/** The copy of the language picked under "Copy in". */
function CopyFields({
  code,
  lang,
  source,
  data,
  text,
  set,
  setField,
  readOnly,
  errors,
  consentRef,
}: {
  code: string
  lang: FormCopyLanguage
  source: FormCopyLanguage
  data: FormCopy
  text: CopyText
  set: <K extends keyof CopyText>(k: K, v: CopyText[K]) => void
  setField: (key: string, v: string) => void
  readOnly: boolean
  errors: Record<string, string>
  consentRef: React.RefObject<HTMLTextAreaElement | null>
}) {
  const t = useT()
  const name = useLanguageName()
  const translation = !lang.isSource
  const hint = (v: string) => (translation && v ? v : undefined)
  const err = (k: string) =>
    errors[k] ? (
      <span className="err" id={`pl-${k}-err`} role="alert">
        {errors[k]}
      </span>
    ) : null
  const aria = (k: string) => ({ 'aria-invalid': errors[k] ? true : undefined, 'aria-describedby': errors[k] ? `pl-${k}-err` : undefined })
  return (
    <>
      <label className="lbl">
        {t(mf.heading)}
        <input className="fb-in" lang={code} value={text.heading} maxLength={120} placeholder={hint(source.copy.heading)} disabled={readOnly} {...aria('heading')} onChange={(e) => set('heading', e.target.value)} />
        {err('heading')}
      </label>
      <label className="lbl">
        {t(mf.description)}
        <textarea
          className="fb-in pl-area"
          lang={code}
          value={text.description}
          maxLength={500}
          placeholder={hint(source.copy.description)}
          disabled={readOnly}
          {...aria('description')}
          onChange={(e) => set('description', e.target.value)}
        />
        {err('description')}
      </label>
      <label className="lbl">
        {t(mf.button)}
        <input className="fb-in" lang={code} value={text.button} maxLength={40} placeholder={hint(source.copy.button)} disabled={readOnly} {...aria('button')} onChange={(e) => set('button', e.target.value)} />
        {err('button')}
      </label>
      <label className="lbl">
        <span className="flex items-center justify-between gap-2">
          {t(mf.consent)}
          {lang.consentVersion > 0 ? (
            <Badge tone="neutral" className="pl-badge-row">
              {t(mf.consentVersion, { language: name(code), version: lang.consentVersion })}
            </Badge>
          ) : null}
        </span>
        <textarea
          ref={consentRef}
          className="fb-in pl-area"
          lang={code}
          value={text.consentText}
          maxLength={600}
          placeholder={hint(source.copy.consentText)}
          disabled={readOnly}
          {...aria('consentText')}
          onChange={(e) => set('consentText', e.target.value)}
        />
        <span className="k font-medium">{translation ? t(mf.consentNote, { source: name(source.code), version: data.consentVersion }) : t(mf.consentNoteSource)}</span>
        {err('consentText')}
      </label>
      <label className="lbl">
        {t(mf.afterSubmit)}
        <input className="fb-in" lang={code} value={text.afterSubmit} maxLength={300} placeholder={lang.defaultAfterSubmit} disabled={readOnly} {...aria('afterSubmit')} onChange={(e) => set('afterSubmit', e.target.value)} />
        {err('afterSubmit')}
      </label>
      {data.fields.length ? (
        <div className="flex flex-col gap-2">
          <span className="k font-semibold">{t(mf.fieldLabels)}</span>
          {data.fields.map((f) => (
            <label key={f.key} className="lbl">
              {f.source}
              <input className="fb-in" lang={code} value={text.fields?.[f.key] ?? ''} maxLength={80} placeholder={hint(f.source)} disabled={readOnly} onChange={(e) => setField(f.key, e.target.value)} />
            </label>
          ))}
        </div>
      ) : null}
    </>
  )
}

/** "Copy by language": each language's status, who wrote it and its consent text's version. */
export function CopyByLanguage({
  data,
  linkFor,
  variant = 'languages',
}: {
  data: FormCopy
  linkFor?: (code: string) => { form: string; language: string }
  variant?: 'languages' | 'builder'
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()
  const fallback = name(data.fallback)
  const sub = (l: FormCopyLanguage) => {
    if (l.isDefault || l.isSource) return l.consentVersion > 0 ? t(mf.rowDefault, { version: l.consentVersion }) : t(mf.rowDefaultNoConsent)
    if (l.status === 'not_set_up') return t(mf.rowNotSetUp, { fallback })
    if (l.status === 'incomplete') return t(mf.rowIncomplete, { count: l.missing, fallback })
    if (l.by && l.at) {
      return l.consentVersion > 0
        ? t(mf.rowWritten, { name: l.by, date: dayMonthYear(l.at), version: l.consentVersion })
        : t(mf.rowWrittenNoConsent, { name: l.by, date: dayMonthYear(l.at) })
    }
    return ''
  }
  return (
    <div className={variant === 'builder' ? 'fb-sec last pl-langs' : 'pl-sec'}>
      <span className={variant === 'builder' ? 'fb-title' : 'k'}>{t(mf.copyByLanguage)}</span>
      <div>
        {data.languages.map((l) => (
          <div key={l.code} className="pl-row">
            <span className="flex min-w-0 flex-col gap-px">
              {linkFor ? (
                <Link
                  to="/w/$workspace/forms"
                  params={{ workspace: slug }}
                  search={linkFor(l.code)}
                  className="font-semibold"
                  lang={l.code}
                  aria-label={t(mf.rowEdit, { language: name(l.code) })}
                >
                  {l.native}
                </Link>
              ) : (
                <span className="font-semibold" lang={l.code}>
                  {l.native}
                </span>
              )}
              <span className="k">{sub(l)}</span>
            </span>
            <CopyStatusBadge status={l.status as CopyStatus} missing={l.missing} />
          </div>
        ))}
      </div>
    </div>
  )
}

/** "Consent text · after the English text changes to v4" (FormsStates). */
function ConsentReview({
  data,
  source,
  lang,
  readOnly,
  onUpdate,
  onKeep,
  busy,
}: {
  data: FormCopy
  source: FormCopyLanguage
  lang: FormCopyLanguage
  readOnly: boolean
  onUpdate: () => void
  onKeep: () => void
  busy: boolean
}) {
  const t = useT()
  const name = useLanguageName()
  const rows = data.languages.filter((l) => l.isSource || l.consentVersion > 0)
  return (
    <div className="pl-sec">
      <span className="font-semibold">{t(mf.reviewTitle, { source: name(source.code), version: data.consentVersion })}</span>
      <div>
        {rows.map((l) => (
          <div key={l.code} className="pl-row">
            <span className="flex min-w-0 flex-col gap-px">
              <span className="font-semibold" lang={l.code}>
                {l.native}
              </span>
              <span className="k">
                {l.isSource
                  ? data.consentChangedBy
                    ? t(mf.reviewSource, { version: data.consentVersion, name: data.consentChangedBy })
                    : t(mf.reviewSourceUnknown, { version: data.consentVersion })
                  : t(mf.reviewWritten, { version: l.consentVersion, source: name(source.code), for: l.consentFor })}
              </span>
            </span>
            {l.consentReview ? (
              <Badge tone="warn" className="pl-badge-row">
                {t(mf.reviewBadge)}
              </Badge>
            ) : (
              <Badge tone="ok" className="pl-badge-row">
                {t(mf.statusLive)}
              </Badge>
            )}
          </div>
        ))}
      </div>
      <span className="k">{t(mf.reviewNote)}</span>
      {lang.consentReview && !readOnly ? (
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={onUpdate}>
            {t(mf.reviewUpdate, { language: name(lang.code) })}
          </Button>
          <Button size="sm" onClick={onKeep} loading={busy}>
            {t(mf.reviewStillRight, { language: name(lang.code) })}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

/** "Consent evidence": what each signup keeps, with the latest consent in the language. */
function Evidence({ lang }: { lang: FormCopyLanguage }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()
  return (
    <div className="pl-sec last">
      <span className="k">{t(mf.evidenceTitle)}</span>
      <span className="pl-evidence">
        {lang.consentVersion > 0 ? t(mf.evidenceText, { language: name(lang.code), version: lang.consentVersion }) : t(mf.evidenceTextNone)}
      </span>
      {lang.evidence ? (
        <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: lang.evidence.contactId }} className="lnk">
          {t(mf.evidenceLink, { name: lang.evidence.name, date: dayMonthDate(lang.evidence.at) })}
        </Link>
      ) : null}
    </div>
  )
}

/** The form as the preview shows it in the language, with the unsaved copy. */
function usePreviewInput(form: SignupForm, data: FormCopy, lang: FormCopyLanguage, text: CopyText): FormInput {
  const labels = useLabelTexts(lang.code)
  return useMemo(() => {
    const base = inputOf(form)
    const content = { ...base.content, heading: text.heading, description: text.description, button: text.button, afterSubmitText: text.afterSubmit }
    if (!lang.isSource) {
      content.footnote = ''
      content.listsLabel = labels['form.lists'] ?? content.listsLabel
      content.emailPlaceholder = labels['form.emailPlaceholder'] ?? content.emailPlaceholder
      content.namePlaceholder = labels['form.namePlaceholder'] ?? content.namePlaceholder
    }
    const fields = base.fields.map((f) => (data.fields.some((x) => x.key === f.key) ? { ...f, label: text.fields?.[f.key] || (lang.isSource ? f.label : '') } : f))
    return { ...base, content, consentText: text.consentText, fields }
  }, [form, data.fields, lang.isSource, labels, text])
}

/** The built-in labels' texts in a language, by key. */
function useLabelTexts(code: string): Record<string, string> {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/labels/{language}', { params: { path: { workspace: slug, language: code } } }, { staleTime: 60_000 })
  return useMemo(() => {
    const out: Record<string, string> = {}
    for (const l of q.data?.labels ?? []) out[l.key] = l.text || l.source
    return out
  }, [q.data])
}

/** The built-in labels, list names and privacy link of the preview's language. */
function usePreviewLanguage(code: string): PreviewLanguage | undefined {
  const { slug } = useWorkspace()
  const labels = useLabelTexts(code)
  const pages = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/copy', { params: { path: { workspace: slug }, query: { language: code } } }, { staleTime: 60_000 })
  return useMemo(() => {
    if (!Object.keys(labels).length) return undefined
    const listNames: Record<string, string> = {}
    let privacyUrl = ''
    for (const s of pages.data?.strings ?? []) {
      if (s.key.endsWith('.name') && s.listId && s.text) listNames[s.listId] = s.text
      if (s.key === 'privacyUrl' && s.text) privacyUrl = s.text
    }
    return {
      lang: code,
      fields: { email: labels['form.email'] ?? '', first_name: labels['form.firstName'] ?? '', last_name: labels['form.lastName'] ?? '', company: labels['form.company'] ?? '' },
      optional: labels['form.optional'] ?? 'optional',
      choose: labels['form.choose'] ?? 'Choose…',
      privacy: labels['form.privacy'] ?? 'privacy policy',
      privacyUrl,
      listNames,
      confirmHeading: labels['form.confirm.heading'] ?? '',
      confirmText: labels['form.confirm.short'] ?? '',
      subscribedHeading: labels['form.subscribed.heading'] ?? '',
      subscribedText: labels['form.subscribed.short'] ?? '',
    }
  }, [code, labels, pages.data])
}
