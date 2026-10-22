import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input, OptionCard, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './contacts.messages'

type ListSummary = Schemas['AudienceListSummary']
type Filter = Schemas['ContactFilter']

/** Invalidates every audience query (lists, counts, views, health). */
export function useInvalidateAudience() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const path = String((q.queryKey as unknown[])[1] ?? '')
        return /\/(audience|contacts|segments|suppressions|lists)/.test(path)
      },
    })
}

interface NewContactForm {
  email: string
  firstName: string
  lastName: string
  lists: string[]
  consent: 'single' | 'double'
  tags: string
}

/**
 * Add contact: one address, its lists and how consent is collected. Each
 * dialog here is mounted when it opens, so it always starts empty.
 */
export function AddContactDialog({ onClose, lists }: { onClose: () => void; lists: ListSummary[] }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts')
  const form = useForm<NewContactForm>({
    defaultValues: { email: '', firstName: '', lastName: '', lists: lists[0] ? [lists[0].id] : [], consent: 'double', tags: '' },
  })
  const [err, setErr] = useState<string | null>(null)
  const consent = useWatch({ control: form.control, name: 'consent' })
  const submit = form.handleSubmit(async (v) => {
    setErr(null)
    try {
      const r = await create.mutateAsync({
        params: { path: { workspace: slug } },
        body: {
          email: v.email.trim(),
          firstName: v.firstName.trim() || undefined,
          lastName: v.lastName.trim() || undefined,
          lists: v.lists,
          consent: v.consent,
          tags: v.tags
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean),
        },
      })
      await invalidate()
      onClose()
      toast({
        kind: 'success',
        title: t(m.added, { email: v.email.trim() }),
        detail: r.confirmationSent ? t(m.confirmationSent) : r.confirmUrl ? t(m.noSystemProvider) : undefined,
      })
    } catch (e) {
      const msg = errorMessage(e)
      if (/email|address/i.test(msg)) form.setError('email', { message: msg })
      else setErr(msg)
    }
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={tr(m.addTitle)}
      description={tr(m.addBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={form.formState.isSubmitting} onClick={() => void submit()}>
            {tr(m.addContact)}
          </Button>
        </>
      }
    >
      <form className="flex flex-col gap-3.5" onSubmit={(e) => void submit(e)} noValidate>
        {err ? <Note tone="bad">{err}</Note> : null}
        <Field label={tr(m.email)} error={form.formState.errors.email?.message}>
          <Input
            type="email"
            autoComplete="off"
            className="mono"
            {...form.register('email', { validate: (v) => (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim()) ? true : t(m.emailInvalid)) })}
          />
        </Field>
        <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
          <Field label={tr(m.firstName)} optional>
            <Input autoComplete="off" {...form.register('firstName')} />
          </Field>
          <Field label={tr(m.lastName)} optional>
            <Input autoComplete="off" {...form.register('lastName')} />
          </Field>
        </div>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">{tr(m.lists)}</legend>
          {lists.map((l) => (
            <Checkbox key={l.id} value={l.id} label={l.name} hint={l.description || undefined} {...form.register('lists')} />
          ))}
        </fieldset>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">{tr(m.consent)}</legend>
          <OptionCard
            name="consent"
            value="double"
            checked={consent === 'double'}
            onChange={() => form.setValue('consent', 'double')}
            title={tr(m.sendConfirmation)}
            description={tr(m.sendConfirmationBody)}
          />
          <OptionCard
            name="consent"
            value="single"
            checked={consent === 'single'}
            onChange={() => form.setValue('consent', 'single')}
            title={tr(m.alreadyAgreed)}
            description={tr(m.alreadyAgreedBody)}
          />
        </fieldset>
        <Field label={tr(m.tags)} optional hint={tr(m.tagsHint)}>
          <Input autoComplete="off" placeholder={'customer, beta'} {...form.register('tags')} />
        </Field>
      </form>
    </Dialog>
  )
}

/** Save current view: a name for the filters, shared or only for me. */
export function SaveViewDialog({ onClose, filter }: { onClose: () => void; filter: Filter }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const save = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/views')
  const [name, setName] = useState('')
  const [personal, setPersonal] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const empty = Object.keys(filter).length === 0
  const submit = async () => {
    if (!name.trim()) {
      setErr(t(m.nameView))
      return
    }
    try {
      await save.mutateAsync({ params: { path: { workspace: slug } }, body: { name: name.trim(), filters: filter, personal } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: t(m.viewSaved, { name: name.trim() }) })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(m.saveViewTitle)}
      description={empty ? tr(m.saveViewEmpty) : tr(m.saveViewBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={() => void submit()}>
            {tr(m.saveView)}
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
        <Field label={tr(m.name)} error={err ?? undefined}>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={'Engaged Outlook customers'} maxLength={80} />
        </Field>
        <Checkbox label={tr(m.onlyMe)} hint={tr(m.onlyMeHint)} checked={personal} onChange={(e) => setPersonal(e.target.checked)} />
      </form>
    </Dialog>
  )
}

export type BulkKind = 'add_to_list' | 'add_tag' | 'resend_optin' | 'suppress'

const bulkCopy: Record<BulkKind, { title: MessageDescriptor; action: MessageDescriptor }> = {
  add_to_list: { title: m.addToList, action: m.addToList },
  add_tag: { title: m.addTag, action: m.addTag },
  resend_optin: { title: m.resendOptin, action: m.resendOptin },
  suppress: { title: m.suppress, action: m.suppress },
}

/** One dialog for the bulk actions on selected contacts. */
export function BulkDialog({
  kind,
  onClose,
  ids,
  lists,
  onDone,
}: {
  kind: BulkKind
  onClose: () => void
  ids: string[]
  lists: ListSummary[]
  onDone: (r: Schemas['ContactBulkResult']) => void
}) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/bulk')
  const [listId, setListId] = useState(lists[0]?.id ?? '')
  const [tag, setTag] = useState('')
  const [reason, setReason] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const n = ids.length
  const who = { n, count: count(n) }
  const submit = async () => {
    setErr(null)
    if (kind === 'add_tag' && !tag.trim()) {
      setErr(t(m.typeTag))
      return
    }
    if (kind === 'suppress' && !reason.trim()) {
      setErr(t(m.sayWhy))
      return
    }
    try {
      const r = await run.mutateAsync({
        params: { path: { workspace: slug } },
        body: {
          ids,
          action: kind,
          listId: kind === 'add_to_list' ? listId : undefined,
          tag: kind === 'add_tag' ? tag.trim() : undefined,
          reason: kind === 'suppress' ? reason.trim() : undefined,
        },
      })
      await invalidate()
      onClose()
      onDone(r)
      if (r.failed === 0) toast({ kind: 'success', title: bulkTitle(r), detail: r.note || undefined })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(bulkCopy[kind].title)}
      description={kind === 'resend_optin' ? tr(m.resendBody, who) : kind === 'suppress' ? tr(m.suppressBody, who) : tr(m.appliesTo, who)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant={kind === 'suppress' ? 'danger-primary' : 'primary'} loading={run.isPending} onClick={() => void submit()}>
            {tr(bulkCopy[kind].action)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {kind === 'add_to_list' ? (
          <Field label={tr(m.list)} error={err ?? undefined}>
            <Select value={listId} onChange={(e) => setListId(e.target.value)}>
              {lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </Select>
          </Field>
        ) : null}
        {kind === 'add_tag' ? (
          <Field label={tr(m.tag)} error={err ?? undefined}>
            <Input value={tag} onChange={(e) => setTag(e.target.value)} placeholder={'re-permission'} maxLength={60} />
          </Field>
        ) : null}
        {kind === 'suppress' ? (
          <Field label={tr(m.reason)} error={err ?? undefined} hint={tr(m.reasonHint)}>
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder={tr(m.reasonPlaceholder)} maxLength={300} />
          </Field>
        ) : null}
        {kind === 'resend_optin' && err ? <Note tone="bad">{err}</Note> : null}
      </div>
    </Dialog>
  )
}

function bulkTitle(r: Schemas['ContactBulkResult']) {
  const v = { n: r.done, count: count(r.done), target: r.target ?? '' }
  switch (r.action) {
    case 'add_to_list':
      return t(m.doneAddToList, v)
    case 'add_tag':
      return t(m.doneAddTag, v)
    case 'resend_optin':
      return t(m.doneResend, v)
    case 'suppress':
      return t(m.doneSuppress, v)
  }
  return t(m.doneOther, v)
}

/** A bulk action that partly failed (AudienceStates › Partly done). */
export function BulkResultPanel({ result, onDismiss }: { result: Schemas['ContactBulkResult']; onDismiss: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const [show, setShow] = useState(false)
  const suppressed = result.failures.filter((f) => f.code === 'suppressed').length
  const failed = { n: result.failed, count: count(result.failed) }
  return (
    <div role="alert" className="card flex flex-col gap-2 border-[#FCD34D] px-4.5 py-4">
      <div className="flex items-center gap-2.5 max-md:flex-wrap">
        <Badge tone="warn">{tr(m.partlyDone)}</Badge>
        <span className="font-semibold">
          {tr(m.partlyLine, {
            action: result.action,
            done: count(result.done),
            matched: count(result.matched),
            target: result.target ?? '',
            failed: count(result.failed),
          })}
        </span>
      </div>
      <span className="k">
        {suppressed === result.failed ? tr(m.partlySuppressed, failed) : `${result.failures[0]?.reason ?? tr(m.someFailed)} ${tr(m.elseSucceeded)}`}
      </span>
      {show ? (
        <ul className="m-0 flex flex-col gap-1 pl-4 text-[13px]">
          {result.failures.map((f) => (
            <li key={f.email}>
              <span className="mono">{f.email}</span> <span className="k">· {f.reason}</span>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {suppressed === result.failed ? (
          <Link className="btn sm" to="/w/$workspace/audience/suppressions" params={{ workspace: slug }}>
            {tr(m.seeAddresses, failed)}
          </Link>
        ) : (
          <Button size="sm" onClick={() => setShow((v) => !v)}>
            {show ? tr(m.hideContacts, failed) : tr(m.seeContacts, failed)}
          </Button>
        )}
        <Button size="sm" onClick={onDismiss}>
          {tr(m.dismiss)}
        </Button>
      </div>
    </div>
  )
}

/** Delete: erase personal data, confirmed by typing the number of contacts. */
export function EraseDialog({ onClose, ids, onDone }: { onClose: () => void; ids: string[]; onDone: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const erase = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/erase')
  const [typed, setTyped] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const n = ids.length
  const ok = typed.replace(/[,\s]/g, '') === String(n)
  const submit = async () => {
    try {
      const r = await erase.mutateAsync({ params: { path: { workspace: slug } }, body: { ids, confirm: n } })
      await invalidate()
      onDone()
      toast({
        kind: 'deleted',
        title: t(m.erased, { n: r.done, count: count(r.done) }),
        detail: t(m.erasedBody),
      })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      dismissible={false}
      title={tr(m.eraseTitle, { n, count: count(n) })}
      description={tr(m.eraseBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="danger-primary" disabledReason={ok ? undefined : tr(m.typeToConfirm, { n: String(n) })} loading={erase.isPending} onClick={() => void submit()}>
            {tr(m.eraseButton, { n, count: count(n) })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <Note tone="info">{tr(m.eraseNote)}</Note>
        <Field label={tr(m.typeLabel, { n: String(n) })} error={err ?? undefined}>
          <Input value={typed} onChange={(e) => setTyped(e.target.value)} inputMode="numeric" autoComplete="off" />
        </Field>
      </div>
    </Dialog>
  )
}
