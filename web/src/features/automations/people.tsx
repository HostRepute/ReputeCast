import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, count, dayMonthDate, shortDate } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './complete.messages'
import { Pager } from '@/components/ui/Pager'

type StepContact = Schemas['AutomationStepContact']
type ContactRun = Schemas['AutomationContactRun']

/** "today 09:12", "29 Oct, 18:40". */
function when(iso: string): string {
  return shortDate(iso) === shortDate(new Date()) ? clock(iso) : `${dayMonthDate(iso)}, ${clock(iso)}`
}

/** What a contact at a step waits for, in words. */
function waitLine(c: StepContact): string {
  const parts = [t(m.hereSince, { when: when(c.since) })]
  if (c.until) parts.push(t(m.movesOn, { when: when(c.until) }))
  else if (c.wait === 'window') parts.push(t(m.heldWindow))
  else if (c.wait === 'cap') parts.push(t(m.heldCap))
  else if (c.wait === 'branch') parts.push(t(m.waitsOpen))
  else if (c.wait === 'webhook') parts.push(t(m.waitsWebhook))
  return parts.join(' · ')
}

/**
 * "See the 412 contacts here" under a step's numbers (review 2026-11 AT-2),
 * opening the contacts at the step, the latest to arrive first.
 */
export function StepContactsButton({ automationId, automationName, nodeId, step, here }: { automationId: string; automationName: string; nodeId: string; step: string; here: number }) {
  const tr = useT()
  const can = useCan('subscribers:view')
  const [open, setOpen] = useState(false)
  if (!can) return null
  return (
    <>
      <Button size="sm" className="self-start" disabled={here === 0} onClick={() => setOpen(true)}>
        {tr(m.contactsHere, { n: here })}
      </Button>
      {open ? <StepContactsDialog automationId={automationId} automationName={automationName} nodeId={nodeId} step={step} onClose={() => setOpen(false)} /> : null}
    </>
  )
}

function StepContactsDialog({ automationId, automationName, nodeId, step, onClose }: { automationId: string; automationName: string; nodeId: string; step: string; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('automations:publish')
  const [removing, setRemoving] = useState<StepContact>()
  const path = '/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/contacts' as const
  const q = useInfiniteQuery({
    queryKey: ['get', path, { workspace: slug, id: automationId, node: nodeId }, 'pages'],
    initialPageParam: '',
    queryFn: async ({ pageParam }) => {
      const res = await http.GET(path, { params: { path: { workspace: slug, id: automationId, node: nodeId }, query: { cursor: pageParam || undefined, limit: 25 } } })
      return res.data!
    },
    getNextPageParam: (last) => last.next || undefined,
  })
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  const total = q.data?.pages[0]?.total ?? 0
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="md"
      title={tr(m.stepContactsTitle, { step, n: total })}
      description={tr(m.newestFirst)}
      footer={
        q.hasNextPage ? (
          <>
            <span className="k mr-auto">{tr(m.shown, { shown: count(items.length), total: count(total) })}</span>
            <Pager query={q} label={tr(m.showMore)} size="md" />
          </>
        ) : undefined
      }
    >
      {q.isError ? (
        <ErrorPanel title={tr(m.loadFailed)}>{errorMessage(q.error)}</ErrorPanel>
      ) : !q.data ? (
        <div className="flex flex-col gap-3" aria-hidden="true">
          {[200, 160, 180].map((w, i) => (
            <Skeleton key={i} w={w} h={14} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <p className="k m-0">{tr(m.noOneHere)}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {items.map((c) => (
            <li key={c.runId} className="flex items-center gap-3 border-t border-line-2 py-2.5 first:border-t-0 max-sm:flex-wrap">
              <span className="min-w-0 flex-1">
                <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: c.contactId }} className="font-semibold">
                  {c.name || c.email}
                </Link>{' '}
                {c.name ? <span className="k mono [overflow-wrap:anywhere]">{c.email}</span> : null}
                <br />
                <span className="k">{waitLine(c)}</span>
              </span>
              <Button size="sm" className="dline" disabledReason={reason} aria-label={tr(m.removeLabel, { name: c.name || c.email })} onClick={() => setRemoving(c)}>
                {tr(m.removeFromFlow)}
              </Button>
            </li>
          ))}
        </ul>
      )}
      {removing ? (
        <RemoveFromFlowDialog
          automationId={automationId}
          automationName={automationName}
          runId={removing.runId}
          name={removing.name || removing.email}
          onClose={() => setRemoving(undefined)}
          onRemoved={() => void q.refetch()}
        />
      ) : null}
    </Dialog>
  )
}

/** Removing someone from a flow, with a reason for the audit log. */
export function RemoveFromFlowDialog({ automationId, automationName, runId, name, onClose, onRemoved }: { automationId: string; automationName: string; runId: string; name: string; onClose: () => void; onRemoved: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const remove = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/runs/{run}/remove')
  const [reason, setReason] = useState('')
  const [touched, setTouched] = useState(false)
  const [err, setErr] = useState<string>()
  const short = reason.trim().length < 3
  const submit = async () => {
    setTouched(true)
    if (short) return
    setErr(undefined)
    try {
      await remove.mutateAsync({ params: { path: { workspace: slug, id: automationId, run: runId } }, body: { reason: reason.trim() } })
      await qc.invalidateQueries({ predicate: (q) => String(q.queryKey[1] ?? '').includes('/automations') })
      toast({ kind: 'success', title: t(m.removed, { name, flow: automationName }) })
      onRemoved()
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(m.removeTitle, { name, flow: automationName })}
      description={tr(m.removeBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="danger-primary" loading={remove.isPending} onClick={() => void submit()}>
            {tr(m.removeFromFlow)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3"
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        {err ? <Note tone="bad">{err}</Note> : null}
        <Field label={tr(m.why)} error={touched && short ? tr(m.sayWhy) : undefined}>
          <Input value={reason} maxLength={300} autoFocus onChange={(e) => setReason(e.target.value)} />
        </Field>
      </form>
    </Dialog>
  )
}

const runTone: Record<ContactRun['status'], Tone> = { active: 'info', completed: 'ok', goal: 'ok', exited: 'neutral', failed: 'bad' }

/** The contact profile's Automations card: the flows they're in and where (AT-2). */
export function ContactAutomationsCard({ contactId, name }: { contactId: string; name: string }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const seeContacts = useCan('subscribers:view')
  const seeFlows = useCan('automations:view')
  const can = seeContacts && seeFlows
  const reason = usePermissionReason('automations:publish')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/automations', { params: { path: { workspace: slug, contactId } } }, { enabled: can })
  const [removing, setRemoving] = useState<ContactRun>()
  if (!can || !q.data || q.data.items.length === 0) return null
  const items = q.data.items
  const statusLabel = (r: ContactRun) =>
    r.status === 'active' ? tr(m.inProgress) : r.status === 'goal' ? tr(m.goal) : r.status === 'exited' ? tr(m.left) : r.status === 'failed' ? tr(m.failed) : tr(m.done)
  const reasonText = (r: ContactRun) =>
    r.exitReason === 'removed' ? tr(m.reasonRemoved) : r.exitReason === 'left_list' ? tr(m.reasonLeftList) : r.exitReason === 'filtered' ? tr(m.reasonFiltered) : 'none'
  return (
    <section className="card" aria-labelledby="ct-auto-h">
      <div className="sec">
        <h2 className="h2 m-0" id="ct-auto-h">
          {tr(m.flowsTitle, { count: count(items.length) })}
        </h2>
      </div>
      <ul className="m-0 flex list-none flex-col p-0">
        {items.map((r) => (
          <li key={r.runId} className="flex items-center gap-2.5 border-t border-line-2 px-4.5 py-3 first:border-t-0 max-sm:flex-wrap">
            <span className="min-w-0 flex-1">
              <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: r.automationId }} className="font-semibold">
                {r.name}
              </Link>
              <br />
              <span className="k">
                {r.status === 'active'
                  ? tr(m.atStep, { step: r.step, since: when(r.stepSince), entered: dayMonthDate(r.enteredAt) })
                  : tr(m.finishedOn, { status: r.status, date: dayMonthDate(r.finishedAt ?? r.stepSince), reason: reasonText(r) })}
              </span>
            </span>
            <Badge tone={runTone[r.status]}>{statusLabel(r)}</Badge>
            {r.status === 'active' ? (
              <Button size="sm" className="dline" disabledReason={reason} aria-label={tr(m.removeLabel, { name })} onClick={() => setRemoving(r)}>
                {tr(m.remove)}
              </Button>
            ) : null}
          </li>
        ))}
      </ul>
      {removing ? (
        <RemoveFromFlowDialog
          automationId={removing.automationId}
          automationName={removing.name}
          runId={removing.runId}
          name={name}
          onClose={() => setRemoving(undefined)}
          onRemoved={() => void q.refetch()}
        />
      ) : null}
    </section>
  )
}
