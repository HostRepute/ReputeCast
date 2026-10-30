import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { SendingIPsFact, splitIPs } from './SendingIPs'

type Card = Schemas['ProviderCard']

const rate = { limit: 5, stale: false }
const card = (patch: Partial<Card>): Card => ({
  id: 'p-1',
  adapter: 'postal',
  name: 'Postal',
  short: 'Postal',
  label: 'self-hosted · mta.acme.io',
  color: 'postal',
  status: 'healthy',
  roles: [],
  ipMode: 'dedicated',
  ips: ['203.0.113.25'],
  ipSource: { from: 'manual', editable: true, setBy: 'Linh Nguyen', setAt: '2025-10-03T09:00:00Z' },
  webhook: { kind: 'Registered' },
  accountKind: 'server',
  account: 'Postal 3.3.4 · reachable',
  bounce: rate,
  complaint: rate,
  quota: { kind: 'none', max: 0, used: 0, fromFailover: 0 },
  rate: { max: 0, now: 0, peakToday: 0, throttled: false },
  cost: { basis: 'self_hosted', toDate: 0 },
  capabilities: [],
  takingFailover: [],
  connectedAt: '2025-10-03T08:48:00Z',
  credentials: [],
  ...patch,
})

const ses = (source: Partial<Card['ipSource']>): Card =>
  card({
    id: 'p-ses',
    adapter: 'ses',
    name: 'Amazon SES',
    short: 'SES',
    color: 'ses',
    ipMode: 'shared',
    ips: [],
    ipSource: { from: 'provider', editable: false, ...source },
  })

const manager = workspaceMe({
  permissions: ['providers:view', 'providers:manage'],
  workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' },
})
const hoursAgo = (h: number) => new Date(Date.now() - h * 3600_000).toISOString()
const ipsPath = 'PUT /api/v1/workspaces/acme-newsletter/providers/p-1/ips'

describe('Sending IPs (review GS-10)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('says where the IPs of a provider that reports them come from, with nothing to edit', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<SendingIPsFact card={ses({ checkedAt: hoursAgo(5) })} />, { me: manager })
    expect(screen.getByText('Sending IPs')).toBeInTheDocument()
    expect(screen.getByText('Shared pool · checked at domain level')).toBeInTheDocument()
    expect(screen.getByText('From Amazon SES, checked daily · 5 h ago')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('says why a check failed, keeps the last list and checks again', async () => {
    let checks = 0
    unmock = mockApi({
      'POST /api/v1/workspaces/acme-newsletter/providers/p-ses/ips/check': () => {
        checks++
        return json(ses({ checkedAt: new Date().toISOString() }))
      },
    })
    const failed = ses({ checkedAt: hoursAgo(30), error: "The role or keys ReputeCast uses can't list the account's dedicated IPs.", errorAt: hoursAgo(5) })
    await renderInWorkspace(<SendingIPsFact card={{ ...failed, ipMode: 'dedicated', ips: ['54.240.8.12', '54.240.8.13'] }} />, { me: manager })
    expect(screen.getByText('54.240.8.12, 54.240.8.13 · dedicated')).toBeInTheDocument()
    expect(screen.getByText("Couldn't check with Amazon SES 5 h ago: The role or keys ReputeCast uses can't list the account's dedicated IPs.")).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: "Check again for Amazon SES's sending IPs" }))
    await waitFor(() => expect(checks).toBe(1))
    expect(await screen.findByText('Amazon SES lists no dedicated IPs: the shared pool')).toBeInTheDocument()
  })

  it('lets people who manage providers set a self-hosted server’s IPs by hand', async () => {
    let sent: unknown
    unmock = mockApi({
      [ipsPath]: async (r) => {
        sent = await r.json()
        return json(card({ ips: ['203.0.113.25', '203.0.113.26'], ipSource: { from: 'manual', editable: true, setBy: 'Linh Nguyen', setAt: new Date().toISOString() } }))
      },
    })
    await renderInWorkspace(<SendingIPsFact card={card({})} />, { me: manager })
    expect(screen.getByText('203.0.113.25 · dedicated')).toBeInTheDocument()
    expect(screen.getByText('Set by Linh Nguyen · Fri 3 Oct')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: "Edit Postal's sending IPs" }))
    const dialog = await screen.findByRole('dialog', { name: "Postal's sending IPs" })
    expect(within(dialog).getByRole('radio', { name: /^Dedicated IPs/ })).toBeChecked()
    const field = within(dialog).getByRole('textbox', { name: 'IP addresses' })
    expect(field).toHaveValue('203.0.113.25')
    fireEvent.change(field, { target: { value: '203.0.113.26\n203.0.113.25, ' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save IPs' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(sent).toEqual({ mode: 'dedicated', ips: ['203.0.113.26', '203.0.113.25'] })
    expect(screen.getByText("Postal's sending IPs saved")).toBeInTheDocument()
  })

  it('shows what the server refuses next to the field, and asks for an address before saving none', async () => {
    let puts = 0
    unmock = mockApi({
      [ipsPath]: () => {
        puts++
        return json(
          {
            status: 422,
            title: 'Unprocessable Entity',
            detail: '10.0.0.5 is a private or reserved address. Enter the public IP your mail leaves from, the one blocklists and mailbox providers see.',
            errors: [{ location: 'body.ips[1]', message: '10.0.0.5 is a private or reserved address.' }],
          },
          422,
        )
      },
    })
    await renderInWorkspace(<SendingIPsFact card={card({})} />, { me: manager })
    fireEvent.click(screen.getByRole('button', { name: "Edit Postal's sending IPs" }))
    const dialog = await screen.findByRole('dialog')
    const field = within(dialog).getByRole('textbox', { name: 'IP addresses' })
    fireEvent.change(field, { target: { value: ' \n ' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save IPs' }))
    expect(within(dialog).getByRole('alert')).toHaveTextContent('Enter at least one IP address, or choose Shared pool.')
    expect(field).toHaveAttribute('aria-invalid', 'true')
    expect(puts).toBe(0)

    fireEvent.change(field, { target: { value: '203.0.113.25\n10.0.0.5' } })
    expect(within(dialog).queryByRole('alert')).not.toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save IPs' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('10.0.0.5 is a private or reserved address.')
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('sets the shared pool, with no IPs', async () => {
    let sent: unknown
    unmock = mockApi({
      [ipsPath]: async (r) => {
        sent = await r.json()
        return json(card({ ipMode: 'shared', ips: [] }))
      },
    })
    await renderInWorkspace(<SendingIPsFact card={card({})} />, { me: manager })
    fireEvent.click(screen.getByRole('button', { name: "Edit Postal's sending IPs" }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('radio', { name: /^Shared pool/ }))
    expect(within(dialog).queryByRole('textbox')).not.toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save IPs' }))
    await waitFor(() => expect(sent).toEqual({ mode: 'shared', ips: [] }))
    expect(await screen.findByText(/Shared pool: checked at domain level\./)).toBeInTheDocument()
  })

  it('says a provider nobody set reports nothing, and keeps Edit from people who can’t manage providers', async () => {
    unmock = mockApi({})
    await renderInWorkspace(
      <SendingIPsFact card={card({ name: 'Postmark', ipMode: 'shared', ips: [], ipSource: { from: 'none', editable: true } })} />,
      { me: workspaceMe({ permissions: ['providers:view'] }) },
    )
    expect(screen.getByText("Postmark doesn't report them")).toBeInTheDocument()
    const edit = screen.getByRole('button', { name: "Edit Postmark's sending IPs" })
    expect(edit).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(edit)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('reads pasted lists one entry at a time', () => {
    expect(splitIPs(' 203.0.113.25,203.0.113.26;\n\n2001:db8::25 ')).toEqual(['203.0.113.25', '203.0.113.26', '2001:db8::25'])
  })
})
