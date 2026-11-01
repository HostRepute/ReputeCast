import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, dayMonthYear, getDisplayTimeZone, until, zonedParts } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { msc } from './schedule.messages'

type Detail = Schemas['TransactionalMessageDetail']

const API = '/api/v1/workspaces/{workspace}/transactional' as const

const pad = (n: number) => String(n).padStart(2, '0')

/** "2026-11-04" and "09:00" of a moment in the display time zone. */
function wall(at: string): { day: string; time: string } {
  const p = zonedParts(at)
  return { day: `${p.year}-${pad(p.month)}-${pad(p.day)}`, time: `${pad(p.hour)}:${pad(p.minute)}` }
}

/** The moment a wall time ("2026-11-04", "09:00") is in a time zone. */
export function zonedInstant(day: string, time: string, tz: string | undefined): Date {
  const [y, mo, d] = day.split('-').map(Number)
  const [h, mi] = time.split(':').map(Number)
  const guess = Date.UTC(y!, mo! - 1, d!, h!, mi!)
  // The zone's offset at that moment, read back through Intl; twice, for
  // the hours around a clock change.
  let at = guess
  for (let i = 0; i < 2; i++) {
    const p = zonedPartsIn(new Date(at), tz)
    const shown = Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute)
    at = guess - (shown - at)
  }
  return new Date(at)
}

function zonedPartsIn(d: Date, tz: string | undefined) {
  const f = new Intl.DateTimeFormat('en-US', { timeZone: tz || undefined, year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric', hourCycle: 'h23' })
  const out: Record<string, number> = {}
  for (const x of f.formatToParts(d)) if (x.type !== 'literal') out[x.type] = Number(x.value)
  return { year: out.year!, month: out.month!, day: out.day!, hour: (out.hour ?? 0) % 24, minute: out.minute ?? 0 }
}

function useRefresh() {
  const qc = useQueryClient()
  // A cancel removes the content: its query is left alone, and the detail
  // that comes back no longer asks for it.
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const path = String((q.queryKey as unknown[])[1] ?? '')
        return (path.includes('/transactional') && !path.endsWith('/content')) || String(q.queryKey[0]).startsWith('transactional')
      },
    })
}

/**
 * A scheduled message's line and actions on its detail (review 2026-11
 * TX-1): when it goes out, Reschedule… and Cancel send…, for people who
 * can send.
 */
export function ScheduledCard({ d }: { d: Detail }) {
  const tr = useT()
  const canSend = useCan('transactional:send')
  const [open, setOpen] = useState<'cancel' | 'move'>()
  if (d.status !== 'scheduled' || !d.sendAt) return null
  const zone = getDisplayTimeZone() ?? 'UTC'
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border border-info-line bg-info-tint px-3.5 py-3">
      <span className="min-w-0 flex-[1_1_320px]">{tr(msc.goesOut, { when: `${dayMonthYear(d.sendAt)}, ${clock(d.sendAt)}`, zone, in: until(d.sendAt) })}</span>
      {canSend ? (
        <span className="flex flex-wrap gap-2">
          <Button size="sm" onClick={() => setOpen('move')}>
            {tr(msc.reschedule)}
          </Button>
          <Button size="sm" className="dline" onClick={() => setOpen('cancel')}>
            {tr(msc.cancelSend)}
          </Button>
        </span>
      ) : null}
      {open === 'cancel' ? <CancelDialog d={d} onClose={() => setOpen(undefined)} /> : null}
      {open === 'move' ? <MoveDialog d={d} onClose={() => setOpen(undefined)} /> : null}
    </div>
  )
}

function CancelDialog({ d, onClose }: { d: Detail; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={tr(msc.cancelBadge)}
      title={tr(msc.cancelTitle)}
      description={tr(msc.cancelBody, { email: d.email })}
      confirmLabel={tr(msc.cancelBadge)}
      cancelLabel={tr(msc.keepIt)}
      busyLabel={tr(msc.cancelling)}
      onConfirm={async () => {
        await http.POST(`${API}/messages/{messageId}/cancel`, { params: { path: { workspace: slug, messageId: d.id } } })
        await refresh()
        toast({ kind: 'success', title: t(msc.cancelled) })
        onClose()
      }}
    />
  )
}

function MoveDialog({ d, onClose }: { d: Detail; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const refresh = useRefresh()
  const zone = getDisplayTimeZone() ?? 'UTC'
  const start = wall(d.sendAt ?? new Date().toISOString())
  const [day, setDay] = useState(start.day)
  const [time, setTime] = useState(start.time)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const at = day && /^\d{2}:\d{2}$/.test(time) ? zonedInstant(day, time, zone) : undefined
  const when = at ? `${dayMonthYear(at)}, ${clock(at)}` : ''
  const save = async () => {
    if (!at) return
    setBusy(true)
    setErr(undefined)
    try {
      await http.POST(`${API}/messages/{messageId}/reschedule`, { params: { path: { workspace: slug, messageId: d.id } }, body: { sendAt: at.toISOString() } })
      await refresh()
      toast({ kind: 'success', title: t(msc.moved, { when }) })
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
      title={tr(msc.moveTitle)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!at} onClick={() => void save()}>
            {when ? tr(msc.moveTo, { when }) : tr(msc.moveTitle)}
          </Button>
        </>
      }
    >
      <form
        noValidate
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="lbl mb-1.5 p-0">{tr(msc.dayAndTime)}</legend>
          <div className="flex flex-wrap gap-2">
            <label className="vh" htmlFor={`${id}-day`}>
              {tr(msc.day)}
            </label>
            <input id={`${id}-day`} type="date" className="in" value={day} onChange={(e) => setDay(e.target.value)} aria-describedby={`${id}-hint`} />
            <label className="vh" htmlFor={`${id}-time`}>
              {tr(msc.time)}
            </label>
            <input id={`${id}-time`} type="time" className="in" value={time} onChange={(e) => setTime(e.target.value)} aria-describedby={`${id}-hint`} />
          </div>
          <span className="k" id={`${id}-hint`}>
            {tr(msc.moveHint, { zone })}
          </span>
        </fieldset>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}

/**
 * A suppressed recipient's actions where the rejection is explained
 * (review 2026-11 TX-2): remove the address, with a reason for the audit
 * log, or open the suppression.
 */
export function SuppressionActions({ d }: { d: Detail }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('subscribers:edit')
  const [open, setOpen] = useState(false)
  const s = d.rejection?.reason === 'suppressed' ? d.rejection.suppression : undefined
  if (!s?.id) return null
  return (
    <>
      {canEdit ? (
        <Button size="sm" onClick={() => setOpen(true)}>
          {tr(msc.removeSuppression)}
        </Button>
      ) : null}
      <Link className="btn sm" to="/w/$workspace/audience/suppressions" params={{ workspace: slug }} search={{ q: d.email }}>
        {tr(msc.openSuppression)}
      </Link>
      {open ? <RemoveDialog d={d} suppressionId={s.id} bounce={['hard_bounce', 'soft_bounce', 'complaint', 'bounce'].includes(s.reason)} onClose={() => setOpen(false)} /> : null}
    </>
  )
}

function RemoveDialog({ d, suppressionId, bounce, onClose }: { d: Detail; suppressionId: string; bounce: boolean; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const refresh = useRefresh()
  const [reason, setReason] = useState('')
  const [touched, setTouched] = useState(false)
  const short = reason.trim().length < 3
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={tr(msc.removeConfirm)}
      title={tr(msc.removeTitle, { email: d.email })}
      description={bounce ? `${tr(msc.removeBody)} ${tr(msc.removeBounce)}` : tr(msc.removeBody)}
      confirmLabel={tr(msc.removeConfirm)}
      busyLabel={tr(msc.removing)}
      onConfirm={async () => {
        setTouched(true)
        if (short) throw new Error(tr(msc.reasonShort))
        await http.POST('/api/v1/workspaces/{workspace}/suppressions/{id}/remove', { params: { path: { workspace: slug, id: suppressionId } }, body: { reason: reason.trim() } })
        await refresh()
        toast({ kind: 'success', title: t(msc.removed, { email: d.email }) })
        onClose()
      }}
    >
      <div className="field">
        <label className="lbl" htmlFor={`${id}-reason`}>
          {tr(msc.reason)}
        </label>
        <textarea
          id={`${id}-reason`}
          className="in min-h-[72px] py-2"
          value={reason}
          maxLength={300}
          aria-invalid={touched && short ? true : undefined}
          onChange={(e) => setReason(e.target.value)}
        />
        {touched && short ? <span className="err">{tr(msc.reasonShort)}</span> : null}
      </div>
    </ConfirmDialog>
  )
}
