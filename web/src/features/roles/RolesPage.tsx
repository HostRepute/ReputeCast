import { useQueryClient } from '@tanstack/react-query'
import { Fragment, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { monthDay } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { AccessTable, GroupRow, HasMark, RoleBadge } from '../access/parts'

type Role = Schemas['RoleSummary']
type RoleList = Schemas['RoleList']

/** A5 §5.8 Roles: the role list and the compare-roles matrix (Roles board). */
export function RolesPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/roles', { params: { path: { workspace: slug } } })
  const canManage = q.data?.canManage ?? false
  return (
    <Page
      title="Settings"
      tabs={settingsTabs(slug)}
      actions={
        canManage ? (
          <ButtonLink variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }}>
            New custom role
          </ButtonLink>
        ) : null
      }
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the roles" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <RolesSkeleton />
      ) : (
        <>
          <RolesCard list={q.data} />
          <CompareCard list={q.data} />
        </>
      )}
    </Page>
  )
}

function RolesCard({ list }: { list: RoleList }) {
  const { slug } = useWorkspace()
  const [deleting, setDeleting] = useState<Role>()
  const reason = list.canManage ? undefined : deniedReason('team:manage', list.helpers)
  return (
    <section className="card" aria-labelledby="roles-h">
      <CardSection
        title={<span id="roles-h">Roles · {list.roles.length}</span>}
        caption="Built-in roles can't be edited. Duplicate one to start a custom role."
        actions={
          list.canManage ? (
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }}>
              New custom role
            </ButtonLink>
          ) : null
        }
      />
      <AccessTable caption="Roles">
        <thead>
          <tr>
            <Th>Role</Th>
            <Th>Type</Th>
            <Th>Members</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {list.roles.map((r) => (
            <tr key={r.id}>
              <Td>
                <div className="col gap-0.5">
                  <span className="row gap-2">
                    <RoleBadge role={r} />
                    {r.builtin ? <Badge tone="neutral">read-only</Badge> : null}
                  </span>
                  <span className="k whitespace-normal">{r.description}</span>
                </div>
              </Td>
              <Td>{r.builtin ? 'Built-in' : `Custom · ${r.createdBy ? `created by ${r.createdBy}, ` : 'created '}${monthDay(r.createdAt)}`}</Td>
              <Td>{membersText(r)}</Td>
              <Td>
                <div className="row gap-1.5">
                  {reason ? (
                    <Button size="sm" disabledReason={reason}>
                      {r.builtin ? 'Duplicate to customize' : 'Edit'}
                    </Button>
                  ) : r.builtin ? (
                    <ButtonLink size="sm" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }} search={{ from: r.id }}>
                      Duplicate to customize
                    </ButtonLink>
                  ) : (
                    <>
                      <ButtonLink size="sm" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: r.id }} aria-label={`Edit ${r.name}`}>
                        Edit
                      </ButtonLink>
                      <ButtonLink
                        size="sm"
                        to="/w/$workspace/settings/roles/$roleId"
                        params={{ workspace: slug, roleId: 'new' }}
                        search={{ from: r.id }}
                        aria-label={`Duplicate ${r.name}`}
                      >
                        Duplicate
                      </ButtonLink>
                      <Button size="sm" variant="danger" onClick={() => setDeleting(r)} aria-label={`Delete ${r.name}`}>
                        Delete
                      </Button>
                    </>
                  )}
                </div>
              </Td>
            </tr>
          ))}
        </tbody>
      </AccessTable>
      {deleting ? <DeleteRoleDialog role={deleting} onClose={() => setDeleting(undefined)} /> : null}
    </section>
  )
}

/** "1 · Linh Nguyen", "1 active · Marcus Chen (Tom Becker deactivated)", "1 · Sofia Rossi (+1 invited)". */
export function membersText(r: Role): string {
  const active = r.members.filter((m) => m.status === 'active').map((m) => m.name)
  const inactive = r.members.filter((m) => m.status !== 'active').map((m) => m.name)
  const names = active.length > 3 ? `${active.slice(0, 3).join(', ')} +${active.length - 3}` : active.join(', ')
  let out = r.active === 0 ? 'None' : `${r.active}${r.deactivated > 0 ? ' active' : ''} · ${names}`
  if (r.deactivated > 0) out += r.deactivated === 1 && inactive[0] ? ` (${inactive[0]} deactivated)` : ` (${r.deactivated} deactivated)`
  if (r.invited > 0) out += ` (+${r.invited} invited)`
  return out
}

function DeleteRoleDialog({ role, onClose }: { role: Role; onClose: () => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const assigned = role.active + role.deactivated
  // A role someone still holds can't go (RoleEditorStates, "Can't delete").
  if (assigned > 0)
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">Dialog · delete role</span>}
        title={`“${role.name}” can't be deleted yet`}
        hideClose
        footer={
          <Button variant="primary" onClick={onClose}>
            Close
          </Button>
        }
      >
        <Note tone="bad" label="Can't delete">
          “{role.name}” is assigned to {assigned} {assigned === 1 ? 'member' : 'members'}. Move {role.members.map((m) => m.name).join(', ')} to another role first,
          then delete.
        </Note>
      </Dialog>
    )
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge="Delete role"
      title={`Delete “${role.name}”?`}
      description="The role and its permission set are deleted. Nobody holds it, so no one loses access; the audit log keeps its history."
      confirmLabel="Delete role"
      busyLabel="Deleting…"
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/roles/{roleId}', { params: { path: { workspace: slug, roleId: role.id } } })
        toast({ kind: 'deleted', title: `Role “${role.name}” deleted`, detail: 'logged as role.deleted' })
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
      }}
    />
  )
}

function CompareCard({ list }: { list: RoleList }) {
  const [hidden, setHidden] = useState<string[]>([])
  const shown = list.roles.filter((r) => !hidden.includes(r.id))
  const perms = list.permissions.filter((p) => !p.ownerOnly)
  const has = (r: Role, key: string) => r.permissions.includes(key) || implied(list.permissions, r.permissions, key)
  return (
    <section className="card" aria-labelledby="cmp-h">
      <CardSection
        title={<span id="cmp-h">Compare roles</span>}
        caption="Side by side · ⚑ marks sensitive permissions that trigger the two-factor policy"
        actions={
          <div className="row gap-2" role="group" aria-label="Columns">
            <span className="k" aria-hidden="true">
              Columns:
            </span>
            {list.roles.map((r) => {
              const on = !hidden.includes(r.id)
              return (
                <button
                  key={r.id}
                  type="button"
                  className={on ? 'chip box-content border-line-strong! bg-white! text-ink! hover:bg-ground!' : 'chip box-content border-dashed! text-ink-3! line-through'}
                  aria-pressed={on}
                  title={on ? `Hide the ${r.name} column` : `Show the ${r.name} column`}
                  onClick={() => setHidden((cur) => (on ? (shown.length > 1 ? [...cur, r.id] : cur) : cur.filter((x) => x !== r.id)))}
                >
                  {r.name}
                </button>
              )
            })}
          </div>
        }
      />
      <AccessTable caption="Compare roles">
        <thead>
          <tr>
            <Th>Permission</Th>
            {shown.map((r) => (
              <Th key={r.id} align="center">
                {r.name}
              </Th>
            ))}
          </tr>
        </thead>
        <tbody>
          {list.groups.map((g) => {
            const rows = perms.filter((p) => p.group === g)
            if (rows.length === 0) return null
            return (
              <Fragment key={g}>
                <GroupRow label={g} cells={shown.length} />
                {rows.map((p) => (
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
                    {shown.map((r) => (
                      <Td key={r.id} align="center">
                        <HasMark on={has(r, p.key)} />
                      </Td>
                    ))}
                  </tr>
                ))}
              </Fragment>
            )
          })}
        </tbody>
      </AccessTable>
    </section>
  )
}

/** Whether a stored permission set grants key through another permission (edit includes view). */
export function implied(catalog: Schemas['PermissionInfo'][], held: string[], key: string): boolean {
  return catalog.some((p) => held.includes(p.key) && p.implies.includes(key))
}

function RolesSkeleton() {
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        Loading the roles
      </span>
      <section className="card">
        <div className="sec">
          <div className="flex flex-col gap-1.5">
            <Skeleton w={90} h={14} />
            <Skeleton w={300} h={10} />
          </div>
        </div>
        <div className="pad flex flex-col gap-4">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="flex items-center gap-6">
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton w={110} h={18} />
                <Skeleton w="70%" h={10} />
              </div>
              <Skeleton w={80} h={12} />
              <Skeleton w={140} h={12} />
              <Skeleton w={150} h={28} />
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
