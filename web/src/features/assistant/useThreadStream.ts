import { useCallback, useEffect, useRef, useState, type Dispatch } from 'react'
import { MAX_ATTEMPTS, retryDelay } from '@/features/live/useLiveSend'
import type { Action, StreamEvent } from './reducer'

export type StreamStatus = 'idle' | 'connecting' | 'open' | 'retrying' | 'stopped' | 'ended'

export interface ThreadStream {
  status: StreamStatus
  /** Attempts since the connection dropped (0 while connected). */
  attempt: number
  lostAt?: Date
  reconnect: () => void
}

const KINDS = ['snapshot', 'turn', 'delta', 'tool', 'message', 'usage', 'problem'] as const

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
 */
export function useThreadStream(slug: string, thread: string | undefined, lastId: number, enabled: boolean, dispatch: Dispatch<Action>): ThreadStream {
  const [state, setState] = useState<{ status: StreamStatus; attempt: number; lostAt?: Date }>({ status: 'idle', attempt: 0 })
  const [nonce, setNonce] = useState(0)
  // The last event applied (or received) of this thread: a new stream starts after it.
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
    const flush = () => {
      frame = undefined
      if (!queue.length || disposed) return
      const events = queue
      queue = []
      dispatch({ type: 'events', events })
    }
    const schedule = () => {
      if (frame !== undefined) return
      // Hidden tabs don't paint: hand the events over on a timer there.
      frame = document.hidden ? window.setTimeout(flush, 100) : window.requestAnimationFrame(flush)
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
          if (id > 0) cursor.current = { thread, id: Math.max(cursor.current.id, id) }
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
