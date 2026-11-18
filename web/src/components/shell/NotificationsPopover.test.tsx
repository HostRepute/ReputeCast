import { act, fireEvent, screen } from '@testing-library/react'
import { useQuery } from '@tanstack/react-query'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { closeAssistant, getAssistant, mountSlot, threadOf, waitingThread, WAITING_KIND } from '@/features/assistant/store'
import { NotificationLink, useNotificationActions, type NotificationItem } from './NotificationsPopover'

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

// A call waiting for its owner (B6): its notification opens the assistant's
// panel on its thread where the screen has the panel, else its full page.
describe('a waiting call’s notification', () => {
  let unmock = () => {}
  afterEach(() => {
    unmock()
    closeAssistant({ refocus: false })
  })
  const note = { id: 'n-2', kind: WAITING_KIND, read: false, title: 'A call waits for you', summary: '', link: '/w/acme-newsletter/assistant/th-9', severity: 'warn' } as NotificationItem

  it('opens the panel on its thread, without leaving the screen', async () => {
    unmock = mockApi({ 'POST /api/v1/workspaces/acme-newsletter/notifications/n-2/read': () => new Response(null, { status: 204 }), 'GET /api/v1/workspaces/acme-newsletter/notifications': () => json(page) })
    const unmount = mountSlot()
    const { router } = await renderInWorkspace(<NotificationLink n={note}>Waiting</NotificationLink>)
    fireEvent.click(screen.getByRole('link', { name: /Waiting/ }))
    expect(getAssistant().open).toBe(true)
    expect(threadOf('acme-newsletter')).toBe('th-9')
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns')
    unmount()
  })

  it('goes to the thread’s page where there is no panel', async () => {
    unmock = mockApi({ 'POST /api/v1/workspaces/acme-newsletter/notifications/n-2/read': () => new Response(null, { status: 204 }), 'GET /api/v1/workspaces/acme-newsletter/notifications': () => json(page) })
    const { router } = await renderInWorkspace(<NotificationLink n={note}>Waiting</NotificationLink>)
    fireEvent.click(screen.getByRole('link', { name: /Waiting/ }))
    expect(getAssistant().open).toBe(false)
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/assistant/th-9')
  })

  it('names its thread only for that kind', () => {
    expect(waitingThread(note)).toBe('th-9')
    expect(waitingThread({ ...note, kind: 'approval_requested' })).toBeUndefined()
  })
})
