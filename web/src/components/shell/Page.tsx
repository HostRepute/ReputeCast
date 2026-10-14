import { Link, type LinkProps } from '@tanstack/react-router'
import { Fragment, useEffect, useRef, type ReactNode } from 'react'
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
  breadcrumb,
  toolsAfterTabs,
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
  /**
   * Replaces the visible title with a breadcrumb ("Settings / Team / Daniel
   * Park"); the h1 stays for screen readers. The last crumb has no link.
   */
  breadcrumb?: Crumb[]
  /**
   * Put the actions, bell and user menu on their own row after the section
   * tabs (Settings › Senders, where a badge takes the end of the title row).
   */
  toolsAfterTabs?: boolean
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
  const acts = (
    <div className="acts">
      {actions}
      <div className="tools">
        <NotificationBell />
        <UserMenu />
      </div>
    </div>
  )
  return (
    <>
      <header className="top">
        {breadcrumb ? (
          <nav aria-label="Breadcrumb" className="flex flex-wrap items-center gap-2 text-[13px]">
            {breadcrumb.map((c, i) => (
              <Fragment key={i}>
                {i > 0 ? <span className="k">/</span> : null}
                {c.to ? (
                  <Link to={c.to} params={c.params as never} className="text-ink-3 hover:text-ink">
                    {c.label}
                  </Link>
                ) : (
                  <span className="font-semibold">{c.label}</span>
                )}
              </Fragment>
            ))}
          </nav>
        ) : null}
        <h1 ref={h1} tabIndex={-1} className={breadcrumb ? 'vh' : undefined}>
          {title}
        </h1>
        {subtitle ? <span className="k">{subtitle}</span> : null}
        {badges}
        {filters}
        {toolsAfterTabs ? null : acts}
        {tabs ? <NavTabs tabs={tabs} label={tabsLabel} className="sections" /> : null}
        {toolsAfterTabs ? acts : null}
      </header>
      <main className={cn('content', contentClassName)} id="main" tabIndex={-1}>
        {children}
      </main>
    </>
  )
}

export type Crumb = { label: string; to?: LinkProps['to']; params?: LinkProps['params'] }

/** Bell with the unread count; opens the notifications popover. */
export function NotificationBell() {
  return <NotificationsPopover />
}
