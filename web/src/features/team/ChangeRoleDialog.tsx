import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuLabel } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { useWorkspace } from '@/lib/session'

type Member = Pick<Schemas['TeamMember'], 'id' | 'name' | 'role' | 'scope' | 'source'>

/**
 * Change a member's role or scope. The role list and the scope choices come
 * from the invitation options, so the same escalation rules apply: roles the
 * actor can't grant are listed but disabled with the reason.
 */
export function ChangeRoleDialog({ member, open, onOpenChange }: { member: Member; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const opts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/invitations/options', { params: { path: { workspace: slug } } })
  if (opts.data) return <ChangeRoleForm opts={opts.data} member={member} open={open} close={() => onOpenChange(false)} />
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Dialog · change role or scope</span>}
      title={`Change role or scope for ${member.name}`}
      hideClose
      footer={<Button onClick={() => onOpenChange(false)}>Cancel</Button>}
    >
      {opts.isError ? (
        <div className="note b" role="alert">
          <span>{errorMessage(opts.error)}</span>
        </div>
      ) : (
        <div className="flex flex-col gap-3" aria-busy="true">
          <span className="vh" role="status">
            Loading
          </span>
          <Skeleton h={36} />
          <Skeleton h={36} />
        </div>
      )}
    </Dialog>
  )
}

function ChangeRoleForm({ opts, member, open, close }: { opts: Schemas['TeamInviteOptions']; member: Member; open: boolean; close: () => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const roles = useMemo(() => orderRoles(opts.roles, member.role.id), [opts.roles, member.role.id])
  const [roleId, setRoleId] = useState(member.role.id)
  const [lists, setLists] = useState<string[]>(member.scope.lists.map((l) => l.id))
  const [domains, setDomains] = useState<string[]>(member.scope.senderDomains)
  const [error, setError] = useState<string>()
  const change = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/members/{memberId}')
  const role = roles.find((r) => r.id === roleId)
  const first = member.name.split(' ')[0] ?? member.name
  const unchanged =
    roleId === member.role.id &&
    sameSet(
      lists,
      member.scope.lists.map((l) => l.id),
    ) &&
    sameSet(domains, member.scope.senderDomains)
  const blocked = role && !role.grantable && role.id !== member.role.id ? role.reason : undefined

  const submit = () => {
    setError(undefined)
    change.mutate(
      {
        params: { path: { workspace: slug, memberId: member.id } },
        body: { roleId, scope: { lists, senderDomains: domains } },
      },
      {
        onSuccess: () => {
          toast({ title: `${member.name} updated`, detail: roleId === member.role.id ? 'scope changed' : `now ${role?.name ?? 'updated'}` })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}'] })
          close()
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : errorMessage(e)),
      },
    )
  }

  const chosenLists = opts.lists.filter((l) => lists.includes(l.id))
  const freeLists = opts.lists.filter((l) => !lists.includes(l.id))
  const freeDomains = opts.senderDomains.filter((d) => !domains.includes(d))

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && close()}
      eyebrow={<span className="k">Dialog · change role or scope</span>}
      title={`Change role or scope for ${member.name}`}
      description={<span className="text-[13.5px]">Changes apply at once, on {first}&apos;s next request. Both the old and the new value go to the audit log.</span>}
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={close}>Cancel</Button>
          <Button variant="primary" loading={change.isPending} disabledReason={blocked ?? (unchanged ? 'Nothing changed yet.' : undefined)} onClick={submit}>
            Save changes
          </Button>
        </>
      }
    >
      <label className="lbl">
        Role
        <select className="in" value={roleId} onChange={(e) => setRoleId(e.target.value)}>
          {roles.map((r) => (
            <option key={r.id} value={r.id} disabled={!r.grantable && r.id !== member.role.id}>
              {r.name}
              {r.id === member.role.id ? ' (current)' : r.grantable ? '' : ' (you can’t grant it)'}
            </option>
          ))}
        </select>
        <span className="k font-medium">{blocked ?? role?.hint}</span>
      </label>
      {member.source === 'sso_group' ? (
        <div className="note i">
          <span>{first}&apos;s role comes from an SSO group mapping. A change here holds until the next sign-in applies the mapping again; change the mapping to make it last.</span>
        </div>
      ) : null}
      <div className="lbl">
        <span>
          Scope <span className="k font-medium">· optional</span>
        </span>
        <div className="row gap-1.5">
          {chosenLists.length + domains.length === 0 ? <span className="k">All lists and sender domains</span> : null}
          {chosenLists.map((l) => (
            <span key={l.id} className="chip">
              {l.name} list
              <button type="button" aria-label={`Remove scope ${l.name}`} onClick={() => setLists((cur) => cur.filter((x) => x !== l.id))}>
                ×
              </button>
            </span>
          ))}
          {domains.map((d) => (
            <span key={d} className="chip">
              {d}
              <button type="button" aria-label={`Remove scope ${d}`} onClick={() => setDomains((cur) => cur.filter((x) => x !== d))}>
                ×
              </button>
            </span>
          ))}
          {freeLists.length + freeDomains.length > 0 ? (
            <Menu trigger={<Button size="sm">Add list or sender domain</Button>} align="start" label="Add to scope">
              {freeLists.length ? <MenuLabel>Lists</MenuLabel> : null}
              {freeLists.map((l) => (
                <MenuItem key={l.id} onSelect={() => setLists((cur) => [...cur, l.id])}>
                  {l.name}
                </MenuItem>
              ))}
              {freeDomains.length ? <MenuLabel>Sender domains</MenuLabel> : null}
              {freeDomains.map((d) => (
                <MenuItem key={d} onSelect={() => setDomains((cur) => [...cur, d])}>
                  {d}
                </MenuItem>
              ))}
            </Menu>
          ) : null}
        </div>
        <span className="k font-medium">Scoped members only see and act on the selected lists or sender domains.</span>
      </div>
      {error ? (
        <div className="note b" role="alert">
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}

function sameSet(a: string[], b: string[]) {
  return a.length === b.length && a.every((x) => b.includes(x))
}

/** Owner, Admin, Editor, Viewer, then custom roles, as on the Roles board. Owner only when it is the current role. */
function orderRoles(roles: Schemas['TeamInviteRole'][], current: string) {
  const rank = (r: Schemas['TeamInviteRole']) => (!r.builtin ? 4 : ({ owner: 0, admin: 1, editor: 2, viewer: 3 } as Record<string, number>)[r.key] ?? 5)
  return roles.filter((r) => !(r.builtin && r.key === 'owner') || r.id === current).sort((a, b) => rank(a) - rank(b))
}
