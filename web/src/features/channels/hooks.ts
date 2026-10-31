import { $api } from '@/api/client'
import { useOptionalWorkspace } from '@/lib/session'
import type { Channel } from './channel'

/** Whether a channel is set up in the current workspace (false while loading). */
export function useChannelSetUp(channel: Channel): boolean {
  const ws = useOptionalWorkspace()
  const slug = ws?.slug ?? ''
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { enabled: Boolean(slug), staleTime: 60_000 })
  return Boolean(q.data?.channels.find((c) => c.channel === channel)?.setUp)
}
