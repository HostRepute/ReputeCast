import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { MCPServersPage } from './MCPServersPage'

// An OAuth server whose client stops the sign-in: no client and nothing to
// register one with (mcp_oauth_client), or only private_key_jwt
// (mcp_oauth_client_auth). A manager lands in Edit with what to do; a
// member reads it; the instance's own metadata document shows as the client.

const base = '/api/v1/workspaces/acme-newsletter'
const servers = `${base}/ai/mcp/servers`
const server = {
  id: 's1',
  key: 'linear',
  name: 'Linear',
  url: 'https://mcp.linear.app/mcp',
  auth: 'oauth',
  transport: 'streamable',
  enabled: true,
  roles: [],
  status: 'ok',
  connected: false,
  headerSet: false,
  clientSecretSet: false,
  clientRegistered: false,
  clientMetadata: false,
  redirectUri: 'https://news.acme.io/api/v1/ai/mcp/oauth/callback',
  tools: [],
  waiting: 0,
}
const noClient = { status: 422, title: 'Unprocessable Entity', code: 'mcp_oauth_client', detail: "Linear's authorization server doesn't let ReputeCast register itself. A manager can register a client with it, using the redirect URI shown, and enter its client ID (and secret)." }
const clientAuth = (status: number) => ({ status, title: 'Bad Gateway', code: 'mcp_oauth_client_auth', detail: "Linear's authorization server accepts only client authentication ReputeCast doesn't support, such as private_key_jwt." })
const manager = workspaceMe({ permissions: ['ai:use', 'integrations:view', 'integrations:manage'] })
const member = workspaceMe({ permissions: ['ai:use', 'integrations:view'] })
const reads = (s: object = server) => ({
  [`GET ${servers}`]: () => json({ servers: [s] }),
  [`GET ${base}/ai`]: () => json({ helpers: ['Linh Nguyen'] }),
})

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('an MCP server’s OAuth client', () => {
  it('takes a manager to Edit when the server needs a client entered', async () => {
    unmock = mockApi({ ...reads(), [`POST ${servers}/s1/oauth`]: () => json(noClient, 422) })
    await renderInWorkspace(<MCPServersPage callback={{}} />, { me: manager })
    fireEvent.click(await screen.findByRole('button', { name: 'Connect your account to Linear' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Linear' })
    const alert = within(dialog).getByRole('alert')
    expect(alert).toHaveTextContent('Enter a client for Linear')
    expect(alert).toHaveTextContent(/register a client with it, using the redirect URI shown/)
    expect(within(dialog).getByRole('textbox', { name: /Client ID/ })).toBeVisible()
    expect(within(dialog).getByText(/Redirect address to register: https:\/\/news\.acme\.io/)).toBeVisible()
  })

  it('tells a member that private_key_jwt isn’t supported, without Edit', async () => {
    unmock = mockApi({ ...reads(), [`POST ${servers}/s1/oauth`]: () => json(clientAuth(502), 502) })
    await renderInWorkspace(<MCPServersPage callback={{}} />, { me: member })
    fireEvent.click(await screen.findByRole('button', { name: 'Connect your account to Linear' }))
    expect(await screen.findByText('Linear’s sign-in method isn’t supported')).toBeInTheDocument()
    expect(screen.getByText(/such as private_key_jwt/)).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('says so in Edit when a saved secret’s server only takes private_key_jwt', async () => {
    unmock = mockApi({ ...reads(), [`PATCH ${servers}/s1`]: () => json(clientAuth(422), 422) })
    await renderInWorkspace(<MCPServersPage callback={{}} />, { me: manager })
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Linear' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Linear' })
    fireEvent.click(within(dialog).getByText('Use a client registered with the server'))
    fireEvent.change(within(dialog).getByRole('textbox', { name: /Client ID/ }), { target: { value: 'reputecast' } })
    fireEvent.change(within(dialog).getByLabelText(/Client secret/), { target: { value: 's3cret' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    const alert = await within(dialog).findByRole('alert')
    expect(alert).toHaveTextContent('Linear’s sign-in method isn’t supported')
    expect(alert).toHaveTextContent(/such as private_key_jwt/)
  })

  it('names this instance as the client when it signs in with its metadata document', async () => {
    const metadata = { ...server, clientMetadata: true, clientId: 'https://news.acme.io/api/v1/ai/mcp/oauth/client-metadata.json' }
    unmock = mockApi(reads(metadata))
    await renderInWorkspace(<MCPServersPage callback={{}} />, { me: manager })
    expect(await screen.findByText('https://mcp.linear.app/mcp · OAuth, each member signs in · Client: this instance (client metadata document)')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Linear' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Linear' })
    await waitFor(() => expect(within(dialog).getByText('Client: this instance (client metadata document)')).toBeVisible())
  })
})
