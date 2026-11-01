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
 */
export function Stepper({ steps, current, label, foot }: { steps: readonly StepItem[]; current: number; label: string; foot?: ReactNode }) {
  const t = useT()
  const ref = useRef<HTMLElement>(null)
  const scrolls = useOverflowX(ref)
  return (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <nav ref={ref} className="card cn-steps" aria-label={label} tabIndex={scrolls ? 0 : undefined}>
      {steps.map((s, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        return (
          <div key={s.key} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
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
      {foot ? <div className="cn-foot k mt-2.5 border-t border-line-2 pt-3">{foot}</div> : null}
    </nav>
  )
}
