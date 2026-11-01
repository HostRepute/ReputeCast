import { useRef, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { useOverflowX } from '@/lib/hooks'
import { cn } from '@/lib/cn'
import { m } from './messages'

/** One step of a wizard: its name and the line under it (what was chosen, or what it's for). */
export interface StepItem {
  key: string
  title: ReactNode
  line?: ReactNode
}

/**
 * A wizard's steps, the one way to draw them (review 2026-11 FE-3b): the
 * setup boards' numbered card, the current step marked for screen readers
 * and finished ones read as "Done: …". On phones the steps are one
 * sideways-scrolling strip that takes keyboard focus while it scrolls
 * (WCAG 2.1.1). `current` counts from 1; `foot` is a line under the steps.
 * The rail's styles are `.stepper` in styles/app.css, so a page needs no
 * other stylesheet to draw it.
 */
export function Stepper({ steps, current, label, foot }: { steps: readonly StepItem[]; current: number; label: string; foot?: ReactNode }) {
  const t = useT()
  const ref = useRef<HTMLElement>(null)
  const scrolls = useOverflowX(ref)
  return (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <nav ref={ref} className="card stepper" aria-label={label} tabIndex={scrolls ? 0 : undefined}>
      {steps.map((s, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        return (
          <div key={s.key} className={cn('step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? <Icon name="check" size={14} /> : n}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">{t(m.stepDone)} </span> : null}
                {s.title}
              </div>
              {s.line ? <div className="k">{s.line}</div> : null}
            </div>
          </div>
        )
      })}
      {foot ? <div className="stepper-foot k mt-2.5 border-t border-line-2 pt-3">{foot}</div> : null}
    </nav>
  )
}

/**
 * A dialog's steps on one line, "✓ Store address › 2 Create the app › 3 …"
 * (review 2026-11 FE-3b): the compact form of Stepper, for wizards inside a
 * dialog. Finished steps read as "Done: …" and the current one is marked for
 * screen readers. `current` counts from 1. `blocked` dashes the steps that
 * can't be reached yet; `tone="soft"` is Earn's look (finished steps in the
 * accent, numbers on the ground colour). Styles: `.trail` in styles/app.css.
 */
export function StepTrail({
  steps,
  current,
  label,
  blocked,
  tone,
  className,
}: {
  steps: readonly Pick<StepItem, 'key' | 'title'>[]
  current: number
  label: string
  blocked?: boolean
  tone?: 'soft'
  className?: string
}) {
  const t = useT()
  return (
    <ol className={cn('trail', tone, className)} aria-label={label}>
      {steps.map((s, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        const off = blocked && n > current
        return (
          <li key={s.key} className={cn(done && 'done', cur && 'cur', off && 'off')} aria-current={cur ? 'step' : undefined}>
            <span className="n" aria-hidden="true">
              {done ? <Icon name="check" size={11} strokeWidth={2.4} /> : n}
            </span>
            <span aria-disabled={off || undefined}>
              {done ? <span className="vh">{t(m.stepDone)} </span> : null}
              {s.title}
            </span>
            {/* After the step, so a line that wraps ends on "›" rather than starting with one. */}
            {i < steps.length - 1 ? (
              <span className="sep" aria-hidden="true">
                ›
              </span>
            ) : null}
          </li>
        )
      })}
    </ol>
  )
}
