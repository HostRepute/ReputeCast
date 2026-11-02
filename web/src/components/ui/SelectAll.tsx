import type { InputHTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

/**
 * A checkbox that stands alone, named by aria-label (a row's pick, every
 * row's): its label is the target, 24px with a mouse and 44px on touch
 * widths around the 16px box, which keeps its place (design system,
 * Targets). className goes on the label, which takes the box's place.
 */
export function Check({ className, ...rest }: Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>) {
  return (
    <label className={cn('hit', className)}>
      <input type="checkbox" {...rest} />
    </label>
  )
}

/**
 * The checkbox that picks every row shown, or none; mixed while some are
 * picked (review 2026-11 FE-3b). Goes with useSelection (lib/selection).
 */
export function SelectAll({ all, some, label, onChange, className }: { all: boolean; some: boolean; label: string; onChange: (on: boolean) => void; className?: string }) {
  return (
    <label className={cn('hit', className)}>
      <input
        type="checkbox"
        checked={all}
        ref={(el) => {
          if (el) el.indeterminate = some && !all
        }}
        aria-label={label}
        onChange={(e) => onChange(e.target.checked)}
      />
    </label>
  )
}
