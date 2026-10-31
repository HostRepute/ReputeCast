import { fireEvent, screen, waitFor } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { EraseDialog } from './ContactDialogs'

const erasePath = 'POST /api/v1/workspaces/acme-newsletter/contacts/erase'
const owner = workspaceMe({
  permissions: ['subscribers:view', 'subscribers:delete', 'earn:view'],
  workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' },
})

describe('Erase contacts (review 2026-11 DP-1)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('says Stripe keeps its records only where the workspace has paid members', async () => {
    unmock = mockApi({})
    const { unmount } = await renderInWorkspace(<EraseDialog ids={['a', 'b']} onClose={() => {}} onDone={() => {}} stripe />, { me: owner })
    expect(screen.getByText('Stripe keeps its own records of paid members. Delete the customer in Stripe too if you need to.')).toBeInTheDocument()
    unmount()
    await renderInWorkspace(<EraseDialog ids={['a', 'b']} onClose={() => {}} onDone={() => {}} />, { me: owner })
    expect(screen.queryByText(/Stripe keeps its own records/)).not.toBeInTheDocument()
  })

  it('shows a refusal for contacts who still pay, with the way to Earn, and stays open', async () => {
    let done = false
    unmock = mockApi({
      [erasePath]: () =>
        json(
          {
            status: 409,
            title: 'Conflict',
            code: 'paid_members',
            detail: "1 contact still pays through Stripe. Cancel their paid subscription in Earn › Paid members first; erasing it here wouldn't stop Stripe from charging.",
            meta: { refused: 1, matched: 2 },
          },
          409,
        ),
    })
    await renderInWorkspace(<EraseDialog ids={['a', 'b']} onClose={() => {}} onDone={() => (done = true)} stripe />, { me: owner })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: '2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Erase 2 contacts' }))
    await waitFor(() => expect(screen.getByText(/1 contact still pays through Stripe/)).toBeInTheDocument())
    expect(screen.getByRole('link', { name: 'Open Earn › Paid members' })).toBeInTheDocument()
    expect(done).toBe(false)
  })
})
