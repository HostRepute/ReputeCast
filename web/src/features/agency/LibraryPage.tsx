import { useNavigate, useSearch } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useAgencyMe, useRefreshAgency } from './api'
import { AgencyPage } from './AgencyShell'
import { kindLabel } from './kinds'
import { m as lm } from './library.messages'
import { HtmlPreview, LibraryThumb } from './LibraryThumb'
import { m } from './messages'
import { ClientTile, dayMonth } from './parts'

type Template = Schemas['AgencyTemplate']
type Item = Schemas['AgencyLibraryItem']
type Kind = 'section' | 'template' | 'automation' | 'sms' | 'transactional'
type Search = { kind?: Kind; item?: string }

const kindTone: Record<string, Tone> = { section: 'neutral', template: 'info', automation: 'purple', sms: 'ok', transactional: 'warn' }

/** Agency › Library (AgencyLibrary): workspace templates and the shared library. */
export function LibraryPage() {
  const t = useT()
  const templates = $api.useQuery('get', '/api/v1/agency/templates')
  const lib = $api.useQuery('get', '/api/v1/agency/library')
  const [q, setQ] = useState('')
  const total = lib.data?.total ?? 0
  return (
    <AgencyPage
      title={t(m.navLibrary)}
      subtitle={lib.data && templates.data ? t(lm.subtitle, { items: total, templates: templates.data.length }) : undefined}
      actions={
        <label className="flex h-9 items-center gap-2 rounded-lg border border-[#8B94A3] bg-white px-2.5 max-md:w-full" style={{ width: 260, maxWidth: '100%' }}>
          <Icon name="search" size={15} className="text-ink-3" />
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t(lm.searchPlaceholder)}
            aria-label={t(lm.searchPlaceholder)}
            className="min-w-0 flex-1 border-0 bg-transparent text-[13px] outline-none"
          />
        </label>
      }
    >
      {lib.data && !lib.data.canEdit && lib.data.reason ? (
        <Note tone="neutral" label={<b>{t(lm.readOnly)}</b>}>
          {lib.data.reason}
        </Note>
      ) : null}
      <TemplatesSection canEdit={lib.data?.canEdit ?? false} />
      <SharedLibrary filter={q} setFilter={setQ} />
    </AgencyPage>
  )
}

function TemplatesSection({ canEdit }: { canEdit: boolean }) {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/agency/templates')
  const [applying, setApplying] = useState<Template>()
  const [editing, setEditing] = useState<Template>()
  const [creating, setCreating] = useState(false)
  const list = q.data ?? []
  return (
    <section aria-label={t(lm.templatesTitle, { n: list.length })} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <h2 className="h2">{t(lm.templatesTitle, { n: list.length })}</h2>
          <span className="k">{t(lm.templatesCaption)}</span>
        </div>
        {canEdit ? (
          <Button size="sm" icon="plus" onClick={() => setCreating(true)}>
            {t(lm.newTemplate)}
          </Button>
        ) : null}
      </div>
      {q.isError ? (
        <ErrorPanel title={t(lm.templatesFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(300px,100%),1fr))' }}>
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} w="100%" h={260} r={10} />
          ))}
        </div>
      ) : (
        <div className="grid items-stretch gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(300px,100%),1fr))' }}>
          {list.map((tpl) => (
            <TemplateCard key={tpl.id} tpl={tpl} canEdit={canEdit} onApply={() => setApplying(tpl)} onEdit={() => setEditing(tpl)} />
          ))}
        </div>
      )}
      <div className="note n" role="note">
        <Icon name="info" size={16} className="text-ink-2" />
        <div className="min-w-0 flex-[1_1_320px]">
          <b>{t(lm.neverCopiedLabel)}</b> {t(lm.neverCopiedBody)}
        </div>
      </div>
      {applying ? <ApplyDialog tpl={applying} onClose={() => setApplying(undefined)} /> : null}
      {editing ? <EditTemplateDialog tpl={editing} onClose={() => setEditing(undefined)} /> : null}
      {creating ? <NewTemplateDialog onClose={() => setCreating(false)} /> : null}
    </section>
  )
}

/** What a template sets up, as the card's chips. */
function setsUpChips(t: ReturnType<typeof useT>, tpl: Template) {
  const s = tpl.setsUp
  const chips: string[] = []
  if (s.lists) chips.push(t(lm.chipLists, { n: s.lists }))
  if (s.automations) chips.push(s.automationsNote ? t(lm.chipAutomationsNote, { n: s.automations, note: s.automationsNote }) : t(lm.chipAutomations, { n: s.automations }))
  if (s.sections) chips.push(t(lm.chipSections, { n: s.sections }))
  if (s.templates) chips.push(t(lm.chipTemplates, { n: s.templates }))
  if (s.smsTexts) chips.push(t(lm.chipSms, { n: s.smsTexts }))
  if (s.forms) chips.push(t(lm.chipForms, { n: s.forms }))
  if (s.clientSignoff) chips.push(t(lm.chipPolicy))
  if (s.roles) chips.push(t(lm.chipRoles, { n: s.roles }))
  return chips
}

function TemplateCard({ tpl, canEdit, onApply, onEdit }: { tpl: Template; canEdit: boolean; onApply: () => void; onEdit: () => void }) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-col gap-3 p-4" aria-label={tpl.name}>
      <div className="flex items-start gap-2.5">
        <span className="inline-flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-line bg-ground" aria-hidden="true">
          <Icon name="layers" size={17} className="text-accent" />
        </span>
        <span className="flex min-w-0 flex-1 flex-col items-start gap-[3px]">
          <span className="font-bold">{tpl.name}</span>
          <span className="k">{tpl.updatedBy ? t(lm.updatedBy, { date: dayMonth(tpl.updatedAt), name: tpl.updatedBy }) : t(lm.updated, { date: dayMonth(tpl.updatedAt) })}</span>
          {tpl.default ? <Badge tone="info">{t(lm.defaultBadge)}</Badge> : null}
        </span>
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="k font-semibold text-[#374151]!">{t(lm.setsUp)}</span>
        <div className="flex flex-wrap gap-1.5">
          {setsUpChips(t, tpl).map((c) => (
            <span key={c} className="chip h-auto! min-h-[26px] whitespace-normal! py-[3px]!">
              {c}
            </span>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="k font-semibold text-[#374151]!">{t(lm.usedBy, { n: tpl.usedBy.length })}</span>
        {tpl.usedBy.length ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="flex flex-wrap gap-1">
              {tpl.usedBy.map((w) => (
                <ClientTile key={w.id} ws={w} size={22} />
              ))}
            </span>
            <span className="k text-ink-2!">{tpl.usedBy.map((w) => w.name).join(', ')}</span>
          </div>
        ) : (
          <span className="k">{t(lm.unused)}</span>
        )}
      </div>
      {canEdit ? (
        <div className="mt-auto flex flex-wrap gap-2 pt-1">
          <Button size="sm" variant="primary" onClick={onApply}>
            {t(lm.applyToClient)}
          </Button>
          <Button size="sm" onClick={onEdit}>
            {t(m.edit)}
          </Button>
        </div>
      ) : null}
    </section>
  )
}

/** Apply a template to an existing client (AgencyLibrary states): the plan, then apply. */
function ApplyDialog({ tpl, onClose }: { tpl: Template; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const { data: agency } = useAgencyMe()
  const clients = agency?.clients ?? []
  const [ws, setWs] = useState(clients.find((c) => !tpl.usedBy.some((u) => u.id === c.workspace.id))?.workspace.id ?? clients[0]?.workspace.id ?? '')
  const plan = $api.useQuery('get', '/api/v1/agency/templates/{templateId}/plan', { params: { path: { templateId: tpl.id }, query: { workspaceId: ws } } }, { enabled: Boolean(ws) })
  const [busy, setBusy] = useState(false)
  const client = clients.find((c) => c.workspace.id === ws)?.workspace
  const apply = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/agency/templates/{templateId}/apply', { params: { path: { templateId: tpl.id } }, body: { workspaceId: ws } })
      await refresh()
      toast({ kind: 'success', title: t(lm.applied, { template: tpl.name, client: client?.name ?? '' }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.applyFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const p = plan.data
  return (
    <Dialog
      open
      size="lg"
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{t(lm.applyCrumb)}</span>}
      title={client ? t(lm.applyTitle, { template: tpl.name, client: client.name }) : t(lm.applyTitlePick, { template: tpl.name })}
      footer={
        <>
          <span className="k mr-auto">{t(lm.applyFoot)}</span>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!p} onClick={apply}>
            {client ? t(lm.applyTo, { client: client.name }) : t(lm.applyToClient)}
          </Button>
        </>
      }
    >
      <Field label={t(lm.client)}>
        <Select value={ws} onChange={(e) => setWs(e.target.value)}>
          {clients.map((c) => (
            <option key={c.workspace.id} value={c.workspace.id}>
              {c.workspace.name}
            </option>
          ))}
        </Select>
      </Field>
      {client ? <span className="text-[#374151]">{t(lm.applyIntro, { client: client.name })}</span> : null}
      {plan.isError ? (
        <span className="err" role="alert">
          {errorMessage(plan.error)}
        </span>
      ) : !p ? (
        <Skeleton w="100%" h={160} r={8} />
      ) : (
        <>
          <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(260px,100%),1fr))' }}>
            <PlanGroup title={t(lm.planLists)} added={p.lists.added.length ? [t(lm.planListsAdded, { names: formatList(p.lists.added) })] : []} renamed={p.lists.renamed.map((r) => t(lm.planListRenamed, { from: r.from, to: r.to }))} />
            <PlanGroup
              title={t(lm.planAutomations)}
              added={p.automations.added.length ? [p.automations.added.join(' · ')] : []}
              renamed={p.automations.renamed.map((r) => t(lm.planRenamed, { from: r.from, to: r.to }))}
            />
            <PlanGroup
              title={t(lm.planContent)}
              added={
                p.sections.added.length || p.templates.added.length
                  ? [t(lm.planContentAdded, { sections: p.sections.added.length, templates: p.templates.added.length })]
                  : []
              }
              renamed={
                p.sections.renamed.length + p.templates.renamed.length
                  ? [t(lm.planContentRenamed, { n: p.sections.renamed.length + p.templates.renamed.length })]
                  : []
              }
            />
            <PlanGroup
              title={t(lm.planPolicy)}
              added={[...(p.clientSignoff === 'on' ? [t(lm.planSignoffOn)] : []), ...(p.clientRoles === 'added' ? [t(lm.planRolesAdded)] : [])]}
              unchanged={[...(p.clientSignoff === 'already' ? [t(lm.planSignoffAlready)] : []), ...(p.clientRoles === 'already' ? [t(lm.planRolesAlready)] : [])]}
            />
          </div>
          {p.skipped.length ? (
            <Note tone="warn">{t(lm.planSkipped, { parts: formatList(p.skipped.map((s) => `${s.part} (${s.permission})`)) })}</Note>
          ) : null}
          <div className="note n">
            <Icon name="info" size={16} className="text-ink-2" />
            <div className="min-w-0 flex-[1_1_320px]">{t(lm.planNotCopied, { client: client?.name ?? '' })}</div>
          </div>
        </>
      )}
    </Dialog>
  )
}

function PlanGroup({ title, added = [], renamed = [], unchanged = [] }: { title: string; added?: string[]; renamed?: string[]; unchanged?: string[] }) {
  if (!added.length && !renamed.length && !unchanged.length) return null
  return (
    <div className="flex flex-col gap-1.5">
      <span className="font-semibold">{title}</span>
      {added.map((l) => (
        <span key={l} className="flex items-start gap-2 text-[13px]">
          <Icon name="plus" size={14} className="mt-0.5 flex-none text-[#15803D]" />
          {l}
        </span>
      ))}
      {renamed.map((l) => (
        <span key={l} className="flex items-start gap-2 text-[13px]">
          <Icon name="edit" size={14} className="mt-0.5 flex-none text-[#B45309]" />
          {l}
        </span>
      ))}
      {unchanged.map((l) => (
        <span key={l} className="flex items-start gap-2 text-[13px]">
          <Icon name="check" size={14} className="mt-0.5 flex-none text-[#15803D]" />
          {l}
        </span>
      ))}
    </div>
  )
}

const partKeys = ['lists', 'automations', 'sections', 'templates', 'clientSignoff'] as const

/** Edit a template: name, description, the parts it sets up; or remove it. */
function EditTemplateDialog({ tpl, onClose }: { tpl: Template; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const [name, setName] = useState(tpl.name)
  const [desc, setDesc] = useState(tpl.description ?? '')
  const [exclude, setExclude] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const labels: Record<(typeof partKeys)[number], string> = {
    lists: t(lm.chipLists, { n: tpl.setsUp.lists }),
    automations: t(lm.chipAutomations, { n: tpl.setsUp.automations }),
    sections: t(lm.chipSections, { n: tpl.setsUp.sections }),
    templates: t(lm.chipTemplates, { n: tpl.setsUp.templates }),
    clientSignoff: t(lm.chipPolicy),
  }
  const present = partKeys.filter((k) => (k === 'clientSignoff' ? tpl.setsUp.clientSignoff : tpl.setsUp[k] > 0))
  const save = async () => {
    setBusy(true)
    try {
      await http.PATCH('/api/v1/agency/templates/{templateId}', { params: { path: { templateId: tpl.id } }, body: { name: name.trim(), description: desc.trim(), exclude } })
      await refresh()
      toast({ kind: 'success', title: t(lm.templateSaved, { name: name.trim() }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/agency/templates/{templateId}', { params: { path: { templateId: tpl.id } } })
      await refresh()
      toast({ kind: 'deleted', title: t(lm.templateRemoved, { name: tpl.name }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.removeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  if (confirmDelete) {
    return (
      <Dialog
        open
        onOpenChange={(o) => (o ? null : onClose())}
        title={t(lm.removeTemplateTitle, { name: tpl.name })}
        description={t(lm.removeTemplateBody)}
        footer={
          <>
            <Button onClick={() => setConfirmDelete(false)}>{t(m.cancel)}</Button>
            <Button variant="danger-primary" loading={busy} onClick={remove}>
              {t(m.remove)}
            </Button>
          </>
        }
      />
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(lm.editTemplate, { name: tpl.name })}
      footer={
        <>
          <Button variant="danger" className="mr-auto" onClick={() => setConfirmDelete(true)}>
            {t(m.remove)}
          </Button>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!name.trim()} onClick={save}>
            {t(m.save)}
          </Button>
        </>
      }
    >
      <Field label={t(lm.templateName)}>
        <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} />
      </Field>
      <Field label={t(lm.templateDescription)}>
        <Textarea value={desc} onChange={(e) => setDesc(e.target.value)} maxLength={300} rows={2} />
      </Field>
      <div className="lbl" role="group" aria-labelledby="et-parts">
        <span id="et-parts">{t(lm.setsUp)}</span>
        {present.map((k) => (
          <label key={k} className="flex items-center gap-2 text-[13px] font-normal text-ink">
            <input type="checkbox" checked={!exclude.includes(k)} onChange={(e) => setExclude(e.target.checked ? exclude.filter((x) => x !== k) : [...exclude, k])} />
            {labels[k]}
          </label>
        ))}
        <span className="k">{t(lm.partsHint)}</span>
      </div>
    </Dialog>
  )
}

/** A new template: a client's setup, saved ("Save as workspace template"). */
function NewTemplateDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const { data: agency } = useAgencyMe()
  const clients = agency?.clients ?? []
  const [ws, setWs] = useState(clients[0]?.workspace.slug ?? '')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/agency/template', { params: { path: { workspace: ws } }, body: { name: name.trim() } })
      await refresh()
      toast({ kind: 'success', title: t(lm.templateSaved, { name: name.trim() }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(lm.newTemplate)}
      description={t(lm.newTemplateBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!name.trim() || !ws} onClick={save}>
            {t(lm.saveTemplate)}
          </Button>
        </>
      }
    >
      <Field label={t(lm.fromClient)}>
        <Select value={ws} onChange={(e) => setWs(e.target.value)}>
          {clients.map((c) => (
            <option key={c.workspace.id} value={c.workspace.slug}>
              {c.workspace.name}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t(lm.templateName)}>
        <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} autoFocus />
      </Field>
    </Dialog>
  )
}

const kinds: Kind[] = ['section', 'template', 'automation', 'sms', 'transactional']

function SharedLibrary({ filter, setFilter }: { filter: string; setFilter: (v: string) => void }) {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/agency/library')
  const search = useSearch({ strict: false }) as Search
  const navigate = useNavigate()
  const { slug } = useWorkspace()
  const { data: agency } = useAgencyMe()
  const [all, setAll] = useState(false)
  const set = (patch: Partial<Search>) =>
    void navigate({ to: '/w/$workspace/agency/library', params: { workspace: slug }, search: { kind: search.kind, item: search.item, ...patch }, replace: true })
  const d = q.data
  const items = useMemo(() => {
    const list = d?.items ?? []
    const f = filter.trim().toLowerCase()
    return list.filter((i) => (!search.kind || i.kind === search.kind) && (!f || i.name.toLowerCase().includes(f)))
  }, [d, search.kind, filter])
  const selectedID = search.item ?? items.find((i) => i.behind > 0)?.id ?? items[0]?.id
  const shown = all ? items : items.slice(0, 6)
  const chipLabel: Record<Kind, string> = { section: t(lm.tabSections), template: t(lm.tabCampaigns), automation: t(lm.tabAutomations), sms: t(lm.tabSms), transactional: t(lm.tabTransactional) }
  return (
    <div className="flex flex-wrap items-start gap-4">
      <div className="min-w-0 flex-[3_1_560px]">
        <section className="card min-w-0" aria-label={t(lm.libraryTitle, { n: d?.total ?? 0 })}>
          <div className="ag-c8h">
            <div>
              <h2 className="h2">{t(lm.libraryTitle, { n: d?.total ?? 0 })}</h2>
              <span className="k">{t(lm.libraryCaption, { agency: agency?.name ?? '' })}</span>
            </div>
            <div>
              <label className="flex h-9 items-center gap-2 rounded-lg border border-[#8B94A3] bg-white px-2.5" style={{ width: 220, maxWidth: '100%' }}>
                <Icon name="search" size={15} className="text-ink-3" />
                <input
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  placeholder={t(lm.searchPlaceholder)}
                  aria-label={t(lm.searchPlaceholder)}
                  className="min-w-0 flex-1 border-0 bg-transparent text-[13px] outline-none"
                />
              </label>
            </div>
          </div>
          <div className="ag-fchips" role="group" aria-label={t(lm.kindLabel)}>
            <button type="button" className="ag-fchip" aria-pressed={!search.kind} onClick={() => set({ kind: undefined })}>
              {t(lm.tabAll)} <b>{d?.total ?? 0}</b>
            </button>
            {kinds.map((k) => (
              <button key={k} type="button" className="ag-fchip" aria-pressed={search.kind === k} onClick={() => set({ kind: k })}>
                {chipLabel[k]} <b>{d?.counts[k] ?? 0}</b>
              </button>
            ))}
          </div>
          <div className="p-4">
            {q.isError ? (
              <ErrorPanel title={t(lm.libraryFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
                {errorMessage(q.error)}
              </ErrorPanel>
            ) : !d ? (
              <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(210px,100%),1fr))' }}>
                {[0, 1, 2, 3].map((i) => (
                  <Skeleton key={i} w="100%" h={230} r={10} />
                ))}
              </div>
            ) : items.length === 0 ? (
              <p className="k m-0">{filter ? t(lm.noMatch, { q: filter }) : t(lm.emptyKind)}</p>
            ) : (
              <div className="grid items-stretch gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(210px,100%),1fr))' }}>
                {shown.map((it) => (
                  <ItemCard key={it.id} item={it} selected={it.id === selectedID} onSelect={() => set({ item: it.id })} />
                ))}
              </div>
            )}
          </div>
          {d && items.length > 6 ? (
            <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 px-[18px] py-3">
              <span className="k">{all ? t(lm.showingAll, { n: items.length }) : t(lm.showing, { shown: Math.min(6, items.length), n: items.length })}</span>
              <Button size="sm" onClick={() => setAll(!all)}>
                {all ? t(lm.showFewer) : t(lm.showAll, { n: items.length })}
              </Button>
            </div>
          ) : null}
        </section>
      </div>
      <div className="min-w-0 flex-[2_1_380px]">{selectedID ? <ItemDetail id={selectedID} canEdit={d?.canEdit ?? false} /> : null}</div>
    </div>
  )
}

function usage(t: ReturnType<typeof useT>, it: Item) {
  if (!it.clients) return t(lm.inNoClients)
  return it.behind ? t(lm.inClientsBehind, { n: it.clients, behind: it.behind }) : t(lm.inClients, { n: it.clients })
}

function ItemCard({ item, selected, onSelect }: { item: Item; selected: boolean; onSelect: () => void }) {
  const t = useT()
  const [copying, setCopying] = useState(false)
  return (
    <article
      className="card flex min-w-0 flex-col overflow-hidden"
      style={selected ? { borderColor: '#0F766E', boxShadow: 'inset 0 0 0 1px #0F766E' } : undefined}
      aria-current={selected ? 'true' : undefined}
    >
      <LibraryThumb item={item} />
      <div className="flex flex-1 flex-col gap-1.5 px-3.5 pt-3 pb-3.5">
        <button type="button" className="cursor-pointer border-0 bg-transparent p-0 text-left font-semibold text-ink" onClick={onSelect}>
          {item.name}
        </button>
        <span className="flex flex-wrap items-center gap-1.5">
          <Badge tone={kindTone[item.kind] ?? 'neutral'} className="h-5! text-[11px]!">
            {kindLabel(t, item.kind)}
          </Badge>
          <span className="k">v{item.version}</span>
        </span>
        <span className="k">{item.updatedBy ? t(lm.updatedOn, { date: dayMonth(item.updatedAt), name: item.updatedBy }) : dayMonth(item.updatedAt)}</span>
        <span className="k" style={{ color: item.behind ? '#92400E' : '#4B5563' }}>
          {usage(t, item)}
        </span>
        <div className="mt-auto pt-1.5">
          {item.kind === 'sms' ? (
            <span className="k">{t(lm.smsFromComposer)}</span>
          ) : (
            <Button size="sm" onClick={() => setCopying(true)}>
              {t(lm.copyToClient)}
            </Button>
          )}
        </div>
      </div>
      {copying ? <CopyDialog itemID={item.id} name={item.name} onClose={() => setCopying(false)} /> : null}
    </article>
  )
}

/** A library item: preview, actions, fields, versions, copies and the diff (AgencyLibrary). */
function ItemDetail({ id, canEdit }: { id: string; canEdit: boolean }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const q = $api.useQuery('get', '/api/v1/agency/library/{itemId}', { params: { path: { itemId: id } } })
  const [copying, setCopying] = useState(false)
  const [editing, setEditing] = useState(false)
  const [removing, setRemoving] = useState(false)
  const [busy, setBusy] = useState(false)
  if (q.isError) {
    return (
      <ErrorPanel title={t(lm.itemFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  }
  const d = q.data
  if (!d) return <Skeleton w="100%" h={520} r={10} />
  const item = d.item
  const behind = d.copies.filter((c) => c.behind)
  const firstBehind = behind[0]
  const update = async (copyID: string, client: string) => {
    setBusy(true)
    try {
      await http.POST('/api/v1/agency/library/copies/{copyId}/update', { params: { path: { copyId: copyID } } })
      await refresh()
      toast({ kind: 'success', title: t(lm.copyUpdated, { client }) })
    } catch (e) {
      toast({ kind: 'error', title: t(lm.updateFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/agency/library/{itemId}', { params: { path: { itemId: id } } })
      await refresh()
      toast({ kind: 'deleted', title: t(lm.itemRemoved, { name: item.name }) })
      setRemoving(false)
    } catch (e) {
      toast({ kind: 'error', title: t(lm.removeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const format = item.kind === 'section' || item.kind === 'template' ? t(lm.formatMjml) : item.kind === 'sms' ? t(lm.formatText) : t(lm.formatFlow)
  return (
    <section className="card min-w-0" aria-label={item.name}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2">{item.name}</h2>
          <span className="k">{t(lm.detailMeta, { kind: kindLabel(t, item.kind), format, version: item.version, n: item.clients })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        {d.preview && item.kind !== 'sms' ? (
          <HtmlPreview html={d.preview} label={t(lm.previewLabel, { name: item.name, version: item.version })} />
        ) : d.preview ? (
          <div className="rounded-lg border border-line bg-ground px-3.5 py-3 text-[13.5px] whitespace-pre-wrap">{d.preview}</div>
        ) : (
          <LibraryThumb item={item} height={150} />
        )}
        <div className="flex flex-wrap gap-2">
          {item.kind !== 'sms' ? (
            <Button size="sm" variant="primary" onClick={() => setCopying(true)}>
              {t(lm.copyToClient)}
            </Button>
          ) : null}
          {canEdit ? (
            <>
              <Button size="sm" onClick={() => setEditing(true)}>
                {t(lm.editInLibrary)}
              </Button>
              <Button size="sm" variant="danger" onClick={() => setRemoving(true)}>
                {t(m.remove)}
              </Button>
            </>
          ) : null}
        </div>
        {item.fields.length ? (
          <div className="flex flex-col gap-1.5">
            <span className="k font-semibold text-[#374151]!">{t(lm.fieldsNeeded)}</span>
            <div className="flex flex-wrap gap-1.5">
              {item.fields.map((f) => (
                <span key={f.key} className="flex items-center gap-1.5">
                  <span className="ag-mk">{f.key}</span>
                  {f.fallback ? <span className="k self-center">{t(lm.fallsBack, { value: f.fallback })}</span> : null}
                </span>
              ))}
            </div>
          </div>
        ) : null}
        <div className="flex flex-col">
          <span className="k mb-0.5 font-semibold text-[#374151]!">{t(lm.versions)}</span>
          {d.versions.map((v, i) => (
            <div key={v.version} className="ag-mrow flex-nowrap! items-start! gap-2.5!">
              <Badge tone={i === 0 ? 'ok' : 'neutral'}>v{v.version}</Badge>
              <span className="flex min-w-0 flex-1 flex-col gap-px">
                <span className="font-semibold">{v.by ? t(lm.versionBy, { date: dayMonth(v.at), name: v.by }) : dayMonth(v.at)}</span>
                {v.note ? <span className="k">{v.note}</span> : null}
              </span>
            </div>
          ))}
        </div>
        {d.copies.length ? (
          <div className="flex flex-col">
            <span className="k mb-0.5 font-semibold text-[#374151]!">
              {behind.length ? t(lm.copiesBehind, { n: behind.length }) : t(lm.copies)}
            </span>
            {d.copies.map((c) => (
              <div key={c.id} className="ag-mrow gap-2.5!">
                <ClientTile ws={c.workspace} size={22} />
                <span className="min-w-0 flex-1 font-semibold">{c.workspace.name}</span>
                {c.behind ? <Badge tone="warn">{t(lm.copyBehind, { v: c.version })}</Badge> : <Badge tone="ok">v{c.version}</Badge>}
              </div>
            ))}
          </div>
        ) : null}
        {d.diff && d.diff.lines.length ? (
          <div className="diff">
            <div className="ctx font-semibold" style={{ background: '#F9FAFB' }}>
              {t(lm.diffHead, { from: d.diff.from, to: d.diff.to })}
            </div>
            {d.diff.lines.map((l, i) => (
              <div key={i} className={l.op}>
                {l.op === 'del' ? '− ' : l.op === 'add' ? '+ ' : ''}
                {l.text}
              </div>
            ))}
          </div>
        ) : null}
        {firstBehind ? (
          <div className="flex flex-wrap gap-2">
            <Button size="sm" loading={busy} disabled={!firstBehind.canUpdate} disabledReason={firstBehind.canUpdate ? undefined : firstBehind.reason} onClick={() => update(firstBehind.id, firstBehind.workspace.name)}>
              {t(lm.updateCopy, { client: firstBehind.workspace.name })}
            </Button>
            <span className="k self-center">{t(lm.orLeaveIt)}</span>
          </div>
        ) : null}
      </div>
      {copying ? <CopyDialog itemID={item.id} name={item.name} onClose={() => setCopying(false)} /> : null}
      {editing ? <EditItemDialog id={id} name={item.name} onClose={() => setEditing(false)} /> : null}
      {removing ? (
        <Dialog
          open
          onOpenChange={(o) => (o ? null : setRemoving(false))}
          title={t(lm.removeItemTitle, { name: item.name })}
          description={t(lm.removeItemBody)}
          footer={
            <>
              <Button onClick={() => setRemoving(false)}>{t(m.cancel)}</Button>
              <Button variant="danger-primary" loading={busy} onClick={remove}>
                {t(m.remove)}
              </Button>
            </>
          }
        />
      ) : null}
    </section>
  )
}

/** Copy to a client: where you can edit that kind of content. */
function CopyDialog({ itemID, name, onClose }: { itemID: string; name: string; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const q = $api.useQuery('get', '/api/v1/agency/library/{itemId}', { params: { path: { itemId: itemID } } })
  const targets = q.data?.clients ?? []
  const [ws, setWs] = useState<string>()
  const [busy, setBusy] = useState(false)
  const chosen = ws ?? targets.find((c) => c.canCopy)?.workspace.id
  const target = targets.find((c) => c.workspace.id === chosen)
  const copy = async () => {
    if (!chosen) return
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/agency/library/{itemId}/copies', { params: { path: { itemId: itemID } }, body: { workspaceId: chosen } })
      await refresh()
      toast({ kind: 'success', title: t(lm.copied, { name: data?.name ?? name, client: target?.workspace.name ?? '' }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.copyFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(lm.copyTitle, { name })}
      description={t(lm.copyBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!target?.canCopy} onClick={copy}>
            {t(lm.makeCopy)}
          </Button>
        </>
      }
    >
      {q.isPending ? (
        <Skeleton w="100%" h={120} r={8} />
      ) : (
        <div className="flex flex-col" role="radiogroup" aria-label={t(lm.client)}>
          {targets.map((c) => (
            <label key={c.workspace.id} className="flex items-center gap-2.5 border-t border-line-2 py-2 first:border-t-0" style={{ opacity: c.canCopy ? 1 : 0.6 }}>
              <input type="radio" name="copy-to" checked={chosen === c.workspace.id} disabled={!c.canCopy} onChange={() => setWs(c.workspace.id)} />
              <ClientTile ws={c.workspace} size={20} />
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="font-medium">{c.workspace.name}</span>
                {!c.canCopy && c.reason ? <span className="k">{c.reason}</span> : null}
              </span>
            </label>
          ))}
        </div>
      )}
    </Dialog>
  )
}

function EditItemDialog({ id, name, onClose }: { id: string; name: string; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const [value, setValue] = useState(name)
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      await http.PATCH('/api/v1/agency/library/{itemId}', { params: { path: { itemId: id } }, body: { name: value.trim() } })
      await refresh()
      toast({ kind: 'success', title: t(lm.itemSaved, { name: value.trim() }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(lm.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(lm.editItemTitle)}
      description={t(lm.editItemBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!value.trim()} onClick={save}>
            {t(m.save)}
          </Button>
        </>
      }
    >
      <Field label={t(lm.itemName)}>
        <Input value={value} onChange={(e) => setValue(e.target.value)} maxLength={120} autoFocus />
      </Field>
    </Dialog>
  )
}
