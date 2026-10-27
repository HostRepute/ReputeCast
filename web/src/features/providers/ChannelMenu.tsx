import { Link } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react'
import { $api } from '@/api/client'
import { channelIcon, channelName, type Channel } from '@/features/channels/ChannelSwitch'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/**
 * The channel of Sending providers as one button with a listbox (Providers
 * board): "Email ▾" opens Email, SMS, Push and WhatsApp; each option opens
 * that channel's section, or its setup when it isn't set up yet.
 */
export function ChannelMenu({ current }: { current: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const [open, setOpen] = useState(false)
  const box = useRef<HTMLDivElement>(null)
  const button = useRef<HTMLButtonElement>(null)
  const listId = useId()
  const channels: Channel[] = q.data?.channels.map((c) => c.channel) ?? ['email', 'sms', 'push', 'whatsapp']
  const setUp = new Map(q.data?.channels.map((c) => [c.channel, c.setUp]) ?? [])
  const params = { workspace: slug }

  useEffect(() => {
    if (!open) return
    const away = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', away)
    return () => document.removeEventListener('mousedown', away)
  }, [open])

  const options = () => Array.from(box.current?.querySelectorAll<HTMLAnchorElement>('[role="option"]') ?? [])
  const openMenu = () => {
    setOpen(true)
    // The selected channel takes focus once the list is drawn.
    requestAnimationFrame(() => (options().find((o) => o.getAttribute('aria-selected') === 'true') ?? options()[0])?.focus())
  }
  const onListKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const all = options()
    const i = all.indexOf(document.activeElement as HTMLAnchorElement)
    switch (e.key) {
      case 'Escape':
        e.preventDefault()
        setOpen(false)
        button.current?.focus()
        return
      case 'ArrowDown':
        e.preventDefault()
        all[(i + 1) % all.length]?.focus()
        return
      case 'ArrowUp':
        e.preventDefault()
        all[(i - 1 + all.length) % all.length]?.focus()
        return
      case 'Home':
        e.preventDefault()
        all[0]?.focus()
        return
      case 'End':
        e.preventDefault()
        all[all.length - 1]?.focus()
        return
      case 'Tab':
        setOpen(false)
    }
  }
  const option = (c: Channel) => {
    const on = c === current
    const body = (
      <>
        {channelIcon[c]}
        <span className="flex-1">{t(channelName[c])}</span>
        {on ? (
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="m3 8.5 3.2 3L13 4.5" />
          </svg>
        ) : null}
      </>
    )
    const props = { role: 'option', 'aria-selected': on, onClick: () => setOpen(false) } as const
    switch (c) {
      case 'email':
        return (
          <Link key={c} to="/w/$workspace/providers" params={params} {...props}>
            {body}
          </Link>
        )
      case 'sms':
        return (
          <Link key={c} to={setUp.get('sms') ? '/w/$workspace/providers/sms' : '/w/$workspace/providers/connect/sms'} params={params} {...props}>
            {body}
          </Link>
        )
      case 'push':
        return (
          <Link key={c} to="/w/$workspace/providers/push" params={params} {...props}>
            {body}
          </Link>
        )
      case 'whatsapp':
        return (
          <Link key={c} to="/w/$workspace/providers/whatsapp" params={params} {...props}>
            {body}
          </Link>
        )
    }
  }
  return (
    <div className="relative ml-1" ref={box}>
      <button
        ref={button}
        type="button"
        className="chbtn"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-label={t(m.channelButton, { channel: t(channelName[current]) })}
        onClick={() => (open ? setOpen(false) : openMenu())}
        onKeyDown={(e) => {
          if (!open && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
            e.preventDefault()
            openMenu()
          }
        }}
      >
        {channelIcon[current]}
        {t(channelName[current])}
        <svg className="chv" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
          <path d="m4 6 4 4 4-4" />
        </svg>
      </button>
      {open ? (
        <div id={listId} className="menu pv-chmenu" role="listbox" tabIndex={-1} aria-label={t(m.channelLabel)} onKeyDown={onListKey}>
          {channels.map(option)}
        </div>
      ) : null}
    </div>
  )
}
