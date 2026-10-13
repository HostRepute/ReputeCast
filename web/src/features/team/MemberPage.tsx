import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { Fragment, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { FullPageState } from '@/components/shell/SystemScreens'
import { ago, logTime, monthDay, usDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { ProviderMark, RoleBadge, SignInText } from '../access/parts'
import { ChangeRoleDialog } from './ChangeRoleDialog'
import { DeactivateDialog } from './DeactivateDialog'
import { LastOwnerDialog } from './LastOwnerDialog'

type Detail = Schemas['TeamMemberDetail']

/** A5 §5.8 member detail (TeamMember board). */
export function MemberPage({ memberId }: { memberId: string }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/members/{memberId}', { params: { path: { workspace: slug, memberId } } })
  const name = q.data?.member.name ?? 'Member'
  return (
    <Page
      title={name}
      breadcrumb={[
        { label: 'Settings', to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: 'Team', to: '/w/$workspace/settings/team', params: { workspace: slug } },
        { label: name },
      ]}
    >
      {q.isError ? (
        q.error instanceof ApiError && q.error.status === 404 ? (
          <FullPageState
            icon="searchLarge"
            badge="Not found"
            title="This member isn't in the workspace"
            actions={
              <ButtonLink to="/w/$workspace/settings/team" params={{ workspace: slug }}>
                Back to Team
              </ButtonLink>
            }
          >
            They may have been removed. The audit log still has their history.
          </FullPageState>
        ) : (
          <ErrorPanel title="Couldn't load this member" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        )
      ) : q.isPending ? (
        <MemberSkeleton />
      ) : (
        <Member d={q.data} />
      )}
    </Page>
  )
}

function Member({ d }: { d: Detail }) {
  const { slug } = useWorkspace()
  const m = d.member
  const now = useNow()
  const first = m.name.split(' ')[0]
  const [dialog, setDialog] = useState<'role' | 'deactivate' | 'remove' | 'lastOwner'>()
  const manageReason = d.canManage ? undefined : `Managing the team needs the team:manage permission.${d.helpers.length ? ` Ask ${d.helpers.join(' or ')}.` : ''}`
  const groups = d.groups
    .map((g) => ({ group: g, rows: d.permissions.filter((p) => p.group === g && !(p.key === 'workspace:transfer' || p.key === 'workspace:delete')) }))
    .filter((g) => g.rows.length > 0)
  return (
    <>
      <section className="card pad flex flex-wrap items-center gap-4.5">
        <Avatar name={m.name} color={m.status === 'active' ? m.avatarColor : 'muted'} size={48} className="text-[13px]!" />
        <div className="col flex-[1_1_320px] gap-1">
          <div className="row gap-2">
            <span className="text-[18px] font-bold">{m.name}</span>
            <RoleBadge role={m.role} />
            {!m.role.builtin ? <span className="k">custom role</span> : null}
            {m.notes.length ? <span className="k">{m.notes.join(' · ')}</span> : null}
            {m.status !== 'active' ? <Badge tone="neutral">Deactivated</Badge> : null}
          </div>
          <div className="row gap-3.5 text-[13px] text-ink-2">
            <span className="mono">{m.email}</span>
            {m.signIn.length ? (
              <span>
                <SignInText methods={m.signIn} />
              </span>
            ) : null}
            <span>
              2FA{' '}
              {m.twoFA.state === 'idp' ? (
                <Badge tone="ok">via IdP</Badge>
              ) : m.twoFA.state === 'on' ? (
                <Badge tone="ok">{m.twoFA.method === 'passkey' ? 'passkey' : 'TOTP'}</Badge>
              ) : m.twoFA.state === 'required' ? (
                <Badge tone="warn">required · off</Badge>
              ) : (
                <Badge tone="neutral">off</Badge>
              )}
            </span>
            {m.status === 'active' ? <span>Last active {ago(m.lastActiveAt, now)}</span> : m.deactivatedAt ? <span>Deactivated {usDate(m.deactivatedAt)}</span> : null}
            <span>Joined {usDate(m.joinedAt)}</span>
          </div>
        </div>
        <div className="row gap-2">
          <Button size="sm" disabledReason={manageReason} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'role')}>
            Change role or scope
          </Button>
          {m.status === 'active' ? (
            <Button size="sm" disabledReason={manageReason ?? (m.you ? 'You can’t deactivate yourself.' : undefined)} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'deactivate')}>
              Deactivate
            </Button>
          ) : null}
          <Button size="sm" variant="danger" disabledReason={manageReason ?? (m.you ? 'Leave the workspace from Account & security instead.' : undefined)} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'remove')}>
            Remove
          </Button>
        </div>
      </section>
      <div className="grid grid-cols-[minmax(0,2fr)_minmax(300px,1fr)] items-start gap-5 max-[1100px]:grid-cols-1">
        <div className="col gap-5">
          <section className="card" aria-labelledby="perm-h">
            <CardSection
              title={<span id="perm-h">Effective permissions</span>}
              caption={`What ${first} can do and where it comes from: role, SSO group mapping or scope`}
            />
            <Table density="compact" className="[&_td]:whitespace-normal" caption="Effective permissions">
              <thead>
                <tr>
                  <Th>Permission</Th>
                  <Th align="center">Effective</Th>
                  <Th>Source</Th>
                  <Th>Scope</Th>
                </tr>
              </thead>
              <tbody>
                {groups.map((g) => (
                  <Fragment key={g.group}>
                    <tr className="bg-ground-2">
                      <Td>
                        <span className="text-[12px] font-bold uppercase tracking-[.04em] text-ink-2">{g.group}</span>
                      </Td>
                      <Td />
                      <Td />
                      <Td />
                    </tr>
                    {g.rows.map((p) => (
                      <tr key={p.key}>
                        <Td>
                          {p.label} <span className="k mono">{p.key}</span>
                          {p.sensitive ? (
                            <>
                              {' '}
                              <span title="Sensitive permission">
                                <SensitiveFlag />
                              </span>
                            </>
                          ) : null}
                        </Td>
                        <Td align="center">{p.granted ? <Badge tone="ok">Granted</Badge> : <Badge tone="neutral">–</Badge>}</Td>
                        <Td>
                          {p.granted ? (
                            <>
                              {p.source}
                              {p.includedBy ? <span className="k"> (included by {p.includedBy.split(':')[1]})</span> : null}
                            </>
                          ) : (
                            <span className="k">{p.source}</span>
                          )}
                        </Td>
                        <Td>
                          <span className="k">{p.granted ? p.scope : '–'}</span>
                        </Td>
                      </tr>
                    ))}
                  </Fragment>
                ))}
              </tbody>
            </Table>
          </section>
          <section className="card" aria-labelledby="act-h">
            <CardSection
              title={<span id="act-h">Recent activity</span>}
              caption="From the audit log"
              actions={
                <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actor: m.userId, range: 'all' }}>
                  Full audit log
                </ButtonLink>
              }
            />
            <Table density="compact" className="[&_td]:whitespace-normal" caption="Recent activity">
              <thead>
                <tr>
                  <Th>When</Th>
                  <Th>Action</Th>
                  <Th>Target</Th>
                  <Th>Result</Th>
                </tr>
              </thead>
              <tbody>
                {d.activity.length === 0 ? (
                  <EmptyRow colSpan={4}>No activity in this workspace yet.</EmptyRow>
                ) : (
                  d.activity.map((a) => (
                    <tr key={a.id}>
                      <Td>{activityTime(a.occurredAt, now)}</Td>
                      <Td>
                        <span className="mono">{a.action}</span>
                      </Td>
                      <Td>{a.target}</Td>
                      <Td>
                        {a.result === 'allowed' ? (
                          <Badge tone="ok">Allowed</Badge>
                        ) : (
                          <span className="row flex-nowrap gap-1.5">
                            <Badge tone="bad">{a.result === 'denied' ? 'Denied' : 'Error'}</Badge>
                            {a.reason ? <span className="k">{a.reason.replace(/^missing /, 'needs ')}</span> : null}
                          </span>
                        )}
                      </Td>
                    </tr>
                  ))
                )}
              </tbody>
            </Table>
          </section>
        </div>
        <div className="col gap-5">
          <KeysCard d={d} />
          <ApprovalsCard d={d} />
          <SessionsCard d={d} manageReason={manageReason} />
        </div>
      </div>
      {dialog === 'role' ? <ChangeRoleDialog member={m} open onOpenChange={(o) => !o && setDialog(undefined)} /> : null}
      {dialog === 'deactivate' ? <DeactivateDialog members={[m]} open onOpenChange={(o) => !o && setDialog(undefined)} /> : null}
      {dialog === 'remove' ? <RemoveDialog member={m} keys={d.apiKeys.length} open onOpenChange={(o) => !o && setDialog(undefined)} /> : null}
      {dialog === 'lastOwner' ? <LastOwnerDialog member={m} open onOpenChange={(o) => !o && setDialog(undefined)} /> : null}
    </>
  )
}

/** "2 h ago" today, "yesterday", then "Oct 8". */
function activityTime(at: string, now: Date) {
  const days = (now.getTime() - new Date(at).getTime()) / 86_400_000
  if (days < 2) return ago(at, now)
  return monthDay(at)
}

function KeysCard({ d }: { d: Detail }) {
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="keys-h">
      <CardSection title={<span id="keys-h">API keys they created</span>} />
      <div className="pad">
        {d.apiKeys.length === 0 ? (
          <div className="note n">
            <span>
              {d.keysNote?.includes('apikeys:manage') ? (
                <>
                  None. Creating API keys needs <span className="mono">apikeys:manage</span>, which this role doesn&apos;t hold.
                </>
              ) : (
                (d.keysNote ?? 'None yet.')
              )}
            </span>
          </div>
        ) : (
          <ul className="m-0 flex list-none flex-col gap-2.5 p-0">
            {d.apiKeys.map((k) => (
              <li key={k.id} className="flex flex-col gap-0.5">
                <span className="row gap-2">
                  <Link to="/w/$workspace/settings/keys" params={{ workspace: slug }} className="font-semibold text-ink no-underline hover:text-accent">
                    {k.name}
                  </Link>
                  {k.kind === 'mcp_agent' ? <Badge tone="purple">MCP agent</Badge> : null}
                  {k.status === 'expired' ? <Badge tone="bad">Expired</Badge> : null}
                </span>
                <span className="k mono">
                  {k.masked} · {k.permissions.join(', ')} · {k.scope}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}

function ApprovalsCard({ d }: { d: Detail }) {
  const { slug } = useWorkspace()
  const pending = d.approvalsRequested.filter((a) => a.status === 'pending')
  const given = d.approvalsGiven
  return (
    <section className="card" aria-labelledby="ap-h">
      <CardSection title={<span id="ap-h">Approvals</span>} />
      <div className="pad col gap-2">
        <KeyValues
          items={[
            [
              'Given',
              given.length > 0
                ? `${given.length} · last ${given[0]!.campaign || 'campaign'} ${monthDay(given[0]!.at)}`
                : d.canApprove
                  ? 'None yet'
                  : "None · can't approve (no campaigns:approve)",
            ],
            [
              'Requested',
              pending.length > 0 ? (
                <>
                  {pending.length} pending ·{' '}
                  {pending[0]!.campaignId ? (
                    <Link to="/w/$workspace/approvals" params={{ workspace: slug }}>
                      {pending[0]!.campaign}
                    </Link>
                  ) : (
                    pending[0]!.campaign
                  )}
                </>
              ) : d.approvalsRequested.length > 0 ? (
                `${d.approvalsRequested.length} · none pending`
              ) : (
                'None'
              ),
            ],
          ]}
        />
      </div>
    </section>
  )
}

function SessionsCard({ d, manageReason }: { d: Detail; manageReason?: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const m = d.member
  const signOut = $api.useMutation('post', '/api/v1/workspaces/{workspace}/members/{memberId}/sign-out')
  const sso = m.signIn.find((s) => s.kind === 'sso')
  const methods = m.signIn.length
    ? m.signIn.map((s) => (s.kind === 'sso' && d.ssoDomain ? `${s.label} (${d.ssoDomain})` : s.label)).join(' · ')
    : 'None (deactivated)'
  const twoFA = { on: m.twoFA.method === 'passkey' ? 'Passkey' : 'Authenticator app (TOTP)', idp: 'Enforced by the IdP', required: 'Required, not set up yet', off: 'Off' }[m.twoFA.state]
  return (
    <section className="card" aria-labelledby="ss-h">
      <CardSection title={<span id="ss-h">Sign-in and sessions</span>} />
      <div className="pad col gap-2">
        <KeyValues
          items={[
            [
              'Methods',
              <>
                {sso ? (
                  <>
                    <ProviderMark preset={sso.preset ?? 'oidc'} />{' '}
                  </>
                ) : null}
                {methods}
              </>,
            ],
            ['Two-factor', twoFA],
            [
              'Sessions',
              <>
                {d.sessions} active ·{' '}
                <Button
                  size="sm"
                  disabledReason={manageReason ?? (d.sessions === 0 ? 'No active sessions.' : undefined)}
                  loading={signOut.isPending}
                  onClick={() =>
                    signOut.mutate(
                      { params: { path: { workspace: slug, memberId: m.id } } },
                      {
                        onSuccess: (r) => {
                          toast({ title: `${m.name} signed out`, detail: `${r.sessionsRevoked} ${r.sessionsRevoked === 1 ? 'session' : 'sessions'} ended` })
                          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}'] })
                        },
                        onError: (e) => toast({ kind: 'error', title: "Couldn't sign out", detail: errorMessage(e) }),
                      },
                    )
                  }
                >
                  Force sign-out
                </Button>
              </>,
            ],
          ]}
        />
      </div>
    </section>
  )
}

function RemoveDialog({ member, keys, open, onOpenChange }: { member: Schemas['TeamMember']; keys: number; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/members/{memberId}')
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Dialog · remove member</span>}
      title={`Remove ${member.name} from ${me.workspace.name}?`}
      description={<span className="text-[13.5px]">The membership is deleted; the audit log still names them. Deactivating instead keeps the membership so they can come back.</span>}
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            variant="danger-primary"
            loading={remove.isPending}
            onClick={() =>
              remove.mutate(
                { params: { path: { workspace: slug, memberId: member.id } } },
                {
                  onSuccess: () => {
                    toast({ kind: 'deleted', title: `${member.name} removed` })
                    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
                    void navigate({ to: '/w/$workspace/settings/team', params: { workspace: slug } })
                  },
                  onError: (e) => toast({ kind: 'error', title: `Couldn't remove ${member.name}`, detail: errorMessage(e) }),
                },
              )
            }
          >
            Remove {member.name.split(' ')[0]}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>Loses access to {me.workspace.name} at once; other workspaces are unaffected</li>
        {keys > 0 ? (
          <li>
            {keys} API {keys === 1 ? 'key or agent they created is' : 'keys or agents they created are'} revoked
          </li>
        ) : null}
        <li>Campaigns, templates and approvals they made stay in the workspace</li>
      </ul>
    </Dialog>
  )
}

function MemberSkeleton() {
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        Loading the member
      </span>
      <section className="card pad flex items-center gap-4.5">
        <Skeleton w={48} h={48} r={24} />
        <div className="flex flex-col gap-2">
          <Skeleton w={220} h={18} />
          <Skeleton w={380} h={12} />
        </div>
      </section>
      <div className="grid grid-cols-[minmax(0,2fr)_minmax(300px,1fr)] items-start gap-5 max-[1100px]:grid-cols-1">
        <section className="card pad flex flex-col gap-3">
          {Array.from({ length: 10 }, (_, i) => (
            <Skeleton key={i} h={12} w={`${90 - (i % 4) * 10}%`} />
          ))}
        </section>
        <section className="card pad flex flex-col gap-3">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} h={12} w={`${80 - i * 8}%`} />
          ))}
        </section>
      </div>
    </div>
  )
}

export { logTime }
