import { screen, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { PlacementReviewCard } from './EntryPoints'

type Row = Schemas['PlacementTestRow']
type Counts = Schemas['PlacementCounts']

const counts = (patch: Partial<Counts>): Counts => ({
  copies: 10,
  failed: 0,
  found: 10,
  inbox: 0,
  notArrived: 0,
  notChecked: 0,
  notCheckedSeeds: [],
  spam: 0,
  tab: 0,
  tabs: [],
  unsent: 0,
  waiting: 0,
  ...patch,
})

const provider = (id: string, short: string, c: Counts) =>
  ({ id, short, name: short, adapter: id.slice(2), color: '', cost: 0, detail: '', status: 'ok', counts: c }) as unknown as Row['providers'][number]

/** The SendReadiness board's last result: #142's test of Sun 12 Oct through both routes. */
const row = {
  id: 't-142',
  name: 'Weekly Digest #142',
  contentKind: 'campaign',
  contentId: 'c142',
  contentName: 'Weekly Digest #142',
  contentNote: 'draft',
  origin: 'review',
  personalization: '',
  status: 'finished',
  through: 'compare',
  startedAt: '2026-10-12T13:05:00Z',
  startedBy: 'Linh Nguyen',
  counts: counts({ inbox: 13, tab: 4, spam: 2, notArrived: 1, copies: 20, found: 19 }),
  providers: [provider('p-ses', 'SES', counts({ inbox: 8, tab: 2 })), provider('p-postal', 'Postal', counts({ inbox: 5, tab: 2, spam: 2, notArrived: 1, found: 9 }))],
} as unknown as Row

const list = { attention: [], canManage: true, canRun: true, deleteAfter: '24h', manageHelpers: [], providers: [], runHelpers: [], seedsConnected: 10, seedsTotal: 10, starters: [], tests: [row] }
const seed = (id: string, providerName: string) => ({ id, providerName, status: 'connected' })
const seeds = {
  attention: 0,
  canManage: true,
  canRun: true,
  connected: 10,
  deleteAfter: '24h',
  manageHelpers: [],
  oauthGoogle: true,
  oauthMicrosoft: true,
  providers: [],
  redirectUri: '',
  seeds: [
    seed('s1', 'Gmail'),
    seed('s2', 'Gmail'),
    seed('s3', 'Outlook'),
    seed('s4', 'Outlook'),
    seed('s5', 'Yahoo'),
    seed('s6', 'Yahoo'),
    seed('s7', 'iCloud'),
    seed('s8', 'GMX'),
    seed('s9', 'Web.de'),
    seed('s10', 'iCloud'),
  ],
}

const me = workspaceMe({ permissions: ['campaigns:view', 'deliverability:view', 'placement:run'] })

describe('Review & send · placement test (SR-9)', () => {
  afterEach(() => mockApi({}))

  it("shows the seed mailboxes and the campaign's last result per sending provider", async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/placement/tests': () => json(list),
      'GET /api/v1/workspaces/acme-newsletter/placement/seeds': () => json(seeds),
    })
    await renderInWorkspace(<PlacementReviewCard campaignId="c142" campaignName="Weekly Digest #142" providers={['p-ses', 'p-postal']} />, { me })
    const card = await screen.findByRole('region', { name: 'Placement test' })
    expect(await within(card).findByText('10 seed mailboxes (Gmail, Outlook, Yahoo, iCloud, GMX, Web.de) · about 3 min')).toBeInTheDocument()
    expect(within(card).getByText('Optional')).toBeInTheDocument()
    expect(within(card).getByText(/^Last result · .* · Weekly Digest #142$/)).toBeInTheDocument()
    expect(within(card).getByText('SES · Inbox 8 · Tab 2 · Spam 0')).toBeInTheDocument()
    expect(within(card).getByText('Postal · Inbox 5 · Tab 2 · Spam 2 · Not arrived 1')).toBeInTheDocument()
    expect(within(card).getByRole('link', { name: 'View results' })).toHaveAttribute('href', '/w/acme-newsletter/deliverability/placement/t-142')
    expect(within(card).getByRole('button', { name: 'Run placement test' })).toBeEnabled()
  })

  it('says when the campaign has no test yet, and which seed mailboxes a test skips', async () => {
    const failed = { ...seeds, seeds: seeds.seeds.map((s) => (s.id === 's10' ? { ...s, status: 'failed' } : s)) }
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/placement/tests': () => json({ ...list, tests: [] }),
      'GET /api/v1/workspaces/acme-newsletter/placement/seeds': () => json(failed),
    })
    await renderInWorkspace(<PlacementReviewCard campaignId="c142" campaignName="Weekly Digest #142" providers={['p-ses']} />, { me })
    const card = await screen.findByRole('region', { name: 'Placement test' })
    expect(await within(card).findByText('9 seed mailboxes (Gmail, Outlook, Yahoo, iCloud, GMX, Web.de) · 1 more needs attention · about 3 min')).toBeInTheDocument()
    expect(within(card).getByText('Not run for this campaign')).toBeInTheDocument()
    expect(within(card).queryByRole('link', { name: 'View results' })).not.toBeInTheDocument()
  })
})
