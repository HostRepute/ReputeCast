import { render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { SaveRoleDialog } from './SaveRoleDialog'

const role = { id: 'r-1', key: 'content-writer', name: 'Content Writer', builtin: false } as Schemas['RoleSummary']

function impact(patch: Partial<Schemas['RoleImpact']> = {}): Schemas['RoleImpact'] {
  return {
    added: ['campaigns:send'],
    removed: [],
    addedSensitive: ['campaigns:send'],
    removedSensitive: [],
    members: [
      { id: 'm-1', name: 'Daniel Park', email: 'daniel@acme.io', avatarColor: 'warn', gains: ['campaigns:send'], loses: [], effect: 'Two-factor becomes required; their IdP already enforces it', twoFactor: 'idp' },
    ],
    membersAffected: 1,
    keys: [],
    apiKeys: 0,
    agents: 0,
    sensitive: 'campaigns:send · two-factor required',
    approvals: 'Holders can send; the approval policy still applies to large or risky sends',
    pendingApprovals: 1,
    pendingRequests: [{ requester: 'Daniel Park', subject: 'November Product Update', stillMatches: ['audience ≥ 10,000'] }],
    summary: '',
    locked: [],
    graceDays: 7,
    ...patch,
  }
}

const noop = () => {}

describe('saving a role that adds a sensitive permission', () => {
  it('names the member, the two-factor change and the pending request (RoleEditorStates)', () => {
    render(<SaveRoleDialog kind="sensitive" role={role} name="Content Writer" impact={impact()} busy={false} onBack={noop} onSave={noop} />)
    const dialog = screen.getByRole('dialog', { name: 'Save changes to Content Writer?' })
    expect(dialog).toHaveTextContent('+1 sensitive')
    expect(dialog).toHaveTextContent(
      'Adding campaigns:send ⚑. 1 member gains it: Daniel Park. Two-factor becomes required for them (7-day grace; Daniel Park signs in through SSO, which already enforces it).',
    )
    expect(dialog).toHaveTextContent('Their pending request for November Product Update stays pending; the policy still matches (audience ≥ 10,000)')
  })

  it('names the requester when the dialog lists several members, and counts several requests', () => {
    const two = impact({
      members: [...impact().members, { id: 'm-2', name: 'Sofia Rossi', email: 'sofia@acme.io', avatarColor: 'ok', gains: ['campaigns:send'], loses: [], effect: 'Gains campaigns:send' }],
    })
    const { unmount } = render(<SaveRoleDialog kind="sensitive" role={role} name="Content Writer" impact={two} busy={false} onBack={noop} onSave={noop} />)
    expect(screen.getByRole('dialog')).toHaveTextContent("Daniel Park's pending request for November Product Update stays pending; the policy still matches (audience ≥ 10,000)")
    unmount()
    const many = impact({
      pendingApprovals: 2,
      pendingRequests: [...impact().pendingRequests, { requester: 'Daniel Park', subject: 'Flash sale', stillMatches: [] }],
    })
    render(<SaveRoleDialog kind="sensitive" role={role} name="Content Writer" impact={many} busy={false} onBack={noop} onSave={noop} />)
    expect(screen.getByRole('dialog')).toHaveTextContent('2 pending requests stay pending; the approval policy still applies')
  })

  it('falls back to what holders can do without pending requests', () => {
    render(<SaveRoleDialog kind="sensitive" role={role} name="Content Writer" impact={impact({ pendingApprovals: 0, pendingRequests: [] })} busy={false} onBack={noop} onSave={noop} />)
    expect(screen.getByRole('dialog')).toHaveTextContent('Holders can send; the approval policy still applies to large or risky sends')
  })
})
