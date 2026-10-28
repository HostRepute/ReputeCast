import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { mr } from '@/features/approvals/review.messages'
import { clock } from '@/lib/format'
import { useCan } from '@/lib/session'
import { T, t as tt, useT } from '@/i18n'
import { useComposer } from './context'
import { CommentsTab, useComments } from './Comments'
import { useReadiness } from './Header'
import { ToolIcon } from './icons'
import { InboxStrip } from './InboxStrip'
import { ChecksContent } from './RightPanel'
import { SendTestButton } from './SendTest'
import { tagOf } from './editor/labels'
import { mh } from './chrome.messages'

/**
 * Quick review on phones (CO-14, ComposerPhone board): the email reads as
 * it is, texts can be edited in place, and Inbox, Checks, Comments and Send
 * test open as sheets. Layout editing waits for a larger screen. While a
 * request waits for the person's decision, Approve is one of them (R4.U6).
 */

type Sheet = 'edit' | 'inbox' | 'checks' | 'comments' | 'test' | 'approve' | null

type ApprovalDetail = Schemas['ApprovalDetail']

/**
 * The campaign's pending request when this person can decide on it now, by
 * the approval page's own rules (an approver who isn't its author and
 * hasn't decided yet); undefined otherwise.
 */
function usePendingDecision(): ApprovalDetail | undefined {
  const c = useComposer()
  const approver = useCan('campaigns:approve')
  const pending = c.campaign.approval?.status === 'pending' ? c.campaign.approval.id : undefined
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/approvals/{approvalId}',
    { params: { path: { workspace: c.slug, approvalId: pending ?? '' } } },
    { enabled: Boolean(pending) && approver },
  )
  const d = q.data
  return pending && approver && d?.status === 'pending' && d.can.decide ? d : undefined
}

/**
 * "Approve …?" on a phone: who asked and for which version, how many
 * approvals are still needed, and the approval page's note; Approve
 * decides as the approval page does, Review request opens it.
 */
function ApproveSheet({ detail: d, open, onClose }: { detail: ApprovalDetail; open: boolean; onClose: () => void }) {
  const c = useComposer()
  const t = useT()
  const qc = useQueryClient()
  const [busy, setBusy] = useState(false)
  const approve = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/decision', {
        params: { path: { workspace: c.slug, approvalId: d.id } },
        body: { decision: 'approve' },
      })
      toast({
        kind: 'success',
        title: tt(mr.decided, { decision: 'approve', name: d.name }),
        detail: data && data.status === 'pending' ? tt(mr.decidedProgress, { given: data.given, needed: data.needed }) : tt(mr.decidedNotified, { name: d.author.name }),
      })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: tt(mr.decideFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
      // The campaign's status and approval, the request and the inbox as they now are.
      await Promise.all([
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}'] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals/{approvalId}'] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] }),
      ])
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && !busy && onClose()} title={t(mh.approveTitle, { name: d.name })} className="qsheet">
      <div className="flex flex-col gap-3 p-3">
        <p className="m-0">{t(mh.approveAsked, { name: d.author.name, version: d.version, time: clock(d.requestedAt) })}</p>
        <dl className="kv m-0 text-[12.5px]">
          <dt>{t(mr.needed)}</dt>
          <dd>{t(mr.neededValue, { left: Math.max(0, d.needed - d.given), needed: d.needed })}</dd>
        </dl>
        <div className="note n text-[12.5px]">
          <span>{t(mr.approveNote, { name: d.author.name.split(' ')[0]! })}</span>
        </div>
        <Button variant="primary" className="min-h-[44px] justify-center" loading={busy} onClick={() => void approve()}>
          {t(mr.approve)}
        </Button>
        <ButtonLink className="min-h-[44px] justify-center" to="/w/$workspace/approvals/$approvalId" params={{ workspace: c.slug, approvalId: d.id }}>
          {t(mh.reviewRequest)}
        </ButtonLink>
      </div>
    </Dialog>
  )
}

/** "Quick review on this screen…": above the email on phones. */
export function QuickReviewNote() {
  return (
    <div className="qnote" role="note">
      <ToolIcon name="info" size={15} strokeWidth={1.6} />
      <span>
        <T msg={mh.quickNote} values={{ b: (chunks: ReactNode[]) => <b>{chunks}</b> }} />
      </span>
    </div>
  )
}

/** While Edit text is on, a tap on a text starts editing it (texts are outlined). */
function useTapToEdit(on: boolean) {
  const c = useComposer()
  const ed = c.editor
  useEffect(() => {
    if (!ed || !on) return
    const body = ed.Canvas.getBody()
    body?.classList.add('rc-qedit')
    const onSelect = (comp: Component) => {
      if (tagOf(comp) !== 'mj-text' || !c.canEdit) return
      ;(comp.view as unknown as { onActive?: (e?: Event) => void } | undefined)?.onActive?.()
    }
    ed.on('component:selected', onSelect)
    return () => {
      body?.classList.remove('rc-qedit')
      ed.off('component:selected', onSelect)
    }
  }, [ed, on, c.canEdit])
}

export function QuickReviewBar() {
  const c = useComposer()
  const t = useT()
  const [sheet, setSheet] = useState<Sheet>(null)
  const ready = useReadiness()
  const comments = useComments().data?.open ?? 0
  const [editing, setEditing] = useState(false)
  useTapToEdit(editing)
  const decision = usePendingDecision()
  const close = () => setSheet(null)
  const item = (key: Exclude<Sheet, null>, icon: string, label: string, badge?: number, aria?: string, className?: string) => (
    <button type="button" className={className} aria-label={aria} aria-pressed={key === 'edit' ? editing : undefined} onClick={() => (key === 'edit' ? setEditing((e) => !e) : setSheet(key))}>
      <span className="qi" aria-hidden="true">
        <ToolIcon name={icon} size={18} strokeWidth={1.6} />
        {/* Drawn from data-n, so the button's name (its aria-label) contains all of its text (WCAG 2.5.3). */}
        {badge ? <span className="qn" data-n={badge} /> : null}
      </span>
      {label}
    </button>
  )
  return (
    <>
      <div className="qbar" role="group" aria-label={t(mh.quickGroup)}>
        {item('edit', 'pencil', t(mh.editText))}
        {item('inbox', 'mail', t(mh.inbox))}
        {item('checks', 'shield', t(mh.checks), ready.blocks + ready.warnings, t(mh.sheetChecks, { blocks: ready.blocks, warnings: ready.warnings }))}
        {item('comments', 'comment', t(mh.comments), comments, t(mh.sheetComments, { count: comments }))}
        {item('test', 'send', t(mh.sendTest))}
        {/* A request waiting for this person's decision (R4.U6). */}
        {decision ? item('approve', 'check', t(mh.approve), undefined, undefined, 'qapprove') : null}
      </div>
      {editing ? (
        <div className="qedit" role="status">
          <span>{t(mh.editTextLead)}</span>
          <button type="button" className="btn sm" onClick={() => setEditing(false)}>
            {t(mh.inboxDone)}
          </button>
        </div>
      ) : null}
      <Dialog open={sheet === 'inbox'} onOpenChange={(o) => !o && close()} title={t(mh.inbox)} className="qsheet">
        <InboxStrip phoneSheet />
      </Dialog>
      <Dialog open={sheet === 'checks'} onOpenChange={(o) => !o && close()} title={t(mh.sheetChecks, { blocks: ready.blocks, warnings: ready.warnings })} className="qsheet">
        <ChecksContent />
      </Dialog>
      <Dialog open={sheet === 'comments'} onOpenChange={(o) => !o && close()} title={t(mh.sheetComments, { count: comments })} className="qsheet">
        <CommentsTab />
      </Dialog>
      <Dialog open={sheet === 'test'} onOpenChange={(o) => !o && close()} title={t(mh.sendTest)} className="qsheet">
        <div className="flex flex-col gap-2 p-3">
          <SendTestButton disabled={c.isEmpty} />
        </div>
      </Dialog>
      {decision ? <ApproveSheet detail={decision} open={sheet === 'approve'} onClose={close} /> : null}
    </>
  )
}
