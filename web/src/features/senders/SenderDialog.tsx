import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { problemMessage } from '@/features/connect/parts'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { sendersPath, type Overview, type Sender } from './SendersPage'

type Stream = 'broadcast' | 'automation' | 'system' | 'transactional'

const STREAMS: { key: Stream; label: string }[] = [
  { key: 'broadcast', label: 'Broadcasts' },
  { key: 'automation', label: 'Automations' },
  { key: 'system', label: 'System' },
  { key: 'transactional', label: 'Transactional' },
]

/** Add a sender, or edit one: name, reply-to, the streams it's the default for, the providers it's used with. */
export function SenderDialog({ open, onOpenChange, sender, data }: { open: boolean; onOpenChange: (o: boolean) => void; sender?: Sender; data: Overview }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('domains:manage', data.helpers)
  const [name, setName] = useState(sender?.name ?? '')
  const [email, setEmail] = useState(sender?.email ?? '')
  const [replyTo, setReplyTo] = useState(sender?.replyTo ?? '')
  const [note, setNote] = useState(sender?.note ?? '')
  const [defaults, setDefaults] = useState<Stream[]>((sender?.defaultFor ?? []) as Stream[])
  const [notUsed, setNotUsed] = useState<string[]>(sender ? sender.verification.filter((v) => v.status === 'not_used').map((v) => v.providerId) : [])
  const [error, setError] = useState<string>()
  // Transactional templates' default From must be on a Transactional domain.
  const domainOf = (sender?.email ?? email).split('@')[1]?.trim().toLowerCase()
  const onTransactionalDomain = Boolean(domainOf && data.domains.some((d) => d.transactional && d.domain === domainOf))
  const [busy, setBusy] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', sendersPath] })
  const body = (): Schemas['SenderInput'] => ({
    name: name.trim(),
    email: sender ? undefined : email.trim(),
    replyTo: replyTo.trim() || undefined,
    note: note.trim() || undefined,
    defaultFor: defaults,
    notUsedWith: notUsed,
  })
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      if (sender) {
        await http.PATCH('/api/v1/workspaces/{workspace}/senders/{senderId}', { params: { path: { workspace: slug, senderId: sender.id } }, body: body() })
      } else {
        await http.POST(sendersPath, { params: { path: { workspace: slug } }, body: body() })
      }
      await refresh()
      toast({ title: sender ? 'Sender saved' : `${email.trim()} added`, detail: sender ? undefined : 'Each provider verifies it through its domain records.' })
      onOpenChange(false)
    } catch (e) {
      setError(problemMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const toggle = <T,>(list: T[], v: T, on: boolean) => (on ? [...list, v] : list.filter((x) => x !== v))
  return (
    <>
      <Dialog
        open={open && !deleting}
        onOpenChange={onOpenChange}
        title={sender ? `Edit ${sender.email}` : 'Add sender'}
        description={sender ? undefined : 'The From address of your mail. Its domain needs records at each provider that sends it.'}
        footer={
          <>
            {sender ? (
              <Button variant="danger" className="mr-auto" onClick={() => setDeleting(true)} disabledReason={reason ?? (sender.defaultFor.length ? 'Make another sender the default first.' : undefined)}>
                Delete sender
              </Button>
            ) : null}
            <Button onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => void save()} loading={busy} disabledReason={reason ?? (!name.trim() || (!sender && !email.trim()) ? 'Enter a name and an address.' : undefined)}>
              {sender ? 'Save' : 'Add sender'}
            </Button>
          </>
        }
      >
        {error ? <Note tone="bad">{error}</Note> : null}
        <Field label="Name" hint="Shown as the sender in inboxes.">
          <Input value={name} onChange={(e) => setName(e.target.value)} autoComplete="off" />
        </Field>
        {sender ? null : (
          <Field label="Email address">
            <Input type="email" className="mono" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" placeholder="hello@news.example.com" />
          </Field>
        )}
        <Field label="Reply-to" optional hint="Where replies go when they shouldn't come back to the From address.">
          <Input type="email" className="mono" value={replyTo} onChange={(e) => setReplyTo(e.target.value)} autoComplete="off" />
        </Field>
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">Default sender for</legend>
          {STREAMS.filter((s) => s.key !== 'transactional' || onTransactionalDomain).map((s) => (
            <Checkbox key={s.key} label={s.label} checked={defaults.includes(s.key)} onChange={(e) => setDefaults((d) => toggle(d, s.key, e.target.checked))} />
          ))}
        </fieldset>
        <Field label="Note" optional hint="Shown in the list when it isn't a stream's default, e.g. personal sends.">
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        {data.providers.length ? (
          <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
            <legend className="mb-1 text-[12px] font-semibold text-ink-2">Sends through</legend>
            {data.providers.map((p) => (
              <Checkbox key={p.id} label={p.name} checked={!notUsed.includes(p.id)} onChange={(e) => setNotUsed((n) => toggle(n, p.id, !e.target.checked))} />
            ))}
          </fieldset>
        ) : null}
      </Dialog>
      {sender ? (
        <ConfirmDialog
          open={deleting}
          onOpenChange={(o) => {
            setDeleting(o)
            if (!o) onOpenChange(false)
          }}
          badge="Delete sender"
          title={`Delete ${sender.email}?`}
          description="Campaigns and automations that use it keep their copy of the address until you edit them. The domain and its records stay."
          confirmLabel="Delete sender"
          busyLabel="Deleting…"
          onConfirm={async () => {
            await http.DELETE('/api/v1/workspaces/{workspace}/senders/{senderId}', { params: { path: { workspace: slug, senderId: sender.id } } })
            await refresh()
            toast({ kind: 'deleted', title: `${sender.email} deleted` })
          }}
        />
      ) : null}
    </>
  )
}
