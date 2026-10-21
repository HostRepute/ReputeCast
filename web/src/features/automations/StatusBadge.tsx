import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { m } from './messages'

/** The flow state word: Live, Paused or Draft. */
export function AutomationStatusBadge({ status, children }: { status: 'draft' | 'live' | 'paused'; children?: React.ReactNode }) {
  const t = useT()
  switch (status) {
    case 'live':
      return (
        <Badge tone="ok" dot>
          {children ?? t(m.statusLive)}
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="warn" dot>
          {children ?? t(m.statusPaused)}
        </Badge>
      )
    default:
      return <Badge tone="neutral">{children ?? t(m.statusDraft)}</Badge>
  }
}
