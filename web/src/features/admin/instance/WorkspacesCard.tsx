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
import { andList, count, usDate } from '@/lib/format'
import { meQueryKey, useMe } from '@/lib/session'

type Workspace = Schemas['InstanceWorkspace']

/** Instance admin › Workspaces (InstanceSettings): every workspace, create, delete with a restore hold, restore. */
export function WorkspacesCard() {
  const q = $api.useQuery('get', '/api/v1/admin/workspaces')
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<Workspace>()
  const live = q.data?.live
  return (
    <section className="card" aria-labelledby="ws-h">
      <CardSection
        title={<span id="ws-h">Workspaces{live !== undefined ? ` · ${live}` : ''}</span>}
        caption="Unlimited workspaces. Each has its own lists, campaigns, providers and team."
        actions={
          <Button size="sm" variant="primary" onClick={() => setCreating(true)}>
            Create workspace
          </Button>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the workspaces" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Nothing was changed.
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3" aria-busy="true">
          <span className="vh" role="status">
            Loading workspaces
          </span>
          <Skeleton w="100%" h={20} />
          <Skeleton w="100%" h={20} />
        </div>
      ) : q.data.items.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>No workspaces yet. Create the first one; you become its Owner and can invite the team from there.</span>
          </div>
        </div>
      ) : (
        <AccessTable caption="Workspaces">
          <thead>
            <tr>
              <Th>Workspace</Th>
              <Th>Members</Th>
              <Th>Subscribers</Th>
              <Th>Providers</Th>
              <Th>Created</Th>
              <Th>
                <span className="vh">Actions</span>
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
  const qc = useQueryClient()
  const restore = $api.useMutation('post', '/api/v1/admin/workspaces/{workspaceId}/restore')
  if (w.deletedAt) {
    return (
      <tr className="text-ink-3">
        <Td>
          <span className="col gap-0.5">
            <span className="font-semibold">{w.name}</span>
            <span className="k">Deleted {usDate(w.deletedAt)} · gone for good on {w.purgeAt ? usDate(w.purgeAt) : 'the 7th day'}</span>
          </span>
        </Td>
        <Td>{w.members}</Td>
        <Td>{count(w.subscribers)}</Td>
        <Td>{w.providers.join(', ') || '–'}</Td>
        <Td>{usDate(w.createdAt)}</Td>
        <Td>
          <div className="row gap-1.5">
            <Badge tone="warn">Deleted</Badge>
            <Button
              size="sm"
              loading={restore.isPending}
              onClick={async () => {
                try {
                  const r = await restore.mutateAsync({ params: { path: { workspaceId: w.id } } })
                  await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                  toast({ kind: 'success', title: `${w.name} restored`, detail: `at /w/${r.slug}` })
                } catch (e) {
                  toast({ kind: 'error', title: `Couldn't restore ${w.name}`, detail: errorMessage(e) })
                }
              }}
            >
              Restore
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
      <Td>
        {w.members}
        {w.invited ? ` · ${w.invited} invited` : ''}
      </Td>
      <Td>{count(w.subscribers)}</Td>
      <Td>{w.providers.join(', ') || <span className="k">None yet</span>}</Td>
      <Td>{usDate(w.createdAt)}</Td>
      <Td>
        <div className="row gap-1.5">
          {w.member ? (
            <ButtonLink size="sm" to="/w/$workspace/settings" params={{ workspace: w.slug }}>
              Open settings
            </ButtonLink>
          ) : (
            <Button size="sm" disabledReason={`You aren't a member of ${w.name}. Ask its Owner${w.owner ? `, ${w.owner},` : ''} to invite you.`}>
              Open settings
            </Button>
          )}
          <Button size="sm" variant="danger" onClick={onDelete}>
            Delete
          </Button>
        </div>
      </Td>
    </tr>
  )
}

function CreateWorkspaceDialog({ onClose }: { onClose: () => void }) {
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
      eyebrow={<span className="k">Dialog · instance admin</span>}
      title="Create a workspace"
      description={<span className="text-[13.5px]">A workspace has its own lists, campaigns, providers and team. The Owner can invite everyone else.</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={create.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            loading={create.isPending}
            disabledReason={name.trim() ? undefined : 'Name the workspace first'}
            onClick={async () => {
              setError(undefined)
              try {
                const w = await create.mutateAsync({ body: { name: name.trim(), ownerId: owner || undefined, timezone: zone } })
                await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                toast({ kind: 'success', title: `${w.name} created`, detail: `at /w/${w.slug}` })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            Create workspace
          </Button>
        </>
      }
    >
      <label className="lbl">
        Workspace name
        <input className="in" value={name} maxLength={80} placeholder="Acme Engineering Blog" onChange={(e) => setName(e.target.value)} autoFocus />
      </label>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          Owner
          <select className="in" value={owner} onChange={(e) => setOwner(e.target.value)}>
            {(users.data?.items ?? []).map((u) => (
              <option key={u.id} value={u.id}>
                {u.name}
                {u.self ? ' (you)' : ''}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          Timezone
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
  const qc = useQueryClient()
  const del = $api.useMutation('post', '/api/v1/admin/workspaces/{workspaceId}/delete')
  const [typed, setTyped] = useState('')
  const match = typed.trim() === w.name
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !del.isPending && onClose()}
      eyebrow={<span className="k">Dialog · instance admin · typed confirmation</span>}
      title={<span className="text-bad-fg">Delete {w.name}?</span>}
      description={
        <span className="text-[13.5px]">
          This permanently deletes the workspace after a 7-day hold during which an instance admin can restore it. Scheduled sends stop immediately.
        </span>
      }
      className="card border-bad-line"
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={onClose} disabled={del.isPending}>
            Cancel
          </Button>
          <Button
            variant="danger-primary"
            loading={del.isPending}
            disabledReason={match ? undefined : 'Type the workspace name to enable'}
            onClick={async () => {
              try {
                const r = await del.mutateAsync({ params: { path: { workspaceId: w.id } }, body: { confirm: w.name } })
                await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/workspaces'] }), qc.invalidateQueries({ queryKey: meQueryKey })])
                toast({ kind: 'deleted', title: `${w.name} deleted`, detail: `restorable until ${usDate(r.purgeAt)}` })
                onClose()
              } catch (e) {
                toast({ kind: 'error', title: "Couldn't delete the workspace", detail: errorMessage(e) })
              }
            }}
          >
            Delete workspace
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{count(w.subscribers)} contacts with their lists, segments and consent history</li>
        <li>Campaigns with reports, automations and templates</li>
        <li>
          {w.providers.length
            ? `${andList(w.providers)}: routing rules and warm-up plans (provider accounts themselves are untouched)`
            : 'Routing rules and warm-up plans'}
        </li>
        <li>API keys and MCP agents of this workspace stop working</li>
        <li>
          <b>Kept:</b> the shared suppression list, the audit log and the members&apos; accounts ({w.members} {w.members === 1 ? 'member' : 'members'})
        </li>
      </ul>
      <div className="note i">
        <Badge tone="info">Recommended</Badge>
        <span className="col gap-1.5">
          <span>Export everything first: contacts with consent, campaigns as MJML, reports as CSV, audit log as JSON.</span>
          <span>
            {w.member ? (
              <ButtonAnchor size="sm" href={`/api/v1/workspaces/${w.slug}/export`} newTab={false} download="">
                Export everything first
              </ButtonAnchor>
            ) : (
              <Button size="sm" disabledReason={`Exports come from inside ${w.name}: ask its Owner${w.owner ? `, ${w.owner},` : ''} to export first.`}>
                Export everything first
              </Button>
            )}
          </span>
        </span>
      </div>
      <label className="lbl">
        <span>
          Type <b>{w.name}</b> to confirm
        </span>
        <input className="in" type="text" value={typed} onChange={(e) => setTyped(e.target.value)} autoComplete="off" spellCheck={false} />
        <span className="k font-medium">Must match exactly.</span>
      </label>
    </Dialog>
  )
}
