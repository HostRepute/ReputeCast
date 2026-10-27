import { Dialog as RDialog } from 'radix-ui'
import { useId, useState } from 'react'
import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { clock, count, money } from '@/lib/format'
import { confirmLanguages } from '@/features/languages/review'
import { mr } from '@/features/languages/messages'
import { formatList, useT } from '@/i18n'
import { acknowledgedKeys, acknowledgements, minutes, scheduleSentence, thresholdsLine, verdictTitle, type Check, type Fix, type Review, type Run } from './logic'
import { m } from './messages'

export type Canary = 'auto' | 'on' | 'off'

export type SendIntent = { acknowledged: string[]; confirm?: string; canary: Canary }

/**
 * The deliberate last step before a send (SendReadinessConfirm): what goes
 * out, to whom and how; each warning confirmed, with its one-click fix as
 * the alternative; the recipient count typed out only for an unusual send
 * (SR-13), and the canary choice (C1).
 */
export function ConfirmSend({
  open,
  onOpenChange,
  review,
  run,
  action,
  onConfirm,
  onFix,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  review: Review
  run: Run
  /** "Schedule send", "Send now", "Schedule as approved". */
  action: string
  onConfirm: (intent: SendIntent) => Promise<void>
  /** "Send via SES only instead": applies a warning's fix and closes. */
  onFix?: (check: Check, fix: Fix) => void
}) {
  const id = useId()
  const t = useT()
  const languages = confirmLanguages(run.checks, t)
  const c = review.campaign
  const recipients = run.estimates.audience.final
  const acks = acknowledgements(run)
  // Only an unusual send asks for the count (the server says why).
  const typedNeeded = review.typed.needed
  const [checked, setChecked] = useState<string[]>([])
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  // The canary phase (C1): the engine decides, or the person asks for one or against one.
  const [canary, setCanary] = useState<Canary>(review.canary.choice)
  const schedule = review.schedule
  const route = run.estimates.route
  const shares = route.shares.filter((s) => s.recipients > 0)
  const a = run.estimates.audience
  const missing = acks.filter((x) => !checked.includes(x.key)).length
  const countOK = !typedNeeded || typed.replace(/\D/g, '') === String(recipients)
  const verb = schedule.mode === 'now' ? 'send' : 'schedule'
  const reason = missing
    ? acks.length > 1 && missing === acks.length
      ? t(m.ackBoth, { count: acks.length, action: verb })
      : t(m.confirmWarnings, { count: missing })
    : !countOK
      ? t(m.typeCount, { count: String(recipients) })
      : undefined
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
      await onConfirm({ acknowledged: acknowledgedKeys(acks, checked), confirm: typedNeeded ? typed : undefined, canary })
      setBusy(false)
      close(false)
    } catch (e) {
      setBusy(false)
      setError(e instanceof ApiError ? e.message : t(m.unreachable))
    }
  }
  const title = schedule.mode === 'now' ? t(m.confirmTitleNow, { name: c.name, recipients }) : t(m.confirmTitleAt, { name: c.name, recipients })
  const at = schedule.at ? new Date(schedule.at) : undefined
  const recheck = at ? clock(new Date(at.getTime() - 15 * 60_000)) : ''
  const when =
    schedule.mode === 'now'
      ? t(m.whenNow)
      : schedule.mode === 'best_time'
        ? t(m.whenBest, { when: scheduleSentence(schedule), hasRecheck: recheck ? 'yes' : 'no', recheck })
        : t(m.whenAt, { when: scheduleSentence(schedule), hasRecheck: recheck ? 'yes' : 'no', recheck })
  const excluded = [
    a.suppressed ? t(m.excludedSuppressed, { count: count(a.suppressed) }) : '',
    a.role ? t(m.excludedRole, { count: count(a.role) }) : '',
    a.duplicates ? t(m.excludedDuplicates, { count: count(a.duplicates) }) : '',
    a.excludedUnengaged ? t(m.excludedUnengaged, { count: count(a.excludedUnengaged) }) : '',
  ]
    .filter(Boolean)
    .join(' · ')
  // "$2.81 · winner 11:00 · done about 11:31" when the engine's timeline says when.
  const tl = route.timeline
  const winner = tl?.ab?.decideAt
  const timing =
    tl && winner && !tl.open
      ? t(m.costTiming, { cost: money(route.totalCost, 2), winner: clock(winner), end: clock(tl.end) })
      : t(m.costValue, { cost: money(route.totalCost, 2), duration: minutes(route.totalMinutes) })
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
    [tl && winner && !tl.open ? t(m.rowCostTiming) : t(m.rowCost), timing],
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
              <section className="c-acks" aria-labelledby={`${id}-acks`}>
                <h3 className="h3" id={`${id}-acks`}>
                  {t(m.confirmTheWarnings, { count: acks.length })}
                </h3>
                {acks.map((x) => (
                  <label key={x.key} className="rv-chk">
                    <input
                      type="checkbox"
                      checked={checked.includes(x.key)}
                      onChange={(e) => setChecked((v) => (e.target.checked ? [...v, x.key] : v.filter((k) => k !== x.key)))}
                    />
                    <span>
                      {x.text}
                      {x.fix && x.check && onFix ? (
                        <>
                          {' '}
                          <a
                            href="#fix"
                            onClick={(e) => {
                              e.preventDefault()
                              close(false)
                              onFix(x.check!, x.fix!)
                            }}
                          >
                            {t(m.fixInstead, { fix: x.fix.label })}
                          </a>
                        </>
                      ) : null}
                    </span>
                  </label>
                ))}
              </section>
            ) : null}
            <section className="c-last">
              {typedNeeded ? (
                <label className="rv-typed">
                  <span className="flex flex-col gap-0.5">
                    <span>{t(m.typeToConfirm)}</span>
                    {review.typed.reasons.length ? <span className="k font-normal">{t(m.typedWhy, { reasons: formatList(review.typed.reasons) })}</span> : null}
                  </span>
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
              ) : (
                <span className="k">{t(m.typedOnlyUnusual)}</span>
              )}
              {error ? (
                <div className="note b" role="alert">
                  <span>{error}</span>
                </div>
              ) : null}
              <div className="flex flex-wrap items-center justify-end gap-2.5">
                {reason ? <span className="k mr-auto">{reason}</span> : null}
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
