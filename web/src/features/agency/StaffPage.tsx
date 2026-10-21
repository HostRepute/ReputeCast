import { DropdownMenu } from 'radix-ui'
import { useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { RoleBadge } from '@/features/access/parts'
import { formatList, useT } from '@/i18n'
import { ago } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useAgencyCan, useAgencyMe, useRefreshAgency } from './api'
import { AgencyPage } from './AgencyShell'
import { m } from './messages'
import { m as sm } from './staff.messages'
import { ClientTile } from './parts'

type View = Schemas['AgencyStaffView']
type Member = Schemas['AgencyStaffMember']
type Row = Schemas['AgencyAssignmentRow']

/** Agency › Staff (AgencyStates): the staff, their levels and the assignment matrix. */
export function StaffPage() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/agency/staff')
  const canManage = useAgencyCan('agency:manage')
  const [adding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Member>()
  const v = q.data
  return (
    <AgencyPage
      title={t(m.navStaff)}
      subtitle={v ? t(sm.subtitle, { staff: v.staff.length, clients: v.clients, agency: v.hasAgencyWorkspace ? 1 : 0 }) : undefined}
      actions={
        canManage ? (
          <Button variant="primary" icon="plus" onClick={() => setAdding(true)}>
            {t(sm.addStaff)}
          </Button>
        ) : (
          <span />
        )
      }
    >
      {q.isError ? (
        <ErrorPanel title={t(sm.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !v ? (
        <div className="flex flex-col gap-5" aria-busy="true">
          <Skeleton w="100%" h={360} r={10} />
          <Skeleton w="100%" h={420} r={10} />
        </div>
      ) : (
        <>
          {!v.canManage && v.reason ? <Note tone="info">{v.reason}</Note> : null}
          <StaffCard view={v} onEdit={setEditing} canManage={canManage} />
          <AssignmentsCard view={v} />
        </>
      )}
      {adding && v ? <AddStaffDialog view={v} onClose={() => setAdding(false)} /> : null}
      {editing && v ? <EditStaffDialog member={editing} view={v} onClose={() => setEditing(undefined)} /> : null}
    </AgencyPage>
  )
}

/** "Contoso Coffee, Fabrikam Outdoors, Coho Winery": names in a list, not a sentence. */
function unitList(names: string[]) {
  try {
    return new Intl.ListFormat(document.documentElement.lang || 'en', { type: 'unit', style: 'short' }).format(names)
  } catch {
    return names.join(', ')
  }
}

/** Client names in the order of the assignment matrix (the order they joined). */
function inMatrixOrder(names: string[], view: View) {
  const order = view.assignments.map((a) => a.workspace.name)
  return [...names].sort((a, b) => order.indexOf(a) - order.indexOf(b))
}

function levelBadge(t: ReturnType<typeof useT>, level: string) {
  return level === 'agency_manager' ? <Badge tone="purple">{t(m.levelAgencyManager)}</Badge> : <Badge tone="info">{t(m.levelAccountManager)}</Badge>
}

function StaffCard({ view, onEdit, canManage }: { view: View; onEdit: (m: Member) => void; canManage: boolean }) {
  const t = useT()
  return (
    <section className="card" aria-label={t(sm.staffTitle, { n: view.staff.length })}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2">{t(sm.staffTitle, { n: view.staff.length })}</h2>
          <span className="k">{t(sm.staffCaption)}</span>
        </div>
      </div>
      <div className="relative overflow-x-auto">
        <table className="tbl" aria-label={t(sm.staffTitle, { n: view.staff.length })}>
          <thead>
            <tr>
              <th scope="col">{t(sm.colPerson)}</th>
              <th scope="col">{t(sm.colLevel)}</th>
              <th scope="col">{t(sm.colClients)}</th>
              <th scope="col">{t(sm.colLastActive)}</th>
              <th scope="col">
                <span className="vh">{t(sm.colActions)}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {view.staff.map((s) => (
              <tr key={s.id}>
                <td>
                  <span className="flex items-center gap-2.5">
                    <Avatar name={s.name} color={s.avatarColor} size={28} />
                    <span className="flex min-w-0 flex-col">
                      <b>{s.name}</b>
                      <span className="k">{s.email}</span>
                    </span>
                  </span>
                </td>
                <td>
                  <span className="flex flex-col items-start gap-1">
                    {levelBadge(t, s.level)}
                    {s.instanceAdmin ? <span className="k">{t(sm.instanceAdmin)}</span> : s.title ? <span className="k">{s.title}</span> : null}
                    {s.invited ? <Badge tone="warn">{t(sm.invited)}</Badge> : null}
                  </span>
                </td>
                <td className="whitespace-normal">
                  <span className="flex flex-col items-start gap-1">
                    <span>{s.everyClient ? t(sm.everyClient) : s.clients.length ? unitList(inMatrixOrder(s.clients, view)) : t(sm.noClients)}</span>
                    {s.roles.length ? (
                      <span className="flex flex-wrap gap-1">
                        {s.roles.map((r) => (
                          <RoleBadge key={r.key} role={r} />
                        ))}
                      </span>
                    ) : null}
                  </span>
                </td>
                <td className="k">{s.invited ? t(sm.notYet) : ago(s.lastActiveAt)}</td>
                <td className="r">
                  {canManage ? (
                    <Button size="sm" onClick={() => onEdit(s)} aria-label={t(sm.editName, { name: s.name })}>
                      {t(m.edit)}
                    </Button>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center gap-2.5 border-t border-line-2 px-[18px] py-3">
        <span className="k">{t(sm.staffFoot)}</span>
      </div>
    </section>
  )
}

function AssignmentsCard({ view }: { view: View }) {
  const t = useT()
  const { data: agency } = useAgencyMe()
  const owner = view.staff.find((s) => s.instanceAdmin)
  const [denied, setDenied] = useState<string>()
  return (
    <section className="card" aria-label={t(sm.matrixTitle)}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2">{t(sm.matrixTitle)}</h2>
          <span className="k">{owner ? t(sm.matrixCaption, { owner: owner.name }) : t(sm.matrixCaptionPlain)}</span>
        </div>
      </div>
      {denied ? (
        <div className="px-[18px] pt-3">
          <Note tone="warn" role="alert">
            {denied}
          </Note>
        </div>
      ) : null}
      <div className="relative overflow-x-auto">
        <table className="tbl" aria-label={t(sm.matrixTitle)}>
          <thead>
            <tr>
              <th scope="col">{t(m.colClient)}</th>
              {view.staff.map((s) => (
                <th key={s.id} scope="col">
                  <span className="inline-flex items-center gap-1.5">
                    <Avatar name={s.name} color={s.avatarColor} size={20} fontSize={9} />
                    {s.name.split(/\s+/)[0]}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {view.assignments.map((row) => (
              <tr key={row.workspace.id}>
                <td>
                  <span className="flex items-center gap-2.5 whitespace-nowrap">
                    <ClientTile ws={row.workspace} size={22} />
                    <b>{row.workspace.kind === 'agency' ? t(m.agencyWorkspaceName, { name: row.workspace.name }) : row.workspace.name}</b>
                  </span>
                </td>
                {view.staff.map((s) => (
                  <td key={s.id}>
                    <AssignmentCell row={row} member={s} onDenied={setDenied} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center gap-2.5 border-t border-line-2 px-[18px] py-3">
        <span className="k">{t(sm.matrixFoot, { agency: agency?.name ?? '' })}</span>
      </div>
    </section>
  )
}

/** One staff member in one workspace: their role, or "Assign". */
function AssignmentCell({ row, member, onDenied }: { row: Row; member: Member; onDenied: (msg: string) => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const [removing, setRemoving] = useState(false)
  const cell = row.cells.find((c) => c.staffId === member.id)
  const role = cell?.role
  const assign = async (roleId: string) => {
    try {
      await http.PUT('/api/v1/agency/assignments', { body: { staffId: member.id, workspaceId: row.workspace.id, roleId } })
      await refresh()
      toast({ kind: 'success', title: t(sm.assigned, { name: member.name, client: row.workspace.name }) })
    } catch (e) {
      if (e instanceof ApiError && e.status === 403) onDenied(e.message)
      else toast({ kind: 'error', title: t(sm.assignFailed), detail: errorMessage(e) })
    }
  }
  if (member.invited && !role) return <span className="k">{t(m.dash)}</span>
  const trigger = role ? (
    <button type="button" className="inline-flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0" aria-label={t(sm.changeRole, { role: roleName(role), name: member.name, client: row.workspace.name })} disabled={!row.canAssign}>
      <RoleBadge role={role} />
      {row.canAssign ? <Icon name="chevronDown" size={12} className="text-ink-3" /> : null}
    </button>
  ) : (
    <Button size="sm" disabled={!row.canAssign} aria-label={t(sm.assignName, { name: member.name, client: row.workspace.name })}>
      {t(sm.assign)}
    </Button>
  )
  if (!row.canAssign) return role ? <RoleBadge role={role} /> : <span className="k">{t(m.dash)}</span>
  return (
    <>
      <DropdownMenu.Root modal={false}>
        <DropdownMenu.Trigger asChild>{trigger}</DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="menu w-[340px]" align="start" sideOffset={6} collisionPadding={12} aria-label={t(sm.rolesIn, { name: member.name, client: row.workspace.name })}>
            <DropdownMenu.Label className="mh">{t(sm.rolesIn, { name: member.name, client: row.workspace.name })}</DropdownMenu.Label>
            {row.roles.map((r) => {
              const current = role?.key === r.key
              const yours = row.yourRole === r.key
              return (
                <DropdownMenu.Item
                  key={r.id}
                  disabled={!r.grantable || current}
                  onSelect={() => void assign(r.id ?? '')}
                  className="h-auto! items-start! py-2!"
                  style={current ? { background: '#F0FDFA' } : undefined}
                >
                  <span className="flex min-w-0 flex-1 flex-col" style={{ opacity: r.grantable || current ? 1 : 0.55 }}>
                    <b className="text-[13px]">{roleName(r)}</b>
                    {r.description ? <span className="k">{yours ? t(sm.yourRole) : r.description}</span> : yours ? <span className="k">{t(sm.yourRole)}</span> : null}
                  </span>
                  {current ? <Icon name="check" size={14} className="mt-0.5 text-accent" /> : !r.grantable ? <span className="k text-[12px] whitespace-nowrap">{t(sm.notYours)}</span> : null}
                </DropdownMenu.Item>
              )
            })}
            {role && member.userId ? (
              <>
                <DropdownMenu.Separator />
                <DropdownMenu.Item onSelect={() => setRemoving(true)} className="text-bad-fg!">
                  {t(sm.removeFrom, { client: row.workspace.name })}
                </DropdownMenu.Item>
              </>
            ) : null}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      {removing ? <RemoveAssignmentDialog row={row} member={member} onClose={() => setRemoving(false)} /> : null}
    </>
  )
}

/** Removing an assignment removes the membership, and asks about the person's API keys there. */
function RemoveAssignmentDialog({ row, member, onClose }: { row: Row; member: Member; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const keys = $api.useQuery('get', '/api/v1/agency/assignments/keys', { params: { query: { staffId: member.id, workspaceId: row.workspace.id } } })
  const [revoke, setRevoke] = useState(true)
  const [busy, setBusy] = useState(false)
  const list = keys.data?.keys ?? []
  const submit = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/agency/assignments/remove', { body: { staffId: member.id, workspaceId: row.workspace.id, revokeKeys: list.length > 0 && revoke } })
      await refresh()
      toast({ kind: 'success', title: t(sm.removed, { name: member.name, client: row.workspace.name }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(sm.removeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(sm.removeTitle, { name: member.name, client: row.workspace.name })}
      description={t(sm.removeBody, { name: member.name, client: row.workspace.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="danger-primary" loading={busy} onClick={submit}>
            {t(sm.removeConfirm)}
          </Button>
        </>
      }
    >
      {keys.isPending ? (
        <Skeleton w="100%" h={60} r={8} />
      ) : list.length ? (
        <div className="flex flex-col gap-2">
          <span className="text-[13px]">{t(sm.keysQuestion, { n: list.length, names: formatList(list.map((k) => k.name)) })}</span>
          <label className="ag-optc">
            <input type="radio" name="keys" checked={revoke} onChange={() => setRevoke(true)} />
            <span className="flex flex-col font-semibold">
              {t(sm.revokeKeys)}
              <span className="k">{t(sm.revokeKeysHint)}</span>
            </span>
          </label>
          <label className="ag-optc">
            <input type="radio" name="keys" checked={!revoke} onChange={() => setRevoke(false)} />
            <span className="flex flex-col font-semibold">
              {t(sm.keepKeys)}
              <span className="k">{t(sm.keepKeysHint)}</span>
            </span>
          </label>
        </div>
      ) : (
        <span className="k">{t(sm.noKeys)}</span>
      )}
    </Dialog>
  )
}

/** Add staff (AgencyStates): an account of the instance, or an invitation by email. */
function AddStaffDialog({ view, onClose }: { view: View; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const candidates = $api.useQuery('get', '/api/v1/agency/staff/candidates')
  const [mode, setMode] = useState<'users' | 'email'>('email')
  const [userId, setUserId] = useState('')
  const [email, setEmail] = useState('')
  const [level, setLevel] = useState<'account_manager' | 'agency_manager'>('account_manager')
  const [picked, setPicked] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const clients = view.assignments.filter((a) => a.workspace.kind === 'client')
  const editor = (row: Row) => row.roles.find((r) => r.key === 'editor')?.id ?? row.roles[0]?.id ?? ''
  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const assignments = level === 'account_manager' ? Object.entries(picked).map(([workspaceId, roleId]) => ({ workspaceId, roleId })) : []
      const { data } = await http.POST('/api/v1/agency/staff', {
        body: { level, ...(mode === 'users' ? { userId } : { email: email.trim() }), assignments },
      })
      await refresh()
      toast({
        kind: 'success',
        title: mode === 'email' ? t(sm.invitedToast, { email: email.trim() }) : t(sm.addedToast),
        detail: data?.assigned ? t(sm.assignedCount, { n: data.assigned }) : undefined,
      })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{t(sm.addCrumb)}</span>}
      title={t(sm.addStaff)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={mode === 'users' ? !userId : !email.trim()} onClick={submit}>
            {mode === 'email' ? t(sm.sendInvitation) : t(sm.addStaff)}
          </Button>
        </>
      }
    >
      <div className="seg w-fit max-w-full" role="group" aria-label={t(sm.addFrom)}>
        <button type="button" className={mode === 'users' ? 'on' : undefined} aria-pressed={mode === 'users'} onClick={() => setMode('users')}>
          {t(sm.fromUsers)}
        </button>
        <button type="button" className={mode === 'email' ? 'on' : undefined} aria-pressed={mode === 'email'} onClick={() => setMode('email')}>
          {t(sm.byEmail)}
        </button>
      </div>
      {mode === 'users' ? (
        <Field label={t(sm.person)}>
          <Select value={userId} onChange={(e) => setUserId(e.target.value)}>
            <option value="">{t(sm.pickPerson)}</option>
            {(candidates.data ?? []).map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} · {c.email}
              </option>
            ))}
          </Select>
        </Field>
      ) : (
        <Field label={t(sm.email)}>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t(sm.emailPlaceholder)} autoFocus />
        </Field>
      )}
      <div className="lbl" role="radiogroup" aria-labelledby="as-level">
        <span id="as-level">{t(sm.level)}</span>
        <label className="ag-optc">
          <input type="radio" name="level" checked={level === 'account_manager'} onChange={() => setLevel('account_manager')} />
          <span className="flex flex-col font-semibold">
            {t(m.levelAccountManager)}
            <span className="k">{t(sm.accountManagerHint)}</span>
          </span>
        </label>
        <label className="ag-optc">
          <input type="radio" name="level" checked={level === 'agency_manager'} onChange={() => setLevel('agency_manager')} />
          <span className="flex flex-col font-semibold">
            {t(m.levelAgencyManager)}
            <span className="k">{t(sm.agencyManagerHint)}</span>
          </span>
        </label>
      </div>
      {level === 'account_manager' ? (
        <div className="lbl" role="group" aria-labelledby="as-clients">
          <span id="as-clients">{t(sm.clientsAndRole)}</span>
          <div className="flex flex-col">
            {clients.map((row) => {
              const on = row.workspace.id in picked
              return (
                <div key={row.workspace.id} className="flex items-center gap-2.5 border-t border-line-2 py-2 first:border-t-0">
                  <input
                    type="checkbox"
                    checked={on}
                    aria-label={row.workspace.name}
                    onChange={(e) => {
                      const next = { ...picked }
                      if (e.target.checked) next[row.workspace.id] = editor(row)
                      else delete next[row.workspace.id]
                      setPicked(next)
                    }}
                  />
                  <ClientTile ws={row.workspace} size={20} />
                  <span className="flex-1 text-[14px] font-medium">{row.workspace.name}</span>
                  <Select
                    value={picked[row.workspace.id] ?? editor(row)}
                    onChange={(e) => setPicked({ ...picked, [row.workspace.id]: e.target.value })}
                    disabled={!on}
                    aria-label={t(sm.roleIn, { client: row.workspace.name })}
                    className="w-[110px]"
                  >
                    {row.roles
                      .filter((r) => r.key !== 'owner')
                      .map((r) => (
                        <option key={r.id} value={r.id} disabled={!r.grantable}>
                          {roleName(r)}
                        </option>
                      ))}
                  </Select>
                </div>
              )
            })}
          </div>
        </div>
      ) : (
        <span className="k">{t(sm.agencyManagerNote)}</span>
      )}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </Dialog>
  )
}

const agencyPerms = [
  { key: 'agency:usage', label: sm.permUsage },
  { key: 'agency:library', label: sm.permLibrary },
] as const

/** Edit a staff member: level, title, what they hold; or take them off the staff. */
function EditStaffDialog({ member, onClose }: { member: Member; view: View; onClose: () => void }) {
  const t = useT()
  const { data: agency } = useAgencyMe()
  const refresh = useRefreshAgency()
  const [level, setLevel] = useState(member.level)
  const [title, setTitle] = useState(member.title ?? '')
  const [perms, setPerms] = useState<string[]>(member.permissions)
  const [busy, setBusy] = useState(false)
  const [removing, setRemoving] = useState(false)
  const [removeAssignments, setRemoveAssignments] = useState(true)
  const [revokeKeys, setRevokeKeys] = useState(false)
  const admin = member.instanceAdmin
  const save = async () => {
    setBusy(true)
    try {
      const narrowed: ('agency:view' | 'agency:usage' | 'agency:library')[] | undefined =
        level === 'account_manager' ? ['agency:view', ...agencyPerms.map((p) => p.key).filter((k) => perms.includes(k))] : undefined
      await http.PATCH('/api/v1/agency/staff/{staffId}', { params: { path: { staffId: member.id } }, body: { level, title: title.trim(), permissions: narrowed } })
      await refresh()
      toast({ kind: 'success', title: t(sm.saved, { name: member.name }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(sm.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/agency/staff/{staffId}/remove', { params: { path: { staffId: member.id } }, body: { removeAssignments, revokeKeys: removeAssignments && revokeKeys } })
      await refresh()
      toast({ kind: 'deleted', title: t(sm.takenOff, { name: member.name }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(sm.removeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  if (removing) {
    return (
      <Dialog
        open
        onOpenChange={(o) => (o ? null : onClose())}
        title={t(sm.takeOffTitle, { name: member.name })}
        description={t(sm.takeOffBody, { name: member.name, agency: agency?.name ?? '' })}
        footer={
          <>
            <Button onClick={() => setRemoving(false)}>{t(m.cancel)}</Button>
            <Button variant="danger-primary" loading={busy} onClick={remove}>
              {t(sm.takeOff)}
            </Button>
          </>
        }
      >
        <label className="flex items-start gap-2 text-[13px]">
          <input type="checkbox" checked={removeAssignments} onChange={(e) => setRemoveAssignments(e.target.checked)} />
          <span>{t(sm.alsoRemoveAssignments)}</span>
        </label>
        {removeAssignments ? (
          <label className="flex items-start gap-2 text-[13px]">
            <input type="checkbox" checked={revokeKeys} onChange={(e) => setRevokeKeys(e.target.checked)} />
            <span>{t(sm.alsoRevokeKeys)}</span>
          </label>
        ) : null}
      </Dialog>
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(sm.editTitle, { name: member.name })}
      description={member.email}
      footer={
        <>
          {!admin && member.userId !== undefined ? (
            <Button variant="danger" className="mr-auto" onClick={() => setRemoving(true)}>
              {t(sm.takeOff)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={save}>
            {t(m.save)}
          </Button>
        </>
      }
    >
      {admin ? <Note tone="info">{t(sm.adminNote)}</Note> : null}
      <div className="lbl" role="radiogroup" aria-labelledby="es-level">
        <span id="es-level">{t(sm.level)}</span>
        {(['account_manager', 'agency_manager'] as const).map((l) => (
          <label key={l} className="ag-optc">
            <input type="radio" name="es-level" checked={level === l} disabled={admin} onChange={() => setLevel(l)} />
            <span className="flex flex-col font-semibold">
              {l === 'agency_manager' ? t(m.levelAgencyManager) : t(m.levelAccountManager)}
              <span className="k">{l === 'agency_manager' ? t(sm.agencyManagerHint) : t(sm.accountManagerHint)}</span>
            </span>
          </label>
        ))}
      </div>
      <Field label={t(sm.titleField)} hint={t(sm.titleHint)}>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={60} />
      </Field>
      {level === 'account_manager' ? (
        <div className="lbl" role="group" aria-labelledby="es-perms">
          <span id="es-perms">{t(sm.holds)}</span>
          <label className="flex items-center gap-2 text-[13px] font-normal text-ink">
            <input type="checkbox" checked disabled />
            {t(sm.permView)}
          </label>
          {agencyPerms.map((p) => (
            <label key={p.key} className="flex items-center gap-2 text-[13px] font-normal text-ink">
              <input type="checkbox" checked={perms.includes(p.key)} onChange={(e) => setPerms(e.target.checked ? [...perms, p.key] : perms.filter((x) => x !== p.key))} />
              {t(p.label)}
            </label>
          ))}
        </div>
      ) : null}
    </Dialog>
  )
}
