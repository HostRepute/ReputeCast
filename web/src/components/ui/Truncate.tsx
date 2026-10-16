import { cloneElement, useLayoutEffect, useState, type CSSProperties, type ReactElement, type Ref } from 'react'
import { cn } from '@/lib/cn'
import { Tooltip } from './Menu'

type Truncatable = ReactElement<{ className?: string; style?: CSSProperties; ref?: Ref<HTMLElement> }>

/**
 * User content in a table cell (campaign, list and member names, email
 * addresses) on one line that ends in an ellipsis when it doesn't fit, with
 * the full text on hover and on keyboard focus (text expansion rule 2,
 * docs/dev/CONVENTIONS.md). Wrap the cell's link to keep it the focus
 * target, or pass only the text:
 *
 *   <Truncate text={c.name}><Link …>{c.name}</Link></Truncate>
 *   <Truncate text={m.email} />
 *
 * `max` is the widest the text gets before it is cut (220 px by default).
 * The text stays whole in the DOM, so screen readers read all of it.
 */
export function Truncate({ text, children, max, className }: { text: string; children?: Truncatable; max?: number; className?: string }) {
  const [node, setNode] = useState<HTMLElement | null>(null)
  const [cut, setCut] = useState(false)
  useLayoutEffect(() => {
    if (!node) return
    const check = () => setCut(node.scrollWidth > node.clientWidth + 1)
    check()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(check)
    ro.observe(node)
    return () => ro.disconnect()
  }, [node, text])
  const child: Truncatable = children ?? <span>{text}</span>
  const style = max ? ({ ...child.props.style, '--ellipsis-max': `${max}px` } as CSSProperties) : child.props.style
  const el = cloneElement(child, { ref: setNode, className: cn(child.props.className, 'ellipsis', className), style })
  return cut ? (
    <Tooltip content={text} side="top">
      {el}
    </Tooltip>
  ) : (
    el
  )
}
