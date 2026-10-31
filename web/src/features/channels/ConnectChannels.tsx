import { useNavigate } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { channelIcon, channelName, type Channel } from './channel'
import { m } from './messages'

/**
 * The connect wizard's channel choice (SmsStates › step 1): Email · SMS ·
 * and each channel this build has. Email and SMS open their catalogs; push
 * and WhatsApp open their own sections, which set them up.
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
    // Push and WhatsApp set up in their own sections (X22).
    if (c === 'push') void navigate({ to: '/w/$workspace/providers/push', params: { workspace: slug } })
    if (c === 'whatsapp') void navigate({ to: '/w/$workspace/providers/whatsapp', params: { workspace: slug } })
  }
  return (
    <div className="seg" role="group" aria-label={t(m.switchLabel)}>
      {channels.map((c) => (
        <button key={c.channel} type="button" aria-pressed={c.channel === current} onClick={() => go(c.channel)}>
          {channelIcon[c.channel]}
          {t(channelName[c.channel])}
        </button>
      ))}
    </div>
  )
}
