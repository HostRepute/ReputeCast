import { Badge } from '@/components/ui/Badge'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { orList } from '@/lib/format'
import { m } from './messages'

/**
 * The read-only line at the top of a settings page for people who can look
 * but not change (TeamViewer's pattern): what they can do, the permission a
 * change needs and who can help. `msg` says the first two in one message,
 * with the permission as {perm}: "You can view the senders… Changing them
 * needs the {perm} permission."
 */
export function ReadOnlyNote({ msg, perm, helpers }: { msg: MessageDescriptor; perm: string; helpers?: string[] }) {
  const t = useT()
  return (
    <div className="note n" role="status">
      <Badge tone="neutral">{t(m.readOnly)}</Badge>
      <span>
        <T msg={msg} values={{ perm: <span className="mono">{perm}</span> }} />
        {helpers?.length ? ` ${t(m.askHelpers, { helpers: orList(helpers) })}` : ''}
      </span>
    </div>
  )
}
