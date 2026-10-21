import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { useAgencyCan, useAgencyMe } from './api'
import { kindLabel } from './kinds'
import { LibraryThumb } from './LibraryThumb'
import { m } from './messages'
import { useWorkspaceAgency } from './WorkspaceParts'

type Item = Schemas['AgencyLibraryItem']
type Copy = Schemas['AgencyGalleryCopy']

const galleryKey = ['get', '/api/v1/workspaces/{workspace}/agency/library'] as const
const mono = (c: React.ReactNode) => <span className="mono">{c}</span>

/** The agency's staff in one of its clients: the library is theirs to use there. */
function useStaffInClient() {
  const { data } = useWorkspaceAgency()
  return Boolean(data?.managed && data.kind === 'client' && data.viewerIsStaff)
}

/**
 * The composer's gallery in a client workspace (AgencyLibrary, "In a client
 * · the composer's template gallery"): the agency library's sections and
 * campaign templates, for the agency's staff. Using one copies it into the
 * workspace and starts the email from the copy.
 */
export function AgencyComposerGallery({ canEdit, onLoad }: { canEdit: boolean; onLoad: (mjml: string, label: string) => void }) {
  const t = useT()
  const qc = useQueryClient()
  const { slug, me } = useWorkspace()
  const staff = useStaffInClient()
  const { data: agency } = useAgencyMe()
  const { data: who } = useMe()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/agency/library', { params: { path: { workspace: slug }, query: { kind: 'composer' } } }, { enabled: staff, staleTime: 60_000 })
  const [busy, setBusy] = useState<string>()
  const [all, setAll] = useState(false)
  if (!staff || !q.data || q.data.items.length === 0) return null
  const d = q.data
  const copyOf = new Map<string, Copy>()
  for (const c of d.copies) {
    const prev = copyOf.get(c.itemId)
    if (!prev || c.version > prev.version) copyOf.set(c.itemId, c)
  }
  const use = async (item: Item) => {
    setBusy(item.id)
    try {
      const { data: copy } = await http.POST('/api/v1/workspaces/{workspace}/agency/library/{itemId}/use', { params: { path: { workspace: slug, itemId: item.id } } })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/templates'] })
      void qc.invalidateQueries({ queryKey: galleryKey })
      const { data: tpl } = await http.GET('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: slug, templateId: copy!.targetId } } })
      if (tpl) onLoad(tpl.mjml, t(m.galleryStartedFrom, { name: item.name, agency: d.agency }))
      toast({ kind: 'success', title: t(m.galleryCopied, { name: copy!.name, workspace: me.workspace.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.galleryCopyFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  const reason = !canEdit ? undefined : d.canUse ? undefined : d.reason
  return (
    <section className="flex flex-col gap-2" aria-labelledby="ag-gallery-h">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <span className="font-semibold" id="ag-gallery-h">
          {t(m.galleryTitle, { agency: d.agency, n: d.items.length })}
        </span>
        {d.role && who ? <span className="k">{t(m.galleryRole, { person: who.user.name, name: me.workspace.name, role: roleName({ name: d.role }), agency: d.agency })}</span> : null}
      </div>
      <div className="grid items-stretch gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(180px,100%),1fr))' }}>
        {(all ? d.items : d.items.slice(0, 8)).map((item) => {
          const copy = copyOf.get(item.id)
          const fields = item.fields.map((f) => f.key)
          return (
            <article key={item.id} className="card flex flex-col overflow-hidden">
              <LibraryThumb item={item} />
              <div className="flex flex-1 flex-col gap-1 px-3 py-2.5">
                <span className="font-semibold">{item.name}</span>
                <span className="k">{t(m.galleryKindVersion, { kind: kindLabel(t, item.kind), version: item.version })}</span>
                {fields.length ? <span className="k text-warn-fg">{t(m.galleryNeeds, { fields: fields.join(', ') })}</span> : null}
                {copy ? (
                  <span className={copy.version < copy.latest ? 'k text-warn-fg' : 'k'}>
                    {copy.version < copy.latest ? t(m.galleryCopyBehind, { version: copy.version }) : t(m.galleryCopyCurrent, { version: copy.version })}
                    {copy.version < copy.latest && agency?.home ? (
                      <>
                        {' '}
                        <Link to="/w/$workspace/agency/library" params={{ workspace: agency.home }} search={{ item: item.id } as never}>
                          {t(m.gallerySeeDiff)}
                        </Link>
                      </>
                    ) : null}
                  </span>
                ) : null}
                <Button
                  size="sm"
                  className="mt-auto self-start"
                  loading={busy === item.id}
                  disabledReason={!canEdit ? t(m.galleryCantEdit) : reason}
                  onClick={() => void use(item)}
                >
                  {t(m.galleryUse)}
                </Button>
              </div>
            </article>
          )
        })}
      </div>
      {!all && d.items.length > 8 ? (
        <Button size="sm" className="self-start" onClick={() => setAll(true)}>
          {t(m.galleryShowAll, { n: d.items.length })}
        </Button>
      ) : null}
      <Note tone="info">
        <T msg={m.galleryNote} values={{ workspace: me.workspace.name, permission: 'templates:edit', mono }} />
      </Note>
    </section>
  )
}

/**
 * "Add to Northwind library" in a client's composer (Templates tab): one of
 * the workspace's saved sections or campaign templates goes to the agency's
 * library, as a new item or a new version of one. Needs agency:library.
 */
export function AddToAgencyLibrary({ sections, templates }: { sections: Schemas['EmailTemplateSummary'][]; templates: Schemas['EmailTemplateSummary'][] }) {
  const t = useT()
  const staff = useStaffInClient()
  const can = useAgencyCan('agency:library')
  const { data } = useWorkspaceAgency()
  const [open, setOpen] = useState(false)
  const own = [...sections, ...templates].filter((x) => !x.builtin)
  if (!staff || !can || own.length === 0) return null
  return (
    <>
      <button type="button" className="btn sm self-start" onClick={() => setOpen(true)}>
        {t(m.addToLibrary, { agency: data?.agency ?? '' })}
      </button>
      {open ? <AddDialog own={own} agency={data?.agency ?? ''} onClose={() => setOpen(false)} /> : null}
    </>
  )
}

function AddDialog({ own, agency, onClose }: { own: Schemas['EmailTemplateSummary'][]; agency: string; onClose: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const { slug } = useWorkspace()
  const [id, setId] = useState(own[0]!.id)
  const picked = own.find((x) => x.id === id) ?? own[0]!
  const [name, setName] = useState(picked.name)
  const [versionOf, setVersionOf] = useState('')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const lib = $api.useQuery('get', '/api/v1/workspaces/{workspace}/agency/library', { params: { path: { workspace: slug }, query: { kind: 'composer' } } }, { staleTime: 60_000 })
  const sameKind = (lib.data?.items ?? []).filter((i) => i.kind === picked.kind)
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/agency/library', {
        params: { path: { workspace: slug } },
        body: { kind: picked.kind, id: picked.id, name: versionOf ? undefined : name.trim() || undefined, newVersionOf: versionOf || undefined, note: note.trim() || undefined },
      })
      void qc.invalidateQueries({ queryKey: galleryKey })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/agency/library'] })
      const target = versionOf ? (sameKind.find((i) => i.id === versionOf)?.name ?? name) : name.trim() || picked.name
      toast({ kind: 'success', title: t(m.addedToLibrary, { name: target, agency, version: data?.version ?? 1 }) })
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
      title={t(m.addToLibrary, { agency })}
      description={t(m.addToLibraryBody, { agency })}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!versionOf && !name.trim()} onClick={save}>
            {versionOf ? t(m.addAsVersion) : t(m.addItem)}
          </Button>
        </>
      }
    >
      <Field label={t(m.addWhat)}>
        <Select
          value={id}
          onChange={(e) => {
            const next = own.find((x) => x.id === e.target.value)
            setId(e.target.value)
            if (next) setName(next.name)
            setVersionOf('')
          }}
        >
          {own.map((x) => (
            <option key={x.id} value={x.id}>
              {t(m.addOption, { name: x.name, kind: kindLabel(t, x.kind) })}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t(m.addAs)}>
        <Select value={versionOf} onChange={(e) => setVersionOf(e.target.value)}>
          <option value="">{t(m.addAsNew)}</option>
          {sameKind.map((i) => (
            <option key={i.id} value={i.id}>
              {t(m.addAsVersionOf, { name: i.name, version: i.version + 1 })}
            </option>
          ))}
        </Select>
      </Field>
      {versionOf ? null : (
        <Field label={t(m.addName)} error={error}>
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={120} />
        </Field>
      )}
      <Field label={t(m.addNote)} optional error={versionOf ? error : undefined}>
        <Input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} placeholder={t(m.addNotePlaceholder)} />
      </Field>
    </Dialog>
  )
}
