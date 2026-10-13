import { useEffect, useState, useSyncExternalStore, type RefObject } from 'react'

/** Subscribes to a CSS media query. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (cb) => {
      const m = window.matchMedia(query)
      m.addEventListener('change', cb)
      return () => m.removeEventListener('change', cb)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}

/** Sets document.title while mounted. */
export function useDocumentTitle(title: string) {
  useEffect(() => {
    const prev = document.title
    document.title = title
    return () => {
      document.title = prev
    }
  }, [title])
}

/** Re-renders every `ms` so relative times ("2 min ago") stay fresh. */
export function useNow(ms = 30_000): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const h = window.setInterval(() => setNow(new Date()), ms)
    return () => window.clearInterval(h)
  }, [ms])
  return now
}

/** True when the user asked the OS to reduce motion. */
export function useReducedMotion(): boolean {
  return useMediaQuery('(prefers-reduced-motion: reduce)')
}

/**
 * Whether an element scrolls sideways (its content is wider than it). A
 * scrolling region must take keyboard focus (WCAG 2.1.1), but only then:
 * a table that fits needs no extra tab stop.
 */
export function useOverflowX(ref: RefObject<HTMLElement | null>): boolean {
  const [over, setOver] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const check = () => setOver(el.scrollWidth > el.clientWidth + 1)
    check()
    const ro = new ResizeObserver(check)
    ro.observe(el)
    if (el.firstElementChild) ro.observe(el.firstElementChild)
    return () => ro.disconnect()
  }, [ref])
  return over
}
