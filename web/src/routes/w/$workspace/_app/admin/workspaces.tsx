import { createFileRoute } from '@tanstack/react-router'
import { AdminPage } from '@/features/admin/AdminPage'
import { WorkspacesCard } from '@/features/admin/instance/WorkspacesCard'
import { useT } from '@/i18n'
import { mr } from '@/routes/-routes.messages'

/** Instance admin › Workspaces: the workspaces card of InstanceSettings on its own. */
export const Route = createFileRoute('/w/$workspace/_app/admin/workspaces')({
  component: function AdminWorkspacesRoute() {
    const t = useT()
    return (
      <AdminPage documentTitle={t(mr.adminWorkspaces)}>
        <WorkspacesCard />
      </AdminPage>
    )
  },
})
