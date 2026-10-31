import { act, fireEvent, screen, within } from '@testing-library/react'
import type { components } from '@/api/schema'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { InboxPage } from './InboxPage'

type Item = components['schemas']['ApprovalListItem']

function item(n: number, status: Item['status']): Item {
  return {
    id: `req-${n}`,
    subjectType: 'automation',
    automationId: `auto-${n}`,
    channel: 'email',
    name: `Request ${n}`,
    rules: 'rules: audience ≥ 10,000',
    author: { id: 'u-1', name: 'Daniel Okafor', email: 'daniel@acme.io', initials: 'DO', color: '#2563eb', role: 'Editor' },
    audience: 12000,
    route: 'SES 100%',
    scheduleMode: 'now',
    requestedAt: '2026-10-20T09:00:00Z',
    status,
  }
}

function page(items: Item[], nextCursor?: string) {
  return {
    items,
    nextCursor,
    counts: { pending: 1, approved: 2, changesRequested: 0, rejected: 0, invalidated: 0, all: 3 },
    authors: [{ id: 'u-1', name: 'Daniel Okafor' }],
    approvers: [],
    policy: { rules: [], approvalsRequired: 1, approvers: [] },
    allRequests: true,
    canManage: false,
  }
}

describe('InboxPage', () => {
  afterEach(() => mockApi({})())

  // Review 2026-11 EN-7: the inbox pages instead of stopping at 200 rows.
  it('loads older requests on demand', async () => {
    const cursors: (string | null)[] = []
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/approvals': (req) => {
        const cursor = new URL(req.url).searchParams.get('cursor')
        cursors.push(cursor)
        return json(cursor ? page([item(3, 'approved')]) : page([item(1, 'pending'), item(2, 'approved')], '1.1760950400000000.req-2'))
      },
    })
    await renderInWorkspace(<InboxPage search={{}} />, { me: workspaceMe({ permissions: ['campaigns:view', 'campaigns:approve'] }) })
    const table = await screen.findByRole('table', { name: 'Approval requests' })
    expect(within(table).getAllByRole('row')).toHaveLength(3)
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Show older requests' })))
    expect(await within(table).findByText('Request 3')).toBeInTheDocument()
    expect(cursors).toEqual([null, '1.1760950400000000.req-2'])
    expect(screen.queryByRole('button', { name: 'Show older requests' })).not.toBeInTheDocument()
  })
})
