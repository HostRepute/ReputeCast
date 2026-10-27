import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { andList, clock, weekdayDate } from '@/lib/format'
import { useT } from '@/i18n'
import type { Campaign } from './context'
import { mn } from './banners.messages'

export type Approval = NonNullable<Campaign['approval']>

/** Who approved ("Marcus Chen", "Marcus Chen and Linh Nguyen"), or who asked. */
function approvers(a: Approval): string {
  return a.approvedBy.length ? andList(a.approvedBy) : a.requestedBy
}

/** The check-in-a-circle the board puts on an approved campaign. */
function ApprovedIcon() {
  return (
    <svg className="bi" width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="m5.2 8.3 1.9 1.8L10.9 6" />
    </svg>
  )
}

/**
 * An approved (or pending) campaign opens without editing (CO-7): editing
 * cancels the approval, so the person says so first. The banner says what
 * is approved and opens the question.
 */
export function ApprovalBanner({ approval, onEdit }: { approval: Approval; onEdit: () => void }) {
  const t = useT()
  const approved = approval.status === 'approved'
  return (
    <div className="bnr w" role="status">
      <ApprovedIcon />
      <span className="bt">
        <b>
          {approved
            ? t(mn.approvedTitle, { name: approvers(approval), time: clock(approval.decidedAt ?? approval.requestedAt), version: approval.version })
            : t(mn.pendingTitle, { name: approval.requestedBy, time: clock(approval.requestedAt), version: approval.version })}
        </b>
        <span>{approved ? t(mn.approvedBody) : t(mn.pendingBody)}</span>
      </span>
      <span className="ba">
        <button type="button" className="btn sm" onClick={onEdit}>
          {t(mn.editCampaign)}
        </button>
      </span>
    </div>
  )
}

/**
 * "Edit an approved campaign?" (ComposerStates · Approval): editing cancels
 * the approval; the schedule stays, but the send waits for a new one.
 */
export function ApprovalDialog({
  approval,
  scheduledAt,
  open,
  onOpenChange,
  onEdit,
}: {
  approval: Approval
  scheduledAt?: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onEdit: () => void | Promise<unknown>
}) {
  const t = useT()
  const approved = approval.status === 'approved'
  const name = approvers(approval)
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="warn"
      badge={approved ? t(mn.approvalBadge) : t(mn.pendingBadge)}
      title={approved ? t(mn.approvalDialogTitle) : t(mn.pendingDialogTitle)}
      description={approved ? t(mn.approvalDialogBody, { name }) : t(mn.pendingDialogBody, { name: approval.requestedBy })}
      confirmLabel={t(mn.editAnyway)}
      onConfirm={onEdit}
    >
      <dl className="kv">
        <dt>{approved ? t(mn.kvApproved) : t(mn.kvRequested)}</dt>
        <dd>
          {approved
            ? t(mn.kvApprovedValue, { name, time: clock(approval.decidedAt ?? approval.requestedAt), version: approval.version })
            : t(mn.kvRequestedValue, { name: approval.requestedBy, time: clock(approval.requestedAt), version: approval.version })}
        </dd>
        {scheduledAt ? (
          <>
            <dt>{t(mn.kvSchedule)}</dt>
            <dd>{t(mn.kvScheduleValue, { time: `${weekdayDate(scheduledAt)} ${clock(scheduledAt)}` })}</dd>
          </>
        ) : null}
        <dt>{t(mn.kvNext)}</dt>
        <dd>{approved ? t(mn.kvNextValue, { names: name, version: approval.version }) : t(mn.kvNextPending, { version: approval.version })}</dd>
      </dl>
    </ConfirmDialog>
  )
}
