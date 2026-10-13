import { Badge } from '@/components/ui/Badge'

/** The flow state word: Live, Paused or Draft. */
export function AutomationStatusBadge({ status, children }: { status: 'draft' | 'live' | 'paused'; children?: React.ReactNode }) {
  switch (status) {
    case 'live':
      return (
        <Badge tone="ok" dot>
          {children ?? 'Live'}
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="warn" dot>
          {children ?? 'Paused'}
        </Badge>
      )
    default:
      return <Badge tone="neutral">{children ?? 'Draft'}</Badge>
  }
}
