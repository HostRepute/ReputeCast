import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { GoogleMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { BulkResultDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { Truncate } from '@/components/ui/Truncate'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { T, t as tt, useT } from '@/i18n'
import { ago, andList, inDays, monthDay, orList, usDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { roleName } from '@/lib/roles'
import { AccessTable, copyText, PersonCell, RoleBadge, scopeLabel, SignInText, TwoFABadge } from '../access/parts'
import { DeactivateDialog } from './DeactivateDialog'
import { InviteDialog } from './InviteDialog'
import { m as tm } from './messages'

export interface TeamSearch {
  q?: string
  role?: string
  status?: 'active' | 'invited' | 'deactivated'
  twofa?: 'on' | 'off' | 'idp' | 'required'
}

type Member = Schemas['TeamMember']

/** A5 §5.8 Team: members with role, scope, sign-in, two-factor and status; pending invitations. */
export function TeamPage({ search }: { search: TeamSearch }) {
  const t = useT()
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
      title={t(tm.title)}
      tabs={settingsTabs(slug)}
      actions={
        canInvite ? (
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            {t(tm.inviteMember)}
          </Button>
        ) : null
      }
    >
      {members.data && !members.data.canInvite ? <ReadOnlyNote helpers={members.data.helpers} /> : null}
      {members.isError ? (
        <ErrorPanel title={t(tm.loadFailed)} actions={<Button onClick={() => members.refetch()}>{t(tm.retry)}</Button>}>
          {errorMessage(members.error)} {t(tm.nothingChanged)}
        </ErrorPanel>
      ) : members.isPending ? (
        <TeamSkeleton />
      ) : (
        <MembersCard list={members.data} search={search} hideMembers={search.status === 'invited'} />
      )}
      {invitations.isError ? (
        <ErrorPanel title={t(tm.invitationsFailed)} actions={<Button onClick={() => invitations.refetch()}>{t(tm.retry)}</Button>}>
          {errorMessage(invitations.error)}
        </ErrorPanel>
      ) : invitations.data ? (
        <InvitationsCard list={invitations.data} onInvite={() => setInviteOpen(true)} />
      ) : null}
      {inviteOpen ? <InviteDialog open onOpenChange={setInviteOpen} /> : null}
    </Page>
  )
}

/** "You can view the team. Inviting needs the team:invite permission…", and who can help. */
function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const mono = (c: ReactNode) => <span className="mono">{c}</span>
  return (
    <div className="note n" role="status">
      <Badge tone="neutral">{t(tm.readOnly)}</Badge>
      <span>{helpers.length ? <T msg={tm.readOnlyNoteAsk} values={{ helpers: orList(helpers), mono }} /> : <T msg={tm.readOnlyNote} values={{ mono }} />}</span>
    </div>
  )
}

function MembersCard({ list, search, hideMembers }: { list: Schemas['TeamMemberList']; search: TeamSearch; hideMembers: boolean }) {
  const t = useT()
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
            {t(tm.members)}
          </h2>
          <span className="k">{t(tm.membersCaption, { workspace: me.workspace.name })}</span>
        </div>
        {manage ? (
          <Filters list={list} search={search} />
        ) : (
          <span className="k">{t(tm.counts, { members: list.counts.members, invited: list.counts.invited })}</span>
        )}
      </div>
      {manage ? <BulkBar list={list} selected={selected} setSelected={setSelected} /> : null}
      <AccessTable caption={t(tm.members)}>
        <thead>
          <tr>
            <Th>{t(tm.member)}</Th>
            <Th>{t(tm.roleScope)}</Th>
            <Th>{t(tm.signIn)}</Th>
            <Th>{t(tm.twoFA)}</Th>
            <Th>{t(tm.lastActive)}</Th>
            <Th>{t(tm.status)}</Th>
            <Th>
              <span className="vh">{t(tm.actions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {items.length === 0 ? (
            <EmptyRow colSpan={7}>
              {hideMembers ? t(tm.invitedBelow) : filtered ? t(tm.noMatch) : t(tm.noMembers)}
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
  const t = useT()
  const active = m.status === 'active'
  const reactivate = useReactivate()
  return (
    <tr style={active ? undefined : { color: '#5B6472' }}>
      <Td>
        <div className="flex min-w-0 items-center gap-2.5">
          {manage ? (
            <input type="checkbox" checked={selected} onChange={(e) => onSelect(e.target.checked)} aria-label={t(tm.select, { name: m.name })} />
          ) : null}
          <PersonCell name={m.name} sub={<Truncate text={m.email} />} color={active ? m.avatarColor : 'muted'} />
        </div>
      </Td>
      <Td>
        {/* As on the board, the space before each note is outside it (body size, not 12px). */}
        <RoleBadge role={m.role} />
        {!m.role.builtin ? (
          <>
            {' '}
            <span className="k">{t(tm.custom)}</span>
          </>
        ) : null}
        {m.scope.all ? null : (
          <>
            {' '}
            <span className="k">· {scopeLabel(m.scope)}</span>
          </>
        )}
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
      <Td>{active ? ago(m.lastActiveAt, now) : m.deactivatedAt ? t(tm.deactivatedOn, { date: usDate(m.deactivatedAt) }) : t(tm.deactivated)}</Td>
      <Td>
        {active ? <Badge tone="ok">{t(tm.active)}</Badge> : <Badge tone="neutral">{t(tm.deactivatedBadge)}</Badge>}
      </Td>
      <Td>
        {!manage ? (
          <span className="k">–</span>
        ) : active ? (
          <ButtonLink size="sm" to="/w/$workspace/settings/team/$memberId" params={{ workspace: slug, memberId: m.id }} aria-label={t(tm.manageName, { name: m.name })}>
            {t(tm.manage)}
          </ButtonLink>
        ) : (
          <Button size="sm" loading={reactivate.isPending} onClick={() => reactivate.run(m)} aria-label={t(tm.reactivateName, { name: m.name })}>
            {t(tm.reactivate)}
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
            toast({ title: tt(tm.reactivated, { name: m.name }), detail: tt(tm.reactivatedDetail) })
            void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          },
          onError: (e) => toast({ kind: 'error', title: tt(tm.reactivateFailed, { name: m.name }), detail: errorMessage(e) }),
        },
      ),
  }
}

function Filters({ list, search }: { list: Schemas['TeamMemberList']; search: TeamSearch }) {
  const t = useT()
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
    <div className="filters row gap-2">
      <label className="vh" htmlFor="team-q">
        {t(tm.searchLabel)}
      </label>
      <input id="team-q" className="in w-[240px] max-md:min-w-full" placeholder={t(tm.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
      {/* Filter selects size to their longest option, the old width being the least (text expansion rule 1). */}
      <select className="in w-auto min-w-[150px] max-md:min-w-full" aria-label={t(tm.filterRole)} value={search.role ?? ''} onChange={(e) => set({ role: e.target.value || undefined })}>
        <option value="">{t(tm.anyRole)}</option>
        {list.roles.map((r) => (
          <option key={r.id} value={r.key}>
            {roleName(r)}
          </option>
        ))}
      </select>
      <select
        className="in w-auto min-w-[140px] max-md:min-w-full"
        aria-label={t(tm.filterStatus)}
        value={search.status ?? ''}
        onChange={(e) => set({ status: (e.target.value || undefined) as TeamSearch['status'] })}
      >
        <option value="">{t(tm.anyStatus)}</option>
        <option value="active">{t(tm.active)}</option>
        <option value="invited">{t(tm.invited)}</option>
        <option value="deactivated">{t(tm.deactivatedBadge)}</option>
      </select>
      <select
        className="in w-auto min-w-[120px] max-md:min-w-full"
        aria-label={t(tm.filter2FA)}
        value={search.twofa ?? ''}
        onChange={(e) => set({ twofa: (e.target.value || undefined) as TeamSearch['twofa'] })}
      >
        <option value="">{t(tm.any2FA)}</option>
        <option value="on">{t(tm.twoFAOn)}</option>
        <option value="off">{t(tm.twoFAOff)}</option>
        <option value="idp">{t(tm.twoFAIdp)}</option>
        <option value="required">{t(tm.twoFARequired)}</option>
      </select>
      <span className="k ml-1" role="status">
        {t(tm.counts, { members: list.counts.members, invited: list.counts.invited })}
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
  const t = useT()
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
  const reason = none ? t(tm.selectFirst) : undefined
  const run = (action: 'change_role' | 'require_2fa') =>
    bulk.mutate(
      { params: { path: { workspace: slug } }, body: { memberIds: selected, action, roleId: action === 'change_role' ? list.roles.find((r) => r.key === roleKey)?.id : undefined } },
      {
        onSuccess: (res) => {
          // A partial result says who was skipped and why (Dialogs board, bulk result).
          const role = roles.find((r) => r.key === roleKey)
          if (res.skipped.length) setResult({ action, role: role ? roleName(role) : undefined, res })
          else toast({ title: tt(action === 'change_role' ? tm.roleChanged : tm.required2FA, { count: res.changed }) })
          setSelected([])
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
        },
        onError: (e) => toast({ kind: 'error', title: tt(action === 'change_role' ? tm.roleChangeFailed : tm.require2FAFailed), detail: errorMessage(e) }),
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
          aria-label={none ? t(tm.selectAll) : t(tm.clearSelection)}
        />
        {t(tm.selected, { count: selected.length })}
      </label>
      <span className="k">{t(tm.bulk)}</span>
      <select className="in w-auto min-w-[160px] max-md:min-w-full" aria-label={t(tm.bulkRole)} value={roleKey} onChange={(e) => setRoleKey(e.target.value)}>
        {roles.map((r) => (
          <option key={r.id} value={r.key}>
            {roleName(r)}
          </option>
        ))}
      </select>
      <Button size="sm" disabledReason={reason} loading={bulk.isPending && bulk.variables?.body.action === 'change_role'} onClick={() => run('change_role')}>
        {t(tm.changeRole)}
      </Button>
      <Button size="sm" disabledReason={reason} loading={bulk.isPending && bulk.variables?.body.action === 'require_2fa'} onClick={() => run('require_2fa')}>
        {t(tm.require2FA)}
      </Button>
      <Button size="sm" variant="danger" disabledReason={reason} onClick={() => setDeactivating(true)}>
        {t(tm.deactivate)}
      </Button>
      {result ? (
        <BulkResultDialog
          open
          onOpenChange={(o) => !o && setResult(undefined)}
          title={
            result.action === 'change_role' ? (result.role ? t(tm.roleChangedTo, { role: result.role }) : t(tm.roleChangedToNew)) : t(tm.twoFARequiredNow)
          }
          done={{
            count: result.res.changed,
            text: result.action === 'change_role' ? (result.role ? t(tm.nowRole, { role: result.role }) : t(tm.nowNewRole)) : t(tm.mustSetUp),
          }}
          skipped={{
            count: result.res.skipped.length,
            text: t(result.action === 'change_role' ? tm.alreadyRole : tm.already2FA, { names: andList(result.res.skipped) }),
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
  const t = useT()
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
            {t(tm.pending, { count: list.items.length })}
          </h2>
          <span className="k">{invite ? t(tm.linksWork, { days: list.defaultDays }) : t(tm.actionsHidden)}</span>
        </div>
        {invite ? (
          <Button size="sm" variant="primary" onClick={onInvite}>
            {t(tm.inviteMember)}
          </Button>
        ) : null}
      </div>
      <AccessTable caption={t(tm.pending, { count: list.items.length })}>
        <thead>
          <tr>
            <Th>{t(tm.invitee)}</Th>
            <Th>{t(tm.roleScope)}</Th>
            <Th>{t(tm.sent)}</Th>
            <Th>{t(tm.expires)}</Th>
            <Th>
              <span className="vh">{t(tm.actions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {list.items.length === 0 ? (
            <EmptyRow colSpan={5}>{t(tm.noInvitations)}</EmptyRow>
          ) : (
            list.items.map((i) => (
              <tr key={i.id}>
                <Td>
                  <PersonCell name={<Truncate text={i.email} />} sub={i.invitedBy ? t(tm.invitedBy, { name: i.invitedBy }) : undefined} muted text="?" />
                </Td>
                <Td>
                  <RoleBadge role={i.role} />
                  {invite && !i.scope.all ? (
                    <>
                      {' '}
                      <span className="k">· {scopeLabel(i.scope)}</span>
                    </>
                  ) : null}
                </Td>
                <Td>
                  {invite
                    ? i.sentVia
                      ? t(tm.sentVia, { date: monthDay(i.lastSentAt ?? i.createdAt), provider: i.sentVia })
                      : t(tm.linkOnly, { date: monthDay(i.lastSentAt ?? i.createdAt) })
                    : monthDay(i.lastSentAt ?? i.createdAt)}
                </Td>
                <Td>{i.expired ? <Badge tone="bad">{t(tm.expired)}</Badge> : inDays(i.expiresAt, now)}</Td>
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
                                if (r.emailed) toast({ title: tt(tm.sentAgain, { email: i.email }), detail: tt(tm.sentAgainVia, { provider: r.sentVia ?? '' }) })
                                else {
                                  const ok = await copyText(r.link)
                                  toast({ kind: 'info', title: tt(tm.cantEmail), detail: ok ? tt(tm.linkCopiedInstead) : r.link })
                                }
                              },
                              onError: (e) => toast({ kind: 'error', title: tt(tm.resendFailed), detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        {t(tm.resend)}
                      </Button>
                      <Button
                        size="sm"
                        onClick={() =>
                          link.mutate(
                            { params: { path: { workspace: slug, invitationId: i.id } } },
                            {
                              onSuccess: async (r) => {
                                const ok = await copyText(r.link)
                                toast({ title: ok ? tt(tm.linkCopied) : tt(tm.inviteLink), detail: ok ? tt(tm.worksUntil, { date: monthDay(r.expiresAt) }) : r.link })
                              },
                              onError: (e) => toast({ kind: 'error', title: tt(tm.copyFailed), detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        {t(tm.copyLink)}
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
                                toast({ kind: 'deleted', title: tt(tm.revoked, { email: i.email }), detail: tt(tm.revokedDetail) })
                              },
                              onError: (e) => toast({ kind: 'error', title: tt(tm.revokeFailed), detail: errorMessage(e) }),
                            },
                          )
                        }
                      >
                        {t(tm.revoke)}
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
            <span>{t(tm.jit, { domain: jit.domain, provider: jit.provider, role: jit.roleName })}</span>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function TeamSkeleton() {
  const t = useT()
  return (
    <section className="card" aria-busy="true">
      <span className="vh" role="status">
        {t(tm.loading)}
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

