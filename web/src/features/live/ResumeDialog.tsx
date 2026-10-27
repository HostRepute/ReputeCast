import { useState } from 'react'
import { ConfirmDialog, DialogOption } from '@/components/ui/ConfirmDialog'
import { Field } from '@/components/ui/Field'
import { Dialog } from '@/components/ui/Dialog'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, count } from '@/lib/format'
import type { LiveActions } from './actions'
import { m } from './messages'
import { pctOf, resumeChoices, scopeName, type Incident, type Snapshot } from './model'

/**
 * "Resume sending" (Dialogs board): resuming a scope an auto-pause holds,
 * as is. The threshold re-arms from the current counts (review GS-2); the
 * person picks between the recommended slower rate and a raised limit for
 * this campaign only, which the audit log records with their name.
 */
export function ResumeDialog({
  open,
  onOpenChange,
  incident,
  actions,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  incident: Incident
  actions: LiveActions
}) {
  const t = useT()
  const c = resumeChoices(incident)
  const [choice, setChoice] = useState<'rearm' | 'raise'>('rearm')
  const who = scopeName(incident)
  const metric = incident.metric
  const scope = incident.scope === 'mailbox_provider' ? { mailbox: incident.mailbox as never } : incident.scope === 'route' ? { providerId: incident.providerId } : {}
  const since = incident.since
  const confirm = () =>
    actions.resume(choice === 'raise' ? { ...scope, limit: c.raised } : c.rate ? { ...scope, rate: c.rate } : scope)
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="ok"
      badge={t(m.resumeBadge)}
      title={who ? t(m.resumeTitleWho, { held: count(incident.held), who }) : t(m.resumeTitleAll, { held: count(incident.held) })}
      description={[
        who
          ? t(m.resumeWasWho, { who, metric, value: pctOf(metric, incident.value ?? 0), time: clock(incident.trippedAt) })
          : t(m.resumeWasAll, { metric, value: pctOf(metric, incident.value ?? 0), time: clock(incident.trippedAt) }),
        since && since.base > 0 ? t(m.resumeSince, { metric, value: pctOf(metric, since.value), base: count(since.base) }) : '',
        t(m.resumeLimit, { limit: pctOf(metric, incident.limit ?? 0) }),
      ]
        .filter(Boolean)
        .join(' ')}
      confirmLabel={who ? t(m.resumeWho, { who }) : t(m.resumeSending)}
      cancelLabel={t(m.keepPaused)}
      onConfirm={confirm}
    >
      <DialogOption
        name={`resume-${incident.id}`}
        checked={choice === 'rearm'}
        onChange={() => setChoice('rearm')}
        title={
          c.rate
            ? t(m.resumeRearmAt, { rate: c.rateText, metric, limit: pctOf(metric, incident.limit ?? 0) })
            : t(m.resumeRearm, { metric, limit: pctOf(metric, incident.limit ?? 0) })
        }
      />
      <DialogOption
        name={`resume-${incident.id}`}
        checked={choice === 'raise'}
        onChange={() => setChoice('raise')}
        title={t(m.resumeRaise, { limit: pctOf(metric, c.raised) })}
        description={t(m.resumeRaiseNote)}
      />
    </ConfirmDialog>
  )
}

type Limits = { bounce: string; complaint: string; mailboxComplaint: string; deferral: string; apiErrors: string; scope: 'route' | 'campaign' }

const ranges = { bounce: [0.1, 20], complaint: [0.01, 5], mailboxComplaint: [0.01, 5], deferral: [1, 100], apiErrors: [1, 100] } as const

/**
 * "Edit for this send": the send's auto-pause thresholds while it sends or
 * waits (review GS-2). The workspace defaults stay; the change is audited.
 */
export function ThresholdsDialog({
  open,
  onOpenChange,
  s,
  actions,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  s: Snapshot
  actions: LiveActions
}) {
  const t = useT()
  const limit = (key: string) => String(s.thresholds.find((x) => x.key === key)?.limit ?? '')
  const initial = (): Limits => ({
    bounce: limit('bounce'),
    complaint: limit('complaint'),
    mailboxComplaint: limit('mailbox_complaint'),
    deferral: limit('deferral'),
    apiErrors: limit('api_errors'),
    scope: s.thresholdScope === 'campaign' ? 'campaign' : 'route',
  })
  const [v, setV] = useState<Limits>(initial)
  const [busy, setBusy] = useState(false)
  const errors = Object.fromEntries(
    (Object.keys(ranges) as (keyof typeof ranges)[]).map((k) => {
      const n = Number(v[k])
      const [lo, hi] = ranges[k]
      return [k, v[k] === '' || Number.isNaN(n) || n < lo || n > hi ? t(m.thresholdRange, { min: String(lo), max: String(hi) }) : undefined]
    }),
  ) as Record<keyof typeof ranges, string | undefined>
  const invalid = Object.values(errors).some(Boolean)
  const close = (next: boolean) => {
    if (busy) return
    if (next) setV(initial())
    onOpenChange(next)
  }
  const save = async () => {
    if (invalid || busy) return
    setBusy(true)
    try {
      await actions.setThresholds({
        bounce: Number(v.bounce),
        complaint: Number(v.complaint),
        mailboxComplaint: Number(v.mailboxComplaint),
        deferral: Number(v.deferral),
        apiErrors: Number(v.apiErrors),
        scope: v.scope,
      })
      setBusy(false)
      onOpenChange(false)
    } catch {
      // The toast said what failed; the values stay for another try.
      setBusy(false)
    }
  }
  const field = (key: keyof typeof ranges, label: string) => (
    <Field label={label} error={errors[key]}>
      <input className="in" type="number" inputMode="decimal" step="0.01" min={ranges[key][0]} max={ranges[key][1]} value={v[key]} onChange={(e) => setV({ ...v, [key]: e.target.value })} />
    </Field>
  )
  return (
    <Dialog
      open={open}
      onOpenChange={close}
      title={t(m.thresholdsEditTitle)}
      description={t(m.thresholdsEditBody, { name: s.name })}
      footer={
        <>
          <Button onClick={() => close(false)} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" onClick={() => void save()} loading={busy} disabledReason={invalid ? t(m.thresholdsFix) : undefined}>
            {t(m.thresholdsSave)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {field('bounce', t(m.thresholdsBounce))}
        {field('complaint', t(m.thresholdsComplaint))}
        {field('mailboxComplaint', t(m.thresholdsMailboxComplaint))}
        {field('deferral', t(m.thresholdsDeferral))}
        {field('apiErrors', t(m.thresholdsApiErrors))}
        <fieldset className="flex flex-col gap-1.5">
          <legend className="mb-1 text-[13px] font-semibold">{t(m.thresholdsScope)}</legend>
          <DialogOption name="threshold-scope" checked={v.scope === 'route'} onChange={() => setV({ ...v, scope: 'route' })} title={t(m.thresholdsScopeRoute)} />
          <DialogOption name="threshold-scope" checked={v.scope === 'campaign'} onChange={() => setV({ ...v, scope: 'campaign' })} title={t(m.thresholdsScopeCampaign)} />
        </fieldset>
      </div>
    </Dialog>
  )
}
