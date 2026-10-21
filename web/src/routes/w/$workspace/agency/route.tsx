import { createFileRoute, redirect } from '@tanstack/react-router'
import { http } from '@/api/client'
import { WorkspaceRouteError } from '@/components/shell/SystemScreens'
import { AgencyShell } from '@/features/agency/AgencyShell'

/**
 * The Agency area (agency accounts): Overview · Staff · Library · Usage,
 * for the agency's staff. Its URLs live under a workspace, as Instance
 * admin's do; anyone else is sent to the workspace's overview.
 */
export const Route = createFileRoute('/w/$workspace/agency')({
  beforeLoad: async ({ context, params }) => {
    const me = await context.queryClient.fetchQuery({
      queryKey: ['get', '/api/v1/agency/me', {}],
      queryFn: async () => (await http.GET('/api/v1/agency/me')).data,
      staleTime: 60_000,
    })
    if (!me?.enabled || !me.staff) throw redirect({ to: '/w/$workspace', params: { workspace: params.workspace } })
  },
  component: AgencyShell,
  errorComponent: WorkspaceRouteError,
})
