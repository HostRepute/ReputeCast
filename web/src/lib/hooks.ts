import { notifyManager, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useRef, useState, useSyncExternalStore, type RefObject } from 'react'

const mediaLists = new Map<string, MediaQueryList>()
let mediaListsOf: typeof window.matchMedia | undefined

// One list per query: React reads the snapshot on every render, and each
// matchMedia call parses the query again. A replaced matchMedia (tests that
// emulate a phone) starts over.
function mediaList(query: string): MediaQueryList {
  if (mediaListsOf !== window.matchMedia) {
    mediaLists.clear()
    mediaListsOf = window.matchMedia
  }
  let m = mediaLists.get(query)
  if (!m) {
    m = window.matchMedia(query)
    mediaLists.set(query, m)
  }
  return m
}

export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (cb) => {
      const m = mediaList(query)
      m.addEventListener('change', cb)
      return () => m.removeEventListener('change', cb)
    },
    () => mediaList(query).matches,
    () => false,
  )
}

/**
 * True once `open` has been true. A dialog that is closed until someone
 * opens it renders nothing before then (its content's text and fields are
 * work on every screen's first render), and stays mounted afterwards, so
 * it closes with its animation.
 */
export function useOpenedOnce(open: boolean): boolean {
  const [opened, setOpened] = useState(open)
  if (open && !opened) setOpened(true)
  return opened || open
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
    // The box resizing, or its content changing (a longer query in a code
    // block, a new first child), can both change whether it scrolls. The
    // observer's first report comes once the box has been laid out, and a
    // change of content is measured once per frame: measuring as soon as the
    // DOM changed made the browser lay out the page in the middle of React's
    // work, once per table.
    const ro = new ResizeObserver(check)
    ro.observe(el)
    if (el.firstElementChild) ro.observe(el.firstElementChild)
    let frame = 0
    const mo = new MutationObserver(() => {
      if (!frame) {
        frame = requestAnimationFrame(() => {
          frame = 0
          check()
        })
      }
    })
    // Style and class changes too: the composer sizes its canvas through them.
    mo.observe(el, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['style', 'class'] })
    return () => {
      ro.disconnect()
      mo.disconnect()
      cancelAnimationFrame(frame)
    }
  }, [ref, vertical])
  return over
}

/**
 * A scroller's ref and the props that make it reachable from the keyboard
 * while it scrolls (WCAG 2.1.1): a tab stop and, given a label, a named
 * region. A scroller that has a role already passes no label. It counts
 * sideways scrolling, and with `vertical` up and down too. With `always`,
 * it is a region from the start, for a scroller whose content arrives
 * later and nearly always overflows (the composer's canvas and its frame).
 */
export function useScrollRegion<T extends HTMLElement>(label?: string, vertical = false, always = false): [RefObject<T | null>, { tabIndex?: number; role?: 'region'; 'aria-label'?: string }] {
  const ref = useRef<T>(null)
  const scrolls = useOverflow(ref, vertical)
  return [ref, scrolls || always ? { tabIndex: 0, role: label ? 'region' : undefined, 'aria-label': label } : {}]
}

/** value, once it has stayed the same for ms (a search box's query). */
export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms)
    return () => clearTimeout(id)
  }, [value, ms])
  return v
}

/**
 * A value changed by the person and saved in the background (review 2026-11
 * FE-7b): it shows the new value at once, keeps showing it while the save
 * runs and until the server's value catches up, and goes back when the save
 * fails. `save` answers whether it was saved; a failing save says why
 * itself (a toast).
 */
export function useOptimistic<T>(server: T): [T, (next: T, save: () => Promise<boolean>) => Promise<boolean>] {
  const [shown, setShown] = useState<{ value: T }>()
  // A new value from the server replaces what was shown, whatever it is
  // (React's "adjust state when a prop changes" pattern, no effect).
  const [seen, setSeen] = useState(server)
  if (!Object.is(seen, server)) {
    setSeen(server)
    setShown(undefined)
  }
  const value = shown ? shown.value : server
  const change = async (next: T, save: () => Promise<boolean>) => {
    setShown({ value: next })
    const ok = await save().catch(() => false)
    if (!ok) setShown(undefined)
    return ok
  }
  return [value, change]
}

/** A query loading for the first time: nothing to show yet. */
const firstLoad = { predicate: (q: { state: { status: string } }) => q.state.status === 'pending' }

/**
 * False until the queries loading for the first time when a screen mounts
 * have come back, or `maxMs` has passed. A screen whose parts each appear when
 * their own data arrives (a filter bar that grows a row per feature) can keep
 * them hidden but mounted until then, so their queries run, and show them at
 * once instead of letting each push the rest down. Queries that already have
 * data, refetching in the background, don't hold it.
 */
export function useFirstLoadSettled(maxMs = 3000): boolean {
  const client = useQueryClient()
  const [state, setState] = useState<'mounting' | 'loading' | 'settled'>('mounting')
  // Only whether a first load is running matters. A count (useIsFetching)
  // re-rendered the whole screen as each of its queries started and came
  // back, a dozen times on a phone's first load; and once settled, the
  // screen stops listening. The news comes in TanStack Query's batches, as
  // the queries' own data does, so the parts are drawn with their last data
  // when they show, not a moment before it.
  const watching = state !== 'settled'
  const subscribe = useCallback((changed: () => void) => (watching ? client.getQueryCache().subscribe(notifyManager.batchCalls(changed)) : () => {}), [client, watching])
  const loading = useSyncExternalStore(
    subscribe,
    () => watching && client.isFetching(firstLoad) > 0,
    () => false,
  )
  // React's "adjust state when a value changes" pattern, no effect.
  if (state === 'loading' && !loading) setState('settled')
  useEffect(() => {
    // After the first commit, once the children have started their queries.
    const first = setTimeout(() => setState((s) => (s === 'mounting' ? (client.isFetching(firstLoad) > 0 ? 'loading' : 'settled') : s)), 0)
    const last = setTimeout(() => setState('settled'), maxMs)
    return () => {
      clearTimeout(first)
      clearTimeout(last)
    }
  }, [client, maxMs])
  return state === 'settled'
}
