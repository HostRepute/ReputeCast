import { $api } from '@/api/client'
import { NavTabs } from '@/components/ui/Tabs'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

/** The audience summary behind the tabs and the empty state. */
export function useAudienceSummary() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/audience', { params: { path: { workspace: slug } } })
}

/**
 * Contacts · Segments · List health · Suppression list, inline after the
 * screen title. Counts appear once the summary has loaded (and not at all
 * while the workspace has no contacts, as on the empty-state board).
 */
export function AudienceTabs() {
  const { slug } = useWorkspace()
  const summary = useAudienceSummary()
  const s = summary.data
  const show = s && s.contacts > 0
  const n = (v: number) => (show ? ` · ${count(v)}` : '')
  return (
    <NavTabs
      label="Audience sections"
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: `Contacts${n(s?.contacts ?? 0)}`, to: '/w/$workspace/audience', params: { workspace: slug }, exact: true },
        { label: `Segments${n(s?.segments ?? 0)}`, to: '/w/$workspace/audience/segments', params: { workspace: slug } },
        { label: 'List health', to: '/w/$workspace/audience/health', params: { workspace: slug } },
        { label: `Suppression list${n(s?.suppressions ?? 0)}`, to: '/w/$workspace/audience/suppressions', params: { workspace: slug } },
      ]}
    />
  )
}
