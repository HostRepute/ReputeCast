import { act, fireEvent, screen } from '@testing-library/react'
import { useQuery } from '@tanstack/react-query'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { useNotificationActions } from './NotificationsPopover'

const key = ['get', '/api/v1/workspaces/{workspace}/notifications', { params: { path: { workspace: 'acme-newsletter' } } }]
const page = { items: [{ id: 'n-1', read: false, title: 'Approval needed' }], unread: 1, total: 1, retentionDays: 90 }

function Probe() {
  const { markRead } = useNotificationActions()
  // Subscribed to the cached list, as the bell and the page are.
  const { data: d } = useQuery({ queryKey: key, queryFn: () => page, staleTime: Infinity })
  return (
    <>
      <span>{d?.items[0]?.read ? 'read' : 'unread'}</span>
      <span>{`unread ${d?.unread}`}</span>
      <button type="button" onClick={() => void markRead('n-1')}>
        Mark
      </button>
    </>
  )
}

// Marking read shows at once, and a refused request puts it back (review
// 2026-11 FE-7).
describe('marking a notification read', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  async function setup(answer: () => Response) {
    const deferred: { resolve?: (r: Response) => void } = {}
    unmock = mockApi({
      'POST /api/v1/workspaces/acme-newsletter/notifications/n-1/read': () => new Promise<Response>((resolve) => (deferred.resolve = resolve)),
      'GET /api/v1/workspaces/acme-newsletter/notifications': () => json(page),
    })
    const r = await renderInWorkspace(<Probe />, { seed: (qc) => qc.setQueryData(key, page) })
    return { ...r, finish: () => act(async () => deferred.resolve?.(answer())) }
  }

  it('shows it read before the server answers', async () => {
    const { finish } = await setup(() => new Response(null, { status: 204 }))
    fireEvent.click(screen.getByRole('button', { name: 'Mark' }))
    expect(await screen.findByText('read', { exact: true })).toBeInTheDocument()
    expect(screen.getByText('unread 0')).toBeInTheDocument()
    await finish()
  })

  it('puts it back when the server refuses', async () => {
    const { finish } = await setup(() => json({ status: 500, title: 'Internal Server Error', detail: 'down' }, 500))
    fireEvent.click(screen.getByRole('button', { name: 'Mark' }))
    expect(await screen.findByText('read', { exact: true })).toBeInTheDocument()
    await finish()
    expect(await screen.findByText('unread')).toBeInTheDocument()
    expect(screen.getByText('unread 1')).toBeInTheDocument()
  })
})
