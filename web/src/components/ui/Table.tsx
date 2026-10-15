import { useLayoutEffect, useRef, useState, type CSSProperties, type HTMLAttributes, type ReactNode, type TableHTMLAttributes } from 'react'
import { useOverflowX } from '@/lib/hooks'
import { cn } from '@/lib/cn'

/**
 * A data table that scrolls inside its card instead of the page. `density`
 * matches the prototype's two table styles: `regular` (12px cells) and
 * `compact` (11px vertical, secondary cells kept on one line).
 *
 * At phone width, badges and buttons in cells stay on one line, as on the
 * team and access boards; pass `phoneWrap` to let them wrap like the rest of
 * the page, as on the other boards.
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

/**
 * The longer line of the best split of a text into two lines at a space,
 * given a way to measure a line: "Lượt mở bởi người đọc" → the width of
 * "Lượt mở bởi". Undefined for a single word, which never wraps.
 */
export function twoLineWidth(text: string, measure: (line: string) => number): number | undefined {
  const words = text.trim().split(/\s+/)
  if (words.length < 2) return undefined
  let best = Infinity
  for (let i = 1; i < words.length; i++) {
    best = Math.min(best, Math.max(measure(words.slice(0, i).join(' ')), measure(words.slice(i).join(' '))))
  }
  return best
}

let canvas: HTMLCanvasElement | undefined

/** Measures a header cell's text with its own font, case and letter spacing. */
function headerLineWidth(th: HTMLElement, text: string): number | undefined {
  canvas ??= document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) return undefined
  const cs = getComputedStyle(th)
  ctx.font = `${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`
  if ('letterSpacing' in ctx) ctx.letterSpacing = cs.letterSpacing
  const shown = cs.textTransform === 'uppercase' ? text.toLocaleUpperCase(document.documentElement.lang || undefined) : text
  const w = twoLineWidth(shown, (line) => ctx.measureText(line).width)
  return w === undefined ? undefined : Math.ceil(w) + 1
}

/**
 * Column header cells with scope="col". When the app isn't in English, a
 * text header of several words measures itself on two lines (--th-line),
 * so a translated header takes at most two lines and its column doesn't
 * squeeze its numbers (text expansion rule 3, styles/expansion.css).
 */
export function Th({ className, children, align, style, ...rest }: HTMLAttributes<HTMLTableCellElement> & { align?: 'left' | 'center' | 'right' }) {
  const text = typeof children === 'string' ? children : undefined
  const [node, setNode] = useState<HTMLTableCellElement | null>(null)
  const [line, setLine] = useState<number>()
  useLayoutEffect(() => {
    if (!node || !text || document.documentElement.lang === 'en') return
    let live = true
    const measure = () => live && setLine(headerLineWidth(node, text))
    measure()
    // The web font may still be loading: measure again once it is in.
    void document.fonts?.ready.then(measure)
    return () => {
      live = false
    }
  }, [node, text])
  return (
    <th
      ref={setNode}
      scope="col"
      className={cn(align === 'center' && 'c', align === 'right' && 'r', className)}
      style={line ? ({ ...style, '--th-line': `${line}px` } as CSSProperties) : style}
      {...rest}
    >
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
