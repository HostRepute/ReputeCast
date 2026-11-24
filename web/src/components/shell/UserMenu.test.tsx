import { fireEvent, screen } from '@testing-library/react'
import { meQueryKey, type Me } from '@/lib/session'
import { renderInWorkspace } from '@/test/app'
import { UserMenu } from './UserMenu'

const me = {
  user: { id: 'u-1', email: 'aisha@acme.io', name: 'Aisha Rahman', avatarColor: '#0F766E', timezone: 'UTC', isInstanceAdmin: false, breakGlass: false, hasPassword: true, totp: true, passkeys: 0, recoveryCodes: 10, theme: 'system' },
  workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleKey: 'viewer', roleName: 'Viewer', memberCount: 6, subscribers: 48213 }],
  instance: { name: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.4.2', setupComplete: true, sourceUrl: 'https://git.acme.io/reputecast/tree/v1.4.2-acme' },
} as Me

// AGPL-3.0 section 13: everyone who uses the app, not only instance admins,
// can get the source of the version the server runs (source_url).
describe('Account menu', () => {
  it('links the source code of the running version', async () => {
    await renderInWorkspace(<UserMenu />, { seed: (qc) => qc.setQueryData(meQueryKey, me) })
    fireEvent.keyDown(await screen.findByRole('button', { name: 'Aisha Rahman, account menu' }), { key: 'Enter' })
    const item = await screen.findByRole('menuitem', { name: 'Source code' })
    expect(item).toHaveAttribute('href', 'https://git.acme.io/reputecast/tree/v1.4.2-acme')
    expect(item).toHaveAttribute('target', '_blank')
  })
})
