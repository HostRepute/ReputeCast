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
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, clock, usDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { useInvalidateUsers } from '../queries'
import { m } from './messages'
import { DeactivateUserDialog, MakeAdminDialog, PasswordResetDialog, RemoveAdminDialog, ResetTwoFactorDialog } from './UserDialogs'
import { AgencyStaffTd, AgencyStaffTh, useAgencyColumn } from '@/features/agency/AdminParts'

type User = Schemas['InstanceUser']

export interface UsersSearch {
  q?: string
  workspace?: string
  status?: 'active' | 'deactivated'
  twofa?: 'on' | 'off' | 'idp'
}

/** Instance admin › Users (InstanceUsers, InstanceUserDialogs). */
export function UsersPage({ search }: { search: UsersSearch }) {
  const t = useT()
  return (
    <AdminPage documentTitle={t(m.documentTitle)}>
      <UsersCard search={search} />
      <div className="grid gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(380px,100%),1fr))] max-md:grid-cols-1">
        <PolicySnapshot />
        <RecentSignIns />
      </div>
    </AdminPage>
  )
}

function UsersCard({ search }: { search: UsersSearch }) {
  const t = useT()
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
            {total !== undefined ? t(m.titleCount, { count: total }) : t(m.title)}
          </h2>
          <span className="k">{t(m.caption)}</span>
        </div>
        <Filters search={search} workspaces={q.data?.workspaces ?? []} />
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(m.nothingChanged)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <UsersSkeleton />
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>{t(m.noMatch)}</span>
          </div>
        </div>
      ) : (
        <UsersTable data={q.data} />
      )}
    </section>
  )
}

function Filters({ search, workspaces }: { search: UsersSearch; workspaces: Schemas['InstanceWorkspaceRef'][] }) {
  const t = useT()
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
    <div className="filters-grow row gap-2 max-md:basis-full">
      <label className="vh" htmlFor="uq">
        {t(m.searchLabel)}
      </label>
      <input id="uq" className="in w-[220px] max-md:w-full" type="search" placeholder={t(m.searchPlaceholder)} value={text} onChange={(e) => setText(e.target.value)} />
      <select className="in w-[170px] min-w-[170px] max-md:w-full" aria-label={t(m.filterWorkspace)} value={search.workspace ?? ''} onChange={(e) => set({ workspace: e.target.value || undefined })}>
        <option value="">{t(m.anyWorkspace)}</option>
        {workspaces.map((w) => (
          <option key={w.slug} value={w.slug}>
            {w.name}
          </option>
        ))}
      </select>
      <select
        className="in w-[130px] min-w-[130px] max-md:w-full"
        aria-label={t(m.filterStatus)}
        value={search.status ?? ''}
        onChange={(e) => set({ status: (e.target.value || undefined) as UsersSearch['status'] })}
      >
        <option value="">{t(m.anyStatus)}</option>
        <option value="active">{t(m.active)}</option>
        <option value="deactivated">{t(m.deactivated)}</option>
      </select>
      <select
        className="in w-[110px] min-w-[110px] max-md:w-full"
        aria-label={t(m.filter2FA)}
        value={search.twofa ?? ''}
        onChange={(e) => set({ twofa: (e.target.value || undefined) as UsersSearch['twofa'] })}
      >
        <option value="">{t(m.any2FA)}</option>
        <option value="on">{t(m.twoFAOn)}</option>
        <option value="off">{t(m.twoFAOff)}</option>
        <option value="idp">{t(m.twoFAIdp)}</option>
      </select>
    </div>
  )
}

function clean(s: UsersSearch): UsersSearch {
  return Object.fromEntries(Object.entries(s).filter(([, v]) => v !== undefined && v !== '')) as UsersSearch
}

type DialogKind = 'twofa' | 'reset' | 'admin' | 'unadmin' | 'deactivate'

function UsersTable({ data }: { data: Schemas['InstanceUserList'] }) {
  const t = useT()
  const [dialog, setDialog] = useState<{ kind: DialogKind; user: User }>()
  const [open, setOpen] = useState<string>()
  const agencyColumn = useAgencyColumn()
  return (
    <>
      <AccessTable caption={t(m.tableCaption)}>
        <thead>
          <tr>
            <Th>{t(m.colUser)}</Th>
            <Th>{t(m.colWorkspaces)}</Th>
            {/* Each person's level on the agency's staff, while agency mode is on (agency accounts). */}
            {agencyColumn ? <AgencyStaffTh /> : null}
            <Th>{t(m.colSignIn)}</Th>
            <Th>{t(m.col2FA)}</Th>
            <Th>{t(m.colLastSignIn)}</Th>
            <Th>{t(m.colStatus)}</Th>
            <Th>
              <span className="vh">{t(m.colActions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((u) => (
            <UserRow
              key={u.id}
              u={u}
              agencyColumn={agencyColumn}
              menuOpen={open === u.id}
              onMenu={(o) => setOpen(o ? u.id : undefined)}
              onDialog={(kind) => setDialog({ kind, user: u })}
            />
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

function UserRow({
  u,
  agencyColumn,
  menuOpen,
  onMenu,
  onDialog,
}: {
  u: User
  agencyColumn: boolean
  menuOpen: boolean
  onMenu: (o: boolean) => void
  onDialog: (k: DialogKind) => void
}) {
  const t = useT()
  const off = u.status === 'deactivated'
  const [first, ...rest] = u.workspaces
  const person = <PersonCell name={u.name} sub={u.email} color={u.avatarColor} muted={off} />
  return (
    <tr className={off ? 'text-ink-3' : undefined} style={menuOpen ? { background: 'var(--ground-2)' } : undefined}>
      <Td>
        {u.instanceAdmin ? (
          <div className="col gap-1">
            {person}
            <Badge tone="purple">{t(m.instanceAdmin)}</Badge>
          </div>
        ) : (
          person
        )}
      </Td>
      <Td>
        {u.workspaces.length === 0 ? (
          <span className="k">{t(m.noWorkspace)}</span>
        ) : rest.length === 0 ? (
          `${first!.name} · ${roleName({ name: first!.role })}`
        ) : (
          <span className="col gap-0.5">
            <span>
              {first!.name} · {roleName({ name: first!.role })}
            </span>
            {rest.map((w) => (
              <span key={w.slug} className="k">
                {w.name} · {roleName({ name: w.role })}
              </span>
            ))}
          </span>
        )}
      </Td>
      {agencyColumn ? <AgencyStaffTd level={u.agencyLevel} /> : null}
      <Td>{off ? <span className="k">–</span> : <SignInText methods={u.methods} />}</Td>
      <Td>{off ? <span className="k">–</span> : <TwoFABadge twoFA={u.twoFactor} />}</Td>
      <Td className="whitespace-nowrap!">{off ? (u.deactivatedAt ? usDate(u.deactivatedAt) : '–') : u.lastSignInAt ? ago(u.lastSignInAt) : <span className="k">{t(m.never)}</span>}</Td>
      <Td>
        {off ? (
          <Badge tone="neutral">{t(m.deactivated)}</Badge>
        ) : u.lockedUntil ? (
          <span className="col gap-0.5">
            <Badge tone="warn">{t(m.locked)}</Badge>
            <span className="k">{t(m.until, { time: clock(u.lockedUntil) })}</span>
          </span>
        ) : (
          <Badge tone="ok">{t(m.active)}</Badge>
        )}
      </Td>
      <Td>
        <UserMenu u={u} open={menuOpen} onOpenChange={onMenu} onDialog={onDialog} />
      </Td>
    </tr>
  )
}

function UserMenu({ u, open, onOpenChange, onDialog }: { u: User; open: boolean; onOpenChange: (o: boolean) => void; onDialog: (k: DialogKind) => void }) {
  const t = useT()
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
      toast({ kind: 'error', title: tt(m.updateFailed, { name: u.name }), detail: errorMessage(e) })
    }
  }
  const off = u.status === 'deactivated'
  return (
    <Menu
      label={t(m.actionsFor, { name: u.name })}
      className="us-menu w-[300px] max-md:w-auto"
      open={open}
      onOpenChange={onOpenChange}
      trigger={
        <button type="button" className="ibtn" aria-label={t(m.actionsFor, { name: u.name })} style={open ? { background: 'var(--line-2)' } : undefined}>
          ⋯
        </button>
      }
    >
      {off ? (
        <MenuItem onSelect={() => void run(() => reactivate.mutateAsync(path), tt(m.canSignInAgain, { name: u.name }), tt(m.membershipsBack))}>
          <Icon name="undo" />
          {t(m.reactivateAccount)}
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
                tt(m.signedOut, { name: u.name }),
                tt(m.sessionsEnded, { count: u.sessions }),
              )
            }
          >
            <Icon name="signOut" />
            {t(m.forceSignOut)} <span className="k ml-auto">{t(m.sessions, { count: u.sessions })}</span>
          </MenuItem>
          {u.twoFactor.state === 'on' && u.twoFactor.method === 'TOTP' ? (
            <MenuItem onSelect={() => onDialog('twofa')}>
              <Icon name="lock" />
              {t(m.resetTwoFactor)} <span className="k ml-auto">{t(m.reasonRequired)}</span>
            </MenuItem>
          ) : (
            <MenuItem disabled>
              <Icon name="lock" />
              {t(m.resetTwoFactor)} <span className="k ml-auto">{u.twoFactor.state === 'idp' ? t(m.viaIdp) : t(m.noAuthenticator)}</span>
            </MenuItem>
          )}
          {u.hasPassword ? (
            <>
              <MenuItem onSelect={() => onDialog('reset')}>
                <Icon name="mail" />
                {t(m.sendPasswordReset)}
              </MenuItem>
              <MenuItem onSelect={() => onDialog('reset')}>
                <Icon name="terminal" />
                {t(m.showCliCommand)}
              </MenuItem>
            </>
          ) : (
            <MenuItem disabled>
              <Icon name="mail" />
              {t(m.sendPasswordReset)} <span className="k ml-auto">{u.ssoProvider ? t(m.managedBy, { provider: u.ssoProvider }) : t(m.noPassword)}</span>
            </MenuItem>
          )}
          {u.lockedUntil ? (
            <MenuItem onSelect={() => void run(() => unlock.mutateAsync(path), tt(m.canSignInAgain, { name: u.name }), tt(m.lockoutCleared))}>
              <Icon name="key" />
              {t(m.unlockSignIn)} <span className="k ml-auto">{t(m.until, { time: clock(u.lockedUntil) })}</span>
            </MenuItem>
          ) : null}
          <MenuSeparator />
          {u.instanceAdmin ? (
            <MenuItem onSelect={() => onDialog('unadmin')}>
              <Icon name="star" />
              {t(m.removeAdmin)}
            </MenuItem>
          ) : (
            <MenuItem onSelect={() => onDialog('admin')}>
              <Icon name="star" />
              {t(m.makeAdmin)}
            </MenuItem>
          )}
          <MenuSeparator />
          <MenuItem disabled={u.self} onSelect={() => onDialog('deactivate')} className="text-bad-fg">
            <Icon name="ban" />
            {t(m.deactivateAccount)} {u.self ? <span className="k ml-auto">{t(m.thatsYou)}</span> : null}
          </MenuItem>
        </>
      )}
    </Menu>
  )
}

function UsersSkeleton() {
  const t = useT()
  return (
    <div aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
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
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/policy')
  const p = q.data
  const enforced = p?.sso.filter((x) => x.status === 'enabled' && x.enforced.length > 0) ?? []
  const twofa = p
    ? p.twoFactor === 'off'
      ? t(m.twoFAPolicyOff)
      : t(m.twoFAPolicy, { who: p.twoFactor === 'everyone' ? 'everyone' : 'sensitive', grace: p.graceDays, inGrace: p.inGrace.length })
    : ''
  return (
    <section className="card" aria-labelledby="snap-h">
      <CardSection
        title={<span id="snap-h">{t(m.policyTitle)}</span>}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/admin/authentication" params={{ workspace: slug }}>
            {t(m.authentication)}
          </ButtonLink>
        }
      />
      <div className="pad">
        {q.isError ? (
          <span className="k">
            {t(m.policyLoadFailed)} {errorMessage(q.error)}
          </span>
        ) : !p ? (
          <div className="col gap-2.5" aria-busy="true">
            <Skeleton w="80%" />
            <Skeleton w="70%" />
            <Skeleton w="60%" />
          </div>
        ) : (
          <dl className="kv">
            <dt>{t(m.ssoEnforced)}</dt>
            <dd>
              {enforced.length === 0
                ? t(m.ssoEnforcedNone)
                : t(m.ssoEnforcedList, { providers: enforced.map((x) => `${x.enforced.join(', ')} → ${x.name}`).join('; ') })}
            </dd>
            <dt>{t(m.twoFA)}</dt>
            <dd>{twofa}</dd>
            <dt>{t(m.lockout)}</dt>
            <dd>{t(m.lockoutValue, { attempts: p.lockoutAttempts, duration: lockWords(p.lockoutMinutes), locked: p.lockedNow })}</dd>
          </dl>
        )}
      </div>
    </section>
  )
}

export function lockWords(minutes: number) {
  if (minutes % 60 === 0) {
    const h = minutes / 60
    return tt(m.lockHours, { count: h })
  }
  return tt(m.lockMinutes, { count: minutes })
}

/** The latest sign-in attempts, from the instance audit log. */
function RecentSignIns() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/sign-ins', { params: { query: { limit: 4 } } })
  return (
    <section className="card" aria-labelledby="signins-h">
      <CardSection
        title={<span id="signins-h">{t(m.signInsTitle)}</span>}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/admin/audit" params={{ workspace: slug }}>
            {t(m.instanceAuditLog)}
          </ButtonLink>
        }
      />
      {q.isError ? (
        <div className="pad">
          <span className="k">
            {t(m.signInsLoadFailed)} {errorMessage(q.error)}
          </span>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3" aria-busy="true">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="90%" />
          ))}
        </div>
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <span className="k">{t(m.noSignIns)}</span>
        </div>
      ) : (
        <AccessTable caption={t(m.signInsTitle)}>
          <thead>
            <tr>
              <Th>{t(m.colWhen)}</Th>
              <Th>{t(m.colEvent)}</Th>
              <Th>{t(m.colAccount)}</Th>
              <Th>{t(m.colResult)}</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.map((e) => (
              <tr key={e.id}>
                <Td className="min-w-0 whitespace-nowrap!">{eventWhen(e.at)}</Td>
                <Td className="whitespace-nowrap!">{e.event}</Td>
                <Td>{e.account}</Td>
                <Td>
                  {e.result === 'allowed' ? (
                    <Badge tone="ok">{t(m.allowed)}</Badge>
                  ) : e.result === 'recovered' ? (
                    <Badge tone="warn">{t(m.thenAllowed)}</Badge>
                  ) : (
                    <span className="col gap-0.5">
                      <Badge tone="bad">{t(m.rejected)}</Badge>
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
  if (days === 1) return tt(m.yesterday)
  return usDate(at)
}

function startOfDay(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}
