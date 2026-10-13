import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { deniedReason, useWorkspace } from '@/lib/session'
import { SaveRoleDialog, type SaveKind } from './SaveRoleDialog'

type RoleList = Schemas['RoleList']
type Role = Schemas['RoleSummary']
type Permission = Schemas['PermissionInfo']
export type Impact = Schemas['RoleImpact']

export interface RoleEditorSearch {
  from?: string
  saved?: 'created' | 'updated'
}

/** A5 §5.8 role editor (RoleEditor, RoleEditorStates): "new", a duplicate (?from=) or an existing custom role. */
export function RoleEditorPage({ roleId, search }: { roleId: string; search: RoleEditorSearch }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/roles', { params: { path: { workspace: slug } } })
  const isNew = roleId === 'new'
  const role = isNew ? undefined : q.data?.roles.find((r) => r.id === roleId)
  const source = isNew && search.from ? q.data?.roles.find((r) => r.id === search.from) : undefined
  const title = isNew ? 'New custom role' : (role?.name ?? 'Role')
  return (
    <Page
      title={title}
      breadcrumb={[
        { label: 'Settings', to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: 'Roles', to: '/w/$workspace/settings/roles', params: { workspace: slug } },
        { label: title },
      ]}
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the role" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <EditorSkeleton />
      ) : !isNew && !role ? (
        <FullPageState
          icon="searchLarge"
          badge="Not found"
          title="This role doesn't exist"
          actions={
            <ButtonLink to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
              Back to Roles
            </ButtonLink>
          }
        >
          It may have been deleted. The audit log keeps its history.
        </FullPageState>
      ) : (
        <RoleEditor key={`${roleId}:${search.from ?? ''}`} list={q.data} role={role} source={source} saved={search.saved} />
      )}
    </Page>
  )
}

/** Explicit picks plus everything they include. */
function closure(catalog: Permission[], picked: string[]): string[] {
  const out = new Set(picked)
  for (const p of catalog) if (out.has(p.key)) p.implies.forEach((k) => out.add(k))
  return catalog.filter((p) => out.has(p.key)).map((p) => p.key)
}

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const h = window.setTimeout(() => setV(value), ms)
    return () => window.clearTimeout(h)
  }, [value, ms])
  return v
}

function RoleEditor({ list, role, source, saved }: { list: RoleList; role?: Role; source?: Role; saved?: RoleEditorSearch['saved'] }) {
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const catalog = useMemo(() => list.permissions.filter((p) => !p.ownerOnly), [list.permissions])
  const builtin = Boolean(role?.builtin)
  const readOnly = !list.canManage || builtin
  const [name, setName] = useState(role?.name ?? (source ? `${source.name} (copy)` : ''))
  const [description, setDescription] = useState(role?.description ?? source?.description ?? '')
  // A duplicate keeps only what the caller can grant; an existing role keeps what it has.
  const [picked, setPicked] = useState<string[]>(() =>
    (role ?? source)?.permissions.filter((k) => catalog.some((p) => p.key === k) && (role ? true : list.held.includes(k))) ?? [],
  )
  const [error, setError] = useState<{ field?: 'name'; message: string }>()
  const [dialog, setDialog] = useState<SaveKind>()
  const effective = useMemo(() => closure(catalog, picked), [catalog, picked])
  const includedBy = (key: string) => catalog.find((p) => p.key !== key && picked.includes(p.key) && p.implies.includes(key))?.key
  const lockedKeys = catalog.filter((p) => !list.held.includes(p.key)).map((p) => p.key)

  const wanted = effective.join(',')
  const debounced = useDebounced(wanted, 250)
  const impactQuery = (perms: string) => ({
    queryKey: ['role-impact', slug, role?.id ?? 'new', perms],
    queryFn: async () =>
      (
        await http.POST('/api/v1/workspaces/{workspace}/roles/impact', {
          params: { path: { workspace: slug } },
          body: { roleId: role?.id, permissions: perms ? perms.split(',') : [] },
        })
      ).data!,
    staleTime: 15_000,
  })
  const impact = useQuery({ ...impactQuery(debounced), enabled: !readOnly, placeholderData: keepPreviousData })
  const recalculating = !readOnly && (impact.isFetching || debounced !== wanted)
  const [waiting, setWaiting] = useState(false)
  const lockReason = (key: string) =>
    impact.data?.locked.find((l) => l.key === key)?.reason ?? `You can't grant ${key} because you don't hold it.${list.helpers.length ? ` Ask ${list.helpers[0]}.` : ''}`

  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/roles')
  const update = $api.useMutation('put', '/api/v1/workspaces/{workspace}/roles/{roleId}')
  const busy = create.isPending || update.isPending

  const fail = (e: unknown) => {
    const field = e instanceof ApiError && e.problem.errors?.some((x) => x.location === 'body.name') ? 'name' : undefined
    setError({ field, message: errorMessage(e) })
    setDialog(undefined)
  }
  const body = { name: name.trim(), description: description.trim() || undefined, permissions: picked }
  const save = () => {
    setError(undefined)
    if (!body.name) {
      setError({ field: 'name', message: 'Give the role a name.' })
      return
    }
    if (!role) {
      create.mutate(
        { params: { path: { workspace: slug } }, body },
        {
          onSuccess: (r) => {
            void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
            void navigate({ to: '/w/$workspace/settings/roles/$roleId', params: { workspace: slug, roleId: r.id }, search: { saved: 'created' }, replace: true })
          },
          onError: fail,
        },
      )
      return
    }
    update.mutate(
      { params: { path: { workspace: slug, roleId: role.id } }, body },
      {
        onSuccess: () => {
          setDialog(undefined)
          toast({ title: `Role “${body.name}” saved`, detail: 'logged as role.updated' })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
          void qc.invalidateQueries({ queryKey: ['role-impact'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          void navigate({ to: '/w/$workspace/settings/roles/$roleId', params: { workspace: slug, roleId: role.id }, search: { saved: 'updated' }, replace: true })
        },
        onError: fail,
      },
    )
  }
  /**
   * Changes that take access away, or add a sensitive permission to holders,
   * are confirmed first. Saving waits for the impact of the latest change.
   */
  const requestSave = async () => {
    if (!role) {
      save()
      return
    }
    setWaiting(true)
    try {
      const i = await qc.fetchQuery(impactQuery(wanted))
      if (i.removed.length > 0 && i.members.length + i.keys.length > 0) setDialog('remove')
      else if (i.addedSensitive.length > 0 && i.members.length > 0) setDialog('sensitive')
      else save()
    } catch (e) {
      fail(e)
    } finally {
      setWaiting(false)
    }
  }
  const dirty =
    name !== (role?.name ?? '') ||
    description !== (role?.description ?? '') ||
    picked.length !== (role?.permissions.length ?? 0) ||
    picked.some((k) => !role?.permissions.includes(k))

  const held = list.held.length
  const ownerNote = me.workspace.roleKey === 'owner' ? 'As Owner you hold all of them.' : lockedKeys.length === 0 ? 'You hold all of them.' : ''
  const groups = list.groups.map((g) => ({ g, rows: catalog.filter((p) => p.group === g) })).filter((x) => x.rows.length > 0)

  return (
    <div className="grid grid-cols-[minmax(0,1fr)_320px] items-start gap-5 max-[1100px]:grid-cols-1">
      <div className="col gap-5">
        {saved && role && !dirty ? (
          <Note tone="ok" label="Saved">
            Role “{role.name}” {saved === 'created' ? 'created' : 'saved'}.{' '}
            {saved === 'created' ? (
              <>
                Assign it from{' '}
                <Link to="/w/$workspace/settings/team" params={{ workspace: slug }}>
                  Team
                </Link>
                .{' '}
              </>
            ) : null}
            Logged as <span className="mono">{saved === 'created' ? 'role.created' : 'role.updated'}</span>.
          </Note>
        ) : null}
        <section className="card" aria-labelledby="role-h">
          <div className="sec">
            <h2 className="h2" id="role-h">
              Role
            </h2>
            {builtin ? <Badge tone="neutral">Built-in · read-only</Badge> : <Badge tone="info">Custom role</Badge>}
          </div>
          <div className="pad col gap-3.5">
            <div className="grid grid-cols-[1fr_2fr] gap-3.5 max-md:grid-cols-1">
              <label className="lbl">
                Name
                <input
                  className="in"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={readOnly}
                  maxLength={60}
                  aria-invalid={error?.field === 'name' || undefined}
                  aria-describedby={error?.field === 'name' ? 'role-name-err' : undefined}
                />
                {error?.field === 'name' ? (
                  <span className="err" id="role-name-err" role="alert">
                    {error.message}
                  </span>
                ) : null}
              </label>
              <label className="lbl">
                Description
                <input className="in" type="text" value={description} onChange={(e) => setDescription(e.target.value)} disabled={readOnly} maxLength={300} />
                <span className="k font-medium">Shown to members and in the audit log.</span>
              </label>
            </div>
          </div>
        </section>
        <section className="card" aria-labelledby="perm-h">
          <div className="sec">
            <div className="col gap-0.5">
              <h2 className="h2" id="perm-h">
                Permissions
              </h2>
              <span className="k">A permission includes its view permission automatically. ⚑ marks sensitive permissions: holders must use two-factor.</span>
            </div>
            {readOnly ? null : (
              <div className="row gap-2">
                {lockedKeys.length > 0 ? (
                  <Badge tone="warn">
                    {lockedKeys.length} {lockedKeys.length === 1 ? 'permission' : 'permissions'} locked
                  </Badge>
                ) : null}
                <Button
                  size="sm"
                  onClick={() => {
                    const editor = list.roles.find((r) => r.builtin && r.key === 'editor')
                    if (editor) setPicked(editor.permissions.filter((k) => list.held.includes(k) && catalog.some((p) => p.key === k)))
                  }}
                >
                  Start from Editor
                </Button>
                <Button size="sm" onClick={() => setPicked([])}>
                  Clear all
                </Button>
              </div>
            )}
          </div>
          {!readOnly && lockedKeys.length > 0 ? (
            <Note tone="neutral" className="mx-4.5 mb-3 mt-3.5">
              Nobody can grant a permission they don&apos;t hold: not to a role, a member, an API key or an MCP agent. Locked rows explain why and who can help.
            </Note>
          ) : null}
          {groups.map(({ g, rows }) => {
            const gid = `g-${g.toLowerCase().replace(/[^a-z]+/g, '-')}`
            return (
              <div key={g} role="group" aria-labelledby={gid}>
                <div id={gid} className="border-t border-line-2 px-4.5 pb-1 pt-2.5 text-[12px] font-bold uppercase tracking-[.04em] text-ink-2">
                  {g}
                </div>
                {rows.map((p) => {
                  const by = includedBy(p.key)
                  const on = effective.includes(p.key)
                  const locked = lockedKeys.includes(p.key) && !readOnly
                  const disabled = readOnly || Boolean(by) || (locked && !picked.includes(p.key))
                  const id = `p-${p.key.replace(':', '-')}`
                  const note = by ? (
                    <span className="k" id={`d-${id}`}>
                      included by {by}
                    </span>
                  ) : locked ? (
                    <span className="k max-w-[260px] whitespace-normal text-bad-fg!" id={`d-${id}`}>
                      {lockReason(p.key)}
                    </span>
                  ) : !readOnly && lockedKeys.length > 0 && on ? (
                    <span className="k" id={`d-${id}`}>
                      Held by you · can be granted
                    </span>
                  ) : null
                  return (
                    <div key={p.key} className="row flex-nowrap gap-3 px-4.5 max-sm:flex-wrap max-sm:gap-y-0">
                      <input
                        type="checkbox"
                        id={id}
                        checked={on}
                        disabled={disabled}
                        aria-describedby={note ? `d-${id}` : undefined}
                        onChange={(e) => setPicked((cur) => (e.target.checked ? [...cur, p.key] : cur.filter((k) => k !== p.key)))}
                      />
                      <label htmlFor={id} className={cn('flex min-w-0 flex-1 flex-col gap-px py-[7px] text-[13.5px]', !disabled && 'cursor-pointer')}>
                        <span>
                          {p.label}
                          {p.sensitive ? (
                            <>
                              {' '}
                              <span title="Sensitive permission">
                                <SensitiveFlag />
                              </span>
                            </>
                          ) : null}
                        </span>
                        <span className="k mono">{p.key}</span>
                      </label>
                      {note}
                    </div>
                  )
                })}
              </div>
            )
          })}
          <div className="h-2" />
        </section>
      </div>
      <section className="card min-[1101px]:sticky min-[1101px]:top-4" aria-labelledby="impact-h">
        <div className="sec">
          <h2 className="h2" id="impact-h">
            Impact preview
          </h2>
        </div>
        <div className={cn('pad col gap-3', recalculating && 'opacity-80')} aria-busy={recalculating || undefined}>
          <ImpactValues role={role} impact={impact.data} readOnly={readOnly} />
          {readOnly ? (
            <Note tone="neutral">
              {builtin
                ? 'Built-in roles can’t be edited. Duplicate this one to start a custom role with the same permissions.'
                : `${deniedReason('team:manage', list.helpers)}`}
            </Note>
          ) : (
            <Note tone="neutral">
              You can only grant permissions you hold yourself.{' '}
              {ownerNote ||
                `You hold ${held} of ${catalog.length}; ${lockedKeys.length} ${lockedKeys.length === 1 ? 'is' : 'are'} locked.${list.helpers.length ? ` Ask ${list.helpers.join(' or ')} for more.` : ''}`}
            </Note>
          )}
          {error ? (
            <Note tone="bad" label="Not saved">
              {error.message}
            </Note>
          ) : null}
          <div className="row justify-end gap-2">
            <ButtonLink to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
              {readOnly ? 'Back to roles' : 'Cancel'}
            </ButtonLink>
            {builtin && list.canManage && role ? (
              <ButtonLink variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }} search={{ from: role.id }}>
                Duplicate to customize
              </ButtonLink>
            ) : readOnly ? null : (
              <Button variant="primary" loading={busy || waiting} onClick={() => void requestSave()}>
                {busy || waiting ? 'Saving…' : role ? 'Save role' : 'Create role'}
              </Button>
            )}
          </div>
          {waiting ? <span className="k">Impact is recalculated after each change; saving waits for it.</span> : null}
        </div>
      </section>
      {dialog && role && impact.data ? (
        <SaveRoleDialog kind={dialog} role={role} name={body.name} impact={impact.data} busy={busy} onBack={() => setDialog(undefined)} onSave={save} />
      ) : null}
    </div>
  )
}

function ImpactValues({ role, impact, readOnly }: { role?: Role; impact?: Impact; readOnly: boolean }) {
  const holders = role ? role.members.filter((m) => m.status === 'active') : []
  const members = !role
    ? '0 · assign the role from Team after saving'
    : holders.length === 0
      ? '0'
      : `${holders.length} · ${holders
          .slice(0, 3)
          .map((m) => m.name)
          .join(', ')}${holders.length > 3 ? ` +${holders.length - 3}` : ''}`
  if (readOnly || !impact) {
    return (
      <KeyValues
        items={[
          ['Members', members],
          ['API keys', readOnly ? '–' : <Skeleton key="k" w={24} />],
          ['MCP agents', readOnly ? '–' : <Skeleton key="a" w={24} />],
          ['Sensitive', readOnly ? '–' : <Skeleton key="s" w="80%" />],
          ['Approvals', readOnly ? '–' : <Skeleton key="p" w="90%" />],
        ]}
      />
    )
  }
  return (
    <KeyValues
      items={[
        ['Members', members],
        ['API keys', String(impact.apiKeys)],
        ['MCP agents', String(impact.agents)],
        ['Sensitive', impact.sensitive],
        ['Approvals', impact.approvals],
      ]}
    />
  )
}

function EditorSkeleton() {
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_320px] items-start gap-5 max-[1100px]:grid-cols-1" aria-busy="true">
      <span className="vh" role="status">
        Loading the role
      </span>
      <div className="col gap-5">
        <section className="card pad flex flex-col gap-3">
          <Skeleton w={60} h={14} />
          <Skeleton h={36} />
        </section>
        <section className="card pad flex flex-col gap-4">
          {Array.from({ length: 10 }, (_, i) => (
            <div key={i} className="flex items-center gap-3">
              <Skeleton w={16} h={16} r={4} />
              <div className="flex flex-col gap-1">
                <Skeleton w={160} h={12} />
                <Skeleton w={110} h={10} />
              </div>
            </div>
          ))}
        </section>
      </div>
      <section className="card pad flex flex-col gap-3">
        <Skeleton w={120} h={14} />
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} w={`${85 - i * 8}%`} h={12} />
        ))}
      </section>
    </div>
  )
}
