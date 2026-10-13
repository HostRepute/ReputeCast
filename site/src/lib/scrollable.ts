import { useEffect, useState, type RefObject } from 'react'

/**
 * True while the element's content is wider than its box, so the caller can
 * make it keyboard-focusable (a scrollable region must be reachable without
 * a mouse) only when it actually scrolls.
 */
export function useScrollable(ref: RefObject<HTMLElement | null>): boolean {
  const [scrollable, setScrollable] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const check = () => setScrollable(el.scrollWidth > el.clientWidth + 1)
    check()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(check)
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref])
  return scrollable
}
