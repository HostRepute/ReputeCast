import type { QueryClient } from '@tanstack/react-query'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { ApiError } from '@/api/client'
import { handleAuthProblem, resetSessionClock } from '@/lib/sessionClock'
import { meQueryKey } from '@/lib/session'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { SessionGuard } from './SessionGuard'

const HOUR = 3_600_000
const iso = (ms: number) => new Date(ms).toISOString()

function timing(lastSeen: number, now = Date.now()) {
  return {
    signedInAt: iso(now - 2 * 24 * HOUR),
    lastSeenAt: iso(lastSeen),
    idleTimeoutSeconds: 12 * 3600,
    expiresAt: iso(now + 12 * 24 * HOUR),
    idleExpiresAt: iso(lastSeen + 12 * HOUR),
    endsAt: iso(lastSeen + 12 * HOUR),
    endsBy: 'idle',
    extendable: true,
    extendsTo: iso(now + 12 * HOUR),
    serverTime: iso(now),
  }
}

let lastSeen = Date.now()
const calls: string[] = []
let unmock = () => {}

beforeEach(() => {
  unmock = mockApi({
    'GET /api/v1/me/session': () => json(timing(lastSeen)),
    'POST /api/v1/me/session/extend': () => {
      calls.push('extend')
      lastSeen = Date.now()
      return json(timing(lastSeen))
    },
    'POST /api/v1/auth/identify': () => json({ email: 'linh@acme.io', next: 'password', breakGlass: false }),
    'POST /api/v1/auth/password': async (request) => {
      const body = (await request.json()) as { password: string }
      if (body.password !== 'reputecast-demo') {
        return json({ status: 401, code: 'invalid_credentials', detail: "That password doesn't match linh@acme.io." }, 401)
      }
      return json({ status: 'two_factor', challenge: 'challenge-123456', methods: ['totp', 'recovery'], email: 'linh@acme.io' })
    },
    'POST /api/v1/auth/two-factor': async (request) => {
      const body = (await request.json()) as { code: string; challenge: string }
      calls.push(`2fa:${body.challenge}:${body.code}`)
      return json({ status: 'signed_in', email: 'linh@acme.io' })
    },
  })
})
afterEach(() => {
  unmock()
  resetSessionClock()
  calls.length = 0
})

const seedMe = (qc: QueryClient) =>
  qc.setQueryData(meQueryKey, { user: { email: 'linh@acme.io', name: 'Linh Nguyen' }, workspaces: [], instance: {} } as never)

describe('SessionGuard', () => {
  it('warns two minutes before an idle timeout and extends the session', async () => {
    lastSeen = Date.now() - (12 * HOUR - 90_000)
    await renderInWorkspace(<SessionGuard />, { seed: seedMe })
    const notice = await screen.findByRole('alert')
    expect(notice).toHaveTextContent('Session ending in 1 minute')
    fireEvent.click(screen.getByRole('button', { name: 'Extend 12 hours' }))
    await waitFor(() => expect(calls).toContain('extend'))
    await waitFor(() => expect(screen.queryByText(/Session ending/)).not.toBeInTheDocument())
    expect(await screen.findByText('Session extended')).toBeInTheDocument()
  })

  it('stays quiet while the session has hours left', async () => {
    lastSeen = Date.now()
    await renderInWorkspace(<SessionGuard />, { seed: seedMe })
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument())
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('asks for the password over the page when the session has ended, then the second factor', async () => {
    lastSeen = Date.now()
    await renderInWorkspace(
      <>
        <textarea aria-label="Draft" defaultValue="Unsaved paragraph" />
        <SessionGuard />
      </>,
      { seed: seedMe },
    )
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    act(() => {
      expect(handleAuthProblem(new ApiError(401, { status: 401, code: 'session_expired' } as never))).toBe(true)
    })
    const dialog = await screen.findByRole('dialog', { name: 'Sign in again to keep working' })
    expect(dialog).toHaveTextContent('Session ended')
    expect(dialog).toHaveTextContent('Your session ended after 12 hours without activity, as set by your instance.')

    // Wrong password: the reason, next to the field.
    fireEvent.change(screen.getByLabelText('Password for linh@acme.io'), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByText("That password doesn't match linh@acme.io.")).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Password for linh@acme.io'), { target: { value: 'reputecast-demo' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    const code = await screen.findByLabelText('Code from your authenticator app')
    fireEvent.change(code, { target: { value: '123 456' } })
    fireEvent.click(screen.getByRole('button', { name: 'Verify' }))
    await waitFor(() => expect(calls).toContain('2fa:challenge-123456:123456'))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Signed in again')).toBeInTheDocument()
    // The work behind the dialog was never unmounted.
    expect(screen.getByLabelText('Draft')).toHaveValue('Unsaved paragraph')
  })
})
