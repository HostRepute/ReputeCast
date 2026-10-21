import type { Schemas } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { m } from './messages'

/**
 * Settings › Approvals in an agency's client workspace (agency accounts):
 * "Every send needs one approval from a Client approver", on every
 * channel, and optionally for publishing automations. The agency's staff
 * can't give it.
 */
export function ClientSignoffRule({
  info,
  value,
  disabled,
  onChange,
}: {
  info: Schemas['ApprovalClientSignoffInfo']
  value: Schemas['ApprovalClientSignoff']
  disabled: boolean
  onChange: (v: Schemas['ApprovalClientSignoff']) => void
}) {
  const t = useT()
  const who = info.approvers.length ? t(m.signoffApprovers, { names: formatList(info.approvers) }) : t(m.signoffNoApprover)
  const text = (
    <>
      <span className="font-semibold">{t(m.signoffTitle)}</span>
      <span className="k whitespace-normal">{t(m.signoffDetail, { agency: info.agency, who })}</span>
    </>
  )
  return (
    <div className="ap-rule flex-col! items-stretch! gap-2!">
      <label className="flex items-start gap-3">
        <input type="checkbox" className="tg" role="switch" checked={value.enabled} disabled={disabled} onChange={(e) => onChange({ ...value, enabled: e.target.checked })} />
        <span className="col min-w-0 flex-1 gap-px!">{text}</span>
      </label>
      {value.enabled ? (
        <label className="flex items-center gap-2 pl-[52px] text-[13px]">
          <input type="checkbox" checked={value.automations ?? false} disabled={disabled} onChange={(e) => onChange({ ...value, automations: e.target.checked })} />
          {t(m.signoffAutomations)}
        </label>
      ) : null}
    </div>
  )
}
