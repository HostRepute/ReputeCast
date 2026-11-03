import type { FetchQueryOptions, QueryClient } from '@tanstack/react-query'
import { fetchWorkspaceMe, workspaceMeKey, type Permission, type WorkspaceMe } from './session'

type HiddenSection = NonNullable<WorkspaceMe['hiddenSections']>[number]

/** A query a screen asks for first, and the permission its endpoint needs. */
export interface FirstQuery {
  // The options of `$api.queryOptions`, which carry their own data types.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  options: FetchQueryOptions<any, any, any, any>
  perm?: Permission
  /** A sidebar section the query belongs to: not asked for while the workspace hides it. */
  section?: HiddenSection
  /** Queries the screen asks for once this one's data says so (texting set up: its overview). */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  next?: (data: any) => FirstQuery[]
}

/**
 * Starts a screen's first queries from its route's loader, while the
 * screen's code is still on its way, instead of once the screen has mounted:
 * on a slow phone the two took turns, a second or more. The route doesn't
 * wait for them. Only what the person may see is asked for (the workspace's
 * permissions come with the route), and data already fresh in the cache is
 * left alone, so a link's preload on hover costs nothing twice.
 */
export function prefetchFirst(client: QueryClient, slug: string, queries: FirstQuery[]) {
  void client
    .ensureQueryData({ queryKey: workspaceMeKey(slug), queryFn: () => fetchWorkspaceMe(slug) })
    .then((me) => {
      const ask = (qs: FirstQuery[]) => {
        for (const q of qs) {
          if (q.perm && !me.permissions.includes(q.perm)) continue
          if (q.section && me.hiddenSections?.includes(q.section)) continue
          const asked = client.prefetchQuery(q.options)
          if (q.next) void asked.then(() => ask(q.next!(client.getQueryData(q.options.queryKey))))
          else void asked
        }
      }
      ask(queries)
    })
    // The route's own loader reports a workspace that can't be read.
    .catch(() => undefined)
}
