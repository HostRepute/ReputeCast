import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'

export type SiteList = Schemas['TrackerSiteList']
export type Site = Schemas['TrackerSite']
export type RecentEvent = Schemas['TrackerRecentEvent']
export type StoreOption = Schemas['TrackerStoreOption']

export const sitesPath = '/api/v1/workspaces/{workspace}/tracker/sites'
export const sitePath = '/api/v1/workspaces/{workspace}/tracker/sites/{site}'
export const recentPath = '/api/v1/workspaces/{workspace}/tracker/sites/{site}/events/recent'
export const secretPath = '/api/v1/workspaces/{workspace}/tracker/sites/{site}/secret'

/** The workspace's sites, their snippets and the stores they may link to. */
export function useSites() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', sitesPath, { params: { path: { workspace: slug } } })
}

/** A site's last 20 events; `live` polls every 5 seconds while the page shows them. */
export function useRecentEvents(site: string | undefined, live: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    recentPath,
    { params: { path: { workspace: slug, site: site ?? '' } } },
    { enabled: !!site, refetchInterval: live ? 5_000 : false, refetchIntervalInBackground: false },
  )
}

/** Refreshes the sites, their events and the integrations catalog's card. */
export function useRefreshSites() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey[1]
          return typeof k === 'string' && (k.startsWith('/api/v1/workspaces/{workspace}/tracker/') || k.startsWith('/api/v1/workspaces/{workspace}/integrations'))
        },
      }),
    [qc],
  )
}

/**
 * Reads what a person typed as a site's address: its origin, or why it
 * isn't one. A path, a query or a fragment is refused, as the board says
 * ("That's an address, not an origin").
 */
export function originOf(raw: string): { origin?: string; path?: string } {
  const v = raw.trim()
  if (!v) return {}
  let u: URL
  try {
    u = new URL(/^[a-z][a-z0-9+.-]*:\/\//i.test(v) ? v : `https://${v}`)
  } catch {
    return {}
  }
  if (u.protocol !== 'https:' && u.protocol !== 'http:') return {}
  if (!u.hostname || u.username || u.password) return {}
  const origin = `${u.protocol}//${u.host}`.toLowerCase()
  if ((u.pathname && u.pathname !== '/') || u.search || u.hash) return { path: origin }
  return { origin }
}
