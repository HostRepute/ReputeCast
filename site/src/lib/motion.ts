import { useEffect, useState, useSyncExternalStore } from 'react'

/**
 * Motion state for the whole page. The source of truth is the <html>
 * element, so the inline script in index.html can restore a paused
 * preference before first paint and CSS can react to it without waiting
 * for React:
 *   data-motion="paused"  loops are frozen, reveals and counters are final
 *   data-settled          motion was paused at least once; entrances stay done
 * Reduced motion comes from the operating system and always wins.
 */
export interface MotionState {
  paused: boolean
  reduced: boolean
  settled: boolean
}

export const STORAGE_KEY = 'rc-motion'
const REDUCE = '(prefers-reduced-motion: reduce)'

const serverState: MotionState = { paused: false, reduced: false, settled: false }
let state: MotionState | null = null
const listeners = new Set<() => void>()
let media: MediaQueryList | null = null

function read(): MotionState {
  const el = document.documentElement
  return {
    paused: el.dataset.motion === 'paused',
    reduced: typeof window.matchMedia === 'function' ? window.matchMedia(REDUCE).matches : false,
    settled: el.hasAttribute('data-settled'),
  }
}

/** Re-reads the state (after something outside this module changed it). */
export function refresh() {
  const next = read()
  const prev = state
  if (prev && prev.paused === next.paused && prev.reduced === next.reduced && prev.settled === next.settled) return
  state = next
  listeners.forEach((l) => l())
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  if (!media && typeof window.matchMedia === 'function') {
    media = window.matchMedia(REDUCE)
    media.addEventListener('change', refresh)
  }
  return () => {
    listeners.delete(listener)
    if (!listeners.size && media) {
      media.removeEventListener('change', refresh)
      media = null
    }
  }
}

export function getMotion(): MotionState {
  if (typeof document === 'undefined') return serverState
  if (!state) state = read()
  return state
}

/** Current motion state; the prerendered HTML always assumes motion plays. */
export function useMotion(): MotionState {
  return useSyncExternalStore(subscribe, getMotion, () => serverState)
}

/** True when animations should run right now. */
export function motionAllowed(m: MotionState = getMotion()): boolean {
  return !m.reduced && !m.paused
}

/** The visible Pause motion control. Remembered on this device. */
export function setPaused(paused: boolean) {
  const el = document.documentElement
  if (paused) {
    el.dataset.motion = 'paused'
    el.setAttribute('data-settled', '')
  } else {
    delete el.dataset.motion
  }
  try {
    if (paused) window.localStorage.setItem(STORAGE_KEY, 'paused')
    else window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Storage can be unavailable (private mode, blocked site data); the
    // control still works for this page view.
  }
  refresh()
}

export function togglePaused() {
  setPaused(!getMotion().paused)
}

const easeOutCubic = (p: number) => 1 - Math.pow(1 - p, 3)

const noopSubscribe = () => () => {}

/**
 * False on the server and while hydrating, true from the first client render
 * after it: a component first renders exactly the prerendered HTML, then
 * switches to its animated state without a hydration mismatch.
 */
export function useHydrated(): boolean {
  return useSyncExternalStore(noopSubscribe, () => true, () => false)
}

/**
 * Eased progress from 0 to 1 for counters and growing bars. The prerendered
 * page shows the final value (p = 1); once hydrated, when motion is allowed
 * and the entrance hasn't been settled, it drops to 0 ("live"), waits until
 * `start` is true plus `delay` ms, then counts up over `duration` ms.
 * Pausing or reducing motion makes it final again at once.
 */
export function useProgress({ start, delay, duration }: { start: boolean; delay: number | (() => number); duration: number }): { p: number; live: boolean } {
  const motion = useMotion()
  const hydrated = useHydrated()
  const live = hydrated && motionAllowed(motion) && !motion.settled
  const [t, setT] = useState(0)

  useEffect(() => {
    if (!live || !start) return
    let raf = 0
    let begin: number | null = null
    const wait = Math.max(0, typeof delay === 'function' ? delay() : delay)
    const step = (ts: number) => {
      if (begin === null) begin = ts
      const x = Math.min(1, (ts - begin) / duration)
      setT(easeOutCubic(x))
      if (x < 1) raf = requestAnimationFrame(step)
    }
    const timer = window.setTimeout(() => {
      raf = requestAnimationFrame(step)
    }, wait)
    return () => {
      window.clearTimeout(timer)
      cancelAnimationFrame(raf)
    }
    // delay is read once, when the count starts
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [live, start, duration])

  return { p: live ? t : 1, live }
}
