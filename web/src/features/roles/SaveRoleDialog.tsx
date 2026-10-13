import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { andList } from '@/lib/format'
import { AccessTable, PersonCell } from '../access/parts'

export type SaveKind = 'sensitive' | 'remove'

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`

/**
 * Confirms a role change before it reaches the holders (RoleEditorStates):
 * adding a sensitive permission (two-factor becomes required) or removing
 * permissions that members and their keys use.
 */
export function SaveRoleDialog({
  kind,
  role,
  name,
  impact,
  busy,
  onBack,
  onSave,
}: {
  kind: SaveKind
  role: Schemas['RoleSummary']
  name: string
  impact: Schemas['RoleImpact']
  busy: boolean
  onBack: () => void
  onSave: () => void
}) {
  const affected = impact.members.filter((m) => m.gains.length + m.loses.length > 0)
  const approvals =
    impact.pendingApprovals > 0
      ? `${plural(impact.pendingApprovals, 'pending request stays', 'pending requests stay')} pending; the approval policy still applies`
      : impact.approvals
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onBack()}
      eyebrow={<span className="k">Dialog · before saving · {role.name}</span>}
      title={`Save changes to ${name || role.name}?`}
      hideClose
      dismissible={false}
      size={kind === 'remove' ? 'lg' : 'md'}
      footer={
        <>
          <Button onClick={onBack}>Back to editing</Button>
          {kind === 'remove' ? (
            <Button variant="danger-primary" loading={busy} onClick={onSave}>
              Save and remove access
            </Button>
          ) : (
            <Button variant="primary" loading={busy} onClick={onSave}>
              Save role
            </Button>
          )}
        </>
      }
    >
      {kind === 'sensitive' ? (
        <>
          <div className="note w">
            <Badge tone="warn">+{impact.addedSensitive.length} sensitive</Badge>
            <span>
              Adding{' '}
              {impact.addedSensitive.map((k, i) => (
                <span key={k}>
                  {i > 0 ? ', ' : null}
                  <span className="mono">{k}</span>
                </span>
              ))}{' '}
              <span aria-hidden="true">⚑</span>. {plural(affected.length, 'member gains', 'members gain')} it: <b>{andList(affected.map((m) => m.name))}</b>.{' '}
              {twoFactorSentence(affected, impact.graceDays)}
            </span>
          </div>
          <KeyValues
            items={[
              ['Members affected', affected.length ? `${affected.length} · ${andList(affected.map((m) => m.name))}` : '0'],
              ['API keys', String(impact.apiKeys)],
              ['MCP agents', String(impact.agents)],
              ['Approvals', approvals],
            ]}
          />
        </>
      ) : (
        <>
          <div className="note b">
            <Badge tone="bad">−{plural(impact.removed.length, 'permission', 'permissions')}</Badge>
            <span>
              Removing{' '}
              {impact.removed.map((k, i) => (
                <span key={k}>
                  {i > 0 ? ', ' : null}
                  <span className="mono">{k}</span>
                </span>
              ))}
              . <b>{impact.summary}</b>
            </span>
          </div>
          <AccessTable caption="Who loses access">
            <thead>
              <tr>
                <Th>Who</Th>
                <Th>Loses</Th>
                <Th>Effect</Th>
              </tr>
            </thead>
            <tbody>
              {impact.members.map((m) => (
                <tr key={m.id}>
                  <Td>
                    <PersonCell name={m.name} sub={m.email} color={m.avatarColor} />
                  </Td>
                  <Td>
                    <span className="mono">{m.loses.join(', ') || '–'}</span>
                  </Td>
                  <Td>{m.effect}</Td>
                </tr>
              ))}
              {impact.keys.map((k) => (
                <tr key={k.id}>
                  <Td>
                    <span className="font-semibold">
                      {k.kind === 'mcp_agent' ? 'MCP agent' : 'API key'} “{k.name}”
                    </span>
                  </Td>
                  <Td>
                    <span className="mono">{k.loses.join(', ')}</span>
                  </Td>
                  <Td>{k.effect}</Td>
                </tr>
              ))}
            </tbody>
          </AccessTable>
          <Note tone="neutral">Scheduled campaigns created by these members keep their schedule. Keys that never had these permissions are unaffected.</Note>
        </>
      )}
    </Dialog>
  )
}

/** "Two-factor becomes required for them (7-day grace; Daniel Park's IdP already enforces it)." */
function twoFactorSentence(members: Schemas['RoleImpactMember'][], grace: number) {
  const needs = members.filter((m) => m.effect.startsWith('Two-factor becomes required'))
  if (needs.length === 0) return 'They already use two-factor, so nothing changes at sign-in.'
  const idp = needs.filter((m) => m.effect.includes('IdP'))
  const who = needs.length === 1 ? 'them' : 'these members'
  const idpPart = idp.length ? `; ${andList(idp.map((m) => m.name))} ${idp.length === 1 ? 'signs' : 'sign'} in through SSO, which already enforces it` : ''
  return `Two-factor becomes required for ${who} (${grace}-day grace${idpPart}).`
}
