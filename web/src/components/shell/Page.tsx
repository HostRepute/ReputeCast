import { useEffect, useRef, type ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { useDocumentTitle } from '@/lib/hooks'
import { useOptionalWorkspace } from '@/lib/session'
import { NavTabs, type NavTab } from '../ui/Tabs'
import { NotificationsPopover } from './NotificationsPopover'
import { UserMenu } from './UserMenu'

/**
 * One screen inside the app shell: the top bar (title, optional caption,
 * actions, bell and user menu, optional section tabs) and the content column.
 */
export function Page({
  title,
  documentTitle,
  subtitle,
  badges,
  filters,
  actions,
  tabs,
  tabsLabel = 'Sections',
  children,
  contentClassName,
}: {
  title: string
  /** Overrides the browser tab title (defaults to the title). */
  documentTitle?: string
  subtitle?: ReactNode
  badges?: ReactNode
  /** Filter tabs shown inline right after the title (Campaigns: All · Drafts · …). */
  filters?: ReactNode
  actions?: ReactNode
  tabs?: NavTab[]
  tabsLabel?: string
  children: ReactNode
  contentClassName?: string
}) {
  const ws = useOptionalWorkspace()
  useDocumentTitle(`${documentTitle ?? title}${ws ? ` · ${ws.me.workspace.name}` : ''} · ReputeCast`)
  const h1 = useRef<HTMLHeadingElement>(null)
  useEffect(() => {
    // Move focus to the screen title after client-side navigation so screen
    // reader users hear where they are (not on the first load).
    if (window.history.state && document.activeElement && document.activeElement !== document.body && document.activeElement.closest('.side, .tabs, .menu')) {
      h1.current?.focus({ preventScroll: true })
    }
  }, [title])
  return (
    <>
      <header className="top">
        <h1 ref={h1} tabIndex={-1}>
          {title}
        </h1>
        {subtitle ? <span className="k">{subtitle}</span> : null}
        {badges}
        {filters}
        <div className="acts">
          {actions}
          <div className="tools">
            <NotificationBell />
            <UserMenu />
          </div>
        </div>
        {tabs ? <NavTabs tabs={tabs} label={tabsLabel} className="sections" /> : null}
      </header>
      <main className={cn('content', contentClassName)} id="main" tabIndex={-1}>
        {children}
      </main>
    </>
  )
}

/** Bell with the unread count; opens the notifications popover. */
export function NotificationBell() {
  return <NotificationsPopover />
}
