import type { HTMLAttributes, ReactNode, TableHTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

/**
 * A data table that scrolls inside its card instead of the page. `density`
 * matches the two table styles in the prototype: `regular` (12px cells) and
 * `compact` (11px vertical, secondary cells kept on one line).
 *
 * At phone width, badges and buttons in cells stay on one line, as on the
 * team and access boards. The other boards (Campaigns, Audience, Providers,
 * Deliverability, Reports…) let them wrap like the rest of the page: pass
 * `phoneWrap` there.
 */
export function Table({
  className,
  density = 'regular',
  caption,
  children,
  scroll = true,
  phoneWrap = false,
  ...rest
}: TableHTMLAttributes<HTMLTableElement> & {
  density?: 'regular' | 'compact' | 'dense'
  caption?: string
  scroll?: boolean
  phoneWrap?: boolean
}) {
  const table = (
    <table
      className={cn('tbl', density === 'compact' && 'compact', density === 'dense' && 'dense', phoneWrap && 'phone-wrap', className)}
      {...rest}
    >
      {caption ? <caption className="vh">{caption}</caption> : null}
      {children}
    </table>
  )
  return scroll ? <div className="tscroll">{table}</div> : table
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
