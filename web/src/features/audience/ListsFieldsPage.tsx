import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { TableScroll, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, dayMonthDate } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { AudienceTabs, useAudienceSummary } from './AudienceTabs'
import { DeleteListDialog } from './DeleteListDialog'
import { useInvalidateAudience } from './ContactDialogs'
import { m as am } from './messages'
import { m } from './listsfields.messages'
import { UsesList } from './UsesList'

type List = Schemas['AudienceListSummary']
type FieldDef = Schemas['CustomFieldDef']
type Use = Schemas['Use']
type FieldType = FieldDef['type']

const typeLabel = (ty: string) =>
  ({ text: t(m.typeText), number: t(m.typeNumber), date: t(m.typeDate), boolean: t(m.typeBoolean), select: t(m.typeSelect) })[ty] ?? ty

/**
 * Audience › Lists & fields (review 2026-11 AU-11, AudienceListsFields
 * board): lists and their opt-in; custom fields, what uses each before it
 * goes, and deleted fields that can still come back.
 */
export function ListsFieldsPage() {
  const tr = useT()
  return (
    <Page title={tr(am.title)} filters={<AudienceTabs />} contentClassName="py-5 gap-4">
      <ListsCard />
      <FieldsCard />
    </Page>
  )
}

function ListsCard() {
  const tr = useT()
  const summary = useAudienceSummary()
  const reason = usePermissionReason('lists:manage')
  const [creating, setCreating] = useState(false)
  const [renaming, setRenaming] = useState<List>()
  const [mode, setMode] = useState<List>()
  const [deleting, setDeleting] = useState<List>()
  const lists = summary.data?.lists ?? []
  const phone = useMediaQuery(PHONE)
  const actions = (l: List) => (
    <>
      <Button size="sm" disabledReason={reason} onClick={() => setRenaming(l)}>
        {tr(m.rename)}
      </Button>
      <Button size="sm" disabledReason={reason} onClick={() => setMode(l)}>
        {tr(m.optInMode)}
      </Button>
      <Button size="sm" disabledReason={reason} onClick={() => setDeleting(l)}>
        {tr(m.deleteList)}
      </Button>
    </>
  )
  const optIn = (l: List) => (l.doubleOptIn ? <Badge tone="ok">{tr(m.double)}</Badge> : <Badge tone="neutral">{tr(m.single)}</Badge>)
  return (
    <section className="card" aria-labelledby="lf-lists">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="lf-lists">
            {tr(m.listsTitle, { count: count(lists.length) })}
          </h2>
          <span className="k">{tr(m.listsCaption)}</span>
        </div>
        <Button variant="primary" disabledReason={reason} onClick={() => setCreating(true)}>
          {tr(m.newList)}
        </Button>
      </div>
      {summary.isError ? (
        <div className="pad">
          <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => void summary.refetch()}>{tr(common.retry)}</Button>}>
            {errorMessage(summary.error)}
          </ErrorPanel>
        </div>
      ) : !summary.data ? (
        <RowsSkeleton />
      ) : lists.length === 0 ? (
        <p className="k m-0 px-4.5 py-6 text-center">{tr(m.listsEmpty)}</p>
      ) : phone ? (
        <ul className="m-0 list-none p-0">
          {lists.map((l) => (
            <PhoneRow
              key={l.id}
              title={l.name}
              sub={l.description}
              meta={
                <>
                  <span>{tr(m.subscribedN, { count: count(l.subscribed) })}</span>
                  {optIn(l)}
                </>
              }
              actions={actions(l)}
            />
          ))}
        </ul>
      ) : (
        <TableScroll label={tr(m.listsTitle, { count: count(lists.length) })}>
          <table className="tbl">
            <thead>
              <tr>
                <Th>{tr(m.colList)}</Th>
                <Th>{tr(m.colSubscribed)}</Th>
                <Th>{tr(m.colOptIn)}</Th>
                <Th>
                  <span className="vh">{tr(m.colActions)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {lists.map((l) => (
                <tr key={l.id}>
                  <td className="wrap">
                    <b>{l.name}</b>
                    {l.description ? <div className="k">{l.description}</div> : null}
                  </td>
                  <td className="n">{count(l.subscribed)}</td>
                  <td>{optIn(l)}</td>
                  <td>
                    <span className="flex flex-wrap justify-end gap-1.5">{actions(l)}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
      {creating ? <ListDialog onClose={() => setCreating(false)} /> : null}
      {renaming ? <ListDialog list={renaming} onClose={() => setRenaming(undefined)} /> : null}
      {mode ? <OptInDialog list={mode} onClose={() => setMode(undefined)} /> : null}
      {deleting ? <DeleteListDialog list={deleting} open onOpenChange={(o) => !o && setDeleting(undefined)} /> : null}
    </section>
  )
}

/** New list, or rename one (name and description). */
function ListDialog({ list, onClose }: { list?: List; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const [name, setName] = useState(list?.name ?? '')
  const [description, setDescription] = useState(list?.description ?? '')
  const [double, setDouble] = useState(list?.doubleOptIn ?? true)
  const [err, setErr] = useState<string>()
  const [busy, setBusy] = useState(false)
  const radio = useId()
  const save = async () => {
    if (!name.trim()) {
      setErr(tr(common.required))
      return
    }
    setBusy(true)
    setErr(undefined)
    try {
      if (list) {
        await http.PATCH('/api/v1/workspaces/{workspace}/lists/{id}', { params: { path: { workspace: slug, id: list.id } }, body: { name: name.trim(), description: description.trim() } })
        toast({ kind: 'success', title: t(m.saved) })
      } else {
        await http.POST('/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } }, body: { name: name.trim(), description: description.trim(), doubleOptIn: double } })
        toast({ kind: 'success', title: t(m.listCreated, { name: name.trim() }) })
      }
      await invalidate()
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={list ? tr(m.renameTitle, { name: list.name }) : tr(m.newList)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {list ? tr(common.save) : tr(m.createList)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <Field label={tr(m.listName)} error={err}>
          <Input value={name} autoFocus maxLength={100} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label={tr(m.listDescription)}>
          <Input value={description} maxLength={500} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        {list ? null : (
          <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
            <legend className="mb-1.5 text-xs font-semibold text-ink-2">{tr(m.optIn)}</legend>
            <label className="flex items-center gap-2 text-[13px]">
              <input type="radio" name={radio} checked={double} onChange={() => setDouble(true)} /> {tr(m.doubleHint)}
            </label>
            <label className="flex items-center gap-2 text-[13px]">
              <input type="radio" name={radio} checked={!double} onChange={() => setDouble(false)} /> {tr(m.single)}
            </label>
          </fieldset>
        )}
      </form>
    </Dialog>
  )
}

function OptInDialog({ list, onClose }: { list: List; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const [double, setDouble] = useState(list.doubleOptIn)
  const [err, setErr] = useState<string>()
  const [busy, setBusy] = useState(false)
  const radio = useId()
  const save = async () => {
    setBusy(true)
    try {
      await http.PATCH('/api/v1/workspaces/{workspace}/lists/{id}', {
        params: { path: { workspace: slug, id: list.id } },
        body: { name: list.name, description: list.description, doubleOptIn: double },
      })
      await invalidate()
      toast({ kind: 'success', title: t(m.saved) })
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(m.optInTitle, { name: list.name })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={double === list.doubleOptIn} onClick={() => void save()}>
            {tr(common.save)}
          </Button>
        </>
      }
    >
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="vh">{tr(m.optIn)}</legend>
        <label className="flex items-center gap-2 text-[13px]">
          <input type="radio" name={radio} checked={double} onChange={() => setDouble(true)} /> {tr(m.double)}
        </label>
        <label className="flex items-center gap-2 text-[13px]">
          <input type="radio" name={radio} checked={!double} onChange={() => setDouble(false)} /> {tr(m.single)}
        </label>
        <span className="k">{tr(m.optInNote, { n: list.subscribed, count: count(list.subscribed) })}</span>
        {err ? <span className="text-[13px] font-semibold text-bad-fg">{err}</span> : null}
      </fieldset>
    </Dialog>
  )
}

function FieldsCard() {
  const tr = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('lists:manage')
  const qc = useQueryClient()
  const fields = $api.useQuery('get', '/api/v1/workspaces/{workspace}/fields', { params: { path: { workspace: slug } } })
  const overview = $api.useQuery('get', '/api/v1/workspaces/{workspace}/fields/overview', { params: { path: { workspace: slug } } })
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<FieldDef>()
  const [deleting, setDeleting] = useState<FieldDef>()
  const items = fields.data?.items ?? []
  const usage = new Map((overview.data?.usage ?? []).map((u) => [u.id, u]))
  const deleted = overview.data?.deleted ?? []
  const phone = useMediaQuery(PHONE)
  const refresh = () => qc.invalidateQueries({ predicate: (q) => String(q.queryKey[1] ?? '').includes('/fields') })
  const restore = async (f: Schemas['DeletedField']) => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/fields/{id}/restore', { params: { path: { workspace: slug, id: f.id } } })
      await refresh()
      toast({ kind: 'success', title: t(m.restored, { name: f.label }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  return (
    <section className="card" aria-labelledby="lf-fields">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="lf-fields">
            {tr(m.fieldsTitle, { count: count(items.length) })}
          </h2>
          <span className="k">{tr(m.fieldsCaption, { token: '{{ contact.plan }}' })}</span>
        </div>
        <Button variant="primary" disabledReason={reason} onClick={() => setCreating(true)}>
          {tr(m.newField)}
        </Button>
      </div>
      {fields.isError ? (
        <div className="pad">
          <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => void fields.refetch()}>{tr(common.retry)}</Button>}>
            {errorMessage(fields.error)}
          </ErrorPanel>
        </div>
      ) : !fields.data ? (
        <RowsSkeleton />
      ) : items.length === 0 && deleted.length === 0 ? (
        <p className="k m-0 px-4.5 py-6 text-center">{tr(m.fieldsEmpty)}</p>
      ) : phone ? (
        <ul className="m-0 list-none p-0">
          {items.map((f) => {
            const u = usage.get(f.id)
            return (
              <PhoneRow
                key={f.id}
                title={f.label}
                sub={<span className="mono">{f.key}</span>}
                meta={
                  <>
                    <span>{typeLabel(f.type)}</span>
                    {u?.filled !== undefined ? <span>{tr(m.filledN, { count: count(u.filled) })}</span> : null}
                    <span>{u ? <UsedBy uses={u.uses} /> : null}</span>
                  </>
                }
                actions={
                  <>
                    <Button size="sm" disabledReason={reason} onClick={() => setEditing(f)}>
                      {tr(m.edit)}
                    </Button>
                    <Button size="sm" disabledReason={reason} onClick={() => setDeleting(f)}>
                      {tr(m.deleteField)}
                    </Button>
                  </>
                }
              />
            )
          })}
          {deleted.map((f) => (
            <PhoneRow
              key={f.id}
              title={f.label}
              sub={<span className="mono">{f.key}</span>}
              meta={<Badge tone="warn">{tr(m.deletedBadge, { date: dayMonthDate(new Date(f.restoreUntil)) })}</Badge>}
              actions={
                <Button size="sm" disabledReason={reason} onClick={() => void restore(f)}>
                  {tr(m.restore)}
                </Button>
              }
            />
          ))}
        </ul>
      ) : (
        <TableScroll label={tr(m.fieldsTitle, { count: count(items.length) })}>
          <table className="tbl">
            <thead>
              <tr>
                <Th>{tr(m.colField)}</Th>
                <Th>{tr(m.colKey)}</Th>
                <Th>{tr(m.colType)}</Th>
                <Th>{tr(m.colFilled)}</Th>
                <Th>{tr(m.colUsedBy)}</Th>
                <Th>
                  <span className="vh">{tr(m.colActions)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {items.map((f) => {
                const u = usage.get(f.id)
                return (
                  <tr key={f.id}>
                    <td className="wrap">
                      <b>{f.label}</b>
                    </td>
                    <td className="mono">{f.key}</td>
                    <td>{typeLabel(f.type)}</td>
                    <td className="n">
                      {!overview.data ? (
                        <Skeleton w={40} h={12} />
                      ) : u?.filled === undefined ? (
                        <span className="k" title={tr(m.notCountedHint)}>
                          {tr(m.notCounted)}
                        </span>
                      ) : (
                        count(u.filled)
                      )}
                    </td>
                    <td className="wrap">{u ? <UsedBy uses={u.uses} /> : <Skeleton w={120} h={12} />}</td>
                    <td>
                      <span className="flex flex-wrap justify-end gap-1.5">
                        <Button size="sm" disabledReason={reason} onClick={() => setEditing(f)}>
                          {tr(m.edit)}
                        </Button>
                        <Button size="sm" disabledReason={reason} onClick={() => setDeleting(f)}>
                          {tr(m.deleteField)}
                        </Button>
                      </span>
                    </td>
                  </tr>
                )
              })}
              {deleted.map((f) => (
                <tr key={f.id} className="opacity-90">
                  <td className="wrap">
                    <span className="flex flex-wrap items-center gap-2">
                      <b>{f.label}</b>
                      <Badge tone="warn">{tr(m.deletedBadge, { date: dayMonthDate(new Date(f.restoreUntil)) })}</Badge>
                    </span>
                  </td>
                  <td className="mono">{f.key}</td>
                  <td>{typeLabel(f.type)}</td>
                  <td className="n">–</td>
                  <td className="wrap">
                    <span className="k">{tr(m.valuesKept, { date: dayMonthDate(new Date(f.restoreUntil)) })}</span>
                  </td>
                  <td>
                    <span className="flex justify-end">
                      <Button size="sm" disabledReason={reason} onClick={() => void restore(f)}>
                        {tr(m.restore)}
                      </Button>
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
      {creating ? <FieldDialog onClose={() => setCreating(false)} onSaved={refresh} /> : null}
      {editing ? <FieldDialog field={editing} onClose={() => setEditing(undefined)} onSaved={refresh} /> : null}
      {deleting ? <DeleteFieldDialog field={deleting} onClose={() => setDeleting(undefined)} onDeleted={refresh} /> : null}
    </section>
  )
}

/** "2 segments · 1 form · scoring rules", or "Nothing". */
function UsedBy({ uses }: { uses: Use[] }) {
  const tr = useT()
  if (!uses.length) return <span className="k">{tr(m.nothing)}</span>
  const n = (k: Use['kind']) => uses.filter((u) => u.kind === k).length
  const line = tr(m.usedBy, {
    segments: n('segment'),
    forms: n('form'),
    scoring: n('scoring'),
    automations: n('automation'),
    campaigns: n('campaign'),
    digests: n('digest'),
    views: n('view'),
  })
  return <span>{line.replace(/^ · /, '')}</span>
}

/** "plan" from "Plan", "company_size" from "Company size". */
function keyFrom(label: string) {
  const k = label
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return /^[a-z]/.test(k) ? k.slice(0, 63) : `field_${k}`.slice(0, 63)
}

/** New field (label, key, type, choices), or edit one's label and choices. */
function FieldDialog({ field, onClose, onSaved }: { field?: FieldDef; onClose: () => void; onSaved: () => Promise<unknown> }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const [label, setLabel] = useState(field?.label ?? '')
  const [key, setKey] = useState(field?.key ?? '')
  const [keyTouched, setKeyTouched] = useState(Boolean(field))
  const [type, setType] = useState<FieldType>(field?.type ?? 'text')
  const [options, setOptions] = useState((field?.options ?? []).join('\n'))
  const [err, setErr] = useState<{ label?: string; key?: string; form?: string }>({})
  const [busy, setBusy] = useState(false)
  const choices = options
    .split('\n')
    .map((o) => o.trim())
    .filter(Boolean)
  const save = async () => {
    if (!label.trim()) {
      setErr({ label: tr(common.required) })
      return
    }
    setBusy(true)
    setErr({})
    try {
      if (field) {
        await http.PATCH('/api/v1/workspaces/{workspace}/fields/{id}', {
          params: { path: { workspace: slug, id: field.id } },
          body: { label: label.trim(), options: field.type === 'select' ? choices : undefined, display: field.display },
        })
        toast({ kind: 'success', title: t(m.saved) })
      } else {
        await http.POST('/api/v1/workspaces/{workspace}/fields', {
          params: { path: { workspace: slug } },
          body: { label: label.trim(), key: key.trim(), type, options: type === 'select' ? choices : undefined },
        })
        toast({ kind: 'success', title: t(m.fieldCreated, { name: label.trim() }) })
      }
      await onSaved()
      onClose()
    } catch (e) {
      setErr({ form: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={field ? tr(m.editFieldTitle, { name: field.label }) : tr(m.newField)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {field ? tr(common.save) : tr(m.createField)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <Field label={tr(m.fieldLabel)} error={err.label}>
          <Input
            value={label}
            autoFocus
            maxLength={80}
            onChange={(e) => {
              setLabel(e.target.value)
              if (!keyTouched) setKey(keyFrom(e.target.value))
            }}
          />
        </Field>
        {field ? null : (
          <>
            <Field label={tr(m.fieldKey)} hint={tr(m.fieldKeyHint, { token: `{{ contact.${key || 'plan'} }}` })} error={err.key}>
              <Input
                className="mono"
                value={key}
                maxLength={63}
                onChange={(e) => {
                  setKeyTouched(true)
                  setKey(e.target.value)
                }}
              />
            </Field>
            <Field label={tr(m.fieldType)}>
              <Select value={type} onChange={(e) => setType(e.target.value as FieldType)}>
                {(['text', 'number', 'date', 'boolean', 'select'] as const).map((ty) => (
                  <option key={ty} value={ty}>
                    {typeLabel(ty)}
                  </option>
                ))}
              </Select>
            </Field>
          </>
        )}
        {(field?.type ?? type) === 'select' ? (
          <Field label={tr(m.fieldOptions)}>
            <Textarea rows={4} value={options} onChange={(e) => setOptions(e.target.value)} />
          </Field>
        ) : null}
        {err.form ? <Note tone="bad">{err.form}</Note> : null}
      </form>
    </Dialog>
  )
}

function DeleteFieldDialog({ field, onClose, onDeleted }: { field: FieldDef; onClose: () => void; onDeleted: () => Promise<unknown> }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const impact = $api.useQuery('get', '/api/v1/workspaces/{workspace}/fields/{id}/impact', { params: { path: { workspace: slug, id: field.id } } })
  const [err, setErr] = useState<string>()
  const [busy, setBusy] = useState(false)
  const remove = async () => {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/fields/{id}', { params: { path: { workspace: slug, id: field.id } } })
      await onDeleted()
      toast({ kind: 'deleted', title: t(m.fieldDeleted, { name: field.label }) })
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const uses = impact.data?.uses ?? []
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(m.deleteFieldTitle, { name: field.label })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="danger-primary" loading={busy} disabled={!impact.data} onClick={() => void remove()}>
            {tr(m.deleteField)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {!impact.data ? (
          <span className="k" role="status">
            {tr(m.checking)}
          </span>
        ) : uses.length ? (
          <>
            <span className="text-[13px]">{tr(m.usedIn)}</span>
            <UsesList uses={uses} />
          </>
        ) : (
          <span className="text-[13px]">{tr(m.unused)}</span>
        )}
        <Note tone="warn">{tr(m.deleteNote)}</Note>
        {err ? <span className="text-[13px] font-semibold text-bad-fg">{err}</span> : null}
      </div>
    </Dialog>
  )
}

/** Phones: one row per item, its facts on a line and its buttons below. */
const PHONE = '(max-width: 767px)'
function PhoneRow({ title, sub, meta, actions }: { title: ReactNode; sub?: ReactNode; meta: ReactNode; actions: ReactNode }) {
  return (
    <li className="flex flex-col gap-1.5 border-t border-line-2 px-4 py-3.5 first:border-t-0">
      <b className="min-w-0 [overflow-wrap:anywhere]">{title}</b>
      {sub ? <span className="k min-w-0 [overflow-wrap:anywhere]">{sub}</span> : null}
      <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px]">{meta}</span>
      <span className="mt-1 flex flex-wrap gap-1.5">{actions}</span>
    </li>
  )
}

function RowsSkeleton() {
  return (
    <div className="flex flex-col" aria-hidden="true">
      {[180, 140, 160].map((w, i) => (
        <div key={i} className="flex items-center gap-6 border-t border-line-2 px-4.5 py-3.5 first:border-t-0">
          <Skeleton w={w} h={13} />
          <Skeleton w={60} h={12} />
          <Skeleton w={90} h={20} r={999} />
          <Skeleton w={120} h={26} className="ml-auto" />
        </div>
      ))}
    </div>
  )
}
