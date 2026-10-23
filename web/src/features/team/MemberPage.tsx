import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { Fragment, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { KeyValues } from '@/components/ui/Data'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { MemberIntegrations, useMemberIntegrations } from '@/features/integrations/MemberIntegrations'
import { FullPageState } from '@/components/shell/SystemScreens'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, count, monthDay, orList, usDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { m as am } from '../access/messages'
import { AccessTable, GroupRow, RoleBadge, SignInText } from '../access/parts'
import { ChangeRoleDialog } from './ChangeRoleDialog'
import { DeactivateDialog } from './DeactivateDialog'
import { LastOwnerDialog } from './LastOwnerDialog'
import { m as mm } from './member.messages'
import { m as tm } from './messages'
import { AgencyMemberNote } from '@/features/agency/WorkspaceParts'

type Detail = Schemas['TeamMemberDetail']

/** A5 §5.8 member detail (TeamMember board). */
export function MemberPage({ memberId }: { memberId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/members/{memberId}', { params: { path: { workspace: slug, memberId } } })
  const name = q.data?.member.name ?? t(mm.member)
  return (
    <Page
      title={name}
      breadcrumb={[
        { label: t(mm.crumbSettings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(mm.crumbTeam), to: '/w/$workspace/settings/team', params: { workspace: slug } },
        { label: name },
      ]}
    >
      {q.isError ? (
        q.error instanceof ApiError && q.error.status === 404 ? (
          <FullPageState
            icon="searchLarge"
            badge={t(mm.notFoundBadge)}
            title={t(mm.notFoundTitle)}
            actions={
              <ButtonLink to="/w/$workspace/settings/team" params={{ workspace: slug }}>
                {t(mm.backToTeam)}
              </ButtonLink>
            }
          >
            {t(mm.notFoundBody)}
          </FullPageState>
        ) : (
          <ErrorPanel title={t(mm.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
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
  const t = useT()
  const { slug } = useWorkspace()
  const m = d.member
  const now = useNow()
  const first = m.name.split(' ')[0] ?? m.name
  const [dialog, setDialog] = useState<'role' | 'deactivate' | 'remove' | 'lastOwner'>()
  const manageReason = d.canManage ? undefined : d.helpers.length ? t(mm.manageNeedsAsk, { helpers: orList(d.helpers) }) : t(mm.manageNeeds)
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
            {!m.role.builtin ? <span className="k">{t(mm.customRole)}</span> : null}
            {m.notes.length ? <span className="k">{m.notes.join(' · ')}</span> : null}
            {m.status !== 'active' ? <Badge tone="neutral">{t(tm.deactivatedBadge)}</Badge> : null}
          </div>
          <div className="row gap-3.5 text-[13px] text-ink-2">
            <span className="mono">{m.email}</span>
            {m.signIn.length ? (
              <span>
                <SignInText methods={m.signIn} />
              </span>
            ) : null}
            <span>
              {'2FA'}{' '}
              {m.twoFA.state === 'idp' ? (
                <Badge tone="ok">{t(mm.twoFAIdp)}</Badge>
              ) : m.twoFA.state === 'on' ? (
                <Badge tone="ok">{t(mm.twoFAOn, { method: m.twoFA.method === 'passkey' ? 'passkey' : 'totp' })}</Badge>
              ) : m.twoFA.state === 'required' ? (
                <Badge tone="warn">{t(mm.twoFARequired)}</Badge>
              ) : (
                <Badge tone="neutral">{t(mm.twoFAOff)}</Badge>
              )}
            </span>
            {m.status === 'active' ? (
              <span>{t(mm.lastActive, { ago: ago(m.lastActiveAt, now) })}</span>
            ) : m.deactivatedAt ? (
              <span>{t(mm.deactivatedOn, { date: usDate(m.deactivatedAt) })}</span>
            ) : null}
            <span>{t(mm.joined, { date: usDate(m.joinedAt) })}</span>
          </div>
          {/* Assignments from an agency say where they come from (agency accounts). */}
          <AgencyMemberNote userId={m.userId} source={m.source} />
        </div>
        <div className="row gap-2">
          <Button size="sm" disabledReason={manageReason} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'role')}>
            {t(mm.changeRole)}
          </Button>
          {m.status === 'active' ? (
            <Button size="sm" disabledReason={manageReason ?? (m.you ? t(mm.cantDeactivateSelf) : undefined)} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'deactivate')}>
              {t(tm.deactivate)}
            </Button>
          ) : null}
          <Button size="sm" variant="danger" disabledReason={manageReason ?? (m.you ? t(mm.leaveInstead) : undefined)} onClick={() => setDialog(m.lastOwner ? 'lastOwner' : 'remove')}>
            {t(mm.remove)}
          </Button>
        </div>
      </section>
      <div className="grid grid-cols-[minmax(0,2fr)_minmax(300px,1fr)] items-start gap-5 max-md:grid-cols-1">
        <div className="col gap-5">
          <section className="card" aria-labelledby="perm-h">
            <CardSection title={<span id="perm-h">{t(mm.permissionsTitle)}</span>} caption={t(mm.permissionsCaption, { name: first })} />
            <AccessTable caption={t(mm.permissionsTitle)}>
              <thead>
                <tr>
                  <Th>{t(mm.colPermission)}</Th>
                  <Th align="center">{t(mm.colEffective)}</Th>
                  <Th>{t(mm.colSource)}</Th>
                  <Th>{t(mm.colScope)}</Th>
                </tr>
              </thead>
              <tbody>
                {groups.map((g) => (
                  <Fragment key={g.group}>
                    <GroupRow label={g.group} cells={3} />
                    {g.rows.map((p) => (
                      <tr key={p.key}>
                        <Td>
                          {p.label} <span className="k mono">{p.key}</span>
                          {p.sensitive ? (
                            <>
                              {' '}
                              <span title={t(mm.sensitive)}>
                                <SensitiveFlag>{t(mm.sensitiveFlag)}</SensitiveFlag>
                              </span>
                            </>
                          ) : null}
                        </Td>
                        <Td align="center">{p.granted ? <Badge tone="ok">{t(mm.granted)}</Badge> : <Badge tone="neutral">–</Badge>}</Td>
                        <Td>
                          {p.granted ? (
                            <>
                              {p.source}
                              {p.includedBy ? (
                                <span className="k">
                                  {' '}
                                  {t(mm.includedBy, { action: p.includedBy.split(':')[1] ?? '' })}
                                </span>
                              ) : null}
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
            </AccessTable>
          </section>
          <section className="card" aria-labelledby="act-h">
            <CardSection
              title={<span id="act-h">{t(mm.activityTitle)}</span>}
              caption={t(mm.activityCaption)}
              actions={
                <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actor: m.userId, range: 'all' }}>
                  {t(mm.fullAuditLog)}
                </ButtonLink>
              }
            />
            <AccessTable caption={t(mm.activityTitle)}>
              <thead>
                <tr>
                  <Th>{t(mm.colWhen)}</Th>
                  <Th>{t(mm.colAction)}</Th>
                  <Th>{t(mm.colTarget)}</Th>
                  <Th>{t(mm.colResult)}</Th>
                </tr>
              </thead>
              <tbody>
                {d.activity.length === 0 ? (
                  <EmptyRow colSpan={4}>{t(mm.noActivity)}</EmptyRow>
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
                          <Badge tone="ok">{t(mm.result, { result: a.result })}</Badge>
                        ) : (
                          <span className="row flex-nowrap gap-1.5">
                            <Badge tone="bad">{t(mm.result, { result: a.result })}</Badge>
                            {a.reason ? <span className="k">{reasonText(a.reason)}</span> : null}
                          </span>
                        )}
                      </Td>
                    </tr>
                  ))
                )}
              </tbody>
            </AccessTable>
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

/** Why an entry was denied: the server's "missing campaigns:send" reads "needs campaigns:send". */
function reasonText(reason: string) {
  const missing = 'missing '
  return reason.startsWith(missing) ? tt(mm.needs, { permission: reason.slice(missing.length) }) : reason
}

/**
 * A sign-in method in the member's list, linked identities included:
 * "Password · passkey · Acme SSO (acme.io)". Password and passkey read in
 * the app language, as in the team list; the others are names.
 */
function methodLabel(s: Schemas['MemberSignInMethod'], i: number, ssoDomain?: string) {
  if (s.kind === 'password') return tt(am.signInPassword)
  if (s.kind === 'passkey') return tt(i === 0 ? am.signInPasskeyFirst : am.signInPasskey)
  return s.kind === 'sso' && ssoDomain ? `${s.label} (${ssoDomain})` : s.label
}

function KeysCard({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="keys-h">
      <CardSection title={<span id="keys-h">{t(mm.keysTitle)}</span>} />
      <div className="pad">
        {d.apiKeys.length === 0 ? (
          <div className="note n">
            <span>
              {/* The server's two notes read in the app language; any other shows as it comes. */}
              {d.keysNote?.includes('apikeys:manage') ? (
                <T msg={mm.keysNoPermission} values={{ mono: (c) => <span className="mono">{c}</span> }} />
              ) : d.keysNote === undefined || d.keysNote === 'None yet.' ? (
                t(mm.keysNoneYet)
              ) : (
                d.keysNote
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
                  {k.kind === 'mcp_agent' ? <Badge tone="purple">{t(mm.mcpAgent)}</Badge> : null}
                  {k.status === 'expired' ? <Badge tone="bad">{t(mm.expired)}</Badge> : null}
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
  const t = useT()
  const { slug } = useWorkspace()
  const pending = d.approvalsRequested.filter((a) => a.status === 'pending')
  const given = d.approvalsGiven
  const last = given[0]
  const next = pending[0]
  return (
    <section className="card" aria-labelledby="ap-h">
      <CardSection title={<span id="ap-h">{t(mm.approvalsTitle)}</span>} />
      <div className="pad col gap-2">
        <KeyValues
          items={[
            [
              t(mm.given),
              last
                ? last.campaign
                  ? t(mm.givenLast, { count: count(given.length), campaign: last.campaign, date: monthDay(last.at) })
                  : t(mm.givenLastUnnamed, { count: count(given.length), date: monthDay(last.at) })
                : d.canApprove
                  ? t(mm.givenNone)
                  : t(mm.givenCantApprove),
            ],
            [
              t(mm.requested),
              next ? (
                <T
                  msg={mm.requestedPending}
                  values={{
                    count: count(pending.length),
                    campaign: next.campaign,
                    link: (c) =>
                      next.campaignId ? (
                        <Link to="/w/$workspace/approvals" params={{ workspace: slug }}>
                          {c}
                        </Link>
                      ) : (
                        c
                      ),
                  }}
                />
              ) : d.approvalsRequested.length > 0 ? (
                t(mm.requestedNonePending, { count: count(d.approvalsRequested.length) })
              ) : (
                t(common.none)
              ),
            ],
          ]}
        />
      </div>
    </section>
  )
}

function SessionsCard({ d, manageReason }: { d: Detail; manageReason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const m = d.member
  const signOut = $api.useMutation('post', '/api/v1/workspaces/{workspace}/members/{memberId}/sign-out')
  // Every method, linked identities included (the team list shows the primary ones).
  const methods = d.signInMethods.length ? d.signInMethods.map((s, i) => methodLabel(s, i, d.ssoDomain)).join(' · ') : t(mm.methodsNone)
  const twoFA = t(mm.twoFactorState, { state: m.twoFA.state === 'on' ? (m.twoFA.method === 'passkey' ? 'passkey' : 'totp') : m.twoFA.state })
  return (
    <section className="card" aria-labelledby="ss-h">
      <CardSection title={<span id="ss-h">{t(mm.sessionsTitle)}</span>} />
      <div className="pad col gap-2">
        <KeyValues
          items={[
            [t(mm.methods), methods],
            [t(mm.twoFactor), twoFA],
            [
              t(mm.sessions),
              <>
                {t(mm.sessionsActive, { count: count(d.sessions) })} ·{' '}
                <Button
                  size="sm"
                  disabledReason={manageReason ?? (d.sessions === 0 ? t(mm.noActiveSessions) : undefined)}
                  loading={signOut.isPending}
                  onClick={() =>
                    signOut.mutate(
                      { params: { path: { workspace: slug, memberId: m.id } } },
                      {
                        onSuccess: (r) => {
                          toast({ title: t(mm.signedOut, { name: m.name }), detail: t(mm.sessionsEnded, { count: r.sessionsRevoked }) })
                          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}'] })
                        },
                        onError: (e) => toast({ kind: 'error', title: t(mm.signOutFailed), detail: errorMessage(e) }),
                      },
                    )
                  }
                >
                  {t(mm.forceSignOut)}
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
  const t = useT()
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const integrations = useMemberIntegrations([member.userId])
  const first = member.name.split(' ')[0] ?? member.name
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="bad"
      badge={t(mm.removeBadge)}
      title={t(mm.removeTitle, { name: member.name, workspace: me.workspace.name })}
      description={t(mm.removeDescription)}
      confirmLabel={t(mm.removeConfirm, { name: first })}
      busyLabel={t(mm.removing)}
      onConfirm={async () => {
        await integrations.apply('')
        await http.DELETE('/api/v1/workspaces/{workspace}/members/{memberId}', { params: { path: { workspace: slug, memberId: member.id } } })
        toast({ kind: 'deleted', title: t(mm.removed, { name: member.name }), detail: t(mm.loggedAs, { action: 'member.removed' }) })
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
        void navigate({ to: '/w/$workspace/settings/team', params: { workspace: slug } })
      }}
    >
      <DialogRow tone="bad" label={t(mm.rowAccess)} first>
        {t(mm.removeAccess, { workspace: me.workspace.name })}
      </DialogRow>
      {keys > 0 ? (
        <DialogRow tone="bad" label={t(mm.rowKeys)}>
          {t(mm.removeKeys, { count: keys })}
        </DialogRow>
      ) : null}
      <DialogRow tone="ok" label={t(mm.rowKeeps)}>
        {t(mm.removeKeeps)}
      </DialogRow>
      <MemberIntegrations state={integrations} name={first} />
    </ConfirmDialog>
  )
}

function MemberSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        {t(mm.loading)}
      </span>
      <section className="card pad flex items-center gap-4.5">
        <Skeleton w={48} h={48} r={24} />
        <div className="flex flex-col gap-2">
          <Skeleton w={220} h={18} />
          <Skeleton w={380} h={12} />
        </div>
      </section>
      <div className="grid grid-cols-[minmax(0,2fr)_minmax(300px,1fr)] items-start gap-5 max-md:grid-cols-1">
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
