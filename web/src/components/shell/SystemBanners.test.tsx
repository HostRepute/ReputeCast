import { act, fireEvent, render, screen } from '@testing-library/react'
import { setDisplayTimeZone } from '@/lib/format'
import { getSystemStatus, resetSystemStatus, systemStatusMiddleware } from '@/lib/systemStatus'
import { MaintenanceBanner, QuotaBanner, ReconnectingBanner, SystemBanners } from './SystemBanners'
import { RateLimitNotice, SessionEndingNotice } from './SystemNotices'
import type { SessionClock } from '@/lib/sessionClock'

beforeAll(() => setDisplayTimeZone('UTC'))

describe('system banners', () => {
  afterEach(() => {
    resetSystemStatus()
    vi.unstubAllGlobals()
  })

  it('appear when the connection drops and go when it is back', async () => {
    render(<SystemBanners />)
    expect(screen.queryByText(/offline/i)).not.toBeInTheDocument()
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    act(() => {
      ;(systemStatusMiddleware.onError as NonNullable<typeof systemStatusMiddleware.onError>)({
        request: new Request('/api/v1/me'),
        error: new TypeError('Failed to fetch'),
        schemaPath: '',
        params: {},
        options: {} as never,
        id: '1',
      })
    })
    const banner = screen.getByRole('status')
    expect(banner).toHaveTextContent("Can't reach ReputeCast. Edits save on this device and sync when the connection is back. Sending and live stats are paused.")
    expect(banner).toHaveTextContent('Offline')

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}')))
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Retry now' }))
    })
    expect(getSystemStatus().connection).toBe('online')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('say the browser is offline when it has no network', () => {
    const online = vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    try {
      render(<SystemBanners />)
      act(() => {
        ;(systemStatusMiddleware.onError as NonNullable<typeof systemStatusMiddleware.onError>)({
          request: new Request('/api/v1/me'),
          error: new TypeError('Failed to fetch'),
          schemaPath: '',
          params: {},
          options: {} as never,
          id: '2',
        })
      })
      expect(getSystemStatus().connection).toBe('offline')
      expect(screen.getByRole('status')).toHaveTextContent("You're offline. Edits save on this device and sync when the connection is back.")
    } finally {
      online.mockRestore()
    }
  })

  it('reconnecting: when it dropped, how old the numbers are, which attempt', () => {
    const onReconnect = vi.fn()
    render(<ReconnectingBanner droppedAt="2026-10-13T09:18:51Z" dataAsOf="2026-10-13T09:18:30Z" attempt={3} maxAttempts={10} onReconnect={onReconnect} />)
    expect(screen.getByRole('status')).toHaveTextContent(
      'Live event stream dropped at 09:18:51. Counts below are from 09:18. Reconnecting automatically, attempt 3 of 10.',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Reconnect now' }))
    expect(onReconnect).toHaveBeenCalledOnce()
  })

  it('read-only upgrade: who, since when, how long, what still works', () => {
    render(<MaintenanceBanner maintenance={{ version: '1.5', startedBy: 'Linh Nguyen', startedAt: '2026-10-13T02:00:00Z', etaSeconds: 240 }} />)
    expect(screen.getByRole('status')).toHaveTextContent(
      'ReputeCast 1.5 is being installed by Linh Nguyen (started 02:00, about 4 minutes). Reading works; edits and sends resume when it finishes.',
    )
    expect(screen.getByRole('link', { name: "What's new" })).toBeInTheDocument()
  })

  it('quota reached: what is held until when, what continues, the ways out', () => {
    const onMove = vi.fn()
    render(
      <QuotaBanner
        provider="Amazon SES"
        quota={50000}
        usedUpAt="2026-10-13T21:14:00Z"
        held={2020}
        resumesAt="00:00 UTC"
        continues="automations continue on Postmark"
        requestHref="/w/acme-newsletter/providers"
        moveTo={{ provider: 'Postal', onMove }}
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Amazon SES 24-hour quota of 50,000 is used up at 21:14. Broadcasts hold 2,020 recipients until 00:00 UTC; automations continue on Postmark.',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Move the rest to Postal' }))
    expect(onMove).toHaveBeenCalledOnce()
  })
})

const clock = (patch: Partial<SessionClock> = {}): SessionClock => ({
  phase: 'warning',
  endsBy: 'idle',
  msLeft: 119_000,
  idleForMs: (11 * 60 + 58) * 60_000,
  signedInForMs: 3 * 24 * 3_600_000,
  extendable: true,
  extendMs: 12 * 3_600_000,
  ...patch,
})

describe('timeouts and limits', () => {
  it('session ending: idle time, Extend 12 hours, sign out', () => {
    const onExtend = vi.fn()
    render(<SessionEndingNotice clock={clock()} onExtend={onExtend} onSignOut={() => {}} workNote="the draft autosaves" />)
    const notice = screen.getByRole('alert')
    expect(notice).toHaveTextContent('Session ending in 1 minute')
    expect(notice).toHaveTextContent("You've been inactive for 11 h 58 min. Extend to keep editing; otherwise the draft autosaves and you'll be asked to sign in again.")
    fireEvent.click(screen.getByRole('button', { name: 'Extend 12 hours' }))
    expect(onExtend).toHaveBeenCalledOnce()
    expect(screen.getByRole('button', { name: 'Sign out now' })).toBeInTheDocument()
  })

  it('session at the end of its lifetime: no extension on offer', () => {
    render(<SessionEndingNotice clock={clock({ endsBy: 'lifetime', extendable: false, msLeft: 120_000 })} onExtend={() => {}} onSignOut={() => {}} />)
    expect(screen.getByRole('alert')).toHaveTextContent('Session ending in 2 minutes')
    expect(screen.getByRole('alert')).toHaveTextContent('the longest your instance allows')
    expect(screen.queryByRole('button', { name: /Extend/ })).not.toBeInTheDocument()
  })

  it('API rate limit: counts down to when requests resume', () => {
    vi.useFakeTimers()
    try {
      render(<RateLimitNotice limit={{ until: Date.now() + 23_000 }} apiKey={{ masked: 'rc_live_bf2…', sent: '1,200', allowed: '600' }} />)
      const notice = screen.getByRole('status')
      expect(notice).toHaveTextContent('Key rc_live_bf2… sent 1,200 requests this minute; the limit is 600. Requests resume in 23 s.')
      for (let i = 0; i < 3; i++) act(() => vi.advanceTimersByTime(1000))
      expect(notice).toHaveTextContent('Requests resume in 20 s.')
    } finally {
      vi.useRealTimers()
    }
  })
})
