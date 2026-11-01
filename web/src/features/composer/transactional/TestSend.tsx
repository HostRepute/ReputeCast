import { useId, useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { useComposer } from '../context'
import { useTx } from './state'
import { mts } from './testsend.messages'

/**
 * Sends the draft as a test email, filled with one of its test data sets,
 * through the System route (review 2026-11 TX-5). Unsaved edits are saved
 * first, so the test is what the editor shows.
 */
export function TemplateTestDialog({ onClose }: { onClose: () => void }) {
  const tr = useT()
  const id = useId()
  const { slug } = useWorkspace()
  const c = useComposer()
  const tx = useTx()
  const me = useMe().data?.user
  const canSend = useCan('transactional:send')
  const [to, setTo] = useState(me?.email ?? '')
  const [data, setData] = useState(tx.previewWith || tx.testData[0]?.name || '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const send = async () => {
    setBusy(true)
    setErr(undefined)
    try {
      if (c.canEdit && c.hasUnsaved?.()) await c.save({ quiet: true })
      const email = to.trim()
      await http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/test', {
        params: { path: { workspace: slug, template: tx.template.id } },
        body: { sendTo: email, testData: data || undefined },
      })
      toast({ kind: 'success', title: data ? t(mts.sentWith, { email, name: data }) : t(mts.sent, { email }) })
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(mts.title, { key: tx.template.key })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!to.trim()} onClick={() => void send()}>
            {busy ? tr(mts.sending) : tr(mts.send)}
          </Button>
        </>
      }
    >
      <form
        noValidate
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void send()
        }}
      >
        <div className="field">
          <label className="lbl" htmlFor={`${id}-to`}>
            {tr(mts.sendTo)}
          </label>
          <input
            id={`${id}-to`}
            type="email"
            className="in w-full"
            value={to}
            readOnly={!canSend}
            aria-describedby={canSend ? undefined : `${id}-own`}
            onChange={(e) => setTo(e.target.value)}
          />
          {canSend ? null : (
            <span className="k" id={`${id}-own`}>
              {tr(mts.ownOnly)}
            </span>
          )}
        </div>
        <div className="field">
          <label className="lbl" htmlFor={`${id}-data`}>
            {tr(mts.fillWith)}
          </label>
          <select id={`${id}-data`} className="in w-full" value={data} onChange={(e) => setData(e.target.value)}>
            {tx.testData.length ? null : <option value="">{tr(mts.samples)}</option>}
            {tx.testData.map((d) => (
              <option key={d.name} value={d.name}>
                {tr(mts.testData, { name: d.name })}
              </option>
            ))}
          </select>
        </div>
        <span className="k">{tr(mts.note)}</span>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}
