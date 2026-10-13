import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { orList } from '@/lib/format'

/**
 * The read-only line at the top of a settings page for people who can look
 * but not change (TeamViewer's pattern): what they can do, the permission a
 * change needs and who can help.
 */
export function ReadOnlyNote({
  children,
  action = 'Changing them',
  perm,
  helpers,
}: {
  children: ReactNode
  /** What needs the permission ("Changing them"). */
  action?: string
  perm: string
  helpers?: string[]
}) {
  return (
    <div className="note n" role="status">
      <Badge tone="neutral">Read-only</Badge>
      <span>
        {children} {action} needs the <span className="mono">{perm}</span> permission.
        {helpers?.length ? ` Ask ${orList(helpers)}.` : ''}
      </span>
    </div>
  )
}
