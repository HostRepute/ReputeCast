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
import { streamName, type Stream } from '@/features/providers/common'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { sendersPath, type Overview, type Sender } from './SendersPage'

const STREAMS: Stream[] = ['broadcast', 'automation', 'system', 'transactional']

/** Add a sender, or edit one: name, reply-to, the streams it's the default for, the providers it's used with. */
export function SenderDialog({ open, onOpenChange, sender, data }: { open: boolean; onOpenChange: (o: boolean) => void; sender?: Sender; data: Overview }) {
  const t = useT()
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
      toast({ title: sender ? t(m.senderSaved) : t(m.senderAdded, { email: email.trim() }), detail: sender ? undefined : t(m.senderAddedDetail) })
      onOpenChange(false)
    } catch (e) {
      setError(problemMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const toggle = <V,>(list: V[], v: V, on: boolean) => (on ? [...list, v] : list.filter((x) => x !== v))
  return (
    <>
      <Dialog
        open={open && !deleting}
        onOpenChange={onOpenChange}
        title={sender ? t(m.editSenderTitle, { email: sender.email }) : t(m.addSenderTitle)}
        description={sender ? undefined : t(m.addSenderDescription)}
        footer={
          <>
            {sender ? (
              <Button variant="danger" className="mr-auto" onClick={() => setDeleting(true)} disabledReason={reason ?? (sender.defaultFor.length ? t(m.makeOtherDefault) : undefined)}>
                {t(m.deleteSender)}
              </Button>
            ) : null}
            <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
            <Button variant="primary" onClick={() => void save()} loading={busy} disabledReason={reason ?? (!name.trim() || (!sender && !email.trim()) ? t(m.enterNameAndAddress) : undefined)}>
              {sender ? t(common.save) : t(m.addSenderSubmit)}
            </Button>
          </>
        }
      >
        {error ? <Note tone="bad">{error}</Note> : null}
        <Field label={t(m.name)} hint={t(m.nameHint)}>
          <Input value={name} onChange={(e) => setName(e.target.value)} autoComplete="off" />
        </Field>
        {sender ? null : (
          <Field label={t(m.emailAddress)}>
            <Input type="email" className="mono" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" placeholder={'hello@news.example.com'} />
          </Field>
        )}
        <Field label={t(m.replyTo)} optional hint={t(m.replyToHint)}>
          <Input type="email" className="mono" value={replyTo} onChange={(e) => setReplyTo(e.target.value)} autoComplete="off" />
        </Field>
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="mb-1 text-[12px] font-semibold text-ink-2">{t(m.defaultSenderFor)}</legend>
          {STREAMS.filter((s) => s !== 'transactional' || onTransactionalDomain).map((s) => (
            <Checkbox key={s} label={streamName[s]} checked={defaults.includes(s)} onChange={(e) => setDefaults((d) => toggle(d, s, e.target.checked))} />
          ))}
        </fieldset>
        <Field label={t(m.note)} optional hint={t(m.noteHint)}>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        {data.providers.length ? (
          <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
            <legend className="mb-1 text-[12px] font-semibold text-ink-2">{t(m.sendsThrough)}</legend>
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
          badge={t(m.deleteSender)}
          title={t(m.deleteSenderTitle, { email: sender.email })}
          description={t(m.deleteSenderBody)}
          confirmLabel={t(m.deleteSender)}
          busyLabel={t(m.deleting)}
          onConfirm={async () => {
            await http.DELETE('/api/v1/workspaces/{workspace}/senders/{senderId}', { params: { path: { workspace: slug, senderId: sender.id } } })
            await refresh()
            toast({ kind: 'deleted', title: t(m.senderDeleted, { email: sender.email }) })
          }}
        />
      ) : null}
    </>
  )
}
