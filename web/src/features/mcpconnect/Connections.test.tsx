import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { ConsentPage } from './ConsentPage'
import { ConnectionsPage } from './ConnectionsPage'

const base = '/api/v1/workspaces/acme-newsletter/mcp-connect'
const handle = 'rc_oauth_request_test'
const consent: Schemas['MCPConsent'] = {
  clientId: 'client-1', clientName: 'Editorial assistant', verified: false,
  callbackHost: 'assistant.example', scopes: ['editorial:read', 'reports:read', 'drafts:propose'],
  expiresAt: '2099-01-01T00:00:00Z',
}
const options: Schemas['MCPConsentOptions'] = {
  endpoint: 'https://news.example/mcp/connect', scopes: ['editorial:read', 'drafts:propose'],
  lists: [{ id: 'list-1', name: 'Weekly' }, { id: 'list-2', name: 'Monthly' }],
  senderDomains: ['news.example', 'updates.example'], defaultLists: ['list-1', 'list-2'], defaultDomains: ['news.example'],
}
const connection: Schemas['MCPConnection'] = {
  id: 'older-connection', clientName: 'Editorial assistant', verified: false, userId: 'user-1', userName: 'Editor',
  scopes: ['editorial:read'], lists: [], senderDomains: [], allowedIps: [],
  createdAt: '2026-01-01T00:00:00Z', expiresAt: '2099-01-01T00:00:00Z', revokedReason: '',
}
const problem = () => json({ status: 422, detail: 'Check request fields.' }, 422)
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

function consentRoutes(pending = false) {
  return {
    'GET /api/v1/me': () => json({ user: { name: 'Editor' }, pending, workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter' }], instance: { languages: ['en'] } }),
    'GET /api/v1/meta': () => json({ instanceName: 'ReputeCast', publicUrl: 'https://news.example', languages: ['en'] }),
    'GET /api/v1/mcp-connect/consent': () => json(consent),
    [`GET ${base}/options`]: () => json(options),
  }
}

it('requires explicit capabilities and preserves the current scope when the optional restriction is cleared', async () => {
  const saved: unknown[] = []
  unmock = mockApi({ ...consentRoutes(), [`POST ${base}/consent`]: async (r) => { saved.push(await r.json()); return problem() } })
  await renderInWorkspace(<ConsentPage request={handle} />)
  fireEvent.change(await screen.findByLabelText('Workspace'), { target: { value: 'acme-newsletter' } })
  const allow = await screen.findByRole('button', { name: 'Allow connection' })
  expect(allow).toBeDisabled()
  const editorial = screen.getByRole('checkbox', { name: /Read editorial context/ })
  expect(editorial).not.toBeChecked()
  expect(screen.getByRole('checkbox', { name: /Read campaign totals/ })).toBeDisabled()
  expect(screen.getByText('assistant.example')).toBeVisible()
  expect(screen.getByText('Unverified client')).toBeVisible()
  fireEvent.click(editorial)
  fireEvent.click(screen.getByRole('checkbox', { name: 'Monthly' }))
  fireEvent.click(screen.getByRole('checkbox', { name: 'Restrict to selected lists' }))
  fireEvent.click(screen.getByRole('checkbox', { name: 'Restrict to selected sender domains' }))
  fireEvent.change(screen.getByLabelText('Allowed client IPs or CIDRs'), { target: { value: '203.0.113.9\n\n2001:db8::/48' } })
  fireEvent.click(allow)
  await screen.findByText('Check the selected capabilities, resource scope and IP addresses, then try again.')
  expect(saved).toEqual([{ request: handle, scopes: ['editorial:read'], lists: ['list-1', 'list-2'], senderDomains: ['news.example'], allowedIps: ['203.0.113.9', '2001:db8::/48'] }])
  expect(editorial).toBeChecked()
  expect(allow).toBeEnabled()
})

it('serializes Allow and Deny, then requires a successful status check after a lost response', async () => {
  let finish: ((response: Response) => void) | undefined
  const saved: unknown[] = []
  let denyCount = 0
  let reads = 0
  let offline = false
  unmock = mockApi({ ...consentRoutes(),
    'GET /api/v1/mcp-connect/consent': () => { reads++; return offline ? json({ status: 503 }, 503) : json(consent) },
    [`POST ${base}/consent`]: async (r) => { saved.push(await r.json()); return new Promise<Response>((resolve) => { finish = resolve }) },
    'POST /api/v1/mcp-connect/consent/deny': () => { denyCount++; return problem() },
  })
  await renderInWorkspace(<ConsentPage request={handle} />)
  fireEvent.change(await screen.findByLabelText('Workspace'), { target: { value: 'acme-newsletter' } })
  fireEvent.click(await screen.findByRole('checkbox', { name: /Read editorial context/ }))
  const allow = screen.getByRole('button', { name: 'Allow connection' })
  const form = allow.closest('form')!
  fireEvent.submit(form)
  fireEvent.submit(form)
  await waitFor(() => expect(saved).toHaveLength(1))
  expect(screen.getByRole('button', { name: 'Deny' })).toBeDisabled()
  expect(screen.getByLabelText('Workspace')).toBeDisabled()
  finish!(json({ status: 503 }, 503))
  await screen.findByText('Could not confirm the result')
  expect(allow).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: 'Deny' }))
  expect(denyCount).toBe(0)
  offline = true
  fireEvent.click(screen.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(reads).toBe(2))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Check saved status' })).not.toHaveAttribute('aria-busy', 'true'))
  expect(allow).toBeDisabled()
  expect(screen.getByRole('checkbox', { name: /Read editorial context/ })).toBeChecked()
  offline = false
  fireEvent.click(screen.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(screen.queryByText('Could not confirm the result')).not.toBeInTheDocument())
  expect(allow).toBeEnabled()
  expect(saved).toHaveLength(1)
})

it('permits denial before account security is complete without querying workspace grants', async () => {
  const saved: unknown[] = []
  unmock = mockApi({ ...consentRoutes(true), 'POST /api/v1/mcp-connect/consent/deny': async (r) => { saved.push(await r.json()); return problem() } })
  await renderInWorkspace(<ConsentPage request={handle} />)
  const deny = await screen.findByRole('button', { name: 'Deny' })
  expect(screen.queryByRole('button', { name: 'Allow connection' })).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Complete account setup' })).toHaveAttribute('href', '/setup')
  fireEvent.click(deny)
  await waitFor(() => expect(saved).toEqual([{ request: handle }]))
})

it('recovers revocation by exact connection ID, even on an older page and after losing MCP management', async () => {
  let revoked = false
  let deletes = 0
  let statusReads = 0
  let offline = true
  unmock = mockApi({
    [`GET ${base}/connections`]: (r) => json(new URL(r.url).searchParams.get('cursor') ? { items: [{ ...connection, ...(revoked ? { revokedAt: '2026-11-14T00:00:00Z' } : {}) }] } : { items: [{ ...connection, id: 'newest', revokedAt: '2026-11-01T00:00:00Z' }], nextCursor: 'older' }),
    [`GET ${base}/connections/${connection.id}`]: () => { statusReads++; return offline ? json({ status: 503 }, 503) : json({ ...connection, revokedAt: '2026-11-14T00:00:00Z' }) },
    [`DELETE ${base}/connections/${connection.id}`]: () => { deletes++; revoked = true; return json({ status: 503 }, 503) },
  })
  await renderInWorkspace(<ConnectionsPage />)
  fireEvent.click(await screen.findByRole('button', { name: 'Older connections' }))
  fireEvent.click(await screen.findByRole('button', { name: 'Revoke connection' }))
  const dialog = within(screen.getByRole('dialog'))
  fireEvent.click(dialog.getByRole('button', { name: 'Revoke connection' }))
  await dialog.findByText('Could not confirm the result')
  expect(dialog.getByRole('button', { name: 'Revoke connection' })).toBeDisabled()
  fireEvent.click(dialog.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(statusReads).toBe(1))
  expect(dialog.getByRole('button', { name: 'Revoke connection' })).toBeDisabled()
  offline = false
  await waitFor(() => expect(dialog.getByRole('button', { name: 'Check saved status' })).not.toHaveAttribute('aria-busy', 'true'))
  fireEvent.click(dialog.getByRole('button', { name: 'Check saved status' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  expect(screen.getByText('Connection revoked. Kept drafts were not changed.')).toBeVisible()
  expect(screen.getByText('Revoked')).toBeVisible()
  expect(deletes).toBe(1)
  expect(statusReads).toBe(2)
})
