import { createFileRoute, redirect } from '@tanstack/react-router'
import { fetchMe, meQueryKey } from '@/lib/session'

/** "/" sends people to their first workspace, the setup flow or sign-in. */
export const Route = createFileRoute('/')({
  beforeLoad: async ({ context }) => {
    const me = await context.queryClient.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe })
    if (!me) throw redirect({ to: '/sign-in' })
    if (me.pending === 'password_change' || me.pending === 'instance_setup') throw redirect({ to: '/setup' })
    const first = me.workspaces[0]
    if (first) throw redirect({ to: '/w/$workspace', params: { workspace: first.slug } })
    throw redirect({ to: '/workspaces/new' })
  },
})
