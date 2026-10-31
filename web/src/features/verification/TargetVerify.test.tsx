import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { DeleteSegmentButton } from '@/features/conditional/usage'
import { TargetVerify } from './HealthEntry'

const ws = '/api/v1/workspaces/acme-newsletter'
const verifier = workspaceMe({ permissions: ['subscribers:view', 'subscribers:verify', 'segments:edit'] })
const notConnected = { connected: false, canManage: false, canVerify: true, manageHelpers: [], verifyHelpers: [], services: [], contacts: 0, cost: 0, enough: false, kind: '', label: '', price: 0, skipRecent: true, skipped: 0, toCheck: 0 }

describe('Verify addresses from segments and growth (review 2026-11 AU-13)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it.each([
    ['a saved segment', { kind: 'segment' as const, segmentId: 'seg-1' }, 'segments' as const, 'Segments · Engaged readers'],
    ['a flagged growth source', { kind: 'selection' as const, filter: { growthSource: 'api_key:k1' } }, 'growth' as const, 'Growth · Partner API'],
  ])('opens the verify dialog for %s', async (_, target, origin, from) => {
    const asked: unknown[] = []
    unmock = mockApi({
      [`POST ${ws}/verification/estimate`]: async (r) => {
        asked.push(await r.json())
        return json(notConnected)
      },
    })
    await renderInWorkspace(<TargetVerify target={target} origin={origin} from={from} />, { me: verifier })
    fireEvent.click(screen.getByRole('button', { name: 'Verify addresses' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText(from)).toBeInTheDocument()
    await waitFor(() => expect(asked).toHaveLength(1))
    expect(asked[0]).toMatchObject({ target })
  })

  it('leaves the button out without the verify permission', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<TargetVerify target={{ kind: 'segment', segmentId: 'seg-1' }} origin="segments" from="Segments" />, {
      me: workspaceMe({ permissions: ['subscribers:view'] }),
    })
    expect(screen.queryByRole('button', { name: 'Verify addresses' })).not.toBeInTheDocument()
  })
})

describe('Deleting a segment (review 2026-11 AU-11)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('lists the campaigns, automations and segments that use it', async () => {
    unmock = mockApi({
      [`GET ${ws}/visibility/usage`]: () => json({ blocks: 0, emails: [] }),
      [`GET ${ws}/segments/seg-1/impact`]: () =>
        json({
          id: 'seg-1',
          name: 'Engaged readers',
          uses: [
            { kind: 'campaign', id: 'c1', name: 'November update' },
            { kind: 'automation', id: 'a1', name: 'Win-back' },
          ],
        }),
    })
    await renderInWorkspace(<DeleteSegmentButton segmentId="seg-1" name="Engaged readers" />, { me: verifier })
    fireEvent.click(screen.getByRole('button', { name: 'Delete segment' }))
    const dialog = await screen.findByRole('dialog')
    expect(await within(dialog).findByText('Campaign')).toBeInTheDocument()
    expect(within(dialog).getByText((_, el) => el?.tagName === 'SPAN' && el.textContent === 'Campaign · November update')).toBeInTheDocument()
    expect(within(dialog).getByText('Automation')).toBeInTheDocument()
    expect(within(dialog).getAllByRole('link', { name: 'Open' })).toHaveLength(2)
  })
})
