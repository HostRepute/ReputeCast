import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, DialogOption } from '@/components/ui/ConfirmDialog'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, t, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './complete.messages'

const b = (c: ReactNode) => <b>{c}</b>

/** Refreshes the automations list and a flow's detail. */
function useRefreshFlows() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => String(q.queryKey[1] ?? '').startsWith('/api/v1/workspaces/{workspace}/automations'),
    })
}

/**
 * A flow's row menu on the Automations list (review 2026-11 AT-1): Open,
 * Duplicate, Delete…
 */
export function FlowRowMenu({ flow }: { flow: { id: string; name: string; status: string; inFlow?: number } }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshFlows()
  const canEdit = useCan('automations:edit')
  const canPublish = useCan('automations:publish')
  const [deleting, setDeleting] = useState(false)
  const duplicate = async () => {
    try {
      const out = await http.POST('/api/v1/workspaces/{workspace}/automations/{id}/duplicate', { params: { path: { workspace: slug, id: flow.id } } })
      await refresh()
      toast({ kind: 'success', title: t(m.duplicated, { name: flow.name }) })
      if (out.data)
        void navigate({
          to: '/w/$workspace/automations/$automationId',
          params: { workspace: slug, automationId: out.data.id },
        })
    } catch (e) {
      toast({
        kind: 'error',
        title: t(m.duplicateFailed),
        detail: errorMessage(e),
      })
    }
  }
  // Deleting a published flow needs the publish permission.
  const canDelete = canEdit && (flow.status === 'draft' || canPublish)
  return (
    <>
      <Menu label={tr(m.moreActions, { name: flow.name })} trigger={<Button size="sm" iconOnly icon="more" aria-label={tr(m.moreActions, { name: flow.name })} />}>
        <MenuItem
          onSelect={() =>
            void navigate({
              to: '/w/$workspace/automations/$automationId',
              params: { workspace: slug, automationId: flow.id },
            })
          }
        >
          {tr(m.open)}
        </MenuItem>
        <MenuItem disabled={!canEdit} onSelect={() => void duplicate()}>
          {tr(m.duplicate)}
        </MenuItem>
        <MenuSeparator />
        <MenuItem className="text-bad-fg" disabled={!canDelete} onSelect={() => setDeleting(true)}>
          {tr(m.deleteEllipsis)}
        </MenuItem>
      </Menu>
      {deleting ? <DeleteFlowDialog flow={flow} onClose={() => setDeleting(false)} /> : null}
    </>
  )
}

/**
 * Deleting a flow says how many contacts are in it, asks for its name when
 * it ever went live, and offers Pause instead.
 */
export function DeleteFlowDialog({
  flow,
  onClose,
  onDeleted,
}: {
  flow: { id: string; name: string; status: string; inFlow?: number }
  onClose: () => void
  onDeleted?: () => void
}) {
  const tr = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshFlows()
  const pause = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/pause')
  const n = flow.inFlow ?? 0
  const published = flow.status !== 'draft'
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={tr(m.deleteBadge)}
      title={tr(m.deleteTitle, { name: flow.name })}
      description={published ? <T msg={m.deleteLive} values={{ n, count: count(n), b }} /> : tr(m.deleteDraft)}
      confirmLabel={tr(m.deleteFlow)}
      typedConfirmation={published ? { label: tr(m.typeName, { name: flow.name }), expected: flow.name } : undefined}
      secondary={
        flow.status === 'live' ? (
          <Button
            loading={pause.isPending}
            onClick={() =>
              pause.mutate(
                {
                  params: { path: { workspace: slug, id: flow.id } },
                  body: {},
                },
                {
                  onSuccess: () => {
                    void refresh()
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {tr(m.pauseInstead)}
          </Button>
        ) : undefined
      }
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/automations/{id}', {
          params: { path: { workspace: slug, id: flow.id } },
        })
        await refresh()
        toast({ kind: 'deleted', title: t(m.deleted, { name: flow.name }) })
        onDeleted?.()
      }}
    />
  )
}

/** The editor's Discard draft: the draft goes back to the live version. */
export function DiscardDraftButton({
  id,
  changes,
  liveVersion,
  disabledReason,
  onDiscarded,
}: {
  id: string
  changes: number
  liveVersion: number
  disabledReason?: string
  onDiscarded: () => void
}) {
  const tr = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button disabledReason={disabledReason} onClick={() => setOpen(true)}>
        {tr(m.discardDraft)}
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        badge={tr(m.discardBadge)}
        title={tr(m.discardTitle, { n: changes })}
        description={tr(m.discardBody, { version: liveVersion })}
        cancelLabel={tr(m.keepEditing)}
        confirmLabel={tr(m.discardChanges)}
        onConfirm={async () => {
          await http.DELETE('/api/v1/workspaces/{workspace}/automations/{id}/draft', { params: { path: { workspace: slug, id } } })
          toast({
            kind: 'success',
            title: t(m.discarded, { version: liveVersion }),
          })
          onDiscarded()
        }}
      />
    </>
  )
}

type Overdue = 'send' | 'skip' | 'spread'

/**
 * Resuming a flow whose contacts' waits ended during the pause asks what
 * happens to them (review 2026-11 AT-4): send now, skip the email they
 * missed, or spread them over a few hours.
 */
export function ResumeDialog({ id, name, overdue, onClose, onResumed }: { id: string; name: string; overdue: number; onClose: () => void; onResumed: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/resume')
  const [choice, setChoice] = useState<Overdue>('send')
  const [hours, setHours] = useState('6')
  const h = Number(hours)
  const valid = choice !== 'spread' || (Number.isInteger(h) && h >= 1 && h <= 72)
  const option = (value: Overdue, title: ReactNode, hint: string) => (
    <DialogOption name="overdue" checked={choice === value} onChange={() => setChoice(value)} title={title} description={hint} />
  )
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="ok"
      badge={tr(m.resume)}
      title={tr(m.resumeTitle, { name })}
      description={<T msg={m.resumeOverdue} values={{ n: overdue, count: count(overdue), b }} />}
      confirmLabel={tr(m.resume)}
      onConfirm={() =>
        new Promise<void>((resolve, reject) => {
          if (!valid) return reject(new Error(t(m.spreadRange)))
          resume.mutate(
            {
              params: { path: { workspace: slug, id } },
              body: {
                overdue: choice,
                spreadHours: choice === 'spread' ? h : undefined,
              },
            },
            {
              onSuccess: () => {
                onResumed()
                resolve()
              },
              onError: (e) => reject(e),
            },
          )
        })
      }
    >
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="vh">{tr(m.resumeTitle, { name })}</legend>
        {option('send', tr(m.sendNow), tr(m.sendNowHint))}
        {option('skip', tr(m.skipOverdue), tr(m.skipOverdueHint))}
        {option(
          'spread',
          <T
            msg={m.spreadOver}
            values={{
              n: () => (
                <input
                  type="number"
                  inputMode="numeric"
                  min={1}
                  max={72}
                  aria-label={tr(m.spreadHours)}
                  className="in mx-1 inline-block h-[28px] w-14 px-1.5 py-0.5 align-baseline"
                  value={hours}
                  onFocus={() => setChoice('spread')}
                  onChange={(e) => setHours(e.target.value)}
                />
              ),
            }}
          />,
          tr(m.spreadHint, {
            perHour: count(Math.max(1, Math.ceil(overdue / Math.max(1, h || 1)))),
          }),
        )}
      </fieldset>
    </ConfirmDialog>
  )
}
