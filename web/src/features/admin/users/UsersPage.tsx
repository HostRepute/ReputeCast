import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, PersonCell, SignInText, TwoFABadge } from '@/features/access/parts'
import { ago, clock, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { useInvalidateUsers } from '../queries'
import { DeactivateUserDialog, MakeAdminDialog, PasswordResetDialog, RemoveAdminDialog, ResetTwoFactorDialog } from './UserDialogs'

type User = Schemas['InstanceUser']

export interface UsersSearch {
  q?: string
  workspace?: string
  status?: 'active' | 'deactivated'
  twofa?: 'on' | 'off' | 'idp'
}

/** Instance admin › Users (InstanceUsers, InstanceUserDialogs). */
export function UsersPage({ search }: { search: UsersSearch }) {
  return (
    <AdminPage documentTitle="Users">
      <UsersCard search={search} />
      <div className="grid gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(380px,100%),1fr))] max-md:grid-cols-1">
        <PolicySnapshot />
        <RecentSignIns />
      </div>
    </AdminPage>
  )
}

function UsersCard({ search }: { search: UsersSearch }) {
  const q = $api.useQuery(
    'get',
    '/api/v1/admin/users',
    { params: { query: { q: search.q || undefined, workspace: search.workspace || undefined, status: search.status, twofa: search.twofa } } },
    { placeholderData: keepPreviousData },
  )
  const total = q.data?.total
  return (
    <section className="card" aria-labelledby="users-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="users-h">
            Users{total !== undefined ? ` · ${total} ${total === 1 ? 'account' : 'accounts'}` : ''}
          </h2>
          <span className="k">Every account on this instance. Deactivating here signs the user out of every workspace.</span>
        </div>
        <Filters search={search} workspaces={q.data?.workspaces ?? []} />
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the users" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Nothing was changed.
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <UsersSkeleton />
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>No accounts match these filters. Clear the search or pick “Any” in the filters.</span>
          </div>
        </div>
      ) : (
        <UsersTable data={q.data} />
      )}
    </section>
  )
}

function Filters({ search, workspaces }: { search: UsersSearch; workspaces: Schemas['InstanceWorkspaceRef'][] }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [text, setText] = useState(search.q ?? '')
  const set = (patch: Partial<UsersSearch>) =>
    void navigate({ to: '/w/$workspace/admin/users', params: { workspace: slug }, search: clean({ ...search, ...patch }), replace: true })
  useEffect(() => {
    if (text === (search.q ?? '')) return
    const h = window.setTimeout(() => set({ q: text.trim() || undefined }), 250)
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text])
  return (
    <div className="row gap-2 max-md:basis-full">
      <label className="vh" htmlFor="uq">
        Search users
      </label>
      <input id="uq" className="in w-[220px] max-md:w-full" type="search" placeholder="Search name or email" value={text} onChange={(e) => setText(e.target.value)} />
      <select className="in w-[170px] max-md:w-full" aria-label="Filter by workspace" value={search.workspace ?? ''} onChange={(e) => set({ workspace: e.target.value || undefined })}>
        <option value="">Any workspace</option>
        {workspaces.map((w) => (
          <option key={w.slug} value={w.slug}>
            {w.name}
          </option>
        ))}
      </select>
      <select
        className="in w-[130px] max-md:w-full"
        aria-label="Filter by status"
        value={search.status ?? ''}
        onChange={(e) => set({ status: (e.target.value || undefined) as UsersSearch['status'] })}
      >
        <option value="">Any status</option>
        <option value="active">Active</option>
        <option value="deactivated">Deactivated</option>
      </select>
      <select
        className="in w-[110px] max-md:w-full"
        aria-label="Filter by two-factor status"
        value={search.twofa ?? ''}
        onChange={(e) => set({ twofa: (e.target.value || undefined) as UsersSearch['twofa'] })}
      >
        <option value="">Any 2FA</option>
        <option value="on">On</option>
        <option value="off">Off</option>
        <option value="idp">Via IdP</option>
      </select>
    </div>
  )
}

function clean(s: UsersSearch): UsersSearch {
  return Object.fromEntries(Object.entries(s).filter(([, v]) => v !== undefined && v !== '')) as UsersSearch
}

type DialogKind = 'twofa' | 'reset' | 'admin' | 'unadmin' | 'deactivate'

function UsersTable({ data }: { data: Schemas['InstanceUserList'] }) {
  const [dialog, setDialog] = useState<{ kind: DialogKind; user: User }>()
  const [open, setOpen] = useState<string>()
  return (
    <>
      <AccessTable caption="Users">
        <thead>
          <tr>
            <Th>User</Th>
            <Th>Workspaces · role</Th>
            <Th>Sign-in methods</Th>
            <Th>2FA</Th>
            <Th>Last sign-in</Th>
            <Th>Status</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((u) => (
            <UserRow key={u.id} u={u} menuOpen={open === u.id} onMenu={(o) => setOpen(o ? u.id : undefined)} onDialog={(kind) => setDialog({ kind, user: u })} />
          ))}
        </tbody>
      </AccessTable>
      {dialog?.kind === 'twofa' ? <ResetTwoFactorDialog user={dialog.user} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'reset' ? <PasswordResetDialog user={dialog.user} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'admin' ? <MakeAdminDialog user={dialog.user} admins={data.adminNames} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'unadmin' ? <RemoveAdminDialog user={dialog.user} admins={data.adminNames} onClose={() => setDialog(undefined)} /> : null}
      {dialog?.kind === 'deactivate' ? <DeactivateUserDialog user={dialog.user} onClose={() => setDialog(undefined)} /> : null}
    </>
  )
}

function UserRow({ u, menuOpen, onMenu, onDialog }: { u: User; menuOpen: boolean; onMenu: (o: boolean) => void; onDialog: (k: DialogKind) => void }) {
  const off = u.status === 'deactivated'
  const [first, ...rest] = u.workspaces
  const person = <PersonCell name={u.name} sub={u.email} color={u.avatarColor} muted={off} />
  return (
    <tr className={off ? 'text-ink-3' : undefined} style={menuOpen ? { background: '#F8FAFC' } : undefined}>
      <Td>
        {u.instanceAdmin ? (
          <div className="col gap-1">
            {person}
            <Badge tone="purple">Instance admin</Badge>
          </div>
        ) : (
          person
        )}
      </Td>
      <Td>
        {u.workspaces.length === 0 ? (
          <span className="k">No workspace</span>
        ) : rest.length === 0 ? (
          `${first!.name} · ${first!.role}`
        ) : (
          <span className="col gap-0.5">
            <span>
              {first!.name} · {first!.role}
            </span>
            {rest.map((w) => (
              <span key={w.slug} className="k">
                {w.name} · {w.role}
              </span>
            ))}
          </span>
        )}
      </Td>
      <Td>{off ? <span className="k">–</span> : <SignInText methods={u.methods} />}</Td>
      <Td>{off ? <span className="k">–</span> : <TwoFABadge twoFA={u.twoFactor} />}</Td>
      <Td className="whitespace-nowrap!">{off ? (u.deactivatedAt ? usDate(u.deactivatedAt) : '–') : u.lastSignInAt ? ago(u.lastSignInAt) : <span className="k">never</span>}</Td>
      <Td>
        {off ? (
          <Badge tone="neutral">Deactivated</Badge>
        ) : u.lockedUntil ? (
          <span className="col gap-0.5">
            <Badge tone="warn">Locked</Badge>
            <span className="k">until {clock(u.lockedUntil)}</span>
          </span>
        ) : (
          <Badge tone="ok">Active</Badge>
        )}
      </Td>
      <Td>
        <UserMenu u={u} open={menuOpen} onOpenChange={onMenu} onDialog={onDialog} />
      </Td>
    </tr>
  )
}

function UserMenu({ u, open, onOpenChange, onDialog }: { u: User; open: boolean; onOpenChange: (o: boolean) => void; onDialog: (k: DialogKind) => void }) {
  const invalidate = useInvalidateUsers()
  const signOut = $api.useMutation('post', '/api/v1/admin/users/{userId}/sign-out')
  const reactivate = $api.useMutation('post', '/api/v1/admin/users/{userId}/reactivate')
  const unlock = $api.useMutation('post', '/api/v1/admin/users/{userId}/unlock')
  const path = { params: { path: { userId: u.id } } }
  const run = async (what: () => Promise<unknown>, done: string, detail?: string) => {
    try {
      await what()
      await invalidate()
      toast({ kind: 'success', title: done, detail })
    } catch (e) {
      toast({ kind: 'error', title: `Couldn't update ${u.name}`, detail: errorMessage(e) })
    }
  }
  const off = u.status === 'deactivated'
  return (
    <Menu
      label={`Actions for ${u.name}`}
      className="w-[300px] max-md:w-auto"
      open={open}
      onOpenChange={onOpenChange}
      trigger={
        <button type="button" className="ibtn" aria-label={`Actions for ${u.name}`} style={open ? { background: '#EEF0F3' } : undefined}>
          ⋯
        </button>
      }
    >
      {off ? (
        <MenuItem onSelect={() => void run(() => reactivate.mutateAsync(path), `${u.name} can sign in again`, 'their workspace memberships are back')}>
          <Icon name="undo" />
          Reactivate account
        </MenuItem>
      ) : (
        <>
          <MenuItem
            disabled={u.sessions === 0}
            onSelect={() =>
              void run(
                async () => {
                  const r = await signOut.mutateAsync({ ...path, body: {} })
                  return r
                },
                `${u.name} was signed out`,
                `${u.sessions} ${u.sessions === 1 ? 'session' : 'sessions'} ended`,
              )
            }
          >
            <Icon name="signOut" />
            Force sign-out <span className="k ml-auto">{u.sessions === 0 ? 'no sessions' : `${u.sessions} ${u.sessions === 1 ? 'session' : 'sessions'}`}</span>
          </MenuItem>
          {u.twoFactor.state === 'on' && u.twoFactor.method === 'TOTP' ? (
            <MenuItem onSelect={() => onDialog('twofa')}>
              <Icon name="lock" />
              Reset two-factor <span className="k ml-auto">reason required</span>
            </MenuItem>
          ) : (
            <MenuItem disabled>
              <Icon name="lock" />
              Reset two-factor <span className="k ml-auto">{u.twoFactor.state === 'idp' ? 'via IdP' : 'no authenticator'}</span>
            </MenuItem>
          )}
          {u.hasPassword ? (
            <>
              <MenuItem onSelect={() => onDialog('reset')}>
                <Icon name="mail" />
                Send password reset
              </MenuItem>
              <MenuItem onSelect={() => onDialog('reset')}>
                <Icon name="terminal" />
                Show CLI reset command
              </MenuItem>
            </>
          ) : (
            <MenuItem disabled>
              <Icon name="mail" />
              Send password reset <span className="k ml-auto">{u.ssoProvider ? `managed by ${u.ssoProvider}` : 'no password'}</span>
            </MenuItem>
          )}
          {u.lockedUntil ? (
            <MenuItem onSelect={() => void run(() => unlock.mutateAsync(path), `${u.name} can sign in again`, 'the lockout was cleared')}>
              <Icon name="key" />
              Unlock sign-in <span className="k ml-auto">until {clock(u.lockedUntil)}</span>
            </MenuItem>
          ) : null}
          <MenuSeparator />
          {u.instanceAdmin ? (
            <MenuItem onSelect={() => onDialog('unadmin')}>
              <Icon name="star" />
              Remove instance admin
            </MenuItem>
          ) : (
            <MenuItem onSelect={() => onDialog('admin')}>
              <Icon name="star" />
              Make instance admin
            </MenuItem>
          )}
          <MenuSeparator />
          <MenuItem disabled={u.self} onSelect={() => onDialog('deactivate')} className="text-bad-fg">
            <Icon name="ban" />
            Deactivate account {u.self ? <span className="k ml-auto">that's you</span> : null}
          </MenuItem>
        </>
      )}
    </Menu>
  )
}

function UsersSkeleton() {
  return (
    <div aria-busy="true">
      <span className="vh" role="status">
        Loading users
      </span>
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex items-center gap-6 border-b border-line-2 px-3 py-4 last:border-b-0">
          <div className="flex min-w-[220px] items-center gap-2.5">
            <Skeleton w={28} h={28} r={999} />
            <div className="flex flex-col gap-1.5">
              <Skeleton w={110} h={12} />
              <Skeleton w={130} h={10} />
            </div>
          </div>
          <Skeleton w={170} h={12} />
          <Skeleton w={110} h={12} />
          <Skeleton w={70} h={20} r={999} />
          <Skeleton w={60} h={12} />
          <Skeleton w={56} h={20} r={999} className="ml-auto" />
        </div>
      ))}
    </div>
  )
}

/** Sign-in policy snapshot: SSO enforcement, two-factor, lockout. */
function PolicySnapshot() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/policy')
  const p = q.data
  const enforced = p?.sso.filter((x) => x.status === 'enabled' && x.enforced.length > 0) ?? []
  const twofa = p
    ? p.twoFactor === 'off'
      ? 'Off'
      : `Required for ${p.twoFactor === 'everyone' ? 'everyone' : 'sensitive permissions'} · ${p.graceDays === 0 ? 'no grace period' : `${p.graceDays}-day grace`} · ${p.inGrace.length} ${p.inGrace.length === 1 ? 'member' : 'members'} in grace`
    : ''
  return (
    <section className="card" aria-labelledby="snap-h">
      <CardSection
        title={<span id="snap-h">Sign-in policy snapshot</span>}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/admin/authentication" params={{ workspace: slug }}>
            Authentication
          </ButtonLink>
        }
      />
      <div className="pad">
        {q.isError ? (
          <span className="k">Couldn't load the policy. {errorMessage(q.error)}</span>
        ) : !p ? (
          <div className="col gap-2.5" aria-busy="true">
            <Skeleton w="80%" />
            <Skeleton w="70%" />
            <Skeleton w="60%" />
          </div>
        ) : (
          <dl className="kv">
            <dt>SSO enforced</dt>
            <dd>
              {enforced.length === 0
                ? 'No domain · password sign-in stays on for every domain'
                : enforced.map((x) => `${x.enforced.join(', ')} → ${x.name}`).join('; ') + ' (except break-glass Owners)'}
            </dd>
            <dt>2FA</dt>
            <dd>{twofa}</dd>
            <dt>Lockout</dt>
            <dd>
              {p.lockoutAttempts} attempts → {lockWords(p.lockoutMinutes)} · {p.lockedNow} {p.lockedNow === 1 ? 'account' : 'accounts'} locked now
            </dd>
          </dl>
        )}
      </div>
    </section>
  )
}

export function lockWords(minutes: number) {
  if (minutes % 60 === 0) {
    const h = minutes / 60
    return `${h} ${h === 1 ? 'hour' : 'hours'}`
  }
  return `${minutes} minutes`
}

/** The latest sign-in attempts, from the instance audit log. */
function RecentSignIns() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/sign-ins', { params: { query: { limit: 4 } } })
  return (
    <section className="card" aria-labelledby="signins-h">
      <CardSection
        title={<span id="signins-h">Recent sign-in events</span>}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/admin/audit" params={{ workspace: slug }}>
            Instance audit log
          </ButtonLink>
        }
      />
      {q.isError ? (
        <div className="pad">
          <span className="k">Couldn't load the sign-in events. {errorMessage(q.error)}</span>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3" aria-busy="true">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="90%" />
          ))}
        </div>
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <span className="k">No sign-ins yet. Every attempt, allowed or rejected, appears here and in the instance audit log.</span>
        </div>
      ) : (
        <AccessTable caption="Recent sign-in events">
          <thead>
            <tr>
              <Th>When</Th>
              <Th>Event</Th>
              <Th>Account</Th>
              <Th>Result</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.map((e) => (
              <tr key={e.id}>
                <Td className="whitespace-nowrap!">{eventWhen(e.at)}</Td>
                <Td className="whitespace-nowrap!">{e.event}</Td>
                <Td>{e.account}</Td>
                <Td>
                  {e.result === 'allowed' ? (
                    <Badge tone="ok">Allowed</Badge>
                  ) : e.result === 'recovered' ? (
                    <Badge tone="warn">Then allowed</Badge>
                  ) : (
                    <span className="col gap-0.5">
                      <Badge tone="bad">Rejected</Badge>
                      {e.detail ? <span className="k">{e.detail}</span> : null}
                    </span>
                  )}
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

/** "09:41" today, "yesterday", then dates. */
function eventWhen(at: string) {
  const d = new Date(at)
  const now = new Date()
  const days = Math.floor((startOfDay(now) - startOfDay(d)) / 86_400_000)
  if (days <= 0) return clock(at)
  if (days === 1) return 'yesterday'
  return usDate(at)
}

function startOfDay(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}
