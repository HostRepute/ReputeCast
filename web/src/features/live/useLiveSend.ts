import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError } from '@/api/client'
import { useI18n } from '@/i18n'
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

/**
 * The server says something at least every 15 s (a heartbeat while nothing
 * changes). Hearing nothing for 90 s means the data is stale, though the
 * connection looks open: a proxy may be buffering it (LiveMonitorStates).
 * The monitor then checks every 10 s until the stream speaks again.
 */
export const STALE_AFTER = 90_000
export const POLL_EVERY = 10_000

export type StreamStatus = 'connecting' | 'open' | 'stale' | 'retrying' | 'stopped'

export interface StreamState {
  status: StreamStatus
  /** Attempts since the connection dropped (0 while connected). */
  attempt: number
  /** When the connection dropped. */
  lostAt?: Date
  /** When the stream last said anything (a snapshot or a heartbeat). */
  heardAt?: Date
}

/** Whether a stream heard nothing for too long while it looks open. */
export function isStale(heardAt: Date | undefined, now: Date): boolean {
  return heardAt !== undefined && now.getTime() - heardAt.getTime() > STALE_AFTER
}

/**
 * Subscribes to a send's server-sent snapshots. On an error the stream is
 * closed and reopened with back-off, passing the last event id so the
 * server knows it's a resume; after MAX_ATTEMPTS it stops until
 * `reconnect()`. A stream that says nothing for 90 s (no snapshot, no
 * heartbeat) is stale: the caller checks every 10 s instead.
 */
export function useSnapshotStream(url: string, enabled: boolean, onSnapshot: (s: Snapshot) => void) {
  const [state, setState] = useState<StreamState>({ status: 'connecting', attempt: 0 })
  const [nonce, setNonce] = useState(0)
  const lastId = useRef('')
  const lost = useRef<Date | undefined>(undefined)
  const heard = useRef<Date | undefined>(undefined)
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
    const hear = () => {
      heard.current = new Date()
      setState((s) => (s.status === 'stale' || s.status === 'connecting' ? { status: 'open', attempt: 0, heardAt: heard.current } : s))
    }
    const connect = () => {
      const target = lastId.current ? `${url}${url.includes('?') ? '&' : '?'}lastEventId=${encodeURIComponent(lastId.current)}` : url
      es = new EventSource(target, { withCredentials: true })
      es.onopen = () => {
        attempt = 0
        lost.current = undefined
        heard.current = new Date()
        setState({ status: 'open', attempt: 0, heardAt: heard.current })
      }
      es.addEventListener('snapshot', (ev) => {
        const msg = ev as MessageEvent<string>
        if (msg.lastEventId) lastId.current = msg.lastEventId
        hear()
        try {
          handler.current(JSON.parse(msg.data) as Snapshot)
        } catch {
          // A malformed event is skipped; the next snapshot is complete.
        }
      })
      es.addEventListener('heartbeat', hear)
      es.onerror = () => {
        es?.close()
        es = null
        if (disposed) return
        attempt += 1
        lost.current ??= new Date()
        if (attempt > MAX_ATTEMPTS) {
          setState({ status: 'stopped', attempt: MAX_ATTEMPTS, lostAt: lost.current, heardAt: heard.current })
          return
        }
        setState({ status: 'retrying', attempt, lostAt: lost.current, heardAt: heard.current })
        timer = window.setTimeout(connect, retryDelay(attempt))
      }
    }
    timer = window.setTimeout(connect, nonce === 0 ? STREAM_DELAY : 0)
    // An open stream that went quiet is stale.
    const watchdog = window.setInterval(() => {
      if (es && es.readyState === EventSource.OPEN && isStale(heard.current, new Date())) {
        setState((s) => (s.status === 'open' ? { ...s, status: 'stale' } : s))
      }
    }, 5000)
    return () => {
      disposed = true
      es?.close()
      window.clearTimeout(timer)
      window.clearInterval(watchdog)
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
 * While the stream is stale or reconnecting, the API is asked every 10 s.
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
  // An EventSource can't send Accept-Language: the stream is told the app's
  // language, so its snapshots read like the ones the API returns.
  const { locale } = useI18n()
  const stream = useSnapshotStream(
    `/api/v1/workspaces/${encodeURIComponent(slug)}/campaigns/${encodeURIComponent(campaignId)}/live/stream?lang=${encodeURIComponent(locale)}`,
    query.isSuccess,
    apply,
  )
  // While the stream is stale or reconnecting, ask the API every 10 s.
  const polling = stream.status === 'stale' || stream.status === 'retrying' || stream.status === 'stopped'
  const { refetch } = query
  useEffect(() => {
    if (!polling) return
    const id = window.setInterval(() => void refetch(), POLL_EVERY)
    return () => window.clearInterval(id)
  }, [polling, refetch])
  const latest = newest(query.data, pushed?.campaignId === campaignId ? pushed : null)
  // The stream's snapshots don't say who can act; the API's do.
  const helpers = query.data?.helpers
  const snapshot = useMemo(() => (latest && !latest.helpers && helpers ? { ...latest, helpers } : latest), [latest, helpers])
  return { query, snapshot, helpers: query.data?.helpers, stream, apply, polling }
}
