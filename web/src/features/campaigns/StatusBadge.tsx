import { Badge } from '@/components/ui/Badge'
import type { Schemas } from '@/api/client'

type Status = Schemas['CampaignListItem']['status']

/** The campaign state word used in lists, headers and the overview. */
export function CampaignStatusBadge({ status, blocked }: { status: Status; blocked?: boolean }) {
  switch (status) {
    case 'scheduled':
      return (
        <Badge tone="info" dot>
          Scheduled
        </Badge>
      )
    case 'sending':
      return (
        <Badge tone="ok" dot className="[&>.dot]:pulse-live">
          Sending
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="bad" pulse>
          Auto-paused
        </Badge>
      )
    case 'approval':
      return (
        <Badge tone="warn" dot>
          Waiting for approval
        </Badge>
      )
    case 'approved':
      return (
        <Badge tone="ok" dot>
          Approved
        </Badge>
      )
    case 'sent':
      return (
        <Badge tone="ok" dot>
          Sent
        </Badge>
      )
    case 'cancelled':
      return <Badge tone="neutral">Cancelled</Badge>
    case 'failed':
      return (
        <Badge tone="bad" dot>
          Failed
        </Badge>
      )
    default:
      return blocked ? (
        <Badge tone="bad" dot>
          Draft · blocked
        </Badge>
      ) : (
        <Badge tone="neutral">Draft</Badge>
      )
  }
}
