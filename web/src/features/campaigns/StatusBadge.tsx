import { Badge } from '@/components/ui/Badge'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { m } from './messages'

type Status = Schemas['CampaignListItem']['status']

/** The campaign state word used in lists, headers and the overview. */
export function CampaignStatusBadge({ status, blocked }: { status: Status; blocked?: boolean }) {
  const t = useT()
  switch (status) {
    case 'scheduled':
      return (
        <Badge tone="info" dot>
          {t(m.statusScheduled)}
        </Badge>
      )
    case 'sending':
      return (
        <Badge tone="ok" dot className="[&>.dot]:pulse-live">
          {t(m.statusSending)}
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="bad" pulse>
          {t(m.statusPaused)}
        </Badge>
      )
    case 'approval':
      // ApprovalReview's word for a campaign that waits for an approver.
      return (
        <Badge tone="warn" dot>
          {t(m.statusApproval)}
        </Badge>
      )
    case 'approved':
      return (
        <Badge tone="ok" dot>
          {t(m.statusApproved)}
        </Badge>
      )
    case 'sent':
      return (
        <Badge tone="ok" dot>
          {t(m.statusSent)}
        </Badge>
      )
    case 'cancelled':
      return <Badge tone="neutral">{t(m.statusCancelled)}</Badge>
    case 'failed':
      return (
        <Badge tone="bad" dot>
          {t(m.statusFailed)}
        </Badge>
      )
    default:
      return blocked ? (
        <Badge tone="bad" dot>
          {t(m.statusDraftBlocked)}
        </Badge>
      ) : (
        <Badge tone="neutral">{t(m.statusDraft)}</Badge>
      )
  }
}
