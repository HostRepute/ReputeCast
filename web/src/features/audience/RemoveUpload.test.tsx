import { fireEvent, screen, waitFor } from '@testing-library/react'
import type { components } from '@/api/schema'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { RemoveUploadDialog } from './SuppressionDialogs'

const ws = '/api/v1/workspaces/acme-newsletter'
const item = {
  id: 'grp-1',
  group: true,
  upload: true,
  importId: 'up-1',
  address: 'role-addresses-oct.csv',
  rows: 412,
  uploadedBy: 'Linh Nguyen',
  reasons: ['manual'],
  originalCode: '',
  context: 'Upload',
  at: '2026-10-01T09:00:00Z',
  scope: 'all',
  source: { label: 'CSV upload · Marcus Chen' },
} as unknown as components['schemas']['SuppressionItem']

describe('Remove a suppression upload (review 2026-11 AU-14)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('takes a whole CSV upload off the list with a reason', async () => {
    const bodies: unknown[] = []
    let closed = false
    unmock = mockApi({
      [`POST ${ws}/suppressions/uploads/up-1/remove`]: async (r) => {
        bodies.push(await r.json())
        return json({ removed: 412 })
      },
    })
    await renderInWorkspace(<RemoveUploadDialog item={item} onClose={() => (closed = true)} />, { me: workspaceMe({ permissions: ['subscribers:view', 'subscribers:edit'] }) })
    const dialog = screen.getByRole('dialog', { name: 'Remove 412 suppressions?' })
    expect(dialog).toHaveTextContent('From role-addresses-oct.csv, uploaded by Linh Nguyen on 1 Oct. These addresses can receive email again')
    fireEvent.click(screen.getByRole('button', { name: 'Remove 412' }))
    expect(await screen.findByText(/Say why/i)).toBeInTheDocument()
    expect(bodies).toHaveLength(0)
    fireEvent.change(screen.getByRole('textbox', { name: 'Why (kept in the audit log)' }), { target: { value: 'Uploaded the wrong file' } })
    fireEvent.click(screen.getByRole('button', { name: 'Remove 412' }))
    await waitFor(() => expect(bodies).toEqual([{ reason: 'Uploaded the wrong file' }]))
    await waitFor(() => expect(closed).toBe(true))
  })
})
