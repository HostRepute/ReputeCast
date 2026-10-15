import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { flushSync } from 'react-dom'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { common } from '@/i18n/common'
import { formatList, T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { deniedReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { editorNote, groupNote } from './notes'
import { SaveRoleDialog, type SaveKind } from './SaveRoleDialog'

type RoleList = Schemas['RoleList']
type Role = Schemas['RoleSummary']
type Permission = Schemas['PermissionInfo']
export type Impact = Schemas['RoleImpact']

export interface RoleEditorSearch {
  from?: string
  saved?: 'created' | 'updated'
}

const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>

/** A5 §5.8 role editor (RoleEditor, RoleEditorStates): "new", a duplicate (?from=) or an existing custom role. */
export function RoleEditorPage({ roleId, search }: { roleId: string; search: RoleEditorSearch }) {
  const { slug } = useWorkspace()
  const tr = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/roles', { params: { path: { workspace: slug } } })
  const isNew = roleId === 'new'
  const role = isNew ? undefined : q.data?.roles.find((r) => r.id === roleId)
  const source = isNew && search.from ? q.data?.roles.find((r) => r.id === search.from) : undefined
  const title = isNew ? tr(m.editorNewTitle) : (role?.name ?? tr(m.editorRoleTitle))
  return (
    <Page
      title={title}
      breadcrumb={[
        { label: tr(m.breadcrumbSettings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: tr(m.breadcrumbRoles), to: '/w/$workspace/settings/roles', params: { workspace: slug } },
        { label: title },
      ]}
    >
      {q.isError ? (
        <ErrorPanel title={tr(m.editorLoadFailed)} actions={<Button onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <EditorSkeleton />
      ) : !isNew && !role ? (
        <FullPageState
          icon="searchLarge"
          badge={tr(m.notFoundBadge)}
          title={tr(m.notFoundTitle)}
          actions={
            <ButtonLink to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
              {tr(m.backToRoles)}
            </ButtonLink>
          }
        >
          {tr(m.notFoundBody)}
        </FullPageState>
      ) : (
        // A saved role remounts the editor with its new values (and a fresh unsaved-changes guard).
        <RoleEditor key={`${roleId}:${search.from ?? ''}:${role?.updatedAt ?? ''}`} list={q.data} role={role} source={source} saved={search.saved} />
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
  const tr = useT()
  const catalog = useMemo(() => list.permissions.filter((p) => !p.ownerOnly), [list.permissions])
  const builtin = Boolean(role?.builtin)
  const readOnly = !list.canManage || builtin
  const [name, setName] = useState(role?.name ?? (source ? tr(m.copyName, { name: source.name }) : ''))
  const [description, setDescription] = useState(role?.description ?? source?.description ?? '')
  // A duplicate keeps only what the caller can grant; an existing role keeps what it has.
  const [picked, setPicked] = useState<string[]>(() =>
    (role ?? source)?.permissions.filter((k) => catalog.some((p) => p.key === k) && (role ? true : list.held.includes(k))) ?? [],
  )
  const [error, setError] = useState<{ field?: 'name'; message: string }>()
  const [dialog, setDialog] = useState<SaveKind>()
  // Set right before the save navigates, so the unsaved-changes prompt stays out of the way.
  const [leaving, setLeaving] = useState(false)
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
    impact.data?.locked.find((l) => l.key === key)?.reason ?? (list.helpers.length ? tr(m.cantGrantAsk, { key, name: list.helpers[0]! }) : tr(m.cantGrant, { key }))

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
      setError({ field: 'name', message: tr(m.nameRequired) })
      return
    }
    if (!role) {
      create.mutate(
        { params: { path: { workspace: slug } }, body },
        {
          onSuccess: (r) => {
            void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
            flushSync(() => setLeaving(true))
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
          toast({ title: tr(m.savedToast, { name: body.name }), detail: tr(m.savedToastDetail) })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/roles'] })
          void qc.invalidateQueries({ queryKey: ['role-impact'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          flushSync(() => setLeaving(true))
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
  const grantNote =
    me.workspace.roleKey === 'owner'
      ? tr(m.grantNoteOwner)
      : lockedKeys.length === 0
        ? tr(m.grantNoteAll)
        : list.helpers.length
          ? tr(m.grantNoteSomeAsk, { held, total: catalog.length, locked: lockedKeys.length, names: formatList(list.helpers, 'disjunction') })
          : tr(m.grantNoteSome, { held, total: catalog.length, locked: lockedKeys.length })
  const groups = list.groups.map((g) => ({ g, rows: catalog.filter((p) => p.group === g) })).filter((x) => x.rows.length > 0)

  return (
    <div className="grid grid-cols-[minmax(0,1fr)_320px] items-start gap-5 max-md:grid-cols-1">
      <div className="col gap-5">
        {saved && role && !dirty ? (
          <Note tone="ok" label={tr(m.savedBadge)}>
            <T
              msg={saved === 'created' ? m.savedCreated : m.savedUpdated}
              values={{
                name: role.name,
                mono,
                team: (chunks) => (
                  <Link to="/w/$workspace/settings/team" params={{ workspace: slug }}>
                    {chunks}
                  </Link>
                ),
              }}
            />
          </Note>
        ) : null}
        <section className="card" aria-labelledby="role-h">
          <div className="sec">
            <h2 className="h2" id="role-h">
              {tr(m.roleCard)}
            </h2>
            {builtin ? <Badge tone="neutral">{tr(m.builtinBadge)}</Badge> : <Badge tone="info">{tr(m.customBadge)}</Badge>}
          </div>
          <div className="pad col gap-3.5">
            <div className="grid grid-cols-[1fr_2fr] gap-3.5 max-md:grid-cols-1 [&>*]:min-w-0">
              <label className="lbl">
                {tr(m.nameLabel)}
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
                {tr(m.descriptionLabel)}
                <input className="in" type="text" value={description} onChange={(e) => setDescription(e.target.value)} disabled={readOnly} maxLength={300} />
                <span className="k font-medium">{tr(m.descriptionHelp)}</span>
              </label>
            </div>
          </div>
        </section>
        <section className="card" aria-labelledby="perm-h">
          <div className="sec">
            <div className="col gap-0.5">
              <h2 className="h2" id="perm-h">
                {tr(m.permissionsTitle)}
              </h2>
              <span className="k">{tr(m.permissionsCaption)}</span>
            </div>
            {readOnly ? null : (
              <div className="row gap-2">
                {lockedKeys.length > 0 ? <Badge tone="warn">{tr(m.lockedBadge, { count: lockedKeys.length })}</Badge> : null}
                <Button
                  size="sm"
                  onClick={() => {
                    const editor = list.roles.find((r) => r.builtin && r.key === 'editor')
                    if (editor) setPicked(editor.permissions.filter((k) => list.held.includes(k) && catalog.some((p) => p.key === k)))
                  }}
                >
                  {tr(m.startFromEditor)}
                </Button>
                <Button size="sm" onClick={() => setPicked([])}>
                  {tr(m.clearAll)}
                </Button>
              </div>
            )}
          </div>
          {!readOnly && lockedKeys.length > 0 ? (
            <Note tone="neutral" className="mx-4.5 mb-3 mt-3.5">
              {tr(m.guardNote)}
            </Note>
          ) : null}
          {groups.map(({ g, rows }) => {
            const gid = `g-${g.toLowerCase().replace(/[^a-z]+/g, '-')}`
            const after = groupNote(g)
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
                  const own = editorNote(p)
                  const note = by ? (
                    <span className="k" id={`d-${id}`}>
                      {tr(m.includedBy, { key: by })}
                    </span>
                  ) : locked ? (
                    <span className="k max-w-[260px] whitespace-normal text-bad-fg!" id={`d-${id}`}>
                      {lockReason(p.key)}
                    </span>
                  ) : own.aside ? (
                    <span className="k" id={`d-${id}`}>
                      {tr(own.aside)}
                    </span>
                  ) : !readOnly && lockedKeys.length > 0 && on ? (
                    <span className="k" id={`d-${id}`}>
                      {tr(m.heldByYou)}
                    </span>
                  ) : null
                  const describedBy = [own.line ? `l-${id}` : '', note ? `d-${id}` : ''].filter(Boolean).join(' ')
                  return (
                    <div key={p.key} className="row flex-nowrap gap-3 px-4.5">
                      <input
                        type="checkbox"
                        id={id}
                        checked={on}
                        disabled={disabled}
                        aria-describedby={describedBy || undefined}
                        onChange={(e) => setPicked((cur) => (e.target.checked ? [...cur, p.key] : cur.filter((k) => k !== p.key)))}
                      />
                      <label htmlFor={id} className={cn('flex min-w-min flex-1 flex-col gap-px py-[7px] text-[13.5px]', !disabled && 'cursor-pointer')}>
                        <span>
                          {p.label}
                          {p.sensitive ? (
                            <>
                              {' '}
                              <span title={tr(m.sensitiveTitle)}>
                                <SensitiveFlag>{tr(m.sensitive)}</SensitiveFlag>
                              </span>
                            </>
                          ) : null}
                        </span>
                        {/* A key never breaks; the note beside it wraps instead. */}
                        <span className="k mono [overflow-wrap:normal]!">{p.key}</span>
                        {own.line ? (
                          <span className="k whitespace-normal" id={`l-${id}`}>
                            {tr(own.line)}
                          </span>
                        ) : null}
                      </label>
                      {note}
                    </div>
                  )
                })}
                {after ? (
                  <div className="k whitespace-normal px-4.5 pb-2.5 pt-0.5">
                    <T msg={after} values={{ mono }} />
                  </div>
                ) : null}
              </div>
            )
          })}
        </section>
      </div>
      <section className="card md:sticky md:top-4" aria-labelledby="impact-h">
        <div className="sec">
          <h2 className="h2" id="impact-h">
            {tr(m.impactTitle)}
          </h2>
        </div>
        <div className={cn('pad col gap-3', recalculating && 'opacity-80')} aria-busy={recalculating || undefined}>
          <ImpactValues role={role} impact={impact.data} readOnly={readOnly} />
          {readOnly ? <Note tone="neutral">{builtin ? tr(m.builtinReadOnly) : deniedReason('team:manage', list.helpers)}</Note> : <Note tone="neutral">{grantNote}</Note>}
          {error ? (
            <Note tone="bad" label={tr(m.notSavedLabel)}>
              {error.message}
            </Note>
          ) : null}
          <div className="row justify-end gap-2">
            <ButtonLink to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
              {readOnly ? tr(m.backToRolesLower) : tr(m.cancel)}
            </ButtonLink>
            {builtin && list.canManage && role ? (
              <ButtonLink variant="primary" to="/w/$workspace/settings/roles/$roleId" params={{ workspace: slug, roleId: 'new' }} search={{ from: role.id }}>
                {tr(m.duplicateToCustomize)}
              </ButtonLink>
            ) : readOnly ? null : (
              <Button variant="primary" loading={busy || waiting} onClick={() => void requestSave()}>
                {busy || waiting ? tr(m.saving) : role ? tr(m.saveRole) : tr(m.createRole)}
              </Button>
            )}
          </div>
          {waiting ? <span className="k">{tr(m.waitingImpact)}</span> : null}
        </div>
      </section>
      <UnsavedChangesGuard dirty={!readOnly && dirty && !leaving} description={role ? tr(m.unsavedRole, { name: role.name }) : tr(m.unsavedNew)} />
      {dialog && role && impact.data ? (
        <SaveRoleDialog kind={dialog} role={role} name={body.name} impact={impact.data} busy={busy} onBack={() => setDialog(undefined)} onSave={save} />
      ) : null}
    </div>
  )
}

function ImpactValues({ role, impact, readOnly }: { role?: Role; impact?: Impact; readOnly: boolean }) {
  const tr = useT()
  const holders = role ? role.members.filter((x) => x.status === 'active') : []
  const members = !role
    ? tr(m.impactMembersNew)
    : holders.length === 0
      ? '0'
      : tr(m.impactMembersList, {
          count: holders.length,
          names: holders.length > 3 ? tr(m.membersNames, { names: holders.slice(0, 3).map((x) => x.name).join(', '), more: holders.length - 3 }) : holders.map((x) => x.name).join(', '),
        })
  if (readOnly || !impact) {
    return (
      <KeyValues
        items={[
          [tr(m.impactMembers), members],
          [tr(m.impactKeys), readOnly ? '–' : <Skeleton key="k" w={24} />],
          [tr(m.impactAgents), readOnly ? '–' : <Skeleton key="a" w={24} />],
          [tr(m.impactSensitive), readOnly ? '–' : <Skeleton key="s" w="80%" />],
          [tr(m.impactApprovals), readOnly ? '–' : <Skeleton key="p" w="90%" />],
        ]}
      />
    )
  }
  return (
    <KeyValues
      items={[
        [tr(m.impactMembers), members],
        [tr(m.impactKeys), String(impact.apiKeys)],
        [tr(m.impactAgents), String(impact.agents)],
        [tr(m.impactSensitive), impact.sensitive],
        [tr(m.impactApprovals), impact.approvals],
      ]}
    />
  )
}

function EditorSkeleton() {
  const tr = useT()
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_320px] items-start gap-5 max-md:grid-cols-1" aria-busy="true">
      <span className="vh" role="status">
        {tr(m.editorLoading)}
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
