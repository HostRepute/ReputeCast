import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { affectsText, membersText, restoredText } from './listDelete'
import { m } from './lists.messages'

/**
 * Restores a deleted list (the toast's Undo, the audit entry's Restore
 * list) and says what came back on. Resolves false when it failed (the
 * error toast is shown).
 */
export function useRestoreList() {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  return useCallback(
    async (id: string, name: string) => {
      try {
        const { data } = await http.POST('/api/v1/workspaces/{workspace}/lists/{id}/restore', { params: { path: { workspace: slug, id } } })
        // Forms, automations, campaigns and every list picker change too.
        await qc.invalidateQueries()
        toast({ kind: 'success', title: t(m.restored, { name }), detail: data ? restoredText(data) : undefined })
        return true
      } catch (e) {
        toast({ kind: 'error', title: t(m.restoreFailed, { name }), detail: errorMessage(e) })
        return false
      }
    },
    [qc, slug],
  )
}

/**
 * Deleting a list (Dialogs board, "Delete list"). The toast offers Undo for
 * 10 seconds; after that the audit log restores it for 30 days.
 */
export function DeleteListDialog({ list, open, onOpenChange }: { list: { id: string; name: string }; open: boolean; onOpenChange: (open: boolean) => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const restore = useRestoreList()
  const path = { workspace: slug, id: list.id }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists/{id}/impact', { params: { path } }, { enabled: open, staleTime: 0 })
  const im = q.data
  const affects = im ? affectsText(im) : ''

  const remove = async () => {
    if (!im) throw new Error(q.isError ? errorMessage(q.error) : t(m.stillCounting, { name: list.name }))
    await http.DELETE('/api/v1/workspaces/{workspace}/lists/{id}', { params: { path } })
    await qc.invalidateQueries()
    toast({ kind: 'deleted', title: t(m.deleted, { name: list.name }), undo: () => void restore(list.id, list.name) })
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      badge={tr(m.badge)}
      title={tr(m.title, { name: list.name })}
      description={im ? membersText(im) : q.isError ? errorMessage(q.error) : tr(m.counting, { name: list.name })}
      confirmLabel={tr(m.confirm)}
      busyLabel={tr(m.deleting)}
      onConfirm={remove}
      typedConfirmation={{
        label: tr(m.typeName),
        expected: list.name,
        hint: tr(m.typeHint),
      }}
    >
      {affects ? (
        <DialogRow label={tr(m.affects)} first>
          {affects}
        </DialogRow>
      ) : null}
    </ConfirmDialog>
  )
}
