import { Link } from '@tanstack/react-router'
import { useT } from '@/i18n'
import { ago } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/** HostRepute's status line: the dot's colour, the title and the detail. */
export function useHostReputeLine(): { color: string; title: string; line: string } {
  const { me } = useWorkspace()
  const t = useT()
  const hr = me.hostRepute
  if (hr?.status === 'connected') {
    return { color: '#15803D', title: t(m.hostReputeConnected), line: t(m.hostReputeSynced, { team: hr.teamName, ago: ago(hr.lastSyncAt) }) }
  }
  if (hr) return { color: '#B45309', title: t(m.hostReputeAttention), line: t(m.hostReputeReconnect) }
  return { color: '#6B7280', title: t(m.hostReputeConnect), line: t(m.hostReputeOptional) }
}

/** The sidebar footer's HostRepute card ("HostRepute connected · Acme Inc · synced 4 min ago"). */
export function HostReputeStatus() {
  const { slug } = useWorkspace()
  const hr = useHostReputeLine()
  return (
    <Link
      to="/w/$workspace/settings/integrations/hostrepute"
      params={{ workspace: slug }}
      className="flex items-center gap-2 rounded-lg border border-line bg-white px-2.5 py-2 text-ink hover:text-ink"
    >
      <span className="dot" style={{ color: hr.color }} aria-hidden="true" />
      <div className="flex flex-col leading-[1.25]">
        <span className="text-[12px] font-semibold">{hr.title}</span>
        <span className="k text-[11px]">{hr.line}</span>
      </div>
    </Link>
  )
}
