import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { formatList, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { channelIcon, channelName, type Channel } from './ChannelSwitch'
import { m } from './messages'
import './channels.css'

/**
 * The campaigns list's channel chips (Campaigns, phase 8): All, Email and
 * each channel the workspace has set up, once a second one is.
 */
export function ChannelFilter({ value, onChange }: { value?: Channel; onChange: (c: Channel | undefined) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  if (!q.data?.switchShown) return null
  return (
    <div className="ch-filter" role="group" aria-label={t(m.switchLabel)}>
      <span className="k mr-1">{t(m.switchLabel)}</span>
      <button type="button" className={value ? undefined : 'on'} aria-pressed={!value} onClick={() => onChange(undefined)}>
        {t(m.all)}
      </button>
      {q.data.channels
        .filter((c) => c.setUp || c.channel === 'email' || c.channel === value)
        .map((c) => (
        <button key={c.channel} type="button" className={value === c.channel ? 'on' : undefined} aria-pressed={value === c.channel} onClick={() => onChange(c.channel)}>
          {channelIcon[c.channel]}
          {t(channelName[c.channel])}
        </button>
      ))}
    </div>
  )
}

/** The channel tag beside a campaign that isn't email: [SMS]. */
export function ChannelTag({ channel }: { channel: Channel }) {
  const t = useT()
  if (channel === 'email') return null
  return (
    <Badge tone="neutral" className="ch-tag">
      {channelIcon[channel]}
      {t(channelName[channel])}
    </Badge>
  )
}

/** The channels a flow sends on, as icons beside its name, once it sends more than email. */
export function ChannelIcons({ channels }: { channels: string[] }) {
  const t = useT()
  const known = channels.filter((c): c is Channel => c in channelName)
  if (!known.some((c) => c !== 'email')) return null
  return (
    <span className="ch-icons" role="img" aria-label={t(m.sendsOn, { channels: formatList(known.map((c) => t(channelName[c]))) })}>
      {known.map((c) => (
        <span key={c}>{channelIcon[c]}</span>
      ))}
    </span>
  )
}
