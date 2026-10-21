import { useQueryClient } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import { useMe } from '@/lib/session'

export type AgencyMe = Schemas['AgencyMe']
export type AgencyClientRow = Schemas['AgencyClientRow']
export type AgencyWorkspaceRef = Schemas['AgencyWorkspaceRef']
export type AgencyHealth = Schemas['AgencyHealth']
export type AgencyOverview = Schemas['AgencyOverviewView']
export type AgencyApproval = Schemas['AgencyApproval']
export type AgencyIncident = Schemas['AgencyIncident']
export type AgencySend = Schemas['AgencySend']
export type AgencyPulse = Schemas['AgencyClientPulse']

/**
 * The agency as the shell shows it: for staff, its clients with their health
 * (the switcher, the More sheet, the Agency area's sidebar); for a client's
 * people, only its name. Off (`enabled: false`) on instances without an
 * agency, which is every instance until agency mode is turned on.
 */
export function useAgencyMe() {
  const { data: me } = useMe()
  return $api.useQuery('get', '/api/v1/agency/me', {}, { enabled: Boolean(me), staleTime: 60_000, retry: false })
}

/** The signed-in person is on the agency's staff. */
export function useIsStaff(): boolean {
  const { data } = useAgencyMe()
  return Boolean(data?.enabled && data.staff)
}

/** Refetches everything the Agency area shows after a change. */
export function useRefreshAgency() {
  const qc = useQueryClient()
  return () =>
    Promise.all([
      qc.invalidateQueries({ predicate: (q) => typeof q.queryKey[1] === 'string' && (q.queryKey[1] as string).startsWith('/api/v1/agency') }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/agency'] }),
    ])
}

/** The agency permissions the signed-in staff member holds. */
export function useAgencyCan(perm: 'agency:view' | 'agency:usage' | 'agency:library' | 'agency:manage'): boolean {
  const { data } = useAgencyMe()
  return Boolean(data?.staff?.permissions.includes(perm))
}
