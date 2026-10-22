import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useInvalidateAudience } from './ContactDialogs'
import { m as cm } from './contacts.messages'
import { m } from './suppressions.messages'

type Item = Schemas['SuppressionItem']

/** "3 added; 1 was already on the list; 2 aren't addresses". */
function addSummary(r: Schemas['SuppressionAddResult']) {
  const parts = [t(m.addAdded, { count: count(r.added) })]
  if (r.already > 0) parts.push(t(m.addAlready, { n: r.already, count: count(r.already) }))
  if (r.invalid.length > 0) parts.push(t(m.addInvalid, { n: r.invalid.length, count: count(r.invalid.length) }))
  return parts.join('; ')
}

/** Splits pasted text on new lines, commas, semicolons and spaces. */
function splitAddresses(s: string) {
  return s
    .split(/[\s,;]+/)
    .map((x) => x.trim())
    .filter(Boolean)
}

/** Add address: one or more addresses typed or pasted, with a note. */
export function AddSuppressionsDialog({ onClose }: { onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/suppressions')
  const [text, setText] = useState('')
  const [note, setNote] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [invalid, setInvalid] = useState<string[]>([])
  const emails = splitAddresses(text)
  const submit = async () => {
    setErr(null)
    if (emails.length === 0) {
      setErr(t(m.enterOne))
      return
    }
    if (emails.length > 1000) {
      setErr(t(m.tooMany, { count: count(emails.length), max: count(1000) }))
      return
    }
    try {
      const r = await add.mutateAsync({ params: { path: { workspace: slug } }, body: { emails, note: note.trim() || undefined } })
      await invalidate()
      if (r.invalid.length > 0 && r.added === 0 && r.already === 0) {
        setInvalid(r.invalid)
        return
      }
      onClose()
      toast({
        kind: 'success',
        title: r.added === 1 && emails.length === 1 ? t(m.suppressedOne, { email: emails[0]! }) : t(m.suppressedMany, { n: r.added, count: count(r.added) }),
        detail: t(m.addDone, { summary: addSummary(r) }),
      })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={tr(m.addTitle)}
      description={tr(m.addBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={add.isPending} onClick={() => void submit()}>
            {emails.length > 1 ? tr(m.suppressMany, { count: count(emails.length) }) : tr(m.suppressOne)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        {err ? <Note tone="bad">{err}</Note> : null}
        {invalid.length > 0 ? (
          <Note tone="bad">
            {invalid.length === 1
              ? tr(m.notAnAddress, { value: invalid[0]! })
              : tr(m.noneAreAddresses, { more: invalid.length > 5 ? 'yes' : 'no', values: andList(invalid.slice(0, 5).map((x) => `"${x}"`)) })}
          </Note>
        ) : null}
        <Field label={tr(m.addressesLabel)} hint={tr(m.addressesHint, { max: count(1000) })}>
          <Textarea
            className="mono min-h-[96px]"
            value={text}
            onChange={(e) => {
              setText(e.target.value)
              setInvalid([])
            }}
            placeholder={'name@example.com'}
            spellCheck={false}
          />
        </Field>
        <Field label={tr(m.why)} optional hint={tr(m.whyHint)}>
          <Input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </form>
    </Dialog>
  )
}

/** Remove: asks for a reason, which goes to the audit log with the old row. */
export function RemoveSuppressionDialog({ item, onClose }: { item: Item; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const remove = $api.useMutation('post', '/api/v1/workspaces/{workspace}/suppressions/{id}/remove')
  const [reason, setReason] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [touched, setTouched] = useState(false)
  const short = reason.trim().length < 3
  const r = item.reasons[0] ?? 'manual'
  const effect =
    r === 'hard_bounce' || r === 'complaint' || r === 'spam_trap' || r === 'invalid'
      ? tr(m.effectBounced)
      : r === 'unsubscribe'
        ? tr(m.effectUnsubscribed)
        : tr(m.effectOther)
  const submit = async () => {
    setTouched(true)
    setErr(null)
    if (short) return
    try {
      await remove.mutateAsync({ params: { path: { workspace: slug, id: item.id } }, body: { reason: reason.trim() } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: t(m.removed, { address: item.address }), detail: t(m.removedBody) })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={tr(m.removeTitle, { address: item.address })}
      description={effect}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="danger-primary" loading={remove.isPending} onClick={() => void submit()}>
            {tr(m.removeButton)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        {err ? <Note tone="bad">{err}</Note> : null}
        <Field label={tr(cm.reason)} hint={tr(m.removeHint)} error={touched && short ? tr(m.sayWhy) : undefined}>
          <Input value={reason} maxLength={300} onChange={(e) => setReason(e.target.value)} placeholder={tr(m.removePlaceholder)} autoFocus />
        </Field>
      </form>
    </Dialog>
  )
}

type ImportMode = 'pull' | 'csv'

/** Import: pull from connected providers, or upload a CSV of addresses. */
export function ImportSuppressionsDialog({ onClose }: { onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const pull = $api.useMutation('post', '/api/v1/workspaces/{workspace}/suppressions/pull')
  const [mode, setMode] = useState<ImportMode>('pull')
  const [file, setFile] = useState<File | null>(null)
  const [over, setOver] = useState(false)
  const [reason, setReason] = useState<'manual' | 'hard_bounce' | 'complaint' | 'unsubscribe'>('manual')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [pulled, setPulled] = useState<Schemas['SuppressionPullResult'] | null>(null)
  const runPull = async () => {
    const r = await pull.mutateAsync({ params: { path: { workspace: slug } } })
    await invalidate()
    setPulled(r)
    toast({ kind: 'success', title: r.added > 0 ? t(m.pulledNew, { n: r.added, count: count(r.added) }) : t(m.nothingNew), detail: t(m.pullInAudit) })
  }
  const runUpload = async () => {
    if (!file) {
      setErr(t(m.chooseCsv))
      return
    }
    if (file.size > 16 << 20) {
      setErr(t(m.tooBig))
      return
    }
    const text = await file.text()
    const { data, error } = await http.POST('/api/v1/workspaces/{workspace}/suppressions/upload', {
      params: { path: { workspace: slug }, query: { fileName: file.name, reason, note: note.trim() || undefined } },
      body: text,
      headers: { 'Content-Type': 'text/csv' },
      bodySerializer: (b: string) => b,
    })
    if (error || !data) throw new Error(errorMessage(error))
    await invalidate()
    onClose()
    toast({
      kind: 'success',
      title: t(m.uploaded, { n: data.added, count: count(data.added), file: file.name }),
      detail: t(m.uploadedDetail, { n: data.rows, count: count(data.rows), summary: addSummary(data) }),
    })
  }
  const submit = async () => {
    setErr(null)
    setBusy(true)
    try {
      await (mode === 'pull' ? runPull() : runUpload())
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={tr(m.importTitle)}
      description={tr(m.importBody)}
      footer={
        pulled ? (
          <Button variant="primary" onClick={onClose}>
            {tr(common.done)}
          </Button>
        ) : (
          <>
            <Button onClick={onClose}>{tr(common.cancel)}</Button>
            <Button variant="primary" loading={busy} onClick={() => void submit()}>
              {mode === 'pull' ? tr(m.pullNow) : tr(m.uploadSuppress)}
            </Button>
          </>
        )
      }
    >
      {pulled ? (
        <div className="flex flex-col gap-2.5" role="status">
          {pulled.providers.length === 0 ? <p className="m-0 text-[13px]">{tr(m.noShare)}</p> : null}
          {pulled.providers.map((p) => (
            <div key={p.name} className="pkv">
              <span className="font-semibold">{p.name}</span>
              <span className="k text-right">{p.skipped ? p.skipped : tr(m.pulledLine, { pulled: count(p.pulled), added: count(p.added) })}</span>
            </div>
          ))}
          <p className="k m-0">{tr(m.skippedNote)}</p>
        </div>
      ) : (
        <form
          className="flex flex-col gap-3.5"
          noValidate
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          {err ? <Note tone="bad">{err}</Note> : null}
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="vh">{tr(m.whatToImport)}</legend>
            <OptionCard
              name="supp-import"
              value="pull"
              checked={mode === 'pull'}
              onChange={() => setMode('pull')}
              title={tr(m.pullTitle)}
              description={tr(m.pullBody)}
            />
            <OptionCard
              name="supp-import"
              value="csv"
              checked={mode === 'csv'}
              onChange={() => setMode('csv')}
              title={tr(m.csvTitle)}
              description={tr(m.csvBody)}
            />
          </fieldset>
          {mode === 'csv' ? (
            <>
              <label
                className={`dropz ${over ? 'over' : ''}`}
                onDragOver={(e) => {
                  e.preventDefault()
                  setOver(true)
                }}
                onDragLeave={() => setOver(false)}
                onDrop={(e) => {
                  e.preventDefault()
                  setOver(false)
                  const f = e.dataTransfer.files[0]
                  if (f) setFile(f)
                }}
              >
                <input type="file" accept=".csv,text/csv" className="vh" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
                <span className="font-semibold">{file ? file.name : tr(m.dropCsv)}</span>
                <span className="k">{file ? tr(m.fileSize, { size: count(Math.ceil(file.size / 1024)) }) : tr(m.upTo16)}</span>
              </label>
              <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
                <Field label={tr(m.reasonFor)}>
                  <Select value={reason} onChange={(e) => setReason(e.target.value as typeof reason)}>
                    <option value="manual">{tr(m.reasonManual)}</option>
                    <option value="hard_bounce">{tr(m.reasonHardBounce)}</option>
                    <option value="complaint">{tr(m.reasonComplaint)}</option>
                    <option value="unsubscribe">{tr(m.reasonUnsubscribe)}</option>
                  </Select>
                </Field>
                <Field label={tr(m.note)} optional>
                  <Input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} placeholder={tr(m.notePlaceholder)} />
                </Field>
              </div>
            </>
          ) : null}
        </form>
      )}
    </Dialog>
  )
}
