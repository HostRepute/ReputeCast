import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ProposalPage } from './ProposalPage'
import { ProposalsPage } from './ProposalsPage'
import { SourceDialog } from './SourceDialog'

const base = '/api/v1/workspaces/acme-newsletter/mcp-connect'
const proposal: Schemas['MCPProposalView'] = {
  id: 'proposal-1', kind: 'campaign', name: 'Export announcement', clientName: 'Editorial assistant', verified: false,
  userName: 'Editor', createdAt: '2026-11-14T00:00:00Z', canDecide: true, baseRevision: 0,
  email: { subject: 'Faster exports', preheader: 'Fewer steps', mjml: '<mjml><mj-body>New content</mj-body></mjml>' },
  html: '<p>Faster exports</p>', decision: '', blocked: '', expiresAt: '2099-01-01T00:00:00Z',
}
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

it('separates new drafts from existing changes on the proposals list', async () => {
  unmock = mockApi({ [`GET ${base}/proposals`]: () => json({ items: [
    { ...proposal, change: false }, { ...proposal, id: 'proposal-2', name: 'Existing issue', change: true },
  ] }) })
  await renderInWorkspace(<ProposalsPage />)
  expect(within(await screen.findByRole('region', { name: 'New drafts' })).getByText(proposal.name)).toBeVisible()
  const existing = within(screen.getByRole('region', { name: 'Changes to existing drafts' }))
  expect(existing.getByRole('link', { name: 'Review proposal' })).toHaveAttribute('href', '/w/acme-newsletter/proposals/proposal-2')
  expect(existing.queryByText(proposal.name)).not.toBeInTheDocument()
})

it.each(['campaign', 'post'])('confirms a %s Keep, recovers a lost response and never submits twice', async (kind) => {
  const path = `${base}/proposals/${proposal.id}`
  let saved = false
  let offline = false
  const decisions: unknown[] = []
  unmock = mockApi({
    [`GET ${path}`]: () => offline ? json({ status: 503 }, 503) : json({ ...proposal, kind, ...(saved ? { decision: 'kept', resultId: 'kept-1' } : {}) }),
    [`POST ${path}/decision`]: async (r) => { decisions.push(await r.json()); saved = true; return json({ status: 503 }, 503) },
  })
  await renderInWorkspace(<ProposalPage id={proposal.id} />)
  fireEvent.click(await screen.findByRole('button', { name: 'Keep draft' }))
  const dialog = within(screen.getByRole('dialog'))
  expect(dialog.getByText(/Nothing is sent or published/)).toBeVisible()
  expect(decisions).toEqual([])
  fireEvent.click(dialog.getByRole('button', { name: 'Keep draft' }))
  fireEvent.click(dialog.getByRole('button', { name: 'Keep draft' }))
  await dialog.findByText('Could not confirm the result')
  expect(decisions).toEqual([{ decision: 'kept' }])
  expect(dialog.getByRole('button', { name: 'Keep draft' })).toBeDisabled()
  offline = true
  fireEvent.click(dialog.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(dialog.getByRole('button', { name: 'Check saved status' })).not.toHaveAttribute('aria-busy', 'true'))
  expect(dialog.getByRole('button', { name: 'Keep draft' })).toBeDisabled()
  offline = false
  fireEvent.click(dialog.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  expect(screen.getByRole('region', { name: 'Kept as a draft' })).toBeVisible()
  expect(screen.queryByText('Proposed · not saved')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: kind === 'post' ? 'Open in post editor' : 'Open in composer' })).toHaveAttribute('href', kind === 'post' ? '/w/acme-newsletter/forms/website/posts/kept-1' : '/w/acme-newsletter/campaigns/kept-1/edit')
  await waitFor(() => expect(document.activeElement).toHaveAttribute('role', 'status'))
  expect(decisions).toHaveLength(1)
})

it('shows the exact before and after content and allows rejection, not acceptance, of a stale change', async () => {
  const path = `${base}/proposals/${proposal.id}`
  let decision = ''
  unmock = mockApi({
    [`GET ${path}`]: () => json({ ...proposal, targetId: 'existing-1', baseRevision: 7, before: { ...proposal.email, subject: 'Old exports', mjml: '<mjml>Old content</mjml>' }, decision, blocked: decision ? '' : 'changed' }),
    [`POST ${path}/decision`]: async (r) => { decision = (await r.json()).decision; return new Response(null, { status: 204 }) },
  })
  await renderInWorkspace(<ProposalPage id={proposal.id} />)
  expect(await screen.findByText('Based on revision 7', { exact: false })).toBeVisible()
  expect(within(screen.getByRole('region', { name: 'Before · saved content' })).getByText('Old exports')).toBeVisible()
  expect(within(screen.getByRole('region', { name: 'Proposed · not saved' })).getByText('Faster exports')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Accept change' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Copy MJML for manual editing' })).toBeEnabled()
  fireEvent.click(screen.getByRole('button', { name: 'Reject proposal' }))
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Reject proposal' }))
  await screen.findByRole('region', { name: 'Rejected' })
  expect(decision).toBe('rejected')
  expect(screen.queryByRole('button', { name: /send|publish/i })).not.toBeInTheDocument()
})

it.each(['expired', 'revoked', 'purged', 'permission'] as const)('blocks application when %s and hides purged content', async (blocked) => {
  unmock = mockApi({ [`GET ${base}/proposals/${proposal.id}`]: () => json({ ...proposal, blocked, ...(blocked === 'permission' ? { canDecide: false } : {}) }) })
  await renderInWorkspace(<ProposalPage id={proposal.id} />)
  expect(await screen.findByRole('button', { name: 'Keep draft' })).toBeDisabled()
  if (blocked === 'permission') expect(screen.getByRole('button', { name: 'Discard draft' })).toBeDisabled()
  if (blocked === 'purged') {
    expect(screen.queryByText(proposal.email.subject)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Copy MJML for manual editing' })).not.toBeInTheDocument()
  } else expect(screen.getByText(proposal.email.subject)).toBeVisible()
})

it('recovers the exact selected source after an ambiguous write before allowing more sharing', async () => {
  const connection: Schemas['MCPConnection'] = {
    id: 'connection-1', clientName: 'Editorial assistant', verified: false, userId: 'user-1', userName: 'Editor',
    scopes: ['editorial:read'], lists: [], senderDomains: [], allowedIps: [],
    createdAt: '2026-01-01T00:00:00Z', expiresAt: '2099-01-01T00:00:00Z', revokedReason: '',
  }
  const path = `${base}/connections/${connection.id}/sources`
  let shared = false
  let offline = false
  const writes: unknown[] = []
  unmock = mockApi({
    'GET /api/v1/workspaces/acme-newsletter/campaigns': () => json({ items: [{ id: 'campaign-1', name: 'First issue' }], total: 1 }),
    [`GET ${path}`]: () => offline ? json({ status: 503 }, 503) : json({ items: shared ? [{ id: 'campaign-1', name: 'First issue', kind: 'campaign', handle: 'opaque-handle', expiresAt: '2099-01-01T00:00:00Z' }] : [] }),
    [`POST ${path}`]: async (r) => { writes.push(await r.json()); shared = true; return json({ status: 503 }, 503) },
  })
  await renderInWorkspace(<SourceDialog connection={connection} onClose={() => {}} />)
  fireEvent.change(await screen.findByLabelText('Select a resource'), { target: { value: 'campaign-1' } })
  const share = screen.getByRole('button', { name: 'Share selected source' })
  await waitFor(() => expect(share).toBeEnabled())
  fireEvent.click(share)
  const sharing = within(await screen.findByRole('dialog', { name: 'Review before sharing' }))
  expect(sharing.getByText(/Editorial assistant/)).toBeVisible()
  expect(sharing.getByText(/external client/)).toBeVisible()
  expect(writes).toEqual([])
  fireEvent.click(sharing.getByRole('button', { name: 'Cancel' }))
  expect(writes).toEqual([])
  fireEvent.click(share)
  fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
  await screen.findByText('Could not confirm the result')
  expect(share).toBeDisabled()
  expect(screen.getByLabelText('Source type')).toBeDisabled()
  offline = true
  fireEvent.click(screen.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Check saved status' })).not.toHaveAttribute('aria-busy', 'true'))
  expect(share).toBeDisabled()
  offline = false
  fireEvent.click(screen.getByRole('button', { name: 'Check saved status' }))
  await screen.findByText(/Source selected until/)
  expect(share).toBeDisabled()
  expect(screen.getByRole('list')).toHaveTextContent('First issue')
  expect(writes).toEqual([{ sharingReviewed: true, kind: 'campaign', id: 'campaign-1' }])
})

it('shares a saved template or the current voice only after explicit selection', async () => {
  const connection: Schemas['MCPConnection'] = {
    id: 'connection-1', clientName: 'Editorial assistant', verified: false, userId: 'user-1', userName: 'Editor',
    scopes: ['editorial:read'], lists: [], senderDomains: [], allowedIps: [],
    createdAt: '2026-01-01T00:00:00Z', expiresAt: '2099-01-01T00:00:00Z', revokedReason: '',
  }
  const writes: unknown[] = []
  const me = workspaceMe({ permissions: ['campaigns:view', 'templates:view'] })
  unmock = mockApi({
    'GET /api/v1/workspaces/acme-newsletter/campaigns': () => json({ items: [], total: 0 }),
    'GET /api/v1/workspaces/acme-newsletter/templates': () => json({ templates: [{ id: 'saved-1', name: 'Product notes', builtin: false }, { id: 'starter:blank', name: 'Blank starter', builtin: true }] }),
    'GET /api/v1/workspaces/acme-newsletter/brand-kit/voice': () => json({ revision: 7, content: { tone: 'Calm' } }),
    [`GET ${base}/connections/connection-1/sources`]: () => json({ items: [] }),
    [`POST ${base}/connections/connection-1/sources`]: async (r) => { const body = await r.json(); writes.push(body); return json({ kind: body.kind, handle: 'opaque', expiresAt: '2099-01-01T00:00:00Z' }) },
  })
  await renderInWorkspace(<SourceDialog connection={connection} onClose={() => {}} />, { me })
  fireEvent.change(await screen.findByLabelText('Source type'), { target: { value: 'template' } })
  expect(await screen.findByRole('option', { name: 'Product notes' })).toBeInTheDocument()
  expect(screen.queryByRole('option', { name: 'Blank starter' })).not.toBeInTheDocument()
  expect(screen.getByText(/read-only references/)).toBeVisible()
  const share = screen.getByRole('button', { name: 'Share selected source' })
  expect(share).toBeDisabled()
  fireEvent.change(screen.getByLabelText('Select a resource'), { target: { value: 'saved-1' } })
  fireEvent.click(share)
  fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
  await screen.findByText(/Source selected until/)
  await waitFor(() => expect(screen.getByLabelText('Source type')).toBeEnabled())
  fireEvent.change(screen.getByLabelText('Source type'), { target: { value: 'voice' } })
  expect(await screen.findByRole('option', { name: /Brand voice · Based on revision 7/ })).toBeInTheDocument()
  expect(screen.queryByLabelText('Search resources')).not.toBeInTheDocument()
  expect(share).toBeDisabled()
  fireEvent.change(screen.getByLabelText('Select a resource'), { target: { value: me.workspace.id } })
  fireEvent.click(share)
  expect(writes).toHaveLength(1)
  fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
  await screen.findByText(/Source selected until/)
  expect(writes).toEqual([{ sharingReviewed: true, kind: 'template', id: 'saved-1' }, { sharingReviewed: true, kind: 'voice', id: me.workspace.id }])
})

it('reviews a segment and flow as one decision and hands off to the flow editor', async () => {
  const path = `${base}/proposals/${proposal.id}`
  const mail = { ...proposal.email, key: 'email1', name: 'First welcome' }
  const workflow: Schemas['MCPWorkflowContent'] = {
    segmentName: 'Quiet readers', segment: { groups: [{ match: 'all', conditions: [{ field: 'tag', op: 'has', value: 'vip' }] }] },
    flow: { emails: [mail], graph: { nodes: [
      { id: 'start', type: 'trigger', position: { x: 0, y: 0 }, data: { trigger: { kind: 'segment_entered', segmentId: 'local-segment', reentry: 'once' } } },
      { id: 'email', type: 'email', position: { x: 0, y: 100 }, data: { email: { emailId: 'email1', emailVersion: 1, subject: mail.subject } } },
      { id: 'end', type: 'exit', position: { x: 0, y: 200 }, data: {} },
    ], edges: [{ id: 'a', source: 'start', target: 'email', data: {} }, { id: 'b', source: 'email', target: 'end', data: {} }] } },
    previews: [proposal.html],
  }
  let decision = ''
  const writes: unknown[] = []
  unmock = mockApi({
    [`GET ${path}`]: () => json({ ...proposal, kind: 'automation', workflow, decision, ...(decision ? { resultId: 'flow-kept' } : {}) }),
    'GET /api/v1/workspaces/acme-newsletter/segments/catalog': () => json({ fields: [{ key: 'tag', label: 'Tag', kind: 'select', ops: [{ key: 'has', label: 'Has' }], options: [{ value: 'vip', label: 'VIP readers' }] }] }),
    [`POST ${path}/decision`]: async (r) => { const body = await r.json(); writes.push(body); decision = body.decision; return new Response(null, { status: 204 }) },
  })
  await renderInWorkspace(<ProposalPage id={proposal.id} />)
  expect(await screen.findByText('VIP readers')).toBeVisible()
  expect(screen.getByText(/One dependency bundle/)).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Keep draft' })).not.toBeInTheDocument()
  fireEvent.click(screen.getByText('First welcome', { selector: 'summary' }))
  expect(screen.getByRole('region', { name: 'Proposed · not saved' })).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Keep workflow' }))
  const dialog = within(screen.getByRole('dialog'))
  expect(dialog.getByText(/either all are saved or none are/)).toBeVisible()
  expect(writes).toEqual([])
  fireEvent.click(dialog.getByRole('button', { name: 'Keep workflow' }))
  expect(await screen.findByRole('link', { name: 'Open in flow editor' })).toHaveAttribute('href', '/w/acme-newsletter/automations/flow-kept')
  expect(writes).toEqual([{ decision: 'kept' }])
  expect(screen.queryByRole('button', { name: /send|publish/i })).not.toBeInTheDocument()
})

it('shares only report totals when the connection has no editorial capability', async () => {
  const connection: Schemas['MCPConnection'] = {
    id: 'report-connection', clientName: 'Reports client', verified: false, userId: 'user-1', userName: 'Editor',
    scopes: ['reports:read'], lists: [], senderDomains: [], allowedIps: [],
    createdAt: '2026-01-01T00:00:00Z', expiresAt: '2099-01-01T00:00:00Z', revokedReason: '',
  }
  const writes: unknown[] = []
  unmock = mockApi({
    'GET /api/v1/workspaces/acme-newsletter/campaigns': (r) => {
      expect(new URL(r.url).searchParams.get('status')).toBe('sent')
      return json({ items: [{ id: 'campaign-1', name: 'Last month' }], total: 1 })
    },
    [`GET ${base}/connections/${connection.id}/sources`]: () => json({ items: [] }),
    [`POST ${base}/connections/${connection.id}/sources`]: async (r) => { writes.push(await r.json()); return json({ kind: 'report', handle: 'opaque', expiresAt: '2099-01-01T00:00:00Z' }) },
  })
  await renderInWorkspace(<SourceDialog connection={connection} onClose={() => {}} />, { me: workspaceMe({ permissions: ['campaigns:view', 'reports:view'] }) })
  expect(await screen.findByRole('option', { name: 'Campaign report' })).toBeInTheDocument()
  expect(screen.queryByRole('option', { name: 'Campaign' })).not.toBeInTheDocument()
  expect(screen.queryByRole('option', { name: 'Saved template' })).not.toBeInTheDocument()
  expect(screen.getByText(/selecting again never refreshes these numbers/)).toBeVisible()
  expect(writes).toEqual([])
  fireEvent.change(await screen.findByLabelText('Select a resource'), { target: { value: 'campaign-1' } })
  const share = screen.getByRole('button', { name: 'Share selected source' })
  await waitFor(() => expect(share).toBeEnabled())
  fireEvent.click(share)
  fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
  await screen.findByText(/Source selected until/)
  expect(writes).toEqual([{ sharingReviewed: true, kind: 'report', id: 'campaign-1' }])
})
