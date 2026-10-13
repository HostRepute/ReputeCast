import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AccountPage } from '@/features/account/AccountPage'

export const Route = createFileRoute('/w/$workspace/_app/account')({
  // verifyEmail comes from the link in the email-change verification email.
  validateSearch: z.object({ verifyEmail: z.string().optional() }),
  component: function AccountRoute() {
    const { verifyEmail } = Route.useSearch()
    return <AccountPage verifyEmail={verifyEmail} />
  },
})
