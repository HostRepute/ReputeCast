import { useId, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Toggle } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, t, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from '@/features/automations/complete.messages'

type Cap = Schemas['MarketingCap']

/**
 * Settings › Sending rules › Marketing cap (review 2026-11 AT-9): at most N
 * marketing emails in X days per contact, campaigns and automations
 * together.
 */
export function MarketingCapCard({ cap, read, reason, onSaved }: { cap: Cap; read: boolean; reason?: string; onSaved: () => Promise<unknown> }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const [on, setOn] = useState(cap.count > 0)
  const [count, setCount] = useState(String(cap.count > 0 ? cap.count : 3))
  const [days, setDays] = useState(String(cap.days || 7))
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const save = async () => {
    const n = Number(count)
    const d = Number(days)
    if (on && (!Number.isInteger(n) || n < 1 || n > 50 || !Number.isInteger(d) || d < 1 || d > 90)) {
      setError(t(m.capRange))
      return
    }
    setError(undefined)
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/marketing-cap', { params: { path: { workspace: slug } }, body: { count: on ? n : 0, days: on ? d : cap.days || 7 } })
      toast({ kind: 'success', title: t(m.capSaved) })
      await onSaved()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const num = (value: string, set: (v: string) => void, label: string, max: number) => () => (
    <input
      type="number"
      inputMode="numeric"
      min={1}
      max={max}
      aria-label={label}
      className="in mx-1 inline-block h-[30px] w-16 px-1.5 py-0.5 align-baseline"
      disabled={read || busy || !on}
      value={value}
      onChange={(e) => set(e.target.value)}
    />
  )
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={`${id}-h`}>
            {tr(m.capTitle)}
          </h2>
          <span className="k">{tr(m.capCaption)}</span>
        </div>
        <Toggle label={tr(m.capSwitch)} hideLabel checked={on} disabled={read || busy} onChange={(e) => setOn(e.target.checked)} />
      </div>
      <div className="pad flex flex-col gap-3">
        <span className="text-[13px] leading-[2.2]">
          <T msg={m.capSentence} values={{ count: num(count, setCount, tr(m.capEmails), 50), days: num(days, setDays, tr(m.capDays), 90) }} />
        </span>
        <span className="k">{tr(m.capNote)}</span>
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        <div className="sr-actions sr-save">
          <Button variant="primary" loading={busy} disabledReason={reason} onClick={() => void save()}>
            {tr(m.capSave)}
          </Button>
        </div>
      </div>
    </section>
  )
}
