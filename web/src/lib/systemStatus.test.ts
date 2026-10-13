import { clearRateLimit, getSystemStatus, probe, resetSystemStatus, systemStatusMiddleware } from './systemStatus'

type OnResponse = NonNullable<typeof systemStatusMiddleware.onResponse>
type OnError = NonNullable<typeof systemStatusMiddleware.onError>

const req = (path = '/api/v1/workspaces/acme-newsletter/campaigns', method = 'GET') => new Request(path, { method })

async function respond(response: Response, request = req()) {
  await (systemStatusMiddleware.onResponse as OnResponse)({ request, response, schemaPath: '', params: {}, options: {} as never, id: '1' })
}

describe('system status middleware', () => {
  beforeEach(() => {
    resetSystemStatus()
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('records API activity', async () => {
    vi.setSystemTime(Date.parse('2026-10-13T09:00:00Z'))
    await respond(new Response('{}', { status: 200 }))
    expect(getSystemStatus().lastActivity).toBe(Date.parse('2026-10-13T09:00:00Z'))
  })

  it('turns a 429 into a rate limit that lifts after Retry-After', async () => {
    await respond(new Response(JSON.stringify({ status: 429, code: 'rate_limited', detail: 'Rate limit reached.' }), { status: 429, headers: { 'Retry-After': '23' } }))
    const limit = getSystemStatus().rateLimit
    expect(limit?.detail).toBe('Rate limit reached.')
    expect(limit!.until - Date.now()).toBe(23_000)
    vi.advanceTimersByTime(23_100)
    expect(getSystemStatus().rateLimit).toBeUndefined()
  })

  it('shows read-only maintenance from a 503 and clears it once a write goes through', async () => {
    await respond(
      new Response(JSON.stringify({ status: 503, code: 'maintenance', meta: { version: '1.5', startedBy: 'Linh Nguyen', etaSeconds: 240 } }), { status: 503 }),
    )
    expect(getSystemStatus().maintenance).toMatchObject({ version: '1.5', startedBy: 'Linh Nguyen', etaSeconds: 240 })
    await respond(new Response('{}', { status: 200 }), req('/api/v1/workspaces/acme-newsletter/campaigns', 'POST'))
    expect(getSystemStatus().maintenance).toBeUndefined()
  })

  it('marks the server unreachable when fetch fails, and online again when it answers', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    ;(systemStatusMiddleware.onError as OnError)({ request: req(), error: new TypeError('Failed to fetch'), schemaPath: '', params: {}, options: {} as never, id: '1' })
    expect(getSystemStatus().connection).toBe('unreachable')
    expect(getSystemStatus().lostAt).toBeDefined()

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 200 })))
    await expect(probe()).resolves.toBe(true)
    expect(getSystemStatus().connection).toBe('online')
  })

  it('ignores requests that are not API calls', async () => {
    await respond(new Response('', { status: 429 }), req('/assets/app.js'))
    expect(getSystemStatus().rateLimit).toBeUndefined()
    clearRateLimit()
  })
})
