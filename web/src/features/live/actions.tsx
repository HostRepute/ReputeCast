import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { ConfirmDialog, DialogOption } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useCan } from '@/lib/session'
import { m } from './messages'
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
  const thresholdsM = $api.useMutation('put', `${base}/thresholds`)
  const acknowledgeM = $api.useMutation('post', `${base}/acknowledge`)
  const pending = pauseM.isPending || resumeM.isPending || cancelM.isPending || moveM.isPending || retryM.isPending || thresholdsM.isPending || acknowledgeM.isPending

  const settle = (r: Result) => {
    apply(r.snapshot)
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
    return r
  }
  const fail = (title: string) => (e: unknown) => {
    toast({ kind: 'error', title, detail: `${errorMessage(e)} ${tt(m.nothingChanged)}` })
    throw e
  }

  return {
    pending,
    pause: (reason?: string) =>
      pauseM
        .mutateAsync({ params: { path }, body: reason ? { reason } : {} })
        .then(settle)
        .catch(fail(tt(m.pauseFailed))),
    resume: (body: Schemas['LiveResumeRequest'] = {}) =>
      resumeM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          if (body.withChanges && r.excluded > 0) {
            const title = body.mailbox ? tt(m.resumedWhoWithChanges, { who: mailboxName(body.mailbox) }) : tt(m.resumedWithChanges)
            toast({ kind: 'success', title, detail: tt(m.excludedDetail, { count: count(r.excluded) }) })
          }
          return r
        })
        .catch(fail(tt(m.resumeFailed))),
    cancel: (body: Schemas['LiveCancelRequest'] = {}) =>
      cancelM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          if (body.mailbox || body.providerId) {
            toast({ kind: 'deleted', title: tt(m.shareCancelled), detail: tt(m.shareCancelledDetail, { count: count(r.cancelled) }) })
          }
          return r
        })
        .catch(fail(tt(m.cancelFailed))),
    move: (body: Schemas['LiveMoveRequest'], target?: string) =>
      moveM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          const title = target ? tt(m.moved, { count: count(r.moved), target }) : tt(m.movedToOther, { count: count(r.moved) })
          toast({ kind: 'success', title, detail: tt(m.movedDetail) })
          return r
        })
        .catch(fail(tt(m.moveFailed))),
    retry: (body: Schemas['LiveRetryRequest']) =>
      retryM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .catch(fail(tt(m.retryFailed))),
    /** Marks an alert as read: complaints after the send (review GS-6). */
    acknowledge: (body: Schemas['LiveAcknowledgeRequest']) =>
      acknowledgeM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .catch(fail(tt(m.acknowledgeFailed))),
    /** Changes the send's auto-pause thresholds while it sends or waits (review GS-2). */
    setThresholds: (body: Schemas['LiveThresholdsRequest']) =>
      thresholdsM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          toast({ kind: 'success', title: tt(m.thresholdsSaved), detail: tt(m.thresholdsSavedDetail) })
          return r
        })
        .catch(fail(tt(m.thresholdsFailed))),
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
      return actions.move(a.body, target)
    case 'retry':
      return actions.retry(a.body)
  }
}

/** Who may do what on the monitor, and why not (add-on §6: pause needs campaigns:pause; resume, move and cancel need campaigns:send). */
export function useLivePermissions(helpers: Schemas['LiveHelpers'] | undefined) {
  const canPause = useCan('campaigns:pause')
  const canSend = useCan('campaigns:send')
  return {
    pause: canPause ? undefined : needs('pause', 'campaigns:pause', helpers?.pause),
    resume: canSend ? undefined : needs('resume', 'campaigns:send', helpers?.send),
    cancel: canSend ? undefined : needs('cancel', 'campaigns:send', helpers?.send),
    decide: canSend ? undefined : needs('decide', 'campaigns:send', helpers?.send),
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
  const t = useT()
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
      badge={t(m.cancelSend)}
      title={t(m.cancelTitle, { name: s.name })}
      description={t(m.cancelBody, { sent: count(s.sent), total: count(s.recipients), remaining: count(remaining) })}
      confirmLabel={choice === 'pause' ? t(m.pauseInstead) : t(m.cancelRemaining)}
      confirmVariant={choice === 'pause' ? 'primary' : 'danger'}
      busyLabel={choice === 'pause' ? t(m.pausing) : t(m.cancelling)}
      cancelLabel={t(m.keepSending)}
      onConfirm={() => (choice === 'pause' ? actions.pause() : actions.cancel({}))}
    >
      <div className="flex flex-col gap-2.5">
        <DialogOption
          name="cancel-send"
          checked={choice === 'cancel'}
          onChange={() => setChoice('cancel')}
          title={t(m.cancelRemainingCount, { count: count(remaining) })}
          description={t(m.cancelRemainingBody)}
        />
        {pausable ? (
          <DialogOption
            name="cancel-send"
            checked={choice === 'pause'}
            onChange={() => setChoice('pause')}
            title={t(m.pauseInstead)}
            description={t(m.pauseInsteadBody)}
          />
        ) : null}
      </div>
    </ConfirmDialog>
  )
}
