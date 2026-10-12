import type { CSSProperties, ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { Icon } from '../icons/Icon'

/** Grey block shaped like the content it stands in for. */
export function Skeleton({ w, h = 12, r, className, style }: { w?: number | string; h?: number | string; r?: number; className?: string; style?: CSSProperties }) {
  return <span className={cn('sk shimmer', className)} style={{ width: w, height: h, borderRadius: r, ...style }} aria-hidden="true" />
}

/** Loading region: hidden skeleton visuals plus one announced sentence. */
export function Loading({ label, children, className }: { label: string; children: ReactNode; className?: string }) {
  return (
    <div className={className} aria-busy="true">
      <span className="vh" role="status">
        {label}
      </span>
      {children}
    </div>
  )
}

export function Spinner({ size = 16, onDark, label }: { size?: 14 | 16; onDark?: boolean; label?: string }) {
  return (
    <span
      className={cn('spin', size === 14 && 'sm', onDark && 'on-dark')}
      role={label ? 'status' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    />
  )
}

/**
 * An empty state says what will appear here and offers the one action that
 * makes it appear.
 */
export function EmptyState({
  title,
  children,
  action,
  icon,
  className,
}: {
  title: ReactNode
  children?: ReactNode
  action?: ReactNode
  icon?: ReactNode
  className?: string
}) {
  return (
    <section className={cn('card flex flex-col items-center gap-3 px-6 py-10 text-center', className)}>
      {icon}
      <h2 className="m-0 text-[17px] font-semibold">{title}</h2>
      {children ? <div className="max-w-[560px] text-ink-2">{children}</div> : null}
      {action ? <div className="mt-1 flex flex-wrap justify-center gap-2.5">{action}</div> : null}
    </section>
  )
}

/**
 * An error says what failed, what ReputeCast already did about it, and what
 * the person can do next.
 */
export function ErrorPanel({
  title,
  children,
  actions,
  className,
  reference,
}: {
  title: ReactNode
  children?: ReactNode
  actions?: ReactNode
  className?: string
  reference?: string
}) {
  return (
    <section className={cn('flex items-start gap-3 rounded-[10px] border border-bad-line bg-bad-tint px-5 py-4', className)} role="alert">
      <span className="mt-0.5 text-bad-strong">
        <Icon name="alertCircle" size={18} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <h2 className="m-0 text-[15px] font-semibold text-bad-fg">{title}</h2>
        {children ? <div className="text-[13px] text-ink-2">{children}</div> : null}
        {reference ? <span className="k">Reference <span className="mono">{reference}</span></span> : null}
        {actions ? <div className="mt-1.5 flex flex-wrap gap-2">{actions}</div> : null}
      </div>
    </section>
  )
}
