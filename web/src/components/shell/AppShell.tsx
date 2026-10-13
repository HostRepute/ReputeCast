import { Link, Outlet } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { useMediaQuery } from '@/lib/hooks'
import { ago } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { Icon, LogoMark, type IconName } from '../icons/Icon'
import { Avatar } from '../ui/Avatar'
import { Tooltip } from '../ui/Menu'
import { SystemBanners } from './SystemBanners'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

type NavItem = { to: string; label: string; icon: IconName; exact?: boolean }

const NAV: NavItem[] = [
  { to: '/w/$workspace', label: 'Overview', icon: 'overview', exact: true },
  { to: '/w/$workspace/campaigns', label: 'Campaigns', icon: 'campaigns' },
  { to: '/w/$workspace/automations', label: 'Automations', icon: 'automations' },
  { to: '/w/$workspace/audience', label: 'Audience', icon: 'audience' },
  { to: '/w/$workspace/forms', label: 'Forms & pages', icon: 'forms' },
  { to: '/w/$workspace/deliverability', label: 'Deliverability', icon: 'deliverability' },
  { to: '/w/$workspace/providers', label: 'Sending providers', icon: 'providers' },
  { to: '/w/$workspace/settings', label: 'Settings', icon: 'settings' },
]

/**
 * The workspace app shell: a 232px sidebar that becomes a 72px icon rail at
 * 1100px and a bottom tab bar at 767px. Screens render their own top bar and
 * content with <Page>. System banners (offline, maintenance) sit at the top
 * of the content column. `children` replaces the routed screen, for the
 * in-shell 404 and 500 states.
 */
export function AppShell({ children }: { children?: ReactNode } = {}) {
  return (
    <div className="app rc-app">
      <a href="#main" className="skip">
        Skip to content
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
  const rail = useMediaQuery('(min-width: 768px) and (max-width: 1100px)')
  const user = me?.user
  return (
    <aside className="side" aria-label="Workspace navigation">
      <Link to="/w/$workspace" params={{ workspace: slug }} className="home" aria-label="ReputeCast home">
        <LogoMark size={26} />
        <span className="brand-t">ReputeCast</span>
      </Link>
      <WorkspaceSwitcher />
      <nav className="nav" aria-label="Main">
        {NAV.map((item) => {
          // The router marks the current item with aria-current="page"; the
          // overview is the workspace root, so it only matches exactly.
          const link = (
            <Link key={item.to} to={item.to} params={{ workspace: slug }} activeOptions={{ exact: Boolean(item.exact), includeSearch: false }}>
              <Icon name={item.icon} />
              <span className="nl">{item.label}</span>
            </Link>
          )
          return rail ? (
            <Tooltip key={item.to} content={item.label}>
              {link}
            </Tooltip>
          ) : (
            link
          )
        })}
      </nav>
      <div className="sidefoot">
        <Link to="/w/$workspace/import" params={{ workspace: slug }} className="btn" style={{ justifyContent: 'flex-start' }} aria-label="Import contacts">
          <Icon name="import" />
          Import contacts
        </Link>
        <HostReputeStatus />
        {user ? (
          <div className="flex items-center gap-2 px-2 py-1">
            <Avatar name={user.name || user.email} color={user.avatarColor} size={26} />
            <div className="flex flex-col leading-[1.25]">
              <span className="text-[12px] font-semibold">{user.name || user.email}</span>
              <span className="k text-[11px]">{ws.workspace.roleName}</span>
            </div>
          </div>
        ) : null}
      </div>
    </aside>
  )
}

function HostReputeStatus() {
  const { slug, me } = useWorkspace()
  const hr = me.hostRepute
  const connected = hr?.status === 'connected'
  return (
    <Link
      to="/w/$workspace/settings/integrations"
      params={{ workspace: slug }}
      className="flex items-center gap-2 rounded-lg border border-line bg-white px-2.5 py-2 text-ink hover:text-ink"
    >
      <span className="dot" style={{ color: connected ? '#15803D' : hr ? '#B45309' : '#8B94A3' }} aria-hidden="true" />
      <div className="flex flex-col leading-[1.25]">
        <span className="text-[12px] font-semibold">{connected ? 'HostRepute connected' : hr ? 'HostRepute needs attention' : 'Connect HostRepute'}</span>
        <span className="k text-[11px]">
          {connected ? `${hr.teamName} · synced ${ago(hr.lastSyncAt)}` : hr ? 'Reconnect to resume monitoring' : 'Optional · no paywall'}
        </span>
      </div>
    </Link>
  )
}

export type { ReactNode }
