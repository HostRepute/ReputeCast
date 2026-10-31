import { Badge } from '@/components/ui/Badge'
import { useT, type MessageDescriptor } from '@/i18n'
import { orList } from '@/lib/format'
import { m } from './messages'

/**
 * The read-only line at the top of a settings page for people who can look
 * but not change (TeamViewer's pattern): what they can do, what their role
 * can't, and who can help. `msg` says the first two in words, never the
 * permission key (review X-1): "You can view the senders… Your role can't
 * change them."
 */
export function ReadOnlyNote({ msg, helpers }: { msg: MessageDescriptor; helpers?: string[] }) {
  const t = useT()
  return (
    <div className="note n" role="status">
      <Badge tone="neutral">{t(m.readOnly)}</Badge>
      <span>
        {t(msg)}
        {helpers?.length ? ` ${t(m.askHelpers, { helpers: orList(helpers) })}` : ''}
      </span>
    </div>
  )
}
