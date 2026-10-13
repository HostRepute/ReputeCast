import { useEffect, useState, useSyncExternalStore } from 'react'
import { http, type ApiError, type Schemas } from '@/api/client'
import { getSystemStatus, subscribeSystemStatus } from './systemStatus'

/**
 * Mirrors when the browser session ends, so the app can warn two minutes
 * before an idle timeout and, when the session does end, ask for the
 * password in place instead of leaving the page (SystemStates board).
 *
 * The server's idle clock restarts on every API request (recorded at most
 * every 30 s), so the deadline here is the later of the server's lastSeenAt
 * and this browser's last API response minus 30 s, plus the idle timeout,
 * capped by the absolute lifetime. Reading the timing never extends the
 * session by itself beyond what any request does.
 */
export type SessionTiming = Schemas['SessionTiming']
export type SessionPhase = 'unknown' | 'ok' | 'warning' | 'ended'
export type EndReason = 'idle' | 'lifetime' | 'signed-out'

/** The warning shows this long before the session ends. */
export const WARNING_MS = 2 * 60_000
/** The server touches sessions at most this often. */
const TOUCH_GRANULARITY_MS = 30_000
const ACTIVITY_KEY = 'rc:last-activity'
const RENEWED_KEY = 'rc:session-renewed'

interface State {
  timing?: SessionTiming
  /** serverTime − local time when the timing arrived. */
  skew: number
  ended?: EndReason
  /** Last API activity seen in other tabs (epoch ms). */
  sharedActivity: number
  guards: number
  workNote?: string
}

let state: State = { skew: 0, sharedActivity: 0, guards: 0 }
const listeners = new Set<() => void>()
function set(patch: Partial<State>) {
  state = { ...state, ...patch }
  listeners.forEach((l) => l())
}
function subscribe(l: () => void) {
  listeners.add(l)
  const off = subscribeSystemStatus(l)
  return () => {
    listeners.delete(l)
    off()
  }
}

export function getSessionState(): Readonly<State> {
  return state
}

/** Test helper. */
export function resetSessionClock() {
  state = { skew: 0, sharedActivity: 0, guards: 0 }
  listeners.forEach((l) => l())
}

// ------------------------------------------------------------ derived

export interface SessionClock {
  phase: SessionPhase
  /** Local epoch ms when the session ends. */
  endsAt?: number
  endsBy?: 'idle' | 'lifetime'
  msLeft: number
  /** Time since the last activity the server saw. */
  idleForMs: number
  /** Time since signing in. */
  signedInForMs: number
  extendable: boolean
  /** How much an extension buys (the idle timeout, capped by the lifetime). */
  extendMs: number
  ended?: EndReason
}

export function computeClock(s: Readonly<State>, lastActivity: number, now: number): SessionClock {
  const t = s.timing
  if (s.ended) return { phase: 'ended', ended: s.ended, msLeft: 0, idleForMs: 0, signedInForMs: 0, extendable: false, extendMs: 0 }
  if (!t) return { phase: 'unknown', msLeft: Infinity, idleForMs: 0, signedInForMs: 0, extendable: false, extendMs: 0 }
  const local = (iso: string) => Date.parse(iso) - s.skew
  const lifetimeEnds = local(t.expiresAt)
  const activity = Math.max(lastActivity, s.sharedActivity)
  const lastSeen = Math.max(local(t.lastSeenAt), activity ? activity - TOUCH_GRANULARITY_MS : 0)
  const idleMs = t.idleTimeoutSeconds * 1000
  const idleEnds = idleMs > 0 ? lastSeen + idleMs : Infinity
  const endsAt = Math.min(idleEnds, lifetimeEnds)
  const endsBy = idleEnds <= lifetimeEnds ? 'idle' : 'lifetime'
  const msLeft = endsAt - now
  const extendTo = idleMs > 0 ? Math.min(now + idleMs, lifetimeEnds) : endsAt
  const phase: SessionPhase = msLeft <= 0 ? 'ended' : msLeft <= WARNING_MS ? 'warning' : 'ok'
  return {
    phase,
    endsAt,
    endsBy,
    msLeft,
    idleForMs: Math.max(0, now - lastSeen),
    signedInForMs: Math.max(0, now - local(t.signedInAt)),
    extendable: endsBy === 'idle' && extendTo - endsAt >= 60_000,
    extendMs: Math.max(0, extendTo - now),
    ended: phase === 'ended' ? endsBy : undefined,
  }
}

/**
 * The session clock, re-rendering when the phase or (during the warning)
 * the countdown changes.
 */
export function useSessionClock(): SessionClock {
  const s = useSyncExternalStore(subscribe, getSessionState, getSessionState)
  const status = useSyncExternalStore(subscribeSystemStatus, getSystemStatus, getSystemStatus)
  const [now, setNow] = useState(() => Date.now())
  const clock = computeClock(s, status.lastActivity, now)

  useEffect(() => {
    if (clock.phase === 'ended' || clock.phase === 'unknown' || clock.endsAt === undefined) return
    // Tick every second in the warning; otherwise wake up when it starts.
    const wait = clock.phase === 'warning' ? 1000 : Math.max(250, clock.endsAt - WARNING_MS - Date.now())
    const h = window.setTimeout(() => setNow(Date.now()), Math.min(wait, 2 ** 31 - 1))
    return () => window.clearTimeout(h)
  }, [clock.phase, clock.endsAt, now])

  useEffect(() => {
    // Background tabs throttle timers; re-check when the tab is seen again.
    const onVisible = () => document.visibilityState === 'visible' && setNow(Date.now())
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  }, [])

  return clock
}

// ------------------------------------------------------------ actions

function accept(t: SessionTiming) {
  set({ timing: t, skew: Date.parse(t.serverTime) - Date.now(), ended: undefined })
}

/** Loads the session timing (on mount and after signing in again). */
export async function loadSessionTiming(): Promise<void> {
  try {
    const { data } = await http.GET('/api/v1/me/session')
    if (data) accept(data)
  } catch {
    // A 401 is handled by the auth listener; anything else leaves the
    // clock unknown (no warning rather than a wrong one).
  }
}

/** "Extend 12 hours": restarts the idle clock. */
export async function extendSession(): Promise<SessionTiming> {
  const { data } = await http.POST('/api/v1/me/session/extend')
  accept(data!)
  return data!
}

/** The session is over (deadline passed locally, or the API said so). */
export function markSessionEnded(reason: EndReason) {
  if (!state.ended) set({ ended: reason })
}

/** Signed in again in place: reload the timing and tell other tabs. */
export async function sessionRenewed(): Promise<void> {
  set({ ended: undefined })
  try {
    window.localStorage.setItem(RENEWED_KEY, String(Date.now()))
  } catch {
    // other tabs will notice on their next request
  }
  await loadSessionTiming()
}

/**
 * Called for every authentication problem from the API. While a session
 * guard is mounted (signed-in screens), an ended session shows the re-sign-in
 * dialog over the page instead of navigating away, so unsaved work stays.
 */
export function handleAuthProblem(e: ApiError): boolean {
  if (state.guards === 0) return false
  // The API says session_expired for the idle timeout; a session past its
  // lifetime (or signed out elsewhere) is simply unknown: unauthenticated.
  if (e.code === 'session_expired') {
    markSessionEnded('idle')
    return true
  }
  if (e.code === 'unauthenticated' || (e.status === 401 && !e.code)) {
    const pastLifetime = state.timing && Date.parse(state.timing.expiresAt) - state.skew <= Date.now()
    markSessionEnded(pastLifetime ? 'lifetime' : 'signed-out')
    return true
  }
  return false
}

/** Mounted by the session guard; while any is mounted, 401s stay in place. */
export function registerSessionGuard(): () => void {
  set({ guards: state.guards + 1 })
  return () => set({ guards: Math.max(0, state.guards - 1) })
}

// ------------------------------------------------------------ unsaved work

/**
 * Lets a screen describe its unsaved work for the session dialogs, e.g. the
 * composer: "The campaign you were editing is saved as draft v18 and will
 * reopen where you left off."
 */
export function useUnsavedWorkNote(note: string | undefined) {
  useEffect(() => {
    if (!note) return
    set({ workNote: note })
    return () => {
      if (state.workNote === note) set({ workNote: undefined })
    }
  }, [note])
}

// ------------------------------------------------------------ other tabs

let shareInstalled = false
let lastShared = 0

/** Shares API activity and re-sign-ins between tabs of this browser. */
export function installSessionSharing() {
  if (shareInstalled || typeof window === 'undefined') return
  shareInstalled = true
  subscribeSystemStatus(() => {
    const a = getSystemStatus().lastActivity
    if (a - lastShared < 10_000) return
    lastShared = a
    try {
      window.localStorage.setItem(ACTIVITY_KEY, String(a))
    } catch {
      // storage unavailable: each tab keeps its own clock
    }
  })
  window.addEventListener('storage', (e) => {
    if (e.key === ACTIVITY_KEY && e.newValue) {
      const a = Number(e.newValue)
      if (Number.isFinite(a) && a > state.sharedActivity) set({ sharedActivity: a })
    }
    if (e.key === RENEWED_KEY && state.ended) {
      set({ ended: undefined })
      void loadSessionTiming()
    }
  })
  try {
    const a = Number(window.localStorage.getItem(ACTIVITY_KEY))
    if (Number.isFinite(a) && a > 0) state = { ...state, sharedActivity: a }
  } catch {
    // ignore
  }
}

