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
import { common } from '@/i18n/common'
import { t, useT } from '@/i18n'
import { monthDay } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { AccessTable, GroupRow, HasMark, RoleBadge } from '../access/parts'
import { m } from './messages'
import { matrixNote } from './notes'

type Role = Schemas['RoleSummary']
type RoleList = Schemas['RoleList']

/** A5 §5.8 Roles: the role list and the compare-roles matrix (Roles board). */
export function RolesPage() {
  const { slug } = useWorkspace()
  const tr = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/roles', { params: { path: { workspace: slug } } })
  const canManage = q.data?.canManage ?? false
  return (
    <Page
      title={tr(m.pageTitle)}
      tabs={settingsTabs(slug)}
      actions={
        canManage ? (
          <ButtonLink variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }}>
            {tr(m.newRole)}
          </ButtonLink>
        ) : null
      }
    >
      {q.isError ? (
        <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
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
  const tr = useT()
  const [deleting, setDeleting] = useState<Role>()
  const reason = list.canManage ? undefined : deniedReason('team:manage', list.helpers)
  return (
    <section className="card" aria-labelledby="roles-h">
      <CardSection
        title={<span id="roles-h">{tr(m.rolesTitle, { count: list.roles.length })}</span>}
        caption={tr(m.rolesCaption)}
        actions={
          list.canManage ? (
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }}>
              {tr(m.newRole)}
            </ButtonLink>
          ) : null
        }
      />
      <AccessTable caption={tr(m.rolesTableCaption)}>
        <thead>
          <tr>
            <Th>{tr(m.colRole)}</Th>
            <Th>{tr(m.colType)}</Th>
            <Th>{tr(m.colMembers)}</Th>
            <Th>
              <span className="vh">{tr(m.colActions)}</span>
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
                    {r.builtin ? <Badge tone="neutral">{tr(m.readOnlyBadge)}</Badge> : null}
                  </span>
                  <span className="k whitespace-normal">{r.description}</span>
                </div>
              </Td>
              <Td>
                {r.builtin
                  ? tr(m.typeBuiltin)
                  : r.createdBy
                    ? tr(m.typeCustomBy, { name: r.createdBy, date: monthDay(r.createdAt) })
                    : tr(m.typeCustom, { date: monthDay(r.createdAt) })}
              </Td>
              <Td>{membersText(r)}</Td>
              <Td>
                <div className="row gap-1.5">
                  {reason ? (
                    <Button size="sm" disabledReason={reason}>
                      {r.builtin ? tr(m.duplicateToCustomize) : tr(m.edit)}
                    </Button>
                  ) : r.builtin ? (
                    <ButtonLink size="sm" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }} search={{ from: r.id }}>
                      {tr(m.duplicateToCustomize)}
                    </ButtonLink>
                  ) : (
                    <>
                      <ButtonLink size="sm" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: r.id }} aria-label={tr(m.editLabel, { name: r.name })}>
                        {tr(m.edit)}
                      </ButtonLink>
                      <ButtonLink
                        size="sm"
                        to="/w/$workspace/settings/roles/$roleId"
                        params={{ workspace: slug, roleId: 'new' }}
                        search={{ from: r.id }}
                        aria-label={tr(m.duplicateLabel, { name: r.name })}
                      >
                        {tr(m.duplicate)}
                      </ButtonLink>
                      <Button size="sm" variant="danger" onClick={() => setDeleting(r)} aria-label={tr(m.deleteLabel, { name: r.name })}>
                        {tr(m.delete)}
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
  const active = r.members.filter((x) => x.status === 'active').map((x) => x.name)
  const inactive = r.members.filter((x) => x.status !== 'active').map((x) => x.name)
  const names = active.length > 3 ? t(m.membersNames, { names: active.slice(0, 3).join(', '), more: active.length - 3 }) : active.join(', ')
  let out = r.active === 0 ? t(m.membersNone) : t(r.deactivated > 0 ? m.membersCountActive : m.membersCount, { count: r.active, names })
  if (r.deactivated > 0) {
    out = r.deactivated === 1 && inactive[0] ? t(m.membersOneDeactivated, { members: out, name: inactive[0] }) : t(m.membersDeactivated, { members: out, count: r.deactivated })
  }
  if (r.invited > 0) out = t(m.membersInvited, { members: out, count: r.invited })
  return out
}

function DeleteRoleDialog({ role, onClose }: { role: Role; onClose: () => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const tr = useT()
  const assigned = role.active + role.deactivated
  // A role someone still holds can't go (RoleEditorStates, "Can't delete").
  if (assigned > 0)
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{tr(m.deleteEyebrow)}</span>}
        title={tr(m.cantDeleteTitle, { name: role.name })}
        hideClose
        footer={
          <Button variant="primary" onClick={onClose}>
            {tr(m.close)}
          </Button>
        }
      >
        <Note tone="bad" label={tr(m.cantDeleteLabel)}>
          {tr(m.cantDeleteBody, { name: role.name, count: assigned, names: role.members.map((x) => x.name).join(', ') })}
        </Note>
      </Dialog>
    )
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={tr(m.deleteBadge)}
      title={tr(m.deleteTitle, { name: role.name })}
      description={tr(m.deleteBody)}
      confirmLabel={tr(m.deleteConfirm)}
      busyLabel={tr(m.deleteBusy)}
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/roles/{roleId}', { params: { path: { workspace: slug, roleId: role.id } } })
        toast({ kind: 'deleted', title: tr(m.deletedToast, { name: role.name }), detail: tr(m.deletedDetail) })
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
      }}
    />
  )
}

function CompareCard({ list }: { list: RoleList }) {
  const tr = useT()
  const [hidden, setHidden] = useState<string[]>([])
  const shown = list.roles.filter((r) => !hidden.includes(r.id))
  const perms = list.permissions.filter((p) => !p.ownerOnly)
  const has = (r: Role, key: string) => r.permissions.includes(key) || implied(list.permissions, r.permissions, key)
  return (
    <section className="card" aria-labelledby="cmp-h">
      <CardSection
        title={<span id="cmp-h">{tr(m.compareTitle)}</span>}
        caption={tr(m.compareCaption)}
        actions={
          <div className="row gap-2" role="group" aria-label={tr(m.columnsLabel)}>
            <span className="k" aria-hidden="true">
              {tr(m.columns)}
            </span>
            {list.roles.map((r) => {
              const on = !hidden.includes(r.id)
              return (
                <button
                  key={r.id}
                  type="button"
                  className={on ? 'chip box-content border-line-strong! bg-white! text-ink! hover:bg-ground!' : 'chip box-content border-dashed! text-ink-3! line-through'}
                  aria-pressed={on}
                  title={on ? tr(m.hideColumn, { name: r.name }) : tr(m.showColumn, { name: r.name })}
                  onClick={() => setHidden((cur) => (on ? (shown.length > 1 ? [...cur, r.id] : cur) : cur.filter((x) => x !== r.id)))}
                >
                  {r.name}
                </button>
              )
            })}
          </div>
        }
      />
      <AccessTable caption={tr(m.compareTitle)}>
        <thead>
          <tr>
            <Th>{tr(m.colPermission)}</Th>
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
                {rows.map((p) => {
                  const note = matrixNote(p)
                  return (
                    <tr key={p.key}>
                      <Td>
                        {p.label} <span className="k mono">{p.key}</span>
                        {p.sensitive ? (
                          <>
                            {' '}
                            <span title={tr(m.sensitiveTitle)}>
                              <SensitiveFlag>{tr(m.sensitive)}</SensitiveFlag>
                            </span>
                          </>
                        ) : null}
                        {note.suffix ? (
                          <>
                            {' '}
                            <span className="k">{tr(note.suffix)}</span>
                          </>
                        ) : null}
                        {note.line ? <div className="k mt-0.5 whitespace-normal">{tr(note.line)}</div> : null}
                      </Td>
                      {shown.map((r) => (
                        <Td key={r.id} align="center">
                          <HasMark on={has(r, p.key)} />
                        </Td>
                      ))}
                    </tr>
                  )
                })}
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
  const tr = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        {tr(m.loading)}
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
