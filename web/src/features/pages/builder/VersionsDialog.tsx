import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useBuilder } from './context'
import { m } from './messages'
import { whenText, type T } from './wording'

type Version = Schemas['PageVersionItem']

function labelOf(t: T, v: Version): string {
  switch (v.label) {
    case 'created':
      return t(m.vCreated)
    case 'autosave':
      return t(m.vAutosave)
    case 'saved':
      return t(m.vSaved)
    case 'settings':
      return t(m.vSettings)
    case 'kept':
      return t(m.vKept)
    case 'restored':
      return t(m.vRestored)
    case 'published':
      return t(m.vPublished)
  }
  return v.label
}

/**
 * Version history (PageBuilder · Versions): every save and publish, newest
 * first, with the live one and the draft marked. A version can be
 * previewed, or restored as a new version of the draft.
 */
export function VersionsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const b = useBuilder()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}/versions', { params: { path: { workspace: b.slug, page: b.page.id } } }, { enabled: open, staleTime: 0 })
  const [restore, setRestore] = useState<Version | null>(null)
  const next = Math.max(b.page.versions, b.status.version) + 1
  const doRestore = async (v: Version) => {
    try {
      // Unsaved edits go into a version of their own first: nothing is lost.
      await b.save({ manual: true, quiet: true })
      await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/versions/{version}/restore', { params: { path: { workspace: b.slug, page: b.page.id, version: v.version } } })
      await b.reloadFromServer()
      toast({ kind: 'success', title: t(m.restored, { version: v.version, next }) })
      setRestore(null)
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(m.restoreFailed), detail: errorMessage(e) })
      throw e
    }
  }
  return (
    <>
      <Dialog open={open && !restore} onOpenChange={(o) => !o && onClose()} title={t(m.versionsTitle)} description={t(m.versionsLead)} size="lg">
        {q.isPending ? (
          <p className="k">{t(m.loadingVersions)}</p>
        ) : q.isError ? (
          <p className="k">{t(m.versionsFailed)}</p>
        ) : (
          <Table caption={t(m.versionsCaption)} density="compact">
            <thead>
              <tr>
                <Th>{t(m.colVersion)}</Th>
                <Th>{t(m.colWhat)}</Th>
                <Th>{t(m.colBy)}</Th>
                <Th>{t(m.colWhen)}</Th>
                <Th>
                  <span className="vh">{t(m.colActions)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {q.data.items.map((v) => (
                <tr key={v.version}>
                  <Td num>
                    <span className="flex items-center gap-1.5">
                      v{v.version}
                      {v.live ? <Badge tone="ok">{t(m.vLive)}</Badge> : null}
                      {v.current ? <Badge tone="neutral">{t(m.vDraft)}</Badge> : null}
                    </span>
                  </Td>
                  <Td>{labelOf(t, v)}</Td>
                  <Td>{v.savedBy?.name ?? ''}</Td>
                  <Td>{whenText(t, v.savedAt)}</Td>
                  <Td align="right" nowrap>
                    <span className="inline-flex gap-1.5">
                      <Button size="sm" onClick={() => b.openDialog({ kind: 'preview', version: v.version })}>
                        {t(m.preview)}
                      </Button>
                      {b.canEdit && !v.current ? (
                        <Button size="sm" onClick={() => setRestore(v)}>
                          {t(m.restore)}
                        </Button>
                      ) : null}
                    </span>
                  </Td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
      </Dialog>
      <ConfirmDialog
        open={Boolean(restore)}
        onOpenChange={(o) => !o && setRestore(null)}
        tone="warn"
        badge={t(m.restore)}
        title={restore ? t(m.restoreTitle, { version: restore.version }) : ''}
        description={restore ? t(m.restoreLead, { version: restore.version, next }) : ''}
        confirmLabel={t(m.restore)}
        confirmVariant="primary"
        onConfirm={() => (restore ? doRestore(restore) : undefined)}
      />
    </>
  )
}
