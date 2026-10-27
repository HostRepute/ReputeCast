import { screen, waitFor } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { ReadinessIssues, ReadinessRows } from './ComposerReadiness'

const path = 'GET /api/v1/workspaces/acme-newsletter/campaigns/c142/readiness/composer'

const rule = {
  key: 'rules.r-utm',
  group: 'rules',
  status: 'warn',
  title: '2 of 14 links have no utm_campaign',
  flatTitle: '',
  label: 'Links need UTM tags',
  detail: 'A workspace rule Marcus Chen added in Sending rules.',
  fixes: [],
}
const listing = {
  key: 'reputation.ip_blocklists.203.0.113.25',
  group: 'reputation',
  status: 'warn',
  title: 'Postal IP 203.0.113.25 is on 1 of 84 IP blocklists (UCEPROTECT Level 3)',
  flatTitle: '',
  detail: 'Low impact: few mailbox providers use that list.',
  fixes: [],
}

describe("The composer's Checks: Send Readiness's rules and reputation (Composer board)", () => {
  afterEach(() => mockApi({}))

  it('adds a row for each with its warnings, and a card that opens Review & send', async () => {
    mockApi({
      [path]: () =>
        json({
          ranAt: '2026-10-13T13:00:00Z',
          checkedAt: '2026-10-13T13:00:00Z',
          rules: { warnings: 1, blocks: 0, checks: [rule] },
          reputation: { warnings: 1, blocks: 0, checks: [listing] },
        }),
    })
    await renderInWorkspace(
      <>
        <ReadinessRows slug="acme-newsletter" campaignId="c142" />
        <ReadinessIssues slug="acme-newsletter" campaignId="c142" />
      </>,
    )
    expect(await screen.findByText('Links need UTM tags')).toBeInTheDocument()
    expect(screen.getAllByText('Workspace rules')).toHaveLength(2)
    expect(screen.getAllByText('Sender reputation')).toHaveLength(2)
    expect(screen.getByText(/^From Review & send · blocklists checked /)).toBeInTheDocument()
    expect(screen.getAllByText('1 warning')).toHaveLength(2)
    expect(screen.getByText('2 of 14 links have no utm_campaign')).toBeInTheDocument()
    expect(screen.getByText(listing.title)).toBeInTheDocument()
    const open = screen.getAllByRole('link', { name: 'Open in Review & send' })
    expect(open).toHaveLength(2)
    expect(open[0]).toHaveAttribute('href', '/w/acme-newsletter/campaigns/c142/review')
  })

  it('shows nothing before the checks ever ran, and a pass when a group is clean', async () => {
    let asked = 0
    mockApi({
      [path]: () => {
        asked++
        return json({})
      },
    })
    const first = await renderInWorkspace(<ReadinessRows slug="acme-newsletter" campaignId="c142" />)
    await waitFor(() => expect(asked).toBe(1))
    expect(screen.queryByText('Sender reputation')).not.toBeInTheDocument()
    expect(screen.queryByText('Workspace rules')).not.toBeInTheDocument()
    first.unmount()

    mockApi({ [path]: () => json({ ranAt: '2026-10-13T13:00:00Z', reputation: { warnings: 0, blocks: 0, checks: [] } }) })
    await renderInWorkspace(<ReadinessRows slug="acme-newsletter" campaignId="c142" />)
    expect(await screen.findByText('Sender reputation')).toBeInTheDocument()
    expect(screen.getByText('Pass')).toBeInTheDocument()
    expect(screen.queryByText('Workspace rules')).not.toBeInTheDocument()
  })
})
