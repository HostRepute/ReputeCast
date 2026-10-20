import { useQueryClient } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import { useCan, useWorkspace } from '@/lib/session'

/** The website's API paths (internal/website). */
export const WS = '/api/v1/workspaces/{workspace}/website' as const

export type Overview = Schemas['WebsiteOverview']
export type Address = Schemas['WebsiteAddress']
export type PostRow = Schemas['WebsitePostRow']
export type Post = Schemas['WebsitePost']
export type PostList = Schemas['WebsitePostList']
export type PostInput = Schemas['WebsitePostInput']
export type HomeNav = Schemas['WebsiteHomeNav']
export type HomeNavInput = Schemas['WebsiteHomeNavInput']
export type HomeSection = Schemas['WebsiteHomeSection']
export type NavLink = Schemas['WebsiteNavLink']
export type Tag = Schemas['WebsiteTag']
export type Access = Schemas['WebsiteAccess']
export type AccessInput = Schemas['WebsiteAccessInput']
export type PastCampaign = Schemas['WebsitePastCampaign']
export type CampaignWeb = Schemas['WebsiteCampaignWeb']
export type WebCheck = Schemas['WebsiteWebCheck']
export type Visibility = 'public' | 'members' | 'paid'

/**
 * The website's overview: status, address, numbers and counts. While it is
 * turning on, it polls so the progress moves.
 */
export function useWebsite(enabled = true) {
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  return $api.useQuery(
    'get',
    WS,
    { params: { path: { workspace: slug } } },
    { enabled: enabled && can, refetchInterval: (q) => (q.state.data?.status === 'turning_on' ? 1500 : false) },
  )
}

/** Refreshes every website query (and the campaigns' Web column) after a change. */
export function useInvalidateWebsite() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const k = q.queryKey[1]
        return typeof k === 'string' && (k.includes('/website') || k.includes('/campaigns-web') || k.endsWith('/web') || k.includes('/approvals'))
      },
    })
}

/** "news.acme.io" or "reputecast.acme.io/p/acme-newsletter": an address without its scheme. */
export function bare(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** The site's address for a path on it ("/posts/x" → https://news.acme.io/posts/x). */
export function siteURL(address: Address, path = ''): string {
  return address.url.replace(/\/$/, '') + path
}
