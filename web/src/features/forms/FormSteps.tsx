import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { m } from './builder.messages'
import { addStep, canMove, maxSteps, moveField, removeStep, severalSteps, splitSteps, type FormField, type FormStep } from './steps'

const up = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M8 13V3M4 7l4-4 4 4" />
  </svg>
)
const down = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M8 3v10M4 9l4 4 4-4" />
  </svg>
)

/**
 * "Steps" (FormBuilder, FormsStates "Several steps"): one step, or up to
 * four with their own heading, description and button; every field sits
 * in one step and moves between them with ↑ ↓. Selecting a step's card
 * shows that step in the preview. With an A/B test's variant selected, it
 * edits that variant's steps.
 */
export function StepsSection({
  steps,
  fields,
  nameOf,
  reason,
  current,
  button,
  error,
  onPick,
  onChange,
}: {
  /** The steps shown (none: one step), in line with the fields. */
  steps: FormStep[] | undefined
  fields: FormField[]
  nameOf: (f: FormField) => string
  /** Why the person can't edit the form, if they can't. */
  reason?: string
  /** The step the preview shows. */
  current: number
  /** The form's (or variant's) button: the last step's until it has its own. */
  button: string
  error?: string
  onPick: (index: number) => void
  onChange: (steps: FormStep[] | undefined) => void
}) {
  const t = useT()
  const readOnly = Boolean(reason)
  const several = severalSteps(steps)
  const pickMode = (on: boolean) => {
    if (readOnly || on === several) return
    onChange(on ? splitSteps(fields.map((f) => f.key)) : undefined)
    onPick(0)
  }
  const edit = (index: number, patch: Partial<FormStep>) => steps && onChange(steps.map((s, i) => (i === index ? { ...s, ...patch } : s)))
  return (
    <div className="fb-sec">
      <span className="k">{t(m.steps)}</span>
      <div className="fb-seg" role="group" aria-label={t(m.steps)}>
        <button type="button" aria-pressed={!several} aria-disabled={readOnly || undefined} onClick={() => pickMode(false)}>
          {t(m.oneStep)}
        </button>
        <button type="button" aria-pressed={several} aria-disabled={readOnly || undefined} onClick={() => pickMode(true)}>
          {t(m.severalSteps)}
        </button>
      </div>
      {several ? (
        <>
          <span className="k">{t(m.severalStepsHelp)}</span>
          {steps.map((s, si) => {
            const n = si + 1
            const on = si === Math.min(current, steps.length - 1)
            const last = si === steps.length - 1
            return (
              <div
                key={s.id}
                role="group"
                aria-label={on ? t(m.stepShown, { n }) : t(m.stepN, { n })}
                aria-current={on || undefined}
                className={cn('card flex flex-col gap-2 px-3 py-2.5', on && 'border-accent')}
                onFocus={() => onPick(si)}
                onClick={() => onPick(si)}
              >
                <div className="flex items-center justify-between gap-2.5">
                  <span className="font-semibold">{t(m.stepN, { n })}</span>
                  {steps.length > 2 ? (
                    <Button
                      size="sm"
                      aria-label={t(m.removeStep, { n })}
                      disabledReason={reason}
                      onClick={(e) => {
                        // The card would pick the step that goes.
                        e.stopPropagation()
                        onChange(removeStep(steps, si))
                        onPick(Math.max(0, si - 1))
                      }}
                    >
                      {t(m.remove)}
                    </Button>
                  ) : null}
                </div>
                <label className="lbl">
                  {t(m.stepHeading)}
                  <input className="fb-in" maxLength={120} value={s.heading ?? ''} disabled={readOnly} onChange={(e) => edit(si, { heading: e.target.value })} />
                </label>
                <label className="lbl">
                  {t(m.stepDescription)}
                  <input className="fb-in" maxLength={500} value={s.description ?? ''} disabled={readOnly} onChange={(e) => edit(si, { description: e.target.value })} />
                </label>
                <label className="lbl">
                  {t(m.stepButton)}
                  <input className="fb-in" maxLength={40} placeholder={last ? button : t(common.next)} value={s.button ?? ''} disabled={readOnly} onChange={(e) => edit(si, { button: e.target.value })} />
                </label>
                {s.fields.map((key, fi) => {
                  const f = fields.find((x) => x.key === key)
                  if (!f) return null
                  const field = nameOf(f)
                  return (
                    <div key={key} className="flex items-center gap-2 text-[13px]">
                      <span className="min-w-0 flex-auto">{t(f.required || f.key === 'email' ? m.fieldRequired : m.fieldOptional, { field })}</span>
                      <Button
                        size="sm"
                        iconOnly
                        className="h-8 min-h-8 w-8 p-0 max-md:h-11 max-md:min-h-11 max-md:w-11"
                        aria-label={t(m.moveUp, { field })}
                        disabled={!canMove(steps, si, fi, -1)}
                        disabledReason={reason}
                        onClick={() => onChange(moveField(steps, si, fi, -1))}
                      >
                        {up}
                      </Button>
                      <Button
                        size="sm"
                        iconOnly
                        className="h-8 min-h-8 w-8 p-0 max-md:h-11 max-md:min-h-11 max-md:w-11"
                        aria-label={t(m.moveDown, { field })}
                        disabled={!canMove(steps, si, fi, 1)}
                        disabledReason={reason}
                        onClick={() => onChange(moveField(steps, si, fi, 1))}
                      >
                        {down}
                      </Button>
                    </div>
                  )
                })}
                {last ? <span className="k">{t(m.consentOnStep)}</span> : null}
              </div>
            )
          })}
          <Button
            size="sm"
            className="self-start"
            disabledReason={reason ?? (steps.length >= maxSteps ? t(m.maxSteps) : undefined)}
            onClick={() => {
              onChange(addStep(steps))
              onPick(steps.length)
            }}
          >
            {t(m.addStep)}
          </Button>
        </>
      ) : (
        <span className="k">{t(m.oneStepHelp)}</span>
      )}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  )
}
