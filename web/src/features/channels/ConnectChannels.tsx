import { useNavigate } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { channelIcon, channelName, type Channel } from './ChannelSwitch'
import { m } from './messages'

/**
 * The connect wizard's channel choice (SmsStates › step 1): Email · SMS ·
 * and each channel this build has. Email and SMS open their catalogs; a
 * channel without one here yet stays off.
 */
export function ConnectChannels({ current }: { current: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const channels = q.data?.channels.filter((c) => c.available) ?? []
  if (channels.length < 2) return null
  const go = (c: Channel) => {
    if (c === 'email') void navigate({ to: '/w/$workspace/providers/connect', params: { workspace: slug } })
    if (c === 'sms') void navigate({ to: '/w/$workspace/providers/connect/sms', params: { workspace: slug } })
  }
  return (
    <div className="seg" role="group" aria-label={t(m.switchLabel)}>
      {channels.map((c) => (
        <button key={c.channel} type="button" aria-pressed={c.channel === current} disabled={c.channel !== 'email' && c.channel !== 'sms'} onClick={() => go(c.channel)}>
          {channelIcon[c.channel]}
          {t(channelName[c.channel])}
        </button>
      ))}
    </div>
  )
}
