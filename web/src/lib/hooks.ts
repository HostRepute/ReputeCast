import { useEffect, useRef, useState, useSyncExternalStore, type RefObject } from 'react'

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
  return useOverflow(ref, false)
}

/** Whether an element scrolls sideways or, with `vertical`, up and down too. */
function useOverflow(ref: RefObject<HTMLElement | null>, vertical: boolean): boolean {
  const [over, setOver] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const check = () => setOver(el.scrollWidth > el.clientWidth + 1 || (vertical && el.scrollHeight > el.clientHeight + 1))
    check()
    // The box resizing, or its content changing (a longer query in a code
    // block, a new first child), can both change whether it scrolls.
    const ro = new ResizeObserver(check)
    ro.observe(el)
    if (el.firstElementChild) ro.observe(el.firstElementChild)
    const mo = new MutationObserver(check)
    // Style and class changes too: the composer sizes its canvas through them.
    mo.observe(el, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['style', 'class'] })
    return () => {
      ro.disconnect()
      mo.disconnect()
    }
  }, [ref, vertical])
  return over
}

/**
 * A scroller's ref and the props that make it reachable from the keyboard
 * while it scrolls (WCAG 2.1.1): a tab stop and, given a label, a named
 * region. A scroller that has a role already passes no label. It counts
 * sideways scrolling, and with `vertical` up and down too.
 */
export function useScrollRegion<T extends HTMLElement>(label?: string, vertical = false): [RefObject<T | null>, { tabIndex?: number; role?: 'region'; 'aria-label'?: string }] {
  const ref = useRef<T>(null)
  const scrolls = useOverflow(ref, vertical)
  return [ref, scrolls ? { tabIndex: 0, role: label ? 'region' : undefined, 'aria-label': label } : {}]
}
