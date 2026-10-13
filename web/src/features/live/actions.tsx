import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { ConfirmDialog, DialogOption } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { useCan } from '@/lib/session'
import { mailboxName, needs, queuedCount, type Incident, type LiveAction, type Snapshot } from './model'

type Result = Schemas['LiveActionResult']

const base = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/live' as const

/**
 * The live monitor's actions. Each returns the send's new snapshot, which
 * replaces the one on screen at once; lists that show the campaign's
 * status are refreshed. Errors are toasted and rethrown (dialogs keep the
 * message open next to their buttons).
 */
export function useLiveActions(slug: string, campaignId: string, apply: (s: Snapshot) => void) {
  const qc = useQueryClient()
  const path = { workspace: slug, campaignId }
  const pauseM = $api.useMutation('post', `${base}/pause`)
  const resumeM = $api.useMutation('post', `${base}/resume`)
  const cancelM = $api.useMutation('post', `${base}/cancel`)
  const moveM = $api.useMutation('post', `${base}/move`)
  const retryM = $api.useMutation('post', `${base}/retry`)
  const pending = pauseM.isPending || resumeM.isPending || cancelM.isPending || moveM.isPending || retryM.isPending

  const settle = (r: Result) => {
    apply(r.snapshot)
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
    return r
  }
  const fail = (title: string) => (e: unknown) => {
    toast({ kind: 'error', title, detail: `${errorMessage(e)} Nothing was changed.` })
    throw e
  }

  return {
    pending,
    pause: (reason?: string) =>
      pauseM
        .mutateAsync({ params: { path }, body: reason ? { reason } : {} })
        .then(settle)
        .catch(fail("Couldn't pause the send")),
    resume: (body: Schemas['LiveResumeRequest'] = {}) =>
      resumeM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          if (body.withChanges && r.excluded > 0) {
            const who = body.mailbox ? mailboxName(body.mailbox) : 'the held share'
            toast({ kind: 'success', title: `${who === 'the held share' ? 'Resumed' : `${who} resumed`} with changes`, detail: `${count(r.excluded)} contacts left out of the rest of this send.` })
          }
          return r
        })
        .catch(fail("Couldn't resume the send")),
    cancel: (body: Schemas['LiveCancelRequest'] = {}) =>
      cancelM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          if (body.mailbox || body.providerId) {
            toast({ kind: 'deleted', title: 'Share cancelled', detail: `${count(r.cancelled)} recipients won't be sent; they're marked "not sent" in the report.` })
          }
          return r
        })
        .catch(fail("Couldn't cancel")),
    move: (body: Schemas['LiveMoveRequest'], target: string) =>
      moveM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          toast({ kind: 'success', title: `Moved ${count(r.moved)} recipients to ${target}`, detail: 'They joined its queue; what was already sent keeps its tracking.' })
          return r
        })
        .catch(fail("Couldn't move the share")),
    retry: (body: Schemas['LiveRetryRequest']) =>
      retryM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .catch(fail("Couldn't retry")),
  }
}

export type LiveActions = ReturnType<typeof useLiveActions>

/** Runs what an incident option does (model.optionAction). */
export function runAction(actions: LiveActions, a: LiveAction, target?: string): Promise<unknown> {
  switch (a.kind) {
    case 'resume':
      return actions.resume(a.body)
    case 'cancel':
      return actions.cancel(a.body)
    case 'move':
      return actions.move(a.body, target ?? 'the other provider')
    case 'retry':
      return actions.retry(a.body)
  }
}

/** Who may do what on the monitor, and why not (add-on §6: pause needs campaigns:pause; resume, move and cancel need campaigns:send). */
export function useLivePermissions(helpers: Schemas['LiveHelpers'] | undefined) {
  const canPause = useCan('campaigns:pause')
  const canSend = useCan('campaigns:send')
  return {
    pause: canPause ? undefined : needs('Pausing a send', 'campaigns:pause', helpers?.pause),
    resume: canSend ? undefined : needs('Resuming a send', 'campaigns:send', helpers?.send),
    cancel: canSend ? undefined : needs('Cancelling a send', 'campaigns:send', helpers?.send),
    decide: canSend ? undefined : needs('Deciding how a held send continues', 'campaigns:send', helpers?.send),
  }
}

export type LivePermissions = ReturnType<typeof useLivePermissions>

/**
 * "Cancel the rest of Weekly Digest #142?" (Dialogs board): cancel what is
 * queued, or pause instead.
 */
export function CancelSendDialog({
  open,
  onOpenChange,
  s,
  incident,
  actions,
  canPause,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  s: Snapshot
  incident?: Incident
  actions: LiveActions
  canPause: boolean
}) {
  const [choice, setChoice] = useState<'cancel' | 'pause'>('cancel')
  const remaining = queuedCount(s, incident) + (incident ? s.held : 0)
  const pausable = canPause && s.options.canPause
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(o) => {
        if (!o) setChoice('cancel')
        onOpenChange(o)
      }}
      tone="bad"
      badge="Cancel send"
      title={`Cancel the rest of ${s.name}?`}
      description={`${count(s.sent)} of ${count(s.recipients)} are already delivered and cannot be recalled. ${count(remaining)} are queued and will not be sent. The report keeps everything sent so far.`}
      confirmLabel={choice === 'pause' ? 'Pause instead' : 'Cancel remaining sends'}
      confirmVariant={choice === 'pause' ? 'primary' : 'danger'}
      busyLabel={choice === 'pause' ? 'Pausing…' : 'Cancelling…'}
      cancelLabel="Keep sending"
      onConfirm={() => (choice === 'pause' ? actions.pause() : actions.cancel({}))}
    >
      <div className="flex flex-col gap-2.5">
        <DialogOption
          name="cancel-send"
          checked={choice === 'cancel'}
          onChange={() => setChoice('cancel')}
          title={`Cancel the remaining ${count(remaining)}`}
          description="Marks the campaign as partially sent. You can send the remainder later as a new campaign."
        />
        {pausable ? (
          <DialogOption
            name="cancel-send"
            checked={choice === 'pause'}
            onChange={() => setChoice('pause')}
            title="Pause instead"
            description="Keeps the queue; resume from the monitor at any time."
          />
        ) : null}
      </div>
    </ConfirmDialog>
  )
}
