import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { meQueryKey, type Me } from '@/lib/session'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { WhatsAppAccountPage } from './AccountPage'

const path = '/w/acme-newsletter/providers/whatsapp'
const base = 'https://reputecast.acme.io/hooks/whatsapp/acme-newsletter/'
const token = 'kP3x9Qm2LrT7vW4yZ8aB1cD5eF6gH0jK2nM4pQ6sU8w'
const callback = base + token

function account(patch: Partial<Schemas['WhatsAppAccount']> = {}): Schemas['WhatsAppAccount'] {
  return {
    id: 'acc-1',
    status: 'connected',
    businessName: 'Acme Inc',
    businessHint: '…4410',
    number: '+1 312 555 0110',
    displayName: 'Acme',
    nameStatus: 'approved',
    phoneStatus: 'connected',
    quality: 'high',
    messagingLimit: 1000,
    connectedAt: '2026-10-08T09:02:00Z',
    connectedBy: 'Marcus Chen',
    webhookUrl: callback,
    oldAddress: false,
    verifyToken: 'rc-wa-7Q2mX9vKp4',
    verifiedAt: '2026-10-08T09:14:00Z',
    demo: false,
    rejectedDeliveries: 0,
    otherAccountEvents: 0,
    wrongAddressDeliveries: 0,
    held: [],
    heldCount: 0,
    ...patch,
  }
}

function overview(a: Schemas['WhatsAppAccount'], canManage = true): Schemas['WhatsAppOverview'] {
  return {
    account: a,
    templates: [],
    templateCounts: { approved: 0, pending: 0, rejected: 0, other: 0 },
    consent: 0,
    consentLists: [],
    ratesAsOf: '2026-10-01T00:00:00Z',
    quiet: true,
    window: '21:00–08:00',
    sameWindowAsSms: false,
    webhookUrl: a.webhookUrl,
    testNumbers: [],
    canManage,
    canSend: canManage,
    helpers: canManage ? [] : ['Marcus Chen'],
  }
}

const channels = { switchShown: true, channels: [{ channel: 'whatsapp', available: true, setUp: true }] }

const aisha: Me = {
  user: {
    id: 'u-2',
    email: 'aisha@acme.io',
    name: 'Aisha Bello',
    avatarColor: 'info',
    timezone: 'UTC',
    isInstanceAdmin: false,
    breakGlass: false,
    hasPassword: true,
    totp: false,
    passkeys: 0,
    recoveryCodes: 0,
    theme: 'system',
  },
  workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleKey: 'viewer', roleName: 'Viewer', memberCount: 6, subscribers: 48213 }],
  instance: { name: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.0.0', setupComplete: true },
}
const owner = workspaceMe({ permissions: ['providers:view', 'providers:manage', 'campaigns:edit'] })

// An account connected before callback URLs had a token keeps the old
// address until Meta calls the new one: the admin is asked to paste it.
describe('WhatsApp › Account: the callback URL', () => {
  afterEach(() => mockApi({}))

  it('asks the admin to paste the new callback URL while Meta still calls the old address', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/whatsapp': () => json(overview(account({ oldAddress: true }))),
      'GET /api/v1/workspaces/acme-newsletter/channels': () => json(channels),
    })
    await renderInWorkspace(<WhatsAppAccountPage />, { path, me: owner })
    const note = (await screen.findByText('Paste the new callback URL in your Meta app')).closest('.note') as HTMLElement
    expect(within(note).getByText(callback)).toBeInTheDocument()
    expect(within(note).getByRole('button', { name: 'Copy the callback URL' })).toBeInTheDocument()
    expect(within(note).getByRole('link', { name: /Open Meta for Developers/ })).toHaveAttribute('href', 'https://developers.facebook.com/apps/')
  })

  it("says nothing once Meta calls the callback URL, and shows the URL on the Connection card", async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/whatsapp': () => json(overview(account())),
      'GET /api/v1/workspaces/acme-newsletter/channels': () => json(channels),
    })
    await renderInWorkspace(<WhatsAppAccountPage />, { path, me: owner })
    expect(await screen.findByText(callback)).toBeInTheDocument()
    expect(screen.queryByText('Paste the new callback URL in your Meta app')).not.toBeInTheDocument()
  })

  it('shows a viewer neither the callback URL nor the notice', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/whatsapp': () => json(overview(account({ oldAddress: true, webhookUrl: '', verifyToken: undefined }), false)),
      'GET /api/v1/workspaces/acme-newsletter/channels': () => json(channels),
    })
    await renderInWorkspace(<WhatsAppAccountPage />, {
      path,
      me: workspaceMe({ permissions: ['providers:view'] }),
      seed: (qc) => qc.setQueryData(meQueryKey, aisha),
    })
    expect(await screen.findByText(/Read-only/)).toBeInTheDocument()
    expect(screen.queryByText('Paste the new callback URL in your Meta app')).not.toBeInTheDocument()
    // The callback URL's token appears nowhere on a read-only page.
    expect(screen.queryByText((text) => text.includes(token))).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'New callback URL' })).not.toBeInTheDocument()
  })

  it('makes a new callback URL once the admin confirms, then waits for Meta', async () => {
    const fresh = base + 'Nw5Y7bC9dE1fG3hJ5kL7mN9pQ1rS3tU5vW7xY9zA2bC'
    let rotated = false
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/whatsapp': () =>
        json(overview(rotated ? account({ status: 'waiting', webhookUrl: fresh, verifiedAt: undefined, statusSince: new Date().toISOString() }) : account())),
      'GET /api/v1/workspaces/acme-newsletter/channels': () => json(channels),
      'POST /api/v1/workspaces/acme-newsletter/whatsapp/account/callback-url/rotate': () => {
        rotated = true
        return json(overview(account({ status: 'waiting', webhookUrl: fresh, verifiedAt: undefined })))
      },
    })
    await renderInWorkspace(<WhatsAppAccountPage />, { path, me: owner })
    fireEvent.click(await screen.findByRole('button', { name: 'New callback URL' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Make a new callback URL?')).toBeInTheDocument()
    expect(rotated).toBe(false)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Make a new URL' }))
    await waitFor(() => expect(rotated).toBe(true))
    expect(await screen.findByText("Waiting for Meta's verification request", { selector: 'b' })).toBeInTheDocument()
    expect(screen.getAllByText(fresh).length).toBeGreaterThan(0)
    expect(screen.queryByText(callback)).not.toBeInTheDocument()
  })
})
