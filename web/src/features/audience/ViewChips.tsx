import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useInvalidateAudience } from './ContactDialogs'
import { parseColumns } from './ContactColumns'
import { tierColor, type Tier } from './labels'
import { m } from './contacts.messages'

type View = Schemas['ContactSavedView']
type Filter = Schemas['ContactFilter']

/**
 * The saved views row (Contacts, ContactsAtScale › Saved views): "All
 * contacts" and each view, whose chip has a menu to rename it, save the
 * current filters and columns in it, or delete it (review 2026-11 AU-8).
 */
export function ViewChips({
  views,
  active,
  filter,
  columns,
  onPick,
  onSave,
}: {
  views?: Schemas['ContactSavedViews']
  /** The view whose filters and columns are showing, if any. */
  active?: string
  filter: Filter
  columns?: string
  onPick: (filters: Filter, columns?: string) => void
  onSave: () => void
}) {
  const t = useT()
  const none = Object.keys(filter).length === 0 && !columns
  const editReason = usePermissionReason('subscribers:edit')
  const [renaming, setRenaming] = useState<View>()
  const [deleting, setDeleting] = useState<View>()
  const update = useUpdateView()
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t(m.savedViews)}>
      <span className="k mr-1">{t(m.savedViews)}</span>
      <button type="button" className="vchip" aria-pressed={none} onClick={() => onPick({})}>
        {views ? t(m.allContactsCount, { count: count(views.all) }) : t(m.allContacts)}
      </button>
      {views?.items.map((v) => {
        const tier = v.filters.tier as Tier | undefined
        const onlyTier = tier && Object.keys(v.filters).length === 1
        const on = active === v.id
        return (
          <span key={v.id} className="vchip-wrap" data-on={on || undefined}>
            <button type="button" className="vchip" aria-pressed={on} onClick={() => onPick(v.filters, v.columns.length ? v.columns.join(',') : undefined)}>
              {onlyTier ? <span className="dot" style={{ color: tierColor[tier] }} aria-hidden="true" /> : null}
              {v.name} · {count(v.count)}
            </button>
            {editReason ? null : (
              <Menu
                align="start"
                label={t(m.viewMenu, { name: v.name })}
                trigger={
                  <button type="button" className="vchip-more" aria-label={t(m.viewMenu, { name: v.name })}>
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                      <circle cx="3" cy="8" r="1.4" />
                      <circle cx="8" cy="8" r="1.4" />
                      <circle cx="13" cy="8" r="1.4" />
                    </svg>
                  </button>
                }
              >
                <MenuItem onSelect={() => setRenaming(v)}>{t(m.renameView)}</MenuItem>
                <MenuItem
                  onSelect={() =>
                    void update(v, { filters: filter, columns: parseColumns(columns) ?? [] }).then(
                      (ok) => ok && toast({ kind: 'success', title: t(m.viewUpdated, { name: v.name }) }),
                    )
                  }
                >
                  {t(m.saveIntoView)}
                </MenuItem>
                <MenuSeparator />
                <MenuItem className="text-bad-fg" onSelect={() => setDeleting(v)}>
                  {t(m.deleteView)}
                </MenuItem>
              </Menu>
            )}
          </span>
        )
      })}
      <button type="button" className="vchip dashed" onClick={onSave}>
        {t(m.saveCurrent)}
      </button>
      {renaming ? <RenameView view={renaming} onClose={() => setRenaming(undefined)} /> : null}
      {deleting ? <DeleteView view={deleting} onClose={() => setDeleting(undefined)} /> : null}
    </div>
  )
}

/** PATCHes a view; toasts the error and says whether it worked. */
function useUpdateView() {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const patch = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/contacts/views/{id}')
  return async (v: View, body: Schemas['ContactViewUpdate']) => {
    try {
      await patch.mutateAsync({ params: { path: { workspace: slug, id: v.id } }, body })
      await invalidate()
      return true
    } catch (e) {
      toast({ kind: 'error', title: t(m.viewUpdateFailed), detail: errorMessage(e) })
      return false
    }
  }
}

function RenameView({ view, onClose }: { view: View; onClose: () => void }) {
  const t = useT()
  const [name, setName] = useState(view.name)
  const [err, setErr] = useState<string>()
  const [busy, setBusy] = useState(false)
  const update = useUpdateView()
  const save = async () => {
    if (!name.trim()) {
      setErr(t(m.nameView))
      return
    }
    setBusy(true)
    const ok = await update(view, { name: name.trim() })
    setBusy(false)
    if (ok) {
      toast({ kind: 'success', title: t(m.viewRenamed) })
      onClose()
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={t(m.renameViewTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {t(common.save)}
          </Button>
        </>
      }
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          void save()
        }}
      >
        <Field label={t(m.viewName)} error={err}>
          <Input value={name} maxLength={80} autoFocus onChange={(e) => setName(e.target.value)} />
        </Field>
      </form>
    </Dialog>
  )
}

function DeleteView({ view, onClose }: { view: View; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const del = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/contacts/views/{id}')
  const [err, setErr] = useState<string>()
  const remove = async () => {
    try {
      await del.mutateAsync({ params: { path: { workspace: slug, id: view.id } } })
      await invalidate()
      toast({ kind: 'deleted', title: t(m.viewDeleted) })
      onClose()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={t(m.deleteViewTitle, { name: view.name })}
      description={view.shared ? t(m.deleteViewBody) : t(m.deleteViewPersonal)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="danger-primary" loading={del.isPending} onClick={() => void remove()}>
            {t(m.deleteView)}
          </Button>
        </>
      }
    >
      {err ? <p className="m-0 text-[13px] font-semibold text-bad-fg">{err}</p> : null}
    </Dialog>
  )
}
