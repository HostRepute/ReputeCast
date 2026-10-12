import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { Icon, type IconName } from '../icons/Icon'

export type Tone = 'ok' | 'warn' | 'bad' | 'info' | 'neutral' | 'purple'

const toneClass: Record<Tone, string> = { ok: 'ok', warn: 'warn', bad: 'bad', info: 'info', neutral: 'mut', purple: 'pur' }

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: Tone
  /** Leading dot in the badge's colour. */
  dot?: boolean
  /** Pulsing dot for live problems (auto-paused). */
  pulse?: boolean
  icon?: IconName
  size?: 'sm' | 'md'
  children: ReactNode
}

/** A state word (Healthy, Warming up, Blocked). Never colour alone. */
export function Badge({ tone = 'neutral', dot, pulse, icon, size, className, children, ...rest }: BadgeProps) {
  return (
    <span className={cn('badge', toneClass[tone], size === 'sm' && 'sm', className)} {...rest}>
      {dot || pulse ? <span className={cn('dot', pulse && 'pulse-bad')} aria-hidden="true" /> : null}
      {icon ? <Icon name={icon} size={12} strokeWidth={2.2} /> : null}
      {children}
    </span>
  )
}

/** A plain coloured dot, e.g. the series colour beside a provider name. */
export function Dot({ color, className, label }: { color: string; className?: string; label?: string }) {
  return <span className={cn('dot', className)} style={{ color }} aria-hidden={label ? undefined : true} aria-label={label} role={label ? 'img' : undefined} />
}

/** The ⚑ mark on sensitive permissions. */
export function SensitiveFlag({ children = 'sensitive' }: { children?: ReactNode }) {
  return (
    <span className="flag">
      <span aria-hidden="true">⚑</span>
      {children}
    </span>
  )
}
