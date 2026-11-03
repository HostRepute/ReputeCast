import { $api, type Schemas } from '@/api/client'
import type { FirstQuery } from '@/lib/prefetch'

/**
 * The workspace's channels, for a route's loader, and SMS's overview once
 * they say texting is set up: the queries the channel parts of Contacts and
 * the Overview ask for, with the same address and parameters as theirs.
 */
export function channelsFirst(slug: string): FirstQuery {
  const init = { params: { path: { workspace: slug } } }
  return {
    options: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/channels', init),
    perm: 'providers:view',
    next: (d?: Schemas['SmsChannels']) =>
      d?.channels.some((c) => c.channel === 'sms' && c.setUp) ? [{ options: $api.queryOptions('get', '/api/v1/workspaces/{workspace}/sms/overview', init), perm: 'providers:view' }] : [],
  }
}
