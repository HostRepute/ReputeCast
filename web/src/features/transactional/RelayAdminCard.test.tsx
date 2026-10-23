import { screen, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { RelayAdminCard } from './RelayAdminCard'

function relay(checks: Schemas['SMTPRelayCheck'][]): Schemas['SMTPRelaySettings'] {
  return {
    enabled: true,
    hostname: 'smtp.acme.io',
    starttlsPort: 587,
    tlsPort: 465,
    certSource: 'acme',
    certFile: '',
    keyFile: '',
    suggestedHostname: 'smtp.acme.io',
    status: { checks },
  }
}

describe('the SMTP relay card', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('lists its checks once there are some', async () => {
    unmock = mockApi({ 'GET /api/v1/admin/smtp-relay': () => json(relay([{ key: 'dns', ok: true, detail: 'smtp.acme.io points at this server' }])) })
    await renderInWorkspace(<RelayAdminCard />)
    const list = await screen.findByRole('list', { name: 'Checks' })
    expect(within(list).getAllByRole('listitem')).toHaveLength(1)
  })

  it("isn't a list before the first check", async () => {
    unmock = mockApi({ 'GET /api/v1/admin/smtp-relay': () => json(relay([])) })
    await renderInWorkspace(<RelayAdminCard />)
    await screen.findByText('Not checked yet: run the checks once DNS points here.')
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })
})
