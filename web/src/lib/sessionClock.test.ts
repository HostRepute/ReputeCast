import { ApiError } from '@/api/client'
import { computeClock, getSessionState, handleAuthProblem, registerSessionGuard, resetSessionClock, WARNING_MS, type SessionTiming } from './sessionClock'

const HOUR = 3_600_000
const now = Date.parse('2026-10-13T09:00:00Z')
const iso = (ms: number) => new Date(ms).toISOString()

function timing(patch: Partial<SessionTiming> = {}): SessionTiming {
  return {
    signedInAt: iso(now - 2 * 24 * HOUR),
    lastSeenAt: iso(now),
    idleTimeoutSeconds: 12 * 3600,
    expiresAt: iso(now + 12 * 24 * HOUR),
    endsAt: iso(now + 12 * HOUR),
    endsBy: 'idle',
    extendable: false,
    serverTime: iso(now),
    ...patch,
  }
}

const state = (t?: SessionTiming, extra: Partial<ReturnType<typeof getSessionState>> = {}) => ({ timing: t, skew: 0, sharedActivity: 0, guards: 0, ...extra })

describe('computeClock', () => {
  it('ends by the idle timeout, 12 hours after the last activity', () => {
    const c = computeClock(state(timing()), 0, now + HOUR)
    expect(c.phase).toBe('ok')
    expect(c.endsBy).toBe('idle')
    expect(c.endsAt).toBe(now + 12 * HOUR)
    expect(c.msLeft).toBe(11 * HOUR)
  })

  it('warns two minutes before, with the extension on offer', () => {
    const at = now + 12 * HOUR - WARNING_MS + 1000
    const c = computeClock(state(timing()), 0, at)
    expect(c.phase).toBe('warning')
    expect(c.extendable).toBe(true)
    expect(c.extendMs).toBe(12 * HOUR)
    expect(Math.round(c.idleForMs / 60_000)).toBe(11 * 60 + 58)
  })

  it('counts any API response, here or in another tab, as activity', () => {
    const at = now + 12 * HOUR - WARNING_MS + 1000
    expect(computeClock(state(timing()), now + 6 * HOUR, at).phase).toBe('ok')
    expect(computeClock(state(timing(), { sharedActivity: now + 6 * HOUR }), 0, at).phase).toBe('ok')
  })

  it('ends at the lifetime when that comes first, and then offers no extension', () => {
    const t = timing({ expiresAt: iso(now + HOUR) })
    const c = computeClock(state(t), 0, now + HOUR - 60_000)
    expect(c.endsBy).toBe('lifetime')
    expect(c.phase).toBe('warning')
    expect(c.extendable).toBe(false)
  })

  it('is over once the deadline passes', () => {
    const c = computeClock(state(timing()), 0, now + 12 * HOUR + 1)
    expect(c.phase).toBe('ended')
    expect(c.ended).toBe('idle')
  })

  it('corrects for a browser clock that runs late', () => {
    // The server is 5 minutes ahead of this browser.
    const c = computeClock(state(timing(), { skew: 5 * 60_000 }), 0, now - 5 * 60_000 + HOUR)
    expect(c.msLeft).toBe(11 * HOUR)
  })

  it('stays unknown without timing, and never warns', () => {
    expect(computeClock(state(), 0, now).phase).toBe('unknown')
  })
})

describe('handleAuthProblem', () => {
  beforeEach(() => resetSessionClock())

  it('leaves 401s to the router when no guard is mounted', () => {
    expect(handleAuthProblem(new ApiError(401, { status: 401, code: 'session_expired' } as never))).toBe(false)
  })

  it('keeps the person on the page while a guard is mounted', () => {
    const off = registerSessionGuard()
    expect(handleAuthProblem(new ApiError(401, { status: 401, code: 'session_expired' } as never))).toBe(true)
    expect(getSessionState().ended).toBe('idle')
    off()
  })

  it('does not take over password-change or two-factor redirects', () => {
    const off = registerSessionGuard()
    expect(handleAuthProblem(new ApiError(403, { status: 403, code: 'password_change_required' } as never))).toBe(false)
    expect(getSessionState().ended).toBeUndefined()
    off()
  })
})
