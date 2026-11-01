import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { BulkResultDialog, ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { FollowUpDialog } from './followup'
import { mm } from './manage.messages'
import { m } from './messages'

type Item = Schemas['CampaignListItem']
type BulkAction = Schemas['CampaignBulkInput']['action']

/** Refetches every campaigns query: the list, its counts, tags, the calendar and Overview's "Recent". */
export function useRefreshCampaigns() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => /\/campaigns|\/overview|\/approvals/.test(String((q.queryKey as unknown[])[1] ?? '')),
    })
}

const tagPattern = /^[\p{Ll}\p{N}][\p{Ll}\p{N} _./-]{0,39}$/u

/** A tag as the server keeps it: lower case, single spaces (manage.go NormalizeTag). */
export function normalizeTag(raw: string) {
  return raw.trim().split(/\s+/).join(' ').toLowerCase()
}

/** The campaign's tags, after its name (CampaignsComplete board, strip 2). */
export function TagPills({ tags }: { tags: string[] }) {
  if (!tags.length) return null
  return (
    <span className="ml-1.5 inline-flex flex-wrap gap-1 align-middle">
      {tags.map((tag) => (
        <span key={tag} className="badge pur sm">
          {tag}
        </span>
      ))}
    </span>
  )
}

/** Was anything delivered? A cancelled or failed campaign that sent is archived, else deleted. */
function sentAny(c: Item) {
  return c.status === 'sent' || c.delivered != null
}

export function canDelete(c: Item) {
  return ['draft', 'approval', 'approved'].includes(c.status) || (['cancelled', 'failed'].includes(c.status) && !sentAny(c))
}

export function canArchive(c: Item) {
  return c.status === 'sent' || (['cancelled', 'failed'].includes(c.status) && sentAny(c))
}

/** Restores a deleted campaign. */
async function restore(slug: string, c: { id: string; name: string }) {
  await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/restore', { params: { path: { workspace: slug, campaignId: c.id } } })
}

/**
 * A row's ⋯ menu (CampaignsComplete board, strip 1): what fits the
 * campaign's state, the destructive item last and apart.
 */
export function RowMenu({ c, onAction }: { c: Item; onAction: (a: RowAction) => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const params = { workspace: slug, campaignId: c.id }
  const editReason = usePermissionReason('campaigns:edit')
  const email = c.channel === 'email'
  const items: ReactNode[] = []
  if (c.archivedAt) {
    items.push(
      <MenuItem key="report" onSelect={() => void navigate({ to: '/w/$workspace/campaigns/$campaignId/report', params })}>
        {tr(m.report)}
      </MenuItem>,
      <MenuSeparator key="s" />,
      <MenuItem key="unarchive" disabled={!!editReason} onSelect={() => onAction({ kind: 'unarchive', c })}>
        {tr(mm.unarchive)}
      </MenuItem>,
    )
  } else {
    if (['draft', 'approval', 'approved', 'scheduled'].includes(c.status) && email) {
      items.push(
        <MenuItem key="edit" onSelect={() => void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params })}>
          {tr(m.edit)}
        </MenuItem>,
      )
    }
    if (['sent', 'cancelled', 'failed'].includes(c.status) && email) {
      items.push(
        <MenuItem key="report" onSelect={() => void navigate({ to: '/w/$workspace/campaigns/$campaignId/report', params })}>
          {tr(m.report)}
        </MenuItem>,
      )
    }
    if (email) {
      items.push(
        <MenuItem key="dup" disabled={!!editReason} onSelect={() => void navigate({ to: '/w/$workspace/campaigns/new', params: { workspace: slug }, search: { from: c.id } })}>
          {tr(m.duplicate)}
        </MenuItem>,
      )
    }
    if (c.status === 'sent' && email) {
      items.push(
        <MenuItem key="follow" disabled={!!editReason} onSelect={() => onAction({ kind: 'followUp', c })}>
          {tr(mm.followUp)}
        </MenuItem>,
      )
    }
    items.push(
      <MenuItem key="tag" disabled={!!editReason} onSelect={() => onAction({ kind: 'tag', c })}>
        {tr(mm.addTag)}
      </MenuItem>,
    )
    if (canArchive(c)) {
      items.push(
        <MenuSeparator key="s" />,
        <MenuItem key="archive" disabled={!!editReason} onSelect={() => onAction({ kind: 'archive', c })}>
          {tr(mm.archive)}
        </MenuItem>,
      )
    } else if (canDelete(c)) {
      items.push(
        <MenuSeparator key="s" />,
        <MenuItem key="delete" className="text-bad-fg" disabled={!!editReason} onSelect={() => onAction({ kind: 'delete', c })}>
          {tr(mm.delete)}
        </MenuItem>,
      )
    }
  }
  return (
    <Menu
      label={tr(mm.moreFor, { name: c.name })}
      trigger={
        <button type="button" className="btn sm" aria-label={tr(mm.moreFor, { name: c.name })} aria-haspopup="menu">
          <span aria-hidden="true">⋯</span>
        </button>
      }
    >
      {items}
    </Menu>
  )
}

export type RowAction = { kind: 'delete' | 'archive' | 'unarchive' | 'tag' | 'followUp'; c: Item }

/** The dialogs a row's menu opens. */
export function RowActionDialogs({ action, onClose }: { action?: RowAction; onClose: () => void }) {
  if (!action) return null
  const { c } = action
  switch (action.kind) {
    case 'delete':
      return <DeleteDialog c={c} onClose={onClose} />
    case 'archive':
      return <ArchiveDialog c={c} onClose={onClose} />
    case 'unarchive':
      return <UnarchiveNow c={c} onClose={onClose} />
    case 'tag':
      return <TagDialog ids={[c.id]} names={[c.name]} mode="tag" onClose={onClose} />
    case 'followUp':
      return <FollowUpDialog campaign={{ id: c.id, name: c.name, subject: c.subject }} kind="non_openers" onClose={onClose} />
  }
}

function DeleteDialog({ c, onClose }: { c: Item; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  const approval = c.status === 'approval' || c.status === 'approved'
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={tr(mm.deleteBadge)}
      title={tr(mm.deleteTitle, { name: c.name })}
      description={approval ? `${tr(mm.deleteBody)} ${tr(mm.deleteApproval)}` : tr(mm.deleteBody)}
      confirmLabel={tr(mm.deleteConfirm)}
      busyLabel={tr(mm.deleting)}
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: c.id } } })
        await refresh()
        onClose()
        toast({
          kind: 'deleted',
          title: t(mm.deleted, { name: c.name }),
          undo: () =>
            void restore(slug, c)
              .then(() => refresh())
              .then(() => toast({ kind: 'success', title: t(mm.restored, { name: c.name }) }))
              .catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e) })),
        })
      }}
    />
  )
}

function ArchiveDialog({ c, onClose }: { c: Item; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="info"
      badge={tr(mm.archiveBadge)}
      title={tr(mm.archiveTitle, { name: c.name })}
      description={tr(mm.archiveBody)}
      confirmLabel={tr(mm.archive)}
      busyLabel={tr(mm.archiving)}
      onConfirm={async () => {
        await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/archive', { params: { path: { workspace: slug, campaignId: c.id } } })
        await refresh()
        onClose()
        toast({
          kind: 'success',
          title: t(mm.archived, { name: c.name }),
          undo: () =>
            void http
              .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unarchive', { params: { path: { workspace: slug, campaignId: c.id } } })
              .then(() => refresh())
              .catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e) })),
        })
      }}
    />
  )
}

/** Unarchiving needs no confirmation: it only puts the campaign back on the list. */
function UnarchiveNow({ c, onClose }: { c: Item; onClose: () => void }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  const started = useRef(false)
  useEffect(() => {
    if (started.current) return
    started.current = true
    void http
      .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unarchive', { params: { path: { workspace: slug, campaignId: c.id } } })
      .then(() => refresh())
      .then(() => toast({ kind: 'success', title: t(mm.unarchived, { name: c.name }) }))
      .catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e) }))
      .finally(onClose)
  }, [c, slug, refresh, onClose])
  return null
}

/** The Deleted tab's row action: bring it back. */
export function RestoreButton({ c }: { c: Item }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  const reason = usePermissionReason('campaigns:edit')
  const [busy, setBusy] = useState(false)
  return (
    <Button
      size="sm"
      disabledReason={reason}
      loading={busy}
      onClick={async () => {
        setBusy(true)
        try {
          await restore(slug, c)
          await refresh()
          toast({ kind: 'success', title: t(mm.restored, { name: c.name }) })
        } catch (e) {
          toast({ kind: 'error', title: errorMessage(e) })
        } finally {
          setBusy(false)
        }
      }}
    >
      {tr(mm.restore)}
    </Button>
  )
}

/** The Deleted tab's line under a name: until when it can come back. */
export function deletedLine(c: Item) {
  if (!c.deletedAt) return ''
  return t(mm.deletedUntil, { date: shortDate(new Date(new Date(c.deletedAt).getTime() + 30 * 24 * 3600_000)) })
}

/** A scheduled row's Unschedule: back to a draft, where it is (CA-2). */
export function UnscheduleButton({ c }: { c: Item }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  const reason = usePermissionReason('campaigns:edit')
  const [busy, setBusy] = useState(false)
  return (
    <Button
      size="sm"
      disabledReason={reason}
      loading={busy}
      onClick={async () => {
        setBusy(true)
        try {
          await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', { params: { path: { workspace: slug, campaignId: c.id } } })
          await refresh()
          toast({ kind: 'success', title: t(mm.unscheduled, { name: c.name }) })
        } catch (e) {
          toast({ kind: 'error', title: errorMessage(e) })
        } finally {
          setBusy(false)
        }
      }}
    >
      {tr(mm.unschedule)}
    </Button>
  )
}

/** Adds or removes a tag on one or more campaigns; existing tags are suggested. */
export function TagDialog({ ids, names, mode, onClose, onDone }: { ids: string[]; names: string[]; mode: 'tag' | 'untag'; onClose: () => void; onDone?: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const refresh = useRefreshCampaigns()
  const tags = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/tags', { params: { path: { workspace: slug } } })
  const [value, setValue] = useState('')
  const [err, setErr] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<Schemas['CampaignBulkResult']>()
  const save = async () => {
    const tag = normalizeTag(value)
    if (!tagPattern.test(tag)) {
      setErr(tr(mm.tagInvalid))
      return
    }
    setBusy(true)
    setErr(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/bulk', { params: { path: { workspace: slug } }, body: { ids, action: mode, tag } })
      await refresh()
      if (data && data.skipped.length > 0) {
        // The selection stays until the result is read (onDone unmounts the bar).
        setResult(data)
        return
      }
      onDone?.()
      onClose()
      toast({
        kind: 'success',
        title: ids.length === 1 && mode === 'tag' ? t(mm.tagged, { name: names[0] ?? '', tag }) : t(mm.bulkDone, { action: mode, count: data?.done ?? ids.length }),
      })
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  if (result)
    return (
      <BulkResult
        action={mode}
        result={result}
        onClose={() => {
          onDone?.()
          onClose()
        }}
      />
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(mode === 'tag' ? mm.tagTitle : mm.untagTitle, { count: ids.length })}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {tr(mode === 'tag' ? mm.tagSave : mm.untagSave)}
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
          <label className="lbl" htmlFor={`${id}-tag`}>
            {tr(mm.tagName)}
          </label>
          <input
            id={`${id}-tag`}
            className="in"
            list={`${id}-tags`}
            value={value}
            maxLength={40}
            autoComplete="off"
            aria-invalid={err ? true : undefined}
            aria-describedby={`${id}-hint`}
            onChange={(e) => setValue(e.target.value)}
          />
          <datalist id={`${id}-tags`}>
            {(tags.data?.tags ?? []).map((tag) => (
              <option key={tag} value={tag} />
            ))}
          </datalist>
          <span className="k" id={`${id}-hint`}>
            {tr(mm.tagHint)}
          </span>
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}

/** What a bulk action did, when it skipped some: why, per the first reason. */
function BulkResult({ action, result, onClose }: { action: BulkAction; result: Schemas['CampaignBulkResult']; onClose: () => void }) {
  const tr = useT()
  const reasons = [...new Set(result.skipped.map((s) => s.reason))]
  return (
    <BulkResultDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={tr(mm.bulkResultTitle, { action })}
      done={{ count: result.done, text: tr(mm.bulkDone, { action, count: result.done }) }}
      skipped={{
        count: result.skipped.length,
        text: (
          <>
            {tr(mm.bulkSkippedText, { count: result.skipped.length })}
            <ul className="m-0 mt-1 flex list-none flex-col gap-0.5 p-0">
              {result.skipped.slice(0, 5).map((s) => (
                <li key={s.id} className="k">
                  {s.name}: {s.reason}
                </li>
              ))}
            </ul>
            {result.skipped.length > 5 && reasons.length === 1 ? <span className="k">{reasons[0]}</span> : null}
          </>
        ),
      }}
    />
  )
}

/**
 * The bulk bar over the selected rows (CampaignsComplete board, strip 2):
 * tags, archive and delete; Archived rows can be unarchived. Each campaign
 * is done on its own and those that can't are skipped with why.
 */
export function BulkBar({ selected, archivedTab, onClear }: { selected: Item[]; archivedTab: boolean; onClear: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCampaigns()
  const [tagging, setTagging] = useState<'tag' | 'untag'>()
  const [confirm, setConfirm] = useState<'archive' | 'delete'>()
  const [result, setResult] = useState<{ action: BulkAction; result: Schemas['CampaignBulkResult'] }>()
  const ids = selected.map((c) => c.id)
  const run = async (action: BulkAction) => {
    const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/bulk', { params: { path: { workspace: slug } }, body: { ids, action } })
    await refresh()
    if (data && data.skipped.length > 0) {
      // The bar stays, with its result, until it is read.
      setResult({ action, result: data })
      return
    }
    onClear()
    if (data) toast({ kind: action === 'delete' ? 'deleted' : 'success', title: t(mm.bulkDone, { action, count: data.done }) })
  }
  return (
    <div className="card flex flex-wrap items-center gap-2.5 border-accent-line bg-accent-tint px-3 py-2" role="region" aria-label={tr(mm.bulkLabel)}>
      <span className="text-[13px] font-semibold" role="status">
        {tr(mm.selected, { count: selected.length })}
      </span>
      <Button size="sm" onClick={() => setTagging('tag')}>
        {tr(mm.bulkAddTag)}
      </Button>
      <Button size="sm" onClick={() => setTagging('untag')}>
        {tr(mm.bulkRemoveTag)}
      </Button>
      {archivedTab ? (
        <Button size="sm" onClick={() => void run('unarchive').catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e) }))}>
          {tr(mm.unarchive)}
        </Button>
      ) : (
        <>
          <Button size="sm" onClick={() => setConfirm('archive')}>
            {tr(mm.bulkArchive)}
          </Button>
          <Button size="sm" className="dline" onClick={() => setConfirm('delete')}>
            {tr(mm.bulkDelete)}
          </Button>
        </>
      )}
      <Button size="sm" className="ml-auto" onClick={onClear}>
        {tr(mm.bulkClear)}
      </Button>
      {tagging ? <TagDialog ids={ids} names={selected.map((c) => c.name)} mode={tagging} onClose={() => setTagging(undefined)} onDone={onClear} /> : null}
      {confirm ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setConfirm(undefined)}
          tone={confirm === 'delete' ? 'bad' : 'info'}
          badge={tr(confirm === 'delete' ? mm.deleteBadge : mm.archiveBadge)}
          title={tr(confirm === 'delete' ? mm.bulkDeleteTitle : mm.bulkArchiveTitle, { count: selected.length })}
          description={tr(confirm === 'delete' ? mm.bulkDeleteBody : mm.bulkArchiveBody)}
          confirmLabel={tr(confirm === 'delete' ? mm.bulkDelete : mm.bulkArchive)}
          busyLabel={tr(confirm === 'delete' ? mm.deleting : mm.archiving)}
          onConfirm={async () => {
            await run(confirm)
            setConfirm(undefined)
          }}
        />
      ) : null}
      {result ? (
        <BulkResult
          action={result.action}
          result={result.result}
          onClose={() => {
            setResult(undefined)
            onClear()
          }}
        />
      ) : null}
    </div>
  )
}

/** The Calendar link of the page's actions. */
export function CalendarLink() {
  const tr = useT()
  const { slug } = useWorkspace()
  return (
    <Link className="btn" to="/w/$workspace/campaigns/calendar" params={{ workspace: slug }}>
      {tr(mm.calendar)}
    </Link>
  )
}
