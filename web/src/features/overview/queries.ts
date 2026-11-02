import { $api } from '@/api/client'
import type { FirstQuery } from '@/lib/prefetch'

/*
 * The Overview's cards' queries. The page and its route's loader build them
 * here, so a prefetch and the page's own query are the same query; this
 * module stays small, as the route definitions load with the app.
 */

export function overviewQueries(slug: string) {
  const path = { workspace: slug }
  return {
    health: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/overview/sender-health', { params: { path } }),
    next: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/overview/next-campaign', { params: { path } }),
    providers: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/overview/providers', { params: { path } }),
    audience: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/overview/audience', { params: { path, query: { days: 30 } } }),
    recent: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/overview/recent-campaigns', { params: { path } }),
  }
}

/** The permission each card needs; a card the person can't see isn't asked for. */
export const overviewPerms = {
  health: 'deliverability:view',
  next: 'campaigns:view',
  providers: 'providers:view',
  audience: 'subscribers:view',
  recent: 'campaigns:view',
} as const

/** What the Overview asks for first, for its route's loader: every card it may show. */
export function overviewFirst(slug: string): FirstQuery[] {
  const q = overviewQueries(slug)
  return (Object.keys(q) as (keyof typeof q)[]).map((k) => ({ options: q[k], perm: overviewPerms[k] }))
}
