import { DropdownMenu } from 'radix-ui'
import { useState } from 'react'
import { errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { LoadFailed } from '@/components/ui/LoadFailed'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago } from '@/lib/format'
import { useDebounced } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { deleteThread, updateThread, useAssistantModels, useRefreshThreads, useThreads, type ThreadListItem } from './api'
import { AsstIcon } from './icons'
import { m } from './messages'

/**
 * The person's threads (AssistantPanel › Threads): search by title, open
 * one, rename or delete it. Only they can read them.
 */
export function ThreadList({ current, onOpen, onDeleted }: { current?: string; onOpen: (id: string) => void; onDeleted: (id: string) => void }) {
  const t = useT()
  const [q, setQ] = useState('')
  const term = useDebounced(q.trim(), 250)
  const list = useThreads(term, true)
  const models = useAssistantModels()
  const modelName = (id?: string) => {
    const x = models.data?.models.find((mm) => mm.id === id)
    return x ? x.modelName || x.model : undefined
  }
  const [renaming, setRenaming] = useState<ThreadListItem>()
  const [deleting, setDeleting] = useState<ThreadListItem>()
  const items = list.data?.items ?? []
  return (
    <div className="asst-threads">
      <label className="asst-search">
        <span className="vh">{t(m.search)}</span>
        <input type="search" className="in" placeholder={t(m.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
      </label>
      {list.isError ? (
        <LoadFailed title={t(m.threadsFailed)} onRetry={() => void list.refetch()} />
      ) : list.isPending ? (
        <div className="asst-threadlist" aria-hidden="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="asst-thread-sk">
              <Skeleton w="70%" h={12} />
              <Skeleton w="45%" h={10} />
            </div>
          ))}
        </div>
      ) : items.length === 0 ? (
        <p className="k asst-threads-empty">{term ? t(m.noMatch, { q: term }) : t(m.noThreads)}</p>
      ) : (
        <ul className="asst-threadlist">
          {items.map((th) => {
            const title = th.title || t(m.untitled)
            const model = modelName(th.modelId)
            const when = ago(th.lastActivityAt)
            return (
              <li key={th.id} className="asst-thread-row">
                <button type="button" className="asst-thread" aria-current={th.id === current ? 'true' : undefined} onClick={() => onOpen(th.id)}>
                  <strong>{title}</strong>
                  <small>
                    {model ? t(m.threadMeta, { when, model }) : t(m.threadMetaNoModel, { when })}
                    {th.state === 'expired' ? (
                      <>
                        {' '}
                        <Badge tone="neutral" size="sm">
                          {t(m.expiredBadge)}
                        </Badge>
                      </>
                    ) : null}
                  </small>
                </button>
                <DropdownMenu.Root modal={false}>
                  <DropdownMenu.Trigger asChild>
                    <button type="button" className="asst-iconbtn" aria-label={t(m.threadActions, { title })}>
                      <AsstIcon name="more" />
                    </button>
                  </DropdownMenu.Trigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.Content className="menu asst-row-menu" align="end" sideOffset={4} collisionPadding={12}>
                      <DropdownMenu.Item onSelect={() => setRenaming(th)}>{t(m.rename)}</DropdownMenu.Item>
                      <DropdownMenu.Item className="text-bad-fg" onSelect={() => setDeleting(th)}>
                        {t(m.delete)}
                      </DropdownMenu.Item>
                    </DropdownMenu.Content>
                  </DropdownMenu.Portal>
                </DropdownMenu.Root>
              </li>
            )
          })}
        </ul>
      )}
      <small className="k">{t(m.privacy)}</small>
      {renaming ? <RenameDialog thread={renaming} onClose={() => setRenaming(undefined)} /> : null}
      {deleting ? <DeleteDialog thread={deleting} onClose={() => setDeleting(undefined)} onDeleted={onDeleted} /> : null}
    </div>
  )
}

function RenameDialog({ thread, onClose }: { thread: ThreadListItem; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshThreads()
  const [title, setTitle] = useState(thread.title)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await updateThread(slug, thread.id, { title: title.trim() })
      await refresh()
      toast({ kind: 'success', title: t(m.renamed) })
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
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
      title={t(m.renameTitle)}
      size="sm"
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
        <Field label={t(m.titleLabel)} error={error}>
          <Input value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} autoFocus />
        </Field>
      </form>
    </Dialog>
  )
}

function DeleteDialog({ thread, onClose, onDeleted }: { thread: ThreadListItem; onClose: () => void; onDeleted: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshThreads()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
      tone="bad"
      badge={t(m.deleteBadge)}
      title={t(m.deleteTitle, { title: thread.title || t(m.untitled) })}
      description={t(m.deleteBody)}
      confirmLabel={t(m.deleteBadge)}
      onConfirm={async () => {
        await deleteThread(slug, thread.id)
        await refresh()
        onDeleted(thread.id)
        toast({ kind: 'deleted', title: t(m.deleted) })
        onClose()
      }}
    />
  )
}
