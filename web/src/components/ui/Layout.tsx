import type { CSSProperties, HTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

/**
 * Layout primitives reproducing the prototype's responsive rules without
 * attribute selectors:
 *  - Row: flex row that wraps on phones (unless nowrap).
 *  - Grid min: auto-fit columns of at least `min` px, one column on phones
 *    when min ≥ 260px.
 *  - Grid cols: N equal columns that become auto-fit 140px columns on phones.
 */
export function Row({ className, gap = 12, nowrap, align = 'center', justify, wrap, style, ...rest }: HTMLAttributes<HTMLDivElement> & { gap?: number | string; nowrap?: boolean; wrap?: boolean; align?: CSSProperties['alignItems']; justify?: CSSProperties['justifyContent'] }) {
  return (
    <div
      className={cn('flex', wrap ? 'flex-wrap' : !nowrap && 'max-md:flex-wrap', className)}
      style={{ gap, alignItems: align, justifyContent: justify, ...style }}
      {...rest}
    />
  )
}

export function Stack({ className, gap = 12, style, ...rest }: HTMLAttributes<HTMLDivElement> & { gap?: number | string }) {
  return <div className={cn('flex flex-col', className)} style={{ gap, ...style }} {...rest} />
}

type GridProps = HTMLAttributes<HTMLDivElement> & { gap?: number | string; min?: number; cols?: number }

export function Grid({ className, gap = 16, min, cols, style, ...rest }: GridProps) {
  const template = min ? `repeat(auto-fit, minmax(min(${min}px, 100%), 1fr))` : `repeat(${cols ?? 2}, minmax(0, 1fr))`
  return (
    <div
      className={cn('grid [&>*]:min-w-0', min && min >= 260 && 'max-md:grid-cols-1!', !min && 'max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!', className)}
      style={{ gap, gridTemplateColumns: template, ...style }}
      {...rest}
    />
  )
}
