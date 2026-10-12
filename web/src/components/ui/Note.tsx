import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { Badge, type Tone } from './Badge'

export type NoteTone = 'info' | 'warn' | 'bad' | 'ok' | 'neutral'

const noteClass: Record<NoteTone, string> = { info: 'i', warn: 'w', bad: 'b', ok: 'g', neutral: 'n' }
const badgeTone: Record<NoteTone, Tone> = { info: 'info', warn: 'warn', bad: 'bad', ok: 'ok', neutral: 'neutral' }

export interface NoteProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  tone?: NoteTone
  /** Optional state word shown as a badge before the text. */
  label?: ReactNode
  children: ReactNode
}

/**
 * Inline explanation: info, warning, problem, success or neutral. Problems
 * are announced (role=alert); the others are polite status messages.
 */
export function Note({ tone = 'info', label, className, children, role, ...rest }: NoteProps) {
  return (
    <div className={cn('note', noteClass[tone], className)} role={role ?? (tone === 'bad' ? 'alert' : 'status')} {...rest}>
      {label ? <Badge tone={badgeTone[tone]}>{label}</Badge> : null}
      <div>{children}</div>
    </div>
  )
}

/**
 * Full-width banner above a screen's content: an active pause or outage
 * (bad), a warning (warn) or a scheduled/informational notice (info). A
 * banner always has an action.
 */
export function Banner({
  tone,
  title,
  children,
  actions,
  className,
}: {
  tone: 'bad' | 'warn' | 'info' | 'ok'
  title: ReactNode
  children?: ReactNode
  actions?: ReactNode
  className?: string
}) {
  const styles = {
    bad: 'border-bad-line bg-bad-tint',
    warn: 'border-[#FDE68A] bg-[#FFFBEB]',
    info: 'border-[#BFDBFE] bg-[#EFF6FF]',
    ok: 'border-ok-line bg-ok-tint',
  }[tone]
  return (
    <section className={cn('flex flex-wrap items-center gap-x-4 gap-y-3 rounded-[10px] border px-5 py-4', styles, className)} role={tone === 'bad' ? 'alert' : 'status'}>
      <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-1">
        <div className="text-[15px] font-semibold">{title}</div>
        {children ? <div className="text-[13px] text-ink-2">{children}</div> : null}
      </div>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </section>
  )
}
