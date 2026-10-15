import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { formatList, T, t, useT } from '@/i18n'
import { AccessTable, PersonCell } from '../access/parts'
import { m } from './messages'

export type SaveKind = 'sensitive' | 'remove'

/** Permission keys in mono, separated by commas. */
function keyList(keys: string[]): ReactNode {
  return keys.map((k, i) => (
    <span key={k}>
      {i > 0 ? ', ' : null}
      <span className="mono">{k}</span>
    </span>
  ))
}

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
  const tr = useT()
  const affected = impact.members.filter((x) => x.gains.length + x.loses.length > 0)
  const approvals = pendingLine(impact, affected)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onBack()}
      eyebrow={<span className="k">{tr(m.dialogEyebrow, { name: role.name })}</span>}
      title={tr(m.dialogTitle, { name: name || role.name })}
      hideClose
      dismissible={false}
      size={kind === 'remove' ? 'lg' : 'md'}
      footer={
        <>
          <Button onClick={onBack}>{tr(m.backToEditing)}</Button>
          {kind === 'remove' ? (
            <Button variant="danger-primary" loading={busy} onClick={onSave}>
              {tr(m.saveAndRemove)}
            </Button>
          ) : (
            <Button variant="primary" loading={busy} onClick={onSave}>
              {tr(m.saveRole)}
            </Button>
          )}
        </>
      }
    >
      {kind === 'sensitive' ? (
        <>
          <div className="note w">
            <Badge tone="warn">{tr(m.addedSensitiveBadge, { count: impact.addedSensitive.length })}</Badge>
            <span>
              <T
                msg={m.addingSensitive}
                values={{
                  keys: keyList(impact.addedSensitive),
                  flag: (chunks) => <span aria-hidden="true">{chunks}</span>,
                  count: affected.length,
                  names: formatList(affected.map((x) => x.name)),
                  b: (chunks) => <b>{chunks}</b>,
                  twoFactor: twoFactorSentence(affected, impact.graceDays),
                }}
              />
            </span>
          </div>
          <KeyValues
            items={[
              [tr(m.membersAffected), affected.length ? tr(m.impactMembersList, { count: affected.length, names: formatList(affected.map((x) => x.name)) }) : '0'],
              [tr(m.impactKeys), String(impact.apiKeys)],
              [tr(m.impactAgents), String(impact.agents)],
              [tr(m.impactApprovals), approvals],
            ]}
          />
        </>
      ) : (
        <>
          <div className="note b">
            <Badge tone="bad">{tr(m.removedBadge, { count: impact.removed.length })}</Badge>
            <span>
              <T msg={m.removing} values={{ keys: keyList(impact.removed), summary: impact.summary, b: (chunks) => <b>{chunks}</b> }} />
            </span>
          </div>
          <AccessTable caption={tr(m.whoLosesCaption)}>
            <thead>
              <tr>
                <Th>{tr(m.colWho)}</Th>
                <Th>{tr(m.colLoses)}</Th>
                <Th>{tr(m.colEffect)}</Th>
              </tr>
            </thead>
            <tbody>
              {impact.members.map((x) => (
                <tr key={x.id}>
                  <Td>
                    <PersonCell name={x.name} sub={x.email} color={x.avatarColor} />
                  </Td>
                  <Td>
                    <span className="mono">{x.loses.join(', ') || '–'}</span>
                  </Td>
                  <Td>{x.effect}</Td>
                </tr>
              ))}
              {impact.keys.map((k) => (
                <tr key={k.id}>
                  <Td>
                    <span className="font-semibold">{k.kind === 'mcp_agent' ? tr(m.keyAgent, { name: k.name }) : tr(m.keyApi, { name: k.name })}</span>
                  </Td>
                  <Td>
                    <span className="mono">{k.loses.join(', ')}</span>
                  </Td>
                  <Td>{k.effect}</Td>
                </tr>
              ))}
            </tbody>
          </AccessTable>
          <Note tone="neutral">{tr(m.keepSchedule)}</Note>
        </>
      )}
    </Dialog>
  )
}

/**
 * What happens to the holders' pending approval requests: one request names
 * what it's for and the rules that keep requiring approval ("Their pending
 * request for November Product Update stays pending; the policy still
 * matches (audience ≥ 10,000)").
 */
function pendingLine(impact: Schemas['RoleImpact'], affected: Schemas['RoleImpactMember'][]): string {
  const pending = impact.pendingRequests
  if (pending.length > 1) return t(m.pendingStay, { count: pending.length })
  const p = pending[0]
  if (!p) return impact.approvals
  const rules = p.stillMatches.join(' · ')
  // "Their" when the dialog names that one member just above.
  if (affected.length === 1 && affected[0]!.name === p.requester) {
    return rules ? t(m.pendingTheir, { subject: p.subject, rules }) : t(m.pendingTheirNoRule, { subject: p.subject })
  }
  return rules ? t(m.pendingNamed, { name: p.requester, subject: p.subject, rules }) : t(m.pendingNamedNoRule, { name: p.requester, subject: p.subject })
}

/** "Two-factor becomes required for them (7-day grace; Daniel Park signs in through SSO, which already enforces it)." */
function twoFactorSentence(members: Schemas['RoleImpactMember'][], grace: number): string {
  const needs = members.filter((x) => x.effect.startsWith('Two-factor becomes required'))
  if (needs.length === 0) return t(m.twoFactorNone)
  const idp = needs.filter((x) => x.effect.includes('IdP'))
  if (idp.length === 0) return t(m.twoFactorRequired, { count: needs.length, grace })
  return t(m.twoFactorRequiredIdp, { count: needs.length, grace, names: formatList(idp.map((x) => x.name)), idp: idp.length })
}
