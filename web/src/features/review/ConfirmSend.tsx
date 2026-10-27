import { Dialog as RDialog } from 'radix-ui'
import { useId, useState } from 'react'
import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { count, money } from '@/lib/format'
import { confirmLanguages } from '@/features/languages/review'
import { mr } from '@/features/languages/messages'
import { useT } from '@/i18n'
import { acknowledgedKeys, acknowledgements, minutes, scheduleSentence, thresholdsLine, verdictTitle, type Review, type Run } from './logic'
import { m } from './messages'

export type Canary = 'auto' | 'on' | 'off'

export type SendIntent = { acknowledged: string[]; confirm?: string; canary: Canary }

/**
 * The deliberate last step before a send (SendReadinessConfirm): what goes
 * out, to whom and how; each warning confirmed; for large sends the
 * recipient count typed out.
 */
export function ConfirmSend({
  open,
  onOpenChange,
  review,
  run,
  action,
  onConfirm,
  onExclude,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  review: Review
  run: Run
  /** "Schedule send", "Send now", "Schedule as approved". */
  action: string
  onConfirm: (intent: SendIntent) => Promise<void>
  /** "Exclude them instead": applies the unengaged fix and closes. */
  onExclude?: () => void
}) {
  const id = useId()
  const t = useT()
  const languages = confirmLanguages(run.checks, t)
  const c = review.campaign
  const recipients = run.estimates.audience.final
  const acks = acknowledgements(run)
  const large = recipients >= review.largeSend
  const [checked, setChecked] = useState<string[]>([])
  const [typed, setTyped] = useState('')
  // The canary phase (C1): the engine decides, or the person asks for one or against one.
  const [canary, setCanary] = useState<Canary>(review.canary.choice)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const schedule = review.schedule
  const route = run.estimates.route
  const shares = route.shares.filter((s) => s.recipients > 0)
  const a = run.estimates.audience
  const missing = acks.filter((x) => !checked.includes(x.key)).length
  const countOK = !large || typed.replace(/\D/g, '') === String(recipients)
  const reason = missing ? t(m.confirmWarnings, { count: missing }) : !countOK ? t(m.typeCount, { count: String(recipients) }) : undefined
  const close = (next: boolean) => {
    if (busy) return
    if (!next) {
      setChecked([])
      setTyped('')
      setError(undefined)
    }
    onOpenChange(next)
  }
  const confirm = async () => {
    if (reason || busy) return
    setBusy(true)
    setError(undefined)
    try {
      await onConfirm({ acknowledged: acknowledgedKeys(acks, checked), confirm: large ? typed : undefined, canary })
      setBusy(false)
      close(false)
    } catch (e) {
      setBusy(false)
      setError(e instanceof ApiError ? e.message : t(m.unreachable))
    }
  }
  const title = schedule.mode === 'now' ? t(m.confirmTitleNow, { name: c.name, recipients }) : t(m.confirmTitleAt, { name: c.name, recipients })
  const when =
    schedule.mode === 'now'
      ? t(m.whenNow)
      : schedule.mode === 'best_time'
        ? t(m.whenBest, { when: scheduleSentence(schedule) })
        : t(m.whenAt, { when: scheduleSentence(schedule) })
  const excluded = [
    a.suppressed ? t(m.excludedSuppressed, { count: count(a.suppressed) }) : '',
    a.role ? t(m.excludedRole, { count: count(a.role) }) : '',
    a.duplicates ? t(m.excludedDuplicates, { count: count(a.duplicates) }) : '',
    a.excludedUnengaged ? t(m.excludedUnengaged, { count: count(a.excludedUnengaged) }) : '',
  ]
    .filter(Boolean)
    .join(' · ')
  const plan = review.canary
  const canaryHint =
    canary === 'on'
      ? t(m.canaryOnHint)
      : canary === 'off'
        ? t(m.canaryOffHint)
        : !plan.planned
          ? t(m.canaryPlanUnknown)
          : plan.auto
            ? t(m.canaryPlanAuto, { size: count(plan.size), reasons: plan.reasons.join('; ') || t(m.canaryPlanNew) })
            : t(m.canaryPlanNone)
  const subjectLabel = t(m.rowSubject)
  const rows: [string, string][] = [
    [t(m.rowFrom), c.from || '—'],
    [t(m.rowReplyTo), c.replyTo || t(m.sameAsFrom)],
    [subjectLabel, c.subject],
    ...(languages ? [[t(mr.confirmLabel), languages] as [string, string]] : []),
    [t(m.rowAb), c.ab ? t(m.abLabel, { kind: c.ab.kind, variants: c.ab.variants, percent: c.ab.testPercent }) : t(m.noTest)],
    [t(m.rowRoute), shares.map((s) => `${s.short} ${count(s.recipients)}`).join(' · ')],
    [t(m.rowCost), t(m.costValue, { cost: money(route.totalCost, 2), duration: minutes(route.totalMinutes) })],
    [t(m.rowAutoPause), thresholdsLine(review.thresholds)],
    [t(m.rowExcluded), excluded || t(m.nobody)],
  ]
  return (
    <RDialog.Root open={open} onOpenChange={close}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl">
          <RDialog.Content className="rv-confirm" aria-describedby={`${id}-d`} onPointerDownOutside={(e) => e.preventDefault()}>
            <section className="c-head">
              <Badge tone={run.verdict === 'warnings' ? 'warn' : 'ok'} dot className="self-start">
                {verdictTitle(run.verdict)}
              </Badge>
              <RDialog.Title className="c-title">{title}</RDialog.Title>
              <RDialog.Description id={`${id}-d`} className="text-ink-2">
                {when}
              </RDialog.Description>
            </section>
            <section>
              {/* Two columns as on the board; one on small screens, where two would break the labels. */}
              <div className="grid grid-cols-2 gap-x-6 gap-y-2.5 max-sm:grid-cols-1 [&>*]:min-w-0">
                {rows.map(([k, v]) => (
                  <div key={k} className="rv-sum">
                    <span className="k shrink-0">{k}</span>
                    <span className={k === subjectLabel ? 'n text-right' : 'n'}>{v}</span>
                  </div>
                ))}
              </div>
            </section>
            <section className="c-canary">
              <Field label={t(m.canaryLabel)} hint={canaryHint}>
                <Select value={canary} disabled={busy} onChange={(e) => setCanary(e.target.value as Canary)}>
                  <option value="auto">{t(m.canaryAuto)}</option>
                  <option value="on">{t(m.canaryOn)}</option>
                  <option value="off">{t(m.canaryOff)}</option>
                </Select>
              </Field>
            </section>
            {acks.length ? (
              <section className="c-acks" aria-label={t(m.warningsToConfirm)}>
                {acks.map((x) => (
                  <label key={x.key} className="rv-chk">
                    <input
                      type="checkbox"
                      checked={checked.includes(x.key)}
                      onChange={(e) => setChecked((v) => (e.target.checked ? [...v, x.key] : v.filter((k) => k !== x.key)))}
                    />
                    <span>
                      {x.text}
                      {x.fix && onExclude ? (
                        <>
                          {' '}
                          <a
                            href="#exclude"
                            onClick={(e) => {
                              e.preventDefault()
                              close(false)
                              onExclude()
                            }}
                          >
                            {t(m.excludeInstead)}
                          </a>
                        </>
                      ) : null}
                    </span>
                  </label>
                ))}
              </section>
            ) : null}
            <section className="c-last">
              {large ? (
                <label className="rv-typed">
                  {t(m.typeToConfirm)}
                  <input
                    className="in mono rv-count"
                    inputMode="numeric"
                    autoComplete="off"
                    value={typed}
                    placeholder={String(recipients)}
                    onChange={(e) => setTyped(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') void confirm()
                    }}
                  />
                  <span className="k font-medium">{t(m.deliberate)}</span>
                </label>
              ) : null}
              {error ? (
                <div className="note b" role="alert">
                  <span>{error}</span>
                </div>
              ) : null}
              <div className="flex flex-wrap justify-end gap-2.5">
                <RDialog.Close className="btn min-h-11">{t(m.back)}</RDialog.Close>
                <Button variant="primary" className="min-h-11 px-5 text-[14px]" disabledReason={reason} loading={busy} onClick={() => void confirm()}>
                  {action}
                </Button>
              </div>
            </section>
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
