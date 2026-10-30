import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { m } from './messages'
import './channels.css'

/** A channel of the workspace: email, SMS, push or WhatsApp. */
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

/** The channels' names, in the app language when read with t(). */
export const channelName = {
  email: m.email,
  sms: m.sms,
  push: m.push,
  whatsapp: m.whatsapp,
} as const
