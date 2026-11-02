import { keepPreviousData } from '@tanstack/react-query'
import { $api } from '@/api/client'
import type { FirstQuery } from '@/lib/prefetch'
import type { CampaignsSearch } from './CampaignsPage'

/*
 * The queries the campaigns list asks for first. The page and its route's
 * loader build them here, so a prefetch and the page's own query are the
 * same query; this module stays small, as the route definitions load with
 * the app.
 */

export const PAGE_SIZE = 25

/** A page of the campaigns list for a search; `range` is the "when" filter's days. */
export function campaignListQuery(slug: string, search: CampaignsSearch, range: { from?: string; to?: string }) {
  const status = search.status ?? 'all'
  return $api.queryOptions(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns',
    {
      params: {
        path: { workspace: slug },
        // Paid only (Earn): every status, campaigns for paid members only.
        query: {
          status: status === 'recurring' || status === 'paid' ? 'all' : status,
          paidOnly: status === 'paid' || undefined,
          q: search.q ?? '',
          channel: search.channel,
          from: range.from,
          to: range.to,
          sender: search.sender,
          list: search.list,
          tag: search.tag,
          page: search.page ?? 1,
          pageSize: PAGE_SIZE,
        },
      },
    },
    { placeholderData: keepPreviousData },
  )
}

/** The KPIs over the last 30 days above the list. */
export const campaignSummaryQuery = (slug: string) => $api.queryOptions('get', '/api/v1/workspaces/{workspace}/campaigns/summary', { params: { path: { workspace: slug }, query: { days: 30 } } })

/**
 * What the campaigns list asks for first, for its route's loader. The list
 * only as the menu opens it: with filters or a page in the address, the
 * screen asks for its list itself.
 */
export function campaignsFirst(slug: string, search: object): FirstQuery[] {
  const perm = 'campaigns:view'
  const out: FirstQuery[] = [{ options: campaignSummaryQuery(slug), perm }]
  if (Object.keys(search).length === 0) out.push({ options: campaignListQuery(slug, {}, {}), perm })
  return out
}
