import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { Link } from '@tanstack/react-router'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import gifts from './__fixtures__/gifts.json'
import offerNew from './__fixtures__/offer-new.json'
import plans from './__fixtures__/plans.json'
import revenue from './__fixtures__/revenue.json'
import status from './__fixtures__/status.json'
import { GiftsPage } from './gifts/GiftsPage'
import { OfferEditorPage } from './offers/OfferEditor'
import { RevenueOverview } from './revenue/RevenueOverview'

const EARN = '/api/v1/workspaces/acme-newsletter/earn'

// Review 2026-11 EA-6: the Earn editors ask before dropping unsaved edits,
// and the exports' buttons say who can export.
describe('Earn editors and exports', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('asks before leaving a new offer with a name typed', async () => {
    unmock = mockApi({
      [`GET ${EARN}/offers/new`]: () => json(offerNew),
      [`GET ${EARN}/status`]: () => json(status),
      [`GET ${EARN}/plans`]: () => json(plans),
    })
    const { router } = await renderInWorkspace(
      <>
        <OfferEditorPage />
        <Link to="/w/$workspace/earn/plans" params={{ workspace: 'acme-newsletter' }}>
          Elsewhere
        </Link>
      </>,
      { path: '/w/acme-newsletter/earn/plans/offers/new', me: workspaceMe({ permissions: ['earn:view', 'earn:manage'] }) },
    )
    fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'Black Friday' } })
    fireEvent.click(screen.getByRole('link', { name: 'Elsewhere' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent("Your changes to this offer aren't saved.")
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/earn/plans/offers/new')
  })

  it('leaves without asking when nothing changed', async () => {
    unmock = mockApi({
      [`GET ${EARN}/offers/new`]: () => json(offerNew),
      [`GET ${EARN}/status`]: () => json(status),
      [`GET ${EARN}/plans`]: () => json(plans),
    })
    const { router } = await renderInWorkspace(
      <>
        <OfferEditorPage />
        <Link to="/w/$workspace/earn/plans" params={{ workspace: 'acme-newsletter' }}>
          Elsewhere
        </Link>
      </>,
      { path: '/w/acme-newsletter/earn/plans/offers/new', me: workspaceMe({ permissions: ['earn:view', 'earn:manage'] }) },
    )
    await screen.findByLabelText('Name')
    fireEvent.click(screen.getByRole('link', { name: 'Elsewhere' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/earn/plans'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('exports gifts as CSV, and says why it can’t without the permission', async () => {
    let exported = 0
    unmock = mockApi({
      [`GET ${EARN}/gifts`]: () => json(gifts),
      [`GET ${EARN}/status`]: () => json(status),
      [`GET ${EARN}/gifts/export`]: () => {
        exported++
        return new Response('bought_at,buyer\n', { headers: { 'Content-Type': 'text/csv', 'Content-Disposition': 'attachment; filename="gifts.csv"' } })
      },
    })
    const viewer = await renderInWorkspace(<GiftsPage />, { me: workspaceMe({ permissions: ['earn:view', 'subscribers:view'] }) })
    const locked = await screen.findByRole('button', { name: 'Export CSV' })
    expect(locked).toBeDisabled()
    expect(locked).toHaveAttribute('title', "Exporting contacts needs a permission you don't have.")
    expect(locked.getAttribute('title')).not.toContain('subscribers:export')
    viewer.unmount()

    await renderInWorkspace(<GiftsPage />, { me: workspaceMe({ permissions: ['earn:view', 'subscribers:view', 'subscribers:export'] }) })
    const button = await screen.findByRole('button', { name: 'Export CSV' })
    expect(button).toBeEnabled()
    URL.createObjectURL = () => 'blob:gifts'
    URL.revokeObjectURL = () => {}
    fireEvent.click(button)
    await waitFor(() => expect(exported).toBe(1))
    expect(await screen.findByText('Exported as CSV')).toBeInTheDocument()
  })

  // Review 2026-11 EA-4: growth of paid members by tier and billing period.
  it('shows who joined and left each tier and period in the period', async () => {
    unmock = mockApi({
      [`GET ${EARN}/revenue`]: () => json(revenue),
      [`GET ${EARN}/status`]: () => json(status),
    })
    await renderInWorkspace(<RevenueOverview />, { me: workspaceMe({ permissions: ['earn:view', 'reports:view'] }) })
    const table = await screen.findByRole('table', { name: 'By tier and period' })
    expect(within(table).getByRole('columnheader', { name: 'Joined · left, 30 days' })).toBeInTheDocument()
    expect(within(table).getByText('12 joined and 3 left in the last 30 days')).toBeInTheDocument()
    // The total adds the rows up.
    expect(within(table).getByText('17 joined and 4 left in the last 30 days')).toBeInTheDocument()
  })
})
