import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/cn'

type As = 'section' | 'article' | 'div' | 'aside'

export interface CardProps extends HTMLAttributes<HTMLElement> {
  as?: As
}

/** White surface with a 1px line border and a 10px radius. */
export function Card({ as: Tag = 'section', className, ...rest }: CardProps) {
  return <Tag className={cn('card', className)} {...rest} />
}

/**
 * Heading row of a card: one h2, an optional caption and one action group in
 * the top-right corner.
 */
export function CardHeader({
  title,
  id,
  caption,
  actions,
  className,
  level = 2,
}: {
  title: ReactNode
  id?: string
  caption?: ReactNode
  actions?: ReactNode
  className?: string
  level?: 2 | 3
}) {
  const H = level === 2 ? 'h2' : 'h3'
  return (
    <div className={cn('flex flex-wrap items-center justify-between gap-3', className)}>
      <div className="flex min-w-0 flex-col gap-0.5">
        <H className="h2" id={id}>
          {title}
        </H>
        {caption ? <span className="k">{caption}</span> : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}

/** The add-on screens' card section bar (`.sec`): heading left, actions right. */
export function CardSection({ title, id, caption, actions, className }: { title: ReactNode; id?: string; caption?: ReactNode; actions?: ReactNode; className?: string }) {
  return (
    <div className={cn('sec', className)}>
      <div className="flex min-w-0 flex-col gap-0.5">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        {caption ? <span className="k">{caption}</span> : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}
