import { ChannelIcons } from '@/features/channels/ChannelFilter'
import { useT } from '@/i18n'
import { money } from '@/lib/format'
import { m } from './automation.messages'
import './sms.css'

/**
 * The automations list's line under a flow that texts (brief §6): its
 * channels' icons and what its texts cost over 30 days.
 */
export function SmsFlowLine({ channels, cost }: { channels: string[]; cost?: number | null }) {
  const t = useT()
  if (!channels.some((c) => c !== 'email')) return null
  return (
    <div className="k sms-flowline">
      <ChannelIcons channels={channels} />
      {channels.includes('sms') ? <span>{cost ? t(m.listCost, { cost: money(cost) }) : t(m.listNoTexts)}</span> : null}
    </div>
  )
}
