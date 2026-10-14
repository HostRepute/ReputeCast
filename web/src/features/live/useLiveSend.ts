import { useCallback, useEffect, useRef, useState } from 'react'
import { $api, ApiError } from '@/api/client'
import type { Snapshot } from './model'

/** Reconnection attempts before the monitor stops and offers "Reconnect now". */
export const MAX_ATTEMPTS = 10

/** Back-off between attempts: 2 s, 4 s, 8 s, 16 s, then every 30 s. */
export function retryDelay(attempt: number): number {
  return Math.min(30_000, 1000 * 2 ** Math.max(1, attempt))
}

/**
 * The stream opens a moment after the first snapshot has rendered, so it
 * never competes with the screen's first requests (the snapshot it sends
 * on connect is the same one the page just loaded).
 */
export const STREAM_DELAY = 1000

export type StreamStatus = 'connecting' | 'open' | 'retrying' | 'stopped'

export interface StreamState {
  status: StreamStatus
  /** Attempts since the connection dropped (0 while connected). */
  attempt: number
  /** When the connection dropped. */
  lostAt?: Date
}

/**
 * Subscribes to a send's server-sent snapshots. On an error the stream is
 * closed and reopened with back-off, passing the last event id so the
 * server knows it's a resume; after MAX_ATTEMPTS it stops until
 * `reconnect()`.
 */
export function useSnapshotStream(url: string, enabled: boolean, onSnapshot: (s: Snapshot) => void) {
  const [state, setState] = useState<StreamState>({ status: 'connecting', attempt: 0 })
  const [nonce, setNonce] = useState(0)
  const lastId = useRef('')
  const lost = useRef<Date | undefined>(undefined)
  const handler = useRef(onSnapshot)
  useEffect(() => {
    handler.current = onSnapshot
  }, [onSnapshot])

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') return
    let es: EventSource | null = null
    let timer: number | undefined
    let disposed = false
    let attempt = 0
    const connect = () => {
      const target = lastId.current ? `${url}?lastEventId=${encodeURIComponent(lastId.current)}` : url
      es = new EventSource(target, { withCredentials: true })
      es.onopen = () => {
        attempt = 0
        lost.current = undefined
        setState({ status: 'open', attempt: 0 })
      }
      es.addEventListener('snapshot', (ev) => {
        const msg = ev as MessageEvent<string>
        if (msg.lastEventId) lastId.current = msg.lastEventId
        try {
          handler.current(JSON.parse(msg.data) as Snapshot)
        } catch {
          // A malformed event is skipped; the next snapshot is complete.
        }
      })
      es.onerror = () => {
        es?.close()
        es = null
        if (disposed) return
        attempt += 1
        lost.current ??= new Date()
        if (attempt > MAX_ATTEMPTS) {
          setState({ status: 'stopped', attempt: MAX_ATTEMPTS, lostAt: lost.current })
          return
        }
        setState({ status: 'retrying', attempt, lostAt: lost.current })
        timer = window.setTimeout(connect, retryDelay(attempt))
      }
    }
    timer = window.setTimeout(connect, nonce === 0 ? STREAM_DELAY : 0)
    return () => {
      disposed = true
      es?.close()
      window.clearTimeout(timer)
    }
  }, [url, enabled, nonce])

  const reconnect = useCallback(() => {
    setState((s) => ({ ...s, status: 'retrying', attempt: 1 }))
    setNonce((n) => n + 1)
  }, [])

  return { ...state, reconnect }
}

/** The newer of two snapshots of the same campaign. */
export function newest(a: Snapshot | null | undefined, b: Snapshot | null | undefined): Snapshot | null {
  if (!a) return b ?? null
  if (!b || b.campaignId !== a.campaignId) return a
  return b.version >= a.version ? b : a
}

/** 403 and 404 aren't worth retrying. */
function retryable(n: number, e: unknown): boolean {
  return n < 2 && !(e instanceof ApiError && (e.status === 403 || e.status === 404))
}

/**
 * The live monitor's data: the snapshot from the API, then every snapshot
 * the event stream pushes (and the ones actions return), newest wins.
 */
export function useLiveSend(slug: string, campaignId: string) {
  const query = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/live',
    { params: { path: { workspace: slug, campaignId } } },
    { refetchOnWindowFocus: false, retry: retryable },
  )
  const [pushed, setPushed] = useState<Snapshot | null>(null)
  const apply = useCallback((s: Snapshot) => {
    setPushed((cur) => (cur && cur.campaignId === s.campaignId && cur.version > s.version ? cur : s))
  }, [])
  const stream = useSnapshotStream(
    `/api/v1/workspaces/${encodeURIComponent(slug)}/campaigns/${encodeURIComponent(campaignId)}/live/stream`,
    query.isSuccess,
    apply,
  )
  const snapshot = newest(query.data, pushed?.campaignId === campaignId ? pushed : null)
  return { query, snapshot, helpers: query.data?.helpers, stream, apply }
}
