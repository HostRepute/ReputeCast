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
  builtinFieldLabels,
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

type Kind = Schemas['SignupFormSummary']['kind']

const kindLabels: Record<Kind, string> = { embedded: 'Embedded', popup: 'Popup', hosted: 'Hosted page' }

/** Built-in fields in the order a form shows them; custom fields follow. */
const fieldOrder = ['email', 'first_name', 'last_name', 'company']

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
  const overview = useFormsOverview()
  throwIfForbidden(overview.error)
  const [creating, setCreating] = useState(false)
  const createReason = usePermissionReason('forms:edit')
  if (overview.isError) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <ErrorPanel title="Couldn't load the signup forms" actions={<Button onClick={() => overview.refetch()}>Retry</Button>}>
          {errorMessage(overview.error)} Nothing was changed; the forms on your site keep working.
        </ErrorPanel>
      </Page>
    )
  }
  if (overview.isPending) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <FormsLoading />
      </Page>
    )
  }
  const o = overview.data
  if (o.forms.length === 0) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
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
  const { slug } = useWorkspace()
  const detail = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}', { params: { path: { workspace: slug, form: formId } } })
  throwIfForbidden(detail.error)
  if (detail.isError) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <ErrorPanel title="Couldn't load this form" actions={<Button onClick={() => detail.refetch()}>Retry</Button>}>
          {errorMessage(detail.error)} Nothing was changed.
        </ErrorPanel>
      </Page>
    )
  }
  if (detail.isPending) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <FormsLoading label="Building the preview" />
      </Page>
    )
  }
  if (language) return <FormLanguagesView form={detail.data} overview={overview} language={language} />
  return <Editor key={`${detail.data.id}:${detail.data.updatedAt}`} form={detail.data} overview={overview} />
}

function Editor({ form, overview }: { form: SignupForm; overview: Overview }) {
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
      toast({ kind: 'info', title: 'Nothing new to publish', detail: `"${form.name}" is live with these settings.` })
      return true
    }
    setErrors({})
    try {
      await save.mutateAsync({ params, body: { ...draft, publish: true } })
      toast({ kind: 'success', title: `Published · ${hostPath(form.hostedUrl)}`, detail: 'Embedded copies pick the changes up on their next page load.' })
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
      toast({ kind: 'error', title: "Couldn't publish the form", detail: errorMessage(e), retry: () => void publish() })
      throw e
    }
  }

  const pickForm = (id: string) => {
    if (id === '__new') {
      setCreating(true)
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
        title: next === 'paused' ? `"${form.name}" is paused` : `"${form.name}" takes signups again`,
        detail: next === 'paused' ? 'Visitors see "This form isn\'t taking signups right now." Nothing else changes.' : undefined,
      })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't change the form", detail: errorMessage(e), retry: () => void togglePause() })
    }
  }

  const where = useMemo(() => {
    const site = draft.allowedOrigins[0] ?? 'your site'
    if (draft.kind === 'hosted') return `hosted at ${hostPath(form.hostedUrl)}`
    return `${draft.kind === 'popup' ? 'popup' : 'embedded'} on ${site}`
  }, [draft.kind, draft.allowedOrigins, form.hostedUrl])

  const blocked = form.blockedOrigins
  const turnstileBroken = draft.spam.turnstile && overview.turnstile.status === 'error'

  return (
    <Page
      title="Forms & public pages"
      filters={<FormsTabs />}
      actions={
        <>
          <FormStatusBadge form={form} />
          <Menu
            label="Form actions"
            trigger={
              <button type="button" className="btn ic" aria-label={`More actions for "${form.name}"`}>
                <Icon name="more" size={16} />
              </button>
            }
          >
            <MenuItem onSelect={() => setCreating(true)} disabled={readOnly}>
              New form
            </MenuItem>
            <MenuItem onSelect={() => setRenaming(true)} disabled={readOnly}>
              Rename
            </MenuItem>
            {form.status !== 'draft' ? (
              <MenuItem onSelect={() => void togglePause()} disabled={readOnly}>
                {form.status === 'paused' ? 'Resume signups' : 'Pause signups'}
              </MenuItem>
            ) : null}
            <MenuSeparator />
            <MenuItem onSelect={() => setDeleting(true)} disabled={readOnly} className="text-bad-strong">
              Delete form
            </MenuItem>
          </Menu>
          <Button onClick={() => setDraft(initial)} disabledReason={reason}>
            Discard
          </Button>
          <Button variant="primary" onClick={() => void publish().catch(() => undefined)} loading={save.isPending} busyLabel="Publishing…" disabledReason={reason}>
            Save &amp; publish
          </Button>
        </>
      }
      contentClassName="fb-main"
    >
      <div className="fb-cols">
        <aside className="fb-left" aria-label="Form settings">
          <div className="fb-sec">
            <div className="fb-row flex items-center justify-between gap-2.5">
              <span className="font-semibold">Form · "{form.name}"</span>
              <select aria-label="Signup form" className="fb-in h-[30px]! w-[140px]!" value={form.id} onChange={(e) => pickForm(e.target.value)}>
                {overview.forms.map((f) => (
                  <option key={f.id} value={f.id}>
                    {f.name}
                  </option>
                ))}
                {readOnly ? null : <option value="__new">New form…</option>}
              </select>
            </div>
          </div>
          <TypeSection draft={draft} form={form} readOnly={readOnly} onKind={(k) => set('kind', k)} onContent={setContent} />
          <div className="fb-sec">
            <span className="k">Fields</span>
            <label className="fb-check">
              <input type="checkbox" checked disabled />
              Email · required
            </label>
            {(['first_name', 'company'] as const).map((key) => (
              <label key={key} className="fb-check">
                <input type="checkbox" checked={hasField(key)} disabled={readOnly} onChange={(e) => toggleField(key, e.target.checked)} />
                {builtinFieldLabels[key]}
                {hasField(key) ? ' · optional' : ''}
              </label>
            ))}
            {draft.fields
              .filter((f) => f.key !== 'email' && f.key !== 'first_name' && f.key !== 'company')
              .map((f) => (
                <label key={f.key} className="fb-check">
                  <input type="checkbox" checked disabled={readOnly} onChange={() => toggleField(f.key, false)} />
                  {builtinFieldLabels[f.key] ?? overview.customFields.find((c) => c.key === f.key)?.label ?? f.key}
                  {f.required ? ' · required' : ' · optional'}
                </label>
              ))}
            <label className="fb-check">
              <input type="checkbox" checked={draft.listChoice} disabled={readOnly} onChange={(e) => set('listChoice', e.target.checked)} />
              Let people pick lists
            </label>
            <AddFieldMenu draft={draft} overview={overview} readOnly={readOnly} onAdd={(key) => toggleField(key, true)} />
          </div>
          <div className="fb-sec">
            <span className="k">Lists</span>
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
                  {l.name}
                  {offered && isDefault ? ' · checked by default' : ''}
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
            <span className="k">Consent</span>
            <label className="fb-check start">
              <input type="checkbox" checked={draft.doubleOptIn} disabled={readOnly} onChange={(e) => set('doubleOptIn', e.target.checked)} />
              <span>
                Double opt-in <span className="k">· {doubleOptInNote(draft.doubleOptIn, overview.systemMail ? overview.systemProvider : '')}</span>
              </span>
            </label>
            <label className="lbl">
              Consent text
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
              IP, timestamp and this exact text are stored as consent evidence.
              {draft.consentText.trim() !== form.consentText && form.consentText
                ? ` Saving starts version ${form.consentVersion + 1}; earlier signups keep the text they agreed to.`
                : ''}
            </span>
          </div>
          <div className="fb-sec">
            <span className="k">Spam protection</span>
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.honeypot} disabled={readOnly} onChange={(e) => setSpam({ honeypot: e.target.checked })} />
              Honeypot field and timing check · invisible
            </label>
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.turnstile} disabled={readOnly} onChange={(e) => setSpam({ turnstile: e.target.checked })} />
              Cloudflare Turnstile · shows only when a submission looks automated
            </label>
            {draft.spam.turnstile && !overview.turnstile.hasSecret ? <TurnstileKeys readOnly={readOnly} /> : null}
            <label className="fb-check">
              <input type="checkbox" checked={draft.spam.blockDisposable} disabled={readOnly} onChange={(e) => setSpam({ blockDisposable: e.target.checked })} />
              Reject disposable domains and role addresses
            </label>
            <FormAddressCheck formKey={form.id} formName={form.name} />
          </div>
          <div className="fb-sec last">
            <span className="k">After submit</span>
            <select aria-label="After submit" className="fb-in" value={draft.afterSubmit} disabled={readOnly} onChange={(e) => set('afterSubmit', e.target.value as FormInput['afterSubmit'])}>
              <option value="message">{draft.doubleOptIn ? 'Show "check your inbox" message' : 'Show "you\'re subscribed" message'}</option>
              <option value="redirect">Redirect to a URL</option>
            </select>
            {draft.afterSubmit === 'redirect' ? (
              <>
                <input
                  className="fb-in"
                  aria-label="Redirect URL"
                  placeholder="https://acme.io/thanks"
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

        <section className="fb-center" aria-label="Live preview">
          <div className="fb-bar">
            <span className="k">Live preview · {where}</span>
            <div className="fb-seg ml-auto" role="group" aria-label="Preview size">
              <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                Desktop
              </button>
              <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                Mobile
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

        <aside className="fb-right" aria-label="Embed code and numbers">
          <EmbedSection form={form} />
          <div className="fb-sec">
            <span className="fb-title">Hosted page</span>
            <input className="fb-in mono" aria-label="Hosted page URL" value={form.hostedUrl} readOnly />
            <div className="flex gap-1.5">
              <Button
                size="sm"
                onClick={async () =>
                  (await copyText(form.hostedUrl))
                    ? toast({ kind: 'success', title: 'Link copied', detail: hostPath(form.hostedUrl) })
                    : toast({ kind: 'error', title: "Couldn't copy the link", detail: 'Select it in the field and copy it yourself.' })
                }
              >
                Copy link
              </Button>
              <a className="btn sm" href={form.hostedUrl} target="_blank" rel="noreferrer">
                Open
              </a>
            </div>
          </div>
          <div className="fb-sec">
            <span className="fb-title">Allowed domains</span>
            <span className="k">Submissions from other origins are rejected.</span>
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
        title="Leave without publishing?"
        description={`You changed "${form.name}" since it was saved. Publish the changes or leave them.`}
        saveLabel="Publish and leave"
        onSave={publish}
      />
      <CreateFormDialog open={creating} onOpenChange={setCreating} />
      <RenameDialog open={renaming} onOpenChange={setRenaming} form={form} />
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        badge="Delete form"
        title={`Delete "${form.name}"?`}
        description="Copies of this form on your sites stop accepting signups, and its hosted page goes away. Contacts it brought in, their consent records and automations they're in stay."
        confirmLabel="Delete form"
        busyLabel="Deleting…"
        secondary={
          form.status === 'live' ? (
            <Button
              onClick={() => {
                setDeleting(false)
                void togglePause()
              }}
            >
              Pause signups instead
            </Button>
          ) : undefined
        }
        onConfirm={async () => {
          await remove.mutateAsync({ params })
          toast({ kind: 'deleted', title: `"${form.name}" deleted`, detail: 'Its contacts and their consent records stay.' })
          await invalidate()
          void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: {} })
        }}
      />
    </Page>
  )
}

/** What double opt-in does, with the provider that sends the confirmations. */
export function doubleOptInNote(on: boolean, provider: string): string {
  if (!on) return 'off: contacts are subscribed as soon as they submit'
  if (!provider) return 'confirmations wait until System mail can be sent; contacts stay "pending" until they click'
  return `confirmation sent on the System stream via ${provider}; contacts stay "pending" until they click`
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
  const at = hostPath(form.hostedUrl)
  const help: Record<Kind, string> = {
    embedded: `Embedded renders inside your site; the hosted page lives at ${at} with your branding.`,
    popup: `Popup opens over your site and stays closed once someone dismisses it; the hosted page lives at ${at} with your branding.`,
    hosted: `Hosted page lives at ${at} with your branding; link to it from anywhere, no code needed.`,
  }
  const c = draft.content
  return (
    <div className="fb-sec">
      <span className="k">Type</span>
      <div className="fb-seg" role="group" aria-label="Type">
        {(Object.keys(kindLabels) as Kind[]).map((k) => (
          <button key={k} type="button" aria-pressed={draft.kind === k} aria-disabled={readOnly || undefined} onClick={() => !readOnly && onKind(k)}>
            {kindLabels[k]}
          </button>
        ))}
      </div>
      <span className="k">{help[draft.kind]}</span>
      {draft.kind === 'popup' ? (
        <div className="grid grid-cols-3 gap-2">
          <label className="lbl justify-between">
            Opens after (s)
            <input className="fb-in" type="number" min={0} max={600} value={c.popupDelay} disabled={readOnly} onChange={(e) => onContent({ popupDelay: Number(e.target.value) })} />
          </label>
          <label className="lbl justify-between">
            Or scrolled (%)
            <input className="fb-in" type="number" min={0} max={100} value={c.popupScroll} disabled={readOnly} onChange={(e) => onContent({ popupScroll: Number(e.target.value) })} />
          </label>
          <label className="lbl justify-between">
            Closed for (days)
            <input className="fb-in" type="number" min={0} max={365} value={c.popupRepeatDays} disabled={readOnly} onChange={(e) => onContent({ popupRepeatDays: Number(e.target.value) })} />
          </label>
        </div>
      ) : null}
    </div>
  )
}

/** "Add custom field": the contact fields the form doesn't ask for yet. */
function AddFieldMenu({ draft, overview, readOnly, onAdd }: { draft: FormInput; overview: Overview; readOnly: boolean; onAdd: (key: string) => void }) {
  const has = (k: string) => draft.fields.some((f) => f.key === k)
  const options = [
    ...(has('last_name') ? [] : [{ key: 'last_name', label: 'Last name' }]),
    ...overview.customFields.filter((c) => !has(c.key)).map((c) => ({ key: c.key, label: c.label })),
  ]
  return (
    <Menu
      align="start"
      label="Contact fields"
      trigger={
        <button type="button" className="btn sm self-start" disabled={readOnly}>
          Add custom field
        </button>
      }
    >
      {options.length === 0 ? (
        <MenuItem disabled>Every contact field is on the form</MenuItem>
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
      if (out.turnstile.status === 'error') setError(`${out.turnstile.error}. Check the keys in the Cloudflare dashboard.`)
      else toast({ kind: 'success', title: 'Turnstile keys saved', detail: 'The secret is stored encrypted and never shown again.' })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <div className="flex flex-col gap-1.5 pl-7">
      <input className="fb-in mono" aria-label="Turnstile site key" placeholder="Site key · 0x4AAAAAAA…" value={siteKey} disabled={readOnly} onChange={(e) => setSiteKey(e.target.value)} />
      <input
        className="fb-in mono"
        type="password"
        autoComplete="off"
        aria-label="Turnstile secret key"
        placeholder="Secret key"
        value={secret}
        disabled={readOnly}
        onChange={(e) => setSecret(e.target.value)}
      />
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : (
        <span className="k">Shared by every form of this workspace. The secret is stored encrypted and never shown again.</span>
      )}
      <Button size="sm" className="self-start" onClick={() => void submit()} loading={save.isPending} disabled={readOnly || !siteKey.trim() || !secret.trim()}>
        Save keys
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
  const snippet = embedSnippet(form.publicId, form.urls.script)
  const copy = async (text: string, what: string) => {
    if (await copyText(text)) toast({ kind: 'success', title: `${what} copied`, detail: 'Paste it into your site where the form should appear.' })
    else toast({ kind: 'error', title: `Couldn't copy the ${what.toLowerCase()}`, detail: 'Your browser blocked the clipboard. Select the code and copy it yourself.' })
  }
  return (
    <div className="fb-sec">
      <span className="fb-title">Embed code</span>
      <span className="k">Loads a {Math.max(form.urls.scriptKb, 1)} KB script; the form renders with your site's fonts and no iframe.</span>
      <ScrollPre className="fb-pre" label="Embed code">
        {snippet}
      </ScrollPre>
      <div className="fb-row flex gap-1.5">
        <Button size="sm" onClick={() => void copy(snippet, 'Script embed')}>
          Copy script embed
        </Button>
        <Button size="sm" onClick={() => void copy(form.plainHtml, 'Plain HTML form')}>
          Copy plain HTML form
        </Button>
      </div>
    </div>
  )
}

/** "Last 30 days": views, submissions, confirmations and spam blocked. */
function NumbersSection({ form, systemMail }: { form: SignupForm; systemMail: boolean }) {
  const s = form.stats
  const [links, setLinks] = useState(false)
  return (
    <div className="fb-sec last">
      <span className="fb-title">Last {s.days} days</span>
      <div className="fb-nums">
        <div>
          <div className="k">Views</div>
          <div className="n text-[18px]">{count(s.views)}</div>
        </div>
        <div>
          <div className="k">Submissions</div>
          <div className="n text-[18px]">
            {count(s.submissions)} · {trimPct(Number(s.submissionRate.toFixed(1)))}
          </div>
        </div>
        <div>
          <div className="k">Confirmed opt-in</div>
          <div className="n text-[18px]">
            {count(s.confirmed)} · {trimPct(Number(s.confirmedRate.toFixed(1)))}
          </div>
        </div>
        <div>
          <div className="k">Blocked as spam</div>
          <div className="n text-[18px]">{count(s.blocked)}</div>
        </div>
        <FormAskedToFix formKey={form.id} />
      </div>
      {form.doubleOptIn && s.pending > 0 ? (
        <span className="k">
          {count(s.pending)} {s.pending === 1 ? 'confirmation is' : 'confirmations are'} pending;{' '}
          {systemMail ? 'reminders go out after 24 h.' : 'reminders wait until System mail can be sent.'}
        </span>
      ) : null}
      {form.doubleOptIn && form.waiting > 0 ? (
        <>
          <span className="k">
            {count(form.waiting)} {form.waiting === 1 ? 'signup is' : 'signups are'} still waiting for the confirmation email. Send them their link yourself.
          </span>
          <Button size="sm" className="self-start" onClick={() => setLinks(true)}>
            Copy confirmation links
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
        toast({ kind: 'success', title: 'Confirmation link copied', detail: `Send it to ${out.email}. It subscribes them when they open it.` })
      } else {
        toast({ kind: 'error', title: "Couldn't copy the link", detail: 'Your browser blocked the clipboard. Try again, or use another browser.' })
      }
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't get the link", detail: errorMessage(e), retry: () => void copy(id) })
    }
  }
  const d = waiting.data
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Confirmation links"
      description={
        d && !d.systemMail
          ? 'Nothing can send System mail yet, so these people never got their confirmation email. Send each one their link yourself; it subscribes them when they open it. Every link you copy is logged.'
          : "These confirmation emails couldn't be sent; ReputeCast tries again for 3 days. You can send a link yourself; it subscribes them when they open it. Every link you copy is logged."
      }
      footer={<Button onClick={onClose}>Close</Button>}
    >
      {waiting.isError ? (
        <ErrorPanel title="Couldn't load the signups" actions={<Button onClick={() => waiting.refetch()}>Retry</Button>}>
          {errorMessage(waiting.error)}
        </ErrorPanel>
      ) : !d ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <span className="vh" role="status">
            Loading the signups
          </span>
          <Skeleton h={44} />
          <Skeleton h={44} />
          <Skeleton h={44} />
        </div>
      ) : d.items.length === 0 ? (
        <p className="text-ink-2">Every confirmation email has gone out. Nothing is waiting.</p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {d.items.map((it) => (
            <li key={it.id} className="fb-wait">
              <div className="flex min-w-0 flex-col">
                <span className="font-semibold [overflow-wrap:anywhere]">{it.email}</span>
                <span className="k">
                  {[andList(it.lists), `signed up ${ago(it.createdAt)}`].filter(Boolean).join(' · ')}
                </span>
              </div>
              <Button size="sm" onClick={() => void copy(it.id)} disabledReason={reason}>
                {copied === it.id ? 'Copied' : 'Copy link'}
                <span className="vh"> for {it.email}</span>
              </Button>
            </li>
          ))}
          {d.total > d.items.length ? (
            <li className="k pt-2">
              Showing the {count(d.items.length)} newest of {count(d.total)}.
            </li>
          ) : null}
        </ul>
      )}
    </Dialog>
  )
}

/** "Add" next to the allowed domains: add a site or take one off. */
function DomainsEditor({ origins, readOnly, onChange }: { origins: string[]; readOnly: boolean; onChange: (next: string[]) => void }) {
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
      setError("That isn't a domain. Use a site's address, like docs.acme.io.")
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
      label="Allowed domains"
      className="w-[280px] gap-2 p-3"
      trigger={
        <button type="button" className="btn sm" disabled={readOnly}>
          Add
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
        <Field label="Domain" hint="The site the form is embedded on. Subdomains need their own entry." error={error}>
          <Input value={value} onChange={(e) => setValue(e.target.value)} placeholder="docs.acme.io" className="w-full" autoFocus />
        </Field>
        <Button type="submit" size="sm" variant="primary" className="self-start">
          Add domain
        </Button>
      </form>
      {origins.length ? (
        <ul className="m-0 flex list-none flex-col gap-1 border-t border-line-2 p-0 pt-2">
          {origins.map((o) => (
            <li key={o} className="flex items-center justify-between gap-2 text-[13px]">
              <span className="mono">{o}</span>
              <button type="button" className="btn xs" onClick={() => onChange(origins.filter((x) => x !== o))} aria-label={`Remove ${o}`}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      <span className="k">Changes apply when you save and publish.</span>
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
  const [name, setName] = useState('')
  const [kind, setKind] = useState<Kind>('embedded')
  const [error, setError] = useState<string>()
  const submit = async () => {
    if (!name.trim()) {
      setError('Name the form, for example "Footer form".')
      return
    }
    try {
      const f = await create.mutateAsync({ params: { path: { workspace: slug } }, body: { name: name.trim(), kind } })
      await invalidate()
      onOpenChange(false)
      toast({ kind: 'success', title: `"${f.name}" created`, detail: 'It stays a draft until you save and publish it.' })
      void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { form: f.id } })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title="New signup form"
      description="It starts with email and first name, your first public list, double opt-in and every spam check."
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="primary" onClick={() => void submit()} loading={create.isPending}>
            Create form
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
        <Field label="Name" error={error}>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Footer form" maxLength={80} className="w-full" autoFocus />
        </Field>
        <div className="lbl">
          <span>Type</span>
          <div className="fb-seg" role="group" aria-label="Type">
            {(Object.keys(kindLabels) as Kind[]).map((k) => (
              <button key={k} type="button" aria-pressed={kind === k} onClick={() => setKind(k)}>
                {kindLabels[k]}
              </button>
            ))}
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
      toast({ kind: 'success', title: `Renamed to "${name.trim()}"`, detail: 'Embed code keeps working; the hosted page follows the new name.' })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title="Rename form"
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="primary" onClick={() => void submit()} loading={rename.isPending}>
            Rename
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
        <Field label="Name" error={error} hint={form.kind !== 'embedded' || form.hostedPage ? undefined : 'Its hosted page moves to the new name; the embed code stays the same.'}>
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} className="w-full" autoFocus />
        </Field>
      </form>
    </Dialog>
  )
}
