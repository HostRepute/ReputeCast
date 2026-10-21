import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import { Popover as RPopover } from 'radix-ui'
import { useState, type FormEvent, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon, LogoMark } from '@/components/icons/Icon'
import { notificationDotColor } from '@/components/shell/NotificationsPopover'
import { Page } from '@/components/shell/Page'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { WorkspaceSwitcher } from '@/components/shell/WorkspaceSwitcher'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Tooltip } from '@/components/ui/Menu'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { notificationAgo } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { useAgencyCan, useAgencyMe, useRefreshAgency } from './api'
import { m } from './messages'
import { NewClientDialog } from './NewClientDialog'

const nav = [
  { to: '/w/$workspace/agency', label: m.navOverview, icon: 'overview', exact: true },
  { to: '/w/$workspace/agency/staff', label: m.navStaff, icon: 'audience' },
  { to: '/w/$workspace/agency/library', label: m.navLibrary, icon: 'layers' },
  { to: '/w/$workspace/agency/usage', label: m.navUsage, icon: 'barChart' },
] as const

/**
 * The Agency area's shell (agency accounts): the workspace shell's layout
 * with the agency's own sections, Overview · Staff · Library · Usage. The
 * rail and the phone's tab bar show the same four.
 */
export function AgencyShell() {
  const t = useT()
  return (
    <div className="app rc-app">
      <a href="#main" className="skip">
        {t(m.skip)}
      </a>
      <AgencySidebar />
      <div className="mainc">
        <SystemBanners />
        <Outlet />
      </div>
    </div>
  )
}

function AgencySidebar() {
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const { data: agency } = useAgencyMe()
  const t = useT()
  const rail = useMediaQuery('(min-width: 768px) and (max-width: 1100px)')
  const user = me?.user
  const level = agency?.staff?.level === 'agency_manager' ? t(m.levelAgencyManager) : t(m.levelAccountManager)
  const host = (
    <>
      <span className="dot" style={{ color: '#15803D' }} aria-hidden="true" />
      <span className="flex min-w-0 flex-col leading-[1.25]">
        <span className="truncate text-[12px] font-semibold">{agency?.host}</span>
        <span className="k text-[11px]">{t(m.hostLine, { clients: agency?.clientCount ?? 0, agency: agency?.agencyWorkspace ? 1 : 0 })}</span>
      </span>
    </>
  )
  return (
    <aside className="side" aria-label={t(m.sidebarLabel, { agency: agency?.name ?? '' })}>
      <Link to="/w/$workspace/agency" params={{ workspace: slug }} className="home" aria-label={t(m.homeLabel)}>
        <LogoMark size={26} />
        <span className="brand-t">ReputeCast</span>
      </Link>
      <WorkspaceSwitcher variant="agency" />
      <nav className="nav" aria-label={t(m.areaLabel)}>
        {nav.map((item) => {
          const link = (
            <Link key={item.to} to={item.to} params={{ workspace: slug }} activeOptions={{ exact: 'exact' in item, includeSearch: false }}>
              <Icon name={item.icon} />
              <span className="nl">{t(item.label)}</span>
            </Link>
          )
          return rail ? (
            <Tooltip key={item.to} content={t(item.label)}>
              {link}
            </Tooltip>
          ) : (
            link
          )
        })}
      </nav>
      <div className="sidefoot">
        {user?.isInstanceAdmin ? (
          <Link to="/w/$workspace/admin/instance" params={{ workspace: slug }} className="ag-instc">
            {host}
          </Link>
        ) : (
          <div className="ag-instc">{host}</div>
        )}
        {user ? (
          <div className="flex items-center gap-2 px-2 py-1">
            <Avatar name={user.name || user.email} color={user.avatarColor} size={26} />
            <div className="flex flex-col leading-[1.25]">
              <span className="text-[12px] font-semibold">{user.name || user.email}</span>
              <span className="k text-[11px]">{level}</span>
            </div>
          </div>
        ) : null}
      </div>
    </aside>
  )
}

/**
 * A screen of the Agency area: the top bar's search across clients, New
 * client (agency managers) and the bell that gathers every client's
 * notifications.
 */
export function AgencyPage({
  title,
  subtitle,
  actions,
  search = true,
  children,
}: {
  title: string
  subtitle?: ReactNode
  /** Replaces search and New client (Library, Usage: their own). */
  actions?: ReactNode
  search?: boolean
  children: ReactNode
}) {
  const canCreate = useAgencyCan('agency:manage')
  const [open, setOpen] = useState(false)
  const t = useT()
  return (
    <Page
      title={title}
      subtitle={subtitle}
      bell={<AgencyBell />}
      actions={
        actions ?? (
          <>
            {search ? <AgencySearchBox /> : null}
            {canCreate ? (
              <Button variant="primary" icon="plus" onClick={() => setOpen(true)}>
                {t(m.newClient)}
              </Button>
            ) : null}
            {open ? <NewClientDialog open onOpenChange={setOpen} /> : null}
          </>
        )
      }
    >
      {children}
    </Page>
  )
}

/** "Search all clients": names everywhere, contacts by exact address. */
export function AgencySearchBox({ initial = '', width = 260 }: { initial?: string; width?: number }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [q, setQ] = useState(initial)
  const t = useT()
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (q.trim().length < 2) return
    void navigate({ to: '/w/$workspace/agency/search', params: { workspace: slug }, search: { q: q.trim() } })
  }
  return (
    <form role="search" onSubmit={submit} className="max-md:w-full">
      <label
        className="flex h-9 items-center gap-2 rounded-lg border border-[#8B94A3] bg-white px-2.5 max-md:w-full!"
        style={{ width, maxWidth: '100%', boxSizing: 'border-box' }}
      >
        <Icon name="search" size={15} className="text-ink-3" />
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t(m.searchPlaceholder)}
          aria-label={t(m.searchLabel)}
          className="min-w-0 flex-1 border-0 bg-transparent text-[13px] outline-none"
        />
      </label>
    </form>
  )
}

/** The bell: the latest notifications from every client the person works in. */
export function AgencyBell() {
  const [open, setOpen] = useState(false)
  const t = useT()
  const list = $api.useQuery('get', '/api/v1/agency/notifications', { params: { query: { limit: 6 } } }, { staleTime: 30_000 })
  const n = list.data?.unread ?? 0
  return (
    <RPopover.Root open={open} onOpenChange={setOpen}>
      <RPopover.Trigger asChild>
        <button type="button" className="ibtn box-content" aria-label={n ? t(m.bellUnread, { count: n }) : t(m.bellTitle)}>
          <Icon name="bell" />
          {n ? (
            <span className="cnt" aria-hidden="true">
              {n > 99 ? '99+' : n}
            </span>
          ) : null}
        </button>
      </RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className="menu w-[420px] p-0!" align="end" sideOffset={10} alignOffset={-20} collisionPadding={12} aria-label={t(m.bellTitle)}>
          {open ? <BellPanel onClose={() => setOpen(false)} /> : null}
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}

function BellPanel({ onClose }: { onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const list = $api.useQuery('get', '/api/v1/agency/notifications', { params: { query: { limit: 6 } } })
  const count = list.data?.unread ?? 0
  return (
    <>
      <div className="sec" style={{ padding: '12px 14px' }}>
        <div className="row" style={{ gap: 8 }}>
          <h2 className="h2 text-[14px]!">{t(m.bellTitle)}</h2>
          {count ? <Badge tone="bad">{t(m.bellUnreadBadge, { count })}</Badge> : null}
        </div>
      </div>
      <div className="col" style={{ gap: 0, padding: 6 }}>
        {list.isError ? (
          <span className="err px-2.5 py-2" role="alert">
            {errorMessage(list.error)}
          </span>
        ) : list.isPending ? (
          <div aria-busy="true">
            <span className="vh" role="status">
              {t(m.bellLoading)}
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
            <span className="font-semibold">{t(m.bellEmptyTitle)}</span>
            <span className="k">{t(m.bellEmptyBody)}</span>
          </div>
        ) : (
          list.data.items.map((it) => (
            <AgencyNotificationLink key={it.id} n={it} slug={it.workspace.slug} onOpen={onClose} onRead={refresh}>
              <span className="dot mt-1.5" style={{ color: notificationDotColor(it.severity) }} aria-hidden="true" />
              <span className="col min-w-0 flex-1" style={{ gap: 1 }}>
                <span className="flex justify-between gap-2">
                  <b className="text-[13px]">{it.title}</b>
                  <span className="k whitespace-nowrap">
                    <time dateTime={it.createdAt}>{notificationAgo(it.createdAt)}</time>
                  </span>
                </span>
                <span className="k flex items-center gap-1.5">
                  <span className="ag-wst" style={{ width: 16, height: 16, borderRadius: 4, fontSize: 8, background: it.workspace.color }} data-initials={it.workspace.initials} aria-hidden="true" />
                  {it.workspace.name}
                </span>
                <span className="text-[12.5px] whitespace-normal text-ink-nav">{it.summary}</span>
              </span>
            </AgencyNotificationLink>
          ))
        )}
      </div>
      <div className="sec border-t border-line-2" style={{ borderBottom: 0, padding: '10px 14px' }}>
        <span className="k">{t(m.bellNote)}</span>
      </div>
    </>
  )
}

/** A notification from one of the clients: opens there, and marks itself read. */
function AgencyNotificationLink({
  n,
  slug,
  children,
  onOpen,
  onRead,
}: {
  n: { id: string; link: string; read: boolean }
  slug: string
  children: ReactNode
  onOpen: () => void
  onRead: () => void
}) {
  const navigate = useNavigate()
  const markRead = async () => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/notifications/{id}/read', { params: { path: { workspace: slug, id: n.id } } })
      onRead()
    } catch {
      // Opening still works; it stays unread.
    }
  }
  return (
    <a
      href={n.link || '#'}
      className={cn('items-start! min-h-0! px-2.5! py-2!', !n.read && 'bg-ground-2!')}
      onClick={(e) => {
        if (!n.read) void markRead()
        if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
        e.preventDefault()
        onOpen()
        if (n.link) void navigate({ to: n.link })
      }}
    >
      {children}
    </a>
  )
}
