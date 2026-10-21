import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './push.css'

/*
 * The channel switch of Sending providers (brief §3.1) on the push and
 * WhatsApp screens. X21 owns the shared switch (features/channels); until
 * its screens land in this branch, push and WhatsApp carry this copy with
 * the same icons and order. When they land, these screens use X21's switch
 * and its push and WhatsApp cases link here.
 */

export type Channel = 'email' | 'sms' | 'push' | 'whatsapp'

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

const channelName = { email: m.channelEmail, sms: m.channelSms, push: m.channelPush, whatsapp: m.channelWhatsApp } as const

/** Email · SMS · Push · WhatsApp, with the current channel marked. */
export function ChannelSwitch({ current }: { current: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const params = { workspace: slug }
  const channels: Channel[] = q.data?.channels.map((c) => c.channel) ?? ['email', 'sms', 'push', 'whatsapp']
  return (
    <nav className="px-chsw" aria-label={t(m.channelLabel)}>
      {channels.map((c) => {
        const on = c === current
        const label = (
          <>
            {channelIcon[c]}
            <span>{t(channelName[c])}</span>
          </>
        )
        const cls = on ? 'on' : undefined
        const aria = on ? ('true' as const) : undefined
        switch (c) {
          case 'email':
            return (
              <Link key={c} to="/w/$workspace/providers" params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
          case 'push':
            return (
              <Link key={c} to="/w/$workspace/providers/push" params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
          case 'whatsapp':
            return (
              <Link key={c} to="/w/$workspace/providers/whatsapp" params={params} className={cls} aria-current={aria}>
                {label}
              </Link>
            )
          default:
            // SMS's screens are X21's: its section, once they land here.
            return (
              <a key={c} href={`/w/${slug}/providers/sms`} className={cls} aria-current={aria}>
                {label}
              </a>
            )
        }
      })}
    </nav>
  )
}
