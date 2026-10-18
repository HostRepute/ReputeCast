import { $api } from '@/api/client'
import { useWorkspace } from '@/lib/session'

/** The workspace's surveys (subscribers:view). */
export function useSurveys(enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys', { params: { path: { workspace: slug } } }, { enabled, retry: 1 })
}
