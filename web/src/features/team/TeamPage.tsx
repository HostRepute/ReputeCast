import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { GoogleMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { BulkResultDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { ago, andList, inDays, monthDay, usDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AccessTable, copyText, PersonCell, RoleBadge, SignInText, TwoFABadge } from '../access/parts'
import { DeactivateDialog } from './DeactivateDialog'
import { InviteDialog } from './InviteDialog'

export interface TeamSearch {
  q?: string
  role?: string
  status?: 'active' | 'invited' | 'deactivated'
  twofa?: 'on' | 'off' | 'idp' | 'required'
}

type Member = Schemas['TeamMember']

/** A5 §5.8 Team: members with role, scope, sign-in, two-factor and status; pending invitations. */
export function TeamPage({ search }: { search: TeamSearch }) {
  const { slug, me } = useWorkspace()
  const [inviteOpen, setInviteOpen] = useState(false)
  const members = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/members',
    {
      params: {
        path: { workspace: slug },
        query: { q: search.q, role: search.role, status: search.status === 'invited' ? undefined : search.status, twofa: search.twofa },
      },
    },
    { placeholderData: keepPreviousData },
  )
  const invitations = $api.useQuery('get', '/api/v1/workspaces/{workspace}/invitations', { params: { path: { workspace: slug } } })
  const canInvite = members.data?.canInvite ?? me.permissions.includes('team:invite')
  return (
    <Page
      title="Settings"
      tabs={settingsTabs(slug)}
      actions={
        canInvite ? (
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            Invite member
          </Button>
        ) : null
      }
    >
      {members.data && !members.data.canInvite ? <ReadOnlyNote helpers={members.data.reason} /> : null}
      {members.isError ? (
        <ErrorPanel title="Couldn't load the team" actions={<Button onClick={() => members.refetch()}>Retry</Button>}>
          {errorMessage(members.error)} Nothing was changed.
        </ErrorPanel>
      ) : members.isPending ? (
        <TeamSkeleton />
      ) : (
        <MembersCard list={members.data} search={search} hideMembers={search.status === 'invited'} />
      )}
      {invitations.isError ? (
        <ErrorPanel title="Couldn't load the invitations" actions={<Button onClick={() => invitations.refetch()}>Retry</Button>}>
          {errorMessage(invitations.error)}
        </ErrorPanel>
      ) : invitations.data ? (
        <InvitationsCard list={invitations.data} onInvite={() => setInviteOpen(true)} />
      ) : null}
      {inviteOpen ? <InviteDialog open onOpenChange={setInviteOpen} /> : null}
    </Page>
  )
}

/** "You can view the team. Inviting needs the team:invite permission…" */
function ReadOnlyNote({ helpers }: { helpers?: string }) {
  const ask = helpers?.match(/Ask (.+)\.$/)?.[1]
  return (
    <div className="note n" role="status">
      <Badge tone="neutral">Read-only</Badge>
      <span>
        You can view the team. Inviting needs the <span className="mono">team:invite</span> permission and changing roles needs{' '}
        <span className="mono">team:manage</span>.{ask ? ` Ask ${ask}.` : ''}
      </span>
    </div>
  )
}

function MembersCard({ list, search, hideMembers }: { list: Schemas['TeamMemberList']; search: TeamSearch; hideMembers: boolean }) {
  const { slug, me } = useWorkspace()
  const now = useNow()
  const [selected, setSelected] = useState<string[]>([])
  const items = hideMembers ? [] : list.items
  const manage = list.canManage
  const filtered = Boolean(search.q || search.role || search.status || search.twofa)
  return (
    <section className="card" aria-labelledby="members-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="members-h">
            Members
          </h2>
          <span className="k">Everyone with access to {me.workspace.name}</span>
        </div>
        {manage ? (
          <Filters list={list} search={search} />
        ) : (
          <span className="k">
            {list.counts.members} {list.counts.members === 1 ? 'member' : 'members'} · {list.counts.invited} invited
          </span>
        )}
      </div>
      {manage ? <BulkBar list={list} selected={selected} setSelected={setSelected} /> : null}
      <AccessTable caption="Members">
        <thead>
          <tr>
            <Th>Member</Th>
            <Th>Role · scope</Th>
            <Th>Sign-in</Th>
            <Th>2FA</Th>
            <Th>Last active</Th>
            <Th>Status</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {items.length === 0 ? (
            <EmptyRow colSpan={7}>
              {hideMembers ? 'Invited people appear under Pending invitations below.' : filtered ? 'No members match these filters.' : 'No members yet.'}
            </EmptyRow>
          ) : (
            items.map((m) => (
              <MemberRow
                key={m.id}
                m={m}
                now={now}
                manage={manage}
                selected={selected.includes(m.id)}
                onSelect={(on) => setSelected((s) => (on ? [...s, m.id] : s.filter((x) => x !== m.id)))}
                slug={slug}
              />
            ))
          )}
        </tbody>
      </AccessTable>
    </section>
  )
}

function MemberRow({
  m,
  now,
  manage,
  selected,
  onSelect,
  slug,
}: {
  m: Member
  now: Date
  manage: boolean
  selected: boolean
  onSelect: (on: boolean) => void
  slug: string
}) {
  const active = m.status === 'active'
  const reactivate = useReactivate()
  return (
    <tr style={active ? undefined : { color: '#5B6472' }}>
      <Td>
        <div className="flex min-w-0 items-center gap-2.5">
          {manage ? (
            <input type="checkbox" checked={selected} onChange={(e) => onSelect(e.target.checked)} aria-label={`Select ${m.name}`} />
          ) : null}
          <PersonCell name={m.name} sub={m.email} color={active ? m.avatarColor : 'muted'} />
        </div>
      </Td>
      <Td>
        {/* As on the board, the space before each note is outside it (body size, not 12px). */}
        <RoleBadge role={m.role} />
        {!m.role.builtin ? (
          <>
            {' '}
            <span className="k">custom</span>
          </>
        ) : null}
        {m.scope.label ? (
          <>
            {' '}
            <span className="k">· {m.scope.label}</span>
          </>
        ) : null}
        {m.notes.length > 0 ? (
          <>
            {' '}
            <span className="k">{m.notes.join(' · ')}</span>
          </>
        ) : null}
      </Td>
      <Td>
        <SignInText methods={m.signIn} />
      </Td>
      <Td>{active ? <TwoFABadge twoFA={m.twoFA} /> : <span className="k">–</span>}</Td>
      <Td>{active ? ago(m.lastActiveAt, now) : m.deactivatedAt ? `deactivated ${usDate(m.deactivatedAt)}` : 'deactivated'}</Td>
      <Td>
        {active ? <Badge tone="ok">Active</Badge> : <Badge tone="neutral">Deactivated</Badge>}
      </Td>
      <Td>
        {!manage ? (
          <span className="k">–</span>
        ) : active ? (
          <ButtonLink size="sm" to="/w/$workspace/settings/team/$memberId" params={{ workspace: slug, memberId: m.id }} aria-label={`Manage ${m.name}`}>
            Manage
          </ButtonLink>
        ) : (
          <Button size="sm" loading={reactivate.isPending} onClick={() => reactivate.run(m)} aria-label={`Reactivate ${m.name}`}>
            Reactivate
          </Button>
        )}
      </Td>
    </tr>
  )
}

function useReactivate() {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const mut = $api.useMutation('post', '/api/v1/workspaces/{workspace}/members/{memberId}/reactivate')
  return {
    isPending: mut.isPending,
    run: (m: Member) =>
      mut.mutate(
        { params: { path: { workspace: slug, memberId: m.id } } },
        {
          onSuccess: () => {
            toast({ title: `${m.name} reactivated`, detail: 'they can sign in to this workspace again' })
            void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          },
          onError: (e) => toast({ kind: 'error', title: `Couldn't reactivate ${m.name}`, detail: errorMessage(e) }),
        },
      ),
  }
}

function Filters({ list, search }: { list: Schemas['TeamMemberList']; search: TeamSearch }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [q, setQ] = useState(search.q ?? '')
  const set = (patch: Partial<TeamSearch>) =>
    void navigate({ to: '/w/$workspace/settings/team', params: { workspace: slug }, search: clean({ ...search, ...patch }), replace: true })
  useEffect(() => {
    if (q === (search.q ?? '')) return
    const h = window.setTimeout(() => set({ q: q.trim() || undefined }), 250)
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q])
  return (
    <div className="row gap-2">
      <label className="vh" htmlFor="team-q">
        Search members
      </label>
      <input id="team-q" className="in w-[240px] max-md:min-w-full" placeholder="Search name or email" value={q} onChange={(e) => setQ(e.target.value)} />
      <select className="in w-[150px] max-md:min-w-full" aria-label="Filter by role" value={search.role ?? ''} onChange={(e) => set({ role: e.target.value || undefined })}>
        <option value="">Any role</option>
        {list.roles.map((r) => (
          <option key={r.id} value={r.key}>
            {r.name}
          </option>
        ))}
      </select>
      <select
        className="in w-[140px] max-md:min-w-full"
        aria-label="Filter by status"
        value={search.status ?? ''}
        onChange={(e) => set({ status: (e.target.value || undefined) as TeamSearch['status'] })}
      >
        <option value="">Any status</option>
        <option value="active">Active</option>
        <option value="invited">Invited</option>
        <option value="deactivated">Deactivated</option>
      </select>
      <select
        className="in w-[120px] max-md:min-w-full"
        aria-label="Filter by two-factor status"
        value={search.twofa ?? ''}
        onChange={(e) => set({ twofa: (e.target.value || undefined) as TeamSearch['twofa'] })}
      >
        <option value="">Any 2FA</option>
        <option value="on">On</option>
        <option value="off">Off</option>
        <option value="idp">Via IdP</option>
        <option value="required">Required</option>
      </select>
      <span className="k ml-1" role="status">
        {list.counts.members} {list.counts.members === 1 ? 'member' : 'members'} · {list.counts.invited} invited
      </span>
    </div>
  )
}

function clean(s: TeamSearch): TeamSearch {
  const out: TeamSearch = {}
  if (s.q) out.q = s.q
  if (s.role) out.role = s.role
  if (s.status) out.status = s.status
  if (s.twofa) out.twofa = s.twofa
  return out
}

/** Bulk bar: change role, require two-factor now, deactivate. */
function BulkBar({ list, selected, setSelected }: { list: Schemas['TeamMemberList']; selected: string[]; setSelected: (s: string[]) => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const selectable = list.items.filter((m) => !m.you)
  const roles = useMemo(() => bulkRoles(list.roles), [list.roles])
  const [roleKey, setRoleKey] = useState(roles[0]?.key ?? '')
  const [deactivating, setDeactivating] = useState(false)
  const [result, setResult] = useState<{ action: 'change_role' | 'require_2fa'; role?: string; res: Schemas['TeamBulkResult'] }>()
  const bulk = $api.useMutation('post', '/api/v1/workspaces/{workspace}/members/bulk')
  const all = selectable.length > 0 && selectable.every((m) => selected.includes(m.id))
  const none = selected.length === 0
  const reason = none ? 'Select members in the table first.' : undefined
  const run = (action: 'change_role' | 'require_2fa', label: string) =>
    bulk.mutate(
      { params: { path: { workspace: slug } }, body: { memberIds: selected, action, roleId: action === 'change_role' ? list.roles.find((r) => r.key === roleKey)?.id : undefined } },
      {
        onSuccess: (res) => {
          // A partial result says who was skipped and why (Dialogs board, bulk result).
          if (res.skipped.length) setResult({ action, role: roles.find((r) => r.key === roleKey)?.name, res })
          else toast({ title: `${label}: ${res.changed} ${res.changed === 1 ? 'member' : 'members'}` })
          setSelected([])
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
        },
        onError: (e) => toast({ kind: 'error', title: `${label} failed`, detail: errorMessage(e) }),
      },
    )
  const selectedMembers = list.items.filter((m) => selected.includes(m.id))
  return (
    <div className="row gap-2 border-b border-line-2 bg-ground-2 px-4.5 py-2.5">
      <label className="row gap-2 text-[13px]">
        <input
          type="checkbox"
          checked={!none}
          ref={(el) => {
            if (el) el.indeterminate = !none && !all
          }}
          onChange={() => setSelected(all || !none ? [] : selectable.map((m) => m.id))}
          aria-label={none ? 'Select every member' : 'Clear the selection'}
        />
        {selected.length} selected
      </label>
      <span className="k">Bulk:</span>
      <select className="in w-[160px] max-md:min-w-full" aria-label="New role for the selected members" value={roleKey} onChange={(e) => setRoleKey(e.target.value)}>
        {roles.map((r) => (
          <option key={r.id} value={r.key}>
            {r.name}
          </option>
        ))}
      </select>
      <Button size="sm" disabledReason={reason} loading={bulk.isPending && bulk.variables?.body.action === 'change_role'} onClick={() => run('change_role', 'Role changed')}>
        Change role
      </Button>
      <Button size="sm" disabledReason={reason} loading={bulk.isPending && bulk.variables?.body.action === 'require_2fa'} onClick={() => run('require_2fa', 'Two-factor required now')}>
        Require 2FA now
      </Button>
      <Button size="sm" variant="danger" disabledReason={reason} onClick={() => setDeactivating(true)}>
        Deactivate
      </Button>
      {result ? (
        <BulkResultDialog
          open
          onOpenChange={(o) => !o && setResult(undefined)}
          title={result.action === 'change_role' ? `Role changed to ${result.role ?? 'the new role'}` : 'Two-factor required now'}
          done={{
            count: result.res.changed,
            text: result.action === 'change_role' ? `now ${result.role ?? 'have the new role'}` : 'must set up two-factor at their next sign-in',
          }}
          skipped={{
            count: result.res.skipped.length,
            text: `${andList(result.res.skipped)} ${result.action === 'change_role' ? 'already had this role' : 'already use two-factor or sign in through an IdP that enforces it'}`,
          }}
        />
      ) : null}
      {deactivating ? (
        <DeactivateDialog
          members={selectedMembers}
          open
          onOpenChange={setDeactivating}
          onDone={() => {
            setSelected([])
            setDeactivating(false)
          }}
        />
      ) : null}
    </div>
  )
}

/** The bulk role select lists Editor first, as on the board. */
function bulkRoles(roles: Schemas['TeamRoleRef'][]) {
  const rank = (r: Schemas['TeamRoleRef']) => (r.key === 'editor' ? 0 : r.key === 'viewer' ? 1 : !r.builtin ? 2 : r.key === 'admin' ? 3 : 4)
  return roles.filter((r) => r.key !== 'owner').sort((a, b) => rank(a) - rank(b))
}

function InvitationsCard({ list, onInvite }: { list: Schemas['TeamInvitationList']; onInvite: () => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const qc = useQueryClient()
  const invite = list.canInvite
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/invitations'] })
  const resend = $api.useMutation('post', '/api/v1/workspaces/{workspace}/invitations/{invitationId}/resend')
  const link = $api.useMutation('post', '/api/v1/workspaces/{workspace}/invitations/{invitationId}/link')
  const revoke = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/invitations/{invitationId}')
  const jit = list.jit[0]
  return (
    <section className="card" aria-labelledby="inv-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="inv-h">
            Pending invitations · {list.items.length}
          </h2>
          <span className="k">{invite ? `Links work for ${list.defaultDays} days by default` : 'Resend, copy link and revoke are hidden'}</span>
        </div>
        {invite ? (
          <Button size="sm" variant="primary" onClick={onInvite}>
            Invite member
          </Button>
        ) : null}
      </div>
      <AccessTable caption="Pending invitations">
        <thead>
          <tr>
            <Th>Invitee</Th>
            <Th>Role · scope</Th>
            <Th>Sent</Th>
            <Th>Expires</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {list.items.length === 0 ? (
            <EmptyRow colSpan={5}>No pending invitations. Invite people by email or share a link.</EmptyRow>
          ) : (
            list.items.map((i) => (
              <tr key={i.id}>
                <Td>
                  <PersonCell name={i.email} sub={i.invitedBy ? `invited by ${i.invitedBy}` : undefined} muted text="?" />
                </Td>
                <Td>
                  <RoleBadge role={i.role} />
                  {invite && i.scope.label ? (
                    <>
                      {' '}
                      <span className="k">· {i.scope.label}</span>
                    </>
                  ) : null}
                </Td>
                <Td>
                  {monthDay(i.lastSentAt ?? i.createdAt)}
                  {invite ? (i.sentVia ? ` · email via ${i.sentVia}` : ' · link only') : ''}
                </Td>
                <Td>{i.expired ? <Badge tone="bad">Expired</Badge> : inDays(i.expiresAt, now)}</Td>
                <Td>
                  {invite ? (
                    <div className="row flex-nowrap gap-1.5">
                      <Button
                        size="sm"
                        loading={resend.isPending && resend.variables?.params.path.invitationId === i.id}
                        onClick={() =>
                          resend.mutate(
                            { params: { path: { workspace: slug, invitationId: i.id } } },
                            {
                              onSuccess: async (r) => {
                                void refresh()
                                if (r.emailed) toast({ title: `Invitation sent again to ${i.email}`, detail: `via ${r.sentVia}` })
                                else {
                                  const ok = await copyText(r.link)
                                  toast({ kind: 'info', title: "Email can't be sent yet", detail: ok ? 'the link was copied instead' : r.link })
                                }
                              },
                              onError: (e) => toast({ kind: 'error', title: "Couldn't resend", detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        Resend
                      </Button>
                      <Button
                        size="sm"
                        onClick={() =>
                          link.mutate(
                            { params: { path: { workspace: slug, invitationId: i.id } } },
                            {
                              onSuccess: async (r) => {
                                const ok = await copyText(r.link)
                                toast({ title: ok ? 'Invite link copied' : 'Invite link', detail: ok ? `works until ${monthDay(r.expiresAt)}` : r.link })
                              },
                              onError: (e) => toast({ kind: 'error', title: "Couldn't copy the link", detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        Copy link
                      </Button>
                      <Button
                        size="sm"
                        variant="danger"
                        onClick={() =>
                          revoke.mutate(
                            { params: { path: { workspace: slug, invitationId: i.id } } },
                            {
                              onSuccess: () => {
                                void refresh()
                                toast({ kind: 'deleted', title: `Invitation for ${i.email} revoked`, detail: 'the link stops working' })
                              },
                              onError: (e) => toast({ kind: 'error', title: "Couldn't revoke", detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        Revoke
                      </Button>
                    </div>
                  ) : (
                    <span className="k">–</span>
                  )}
                </Td>
              </tr>
            ))
          )}
        </tbody>
      </AccessTable>
      {invite && jit ? (
        <div className="pad pt-3">
          <div className="note i">
            <GoogleMark size={14} />
            <span>
              {jit.domain} users can also join automatically with {jit.provider}: single sign-on creates their account as {jit.roleName} on first sign-in (JIT). Invite them only to give a
              different role.
            </span>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function TeamSkeleton() {
  return (
    <section className="card" aria-busy="true">
      <span className="vh" role="status">
        Loading the team
      </span>
      <div className="sec">
        <div className="flex flex-col gap-1.5">
          <Skeleton w={90} h={14} />
          <Skeleton w={220} h={10} />
        </div>
      </div>
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex items-center gap-6 border-b border-line-2 px-3 py-3.5 last:border-b-0">
          <div className="flex min-w-[180px] items-center gap-2.5">
            <Skeleton w={28} h={28} r={14} />
            <div className="flex flex-col gap-1.5">
              <Skeleton w={110} h={12} />
              <Skeleton w={130} h={10} />
            </div>
          </div>
          <Skeleton w={70} h={20} r={999} />
          <Skeleton w={120} h={12} />
          <Skeleton w={70} h={20} r={999} />
          <Skeleton w={70} h={12} />
          <Skeleton w={60} h={20} r={999} className="ml-auto" />
        </div>
      ))}
    </section>
  )
}

