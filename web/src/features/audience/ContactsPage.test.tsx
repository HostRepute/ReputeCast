import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { components } from '@/api/schema'
import { setTestFetch } from '@/test/fetch'
import { json } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ContactsPage } from './ContactsPage'

type Item = components['schemas']['ContactListItem']

const editor = workspaceMe({ permissions: ['subscribers:view', 'subscribers:edit', 'subscribers:export'] })

function contact(n: number): Item {
  return {
    id: `c-${n}`,
    name: `Reader ${n}`,
    email: `reader${n}@example.com`,
    lists: [],
    tier: 'active',
    consent: 'double',
    mailboxProvider: 'gmail',
    mailboxLabel: 'Gmail',
    lastOpenMachine: false,
    joinedAt: '2026-10-01T09:00:00Z',
    source: 'api',
    sourceLabel: 'API',
    suppressed: false,
  }
}

/** Answers the page's calls; records the contacts and bulk requests. */
function serve(record: { lists: URL[]; bulk: unknown[] }) {
  setTestFetch(async (input, init) => {
    const req = input instanceof Request ? input : new Request(input, init)
    const url = new URL(req.url)
    const path = url.pathname.replace('/api/v1/workspaces/acme-newsletter', '')
    if (path === '/contacts' && req.method === 'GET') {
      record.lists.push(url)
      const cursor = url.searchParams.get('cursor')
      const exact = url.searchParams.get('exact') === 'true'
      const items = cursor ? [contact(3), contact(4)] : [contact(1), contact(2)]
      return json({
        items,
        total: exact ? 1048213 : 10000,
        totalCapped: !exact,
        page: 1,
        pageSize: 2,
        nextCursor: 'next-' + (cursor ?? 'first'),
        prevCursor: cursor ? 'prev-' + cursor : undefined,
      })
    }
    if (path === '/contacts/bulk') {
      record.bulk.push(await req.json())
      return json({ action: 'add_tag', matched: 1048213, done: 1048213, failed: 0, failures: [], target: 'vip' })
    }
    if (path === '/audience') {
      return json({ contacts: 1048213, lists: [], paidTiers: [], segments: 0, stripe: false, suppressions: 0, tiers: { active: 0, atRisk: 0, cooling: 0, sunset: 0 } })
    }
    if (path === '/contacts/views')
      return json({
        all: 1048213,
        items: [{ id: 'view-1', name: 'Team accounts', filters: { tier: 'active', tag: 'team' }, columns: ['email', 'field:plan'], count: 12, createdAt: '2026-10-01T09:00:00Z' }],
      })
    if (path === '/jobs') return json({ items: [] })
    // Everything else on the page (channels, website, languages…) is off here.
    return json({ title: 'Not Found', status: 404, detail: 'not here' }, 404)
  })
}

describe("A saved view's own address (review 2026-11 AU-19)", () => {
  afterEach(() => setTestFetch(undefined))

  it("opens ?view= with the view's filters and columns, reading no list before", async () => {
    const record = { lists: [] as URL[], bulk: [] as unknown[] }
    serve(record)
    const { router } = await renderInWorkspace(<ContactsPage search={{ view: 'view-1' } as never} />, { path: '/w/acme-newsletter/audience?view=view-1', me: editor })
    await waitFor(() => expect(router.state.location.search).toMatchObject({ tier: 'active', tag: 'team', cols: 'email,field:plan' }))
    expect(router.state.location.search).not.toHaveProperty('view')
    expect(record.lists).toHaveLength(0)
  })

  it('says so when the view was deleted', async () => {
    serve({ lists: [], bulk: [] })
    const { router } = await renderInWorkspace(<ContactsPage search={{ view: 'gone' } as never} />, { path: '/w/acme-newsletter/audience?view=gone', me: editor })
    expect(await screen.findByText('That saved view was deleted')).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.search).not.toHaveProperty('view'))
  })
})

describe('Contacts at scale (review 2026-11 AU-1, AU-5, D5)', () => {
  afterEach(() => setTestFetch(undefined))

  it('reads 10,000+, counts all on request, and pages with cursors', async () => {
    const record = { lists: [] as URL[], bulk: [] as unknown[] }
    serve(record)
    const { router } = await renderInWorkspace(<ContactsPage search={{ size: 2 } as never} />, { path: '/w/acme-newsletter/audience', me: editor })
    expect(await screen.findByText(/1 to 2 of 10,000\+/)).toBeInTheDocument()
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Count all' })))
    await waitFor(() => expect(record.lists.some((u) => u.searchParams.get('exact') === 'true')).toBe(true))
    expect(await screen.findByText(/1 to 2 of 1,048,213/)).toBeInTheDocument()
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Next' })))
    await waitFor(() => expect(router.state.location.search).toMatchObject({ cursor: 'next-first', page: 2 }))
  })

  it('selects everything matching after a whole page, and acts on all of it', async () => {
    const record = { lists: [] as URL[], bulk: [] as unknown[] }
    serve(record)
    await renderInWorkspace(<ContactsPage search={{ size: 2, tier: 'active' } as never} />, { path: '/w/acme-newsletter/audience', me: editor })
    await screen.findByText(/1 to 2 of 10,000\+/)
    await act(async () => fireEvent.click(screen.getByRole('checkbox', { name: /select all/i })))
    expect(await screen.findByText(/All 2 contacts on this page are selected/)).toBeInTheDocument()
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Select all 10,000+ matching' })))
    // Acting on everything waits for the exact number.
    const bar = screen.getByRole('region', { name: 'Bulk actions' })
    expect(await within(bar).findByText('All 1,048,213 matching')).toBeInTheDocument()
    expect(screen.getByText(/All 1,048,213 contacts matching this view are selected/)).toBeInTheDocument()
    await act(async () => fireEvent.click(within(bar).getByRole('button', { name: 'Add tag' })))
    fireEvent.change(await screen.findByRole('textbox', { name: 'Tag' }), { target: { value: 'vip' } })
    await act(async () => fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Add tag' })))
    await waitFor(() => expect(record.bulk).toHaveLength(1))
    expect(record.bulk[0]).toMatchObject({ action: 'add_tag', tag: 'vip', allMatching: true, filter: { tier: 'active' } })
  })
})
