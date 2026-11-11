import { Link } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Textarea, Toggle } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, t, useT } from '@/i18n'
import { dayMonthDate, trimPct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './holdout.messages'

type Holdout = Schemas['HoldoutSettings']
type Input = Schemas['HoldoutInput']

const holdoutPath = '/api/v1/workspaces/{workspace}/holdout' as const

/** A share typed in percent ("5", "2.5", "2,5") as basis points, or undefined when it's outside 0.1–20%. */
export function shareToBasisPoints(text: string): number | undefined {
  const v = Number(text.trim().replace(',', '.'))
  if (!text.trim() || !Number.isFinite(v)) return undefined
  const bp = Math.round(v * 100)
  if (Math.abs(v * 100 - bp) > 1e-6 || bp % 10 !== 0) return undefined // one decimal at most
  return bp >= 10 && bp <= 2000 ? bp : undefined
}

/**
 * Settings › Sending rules › Holdout group (M63 G-33): keep a random share
 * of contacts out of campaigns and/or automations, to measure what
 * marketing adds. The switch saves at once and goes back if the server
 * refuses; the share and scope save with Save; Start over draws new people.
 */
export function HoldoutCard({ holdout, read, reason, onSaved }: { holdout: Holdout; read: boolean; reason?: string; onSaved: () => Promise<unknown> }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const [on, setOn] = useState(holdout.on)
  const [share, setShare] = useState(String(holdout.percent))
  const [campaigns, setCampaigns] = useState(holdout.campaigns)
  const [automations, setAutomations] = useState(holdout.automations)
  const [shareError, setShareError] = useState<string>()
  const [scopeError, setScopeError] = useState<string>()
  const [busy, setBusy] = useState<'switch' | 'save'>()
  const [restarting, setRestarting] = useState(false)

  const put = (body: Input) => http.PUT(holdoutPath, { params: { path: { workspace: slug } }, body })

  // The switch saves at once with the saved share and scope, and goes back to
  // where it was when the server refuses.
  const toggle = async (next: boolean) => {
    setOn(next)
    setBusy('switch')
    try {
      await put({ on: next, basisPoints: holdout.basisPoints, campaigns: holdout.campaigns, automations: holdout.automations })
      toast({ kind: 'success', title: t(next ? m.turnedOn : m.turnedOff) })
      await onSaved()
    } catch (e) {
      setOn(!next)
      toast({ kind: 'error', title: t(m.notChanged), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }

  const save = async () => {
    const bp = shareToBasisPoints(share)
    const se = bp === undefined ? t(m.shareRange) : undefined
    const ce = !campaigns && !automations ? t(m.scopeRequired) : undefined
    setShareError(se)
    setScopeError(ce)
    if (se || ce || bp === undefined) return
    setBusy('save')
    try {
      await put({ on: true, basisPoints: bp, campaigns, automations })
      toast({ kind: 'success', title: t(m.saved) })
      await onSaved()
    } catch (e) {
      toast({ kind: 'error', title: t(m.notChanged), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }

  const locked = read || busy !== undefined
  const fieldsOff = locked || !on
  const shareInput = () => (
    <input
      type="number"
      inputMode="decimal"
      min={0.1}
      max={20}
      step={0.1}
      aria-label={tr(m.shareLabel)}
      aria-invalid={shareError ? true : undefined}
      aria-describedby={shareError ? `${id}-se` : undefined}
      className="in mx-1 inline-block h-[30px] w-16 min-w-0 px-1.5 py-0.5 align-baseline"
      disabled={fieldsOff}
      value={share}
      onChange={(e) => setShare(e.target.value)}
    />
  )
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={`${id}-h`}>
            {tr(m.title)}
          </h2>
          <span className="k">{tr(m.caption)}</span>
        </div>
        <Toggle
          label={tr(m.switchLabel)}
          hideLabel
          checked={on}
          disabled={locked}
          aria-busy={busy === 'switch' || undefined}
          onChange={(e) => void toggle(e.target.checked)}
        />
      </div>
      <div className="pad flex flex-col gap-3">
        <span className={on ? 'text-[13px] leading-[2.2]' : 'text-[13px] leading-[2.2] text-ink-3'}>
          <T msg={m.sentence} values={{ share: shareInput }} />
        </span>
        {shareError ? (
          <span className="err -mt-2" id={`${id}-se`} role="alert">
            {shareError}
          </span>
        ) : null}
        <fieldset className={on ? 'sr-fs' : 'sr-fs text-ink-3'} aria-describedby={scopeError ? `${id}-fe` : undefined}>
          <legend className={on ? undefined : 'text-ink-3'}>{tr(m.scope)}</legend>
          <div className="flex flex-wrap gap-x-6 gap-y-1">
            <label className="chk sr-chk">
              <input type="checkbox" checked={campaigns} disabled={fieldsOff} onChange={(e) => setCampaigns(e.target.checked)} />
              <span>{tr(m.campaigns)}</span>
            </label>
            <label className="chk sr-chk">
              <input type="checkbox" checked={automations} disabled={fieldsOff} onChange={(e) => setAutomations(e.target.checked)} />
              <span>{tr(m.automations)}</span>
            </label>
          </div>
          {scopeError ? (
            <span className="err" id={`${id}-fe`} role="alert">
              {scopeError}
            </span>
          ) : null}
        </fieldset>
        <div className="flex flex-col gap-0.5">
          <span className="k">{tr(m.noteAlways)}</span>
          <span className="k">{tr(m.noteRaise)}</span>
        </div>
        {holdout.on && holdout.startedAt ? (
          <span className="k">
            <T
              msg={m.status}
              values={{
                count: holdout.heldOut,
                date: dayMonthDate(holdout.startedAt),
                link: (c) => (
                  <Link to="/w/$workspace/audience/holdout" params={{ workspace: slug }} className="font-semibold">
                    {c}
                  </Link>
                ),
              }}
            />
          </span>
        ) : (
          <span className="k">{tr(m.off)}</span>
        )}
        {holdout.on ? (
          <div className="sr-actions sr-save">
            <Button disabledReason={reason} disabled={busy !== undefined} onClick={() => setRestarting(true)}>
              {tr(m.startOver)}
            </Button>
            <Button variant="primary" loading={busy === 'save'} disabledReason={reason} disabled={busy === 'switch' || !on} onClick={() => void save()}>
              {tr(m.save)}
            </Button>
          </div>
        ) : null}
      </div>
      {restarting ? (
        <StartOverDialog
          percent={trimPct(holdout.percent)}
          onClose={() => setRestarting(false)}
          onDone={async () => {
            setRestarting(false)
            await onSaved()
          }}
        />
      ) : null}
    </section>
  )
}

/** "Start over with new people?": the consequences, an optional reason for the audit log, Cancel · Start over. */
function StartOverDialog({ percent, onClose, onDone }: { percent: string; onClose: () => void; onDone: () => Promise<void> }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const [why, setWhy] = useState('')
  const [busy, setBusy] = useState(false)
  const restart = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/holdout/restart', { params: { path: { workspace: slug } }, body: { reason: why.trim() || undefined } })
      toast({ kind: 'success', title: t(m.restarted) })
      await onDone()
    } catch (e) {
      toast({ kind: 'error', title: t(m.restartFailed), detail: errorMessage(e) })
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o || busy ? undefined : onClose())}
      title={tr(m.dialogTitle)}
      hideClose
      dismissible={!busy}
      footer={
        <>
          <Button disabled={busy} onClick={onClose}>
            {tr(m.cancel)}
          </Button>
          <Button variant="primary" loading={busy} onClick={() => void restart()}>
            {tr(m.dialogConfirm)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex flex-col gap-1.5 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{tr(m.dialogEnds)}</li>
        <li>{tr(m.dialogNew, { percent })}</li>
        <li>{tr(m.dialogAgain)}</li>
      </ul>
      <div className="lbl">
        <label htmlFor={`${id}-why`}>{tr(m.dialogReason)}</label>
        <Textarea id={`${id}-why`} rows={3} maxLength={500} className="h-auto px-2.5 py-2" value={why} onChange={(e) => setWhy(e.target.value)} />
      </div>
    </Dialog>
  )
}
