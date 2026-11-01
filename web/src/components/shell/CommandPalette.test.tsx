import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { GlobalShortcuts } from './Shortcuts'
import { destOf, detailOf } from './CommandPalette'

const results = {
  query: 'week',
  groups: [
    {
      kind: 'campaigns',
      items: [
        { id: 'c-1', title: 'Weekly Digest #143', status: 'draft', channel: 'email' },
        { id: 'c-2', title: 'Weekly Digest #142', status: 'sent', channel: 'email', count: 31192, at: '2026-10-29T09:00:00Z' },
      ],
    },
    { kind: 'contacts', items: [{ id: 'p-1', title: 'priya.weeks@example.net', detail: 'Priya Weeks', status: 'active' }] },
  ],
}

const pressModK = () => fireEvent.keyDown(window, { key: 'k', metaKey: true, ctrlKey: true })

describe('the command palette', () => {
  let unmock = () => {}
  afterEach(() => {
    unmock()
    window.localStorage.clear()
  })

  it('opens on ⌘K, searches as you type, and opens the highlighted result', async () => {
    const asked: string[] = []
    unmock = mockApi({
      'GET /api/v1/workspaces/acme-newsletter/search': (req) => {
        asked.push(new URL(req.url).searchParams.get('q') ?? '')
        return json(results)
      },
    })
    const { router } = await renderInWorkspace(<GlobalShortcuts />, {
      path: '/w/acme-newsletter',
      me: workspaceMe({ permissions: ['campaigns:view', 'subscribers:view', 'campaigns:edit'] }),
    })
    pressModK()
    const box = await screen.findByRole('combobox', { name: 'Search campaigns, contacts, automations, lists, forms, pages and settings' })
    expect(box).toHaveFocus()
    // Opened: places the role can open, starting with New campaign.
    expect(screen.getByRole('group', { name: 'Go to' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: /New campaign/ })).toHaveAttribute('aria-selected', 'true')

    fireEvent.change(box, { target: { value: 'week' } })
    expect(await screen.findByRole('group', { name: 'Campaigns · 2' })).toBeInTheDocument()
    expect(asked).toEqual(['week'])
    const contacts = screen.getByRole('group', { name: 'Contacts · 1' })
    expect(within(contacts).getByText('Contact · Priya Weeks')).toBeInTheDocument()
    // ↓ moves to the second result; ↵ opens the sent campaign's report.
    fireEvent.keyDown(box, { key: 'ArrowDown' })
    expect(screen.getByRole('option', { name: /Weekly Digest #142/ })).toHaveAttribute('aria-selected', 'true')
    expect(box).toHaveAttribute('aria-activedescendant', screen.getByRole('option', { name: /Weekly Digest #142/ }).id)
    fireEvent.keyDown(box, { key: 'Enter' })
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns/c-2/report'))
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()

    // It is in Recent next time.
    pressModK()
    const recent = await screen.findByRole('group', { name: 'Recent' })
    expect(within(recent).getByText('Weekly Digest #142')).toBeInTheDocument()
  })

  it('says when nothing matches', async () => {
    unmock = mockApi({ 'GET /api/v1/workspaces/acme-newsletter/search': () => json({ query: 'wekly', groups: [] }) })
    await renderInWorkspace(<GlobalShortcuts />, { path: '/w/acme-newsletter' })
    pressModK()
    fireEvent.change(await screen.findByRole('combobox'), { target: { value: 'wekly' } })
    expect(await screen.findByText('No results for “wekly”')).toBeInTheDocument()
  })

  it("leaves Recent items out once the role can't open them", async () => {
    window.localStorage.setItem(
      'rc.palette.recent.acme-newsletter',
      JSON.stringify([{ key: 'contacts:p-1', kind: 'contacts', title: 'priya.weeks@example.net', detail: 'Contact', dest: { to: '/w/$workspace/audience/contacts/$contactId', params: { workspace: 'acme-newsletter', contactId: 'p-1' } } }]),
    )
    unmock = mockApi({})
    await renderInWorkspace(<GlobalShortcuts />, { path: '/w/acme-newsletter', me: workspaceMe({ permissions: ['campaigns:view'] }) })
    pressModK()
    await screen.findByRole('combobox')
    expect(screen.queryByRole('group', { name: 'Recent' })).not.toBeInTheDocument()
    expect(screen.queryByText('priya.weeks@example.net')).not.toBeInTheDocument()
  })

  it('opens a campaign where its status goes', () => {
    const at = (status: string, channel = 'email') => destOf('campaigns', { id: 'c', title: 'x', status, channel }, 'w').to
    expect(at('draft')).toBe('/w/$workspace/campaigns/$campaignId/edit')
    expect(at('sending')).toBe('/w/$workspace/campaigns/$campaignId/live')
    expect(at('sent', 'sms')).toBe('/w/$workspace/campaigns/$campaignId/sms-report')
    expect(at('draft', 'push')).toBe('/w/$workspace/campaigns/$campaignId/push')
    expect(detailOf('lists', { id: 'l', title: 'Beta', count: 1204 })).toBe('List · 1,204 subscribed')
  })
})
