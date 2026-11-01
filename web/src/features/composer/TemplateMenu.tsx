import { useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useComposer } from './context'
import { mt } from './templates.messages'

export interface SavedTemplate {
  id: string
  name: string
  kind: 'template' | 'section'
  description?: string
}

type Open = 'rename' | 'replace' | 'delete'

function useRefreshTemplates() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/templates'] })
}

/**
 * A saved template's or section's ⋯ menu (review 2026-11 CA-8): rename it,
 * replace a template with the email open in the composer, or delete it.
 */
export function TemplateMenu({ tpl, className }: { tpl: SavedTemplate; className?: string }) {
  const tr = useT()
  const c = useComposer()
  const [open, setOpen] = useState<Open>()
  if (!c.canEdit || !c.campaign.access.canEditTemplates) return null
  return (
    <>
      <Menu
        label={tr(mt.moreFor, { name: tpl.name })}
        trigger={
          <button type="button" className={`btn xs ic ${className ?? ''}`} aria-label={tr(mt.moreFor, { name: tpl.name })} aria-haspopup="menu">
            <span aria-hidden="true">⋯</span>
          </button>
        }
      >
        <MenuItem onSelect={() => setOpen('rename')}>{tr(mt.rename)}</MenuItem>
        {tpl.kind === 'template' ? (
          <MenuItem disabled={!c.editor} onSelect={() => setOpen('replace')}>
            {tr(mt.replace)}
          </MenuItem>
        ) : null}
        <MenuSeparator />
        <MenuItem className="text-bad-fg" onSelect={() => setOpen('delete')}>
          {tr(mt.delete)}
        </MenuItem>
      </Menu>
      {open === 'rename' ? <RenameDialog tpl={tpl} onClose={() => setOpen(undefined)} /> : null}
      {open === 'replace' ? <ReplaceDialog tpl={tpl} onClose={() => setOpen(undefined)} /> : null}
      {open === 'delete' ? <DeleteDialog tpl={tpl} onClose={() => setOpen(undefined)} /> : null}
    </>
  )
}

/** Saves the template under a new name, its content as stored. */
function RenameDialog({ tpl, onClose }: { tpl: SavedTemplate; onClose: () => void }) {
  const tr = useT()
  const c = useComposer()
  const id = useId()
  const refresh = useRefreshTemplates()
  const [name, setName] = useState(tpl.name)
  const [description, setDescription] = useState(tpl.description ?? '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const save = async () => {
    if (!name.trim()) {
      setErr(tr(common.required))
      return
    }
    setBusy(true)
    setErr(undefined)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: c.slug, templateId: tpl.id } } })
      if (!data) return
      await http.PUT('/api/v1/workspaces/{workspace}/templates/{templateId}', {
        params: { path: { workspace: c.slug, templateId: tpl.id } },
        body: { kind: tpl.kind, name: name.trim(), description: description.trim() || undefined, mjml: data.mjml, html: data.html || undefined },
      })
      await refresh()
      toast({ kind: 'success', title: t(mt.renamed, { name: name.trim() }) })
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
      title={tr(mt.renameTitle, { name: tpl.name })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {tr(common.save)}
          </Button>
        </>
      }
    >
      <form
        noValidate
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <div className="field">
          <label className="lbl" htmlFor={`${id}-name`}>
            {tr(mt.name)}
          </label>
          <input id={`${id}-name`} className="in" value={name} maxLength={120} aria-invalid={err && !name.trim() ? true : undefined} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="field">
          <label className="lbl" htmlFor={`${id}-desc`}>
            {tr(mt.description)}
          </label>
          <input id={`${id}-desc`} className="in" value={description} maxLength={300} onChange={(e) => setDescription(e.target.value)} />
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}

/** The template takes the email open now: its MJML and the HTML compiled from it. */
function ReplaceDialog({ tpl, onClose }: { tpl: SavedTemplate; onClose: () => void }) {
  const tr = useT()
  const c = useComposer()
  const refresh = useRefreshTemplates()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={tr(mt.replaceBadge)}
      title={tr(mt.replaceTitle, { name: tpl.name })}
      description={tr(mt.replaceBody)}
      confirmLabel={tr(mt.replaceConfirm)}
      busyLabel={tr(mt.replacing)}
      onConfirm={async () => {
        await http.PUT('/api/v1/workspaces/{workspace}/templates/{templateId}', {
          params: { path: { workspace: c.slug, templateId: tpl.id } },
          body: { kind: tpl.kind, name: tpl.name, description: tpl.description || undefined, mjml: c.currentMjml(), html: c.compileCurrent()?.html },
        })
        await refresh()
        toast({ kind: 'success', title: t(mt.replaced, { name: tpl.name }) })
        onClose()
      }}
    />
  )
}

function DeleteDialog({ tpl, onClose }: { tpl: SavedTemplate; onClose: () => void }) {
  const tr = useT()
  const c = useComposer()
  const refresh = useRefreshTemplates()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={tr(mt.deleteBadge)}
      title={tr(mt.deleteTitle, { name: tpl.name })}
      description={tr(tpl.kind === 'section' ? mt.deleteSectionBody : mt.deleteTemplateBody)}
      confirmLabel={tr(mt.deleteConfirm)}
      busyLabel={tr(mt.deleting)}
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: c.slug, templateId: tpl.id } } })
        await refresh()
        toast({ kind: 'deleted', title: t(mt.deleted, { name: tpl.name }) })
        onClose()
      }}
    />
  )
}

/** Matches a name to the gallery's search, ignoring case and accents. */
export function matchesSearch(name: string, q: string) {
  if (!q.trim()) return true
  const fold = (s: string) =>
    s
      .normalize('NFD')
      .replace(/\p{M}/gu, '')
      .toLowerCase()
  return fold(name).includes(fold(q.trim()))
}

