import { createFileRoute } from '@tanstack/react-router'
import { AppShell } from '@/components/shell/AppShell'
import { WorkspaceNotFound, WorkspaceRouteError } from '@/components/shell/SystemScreens'

/**
 * Screens with the sidebar (everything except wizards and full-screen
 * editors). Unknown paths and screens that fail to load keep the shell: the
 * 404, 403 and 500 states render in the content column.
 */
export const Route = createFileRoute('/w/$workspace/_app')({
  component: () => <AppShell />,
  notFoundComponent: WorkspaceNotFound,
  errorComponent: (props) => (
    <AppShell>
      <WorkspaceRouteError {...props} />
    </AppShell>
  ),
})
