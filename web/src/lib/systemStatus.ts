import { onlineManager } from '@tanstack/react-query'
import type { Middleware } from 'openapi-fetch'
import { useSyncExternalStore } from 'react'

/**
 * App-wide system status behind the banners at the top of the content
 * column (SystemStates board): connectivity, server-side read-only
 * maintenance and the API rate limit. One middleware on the API client feeds
 * it, so every screen gets these states without doing anything.
 *
 *   connection  online · offline (the browser has no network) · unreachable
 *               (the network works but ReputeCast doesn't answer)
 *   maintenance set while the API answers 503 with code "maintenance" or
 *               "read_only" (problem meta: version, startedBy, startedAt,
 *               etaSeconds, notesUrl)
 *   rateLimit   set while the API answers 429 "rate_limited"; cleared when
 *               Retry-After has passed
 */
export type Connection = 'online' | 'offline' | 'unreachable'

export interface Maintenance {
  version?: string
  startedBy?: string
  startedAt?: string
  etaSeconds?: number
  notesUrl?: string
  detail?: string
}

export interface RateLimit {
  /** Epoch ms when requests may resume. */
  until: number
  detail?: string
}

export interface SystemStatus {
  connection: Connection
  /** When the connection was lost (epoch ms). */
  lostAt?: number
  maintenance?: Maintenance
  rateLimit?: RateLimit
  /** Epoch ms of the last API response that reached the server. */
  lastActivity: number
}

const initial: SystemStatus = {
  connection: typeof navigator !== 'undefined' && navigator.onLine === false ? 'offline' : 'online',
  lastActivity: 0,
}
let state: SystemStatus = initial
const listeners = new Set<() => void>()

function set(patch: Partial<SystemStatus>) {
  const next = { ...state, ...patch }
  const changed = (Object.keys(patch) as (keyof SystemStatus)[]).some((k) => next[k] !== state[k])
  if (!changed) return
  state = next
  listeners.forEach((l) => l())
}

export function getSystemStatus(): SystemStatus {
  return state
}

export function subscribeSystemStatus(l: () => void): () => void {
  listeners.add(l)
  return () => listeners.delete(l)
}

/** Current system status (re-renders on change). */
export function useSystemStatus(): SystemStatus {
  return useSyncExternalStore(subscribeSystemStatus, getSystemStatus, () => initial)
}

/** Test helper: back to a clean, online state. */
export function resetSystemStatus() {
  stopProbing()
  state = { ...initial, connection: 'online' }
  listeners.forEach((l) => l())
}

// ------------------------------------------------------------ connectivity

const PROBE_URL = '/api/v1/meta'
let probeTimer: number | undefined
let probeDelay = 2000

function stopProbing() {
  if (probeTimer !== undefined) window.clearTimeout(probeTimer)
  probeTimer = undefined
  probeDelay = 2000
}

function markLost(connection: Exclude<Connection, 'online'>) {
  set({ connection, lostAt: state.connection === 'online' ? Date.now() : state.lostAt })
  if (probeTimer === undefined) scheduleProbe()
}

function markOnline() {
  if (state.connection === 'online') return
  stopProbing()
  set({ connection: 'online', lostAt: undefined })
}

function scheduleProbe() {
  probeTimer = window.setTimeout(() => {
    probeTimer = undefined
    void probe()
  }, probeDelay)
  // Back off to every 30 s while the connection stays down.
  probeDelay = Math.min(probeDelay * 2, 30_000)
}

/**
 * Checks whether ReputeCast answers ("Retry now" calls it too). Resolves to
 * true when the connection is back.
 */
export async function probe(): Promise<boolean> {
  if (typeof navigator !== 'undefined' && navigator.onLine === false) {
    markLost('offline')
    return false
  }
  try {
    const r = await fetch(PROBE_URL, { method: 'GET', credentials: 'same-origin', cache: 'no-store', headers: { Accept: 'application/json' } })
    if (r.status >= 500 && r.status !== 503) throw new Error(`HTTP ${r.status}`)
    markOnline()
    return true
  } catch {
    markLost(navigator.onLine === false ? 'offline' : 'unreachable')
    return false
  }
}

// ------------------------------------------------------------ rate limit

let rateTimer: number | undefined

function retryAfterMs(header: string | null): number {
  if (!header) return 60_000
  const secs = Number(header)
  if (Number.isFinite(secs)) return Math.max(1, secs) * 1000
  const at = Date.parse(header)
  return Number.isFinite(at) ? Math.max(1000, at - Date.now()) : 60_000
}

function markRateLimited(retryAfter: string | null, detail?: string) {
  const until = Date.now() + retryAfterMs(retryAfter)
  set({ rateLimit: { until, detail } })
  if (rateTimer !== undefined) window.clearTimeout(rateTimer)
  rateTimer = window.setTimeout(() => {
    rateTimer = undefined
    if (state.rateLimit && state.rateLimit.until <= Date.now()) set({ rateLimit: undefined })
  }, until - Date.now() + 50)
}

/** Dismisses the rate-limit notice (after its countdown, or by the person). */
export function clearRateLimit() {
  if (rateTimer !== undefined) window.clearTimeout(rateTimer)
  rateTimer = undefined
  set({ rateLimit: undefined })
}

// ------------------------------------------------------------ middleware

const isApi = (url: string) => {
  try {
    return new URL(url, window.location.href).pathname.startsWith('/api/')
  } catch {
    return false
  }
}

/**
 * Observes every API response. Registered after the client's own error
 * middleware, so it runs first (openapi-fetch runs onResponse in reverse
 * order) and sees 429/503 before they are thrown as ApiError.
 */
export const systemStatusMiddleware: Middleware = {
  async onResponse({ request, response }) {
    if (!isApi(request.url)) return undefined
    if (state.connection !== 'online') markOnline()
    if (response.status !== 401) set({ lastActivity: Date.now() })
    if (response.status === 429 || response.status === 503) {
      let problem: { code?: string; detail?: string; meta?: Record<string, unknown> } = {}
      try {
        problem = await response.clone().json()
      } catch {
        // not JSON: still a limit or an outage
      }
      if (response.status === 429) markRateLimited(response.headers.get('Retry-After'), problem.detail)
      else if (problem.code === 'maintenance' || problem.code === 'read_only') {
        const m = problem.meta ?? {}
        set({
          maintenance: {
            version: typeof m.version === 'string' ? m.version : undefined,
            startedBy: typeof m.startedBy === 'string' ? m.startedBy : undefined,
            startedAt: typeof m.startedAt === 'string' ? m.startedAt : undefined,
            etaSeconds: typeof m.etaSeconds === 'number' ? m.etaSeconds : undefined,
            notesUrl: typeof m.notesUrl === 'string' ? m.notesUrl : undefined,
            detail: problem.detail,
          },
        })
      }
    } else if (response.ok && state.maintenance && request.method !== 'GET') {
      // A write went through: maintenance is over.
      set({ maintenance: undefined })
    }
    return undefined
  },
  onError({ request, error }) {
    if (isApi(request.url) && error instanceof TypeError) markLost(navigator.onLine === false ? 'offline' : 'unreachable')
    return undefined
  },
}

let installed = false

/**
 * Installs the middleware on the API client and listens to the browser's
 * online/offline events. TanStack Query follows the same state, so queries
 * and saves pause while offline and resume (in order) when it's back.
 */
export function installSystemStatus(client: { use: (...m: Middleware[]) => void }) {
  if (installed || typeof window === 'undefined') return
  installed = true
  client.use(systemStatusMiddleware)
  window.addEventListener('offline', () => markLost('offline'))
  window.addEventListener('online', () => void probe())
  onlineManager.setEventListener((setOnline) => {
    // Unreachable counts as offline too: saves wait on this device instead of
    // failing, and go out once the probe sees ReputeCast again.
    const update = () => setOnline(state.connection === 'online')
    const off = subscribeSystemStatus(update)
    update()
    return off
  })
}
