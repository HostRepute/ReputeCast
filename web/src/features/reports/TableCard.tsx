import type { ReactNode } from 'react'
import { TableScroll } from '@/components/ui/Table'
import { cn } from '@/lib/cn'

/**
 * A card with the add-on boards' section bar on top (its title, a caption
 * and, when given, controls after the caption), the table scrolling inside
 * it and notes underneath.
 */
export function TableCard({
  title,
  caption,
  actions,
  footer,
  className,
  children,
  id,
}: {
  title: string
  caption: string
  actions?: ReactNode
  footer?: ReactNode
  className?: string
  children: ReactNode
  id: string
}) {
  return (
    <section className={cn('card min-w-0', className)} aria-labelledby={id}>
      <div className="sec">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        {actions ? (
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className="k">{caption}</span>
            {actions}
          </div>
        ) : (
          <span className="k">{caption}</span>
        )}
      </div>
      <TableScroll label={title}>{children}</TableScroll>
      {footer ? <div className="k flex flex-col gap-1 border-t border-line-2 px-4.5 py-2.5">{footer}</div> : null}
    </section>
  )
}
