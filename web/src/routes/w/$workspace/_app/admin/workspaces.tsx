import { createFileRoute } from '@tanstack/react-router'
import { AdminPage } from '@/features/admin/AdminPage'
import { WorkspacesCard } from '@/features/admin/instance/WorkspacesCard'

/** Instance admin › Workspaces: the workspaces card of InstanceSettings on its own. */
export const Route = createFileRoute('/w/$workspace/_app/admin/workspaces')({
  component: function AdminWorkspacesRoute() {
    return (
      <AdminPage documentTitle="Workspaces">
        <WorkspacesCard />
      </AdminPage>
    )
  },
})
