import { Link, useNavigate } from '@tanstack/react-router'
import { DropdownMenu } from 'radix-ui'
import { useMemo, useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { roleName } from '@/lib/roles'
import { useMe } from '@/lib/session'
import type { AgencyClientRow, AgencyMe } from './api'
import { m } from './messages'
import { ClientTile, healthColors, isToday } from './parts'

/**
 * The workspace switcher for the agency's staff (ShellSwitcher, phase 8):
 * the Agency area first, then the clients with their health dots, then the
 * agency's own workspace, and the person's other workspaces. A search field
 * appears once there are more than 8 workspaces.
 */
export function AgencySwitcherMenu({ agency, current }: { agency: AgencyMe; current?: string }) {
  const t = useT()
  const navigate = useNavigate()
  const { data: me } = useMe()
  const [q, setQ] = useState('')
  const home = agency.home || current || ''
  const clients = useMemo(() => [...agency.clients].sort((a, b) => a.createdAt.localeCompare(b.createdAt)), [agency.clients])
  const agencySlugs = new Set([...agency.clients.map((c) => c.workspace.slug), agency.agencyWorkspace?.workspace.slug])
  const others = (me?.workspaces ?? []).filter((w) => !agencySlugs.has(w.slug))
  const total = clients.length + (agency.agencyWorkspace ? 1 : 0) + others.length
  const searching = total > 8
  const match = (name: string) => !q.trim() || name.toLowerCase().includes(q.trim().toLowerCase())
  const shown = clients.filter((c) => match(c.workspace.name))
  const own = agency.agencyWorkspace && match(agency.agencyWorkspace.workspace.name) ? agency.agencyWorkspace : undefined
  const otherShown = others.filter((w) => match(w.name))
  return (
    <>
      {searching ? (
        <div className="px-1.5 pt-1 pb-1.5">
          <label className="flex h-9 items-center gap-2 rounded-lg border border-[#8B94A3] bg-white px-2.5">
            <Icon name="search" size={15} className="text-ink-3" />
            <input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              onKeyDown={(e) => e.stopPropagation()}
              placeholder={t(m.switcherSearch)}
              aria-label={t(m.switcherSearch)}
              className="min-w-0 flex-1 border-0 bg-transparent text-[13px] outline-none"
            />
          </label>
        </div>
      ) : null}
      {home && !q.trim() ? (
        <DropdownMenu.Item asChild onSelect={() => navigate({ to: '/w/$workspace/agency', params: { workspace: home } })}>
          <Link to="/w/$workspace/agency" params={{ workspace: home }} className="h-auto! py-2!" style={{ background: '#F0FDFA' }}>
            <ClientTile ws={{ initials: agency.initials, color: '#0B1220' }} size={28} />
            <span className="col min-w-0 flex-1" style={{ gap: 0 }}>
              <b className="text-[13px]">{t(m.agencyEntry, { agency: agency.name })}</b>
              <span className="k">{t(m.agencyEntryLine)}</span>
            </span>
            <Icon name="chevronRight" size={14} className="text-accent" />
          </Link>
        </DropdownMenu.Item>
      ) : null}
      {shown.length ? (
        <>
          <DropdownMenu.Separator />
          <DropdownMenu.Label className="mh">{t(m.switcherClients, { count: clients.length })}</DropdownMenu.Label>
          {shown.map((c) => (
            <ClientItem key={c.workspace.id} row={c} current={current} />
          ))}
        </>
      ) : null}
      {own ? (
        <>
          <DropdownMenu.Separator />
          <DropdownMenu.Label className="mh">{t(m.switcherAgencyWorkspace)}</DropdownMenu.Label>
          <ClientItem row={own} current={current} />
        </>
      ) : null}
      {otherShown.length ? (
        <>
          <DropdownMenu.Separator />
          <DropdownMenu.Label className="mh">{t(m.switcherOther)}</DropdownMenu.Label>
          {otherShown.map((w) => (
            <DropdownMenu.Item key={w.id} asChild onSelect={() => navigate({ to: '/w/$workspace', params: { workspace: w.slug } })}>
              <Link to="/w/$workspace" params={{ workspace: w.slug }} aria-current={w.slug === current ? 'true' : undefined}>
                <ClientTile ws={{ initials: initialsOf(w.name), color: '#475569' }} size={26} />
                <span className="col min-w-0 flex-1" style={{ gap: 0 }}>
                  <b className="text-[13px]">{w.name}</b>
                  <span className="k">{roleName({ key: w.roleKey, name: w.roleName })}</span>
                </span>
              </Link>
            </DropdownMenu.Item>
          ))}
        </>
      ) : null}
      {q.trim() && !shown.length && !own && !otherShown.length ? <p className="k px-2.5 py-2">{t(m.switcherNoMatch, { q: q.trim() })}</p> : null}
      {agency.staff?.permissions.includes('agency:manage') && home ? (
        <>
          <DropdownMenu.Separator />
          <DropdownMenu.Item asChild>
            <Link to="/w/$workspace/agency" params={{ workspace: home }} search={{ new: 1 }}>
              <Icon name="plus" strokeWidth={1.6} />
              {t(m.newClient)}
            </Link>
          </DropdownMenu.Item>
        </>
      ) : null}
      <p className="k px-2.5 pt-1 pb-2" style={{ fontSize: 12 }}>
        {t(m.switcherNote)}
      </p>
    </>
  )
}

function ClientItem({ row, current }: { row: AgencyClientRow; current?: string }) {
  const t = useT()
  const navigate = useNavigate()
  const role = row.role ? roleName({ key: row.role.key, name: row.role.name }) : ''
  const today = isToday(row.createdAt)
  const line = row.settingUp && today ? t(m.switcherCreatedToday, { role }) : row.attention ? t(m.switcherNeedsAttention, { role }) : t(m.switcherClientLine, { role, contacts: row.contacts })
  const h = row.health
  return (
    <DropdownMenu.Item asChild onSelect={() => navigate({ to: '/w/$workspace', params: { workspace: row.workspace.slug } })}>
      <Link
        to="/w/$workspace"
        params={{ workspace: row.workspace.slug }}
        aria-current={row.workspace.slug === current ? 'true' : undefined}
        style={row.workspace.slug === current ? { background: '#E3F1EE' } : undefined}
        className="h-auto! py-1.5!"
      >
        <ClientTile ws={row.workspace} size={26} />
        <span className="col min-w-0 flex-1" style={{ gap: 0 }}>
          <b className="text-[13px]">{row.workspace.name}</b>
          <span className="k">{line}</span>
        </span>
        {row.settingUp ? (
          <span className="k text-[12px]">{t(m.switcherSettingUp)}</span>
        ) : h && h.state === 'ready' ? (
          <span className="ag-hsc text-[12px]">
            <span className="dot" style={{ color: healthColors(h.tone).dot }} aria-hidden="true" />
            {h.score}
          </span>
        ) : null}
      </Link>
    </DropdownMenu.Item>
  )
}

function initialsOf(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('')
}
