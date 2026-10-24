import type { QueryClient } from '@tanstack/react-query'
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { ApiError } from '@/api/client'
import { dismissToast } from '@/components/ui/Toast'
import { meQueryKey } from '@/lib/session'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ForbiddenState, WorkspaceNotFound, WorkspaceRouteError, problemReference } from './SystemScreens'

const seedMe = (qc: QueryClient) => qc.setQueryData(meQueryKey, { user: { email: 'aisha@acme.io', name: 'Aisha Rahman' }, workspaces: [], instance: {} } as never)

let requestedAt: string | undefined
const sent: unknown[] = []
let unmock = () => {}

beforeEach(() => {
  requestedAt = undefined
  unmock = mockApi({
    'GET /api/v1/workspaces/acme-newsletter/access-requests/suggestion': (request) =>
      json({
        permission: new URL(request.url).searchParams.get('permission'),
        action: 'Routing, providers and sending',
        roleKey: 'editor',
        roleName: 'Editor',
        yourRole: 'Viewer',
        owners: ['Linh Nguyen', 'Marcus Chen'],
        managers: ['Linh Nguyen', 'Marcus Chen'],
        requestedAt,
      }),
    'POST /api/v1/workspaces/acme-newsletter/access-requests': async (request) => {
      sent.push(await request.json())
      requestedAt = new Date().toISOString()
      return json({ notified: ['Linh Nguyen', 'Marcus Chen'], roleName: 'Editor' }, 202)
    },
  })
})
afterEach(() => {
  unmock()
  sent.length = 0
  act(() => dismissToast())
})

describe('403 inside the shell', () => {
  it('says what the role allows, who can change it, and sends a request', async () => {
    await renderInWorkspace(<ForbiddenState permission="providers:manage" title="You can look, but not change routing" readOnly={<a href="/routing">View routing read-only</a>} />, {
      seed: seedMe,
    })
    expect(screen.getByRole('heading', { name: 'You can look, but not change routing' })).toBeInTheDocument()
    expect(screen.getByText('403 · Viewer role')).toBeInTheDocument()
    expect(
      await screen.findByText('Your role in Acme Newsletter is Viewer. Routing, providers and sending needs Editor or above. Owners: Linh Nguyen, Marcus Chen.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Requests go to the owners with a note; they usually answer within the day.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'View routing read-only' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Request Editor access' }))
    const dialog = await screen.findByRole('dialog', { name: 'Ask for Editor access' })
    expect(dialog).toHaveTextContent("Linh Nguyen and Marcus Chen can change roles. They'll see your note in their notifications; nothing changes until one of them does.")
    fireEvent.change(within(dialog).getByLabelText(/Why you need it/), { target: { value: 'I run the Monday send while Marcus is away.' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Send request' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(sent).toEqual([{ permission: 'providers:manage', note: 'I run the Monday send while Marcus is away.' }])
    expect(await screen.findByText('Request sent')).toBeInTheDocument()
    // The request is on record: no second button, and the footnote says so.
    expect(await screen.findByText("You asked just now; Linh Nguyen and Marcus Chen will see it. Ask them directly if it's urgent.")).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Request Editor access' })).not.toBeInTheDocument()
  })

  it('a screen that fails with 403 shows the role-limited state', async () => {
    const error = new ApiError(403, { status: 403, title: 'Forbidden', permission: 'providers:manage' } as never)
    await renderInWorkspace(<WorkspaceRouteError error={error} reset={() => {}} />, { seed: seedMe })
    expect(screen.getByRole('heading', { name: "Your role can't open this page" })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Request Editor access' })).toBeInTheDocument()
  })
})

describe('404 inside the shell', () => {
  it('names the missing path and offers the way back and the audit log', async () => {
    await renderInWorkspace(<WorkspaceNotFound />, { path: '/w/acme-newsletter/campaigns/weekly-digest-143', seed: seedMe })
    expect(screen.getByRole('heading', { name: "There's nothing at /campaigns/weekly-digest-143" })).toBeInTheDocument()
    expect(screen.getByText('The page may have been deleted, or the link was copied before it was saved.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Go to overview' })).toHaveAttribute('href', '/w/acme-newsletter')
    expect(screen.getByRole('link', { name: 'Search the audit log' })).toHaveAttribute('href', '/w/acme-newsletter/settings/audit')
  })

  it('leaves out the audit log for members who can’t read it', async () => {
    await renderInWorkspace(<WorkspaceNotFound />, { path: '/w/acme-newsletter/nope', me: workspaceMe({ permissions: ['campaigns:view'] }), seed: seedMe })
    expect(screen.getByRole('heading', { name: "There's nothing at /nope" })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Search the audit log' })).not.toBeInTheDocument()
  })

  it('a screen that fails with 404 shows what the server said and the request id', async () => {
    const error = new ApiError(404, { status: 404, title: 'Not Found', detail: "Campaign weekly-digest-143 doesn't exist.", meta: { requestId: 'req_01J9ZK' } } as never)
    await renderInWorkspace(<WorkspaceRouteError error={error} reset={() => {}} />, { path: '/w/acme-newsletter/campaigns/weekly-digest-143', seed: seedMe })
    expect(screen.getByRole('heading', { name: "There's nothing at /campaigns/weekly-digest-143" })).toBeInTheDocument()
    expect(screen.getByText("Campaign weekly-digest-143 doesn't exist. It may have been deleted, or the link was copied before it was saved.")).toBeInTheDocument()
    expect(screen.getByText('req_01J9ZK')).toBeInTheDocument()
  })
})

describe('500 inside the shell', () => {
  it('says what broke, where the log is, and retries', async () => {
    const reset = vi.fn()
    const error = new ApiError(500, { status: 500, title: 'Internal Server Error', meta: { errorId: 'err_7f3a' } } as never)
    await renderInWorkspace(<WorkspaceRouteError error={error} reset={reset} />, { seed: seedMe })
    expect(screen.getByRole('heading', { name: 'Something broke on our side' })).toBeInTheDocument()
    expect(screen.getByText(/the error is already in the server log with id/)).toHaveTextContent(
      "The page couldn't load. Retry now, or wait a minute; the error is already in the server log with id err_7f3a.",
    )
    expect(screen.getByText(/docker compose logs app/).closest('.k')).toHaveTextContent('Self-hosted? Check docker compose logs app for err_7f3a.')
    expect(screen.getByRole('link', { name: 'Back to overview' })).toHaveAttribute('href', '/w/acme-newsletter')
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Retry' })))
    expect(reset).toHaveBeenCalledOnce()
  })

  it('shows the reference the server logged the error under', () => {
    const error = new ApiError(500, { status: 500, title: 'Internal Server Error', detail: 'unexpected error occurred', code: 'internal_error', meta: { reference: 'err_7KQ2M4XA' } } as never)
    expect(problemReference(error)).toBe('err_7KQ2M4XA')
  })
})
