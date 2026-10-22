import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable } from '@/features/access/parts'
import { timezoneOptions } from '@/features/workspace/timezones'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count, usDate } from '@/lib/format'
import { meQueryKey, useMe } from '@/lib/session'
import { m } from './messages'

type Workspace = Schemas['InstanceWorkspace']

/** The hold before a deleted workspace is gone for good. */
const HOLD_DAYS = 7

/** Instance admin › Workspaces (InstanceSettings): every workspace, create, delete with a restore hold, restore. */
export function WorkspacesCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/workspaces')
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<Workspace>()
  const live = q.data?.live
  return (
    <section className="card" aria-labelledby="ws-h">
      <CardSection
        title={<span id="ws-h">{live !== undefined ? t(m.workspacesTitleCount, { count: count(live) }) : t(m.workspacesTitle)}</span>}
        caption={t(m.workspacesCaption)}
        actions={
          <Button size="sm" variant="primary" onClick={() => setCreating(true)}>
            {t(m.createWorkspace)}
          </Button>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.workspacesLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(m.nothingChanged)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3" aria-busy="true">
          <span className="vh" role="status">
            {t(m.workspacesLoading)}
          </span>
          <Skeleton w="100%" h={20} />
          <Skeleton w="100%" h={20} />
        </div>
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>{t(m.workspacesEmpty)}</span>
          </div>
        </div>
      ) : (
        <AccessTable caption={t(m.workspacesTable)}>
          <thead>
            <tr>
              <Th>{t(m.colWorkspace)}</Th>
              <Th>{t(m.colMembers)}</Th>
              <Th>{t(m.colSubscribers)}</Th>
              <Th>{t(m.colProviders)}</Th>
              <Th>{t(m.colCreated)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.map((w) => (
              <WorkspaceRow key={w.id} w={w} onDelete={() => setDeleting(w)} />
            ))}
          </tbody>
        </AccessTable>
      )}
      {creating ? <CreateWorkspaceDialog onClose={() => setCreating(false)} /> : null}
      {deleting ? <DeleteWorkspaceDialog w={deleting} onClose={() => setDeleting(undefined)} /> : null}
    </section>
  )
}

function WorkspaceRow({ w, onDelete }: { w: Workspace; onDelete: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const restore = $api.useMutation('post', '/api/v1/admin/workspaces/{workspaceId}/restore')
  if (w.deletedAt) {
    return (
      <tr className="text-ink-3">
        <Td>
          <span className="col gap-0.5">
            <span className="font-semibold">{w.name}</span>
            <span className="k">
              {t(m.deletedLine, { date: usDate(w.deletedAt), hasPurge: w.purgeAt ? 'yes' : 'no', purge: w.purgeAt ? usDate(w.purgeAt) : '' })}
            </span>
          </span>
        </Td>
        <Td>{w.members}</Td>
        <Td>{count(w.subscribers)}</Td>
        <Td>{w.providers.join(', ') || '–'}</Td>
        <Td>{usDate(w.createdAt)}</Td>
        <Td>
          <div className="row gap-1.5">
            <Badge tone="warn">{t(m.deletedBadge)}</Badge>
            <Button
              size="sm"
              loading={restore.isPending}
              onClick={async () => {
                try {
                  const r = await restore.mutateAsync({ params: { path: { workspaceId: w.id } } })
                  await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                  toast({ kind: 'success', title: t(m.restored, { name: w.name }), detail: t(m.atPath, { path: `/w/${r.slug}` }) })
                } catch (e) {
                  toast({ kind: 'error', title: t(m.restoreFailed, { name: w.name }), detail: errorMessage(e) })
                }
              }}
            >
              {t(m.restore)}
            </Button>
          </div>
        </Td>
      </tr>
    )
  }
  return (
    <tr>
      <Td>
        <span className="font-semibold">{w.name}</span>
      </Td>
      <Td>{w.invited ? t(m.membersInvited, { members: w.members, invited: w.invited }) : w.members}</Td>
      <Td>{count(w.subscribers)}</Td>
      <Td>{w.providers.join(', ') || <span className="k">{t(m.noProviders)}</span>}</Td>
      <Td>{usDate(w.createdAt)}</Td>
      <Td>
        <div className="row gap-1.5">
          {w.member ? (
            <ButtonLink size="sm" to="/w/$workspace/settings" params={{ workspace: w.slug }}>
              {t(m.openSettings)}
            </ButtonLink>
          ) : (
            <Button size="sm" disabledReason={t(m.notMember, { name: w.name, hasOwner: w.owner ? 'yes' : 'no', owner: w.owner ?? '' })}>
              {t(m.openSettings)}
            </Button>
          )}
          <Button size="sm" variant="danger" onClick={onDelete}>
            {t(m.delete)}
          </Button>
        </div>
      </Td>
    </tr>
  )
}

function CreateWorkspaceDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const { data: me } = useMe()
  const users = $api.useQuery('get', '/api/v1/admin/users', { params: { query: { status: 'active' } } })
  const settings = $api.useQuery('get', '/api/v1/admin/instance')
  const create = $api.useMutation('post', '/api/v1/admin/workspaces')
  const [name, setName] = useState('')
  const [owner, setOwner] = useState(me?.user.id ?? '')
  const [tz, setTz] = useState<string>()
  const zone = tz ?? settings.data?.defaultTimezone ?? 'UTC'
  const zones = useMemo(() => timezoneOptions(zone), [zone])
  const [error, setError] = useState<string>()
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !create.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(m.dialogEyebrow)}</span>}
      title={t(m.createTitle)}
      description={<span className="text-[13.5px]">{t(m.createDescription)}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={create.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={create.isPending}
            disabledReason={name.trim() ? undefined : t(m.nameFirst)}
            onClick={async () => {
              setError(undefined)
              try {
                const w = await create.mutateAsync({ body: { name: name.trim(), ownerId: owner || undefined, timezone: zone } })
                await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                toast({ kind: 'success', title: t(m.created, { name: w.name }), detail: t(m.atPath, { path: `/w/${w.slug}` }) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            {t(m.createWorkspace)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(m.workspaceName)}
        <input className="in" value={name} maxLength={80} placeholder={t(m.workspaceNamePlaceholder)} onChange={(e) => setName(e.target.value)} autoFocus />
      </label>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          {t(m.owner)}
          <select className="in" value={owner} onChange={(e) => setOwner(e.target.value)}>
            {(users.data?.items ?? []).map((u) => (
              <option key={u.id} value={u.id}>
                {u.self ? t(m.ownerYou, { name: u.name }) : u.name}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          {t(m.timezone)}
          <select className="in" value={zone} onChange={(e) => setTz(e.target.value)}>
            {zones.map((z) => (
              <option key={z.value} value={z.value}>
                {z.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </Dialog>
  )
}

/**
 * Delete a workspace as an instance admin, with the WorkspaceDialogs delete
 * dialog, whether or not the admin belongs to the workspace.
 */
function DeleteWorkspaceDialog({ w, onClose }: { w: Workspace; onClose: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const del = $api.useMutation('post', '/api/v1/admin/workspaces/{workspaceId}/delete')
  const [typed, setTyped] = useState('')
  const match = typed.trim() === w.name
  const owner = { name: w.name, hasOwner: w.owner ? 'yes' : 'no', owner: w.owner ?? '' }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !del.isPending && onClose()}
      eyebrow={<span className="k">{t(m.deleteEyebrow)}</span>}
      title={<span className="text-bad-fg">{t(m.deleteTitle, { name: w.name })}</span>}
      description={<span className="text-[13.5px]">{t(m.deleteDescription, { days: HOLD_DAYS })}</span>}
      className="card border-bad-line"
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={onClose} disabled={del.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="danger-primary"
            loading={del.isPending}
            disabledReason={match ? undefined : t(m.typeToEnable)}
            onClick={async () => {
              try {
                const r = await del.mutateAsync({ params: { path: { workspaceId: w.id } }, body: { confirm: w.name } })
                await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                toast({ kind: 'deleted', title: t(m.deleted, { name: w.name }), detail: t(m.restorableUntil, { date: usDate(r.purgeAt) }) })
                onClose()
              } catch (e) {
                toast({ kind: 'error', title: t(m.deleteFailed), detail: errorMessage(e) })
              }
            }}
          >
            {t(m.deleteWorkspace)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(m.deleteContacts, { count: w.subscribers, contacts: count(w.subscribers) })}</li>
        <li>{t(m.deleteCampaigns)}</li>
        <li>{w.providers.length ? t(m.deleteProviders, { providers: andList(w.providers) }) : t(m.deleteRouting)}</li>
        <li>{t(m.deleteKeys)}</li>
        <li>
          <T msg={m.deleteKept} values={{ count: w.members, b: (c) => <b>{c}</b> }} />
        </li>
      </ul>
      <div className="note i">
        <Badge tone="info">{t(m.recommended)}</Badge>
        <span className="col gap-1.5">
          <span>{t(m.exportBody)}</span>
          <span>
            {w.member ? (
              <ButtonAnchor size="sm" href={`/api/v1/workspaces/${w.slug}/export`} newTab={false} download="">
                {t(m.exportButton)}
              </ButtonAnchor>
            ) : (
              <Button size="sm" disabledReason={t(m.exportNotMember, owner)}>
                {t(m.exportButton)}
              </Button>
            )}
          </span>
        </span>
      </div>
      <label className="lbl">
        <span>
          <T msg={m.typeName} values={{ name: w.name, b: (c) => <b>{c}</b> }} />
        </span>
        <input className="in" type="text" value={typed} onChange={(e) => setTyped(e.target.value)} autoComplete="off" spellCheck={false} />
        <span className="k font-medium">{t(m.mustMatch)}</span>
      </label>
    </Dialog>
  )
}
