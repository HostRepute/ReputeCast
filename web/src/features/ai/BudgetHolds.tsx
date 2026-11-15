import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { aiBase, useRefreshAI } from './api'
import { dollars, tokens } from './format'
import { m } from './messages'
import { AINote } from './parts'

type Hold = Schemas['AIBudgetHold']

export function BudgetHolds() {
  const t = useT()
  const { slug } = useWorkspace()
  const [selected, setSelected] = useState<Hold>()
  const q = $api.useQuery('get', `${aiBase}/budget-holds`, { params: { path: { workspace: slug } } }, { refetchInterval: 30_000 })
  if (q.isPending) return null
  if (q.error) return <AINote tone="bad">{errorMessage(q.error)}</AINote>
  if (!q.data.items.length) return null
  return (
    <section className="card ai-card" aria-labelledby="ai-holds-title">
      <header className="ai-ch"><div><h2 id="ai-holds-title" className="h2">{t(m.holdsTitle)}</h2><p className="k">{t(m.holdsDescription)}</p></div></header>
      <div className="ai-cb ai-rows">
        {q.data.items.map((hold) => (
          <div key={hold.id} className="flex-wrap gap-3">
            <div className="flex min-w-0 flex-col gap-1 [overflow-wrap:anywhere]">
              <strong>{hold.model}</strong>
              <span className="k">{shortDate(hold.createdAt)} · {clock(hold.createdAt)} · {tokens(hold.reservedTokens)} · {dollars(hold.reservedCostMicros)}</span>
              <span className="k mono text-xs">{hold.id}</span>
            </div>
            <Button className="min-h-11" onClick={() => setSelected(hold)}>{t(m.reconcile)}</Button>
          </div>
        ))}
      </div>
      {selected ? <ReconcileDialog key={selected.id} hold={selected} onClose={() => setSelected(undefined)} /> : null}
    </section>
  )
}

function ReconcileDialog({ hold, onClose }: { hold: Hold; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshAI()
  const now = useNow()
  const [input, setInput] = useState('')
  const [output, setOutput] = useState('')
  const [confirmed, setConfirmed] = useState(false)
  const save = $api.useMutation('post', `${aiBase}/budget-holds/{id}/reconcile`, {
    onSuccess: () => { void refresh(); toast({ title: t(m.reconcileSaved), kind: 'success' }); onClose() },
  })
  const ready = now.getTime() >= Date.parse(hold.reconcileAfter)
  return (
    <Dialog open onOpenChange={(open) => { if (!open && !save.isPending) onClose() }} dismissible={!save.isPending} title={t(m.reconcile)} description={t(m.holdsDescription)}
      footer={<><Button disabled={save.isPending} onClick={onClose}>{t(m.cancel)}</Button><Button variant="primary" type="submit" form="ai-reconcile-form" disabled={!ready || !confirmed || input === '' || output === '' || save.isPending}>{t(m.reconcile)}</Button></>}>
      <form id="ai-reconcile-form" className="flex flex-col gap-4" onSubmit={(e) => {
        e.preventDefault()
        if (ready && confirmed && !save.isPending) save.mutate({ params: { path: { workspace: slug, id: hold.id } }, body: { inputTokens: Number(input), outputTokens: Number(output), confirmed } })
      }}>
        <p className="[overflow-wrap:anywhere]">{hold.model} · <span className="mono text-xs">{hold.id}</span></p>
        {!ready ? <AINote tone="warn">{t(m.reconcileWait, { time: `${shortDate(hold.reconcileAfter)} ${clock(hold.reconcileAfter)}` })}</AINote> : null}
        {save.error ? <AINote tone="bad">{errorMessage(save.error)}</AINote> : null}
        <Field label={t(m.reconcileInput)}><Input type="number" inputMode="numeric" min={0} max={1_000_000_000} step={1} required value={input} onChange={(e) => setInput(e.target.value)} disabled={save.isPending} /></Field>
        <Field label={t(m.reconcileOutput)}><Input type="number" inputMode="numeric" min={0} max={1_000_000_000} step={1} required value={output} onChange={(e) => setOutput(e.target.value)} disabled={save.isPending} /></Field>
        <Checkbox label={t(m.reconcileConfirm)} className="min-h-11" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} disabled={save.isPending} required />
      </form>
    </Dialog>
  )
}
