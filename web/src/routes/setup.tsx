import { createFileRoute, redirect } from '@tanstack/react-router'
import { SetupPage } from '@/features/auth/SetupPage'
import { fetchMe, meQueryKey } from '@/lib/session'

/** First sign-in of the install Owner, and required personal steps for everyone else. */
export const Route = createFileRoute('/setup')({
  beforeLoad: async ({ context }) => {
    const me = await context.queryClient.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe })
    if (!me) throw redirect({ to: '/sign-in', search: { next: '/setup' } })
  },
  component: SetupPage,
})
