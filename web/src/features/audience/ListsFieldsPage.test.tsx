import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ListsFieldsPage } from './ListsFieldsPage'

const ws = '/api/v1/workspaces/acme-newsletter'
const manager = workspaceMe({ permissions: ['subscribers:view', 'lists:manage'] })
const field = (id: string, key: string, label: string) => ({ id, key, label, type: 'text' as const, options: [], display: '' as const, createdAt: '2026-09-01T09:00:00Z' })

describe('Audience › Lists & fields (review 2026-11 AU-11)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('says what uses each field, shows it before deleting, and restores a deleted one', async () => {
    const calls: string[] = []
    unmock = mockApi({
      [`GET ${ws}/audience`]: () =>
        json({
          contacts: 120,
          lists: [{ id: 'l1', name: 'Weekly Digest', description: 'Every Friday', doubleOptIn: true, subscribed: 118, pending: 2, createdAt: '2026-01-01T00:00:00Z' }],
          paidTiers: [],
          segments: 1,
          stripe: false,
          suppressions: 0,
          tiers: { active: 0, atRisk: 0, cooling: 0, sunset: 0 },
        }),
      [`GET ${ws}/fields`]: () => json({ items: [field('f1', 'plan', 'Plan'), field('f2', 'company', 'Company')] }),
      [`GET ${ws}/fields/overview`]: () =>
        json({
          usage: [
            { id: 'f1', filled: 96, uses: [{ kind: 'segment', id: 's1', name: 'Paid plans' }, { kind: 'form', id: 'fm1', name: 'Signup' }] },
            { id: 'f2', uses: [] },
          ],
          deleted: [{ ...field('f3', 'old_score', 'Old score'), deletedAt: '2026-10-20T09:00:00Z', restoreUntil: '2026-11-19T09:00:00Z' }],
        }),
      [`GET ${ws}/fields/f1/impact`]: () => json({ id: 'f1', key: 'plan', label: 'Plan', uses: [{ kind: 'segment', id: 's1', name: 'Paid plans' }] }),
      [`DELETE ${ws}/fields/f1`]: () => {
        calls.push('delete f1')
        return new Response(null, { status: 204 })
      },
      [`POST ${ws}/fields/f3/restore`]: () => {
        calls.push('restore f3')
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<ListsFieldsPage />, { path: '/w/acme-newsletter/audience/lists', me: manager })

    expect(await screen.findByText('Weekly Digest')).toBeInTheDocument()
    expect(await screen.findByText('1 segment · 1 form')).toBeInTheDocument()
    const plan = screen.getByText('Plan').closest('tr')!
    expect(within(plan).getByText('96')).toBeInTheDocument()
    const company = screen.getByText('Company').closest('tr')!
    expect(within(company).getByText('Nothing')).toBeInTheDocument()
    expect(within(company).getByText('Not counted')).toBeInTheDocument()

    fireEvent.click(within(plan).getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog', { name: 'Delete the field "Plan"?' })
    expect(await within(dialog).findByText("It's used in places that stop working without it:")).toBeInTheDocument()
    expect(within(dialog).getByText('Segment')).toBeInTheDocument()
    expect(within(dialog).getByRole('link', { name: 'Open' })).toHaveAttribute('href', '/w/acme-newsletter/audience/segments/s1')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(calls).toContain('delete f1'))

    const old = screen.getByText('Old score').closest('tr')!
    fireEvent.click(within(old).getByRole('button', { name: 'Restore' }))
    await waitFor(() => expect(calls).toContain('restore f3'))
  })

  it("disables the buttons for people who can't manage lists", async () => {
    unmock = mockApi({
      [`GET ${ws}/audience`]: () => json({ contacts: 0, lists: [], paidTiers: [], segments: 0, stripe: false, suppressions: 0, tiers: { active: 0, atRisk: 0, cooling: 0, sunset: 0 } }),
      [`GET ${ws}/fields`]: () => json({ items: [] }),
      [`GET ${ws}/fields/overview`]: () => json({ usage: [], deleted: [] }),
    })
    await renderInWorkspace(<ListsFieldsPage />, { path: '/w/acme-newsletter/audience/lists', me: workspaceMe({ permissions: ['subscribers:view'] }) })
    expect(await screen.findByText('No lists yet. Create one to start collecting subscribers.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'New list' })).toHaveAttribute('aria-disabled', 'true')
  })
})
