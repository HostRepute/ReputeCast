import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useInvalidateAudience } from './ContactDialogs'

type Item = Schemas['SuppressionItem']

/** "3 added; 1 was already on the list; 2 aren't addresses". */
function addSummary(r: Schemas['SuppressionAddResult']) {
  const parts = [`${count(r.added)} added`]
  if (r.already > 0) parts.push(`${count(r.already)} ${r.already === 1 ? 'was' : 'were'} already on the list`)
  if (r.invalid.length > 0) parts.push(`${count(r.invalid.length)} ${r.invalid.length === 1 ? "isn't an address" : "aren't addresses"}`)
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
      setErr('Enter at least one address.')
      return
    }
    if (emails.length > 1000) {
      setErr(`That's ${count(emails.length)} addresses; add up to 1,000 at a time, or upload a CSV.`)
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
        title: r.added === 1 && emails.length === 1 ? `Suppressed ${emails[0]}` : `Suppressed ${count(r.added)} ${r.added === 1 ? 'address' : 'addresses'}`,
        detail: `${addSummary(r)}. Every provider stops sending to them now; it's in the audit log.`,
      })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Add to the suppression list"
      description="No provider sends to these addresses again, on any stream, until someone removes them with a reason."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" loading={add.isPending} onClick={() => void submit()}>
            {emails.length > 1 ? `Suppress ${count(emails.length)} addresses` : 'Suppress address'}
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
            {invalid.length === 1 ? `"${invalid[0]}" isn't an email address.` : `None of these are email addresses: ${andList(invalid.slice(0, 5).map((x) => `"${x}"`))}${invalid.length > 5 ? ' …' : ''}.`}
          </Note>
        ) : null}
        <Field label="Addresses" hint="One per line, or separated by commas. Up to 1,000; for more, upload a CSV.">
          <Textarea
            className="mono min-h-[96px]"
            value={text}
            onChange={(e) => {
              setText(e.target.value)
              setInvalid([])
            }}
            placeholder="name@example.com"
            spellCheck={false}
          />
        </Field>
        <Field label="Why" optional hint="Shown as the original code, e.g. “customer asked by phone”.">
          <Input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </form>
    </Dialog>
  )
}

/** Remove: asks for a reason, which goes to the audit log with the old row. */
export function RemoveSuppressionDialog({ item, onClose }: { item: Item; onClose: () => void }) {
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
      ? 'Their contact goes back to pending: they only get mail again after confirming a new opt-in, so a bad address stays unmailed.'
      : r === 'unsubscribe'
        ? "They'll get Broadcasts and Automations again. Only do this when they asked to resubscribe."
        : 'Mail to this address resumes on every provider.'
  const submit = async () => {
    setTouched(true)
    setErr(null)
    if (short) return
    try {
      await remove.mutateAsync({ params: { path: { workspace: slug, id: item.id } }, body: { reason: reason.trim() } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: `Removed ${item.address} from the suppression list`, detail: 'The reason is in the audit log.' })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`Remove ${item.address}?`}
      description={effect}
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="danger-primary" loading={remove.isPending} onClick={() => void submit()}>
            Remove from the list
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
        <Field label="Reason" hint="Logged in the audit log with your name." error={touched && short ? 'Say why, in a few words.' : undefined}>
          <Input value={reason} maxLength={300} onChange={(e) => setReason(e.target.value)} placeholder="e.g. customer fixed their mailbox and asked to resubscribe" autoFocus />
        </Field>
      </form>
    </Dialog>
  )
}

type ImportMode = 'pull' | 'csv'

/** Import: pull from connected providers, or upload a CSV of addresses. */
export function ImportSuppressionsDialog({ onClose }: { onClose: () => void }) {
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
    toast({ kind: 'success', title: r.added > 0 ? `Pulled in ${count(r.added)} new ${r.added === 1 ? 'address' : 'addresses'}` : 'Nothing new to pull', detail: 'The pull is in the audit log.' })
  }
  const runUpload = async () => {
    if (!file) {
      setErr('Choose a CSV file with an email column.')
      return
    }
    if (file.size > 16 << 20) {
      setErr('That file is over 16 MB. Split it, or use the API.')
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
      title: `Suppressed ${count(data.added)} ${data.added === 1 ? 'address' : 'addresses'} from ${file.name}`,
      detail: `${count(data.rows)} rows read; ${addSummary(data)}.`,
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
      title="Import suppressions"
      description="Bring in addresses that must never be mailed: what your providers already block, or a list from your old tool."
      footer={
        pulled ? (
          <Button variant="primary" onClick={onClose}>
            Done
          </Button>
        ) : (
          <>
            <Button onClick={onClose}>Cancel</Button>
            <Button variant="primary" loading={busy} onClick={() => void submit()}>
              {mode === 'pull' ? 'Pull now' : 'Upload and suppress'}
            </Button>
          </>
        )
      }
    >
      {pulled ? (
        <div className="flex flex-col gap-2.5" role="status">
          {pulled.providers.length === 0 ? <p className="m-0 text-[13px]">No connected provider can share its suppressions.</p> : null}
          {pulled.providers.map((p) => (
            <div key={p.name} className="pkv">
              <span className="font-semibold">{p.name}</span>
              <span className="k text-right">{p.skipped ? p.skipped : `${count(p.pulled)} pulled · ${count(p.added)} new`}</span>
            </div>
          ))}
          <p className="k m-0">Addresses already on the list are skipped; nothing is pushed back until the next sync.</p>
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
            <legend className="vh">What to import</legend>
            <OptionCard
              name="supp-import"
              value="pull"
              checked={mode === 'pull'}
              onChange={() => setMode('pull')}
              title="Pull from your sending providers"
              description="Copies the bounces, complaints and unsubscribes SES, Postmark and others already block. Providers without a suppression API are skipped."
            />
            <OptionCard
              name="supp-import"
              value="csv"
              checked={mode === 'csv'}
              onChange={() => setMode('csv')}
              title="Upload a CSV"
              description="A file with an email column, e.g. the blocklist exported from Mailchimp or Brevo. Other columns are ignored."
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
                <span className="font-semibold">{file ? file.name : 'Drop a CSV here, or choose a file'}</span>
                <span className="k">{file ? `${count(Math.ceil(file.size / 1024))} KB` : 'Up to 16 MB'}</span>
              </label>
              <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
                <Field label="Reason for these addresses">
                  <Select value={reason} onChange={(e) => setReason(e.target.value as typeof reason)}>
                    <option value="manual">Manual</option>
                    <option value="hard_bounce">Hard bounce</option>
                    <option value="complaint">Complaint</option>
                    <option value="unsubscribe">Unsubscribed</option>
                  </Select>
                </Field>
                <Field label="Note" optional>
                  <Input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} placeholder="blocklist from the old tool" />
                </Field>
              </div>
            </>
          ) : null}
        </form>
      )}
    </Dialog>
  )
}
