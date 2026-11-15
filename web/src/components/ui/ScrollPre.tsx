import { type ReactNode } from 'react'
import { useScrollRegion } from '@/lib/hooks'

/**
 * A code block that may scroll. While it scrolls it takes keyboard
 * focus, so it can be read without a mouse (WCAG 2.1.1); `label` names it.
 */
export function ScrollPre({ className, label, children, vertical = false }: { className?: string; label?: string; children: ReactNode; vertical?: boolean }) {
  const [ref, scrollProps] = useScrollRegion<HTMLPreElement>(label, vertical)
  return (
    <pre ref={ref} className={className} {...scrollProps}>
      {children}
    </pre>
  )
}
