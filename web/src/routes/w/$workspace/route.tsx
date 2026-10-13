import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { ApiError } from '@/api/client'
import { AppShell } from '@/components/shell/AppShell'
import { WorkspaceNotFound } from '@/components/shell/SystemScreens'
import { WorkspaceSystemLayer } from '@/components/shell/WorkspaceSystemLayer'
import { setDisplayTimeZone } from '@/lib/format'
import { fetchMe, fetchWorkspaceMe, meQueryKey, WorkspaceProvider, workspaceMeKey } from '@/lib/session'

/**
 * Every workspace screen lives under /w/$workspace. The loader makes sure the
 * person is signed in and a member, and loads their permissions there.
 */
export const Route = createFileRoute('/w/$workspace')({
  beforeLoad: async ({ context, location }) => {
    const me = await context.queryClient.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe })
    if (!me) throw redirect({ to: '/sign-in', search: { next: location.href } })
    if (me.pending === 'password_change' || me.pending === 'instance_setup' || me.pending === 'two_factor_setup') throw redirect({ to: '/setup' })
  },
  loader: async ({ context, params }) => {
    try {
      await context.queryClient.ensureQueryData({ queryKey: workspaceMeKey(params.workspace), queryFn: () => fetchWorkspaceMe(params.workspace) })
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) throw redirect({ to: '/' })
      throw e
    }
  },
  component: WorkspaceLayout,
  // Unknown paths under a workspace that match no screen at all.
  notFoundComponent: () => (
    <AppShell>
      <WorkspaceNotFound />
    </AppShell>
  ),
})

function WorkspaceLayout() {
  const { workspace } = Route.useParams()
  const { data } = useSuspenseQuery({ queryKey: workspaceMeKey(workspace), queryFn: () => fetchWorkspaceMe(workspace), staleTime: 30_000 })
  // Idempotent: every date below renders in the workspace's timezone.
  setDisplayTimeZone(data.timezone)
  return (
    <WorkspaceProvider slug={workspace} me={data}>
      <WorkspaceSystemLayer>
        <Outlet />
      </WorkspaceSystemLayer>
    </WorkspaceProvider>
  )
}
