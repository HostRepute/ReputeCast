import { useQueryClient } from '@tanstack/react-query'
import { Link, useRouter } from '@tanstack/react-router'
import { Popover as RPopover } from 'radix-ui'
import { useState, type MouseEvent, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { cn } from '@/lib/cn'
import { notificationAgo } from '@/lib/format'
import { useOptionalWorkspace, useWorkspace, workspaceMeKey } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { Badge } from '../ui/Badge'
import { Skeleton } from '../ui/Feedback'
import { toast } from '../ui/Toast'

export type NotificationItem = Schemas['NotificationItem']

/** Refetches the lists, the bell popover and the bell's unread count. */
export function useRefreshNotifications() {
  const qc = useQueryClient()
  const { slug } = useWorkspace()
  return () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/notifications'] }),
      qc.invalidateQueries({ queryKey: workspaceMeKey(slug) }),
    ])
}

/** Mark one or every notification as read. */
export function useNotificationActions() {
  const { slug } = useWorkspace()
  const refresh = useRefreshNotifications()
  const [marking, setMarking] = useState(false)
  const markRead = async (id: string) => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/notifications/{id}/read', { params: { path: { workspace: slug, id } } })
      await refresh()
    } catch {
      // Opening still works; the item stays unread.
    }
  }
  const markAll = async () => {
    setMarking(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/notifications/read-all', { params: { path: { workspace: slug } } })
      await refresh()
      if (data?.marked) toast({ kind: 'success', title: `${data.marked} ${data.marked === 1 ? 'notification' : 'notifications'} marked as read` })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't mark them as read", detail: errorMessage(e) })
    } finally {
      setMarking(false)
    }
  }
  return { markRead, markAll, marking }
}

/**
 * A notification that opens the screen where you can act, and marks itself
 * read. It is a real link, so it can also open in a new tab.
 */
export function NotificationLink({
  n,
  className,
  unreadClassName,
  children,
  role,
  onOpen,
  srUnread = true,
}: {
  n: NotificationItem
  className?: string
  unreadClassName?: string
  children: ReactNode
  role?: string
  onOpen?: () => void
  /** Adds "unread" for screen readers (when no visible "New" badge says it). */
  srUnread?: boolean
}) {
  const router = useRouter()
  const { markRead } = useNotificationActions()
  const href = n.link || '#'
  const click = (e: MouseEvent<HTMLAnchorElement>) => {
    if (!n.read) void markRead(n.id)
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
    e.preventDefault()
    onOpen?.()
    if (n.link) router.history.push(n.link)
  }
  return (
    <a href={href} className={cn(className, !n.read && unreadClassName)} role={role} onClick={click}>
      {children}
      {n.read || !srUnread ? null : <span className="vh">unread</span>}
    </a>
  )
}

/** Board dot colours per severity (warn-bar, bad-strong, ok-bar, info). */
export function notificationDotColor(severity: string) {
  switch (severity) {
    case 'warn':
      return '#B45309'
    case 'bad':
      return '#B91C1C'
    case 'ok':
      return '#15803D'
    default:
      return '#1D4ED8'
  }
}

/**
 * The bell in the top bar: unread count, and a popover with the latest
 * notifications, Mark all as read, All notifications and Preferences
 * (ShellNotifications board).
 */
export function NotificationsPopover() {
  const ws = useOptionalWorkspace()
  const [open, setOpen] = useState(false)
  if (!ws) return null
  const n = ws.me.unread
  return (
    <RPopover.Root open={open} onOpenChange={setOpen}>
      <RPopover.Trigger asChild>
        <button type="button" className="ibtn" aria-label={n ? `Notifications, ${n} unread` : 'Notifications'}>
          <Icon name="bell" />
          {n ? (
            <span className="cnt" aria-hidden="true">
              {n > 99 ? '99+' : n}
            </span>
          ) : null}
        </button>
      </RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className="menu w-[420px] p-0!" align="end" sideOffset={11} alignOffset={-20} collisionPadding={12} aria-label="Notifications">
          {open ? <Panel slug={ws.slug} unread={n} onClose={() => setOpen(false)} /> : null}
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}

function Panel({ slug, unread, onClose }: { slug: string; unread: number; onClose: () => void }) {
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/notifications', { params: { path: { workspace: slug }, query: { limit: 6 } } })
  const { markAll, marking } = useNotificationActions()
  const count = list.data?.unread ?? unread
  return (
    <>
      <div className="sec" style={{ padding: '12px 14px' }}>
        <div className="row" style={{ gap: 8 }}>
          <h2 className="h2 text-[14px]!">Notifications</h2>
          {count ? <Badge tone="bad">{count} unread</Badge> : null}
        </div>
        <button type="button" className="btn sm" disabled={marking || !count} onClick={markAll}>
          Mark all as read
        </button>
      </div>
      <div className="col" style={{ gap: 0, padding: 6 }}>
        {list.isError ? (
          <span className="err px-2.5 py-2" role="alert">
            {errorMessage(list.error)}
          </span>
        ) : list.isPending ? (
          <div aria-busy="true">
            <span className="vh" role="status">
              Loading notifications…
            </span>
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex gap-2.5 px-2.5 py-2">
                <Skeleton w={8} h={8} r={4} className="mt-1.5" />
                <div className="flex flex-1 flex-col gap-1.5">
                  <Skeleton w="40%" h={13} />
                  <Skeleton w="85%" h={12} />
                </div>
              </div>
            ))}
          </div>
        ) : list.data.items.length === 0 ? (
          <div className="note n m-1 flex-col items-center gap-1 text-center">
            <span className="font-semibold">You're all caught up</span>
            <span className="k">Approvals, auto-pauses, provider incidents, team changes and HostRepute alerts show up here.</span>
          </div>
        ) : (
          list.data.items.map((it) => (
            <NotificationLink key={it.id} n={it} className="items-start! min-h-0! px-2.5! py-2!" unreadClassName="bg-ground-2!" onOpen={onClose}>
              <span className="dot mt-1.5" style={{ color: notificationDotColor(it.severity) }} aria-hidden="true" />
              <span className="col min-w-0 flex-1" style={{ gap: 1 }}>
                <span className="flex justify-between gap-2">
                  <b className="text-[13px]">{it.title}</b>
                  <span className="k whitespace-nowrap">
                    <time dateTime={it.createdAt}>{notificationAgo(it.createdAt)}</time>
                  </span>
                </span>
                <span className="text-[12.5px] whitespace-normal text-ink-nav">{it.summary}</span>
              </span>
            </NotificationLink>
          ))
        )}
      </div>
      <div className="sec border-t border-line-2" style={{ borderBottom: 0, padding: '10px 14px' }}>
        <Link to="/w/$workspace/notifications" params={{ workspace: slug }} className="text-[13px] font-semibold!" onClick={onClose}>
          All notifications
        </Link>
        <Link to="/w/$workspace/notifications" params={{ workspace: slug }} hash="preferences" className="k" onClick={onClose}>
          Preferences
        </Link>
      </div>
    </>
  )
}
