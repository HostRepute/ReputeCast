import { useNavigate } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input } from '@/components/ui/Field'
import { Menu, MenuItem, MenuSeparator, Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { WelcomePromptCard } from '@/features/welcome-prompt/WelcomePromptCard'
import { ago, andList, count, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { Icon } from '@/components/icons/Icon'
import { FormAddressCheck, FormAskedToFix, FormChecksPaused } from '@/features/verification/FormCheck'
import {
  copyText,
  embedSnippet,
  FormStatusBadge,
  FormsTabs,
  hostPath,
  inputOf,
  sameInput,
  throwIfForbidden,
  useFormsOverview,
  useInvalidateForms,
  type FormInput,
  type Overview,
  type SignupForm,
} from './common'
import { FormPreview } from './FormPreview'
import { BlockedOriginAlert, FormsEmpty, FormsLoading, TurnstileAlert } from './FormsStates'
import { FormDownloadSection } from './LeadMagnet'
import { FormPagesSection, OwnOriginsNote } from './UsedOnPages'
import { CopyByLanguage, FormLanguagesView, useFormCopy } from '@/features/page-languages/FormLanguagesView'
import { useWorkspaceLanguages } from '@/features/page-languages/contacts'
import { FormRefNote } from '@/features/grow/FormRefNote'
import { SmsConsentSettings, SmsPhoneToggle } from '@/features/sms/FormFields'
import { useChannelSetUp } from '@/features/channels/hooks'
import { m as promptMsg } from '@/features/push/prompt.messages'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './builder.messages'

// Push prompts are forms too, with their own designer (X22).
type Kind = Exclude<Schemas['SignupFormSummary']['kind'], 'push'>

const kindLabels: Record<Kind, MessageDescriptor> = { embedded: m.kindEmbedded, popup: m.kindPopup, hosted: m.kindHosted }

/** The built-in contact fields, named in the app's language (the preview shows the form's own words). */
const builtinNames: Record<string, MessageDescriptor> = { first_name: m.fieldFirstName, last_name: m.fieldLastName, company: m.fieldCompany }

/** Built-in fields in the order a form shows them; custom fields follow. */
const fieldOrder = ['email', 'first_name', 'last_name', 'company', 'phone']

function sortFields(fields: FormInput['fields']): FormInput['fields'] {
  const rank = (k: string) => {
    const i = fieldOrder.indexOf(k)
    return i < 0 ? fieldOrder.length : i
  }
  return [...fields].sort((a, b) => rank(a.key) - rank(b.key))
}

/**
 * 7.12 Signup form builder (board FormBuilder) with the FormsStates states;
 * with a language, the builder in that language (FormBuilderLanguages).
 */
export function FormBuilderPage({ formId, language }: { formId?: string; language?: string }) {
  const t = useT()
  const overview = useFormsOverview()
  throwIfForbidden(overview.error)
  const [creating, setCreating] = useState(false)
  const createReason = usePermissionReason('forms:edit')
  if (overview.isError) {
    return (
      <Page title={t(m.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(m.loadFormsFailed)} actions={<Button onClick={() => overview.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(overview.error)} {t(m.formsKeepWorking)}
        </ErrorPanel>
      </Page>
    )
  }
  if (overview.isPending) {
    return (
      <Page title={t(m.pageTitle)} filters={<FormsTabs />}>
        <FormsLoading />
      </Page>
    )
  }
  const o = overview.data
  if (o.forms.length === 0) {
    return (
      <Page title={t(m.pageTitle)} filters={<FormsTabs />}>
        <FormsEmpty hostedUrl={o.hostedUrl} hostedLabel={hostPath(o.hostedUrl)} onCreate={() => setCreating(true)} createReason={createReason} />
        <CreateFormDialog open={creating} onOpenChange={setCreating} />
      </Page>
    )
  }
  const current = o.forms.find((f) => f.id === formId) ?? o.forms[0]!
  return <FormLoader key={current.id} formId={current.id} overview={o} language={language} />
}

/** Loads one form; its editor starts over when the saved form changes. */
function FormLoader({ formId, overview, language }: { formId: string; overview: Overview; language?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const detail = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}', { params: { path: { workspace: slug, form: formId } } })
  throwIfForbidden(detail.error)
  if (detail.isError) {
    return (
      <Page title={t(m.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(m.loadFormFailed)} actions={<Button onClick={() => detail.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(detail.error)} {t(m.nothingChanged)}
        </ErrorPanel>
      </Page>
    )
  }
  if (detail.isPending) {
    return (
      <Page title={t(m.pageTitle)} filters={<FormsTabs />}>
        <FormsLoading label={t(m.buildingPreview)} />
      </Page>
    )
  }
  if (language) return <FormLanguagesView form={detail.data} overview={overview} language={language} />
  return <Editor key={`${detail.data.id}:${detail.data.updatedAt}`} form={detail.data} overview={overview} />
}

function Editor({ form, overview }: { form: SignupForm; overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason)
  const initial = useMemo(() => inputOf(form), [form])
  const [draft, setDraft] = useState<FormInput>(initial)
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [creating, setCreating] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const dirty = !sameInput(draft, initial)
  const params = { path: { workspace: slug, form: form.id } }
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/forms/{form}')
  const setStatus = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/status')
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/forms/{form}')

  const set = <K extends keyof FormInput>(k: K, v: FormInput[K]) => setDraft((d) => ({ ...d, [k]: v }))
  const setSpam = (patch: Partial<FormInput['spam']>) => setDraft((d) => ({ ...d, spam: { ...d.spam, ...patch } }))
  const setContent = (patch: Partial<FormInput['content']>) => setDraft((d) => ({ ...d, content: { ...d.content, ...patch } }))
  const hasField = (key: string) => draft.fields.some((f) => f.key === key)
  const toggleField = (key: string, on: boolean) =>
    setDraft((d) => ({ ...d, fields: on ? sortFields([...d.fields.filter((f) => f.key !== key), { key, label: '', required: false }]) : d.fields.filter((f) => f.key !== key) }))

  const publish = async () => {
    if (!dirty && form.status === 'live') {
      toast({ kind: 'info', title: tt(m.nothingNew), detail: tt(m.liveAlready, { name: form.name }) })
      return true
    }
    setErrors({})
    try {
      await save.mutateAsync({ params, body: { ...draft, publish: true } })
      toast({ kind: 'success', title: tt(m.published, { where: hostPath(form.hostedUrl) }), detail: tt(m.publishedBody) })
      await invalidate()
      return true
    } catch (e) {
      const fieldErrors: Record<string, string> = {}
      if (e instanceof ApiError) {
        for (const x of e.problem.errors ?? []) {
          if (x.location && x.message) fieldErrors[x.location.replace(/^body\./, '').replace(/\[\d+\].*$/, '')] = x.message
        }
      }
      setErrors(fieldErrors)
      toast({ kind: 'error', title: tt(m.publishFailed), detail: errorMessage(e), retry: () => void publish() })
      throw e
    }
  }

  const pickForm = (id: string) => {
    if (id === '__new') {
      setCreating(true)
      return
    }
    // A push prompt opens in its own designer (X22).
    if (overview.forms.find((f) => f.id === id)?.kind === 'push') {
      void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { prompt: id } })
      return
    }
    void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { form: id } })
  }

  const togglePause = async () => {
    const next = form.status === 'paused' ? 'live' : 'paused'
    try {
      await setStatus.mutateAsync({ params, body: { status: next } })
      toast({
        kind: 'success',
        title: next === 'paused' ? tt(m.paused, { name: form.name }) : tt(m.resumed, { name: form.name }),
        detail: next === 'paused' ? tt(m.pausedBody) : undefined,
      })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.changeFailed), detail: errorMessage(e), retry: () => void togglePause() })
    }
  }

  const where = useMemo(() => {
    const site = draft.allowedOrigins[0]
    if (draft.kind === 'hosted') return tt(m.whereHosted, { url: hostPath(form.hostedUrl) })
    return tt(draft.kind === 'popup' ? m.wherePopup : m.whereEmbedded, { named: site ? 'yes' : 'no', site: site ?? '' })
  }, [draft.kind, draft.allowedOrigins, form.hostedUrl])

  const blocked = form.blockedOrigins
  const turnstileBroken = draft.spam.turnstile && overview.turnstile.status === 'error'

  return (
    <Page
      title={t(m.pageTitle)}
      filters={<FormsTabs />}
      actions={
        <>
          <FormStatusBadge form={form} />
          <Menu
            label={t(m.formActions)}
            trigger={
              <button type="button" className="btn ic" aria-label={t(m.moreActions, { name: form.name })}>
                <Icon name="more" size={16} />
              </button>
            }
          >
            <MenuItem onSelect={() => setCreating(true)} disabled={readOnly}>
              {t(m.newForm)}
            </MenuItem>
            <MenuItem onSelect={() => setRenaming(true)} disabled={readOnly}>
              {t(m.rename)}
            </MenuItem>
            {form.status !== 'draft' ? (
              <MenuItem onSelect={() => void togglePause()} disabled={readOnly}>
                {form.status === 'paused' ? t(m.resumeSignups) : t(m.pauseSignups)}
              </MenuItem>
            ) : null}
            <MenuSeparator />
            <MenuItem onSelect={() => setDeleting(true)} disabled={readOnly} className="text-bad-strong">
              {t(m.deleteForm)}
            </MenuItem>
          </Menu>
          <Button onClick={() => setDraft(initial)} disabledReason={reason}>
            {t(m.discard)}
          </Button>
          <Button variant="primary" onClick={() => void publish().catch(() => undefined)} loading={save.isPending} busyLabel={t(m.publishing)} disabledReason={reason}>
            {t(m.savePublish)}
          </Button>
        </>
      }
      contentClassName="fb-main"
    >
      <div className="fb-cols">
        <aside className="fb-left" aria-label={t(m.formSettings)}>
          <div className="fb-sec">
            <div className="fb-row flex items-center justify-between gap-2.5">
              <span className="font-semibold">{t(m.formName, { name: form.name })}</span>
              <select aria-label={t(m.signupForm)} className="fb-in h-[30px]! w-[140px]!" value={form.id} onChange={(e) => pickForm(e.target.value)}>
                {overview.forms.map((f) => (
                  <option key={f.id} value={f.id}>
                    {f.name}
                  </option>
                ))}
                {readOnly ? null : <option value="__new">{t(m.newFormOption)}</option>}
              </select>
            </div>
          </div>
          <TypeSection draft={draft} form={form} readOnly={readOnly} onKind={(k) => set('kind', k)} onContent={setContent} />
          <div className="fb-sec">
            <span className="k">{t(m.fields)}</span>
            <label className="fb-check">
              <input type="checkbox" checked disabled />
              {t(m.emailRequired)}
            </label>
            {(['first_name', 'company'] as const).map((key) => (
              <label key={key} className="fb-check">
                <input type="checkbox" checked={hasField(key)} disabled={readOnly} onChange={(e) => toggleField(key, e.target.checked)} />
                {hasField(key) ? t(m.fieldOptional, { field: t(builtinNames[key]!) }) : t(builtinNames[key]!)}
              </label>
            ))}
            {/* Phase 8: the phone field, once texting is set up. */}
            <SmsPhoneToggle
              on={hasField('phone')}
              required={draft.fields.some((f) => f.key === 'phone' && f.required)}
              readOnly={readOnly}
              onToggle={(on) => {
                toggleField('phone', on)
                if (!on) setContent({ smsConsent: false })
              }}
            />
            {draft.fields
              .filter((f) => f.key !== 'email' && f.key !== 'first_name' && f.key !== 'company' && f.key !== 'phone')
              .map((f) => (
                <label key={f.key} className="fb-check">
                  <input type="checkbox" checked disabled={readOnly} onChange={() => toggleField(f.key, false)} />
                  {t(f.required ? m.fieldRequired : m.fieldOptional, {
                    field: builtinNames[f.key] ? t(builtinNames[f.key]!) : (overview.customFields.find((c) => c.key === f.key)?.label ?? f.key),
                  })}
                </label>
              ))}
            <label className="fb-check">
              <input type="checkbox" checked={draft.listChoice} disabled={readOnly} onChange={(e) => set('listChoice', e.target.checked)} />
              {t(m.pickLists)}
            </label>
            <AddFieldMenu draft={draft} overview={overview} readOnly={readOnly} onAdd={(key) => toggleField(key, true)} />
          </div>
          <div className="fb-sec">
            <span className="k">{t(m.lists)}</span>
            {overview.lists.map((l) => {
              const offered = draft.listIds.includes(l.id)
              const isDefault = draft.defaultListIds.includes(l.id)
              return (
                <label key={l.id} className="fb-check">
                  <input
                    type="checkbox"
                    checked={offered}
                    disabled={readOnly}
                    onChange={(e) =>
                      setDraft((d) => ({
                        ...d,
                        listIds: e.target.checked ? [...d.listIds, l.id] : d.listIds.filter((x) => x !== l.id),
                        defaultListIds: e.target.checked ? d.defaultListIds : d.defaultListIds.filter((x) => x !== l.id),
                      }))
                    }
                  />
                  {offered && isDefault ? t(m.listDefault, { list: l.name }) : l.name}
                </label>
              )
            })}
            {errors.listIds ? (
              <span className="err" role="alert">
                {errors.listIds}
              </span>
            ) : null}
          </div>
          <div className="fb-sec">
            <span className="k">{t(m.consent)}</span>
            <label className="fb-check start">
              <input type="checkbox" checked={draft.doubleOptIn} disabled={readOnly} onChange={(e) => set('doubleOptIn', e.target.checked)} />
              <span>
                {t(m.doubleOptIn)} <span className="k">· {doubleOptInNote(draft.doubleOptIn, overview.systemMail ? overview.systemProvider : '')}</span>
              </span>
            </label>
            <label className="lbl">
              {t(m.consentText)}
              <textarea
                className="fb-in"
                value={draft.consentText}
                maxLength={600}
                disabled={readOnly}
                aria-invalid={errors.consentText ? true : undefined}
                onChange={(e) => set('consentText', e.target.value)}
              />
            </label>
            <span className="k">
              {t(m.evidence)}
              {draft.consentText.trim() !== form.consentText && form.consentText ? ` ${t(m.newVersion, { version: form.consentVersion + 1 })}` : ''}
            </span>
          </div>
          {/* Phase 8: the SMS consent box, with its own text. */}
          <SmsConsentSettings
            content={draft.content}
            saved={form.content}
            hasPhone={hasField('phone')}
            readOnly={readOnly}
            error={errors['content.smsConsent'] ?? errors['content.smsConsentText']}
            onContent={setContent}
          />
          <div className="fb-sec">
            <span className="k">{t(m.spam)}</span>
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.honeypot} disabled={readOnly} onChange={(e) => setSpam({ honeypot: e.target.checked })} />
              {t(m.honeypot)}
            </label>
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.turnstile} disabled={readOnly} onChange={(e) => setSpam({ turnstile: e.target.checked })} />
              {t(m.turnstile)}
            </label>
            {draft.spam.turnstile && !overview.turnstile.hasSecret ? <TurnstileKeys readOnly={readOnly} /> : null}
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.blockDisposable} disabled={readOnly} onChange={(e) => setSpam({ blockDisposable: e.target.checked })} />
              {t(m.rejectDisposable)}
            </label>
            <FormAddressCheck formKey={form.id} formName={form.name} />
          </div>
          <div className="fb-sec last">
            <span className="k">{t(m.afterSubmit)}</span>
            <select aria-label={t(m.afterSubmit)} className="fb-in" value={draft.afterSubmit} disabled={readOnly} onChange={(e) => set('afterSubmit', e.target.value as FormInput['afterSubmit'])}>
              <option value="message">{draft.doubleOptIn ? t(m.showCheckInbox) : t(m.showSubscribed)}</option>
              <option value="redirect">{t(m.redirect)}</option>
            </select>
            {draft.afterSubmit === 'redirect' ? (
              <>
                <input
                  className="fb-in"
                  aria-label={t(m.redirectUrl)}
                  placeholder={'https://acme.io/thanks'}
                  value={draft.redirectUrl}
                  disabled={readOnly}
                  aria-invalid={errors.redirectUrl ? true : undefined}
                  aria-describedby={errors.redirectUrl ? 'redirect-err' : undefined}
                  onChange={(e) => set('redirectUrl', e.target.value)}
                />
                {errors.redirectUrl ? (
                  <span className="err" id="redirect-err" role="alert">
                    {errors.redirectUrl}
                  </span>
                ) : null}
              </>
            ) : null}
          </div>
          <FormDownloadSection form={form} readOnly={readOnly} doubleOptIn={draft.doubleOptIn} />
        </aside>

        <section className="fb-center" aria-label={t(m.livePreview)}>
          <div className="fb-bar">
            <span className="k">{t(m.livePreviewWhere, { where })}</span>
            <div className="fb-seg ml-auto" role="group" aria-label={t(m.previewSize)}>
              <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                {t(m.desktop)}
              </button>
              <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                {t(m.mobile)}
              </button>
            </div>
          </div>
          <WelcomePromptCard where={{ context: 'form', id: form.id }} className="mx-3.5 mt-3" raised />
          <div className="fb-stage">
            {blocked.map((b) => (
              <BlockedOriginAlert key={b.origin} form={form} blocked={b} />
            ))}
            {turnstileBroken ? <TurnstileAlert turnstile={overview.turnstile} /> : null}
            <FormChecksPaused formKey={form.id} />
            <FormPreview
              form={draft}
              overview={overview}
              mobile={device === 'mobile'}
              readOnly={readOnly}
              onToggleDefault={(id) =>
                setDraft((d) => ({ ...d, defaultListIds: d.defaultListIds.includes(id) ? d.defaultListIds.filter((x) => x !== id) : [...d.defaultListIds, id] }))
              }
            />
          </div>
        </section>

        <aside className="fb-right" aria-label={t(m.embedAndNumbers)}>
          <EmbedSection form={form} />
          <div className="fb-sec">
            <span className="fb-title">{t(m.hostedPage)}</span>
            <input className="fb-in mono" aria-label={t(m.hostedPageUrl)} value={form.hostedUrl} readOnly />
            <div className="flex gap-1.5">
              <Button
                size="sm"
                onClick={async () =>
                  (await copyText(form.hostedUrl))
                    ? toast({ kind: 'success', title: tt(m.linkCopied), detail: hostPath(form.hostedUrl) })
                    : toast({ kind: 'error', title: tt(m.copyFailed), detail: tt(m.copyYourself) })
                }
              >
                {t(m.copyLink)}
              </Button>
              <a className="btn sm" href={form.hostedUrl} target="_blank" rel="noreferrer">
                {t(m.open)}
              </a>
            </div>
          </div>
          <div className="fb-sec">
            <span className="fb-title">{t(m.allowedDomains)}</span>
            <span className="k">{t(m.otherOrigins)}</span>
            <OwnOriginsNote origins={overview.ownOrigins} />
            <div className="flex flex-wrap gap-1.5">
              {draft.allowedOrigins.map((o) => (
                <span key={o} className="badge mut">
                  {o}
                </span>
              ))}
              <DomainsEditor origins={draft.allowedOrigins} readOnly={readOnly} onChange={(next) => set('allowedOrigins', next)} />
            </div>
            {errors.allowedOrigins ? (
              <span className="err" role="alert">
                {errors.allowedOrigins}
              </span>
            ) : null}
          </div>
          <FormPagesSection form={form} />
          <NumbersSection form={form} systemMail={overview.systemMail} />
          <FormLanguagesSection form={form} />
        </aside>
      </div>
      <UnsavedChangesGuard
        dirty={dirty && !readOnly}
        title={t(m.leaveTitle)}
        description={t(m.leaveBody, { name: form.name })}
        saveLabel={t(m.publishLeave)}
        onSave={publish}
      />
      <CreateFormDialog open={creating} onOpenChange={setCreating} />
      <RenameDialog open={renaming} onOpenChange={setRenaming} form={form} />
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        badge={t(m.deleteForm)}
        title={t(m.deleteTitle, { name: form.name })}
        description={t(m.deleteBody)}
        confirmLabel={t(m.deleteForm)}
        busyLabel={t(m.deleting)}
        secondary={
          form.status === 'live' ? (
            <Button
              onClick={() => {
                setDeleting(false)
                void togglePause()
              }}
            >
              {t(m.pauseInstead)}
            </Button>
          ) : undefined
        }
        onConfirm={async () => {
          await remove.mutateAsync({ params })
          toast({ kind: 'deleted', title: tt(m.deleted, { name: form.name }), detail: tt(m.deletedBody) })
          await invalidate()
          void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: {} })
        }}
      />
    </Page>
  )
}

/** What double opt-in does, with the provider that sends the confirmations. */
export function doubleOptInNote(on: boolean, provider: string): string {
  if (!on) return tt(m.doubleOptInOff)
  if (!provider) return tt(m.doubleOptInWaiting)
  return tt(m.doubleOptInVia, { provider })
}

/** "Type": Embedded · Popup · Hosted page, and what that means. */
function TypeSection({
  draft,
  form,
  readOnly,
  onKind,
  onContent,
}: {
  draft: FormInput
  form: SignupForm
  readOnly: boolean
  onKind: (k: Kind) => void
  onContent: (patch: Partial<FormInput['content']>) => void
}) {
  const t = useT()
  const at = hostPath(form.hostedUrl)
  const help: Record<Kind, MessageDescriptor> = { embedded: m.helpEmbedded, popup: m.helpPopup, hosted: m.helpHosted }
  const c = draft.content
  return (
    <div className="fb-sec">
      <span className="k">{t(m.type)}</span>
      <div className="fb-seg" role="group" aria-label={t(m.type)}>
        {(Object.keys(kindLabels) as Kind[]).map((k) => (
          <button key={k} type="button" aria-pressed={draft.kind === k} aria-disabled={readOnly || undefined} onClick={() => !readOnly && onKind(k)}>
            {t(kindLabels[k])}
          </button>
        ))}
      </div>
      <span className="k">{t(help[draft.kind], { at })}</span>
      {draft.kind === 'popup' ? (
        <div className="grid grid-cols-3 gap-2">
          <label className="lbl justify-between">
            {t(m.opensAfter)}
            <input className="fb-in" type="number" min={0} max={600} value={c.popupDelay} disabled={readOnly} onChange={(e) => onContent({ popupDelay: Number(e.target.value) })} />
          </label>
          <label className="lbl justify-between">
            {t(m.orScrolled)}
            <input className="fb-in" type="number" min={0} max={100} value={c.popupScroll} disabled={readOnly} onChange={(e) => onContent({ popupScroll: Number(e.target.value) })} />
          </label>
          <label className="lbl justify-between">
            {t(m.closedFor)}
            <input className="fb-in" type="number" min={0} max={365} value={c.popupRepeatDays} disabled={readOnly} onChange={(e) => onContent({ popupRepeatDays: Number(e.target.value) })} />
          </label>
        </div>
      ) : null}
    </div>
  )
}

/** "Add custom field": the contact fields the form doesn't ask for yet. */
function AddFieldMenu({ draft, overview, readOnly, onAdd }: { draft: FormInput; overview: Overview; readOnly: boolean; onAdd: (key: string) => void }) {
  const t = useT()
  const has = (k: string) => draft.fields.some((f) => f.key === k)
  const options = [
    ...(has('last_name') ? [] : [{ key: 'last_name', label: t(m.fieldLastName) }]),
    ...overview.customFields.filter((c) => !has(c.key)).map((c) => ({ key: c.key, label: c.label })),
  ]
  return (
    <Menu
      align="start"
      label={t(m.contactFields)}
      trigger={
        <button type="button" className="btn sm self-start" disabled={readOnly}>
          {t(m.addCustomField)}
        </button>
      }
    >
      {options.length === 0 ? (
        <MenuItem disabled>{t(m.everyField)}</MenuItem>
      ) : (
        options.map((o) => (
          <MenuItem key={o.key} onSelect={() => onAdd(o.key)}>
            {o.label}
          </MenuItem>
        ))
      )}
    </Menu>
  )
}

/** Turnstile keys, asked for when the check is turned on without keys. */
function TurnstileKeys({ readOnly }: { readOnly: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/turnstile')
  const [siteKey, setSiteKey] = useState('')
  const [secret, setSecret] = useState('')
  const [error, setError] = useState<string>()
  const submit = async () => {
    setError(undefined)
    try {
      const out = await save.mutateAsync({ params: { path: { workspace: slug } }, body: { siteKey: siteKey.trim(), secret: secret.trim() } })
      await invalidate()
      if (out.turnstile.status === 'error') setError(tt(m.checkKeys, { error: out.turnstile.error ?? '' }))
      else toast({ kind: 'success', title: tt(m.keysSaved), detail: tt(m.secretStored) })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <div className="flex flex-col gap-1.5 pl-7">
      <input
        className="fb-in mono"
        aria-label={t(m.siteKey)}
        placeholder={t(m.siteKeyPlaceholder, { example: '0x4AAAAAAA…' })}
        value={siteKey}
        disabled={readOnly}
        onChange={(e) => setSiteKey(e.target.value)}
      />
      <input
        className="fb-in mono"
        type="password"
        autoComplete="off"
        aria-label={t(m.secretKey)}
        placeholder={t(m.secretPlaceholder)}
        value={secret}
        disabled={readOnly}
        onChange={(e) => setSecret(e.target.value)}
      />
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : (
        <span className="k">{t(m.keysShared)}</span>
      )}
      <Button size="sm" className="self-start" onClick={() => void submit()} loading={save.isPending} disabled={readOnly || !siteKey.trim() || !secret.trim()}>
        {t(m.saveKeys)}
      </Button>
    </div>
  )
}

/**
 * "Copy by language" (FormBuilderLanguages' list) at the end of the default
 * builder, for a workspace with more than one language: each language
 * opens the builder in it.
 */
function FormLanguagesSection({ form }: { form: SignupForm }) {
  const langs = useWorkspaceLanguages()
  const many = (langs.data?.languages.length ?? 0) > 1
  return many ? <FormLanguagesRows form={form} /> : null
}

function FormLanguagesRows({ form }: { form: SignupForm }) {
  const copy = useFormCopy(form.id)
  if (!copy.data) return null
  return <CopyByLanguage data={copy.data} linkFor={(language) => ({ form: form.id, language })} variant="builder" />
}

/** "Embed code": the script snippet and the plain HTML form. */
function EmbedSection({ form }: { form: SignupForm }) {
  const t = useT()
  const snippet = embedSnippet(form.publicId, form.urls.script)
  const copy = async (text: string, what: 'script' | 'html') => {
    if (await copyText(text)) toast({ kind: 'success', title: tt(what === 'script' ? m.scriptCopied : m.htmlCopied), detail: tt(m.pasteIt) })
    else toast({ kind: 'error', title: tt(what === 'script' ? m.scriptCopyFailed : m.htmlCopyFailed), detail: tt(m.clipboardBlocked) })
  }
  return (
    <div className="fb-sec">
      <span className="fb-title">{t(m.embedCode)}</span>
      <span className="k">{t(m.scriptSize, { kb: count(Math.max(form.urls.scriptKb, 1)) })}</span>
      <ScrollPre className="fb-pre" label={t(m.embedCode)}>
        {snippet}
      </ScrollPre>
      <div className="fb-row flex gap-1.5">
        <Button size="sm" onClick={() => void copy(snippet, 'script')}>
          {t(m.copyScript)}
        </Button>
        <Button size="sm" onClick={() => void copy(form.plainHtml, 'html')}>
          {t(m.copyHtml)}
        </Button>
      </div>
      {/* Grow (design expansion, X15): referral codes on the form's page count. */}
      <FormRefNote />
    </div>
  )
}

/** "Last 30 days": views, submissions, confirmations and spam blocked. */
function NumbersSection({ form, systemMail }: { form: SignupForm; systemMail: boolean }) {
  const t = useT()
  const s = form.stats
  const [links, setLinks] = useState(false)
  return (
    <div className="fb-sec last">
      <span className="fb-title">{t(m.lastDays, { days: s.days })}</span>
      <div className="fb-nums">
        <div>
          <div className="k">{t(m.views)}</div>
          <div className="n text-[18px]">{count(s.views)}</div>
        </div>
        <div>
          <div className="k">{t(m.submissions)}</div>
          <div className="n text-[18px]">
            {count(s.submissions)} · {trimPct(Number(s.submissionRate.toFixed(1)))}
          </div>
        </div>
        <div>
          <div className="k">{t(m.confirmedOptIn)}</div>
          <div className="n text-[18px]">
            {count(s.confirmed)} · {trimPct(Number(s.confirmedRate.toFixed(1)))}
          </div>
        </div>
        <div>
          <div className="k">{t(m.blockedSpam)}</div>
          <div className="n text-[18px]">{count(s.blocked)}</div>
        </div>
        <FormAskedToFix formKey={form.id} />
      </div>
      {form.doubleOptIn && s.pending > 0 ? (
        <span className="k">{t(systemMail ? m.pendingReminders : m.pendingWait, { n: s.pending, count: count(s.pending) })}</span>
      ) : null}
      {form.doubleOptIn && form.waiting > 0 ? (
        <>
          <span className="k">{t(m.stillWaiting, { n: form.waiting, count: count(form.waiting) })}</span>
          <Button size="sm" className="self-start" onClick={() => setLinks(true)}>
            {t(m.copyLinks)}
          </Button>
        </>
      ) : null}
      {links ? <WaitingDialog form={form} onClose={() => setLinks(false)} /> : null}
    </div>
  )
}

/**
 * The copy-link fallback: signups whose confirmation email couldn't go out,
 * each with its link to send by hand. Every copied link is logged.
 */
function WaitingDialog({ form, onClose }: { form: SignupForm; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('subscribers:edit')
  const params = { path: { workspace: slug, form: form.id } }
  const waiting = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}/waiting', { params })
  const link = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/waiting/{submission}/link')
  const [copied, setCopied] = useState<string>()
  const copy = async (id: string) => {
    try {
      const out = await link.mutateAsync({ params: { path: { workspace: slug, form: form.id, submission: id } } })
      if (await copyText(out.url)) {
        setCopied(id)
        toast({ kind: 'success', title: tt(m.confirmCopied), detail: tt(m.sendIt, { email: out.email }) })
      } else {
        toast({ kind: 'error', title: tt(m.copyFailed), detail: tt(m.clipboardRetry) })
      }
    } catch (e) {
      toast({ kind: 'error', title: tt(m.linkFailed), detail: errorMessage(e), retry: () => void copy(id) })
    }
  }
  const d = waiting.data
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.linksTitle)}
      description={d && !d.systemMail ? t(m.linksNoSystem) : t(m.linksRetrying)}
      footer={<Button onClick={onClose}>{t(common.close)}</Button>}
    >
      {waiting.isError ? (
        <ErrorPanel title={t(m.signupsFailed)} actions={<Button onClick={() => waiting.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(waiting.error)}
        </ErrorPanel>
      ) : !d ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <span className="vh" role="status">
            {t(m.loadingSignups)}
          </span>
          <Skeleton h={44} />
          <Skeleton h={44} />
          <Skeleton h={44} />
        </div>
      ) : d.items.length === 0 ? (
        <p className="text-ink-2">{t(m.allSent)}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {d.items.map((it) => (
            <li key={it.id} className="fb-wait">
              <div className="flex min-w-0 flex-col">
                <span className="font-semibold [overflow-wrap:anywhere]">{it.email}</span>
                <span className="k">
                  {[andList(it.lists), t(m.signedUp, { ago: ago(it.createdAt) })].filter(Boolean).join(' · ')}
                </span>
              </div>
              <Button size="sm" onClick={() => void copy(it.id)} disabledReason={reason}>
                {copied === it.id ? t(common.copied) : t(m.copyLink)}
                <span className="vh"> {t(m.copyFor, { email: it.email })}</span>
              </Button>
            </li>
          ))}
          {d.total > d.items.length ? (
            <li className="k pt-2">{t(m.newestOf, { shown: count(d.items.length), total: count(d.total) })}</li>
          ) : null}
        </ul>
      )}
    </Dialog>
  )
}

/** "Add" next to the allowed domains: add a site or take one off. */
function DomainsEditor({ origins, readOnly, onChange }: { origins: string[]; readOnly: boolean; onChange: (next: string[]) => void }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [value, setValue] = useState('')
  const [error, setError] = useState<string>()
  const toggle = (o: boolean) => {
    setOpen(o)
    if (!o) {
      setValue('')
      setError(undefined)
    }
  }
  const add = () => {
    const host = value
      .trim()
      .toLowerCase()
      .replace(/^[a-z]+:\/\//, '')
      .replace(/[/?#].*$/, '')
    if (!/^[a-z0-9.-]+\.[a-z]{2,}(:\d+)?$|^localhost(:\d+)?$/.test(host)) {
      setError(t(m.notDomain, { example: 'docs.acme.io' }))
      return
    }
    if (!origins.includes(host)) onChange([...origins, host])
    setValue('')
    setError(undefined)
  }
  return (
    <Popover
      open={open}
      onOpenChange={toggle}
      align="start"
      label={t(m.allowedDomains)}
      className="w-[280px] gap-2 p-3"
      trigger={
        <button type="button" className="btn sm" disabled={readOnly}>
          {t(m.add)}
        </button>
      }
    >
      <form
        className="flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          add()
        }}
      >
        <Field label={t(m.domain)} hint={t(m.domainHint)} error={error}>
          <Input value={value} onChange={(e) => setValue(e.target.value)} placeholder={'docs.acme.io'} className="w-full" autoFocus />
        </Field>
        <Button type="submit" size="sm" variant="primary" className="self-start">
          {t(m.addDomain)}
        </Button>
      </form>
      {origins.length ? (
        <ul className="m-0 flex list-none flex-col gap-1 border-t border-line-2 p-0 pt-2">
          {origins.map((o) => (
            <li key={o} className="flex items-center justify-between gap-2 text-[13px]">
              <span className="mono">{o}</span>
              <button type="button" className="btn xs" onClick={() => onChange(origins.filter((x) => x !== o))} aria-label={t(m.removeOrigin, { origin: o })}>
                {t(m.remove)}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      <span className="k">{t(m.applyOnPublish)}</span>
    </Popover>
  )
}

/** "New form": a name and a type; it starts as a draft. */
export function CreateFormDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  return open ? <CreateFormDialogOpen onOpenChange={onOpenChange} /> : null
}

function CreateFormDialogOpen({ onOpenChange }: { onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateForms()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms')
  // The Push prompt type (X22) opens the prompt designer once push is set up.
  const t = useT()
  const pushOn = useChannelSetUp('push')
  const createPrompt = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/prompts')
  const [name, setName] = useState('')
  const [kind, setKind] = useState<Kind | 'push'>('embedded')
  const [error, setError] = useState<string>()
  const submit = async () => {
    if (!name.trim()) {
      setError(t(m.nameIt, { example: t(m.exampleName) }))
      return
    }
    if (kind === 'push') {
      try {
        const pr = await createPrompt.mutateAsync({ params: { path: { workspace: slug } }, body: { name: name.trim() } })
        await invalidate()
        onOpenChange(false)
        void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { prompt: pr.formId } })
      } catch (e) {
        setError(errorMessage(e))
      }
      return
    }
    try {
      const f = await create.mutateAsync({ params: { path: { workspace: slug } }, body: { name: name.trim(), kind } })
      await invalidate()
      onOpenChange(false)
      toast({ kind: 'success', title: tt(m.created, { name: f.name }), detail: tt(m.createdBody) })
      void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { form: f.id } })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.newTitle)}
      description={kind === 'push' ? t(promptMsg.typeHint) : t(m.newBody)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={() => void submit()} loading={create.isPending || createPrompt.isPending}>
            {t(m.createForm)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <Field label={t(m.name)} error={error}>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t(m.exampleName)} maxLength={80} className="w-full" autoFocus />
        </Field>
        <div className="lbl">
          <span>{t(m.type)}</span>
          <div className="fb-seg" role="group" aria-label={t(m.type)}>
            {(Object.keys(kindLabels) as Kind[]).map((k) => (
              <button key={k} type="button" aria-pressed={kind === k} onClick={() => setKind(k)}>
                {t(kindLabels[k])}
              </button>
            ))}
            {pushOn ? (
              <button type="button" aria-pressed={kind === 'push'} onClick={() => setKind('push')}>
                {t(promptMsg.typePush)}
              </button>
            ) : null}
          </div>
        </div>
      </form>
    </Dialog>
  )
}

function RenameDialog({ open, onOpenChange, form }: { open: boolean; onOpenChange: (o: boolean) => void; form: SignupForm }) {
  return open ? <RenameDialogOpen onOpenChange={onOpenChange} form={form} /> : null
}

function RenameDialogOpen({ onOpenChange, form }: { onOpenChange: (o: boolean) => void; form: SignupForm }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const rename = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/rename')
  const [name, setName] = useState(form.name)
  const [error, setError] = useState<string>()
  const submit = async () => {
    try {
      await rename.mutateAsync({ params: { path: { workspace: slug, form: form.id } }, body: { name: name.trim() } })
      await invalidate()
      onOpenChange(false)
      toast({ kind: 'success', title: tt(m.renamed, { name: name.trim() }), detail: tt(m.renamedBody) })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.renameTitle)}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={() => void submit()} loading={rename.isPending}>
            {t(m.rename)}
          </Button>
        </>
      }
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <Field label={t(m.name)} error={error} hint={form.kind !== 'embedded' || form.hostedPage ? undefined : t(m.renameHint)}>
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} className="w-full" autoFocus />
        </Field>
      </form>
    </Dialog>
  )
}
