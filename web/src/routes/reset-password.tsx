import { createFileRoute, Outlet, useChildMatches } from '@tanstack/react-router'
import { z } from 'zod'
import { ResetRequestPage } from '@/features/auth/ResetPages'

export const Route = createFileRoute('/reset-password')({
  validateSearch: z.object({ email: z.string().optional() }),
  component: function ResetPasswordRoute() {
    const { email } = Route.useSearch()
    // /reset-password/$token is a child route: the link from the email.
    const children = useChildMatches()
    return children.length > 0 ? <Outlet /> : <ResetRequestPage email={email} />
  },
})
