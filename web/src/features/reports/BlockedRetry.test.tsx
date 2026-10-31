import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { BlockedRetry } from './ReportPage'

type Retry = Schemas['ReportBlockedRetry']

const retry = (o: Partial<Retry>): Retry => ({ blocked: 11, recipients: 9, ...o })

describe('Retry after fixing', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('drafts a retry to the blocked still on the list, and opens it in Review & send', async () => {
    let posted = ''
    unmock = mockApi({
      'POST /api/v1/workspaces/acme-newsletter/campaigns/c-1/retry': (r) => {
        posted = new URL(r.url).pathname
        return json({ id: 'c-2', name: 'October Product Update · retry', status: 'draft' }, 201)
      },
    })
    const onClose = vi.fn()
    const { router } = await renderInWorkspace(<BlockedRetry retry={retry({})} campaignId="c-1" onClose={onClose} />, {
      path: '/w/acme-newsletter/campaigns/c-1/report',
    })
    expect(screen.getByRole('heading', { name: 'Retry after fixing' })).toBeInTheDocument()
    expect(
      screen.getByText(
        'Once the block is fixed, send this issue again to the 9 blocked addresses still on your list. Each gets it once, and it goes out from Review & send with its checks. 2 more unsubscribed or were suppressed since, and are left out.',
      ),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry 9 after fixing' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns/c-2/review'))
    expect(posted).toBe('/api/v1/workspaces/acme-newsletter/campaigns/c-1/retry')
    expect(onClose).toHaveBeenCalled()
    expect(screen.getByText('Retry drafted: October Product Update · retry')).toBeInTheDocument()
  })

  it("can't be drafted without the right to edit campaigns, or when nobody is left on the list", async () => {
    unmock = mockApi({})
    const { unmount } = await renderInWorkspace(<BlockedRetry retry={retry({})} campaignId="c-1" reason="Your role can't edit campaigns." onClose={() => {}} />)
    expect(screen.getByRole('button', { name: 'Retry 9 after fixing' })).toHaveAttribute('aria-disabled', 'true')
    unmount()
    await renderInWorkspace(<BlockedRetry retry={retry({ blocked: 3, recipients: 0 })} campaignId="c-1" onClose={() => {}} />)
    const none = screen.getByRole('button', { name: 'Retry 0 after fixing' })
    expect(none).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(none)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('names the retry made already, or why there is none', async () => {
    const { unmount } = await renderInWorkspace(
      <BlockedRetry retry={retry({ campaign: { id: 'c-2', name: 'October Product Update · retry', status: 'draft' } })} campaignId="c-1" onClose={() => {}} />,
    )
    expect(screen.getByText('The retry is a draft: October Product Update · retry. Send it from Review & send once the block is fixed.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open in Review & send' })).toHaveAttribute('href', '/w/acme-newsletter/campaigns/c-2/review')
    expect(within(screen.getByRole('region', { name: 'Retry after fixing' })).queryByRole('button')).not.toBeInTheDocument()
    unmount()
    const sent = await renderInWorkspace(
      <BlockedRetry
        retry={retry({ campaign: { id: 'c-2', name: 'October Product Update · retry', status: 'sent', startedAt: '2026-10-14T09:00:00Z' } })}
        campaignId="c-1"
        onClose={() => {}}
      />,
    )
    expect(screen.getByRole('link', { name: 'Open its report' })).toHaveAttribute('href', '/w/acme-newsletter/campaigns/c-2/report')
    sent.unmount()
    const ofRetry = await renderInWorkspace(
      <BlockedRetry retry={retry({ blocked: 0, recipients: 0, unavailable: 'is_retry', of: { id: 'c-0', name: 'September Product Update' } })} campaignId="c-1" onClose={() => {}} />,
    )
    expect(
      screen.getByText('This issue is a retry of September Product Update, so its blocked addresses aren’t retried again. Check the block with your provider first.'),
    ).toBeInTheDocument()
    expect(within(screen.getByRole('region', { name: 'Retry after fixing' })).queryByRole('button')).not.toBeInTheDocument()
    ofRetry.unmount()
    await renderInWorkspace(<BlockedRetry retry={retry({ blocked: 0, recipients: 0, unavailable: 'none_left' })} campaignId="c-1" onClose={() => {}} />)
    expect(screen.getByText('Nobody is left to retry: every blocked address was delivered in the end, or had a retry already.')).toBeInTheDocument()
  })
})
