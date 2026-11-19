import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { CatalogPage } from '@/features/integrations/CatalogPage'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import type { AISettings } from './api'
import { connectTarget } from './ConnectDialog'
import { ModelAccounts } from './ModelSettings'
import { AISettingsPage } from './SettingsPage'

// Review of B4: a disconnected workspace account is connected again, not
// added a second time (finding 2); someone with ai:use but not
// integrations:view still manages their own key (finding 5).

const base = '/api/v1/workspaces/acme-newsletter'
const at = '2026-11-18T10:00:00Z'
const prices = { input: 3, output: 15, known: true }
const kept = {
  id: 'acc-1',
  primary: true,
  provider: 'anthropic',
  providerName: 'Anthropic',
  status: 'disconnected',
  demo: false,
  privateNetwork: false,
  connectedAt: at,
  key: { set: false },
  models: [{ id: 'm1', model: 'claude-sonnet-5-5', modelName: 'Claude Sonnet 5.5', imageInput: true, prices, tools: { state: 'supported' }, features: ['assistant'] }],
}
const settings = {
  accounts: [kept],
  canManage: true,
  connected: false,
  disconnected: true,
  connection: { provider: 'anthropic', providerName: 'Anthropic', model: 'claude-sonnet-5-5', modelName: 'Claude Sonnet 5.5', cap: { kind: 'usd', usd: 25, tokens: 0 }, prices, key: { set: false }, demo: false, privateNetwork: false, imageInput: true, connectedAt: at, status: 'connected' },
  usage: { costMicros: 0, tokens: 0 },
  catalog: [{ kind: 'anthropic', name: 'Anthropic', models: [{ id: 'claude-sonnet-5-5', name: 'Claude Sonnet 5.5', known: true, imageInput: true }], presets: [], needsKey: true, needsBaseUrl: false, capabilities: [] }],
  instanceAdmins: [],
  aiUseRoles: ['Owner'],
  priceListDate: '2026-11-01',
  helpers: [],
} as unknown as AISettings

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('a disconnected workspace account', () => {
  it('is connected again from its row and from Manage, through the connection endpoint', async () => {
    const calls: string[] = []
    unmock = mockApi({
      [`POST ${base}/ai/models`]: () => json({ models: settings.catalog[0]!.models, fromProvider: false }),
      [`PUT ${base}/ai/connection`]: async (r) => {
        calls.push(`PUT ${JSON.stringify(await r.json())}`)
        return json(settings)
      },
      [`POST ${base}/ai/accounts`]: () => {
        calls.push('POST accounts')
        return json(settings)
      },
    })
    await renderInWorkspace(<ModelAccounts s={settings} />, { me: workspaceMe({ permissions: ['integrations:view', 'integrations:manage', 'ai:use'] }) })
    // Manage offers Reconnect where the account-key endpoint would refuse a key.
    fireEvent.click(screen.getAllByRole('button', { name: 'Manage Anthropic' })[0]!)
    const manage = await screen.findByRole('dialog')
    expect(within(manage).queryByRole('button', { name: 'Add key' })).toBeNull()
    fireEvent.click(within(manage).getByRole('button', { name: 'Reconnect' }))
    const dialog = await screen.findByRole('dialog', { name: 'Connect an AI model' })
    fireEvent.change(within(dialog).getByLabelText(/API key/), { target: { value: 'sk-ant-new' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Connect Anthropic' }))
    await waitFor(() => expect(calls).toHaveLength(1))
    expect(calls[0]).toMatch(/^PUT /)
    expect(calls[0]).toContain('"model":"claude-sonnet-5-5"')
    expect(screen.getAllByRole('button', { name: 'Reconnect' }).length).toBeGreaterThan(0)
  })

  it('is what the catalog’s Connect on its provider reconnects; another provider is another account', () => {
    expect(connectTarget(settings, 'anthropic')).toBeUndefined()
    expect(connectTarget(settings, 'openai')).toBe('account')
    expect(connectTarget({ ...settings, disconnected: false } as AISettings, 'anthropic')).toBe('account')
    expect(connectTarget({ ...settings, accounts: [] } as AISettings, 'openai')).toBeUndefined()
  })
})

describe('someone with ai:use but not integrations:view', () => {
  const me = workspaceMe({ permissions: ['ai:use', 'campaigns:view'] })
  const catalog = [{ kind: 'anthropic', name: 'Anthropic', models: [{ id: 'claude-sonnet-5-5', name: 'Claude Sonnet 5.5', known: true, imageInput: true, input: 3, output: 15 }], presets: [], needsKey: true, needsBaseUrl: false, capabilities: [] }]
  const personal = { accounts: [], canAdd: true, capUsd: 10, choices: [], enabled: true, people: [], usage: { costMicros: 0, requests: 0, reservedCostMicros: 0, resetsAt: '2026-12-01T00:00:00Z', share: 0, state: 'ok' }, catalog, priceListDate: '2026-11-01T00:00:00Z' }
  const status = { helpers: ['Linh Nguyen'], accounts: [], defaults: [], features: [], prices: { known: true }, agent: { enabled: false }, usage: { costMicros: 0 } }

  it('manages their own key on AI models, from the personal endpoint', async () => {
    unmock = mockApi({
      [`GET ${base}/ai/personal`]: () => json(personal),
      [`GET ${base}/ai/status`]: () => json(status),
    })
    await renderInWorkspace(<AISettingsPage />, { me })
    expect(await screen.findByRole('heading', { name: 'Your personal key' })).toBeInTheDocument()
    expect(screen.getByText(/for people who can view integrations\. Ask Linh Nguyen\./)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Add a personal key' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add your personal key' })
    // The personal endpoint's price list: the model is picked, its prices come with it.
    expect(within(dialog).getByRole('combobox', { name: /Model/ })).toHaveValue('claude-sonnet-5-5')
    expect(within(dialog).getByText(/Prices pre-filled from ReputeCast's price list of/)).toBeInTheDocument()
  })

  it('finds the way to it from Settings › Integrations', async () => {
    unmock = mockApi({ [`GET ${base}/ai/status`]: () => json(status) })
    await renderInWorkspace(<CatalogPage />, { me })
    expect(await screen.findByText(/Integrations needs a permission you don.t have\. Ask Linh Nguyen\./)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Your personal key' })).toHaveAttribute('href', '/w/acme-newsletter/settings/integrations/ai')
  })
})
