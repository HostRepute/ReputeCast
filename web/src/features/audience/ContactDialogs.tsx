import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input, OptionCard, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

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
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts')
  const form = useForm<NewContactForm>({
    defaultValues: { email: '', firstName: '', lastName: '', lists: lists[0] ? [lists[0].id] : [], consent: 'double', tags: '' },
  })
  const [err, setErr] = useState<string | null>(null)
  const consent = form.watch('consent')
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
        title: `Added ${v.email.trim()}`,
        detail: r.confirmationSent
          ? 'A confirmation email is on its way; they stay pending until they click it.'
          : r.confirmUrl
            ? 'No System provider could send the confirmation; share the link from their profile.'
            : undefined,
      })
    } catch (e) {
      const m = errorMessage(e)
      if (/email|address/i.test(m)) form.setError('email', { message: m })
      else setErr(m)
    }
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Add contact"
      description="Contacts added here are logged with your name as the source."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" loading={form.formState.isSubmitting} onClick={() => void submit()}>
            Add contact
          </Button>
        </>
      }
    >
      <form className="flex flex-col gap-3.5" onSubmit={(e) => void submit(e)} noValidate>
        {err ? <Note tone="bad">{err}</Note> : null}
        <Field label="Email" error={form.formState.errors.email?.message}>
          <Input
            type="email"
            autoComplete="off"
            className="mono"
            {...form.register('email', { validate: (v) => (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim()) ? true : 'Enter an email address like name@example.com.') })}
          />
        </Field>
        <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
          <Field label="First name" optional>
            <Input autoComplete="off" {...form.register('firstName')} />
          </Field>
          <Field label="Last name" optional>
            <Input autoComplete="off" {...form.register('lastName')} />
          </Field>
        </div>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">Lists</legend>
          {lists.map((l) => (
            <Checkbox key={l.id} value={l.id} label={l.name} hint={l.description || undefined} {...form.register('lists')} />
          ))}
        </fieldset>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">Consent</legend>
          <OptionCard
            name="consent"
            value="double"
            checked={consent === 'double'}
            onChange={() => form.setValue('consent', 'double')}
            title="Send a confirmation email"
            description="Double opt-in: they stay pending until they click the link."
          />
          <OptionCard
            name="consent"
            value="single"
            checked={consent === 'single'}
            onChange={() => form.setValue('consent', 'single')}
            title="They already agreed"
            description="Single opt-in: subscribed now. Only when you have their consent on record."
          />
        </fieldset>
        <Field label="Tags" optional hint="Separate tags with commas.">
          <Input autoComplete="off" placeholder="customer, beta" {...form.register('tags')} />
        </Field>
      </form>
    </Dialog>
  )
}

/** Save current view: a name for the filters, shared or only for me. */
export function SaveViewDialog({ onClose, filter }: { onClose: () => void; filter: Filter }) {
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const save = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/views')
  const [name, setName] = useState('')
  const [personal, setPersonal] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const empty = Object.keys(filter).length === 0
  const submit = async () => {
    if (!name.trim()) {
      setErr('Give the view a name.')
      return
    }
    try {
      await save.mutateAsync({ params: { path: { workspace: slug } }, body: { name: name.trim(), filters: filter, personal } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: `Saved the view "${name.trim()}"` })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title="Save current view"
      description={empty ? 'No filters are set, so this view shows every contact.' : 'The view keeps the current filters; its count updates as contacts change.'}
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" loading={save.isPending} onClick={() => void submit()}>
            Save view
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
        <Field label="Name" error={err ?? undefined}>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Engaged Outlook customers" maxLength={80} />
        </Field>
        <Checkbox label="Only for me" hint="Shared views show for everyone in the workspace." checked={personal} onChange={(e) => setPersonal(e.target.checked)} />
      </form>
    </Dialog>
  )
}

export type BulkKind = 'add_to_list' | 'add_tag' | 'resend_optin' | 'suppress'

const bulkCopy: Record<BulkKind, { title: string; action: string }> = {
  add_to_list: { title: 'Add to list', action: 'Add to list' },
  add_tag: { title: 'Add tag', action: 'Add tag' },
  resend_optin: { title: 'Resend opt-in', action: 'Resend opt-in' },
  suppress: { title: 'Suppress', action: 'Suppress' },
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
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/bulk')
  const [listId, setListId] = useState(lists[0]?.id ?? '')
  const [tag, setTag] = useState('')
  const [reason, setReason] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const n = ids.length
  const who = `${count(n)} selected ${n === 1 ? 'contact' : 'contacts'}`
  const submit = async () => {
    setErr(null)
    if (kind === 'add_tag' && !tag.trim()) {
      setErr('Type the tag to add.')
      return
    }
    if (kind === 'suppress' && !reason.trim()) {
      setErr('Say why, for the audit log.')
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
      title={bulkCopy[kind].title}
      description={
        kind === 'resend_optin'
          ? `Sends one confirmation email through the System stream to the ${who} who are still pending. Confirmed contacts are skipped.`
          : kind === 'suppress'
            ? `Adds the ${who} to the shared suppression list. Every provider stops sending to them; removing an address later needs a reason.`
            : `Applies to the ${who}.`
      }
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant={kind === 'suppress' ? 'danger-primary' : 'primary'} loading={run.isPending} onClick={() => void submit()}>
            {bulkCopy[kind].action}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {kind === 'add_to_list' ? (
          <Field label="List" error={err ?? undefined}>
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
          <Field label="Tag" error={err ?? undefined}>
            <Input value={tag} onChange={(e) => setTag(e.target.value)} placeholder="re-permission" maxLength={60} />
          </Field>
        ) : null}
        {kind === 'suppress' ? (
          <Field label="Reason" error={err ?? undefined} hint="Recorded in the audit log with your name.">
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Asked to be removed by phone" maxLength={300} />
          </Field>
        ) : null}
        {kind === 'resend_optin' && err ? <Note tone="bad">{err}</Note> : null}
      </div>
    </Dialog>
  )
}

function bulkTitle(r: Schemas['ContactBulkResult']) {
  const n = count(r.done)
  switch (r.action) {
    case 'add_to_list':
      return `Added ${n} ${r.done === 1 ? 'contact' : 'contacts'} to "${r.target}"`
    case 'add_tag':
      return `Tagged ${n} ${r.done === 1 ? 'contact' : 'contacts'} "${r.target}"`
    case 'resend_optin':
      return `Sent ${n} confirmation ${r.done === 1 ? 'email' : 'emails'}`
    case 'suppress':
      return `Suppressed ${n} ${r.done === 1 ? 'address' : 'addresses'}`
  }
  return `Done for ${n}`
}

/** A bulk action that partly failed (AudienceStates › Partly done). */
export function BulkResultPanel({ result, onDismiss }: { result: Schemas['ContactBulkResult']; onDismiss: () => void }) {
  const { slug } = useWorkspace()
  const [show, setShow] = useState(false)
  const verb: Record<string, string> = { add_to_list: 'Added', add_tag: 'Tagged', resend_optin: 'Sent opt-in to', suppress: 'Suppressed' }
  const target = result.action === 'add_to_list' ? ` to "${result.target}"` : result.action === 'add_tag' ? ` "${result.target}"` : ''
  const suppressed = result.failures.filter((f) => /suppress/i.test(f.reason)).length
  return (
    <div role="alert" className="card flex flex-col gap-2 border-[#FCD34D] px-4.5 py-4">
      <div className="flex flex-wrap items-center gap-2.5">
        <Badge tone="warn">Partly done</Badge>
        <span className="font-semibold">
          {verb[result.action] ?? 'Done for'} {count(result.done)} of {count(result.matched)} contacts{target}; {count(result.failed)} failed
        </span>
      </div>
      <span className="k">
        {suppressed === result.failed
          ? `${count(result.failed)} ${result.failed === 1 ? 'address is' : 'addresses are'} on the suppression list and can't be added to a list without a new opt-in. Everything else succeeded.`
          : `${result.failures[0]?.reason ?? 'Some contacts could not be changed.'} Everything else succeeded.`}
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
            See the {count(result.failed)} {result.failed === 1 ? 'address' : 'addresses'}
          </Link>
        ) : (
          <Button size="sm" onClick={() => setShow((v) => !v)}>
            {show ? 'Hide' : 'See'} the {count(result.failed)} {result.failed === 1 ? 'contact' : 'contacts'}
          </Button>
        )}
        <Button size="sm" onClick={onDismiss}>
          Dismiss
        </Button>
      </div>
    </div>
  )
}

/** Delete: erase personal data, confirmed by typing the number of contacts. */
export function EraseDialog({ onClose, ids, onDone }: { onClose: () => void; ids: string[]; onDone: () => void }) {
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
        title: `Erased ${count(r.done)} ${r.done === 1 ? 'contact' : 'contacts'}`,
        detail: 'Their addresses stay on the suppression list so they are never mailed again.',
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
      title={`Erase ${count(n)} ${n === 1 ? 'contact' : 'contacts'}?`}
      description="Names, fields, tags, list memberships, consent history and preferences are removed for good. This can't be undone."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="danger-primary" disabledReason={ok ? undefined : `Type ${n} to confirm.`} loading={erase.isPending} onClick={() => void submit()}>
            Erase {count(n)} {n === 1 ? 'contact' : 'contacts'}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <Note tone="info">Their addresses are kept on the suppression list, so an erased contact is never emailed again, even if an import brings them back.</Note>
        <Field label={`Type ${n} to confirm`} error={err ?? undefined}>
          <Input value={typed} onChange={(e) => setTyped(e.target.value)} inputMode="numeric" autoComplete="off" />
        </Field>
      </div>
    </Dialog>
  )
}
