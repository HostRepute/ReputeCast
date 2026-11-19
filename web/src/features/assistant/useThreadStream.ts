import { useCallback, useEffect, useMemo, useReducer, useRef, useState, type Dispatch } from 'react'
import { flushSync } from 'react-dom'
import { MAX_ATTEMPTS, retryDelay } from '@/features/live/useLiveSend'
import { initialThreadState, threadReducer, type Action, type StreamEvent, type ThreadState } from './reducer'

/** The streaming answer's text, ahead of the thread's state (useThreadState). */
export interface LiveText {
  get: () => ThreadState['live']
  subscribe: (listener: () => void) => () => void
}

/**
 * The thread's state (threadReducer) with one shortcut: deltas that only
 * lengthen the answer already on screen don't change it. They go to
 * LiveText, which only that answer's text reads, so the rest of the panel
 * doesn't render again for each frame of deltas; the next other action
 * applies them first, so the state is the same as if each had been
 * dispatched.
 */
export function useThreadState(): [ThreadState, Dispatch<Action>, LiveText] {
  const [state, apply] = useReducer(threadReducer, initialThreadState)
  // What the state is once React has applied every action, and the deltas it hasn't been given.
  const ref = useRef({ latest: initialThreadState, held: [] as StreamEvent[], listeners: new Set<() => void>() })
  const dispatch = useCallback((a: Action) => {
    const box = ref.current
    const before = box.latest
    box.latest = threadReducer(before, a)
    const turn = before.live?.turnId
    // Only text that lengthens the answer waits; a withheld piece (empty) goes at once.
    if (turn && !before.live?.withheld && a.type === 'events' && a.events.every((e) => e.type === 'delta' && e.data.turnId === turn && e.data.text)) box.held.push(...a.events)
    else {
      if (box.held.length) apply({ type: 'events', events: box.held })
      box.held = []
      apply(a)
    }
    if (box.latest.live !== before.live) for (const l of box.listeners) l()
  }, [])
  const live = useMemo<LiveText>(
    () => ({
      get: () => ref.current.latest.live,
      subscribe: (l) => {
        ref.current.listeners.add(l)
        return () => ref.current.listeners.delete(l)
      },
    }),
    [],
  )
  return [state, dispatch, live]
}

export type StreamStatus = 'idle' | 'connecting' | 'open' | 'retrying' | 'stopped' | 'ended'

export interface ThreadStream {
  status: StreamStatus
  /** Attempts since the connection dropped (0 while connected). */
  attempt: number
  lostAt?: Date
  reconnect: () => void
}

const KINDS = ['snapshot', 'turn', 'delta', 'tool', 'message', 'usage', 'confirmation', 'problem'] as const

/**
 * The server says something at least every 15 s (a heartbeat while nothing
 * happens). A stream silent for three of those is gone even when it looks
 * open (a proxy holding a dead connection): it is closed and opened again.
 */
export const SILENT_AFTER = 45_000

/**
 * A thread's server-sent events, as the live monitor's stream works
 * (useLiveSend): on an error the stream closes and opens again with back-off
 * after the last event it applied (lastEventId), so nothing comes twice or
 * goes missing; after MAX_ATTEMPTS it waits for reconnect(). A problem
 * event ends it for good (the thread isn't the person's, they lost ai:use,
 * the assistant is off). Events are handed over once per animation frame,
 * so a burst of deltas renders once.
 *
 * `applied` is the last event the thread's state applied, and whose thread
 * that state is: right after a switch it is still the previous thread's,
 * whose number means nothing here.
 */
export function useThreadStream(slug: string, thread: string | undefined, applied: { thread?: string; id: number }, enabled: boolean, dispatch: Dispatch<Action>): ThreadStream {
  const [state, setState] = useState<{ status: StreamStatus; attempt: number; lostAt?: Date }>({ status: 'idle', attempt: 0 })
  const [nonce, setNonce] = useState(0)
  // The last event applied (or received) of this thread: a new stream starts after it.
  const lastId = applied.thread === thread ? applied.id : 0
  const cursor = useRef<{ thread?: string; id: number }>({ thread, id: lastId })
  useEffect(() => {
    cursor.current = cursor.current.thread === thread ? { thread, id: Math.max(cursor.current.id, lastId) } : { thread, id: lastId }
  }, [thread, lastId])

  useEffect(() => {
    if (!thread || !enabled || typeof EventSource === 'undefined') return
    let es: EventSource | null = null
    let timer: number | undefined
    let frame: number | undefined
    let disposed = false
    let attempt = 0
    let lost: Date | undefined
    let queue: StreamEvent[] = []
    let heard = Date.now()
    // What applying a batch costs here (render and layout), and when the last ended.
    let cost = 0
    let ended = 0
    const flush = (now = false) => {
      frame = undefined
      if (!queue.length || disposed) return
      const events = queue
      queue = []
      const start = performance.now()
      // In the frame's callback the update renders at once, so this frame
      // paints it (not the next one, after a task React would schedule).
      if (now) flushSync(() => dispatch({ type: 'events', events }))
      else dispatch({ type: 'events', events })
      ended = performance.now()
      cost = cost * 0.7 + (ended - start) * 0.3
    }
    const schedule = () => {
      if (frame !== undefined) return
      // Hidden tabs don't paint: hand the events over on a timer there.
      if (document.hidden) {
        frame = window.setTimeout(() => flush(), 100)
        return
      }
      // Where a batch takes longer than a frame (a slow phone), the next
      // waits as long again, so taps and typing get half the time; the
      // deltas meanwhile render together.
      const wait = ended + cost - performance.now()
      if (cost > 12 && wait > 4) {
        frame = window.setTimeout(() => {
          frame = window.requestAnimationFrame(() => flush(true))
        }, wait)
      } else frame = window.requestAnimationFrame(() => flush(true))
    }
    const url = `/api/v1/workspaces/${encodeURIComponent(slug)}/ai/assistant/threads/${encodeURIComponent(thread)}/stream`
    const connect = () => {
      const at = cursor.current.id
      es = new EventSource(at > 0 ? `${url}?lastEventId=${at}` : url, { withCredentials: true })
      setState((s) => (s.status === 'retrying' ? s : { status: 'connecting', attempt: 0 }))
      heard = Date.now()
      es.onopen = () => {
        heard = Date.now()
        attempt = 0
        lost = undefined
        setState({ status: 'open', attempt: 0 })
      }
      for (const kind of KINDS) {
        es.addEventListener(kind, (ev) => {
          const msg = ev as MessageEvent<string>
          heard = Date.now()
          const id = Number(msg.lastEventId) || 0
          let data: unknown
          try {
            data = JSON.parse(msg.data)
          } catch {
            return // A malformed event is skipped; the next snapshot is whole.
          }
          // A snapshot is the whole thread as of its number: the cursor is that, even when lower.
          if (id > 0) cursor.current = { thread, id: kind === 'snapshot' ? id : Math.max(cursor.current.id, id) }
          queue.push({ type: kind, id, data } as StreamEvent)
          if (kind === 'problem') {
            es?.close()
            es = null
            flush()
            setState({ status: 'ended', attempt: 0 })
            return
          }
          schedule()
        })
      }
      es.addEventListener('heartbeat', () => {
        heard = Date.now()
        setState((s) => (s.status === 'open' ? s : { status: 'open', attempt: 0 }))
      })
      es.onerror = () => {
        es?.close()
        es = null
        if (disposed) return
        attempt += 1
        lost ??= new Date()
        if (attempt > MAX_ATTEMPTS) {
          setState({ status: 'stopped', attempt: MAX_ATTEMPTS, lostAt: lost })
          return
        }
        setState({ status: 'retrying', attempt, lostAt: lost })
        timer = window.setTimeout(connect, retryDelay(attempt))
      }
    }
    connect()
    const watchdog = window.setInterval(() => {
      if (es && es.readyState === EventSource.OPEN && Date.now() - heard > SILENT_AFTER) es.onerror?.call(es, new Event('error'))
    }, 5000)
    return () => {
      window.clearInterval(watchdog)
      // What arrived is applied before the stream goes: the cursor is past it.
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame)
        window.clearTimeout(frame)
      }
      flush()
      disposed = true
      es?.close()
      window.clearTimeout(timer)
    }
  }, [slug, thread, enabled, nonce, dispatch])

  const reconnect = useCallback(() => {
    setState((s) => ({ ...s, status: 'retrying', attempt: 1 }))
    setNonce((n) => n + 1)
  }, [])

  return { ...state, reconnect }
}
