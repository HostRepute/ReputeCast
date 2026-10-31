import { screen } from '@testing-library/react'
import type { components } from '@/api/schema'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { CampaignsPage } from './CampaignsPage'

type Item = components['schemas']['CampaignListItem']

const sent: Item = {
  id: '0192f0a0-0000-7000-8000-000000000001',
  name: 'October digest',
  status: 'sent',
  channel: 'email',
  audience: 'All subscribers',
  blocked: false,
  notes: [],
  paidOnly: false,
  routes: [],
  createdAt: '2026-10-01T09:00:00Z',
  updatedAt: '2026-10-02T09:00:00Z',
  updatedBy: 'Aisha',
  startedAt: '2026-10-02T09:00:00Z',
  recipients: 1200,
}

function mock() {
  return mockApi({
    'GET /api/v1/workspaces/acme-newsletter/campaigns': () =>
      json({ items: [sent], total: 1, page: 1, pageSize: 25, counts: { all: 1, drafts: 0, paidOnly: 0, scheduled: 0, sending: 0, sent: 1 } }),
    'GET /api/v1/workspaces/acme-newsletter/campaigns/summary': () => json({ days: 30, campaigns: 1, sent: 1200, cost: 0, costBy: [] }),
  })
}

describe('CampaignsPage', () => {
  afterEach(() => mock()())

  // Review 2026-11 EN-7: Duplicate on a sent row opens a copy, like the report.
  it('duplicates a sent campaign from its row', async () => {
    mock()
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] }) })
    const link = await screen.findByRole('link', { name: 'Duplicate' })
    expect(link).toHaveAttribute('href', `/w/acme-newsletter/campaigns/new?from=${sent.id}`)
  })

  it('explains why a viewer cannot duplicate', async () => {
    mock()
    await renderInWorkspace(<CampaignsPage search={{}} />)
    expect(await screen.findByRole('button', { name: 'Duplicate' })).toHaveAttribute('aria-disabled', 'true')
    expect(screen.queryByRole('link', { name: 'Duplicate' })).not.toBeInTheDocument()
  })
})
