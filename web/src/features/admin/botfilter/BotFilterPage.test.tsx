import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { QueryClient } from '@tanstack/react-query'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { meQueryKey } from '@/lib/session'
import { BotFilterPage } from './BotFilterPage'

type Network = Schemas['TrackingNetwork']
type Agent = Schemas['TrackingUserAgent']

const at = '2026-10-14T09:12:00Z'
const net = (o: Partial<Network>): Network => ({ cidr: '17.0.0.0/8', name: 'Apple Mail Privacy Protection', class: 'privacy', source: 'builtin', note: '', createdAt: at, ...o })
const ua = (o: Partial<Agent>): Agent => ({ pattern: 'barracuda', kind: 'scanner', name: 'Barracuda', source: 'builtin', note: '', createdAt: at, ...o })

const networks = [
  net({}),
  net({ cidr: '40.92.0.0/15', name: 'Microsoft Defender', class: 'scanner' }),
  net({ cidr: '203.0.113.0/24', name: 'Our mail gateway', class: 'scanner', source: 'admin', note: 'Follows every link in inbound mail', createdByName: 'Linh Nguyen' }),
]
const agents = [
  ua({}),
  ua({ pattern: 'curl/', kind: 'script', name: '' }),
  ua({ pattern: 'cubot', kind: 'person', name: 'CUBOT phones' }),
  ua({ pattern: 'acmegate-linkcheck', name: 'Our mail gateway', source: 'admin', createdByName: 'Linh Nguyen' }),
]

/** Linh, an instance admin. */
const admin = (qc: QueryClient) => {
  qc.setQueryData(meQueryKey, {
    user: { id: 'u-1', name: 'Linh Nguyen', email: 'linh@acme.io', isInstanceAdmin: true },
    workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleName: 'Owner' }],
    instance: { name: 'Acme ReputeCast', version: 'v1.4.2' },
  } as never)
  qc.setQueryData(['meta'], { instanceName: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.4.2' } as never)
}

describe('the bot filter', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('lists what admins added, and folds the built-in entries', async () => {
    unmock = mockApi({
      'GET /api/v1/admin/tracking-networks': () => json({ items: networks }),
      'GET /api/v1/admin/tracking-user-agents': () => json({ items: agents }),
    })
    await renderInWorkspace(<BotFilterPage />, { path: '/w/acme-newsletter/admin/bot-filter', seed: admin })
    expect(await screen.findByRole('heading', { name: 'Networks · 3' })).toBeInTheDocument()
    const added = screen.getByRole('table', { name: 'Networks admins added' })
    expect(within(added).getAllByRole('row')).toHaveLength(2)
    expect(within(added).getByText('203.0.113.0/24')).toBeInTheDocument()
    expect(within(added).getByText('Mail scanner')).toBeInTheDocument()
    // The built-in ones are folded under one line naming who runs them.
    expect(screen.getByText('Built-in · 2 networks')).toBeInTheDocument()
    expect(screen.getByText('Apple Mail Privacy Protection and Microsoft Defender · they come with ReputeCast releases')).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'User agents · 4' })).toBeInTheDocument()
    const agentsAdded = screen.getByRole('table', { name: 'User agents admins added' })
    expect(within(agentsAdded).getByText('acmegate-linkcheck')).toBeInTheDocument()
    expect(screen.getByText('Built-in · 3 user agents')).toBeInTheDocument()
    expect(screen.getByText(/and CUBOT phones as people’s · they come with ReputeCast releases$/)).toBeInTheDocument()
    // Only what an admin added can be removed.
    expect(screen.getByRole('button', { name: 'Remove 203.0.113.0/24' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Remove 17.0.0.0/8' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Remove barracuda' })).not.toBeInTheDocument()
    // Every change is in the instance audit log.
    expect(screen.getByRole('link', { name: 'audit log' })).toHaveAttribute('href', '/w/acme-newsletter/admin/audit')
  })

  it("adds a person's device, and shows the server's reason on its field", async () => {
    const posted: unknown[] = []
    let list = agents
    unmock = mockApi({
      'GET /api/v1/admin/tracking-networks': () => json({ items: networks }),
      'GET /api/v1/admin/tracking-user-agents': () => json({ items: list }),
      'POST /api/v1/admin/tracking-user-agents': async (r) => {
        const body = (await r.json()) as { pattern: string; kind: string; name: string }
        posted.push(body)
        if (body.pattern === 'Chrome') {
          return json(
            {
              status: 422,
              title: 'Unprocessable Entity',
              detail: '“chrome” is in the user agents of browsers and mail clients people read email with, so it would take them for machines.',
              errors: [{ location: 'body.pattern', message: 'x' }],
            },
            422,
          )
        }
        const added = ua({ pattern: body.pattern.toLowerCase(), kind: 'person', name: body.name, source: 'admin', createdByName: 'Linh Nguyen' })
        list = [...agents, added]
        return json(added, 201)
      },
    })
    await renderInWorkspace(<BotFilterPage />, { path: '/w/acme-newsletter/admin/bot-filter', seed: admin })
    fireEvent.click(await screen.findByRole('button', { name: 'Add user agent' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add a user agent' })
    const pattern = within(dialog).getByRole('textbox', { name: 'Text in the user agent' })
    fireEvent.change(pattern, { target: { value: 'Chrome' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Who or what it is' }), { target: { value: 'Our link checker' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add user agent' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/would take them for machines/)
    expect(pattern).toHaveAttribute('aria-invalid', 'true')
    // A person's device: one word the rules take for a machine's.
    fireEvent.click(within(dialog).getByRole('radio', { name: /A person’s device/ }))
    fireEvent.change(pattern, { target: { value: 'KioskBot' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add user agent' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(posted[1]).toEqual({ pattern: 'KioskBot', kind: 'person', name: 'Our link checker' })
    expect(await screen.findByText('kioskbot')).toBeInTheDocument()
    expect(screen.getByText('“kioskbot” added to the bot filter')).toBeInTheDocument()
  })

  it('removes a network an admin added', async () => {
    let removed = ''
    let list = networks
    unmock = mockApi({
      'GET /api/v1/admin/tracking-networks': () => json({ items: list }),
      'GET /api/v1/admin/tracking-user-agents': () => json({ items: agents }),
      'DELETE /api/v1/admin/tracking-networks': (r) => {
        removed = new URL(r.url).searchParams.get('cidr') ?? ''
        list = networks.filter((n) => n.cidr !== removed)
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<BotFilterPage />, { path: '/w/acme-newsletter/admin/bot-filter', seed: admin })
    fireEvent.click(await screen.findByRole('button', { name: 'Remove 203.0.113.0/24' }))
    const dialog = await screen.findByRole('dialog', { name: 'Remove 203.0.113.0/24?' })
    expect(within(dialog).getByText('Linh Nguyen on Oct 14, 2026 · Our mail gateway')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(removed).toBe('203.0.113.0/24'))
    expect(await screen.findByText('No network added yet: the built-in ones below apply.')).toBeInTheDocument()
  })
})
