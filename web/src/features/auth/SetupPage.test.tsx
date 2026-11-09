import { fireEvent, screen, waitFor } from '@testing-library/react'
import type { QueryClient } from '@tanstack/react-query'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { SetupPage } from './SetupPage'

const keys = ['password', 'second_factor', 'recovery_codes', 'instance', 'workspace', 'provider'] as const

/** Linh at step 4, instance basics, on a server whose operator pins the public URL and turned the update check off. */
const state: Schemas['SetupState'] = {
  mode: 'instance',
  current: 'instance',
  email: 'linh@acme.io',
  name: 'Linh Nguyen',
  steps: keys.map((key, i) => ({ key, label: key, done: i < 3 })),
  password: { mustChange: false, minLength: 12, breachedCheck: false },
  secondFactor: { totp: true, passkeys: [] },
  recoveryCodes: { remaining: 10 },
  instance: {
    name: 'ReputeCast',
    publicUrl: 'https://news.acme.io',
    timezone: 'UTC',
    updateCheck: true,
    confirmed: false,
    version: 'v1.4.2',
    publicUrlLocked: true,
    updateCheckAllowed: false,
  },
} as never

const line = (detail: string): Schemas['CheckLine'] => ({ status: 'ok', detail })
const check: Schemas['URLCheck'] = { checkedAt: new Date().toISOString(), dns: line('resolves'), tls: line('valid'), reachable: line('answered') }

const seed = (qc: QueryClient) => qc.setQueryData(['meta'], { instanceName: 'ReputeCast', publicUrl: 'https://news.acme.io', version: 'v1.4.2' } as never)

describe('First sign-in › instance basics', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('shows the operator’s public URL read only and the update check off', async () => {
    const saved: unknown[] = []
    unmock = mockApi({
      'GET /api/v1/setup': () => json(state),
      'POST /api/v1/setup/check-url': () => json(check),
      'PUT /api/v1/setup/instance': async (r) => {
        saved.push(await r.json())
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<SetupPage />, { path: '/setup', seed })
    const url = await screen.findByDisplayValue('https://news.acme.io')
    expect(url).toHaveAttribute('readonly')
    expect(url).toHaveAccessibleDescription(/Set by the server's operator \(http\.public_url_locked\)/)
    const box = screen.getByRole('checkbox', { name: 'Check GitHub for new releases once a day' })
    expect(box).toBeDisabled()
    expect(box).not.toBeChecked()
    expect(box).toHaveAccessibleDescription('Turned off in the server configuration (updates.check = false).')

    fireEvent.change(screen.getByLabelText('Instance name'), { target: { value: 'Acme ReputeCast' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save and create the first workspace' }))
    await waitFor(() => expect(saved).toHaveLength(1))
    expect(saved[0]).toMatchObject({ name: 'Acme ReputeCast', publicUrl: 'https://news.acme.io', updateCheck: false })
  })
})
