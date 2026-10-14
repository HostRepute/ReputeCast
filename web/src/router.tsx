import { createRouter } from '@tanstack/react-router'
import { onAuthProblem } from './api/client'
import { handleAuthProblem } from './lib/sessionClock'
import { queryClient } from './lib/query'
import { routeTree } from './routeTree.gen'

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
  // Data comes from TanStack Query; the router shouldn't cache loader results on its own.
  defaultPreloadStaleTime: 0,
  scrollRestoration: true,
  defaultViewTransition: false,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// Any API call that finds the session gone sends the person to sign in,
// keeping the page they were on so they come back to it.
onAuthProblem((e) => {
  // Signed-in screens handle it in place (SessionGuard), so unsaved work isn't lost.
  if (handleAuthProblem(e)) return
  const here = router.state.location.href
  if (e.code === 'password_change_required' || e.code === 'two_factor_setup_required') {
    if (!here.startsWith('/setup')) void router.navigate({ to: '/setup' })
    return
  }
  if (!here.startsWith('/sign-in')) {
    queryClient.removeQueries({ queryKey: ['me'] })
    void router.navigate({ to: '/sign-in', search: { next: here, reason: e.code === 'session_expired' ? 'expired' : undefined } })
  }
})
