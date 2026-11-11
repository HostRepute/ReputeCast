import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { confidenceText, HoldoutPage, liftText } from './HoldoutPage'

type Results = Schemas['HoldoutResults']

const period: Schemas['HoldoutPeriodItem'] = {
  id: '0199a000-0000-7000-8000-000000000001',
  startedAt: '2026-08-15T14:00:00Z',
  basisPoints: 500,
  percent: 5,
  campaigns: true,
  automations: true,
  open: true,
}

/** The AudienceHoldout board's results. */
const board = (patch: Partial<Results> = {}): Results => ({
  periods: [period],
  period,
  window: 'period',
  from: '2026-08-15T14:00:00Z',
  to: '2026-10-14T14:00:00Z',
  verdict: 'added',
  computedAt: '2026-10-14T08:00:00Z',
  computing: false,
  holdout: { contacts: 2316, converters: 148, orders: 164, revenueCents: 1070000 },
  marketing: { contacts: 44021, converters: 3346, orders: 3654, revenueCents: 23991400 },
  metrics: [
    { key: 'conversion_rate', holdout: 6.39, marketing: 7.6, lift: 18.9, confidence: 0.972, tested: true, significant: true },
    { key: 'revenue_per_contact', holdout: 462, marketing: 545, lift: 18.0, confidence: 0.961, tested: true, significant: true },
    { key: 'orders_per_contact', holdout: 0.0708, marketing: 0.083, lift: 17.2, confidence: 0.946, tested: true, significant: false },
  ],
  currency: 'usd',
  otherCurrencies: [],
  campaignsIncluded: 8,
  minHoldout: 1000,
  ...patch,
})

const summary = { contacts: 48213, holdout: true, lists: [], paidTiers: [], segments: 7, stripe: false, suppressions: 1874, tiers: { active: 0, atRisk: 0, cooling: 0, sunset: 0 } }
const path = '/w/acme-newsletter/audience/holdout'
const viewer = workspaceMe({ permissions: ['reports:view', 'contacts:view'] })

describe('Audience › Holdout', () => {
  afterEach(() => mockApi({}))

  it('rounds the lift and never overstates the confidence', () => {
    expect(liftText(18.04)).toBe('+18%')
    expect(liftText(-8.6)).toBe('−9%')
    expect(confidenceText(0.9649)).toBe('96%')
    expect(confidenceText(0.9999)).toBe('99%')
  })

  it('puts the verdict first, then the tiles, the comparison and the note, as AudienceHoldout does', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/holdout/results': () => json(board()),
      'GET /api/v1/workspaces/acme-newsletter/audience': () => json(summary),
    })
    await renderInWorkspace(<HoldoutPage search={{}} />, { me: viewer, path })
    expect(await screen.findByRole('heading', { name: 'Marketing added 18% more revenue per contact (96% confidence).' })).toBeInTheDocument()
    expect(screen.getByText('2,316 contacts in the holdout group (5%) got no campaigns or automations from 15 Aug to 14 Oct; 44,021 contacts got them.')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Window' })).toHaveTextContent('Since 15 AugLast 30 daysLast 90 days')
    expect(screen.getByText('+19% · 97% confidence')).toBeInTheDocument()
    expect(screen.getByText('+18% · 96% confidence')).toBeInTheDocument()
    expect(screen.getByText('+17% · 94% confidence')).toBeInTheDocument()
    expect(screen.getByText('Holdout $4.62')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Revenue per contact: marketing group $5.45, holdout group $4.62' })).toBeInTheDocument()
    expect(screen.getByText('Campaigns that included the holdout: 8')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Holdout' })).toHaveAttribute('aria-current', 'page')
    fireEvent.click(screen.getByRole('button', { name: 'Table' }))
    const table = screen.getByRole('table', { name: 'Revenue per contact, marketing group and holdout group' })
    const rows = within(table).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('Marketing group44,0213,3467.6%3,654$239,914$5.45')
    expect(rows[2]).toHaveTextContent('Holdout group2,3161486.4%164$10,700$4.62')
  })

  it('says why there is no verdict yet', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/holdout/results': () => json(board({ verdict: 'too_few', holdout: { contacts: 640, converters: 3, orders: 3, revenueCents: 1000 } })),
      'GET /api/v1/workspaces/acme-newsletter/audience': () => json(summary),
    })
    await renderInWorkspace(<HoldoutPage search={{}} />, { me: viewer, path })
    expect(await screen.findByRole('heading', { name: 'Too few contacts in the holdout: 640 of 1,000 needed.' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Settings › Sending rules' })).toBeInTheDocument()
    expect(screen.queryByText('Conversion rate')).not.toBeInTheDocument()
  })

  it('offers to set one up when the workspace never had a holdout group', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/holdout/results': () => json(board({ verdict: 'none', periods: [], period: undefined })),
      'GET /api/v1/workspaces/acme-newsletter/audience': () => json({ ...summary, holdout: false }),
    })
    await renderInWorkspace(<HoldoutPage search={{}} />, { me: viewer, path })
    expect(await screen.findByRole('heading', { name: 'No holdout group yet.' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Set up a holdout group' })).toHaveAttribute('href', '/w/acme-newsletter/settings/sending-rules')
  })

  it('says when a recompute comes too soon', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/holdout/results': () => json(board()),
      'GET /api/v1/workspaces/acme-newsletter/audience': () => json(summary),
      'POST /api/v1/workspaces/acme-newsletter/holdout/measure': () =>
        json({ status: 429, title: 'Too Many Requests', detail: 'You can recompute once every 10 minutes. Try again in 6 minutes.' }, 429, { 'Retry-After': '360' }),
    })
    await renderInWorkspace(<HoldoutPage search={{}} />, { me: viewer, path })
    fireEvent.click(await screen.findByRole('button', { name: 'Recompute' }))
    await waitFor(() => expect(screen.getByText(/You can recompute once every 10 minutes\./)).toBeInTheDocument())
  })
})
