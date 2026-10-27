import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { useNavigate } from '@tanstack/react-router'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/**
 * More (SendReadinessApproval, SR-13): a link to the request, the approval
 * policy, and the emergency send, which sits here instead of beside the
 * main actions. Owners use it when the policy allows; everyone else sees
 * it disabled with who can, never a permission key.
 */
export function MoreMenu({
  canBypass,
  policyOff,
  owners,
  approvers,
  requestId,
  onBypass,
}: {
  canBypass: boolean
  policyOff: boolean
  /** The workspace's Owners, who can use the emergency send. */
  owners: string[]
  /** Who is told at once when it's used. */
  approvers: string[]
  requestId?: string
  onBypass: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const sub = policyOff
    ? t(m.emergencyOff)
    : canBypass
      ? t(m.emergencyOwner, { names: approvers.length ? formatList(approvers) : 'none' })
      : t(m.emergencyOwnersOnly, { names: owners.length ? formatList(owners, 'disjunction') : 'none' })
  const copy = async () => {
    const url = `${window.location.origin}/w/${slug}/approvals/${requestId}`
    try {
      await navigator.clipboard.writeText(url)
      toast({ kind: 'success', title: t(m.linkCopied) })
    } catch {
      toast({ kind: 'error', title: t(m.linkNotCopied), detail: url })
    }
  }
  return (
    <Menu
      label={t(m.moreActions)}
      trigger={
        <Button aria-label={t(m.moreActions)}>
          {t(m.more)}
          <Icon name="chevronDown" size={12} />
        </Button>
      }
    >
      {requestId ? <MenuItem onSelect={() => void copy()}>{t(m.copyRequestLink)}</MenuItem> : null}
      <MenuItem onSelect={() => void navigate({ to: '/w/$workspace/settings/approvals', params: { workspace: slug } })}>{t(m.viewPolicy)}</MenuItem>
      <MenuItem disabled={!canBypass} onSelect={canBypass ? onBypass : undefined}>
        <span className="flex flex-col gap-0.5">
          <span className={canBypass ? 'text-bad-fg' : undefined}>{t(m.emergencySend)}</span>
          <span className="k">{sub}</span>
        </span>
      </MenuItem>
    </Menu>
  )
}

/**
 * The main action held at the bottom of the screen on phones (SR-13,
 * SendReadinessPhone): when the send goes out and the verdict, with the
 * short action in reach of the thumb.
 */
export function StickyAction({
  when,
  verdict,
  label,
  aria,
  disabledReason,
  onClick,
}: {
  when: string
  verdict: string
  label: string
  aria: string
  disabledReason?: string
  onClick: () => void
}) {
  const t = useT()
  return (
    <div className="rv-actbar" role="region" aria-label={t(m.sendRegion)}>
      <div className="rv-abm">
        <b>{when}</b>
        <span className="k">{verdict}</span>
      </div>
      <Button variant="primary" aria-label={aria} disabledReason={disabledReason} onClick={onClick}>
        {label}
      </Button>
    </div>
  )
}
