import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

/**
 * Nothing matches a search or filter (Feedback board): say what was searched
 * where, why nothing came back, and the ways out (widen the search, clear the
 * filters, add the thing). Announced politely so screen reader users know
 * the search ran.
 */
export function NoResults({ title, children, actions, className }: { title: ReactNode; children?: ReactNode; actions?: ReactNode; className?: string }) {
  return (
    <section className={cn('card noresults', className)} role="status">
      <h2 className="nrt">{title}</h2>
      {children ? <span className="k max-w-[360px]">{children}</span> : null}
      {actions ? <div className="mt-1 flex flex-wrap justify-center gap-2">{actions}</div> : null}
    </section>
  )
}
