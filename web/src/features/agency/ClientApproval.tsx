import type { Schemas } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { dayMonth, hhmm } from './parts'
import { useWorkspaceAgency } from './WorkspaceParts'

type Detail = Schemas['ApprovalDetail']

/**
 * The review page of a request under client sign-off (AgencyClientApproval):
 * who asked, from the agency; the rule; and that the agency's staff can't
 * give the approval it needs.
 */
function useSignoff(d: Detail) {
  const t = useT()
  const { me } = useWorkspace()
  const { data } = useWorkspaceAgency()
  const agency = data?.agency ?? ''
  const staffIds = new Set((data?.staff ?? []).map((s) => s.userId))
  const fromStaff = staffIds.has(d.requestedBy.id)
  const when = (iso: string) => t(m.when, { day: dayMonth(iso), time: hhmm(iso) })
  return { t, agency, fromStaff, when, workspace: me.workspace.name, s: d.clientSignoff! }
}

/** "Requested by Mei Lin, Northwind Studio · 14 Oct, 16:20 · you're reviewing as Jane Morales". */
export function ClientSignoffLine({ detail: d, me }: { detail: Detail; me?: string }) {
  const { t, agency, fromStaff, when, s } = useSignoff(d)
  const role = s.viewerIsStaff
    ? s.approvers.length
      ? t(m.signoffWaitingFor, { names: formatList(s.approvers) })
      : t(m.signoffNobodyYet)
    : d.can.decide
      ? t(m.signoffReviewingAs, { name: me ?? '' })
      : t(m.signoffViewingAs, { name: me ?? '' })
  return (
    <span className="k">
      {fromStaff
        ? t(m.signoffRequestedStaff, { name: d.requestedBy.name, agency, when: when(d.requestedAt), role })
        : t(m.signoffRequested, { name: d.requestedBy.name, when: when(d.requestedAt), role })}
    </span>
  )
}

/** Under "Your decision": who the viewer is for this rule. */
export function ClientSignoffIntro({ detail: d }: { detail: Detail }) {
  const { t, agency, workspace, s } = useSignoff(d)
  if (d.status !== 'pending') return null
  // A line of its own under the card's heading.
  const line = 'k -mt-1.5 basis-full'
  if (s.viewerIsStaff) return <span className={line}>{t(m.signoffStaffIntro, { agency })}</span>
  if (!d.can.decide) return null
  return <span className={line}>{t(m.signoffApproverIntro, { workspace })}</span>
}

/** The note under the decision buttons. */
export function ClientSignoffNote({ detail: d }: { detail: Detail }) {
  const { t } = useSignoff(d)
  return (
    <div className="note n text-[12.5px]">
      <span>{t(m.signoffSnapshot, { name: d.author.name })}</span>
    </div>
  )
}

/** The facts under the decision: requested by, the rule, the agency's staff, how long it has waited. */
export function ClientSignoffFacts({ detail: d }: { detail: Detail }) {
  const { t, agency, fromStaff, when, s } = useSignoff(d)
  const now = useNow()
  const minutes = Math.max(0, Math.floor((now.getTime() - new Date(d.requestedAt).getTime()) / 60_000))
  const days = Math.floor(minutes / 1440)
  const hours = Math.floor((minutes % 1440) / 60)
  const mins = minutes % 60
  const waited = days > 0 ? t(m.signoffDaysHours, { days, hours }) : hours > 0 ? t(m.signoffHoursMinutes, { hours, minutes: mins }) : t(m.signoffMinutes, { minutes: mins })
  return (
    <dl className="kv m-0 text-[12.5px]">
      <dt>{t(m.signoffFactRequestedBy)}</dt>
      <dd>{fromStaff ? t(m.signoffPersonAgency, { name: d.requestedBy.name, agency }) : d.requestedBy.name}</dd>
      <dt>{t(m.signoffFactRule)}</dt>
      <dd>{t(m.signoffFactRuleValue)}</dd>
      <dt>{t(m.signoffFactStaff)}</dt>
      <dd>{s.staffApprovals.length ? t(m.signoffFactStaffApproved, { names: formatList(s.staffApprovals) }) : t(m.signoffFactStaffValue)}</dd>
      {s.met && s.signedOffBy ? (
        <>
          <dt>{t(m.signoffFactSignedOff)}</dt>
          <dd>{s.signedOffBy}</dd>
        </>
      ) : d.status === 'pending' ? (
        <>
          <dt>{t(m.signoffFactWaiting)}</dt>
          <dd>{t(m.signoffFactWaitingValue, { duration: waited, when: when(d.requestedAt) })}</dd>
        </>
      ) : (
        <>
          <dt>{t(m.signoffFactRequested)}</dt>
          <dd>{when(d.requestedAt)}</dd>
        </>
      )}
    </dl>
  )
}
