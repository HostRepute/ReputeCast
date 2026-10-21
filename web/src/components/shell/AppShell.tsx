import { Link, Outlet, useMatchRoute, useRouterState } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ManagedByCard, useIsClientPerson } from '@/features/agency/WorkspaceParts'
import { formatList, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useMediaQuery } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { Icon, LogoMark } from '../icons/Icon'
import { Avatar } from '../ui/Avatar'
import { Tooltip } from '../ui/Menu'
import { HostReputeStatus } from './HostReputeStatus'
import { m } from './messages'
import { MoreSheet } from './MoreSheet'
import { moreIsCurrent, phoneNav, sectionOf, visibleNav, type NavItem, type SectionKey } from './nav'
import { SystemBanners } from './SystemBanners'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'
import './shell.css'

/**
 * The workspace app shell: a 232px sidebar that becomes a 72px icon rail at
 * 1100px and, at 767px, a bottom tab bar with four sections and More, which
 * opens a sheet with the rest. Sections the workspace hides (Settings ›
 * General) are left out everywhere; their screens still open by link.
 * `children` replaces the routed screen, for the in-shell 404 and 500 states.
 */
export function AppShell({ children }: { children?: ReactNode } = {}) {
  const t = useT()
  return (
    <div className="app rc-app">
      <a href="#main" className="skip">
        {t(m.skipToContent)}
      </a>
      <Sidebar />
      <div className="mainc">
        <SystemBanners />
        {children ?? <Outlet />}
      </div>
    </div>
  )
}

function Sidebar() {
  const { slug, me: ws } = useWorkspace()
  const { data: me } = useMe()
  const t = useT()
  const rail = useMediaQuery('(min-width: 768px) and (max-width: 1100px)')
  const phone = useMediaQuery('(max-width: 767px)')
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const current = sectionOf(pathname)
  const hidden = ws.hiddenSections ?? []
  const { more } = phoneNav(hidden)
  // The sheet belongs to the screen it was opened on: navigating, or leaving
  // phone width, closes it.
  const [openOn, setOpenOn] = useState<string>()
  const sheetOpen = phone && openOn === pathname
  const user = me?.user
  // An agency's client people see who manages the workspace (agency accounts).
  const clientPerson = useIsClientPerson()
  return (
    <aside className={cn('side', sheetOpen && 'sheet-open')} aria-label={t(m.sidebarLabel)}>
      <Link to="/w/$workspace" params={{ workspace: slug }} className="home" aria-label={t(m.homeLabel)}>
        <LogoMark size={26} />
        <span className="brand-t">ReputeCast</span>
      </Link>
      <WorkspaceSwitcher />
      <nav className="nav" aria-label={t(m.mainNavLabel)}>
        {visibleNav(hidden).map((item) => (
          <NavLink key={item.key} item={item} current={current} rail={rail} />
        ))}
        <MoreSheet open={sheetOpen} onOpenChange={(o) => setOpenOn(o ? pathname : undefined)} items={more} current={current}>
          <button
            type="button"
            className={cn('more', moreIsCurrent(current, hidden, sheetOpen) && 'on')}
            aria-label={t(m.moreTabLabel, { sections: formatList(more.map((i) => t(i.label))) })}
            aria-current={moreIsCurrent(current, hidden) ? 'true' : undefined}
          >
            <Icon name="moreSections" />
            <span className="nl">{t(m.moreTab)}</span>
          </button>
        </MoreSheet>
      </nav>
      <div className="sidefoot">
        {clientPerson ? null : (
          <Link to="/w/$workspace/import" params={{ workspace: slug }} className="btn" style={{ justifyContent: 'flex-start' }} aria-label={t(m.importContacts)}>
            <Icon name="import" />
            {t(m.importContacts)}
          </Link>
        )}
        {clientPerson ? <ManagedByCard /> : <HostReputeStatus />}
        {user ? (
          <div className="flex items-center gap-2 px-2 py-1">
            <Avatar name={user.name || user.email} color={user.avatarColor} size={26} />
            <div className="flex flex-col leading-[1.25]">
              <span className="text-[12px] font-semibold">{user.name || user.email}</span>
              <span className="k text-[11px]">{roleName({ key: ws.workspace.roleKey, name: ws.workspace.roleName })}</span>
            </div>
          </div>
        ) : null}
      </div>
    </aside>
  )
}

/**
 * One section. The router marks it current (aria-current="page") on its own
 * screens; it is also current, as a section, on the screens it stands for
 * (approvals under Campaigns, Instance admin under Settings). On phones the
 * sections that live in the More sheet (`xm`) leave the tab bar.
 */
function NavLink({ item, current, rail }: { item: NavItem; current: SectionKey | undefined; rail: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const matchRoute = useMatchRoute()
  const label = t(item.label)
  const section = current === item.key && item.also?.some((to) => matchRoute({ to, params: { workspace: slug }, fuzzy: true }))
  const link = (
    <Link
      to={item.to}
      params={{ workspace: slug }}
      activeOptions={{ exact: Boolean(item.exact), includeSearch: false }}
      className={cn(!item.tab && 'xm', section && 'on')}
      aria-current={section ? 'true' : undefined}
    >
      <Icon name={item.icon} />
      <span className="nl">{label}</span>
    </Link>
  )
  return rail ? <Tooltip content={label}>{link}</Tooltip> : link
}

export type { ReactNode }
