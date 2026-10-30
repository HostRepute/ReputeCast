import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { ConfirmDialog, DialogOption } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { count, orList } from '@/lib/format'
import { useCan } from '@/lib/session'
import { mw } from '@/features/providers/warmup.messages'
import { mc } from './canary.messages'
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
  const inDoubtM = $api.useMutation('post', `${base}/in-doubt`)
  const thresholdsM = $api.useMutation('put', `${base}/thresholds`)
  const acknowledgeM = $api.useMutation('post', `${base}/acknowledge`)
  const ratesM = $api.useMutation('put', `${base}/rates`)
  const canaryContinueM = $api.useMutation('post', `${base}/canary/continue`)
  const canaryExtendM = $api.useMutation('post', `${base}/canary/extend`)
  const canaryStopM = $api.useMutation('post', `${base}/canary/stop`)
  // A share the warm-up cap holds is decided through the warm-up plan (review PR-8).
  const warmupHoldM = $api.useMutation('post', '/api/v1/workspaces/{workspace}/warmup/{planId}/held/{campaignId}')
  const pending =
    pauseM.isPending ||
    resumeM.isPending ||
    cancelM.isPending ||
    moveM.isPending ||
    retryM.isPending ||
    inDoubtM.isPending ||
    thresholdsM.isPending ||
    acknowledgeM.isPending ||
    ratesM.isPending ||
    canaryContinueM.isPending ||
    canaryExtendM.isPending ||
    canaryStopM.isPending ||
    warmupHoldM.isPending

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
    /** Messages in doubt (review GS-1): send them again, or leave them out of the send. */
    decideInDoubt: (body: Schemas['LiveInDoubtRequest']) =>
      inDoubtM
        .mutateAsync({ params: { path }, body })
        .then(settle)
        .then((r) => {
          const n = count(r.decided)
          toast(
            body.action === 'resend'
              ? { kind: 'success', title: tt(m.inDoubtResent, { count: r.decided, n }) }
              : { kind: 'success', title: tt(m.inDoubtLeftOut, { count: r.decided, n }), detail: tt(m.inDoubtLeftOutDetail) },
          )
          return r
        })
        .catch(fail(tt(m.inDoubtFailed))),
    /** The canary (review GS-10): everyone now, 10 more minutes, or stop. */
    continueCanary: () =>
      canaryContinueM
        .mutateAsync({ params: { path } })
        .then(settle)
        .then((r) => {
          toast({ kind: 'success', title: tt(mc.continued) })
          return r
        })
        .catch(fail(tt(mc.decideFailed))),
    extendCanary: () =>
      canaryExtendM
        .mutateAsync({ params: { path } })
        .then(settle)
        .then((r) => {
          toast({ kind: 'success', title: tt(mc.watched) })
          return r
        })
        .catch(fail(tt(mc.decideFailed))),
    stopCanary: () =>
      canaryStopM
        .mutateAsync({ params: { path } })
        .then(settle)
        .then((r) => {
          toast({ kind: 'success', title: tt(mc.stopped) })
          return r
        })
        .catch(fail(tt(mc.decideFailed))),
    /** Changes the rest of the send's rates per mailbox provider (review GS-13). */
    setRates: (rates: Record<string, number>) =>
      ratesM
        .mutateAsync({ params: { path }, body: { rates } })
        .then(settle)
        .catch(fail(tt(m.ratesFailed))),
    /**
     * A share today's warm-up cap holds (review PR-8): wait for tomorrow's
     * cap, move it to a warm provider, or cancel the rest. The answer isn't
     * a snapshot: the monitor asks for a new one.
     */
    decideWarmupHold: (planId: string, body: Schemas['WarmupHoldDecision'], target?: string) =>
      warmupHoldM
        .mutateAsync({ params: { path: { workspace: slug, planId, campaignId } }, body })
        .then((r) => {
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/live'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
          const n = count(r.moved || r.cancelled)
          const title =
            body.action === 'move' ? tt(mw.holdMovedToast, { count: n, provider: target ?? '' }) : body.action === 'cancel' ? tt(mw.holdCancelledToast, { count: n }) : tt(mw.holdWaitToast)
          toast({ kind: body.action === 'cancel' ? 'deleted' : 'success', title })
          return r
        })
        .catch(fail(tt(mw.holdFailed))),
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
    case 'inDoubt':
      return actions.decideInDoubt(a.body)
  }
}

/** Who may do what on the monitor, and why not (add-on §6: pause needs campaigns:pause; resume, move and cancel need campaigns:send). */
export function useLivePermissions(helpers: Schemas['LiveHelpers'] | undefined) {
  const canPause = useCan('campaigns:pause')
  const canSend = useCan('campaigns:send')
  const canApprove = useCan('campaigns:approve')
  return {
    // Continuing a canary early needs someone who can approve sends (review GS-10).
    approve: canApprove ? undefined : tt(mc.needsApprove, { helpers: helpers?.approve?.length ? orList(helpers.approve) : 'none' }),
    pause: canPause ? undefined : needs('pause', 'campaigns:pause', helpers?.pause),
    resume: canSend ? undefined : needs('resume', 'campaigns:send', helpers?.send),
    cancel: canSend ? undefined : needs('cancel', 'campaigns:send', helpers?.send),
    rates: canSend ? undefined : needs('rates', 'campaigns:send', helpers?.send),
    decide: canSend ? undefined : needs('decide', 'campaigns:send', helpers?.send),
    // Cancelling what the warm-up cap holds needs campaigns:pause (warm-up holds, review PR-8).
    warmupCancel: canPause ? undefined : needs('warmupCancel', 'campaigns:pause', helpers?.pause),
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
      badge={t(m.cancelBadge)}
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
        {/* Cancelling leaves the messages in doubt out as well (review GS-1). */}
        {s.inDoubt?.count && choice === 'cancel' ? <p className="k m-0">{t(m.cancelInDoubt, { count: s.inDoubt.count })}</p> : null}
      </div>
    </ConfirmDialog>
  )
}
