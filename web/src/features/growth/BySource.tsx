import { Link } from '@tanstack/react-router'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/** "By source": Audience › Growth, from Overview's Subscriber growth and List health (X13). */
export function BySourceLink() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/audience/growth" params={{ workspace: slug }}>
      {t(m.bySource)}
    </Link>
  )
}

/** List health's "net growth counts list members · By source". */
export function NetGrowthBySource() {
  const t = useT()
  return (
    <>
      {t(m.netCountsMembers)} · <BySourceLink />
    </>
  )
}
