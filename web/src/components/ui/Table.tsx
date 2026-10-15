import { useRef, type HTMLAttributes, type ReactNode, type TableHTMLAttributes } from 'react'
import { useOverflowX } from '@/lib/hooks'
import { cn } from '@/lib/cn'

/**
 * A data table that scrolls inside its card instead of the page. `density`
 * matches the prototype's two table styles: `regular` (12px cells) and
 * `compact` (11px vertical, secondary cells kept on one line).
 *
 * At phone width, badges and buttons in cells stay on one line, as on the
 * team and access boards; pass `phoneWrap` to let them wrap like the rest of
 * the page, as on the other boards. `fit` lets every cell's text wrap (badges
 * and buttons stay whole) so the columns fit the card before it scrolls.
 */
export function Table({
  className,
  density = 'regular',
  caption,
  children,
  scroll = true,
  phoneWrap = false,
  fit = false,
  ...rest
}: TableHTMLAttributes<HTMLTableElement> & {
  density?: 'regular' | 'compact' | 'dense'
  caption?: string
  scroll?: boolean
  phoneWrap?: boolean
  fit?: boolean
}) {
  const table = (
    <table
      className={cn('tbl', density === 'compact' && 'compact', density === 'dense' && 'dense', phoneWrap && 'phone-wrap', fit && 'fit', className)}
      {...rest}
    >
      {caption ? <caption className="vh">{caption}</caption> : null}
      {children}
    </table>
  )
  return scroll ? <TableScroll label={caption}>{table}</TableScroll> : table
}

/**
 * The sideways scroller around a wide table. While the table overflows it
 * takes keyboard focus, so it can be scrolled without a mouse, and `label`
 * names it for screen readers.
 */
export function TableScroll({ label, className, children }: { label?: string; className?: string; children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)
  const scrolls = useOverflowX(ref)
  return (
    <div
      ref={ref}
      className={cn('tscroll', className)}
      tabIndex={scrolls ? 0 : undefined}
      role={scrolls && label ? 'region' : undefined}
      aria-label={scrolls ? label : undefined}
    >
      {children}
    </div>
  )
}

/** Column header cells with scope="col". */
export function Th({ className, children, align, ...rest }: HTMLAttributes<HTMLTableCellElement> & { align?: 'left' | 'center' | 'right' }) {
  return (
    <th scope="col" className={cn(align === 'center' && 'c', align === 'right' && 'r', className)} {...rest}>
      {children}
    </th>
  )
}

export function Td({
  className,
  children,
  align,
  num,
  wrap,
  nowrap,
  ...rest
}: HTMLAttributes<HTMLTableCellElement> & { align?: 'left' | 'center' | 'right'; num?: boolean; wrap?: boolean; nowrap?: boolean; colSpan?: number }) {
  return (
    <td className={cn(num && 'n', align === 'center' && 'c', align === 'right' && 'r', wrap && 'wrap', nowrap && 'nw', className)} {...rest}>
      {children}
    </td>
  )
}

/** Empty row spanning the table, for filters that match nothing. */
export function EmptyRow({ colSpan, children }: { colSpan: number; children: ReactNode }) {
  return (
    <tr>
      <td colSpan={colSpan} className="py-8 text-center text-ink-3">
        {children}
      </td>
    </tr>
  )
}
