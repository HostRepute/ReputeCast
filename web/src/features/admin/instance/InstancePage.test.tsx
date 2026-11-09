import { screen } from '@testing-library/react'
import type { QueryClient } from '@tanstack/react-query'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { meQueryKey } from '@/lib/session'
import { InstancePage } from './InstancePage'

const settings = (o: Partial<Schemas['InstanceSettings']>): Schemas['InstanceSettings'] => ({
  name: 'Acme ReputeCast',
  publicUrl: 'https://reputecast.acme.io',
  configPublicUrl: 'https://reputecast.acme.io',
  defaultTimezone: 'America/New_York',
  defaultLanguage: 'en',
  defaultRegionFormat: 'standard',
  publicUrlLocked: false,
  ...o,
})

const version = (o: Partial<Schemas['InstanceVersion']>): Schemas['InstanceVersion'] => ({
  installed: 'v1.4.2',
  release: true,
  builtAt: '2026-09-18T10:00:00Z',
  checkEnabled: true,
  checkAllowed: true,
  updateAvailable: false,
  upgradeCommand: 'cd /opt/reputecast && docker compose pull && docker compose up -d',
  releasesUrl: 'https://github.com/hostrepute/reputecast/releases',
  ...o,
})

/** Linh, an instance admin. */
const admin = (qc: QueryClient) => {
  qc.setQueryData(meQueryKey, {
    user: { id: 'u-1', name: 'Linh Nguyen', email: 'linh@acme.io', isInstanceAdmin: true },
    workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleName: 'Owner' }],
    instance: { name: 'Acme ReputeCast', version: 'v1.4.2' },
  } as never)
  qc.setQueryData(['meta'], { instanceName: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.4.2' } as never)
}

const others = {
  'GET /api/v1/admin/smtp-relay': () => json({ status: 404, title: 'Not Found', detail: 'not in this test' }, 404),
  'GET /api/v1/admin/agency': () => json({ status: 404, title: 'Not Found', detail: 'not in this test' }, 404),
  'GET /api/v1/admin/workspaces': () => json({ status: 404, title: 'Not Found', detail: 'not in this test' }, 404),
}

describe('Instance admin › Instance & updates', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('lets an admin edit the public URL and shows how to upgrade', async () => {
    unmock = mockApi({ ...others, 'GET /api/v1/admin/instance': () => json(settings({})), 'GET /api/v1/admin/updates': () => json(version({})) })
    await renderInWorkspace(<InstancePage />, { path: '/w/acme-newsletter/admin/instance', seed: admin })
    const url = await screen.findByDisplayValue('https://reputecast.acme.io')
    expect(url).not.toHaveAttribute('readonly')
    expect(screen.getByText(/Used for provider webhooks/)).toBeInTheDocument()
    expect(await screen.findByText('Upgrade on the server')).toBeInTheDocument()
    expect(screen.getByText(/docker compose pull/)).toBeInTheDocument()
  })

  it('shows what the server’s operator manages: the public URL and upgrades', async () => {
    unmock = mockApi({
      ...others,
      'GET /api/v1/admin/instance': () => json(settings({ publicUrl: 'https://news.acme.io', configPublicUrl: 'https://news.acme.io', publicUrlLocked: true })),
      // The instance switch was on before the server turned the check off.
      'GET /api/v1/admin/updates': () => json(version({ checkEnabled: true, checkAllowed: false, managedBy: 'Northwind Hosting' })),
    })
    await renderInWorkspace(<InstancePage />, { path: '/w/acme-newsletter/admin/instance', seed: admin })
    const url = await screen.findByDisplayValue('https://news.acme.io')
    expect(url).toHaveAttribute('readonly')
    expect(url).toHaveAccessibleDescription(/Set by the server's operator \(http\.public_url_locked\)/)
    expect(screen.queryByText(/Used for provider webhooks/)).not.toBeInTheDocument()
    expect(await screen.findByText('Northwind Hosting')).toBeInTheDocument()
    expect(screen.getByText(/They choose when this instance moves to a new release/)).toBeInTheDocument()
    expect(screen.queryByText('Upgrade on the server')).not.toBeInTheDocument()
    expect(screen.queryByText(/docker compose pull/)).not.toBeInTheDocument()
    const toggle = screen.getByRole('switch', { name: /Check GitHub releases once a day/ })
    expect(toggle).toBeDisabled()
    expect(toggle).not.toBeChecked()
  })
})
