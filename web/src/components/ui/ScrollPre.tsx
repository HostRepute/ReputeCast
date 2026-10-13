import { useRef, type ReactNode } from 'react'
import { useOverflowX } from '@/lib/hooks'

/**
 * A code block that may scroll sideways. While it scrolls it takes keyboard
 * focus, so it can be read without a mouse (WCAG 2.1.1); `label` names it.
 */
export function ScrollPre({ className, label, children }: { className?: string; label?: string; children: ReactNode }) {
  const ref = useRef<HTMLPreElement>(null)
  const scrolls = useOverflowX(ref)
  return (
    // A scrolling region is the exception to the rule: it must take focus.
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <pre ref={ref} className={className} aria-label={label} tabIndex={scrolls ? 0 : undefined}>
      {children}
    </pre>
  )
}
