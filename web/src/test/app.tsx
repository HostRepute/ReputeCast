import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } from '@tanstack/react-router'
import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { TooltipProvider } from '@/components/ui/Menu'
import { Toaster } from '@/components/ui/Toast'
import { WorkspaceProvider, type WorkspaceMe } from '@/lib/session'

/** A workspace member for tests: Aisha, a Viewer, unless overridden. */
export function workspaceMe(patch: Partial<WorkspaceMe> = {}): WorkspaceMe {
  return {
    actorType: 'user',
    permissions: ['campaigns:view', 'audit:view'],
    scope: {} as WorkspaceMe['scope'],
    timezone: 'UTC',
    unread: 0,
    workspace: { id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleKey: 'viewer', roleName: 'Viewer', memberCount: 6, subscribers: 48213 },
    ...patch,
  }
}

/**
 * Renders `ui` inside a memory router at `path`, a fresh query client, the
 * tooltip provider, the toaster and a workspace. Returns the router so tests
 * can check navigation.
 */
export async function renderInWorkspace(
  ui: ReactNode,
  { path = '/w/acme-newsletter/campaigns', me = workspaceMe(), seed }: { path?: string; me?: WorkspaceMe; seed?: (qc: QueryClient) => void } = {},
) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  seed?.(queryClient)
  const rootRoute = createRootRoute({ component: () => <Outlet /> })
  const page = createRoute({
    getParentRoute: () => rootRoute,
    path: '$',
    component: () => (
      <WorkspaceProvider slug={me.workspace.slug} me={me}>
        {ui}
      </WorkspaceProvider>
    ),
  })
  const router = createRouter({ routeTree: rootRoute.addChildren([page]), history: createMemoryHistory({ initialEntries: [path] }) })
  await router.load()
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={0}>
        <RouterProvider router={router} />
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>,
  )
  return { ...utils, router, queryClient }
}
