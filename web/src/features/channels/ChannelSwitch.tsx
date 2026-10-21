import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './channels.css'

export type Channel = Schemas['ChannelStatus']['channel']

/** The channels' icons, as the boards draw them. */
export const channelIcon: Record<Channel, ReactNode> = {
  email: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="1.5" y="3.5" width="13" height="9" rx="1.5" />
      <path d="m2 4.5 6 4.5 6-4.5" />
    </svg>
  ),
  sms: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 2.5h10A1.5 1.5 0 0 1 14.5 4v6a1.5 1.5 0 0 1-1.5 1.5H7.6L4.5 14v-2.5H3A1.5 1.5 0 0 1 1.5 10V4A1.5 1.5 0 0 1 3 2.5z" />
      <path d="M5.2 7h.01M8 7h.01M10.8 7h.01" strokeWidth="2.2" />
    </svg>
  ),
  push: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0" />
    </svg>
  ),
  whatsapp: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 1.8a6.2 6.2 0 0 0-5.33 9.36L1.8 14.2l3.13-.84A6.2 6.2 0 1 0 8 1.8z" />
      <path
        d="M6.1 5.2c-.5 0-.95.45-.85 1.05.35 2.2 2.15 4.05 4.4 4.45.6.1 1.05-.35 1.05-.85v-.55l-1.5-.65-.6.6a3.7 3.7 0 0 1-2-2l.6-.6-.65-1.5z"
        strokeWidth="1.25"
      />
    </svg>
  ),
}

export const channelName = {
  email: m.email,
  sms: m.sms,
  push: m.push,
  whatsapp: m.whatsapp,
} as const

/**
 * The channel switch of Sending providers (brief §3.1): Email · SMS · Push ·
 * WhatsApp, once a second channel is set up. Until then the header offers
 * "Add a channel". Each channel's own section takes its link; a channel
 * this build doesn't have yet opens the connect catalog.
 */
export function ChannelSwitch({ current }: { current: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const data = q.data
  if (!data) return null
  const params = { workspace: slug }
  if (!data.switchShown && current === 'email') {
    return (
      <Link to="/w/$workspace/providers/connect/sms" params={params} className="btn sm ch-add" title={t(m.addChannelHint)}>
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
          <path d="M8 3v10M3 8h10" />
        </svg>
        {t(m.addChannel)}
      </Link>
    )
  }
  return (
    <nav className="ch-switch" aria-label={t(m.switchLabel)}>
      {data.channels.map((c) => {
        const on = c.channel === current
        const label = (
          <>
            {channelIcon[c.channel]}
            <span>{t(channelName[c.channel])}</span>
          </>
        )
        const cls = on ? 'on' : undefined
        const aria = on ? ('true' as const) : undefined
        switch (c.channel) {
          case 'email':
            return (
              <Link key={c.channel} to="/w/$workspace/providers" params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
          case 'sms':
            return (
              <Link key={c.channel} to={c.setUp ? '/w/$workspace/providers/sms' : '/w/$workspace/providers/connect/sms'} params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
          default:
            return (
              <Link key={c.channel} to="/w/$workspace/providers/connect" params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
        }
      })}
    </nav>
  )
}
