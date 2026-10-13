import { createFileRoute, redirect } from '@tanstack/react-router'
import { CreateWorkspacePage } from '@/features/workspace/CreateWorkspacePage'
import { fetchMe, meQueryKey } from '@/lib/session'

export const Route = createFileRoute('/workspaces/new')({
  beforeLoad: async ({ context, location }) => {
    const me = await context.queryClient.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe })
    if (!me) throw redirect({ to: '/sign-in', search: { next: location.href } })
    if (me.pending) throw redirect({ to: '/setup' })
  },
  component: CreateWorkspacePage,
})
