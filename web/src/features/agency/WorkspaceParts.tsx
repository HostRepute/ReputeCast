import { Link } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { useOptionalWorkspace } from '@/lib/session'
import { useAgencyMe } from './api'
import { m } from './messages'
import { ClientTile } from './parts'

/**
 * The agency that manages the current workspace (agency accounts): the
 * "Managed by" card, "via Northwind Studio" on Team, the client details in
 * Settings › General. Nothing on instances without an agency.
 */
export function useWorkspaceAgency() {
  const ws = useOptionalWorkspace()
  const { data: agency } = useAgencyMe()
  const enabled = Boolean(ws && agency?.enabled)
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/agency', { params: { path: { workspace: ws?.slug ?? '' } } }, { enabled, staleTime: 60_000, retry: false })
}

/** The client's own people in a workspace the agency manages: they see "Managed by", not the agency's tools. */
export function useIsClientPerson(): boolean {
  const { data } = useWorkspaceAgency()
  return Boolean(data?.managed && data.kind === 'client' && !data.viewerIsStaff)
}

/** The sidebar card client people see: "Managed by Northwind Studio · Your contact: Mei Lin". */
export function ManagedByCard() {
  const t = useT()
  const { data } = useWorkspaceAgency()
  if (!data?.managed) return null
  return (
    <div className="flex items-center gap-2 rounded-lg border border-line bg-white px-2.5 py-2" aria-label={t(m.managedBy, { agency: data.agency ?? '' })}>
      <ClientTile ws={{ initials: data.agencyInitials ?? '', color: '#0B1220' }} size={22} />
      <div className="flex min-w-0 flex-col leading-[1.25]">
        <span className="text-[12px] font-semibold">{t(m.managedBy, { agency: data.agency ?? '' })}</span>
        {data.contact ? <span className="k text-[11px]">{t(m.yourContact, { name: data.contact.name })}</span> : null}
      </div>
    </div>
  )
}

/**
 * Settings › Team: a membership that comes from the agency, as an SSO
 * group mapping does ("via Northwind Studio · Account manager · your
 * contact"). Null for everyone else.
 */
export function AgencyMemberSource({ userId, source }: { userId?: string; source?: string }) {
  const t = useT()
  const { data } = useWorkspaceAgency()
  if (!data?.managed || source !== 'agency' || !userId) return null
  const staff = data.staff.find((s) => s.userId === userId)
  const level = staff?.level === 'agency_manager' ? t(m.levelAgencyManager) : t(m.levelAccountManager)
  const line = !staff
    ? ''
    : staff.everyClient && staff.level === 'agency_manager'
      ? t(m.viaEveryClient, { level: level.toLowerCase() })
      : staff.contact
        ? t(m.viaContact, { level })
        : staff.title
          ? t(m.viaTitle, { level, title: staff.title })
          : level
  return (
    <span className="flex flex-col gap-0.5">
      <span className="flex items-center gap-1.5">
        <ClientTile ws={{ initials: data.agencyInitials ?? '', color: '#0B1220' }} size={16} style={{ fontSize: 7, borderRadius: 4 }} />
        {t(m.via, { agency: data.agency ?? '' })}
      </span>
      {line ? <span className="k">{line}</span> : null}
    </span>
  )
}

/** Settings › Team: the action cell of an agency membership. */
export function AgencyMemberManaged({ source }: { source?: string }) {
  const t = useT()
  const { data } = useWorkspaceAgency()
  const { data: agency } = useAgencyMe()
  if (!data?.managed || source !== 'agency') return null
  if (data.viewerIsStaff && agency?.home) {
    return (
      <Link to="/w/$workspace/agency/staff" params={{ workspace: agency.home }} className="k whitespace-nowrap">
        {t(m.managedByAgency)}
      </Link>
    )
  }
  return <span className="k whitespace-nowrap">{t(m.managedByAgency)}</span>
}

/** The invite dialog in a client workspace: the email carries the agency's logo. */
export function AgencyInviteNote() {
  const t = useT()
  const { data } = useWorkspaceAgency()
  if (!data?.managed || data.kind !== 'client') return null
  return <span className="k">{t(m.inviteLogo, { agency: data.agency ?? '' })}</span>
}

/**
 * The More sheet's first entry for the agency's staff (ShellMore, phase 8):
 * the Agency area, above the workspace switcher.
 */
export function AgencySheetEntry({ onFollow }: { onFollow: () => void }) {
  const t = useT()
  const { data: agency } = useAgencyMe()
  if (!agency?.enabled || !agency.staff || !agency.home) return null
  return (
    <Link
      to="/w/$workspace/agency"
      params={{ workspace: agency.home }}
      className="flex items-center gap-2.5 rounded-[10px] border px-3 py-2.5 text-ink! no-underline"
      style={{ borderColor: '#99D5C9', background: '#F0FDFA' }}
      onClick={onFollow}
    >
      <ClientTile ws={{ initials: agency.initials, color: '#0B1220' }} size={28} />
      <span className="flex min-w-0 flex-1 flex-col">
        <b className="text-[13.5px]">{t(m.agencyEntry, { agency: agency.name })}</b>
        <span className="k">{t(m.agencyEntrySheetLine)}</span>
      </span>
      <Icon name="chevronRight" size={14} className="text-accent" />
    </Link>
  )
}
